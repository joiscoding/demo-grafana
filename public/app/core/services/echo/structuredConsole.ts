import { createMonitoringLogger } from '@grafana/runtime';
import config from 'app/core/config';

const consoleLevels: ConsoleLogLevel[] = ['log', 'info', 'warn', 'error', 'debug', 'trace'];
let monitoringLogger: ReturnType<typeof createMonitoringLogger> | undefined;

function getMonitoringLogger() {
  monitoringLogger ??= createMonitoringLogger('frontend-console');
  return monitoringLogger;
}

function toContext(args: unknown[], timestamp: number) {
  return {
    timestamp,
    argumentCount: args.length,
  };
}

const PII_REDACT_PATTERNS = [
  { pattern: /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b/g, replacement: '[REDACTED]' },
  { pattern: /(Bearer|token|session|auth)\s*[:=]\s*['"]?\S+['"]?/gi, replacement: '[REDACTED]' },
];

function sanitizeForLogging(value: string): string {
  let result = value;
  for (const { pattern, replacement } of PII_REDACT_PATTERNS) {
    result = result.replace(pattern, replacement);
  }
  return result;
}

function toMessage(level: ConsoleLogLevel, args: unknown[]) {
  const firstArg = args[0];

  if (typeof firstArg === 'string' && firstArg.length > 0) {
    return sanitizeForLogging(firstArg);
  }

  if (firstArg instanceof Error && firstArg.message) {
    return sanitizeForLogging(firstArg.message);
  }

  return `console.${level}`;
}

function forwardLog(level: ConsoleLogLevel, args: unknown[], timestamp: number) {
  const logger = getMonitoringLogger();
  const message = toMessage(level, args);
  const context = toContext(args, timestamp);

  switch (level) {
    case 'warn':
      logger.logWarning(message, context);
      return;
    case 'error': {
      const firstError = args.find((arg): arg is Error => arg instanceof Error);
      const errorToLog = firstError
        ? Object.assign(new Error(sanitizeForLogging(firstError.message)), {
            stack: firstError.stack,
            name: firstError.name,
          })
        : new Error(message);
      logger.logError(errorToLog, context);
      return;
    }
    case 'debug':
      logger.logDebug(message, context);
      return;
    case 'trace':
      logger.logDebug(message, { ...context, level: 'trace' });
      return;
    case 'log':
    case 'info':
      logger.logInfo(message, context);
      return;
  }
}

export function enableStructuredConsoleForwarding() {
  if (window.__grafana_console_structured_forwarding__) {
    return;
  }

  const originalConsoleMethods = window.__grafana_console_original__ ?? {};
  const bufferedEntries = window.__grafana_console_buffer__ ?? [];

  for (const entry of bufferedEntries) {
    forwardLog(entry.level, entry.args, entry.timestamp);
  }

  window.__grafana_console_buffer__ = undefined;
  const consoleInstrumentationEnabled =
    config.grafanaJavascriptAgent.enabled && config.grafanaJavascriptAgent.consoleInstrumentalizationEnabled;

  if (consoleInstrumentationEnabled) {
    window.__grafana_console_structured_forwarding__ = true;
    return;
  }

  let isForwarding = false;

  for (const level of consoleLevels) {
    const original = originalConsoleMethods[level] ?? window.console[level].bind(window.console);

    window.console[level] = (...args: unknown[]) => {
      const timestamp = Date.now();
      if (!isForwarding) {
        isForwarding = true;
        try {
          forwardLog(level, args, timestamp);
        } catch {
          // Ensure original console always runs even if forwarding fails
        } finally {
          isForwarding = false;
        }
      }
      original(...args);
    };
  }

  window.__grafana_console_structured_forwarding__ = true;
}
