import { createMonitoringLogger } from '@grafana/runtime';
import config from 'app/core/config';

const consoleLevels: ConsoleLogLevel[] = ['log', 'info', 'warn', 'error', 'debug', 'trace'];
let monitoringLogger: ReturnType<typeof createMonitoringLogger> | undefined;

function getMonitoringLogger() {
  monitoringLogger ??= createMonitoringLogger('frontend-console');
  return monitoringLogger;
}

function toSerializableArgument(arg: unknown): unknown {
  if (arg instanceof Error) {
    return {
      name: arg.name,
      message: arg.message,
      stack: arg.stack,
    };
  }

  if (typeof arg === 'function') {
    return `[Function ${arg.name || 'anonymous'}]`;
  }

  if (typeof arg === 'symbol') {
    return arg.toString();
  }

  if (typeof arg === 'bigint') {
    return arg.toString();
  }

  return arg;
}

function toContext(args: unknown[], timestamp: number) {
  return {
    timestamp,
    argumentCount: args.length,
    arguments: args.map(toSerializableArgument),
  };
}

function toMessage(level: ConsoleLogLevel, args: unknown[]) {
  const firstArg = args[0];

  if (typeof firstArg === 'string' && firstArg.length > 0) {
    return firstArg;
  }

  if (firstArg instanceof Error && firstArg.message) {
    return firstArg.message;
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
      logger.logError(firstError ?? new Error(message), context);
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

  for (const level of consoleLevels) {
    const original = originalConsoleMethods[level] ?? window.console[level].bind(window.console);

    window.console[level] = (...args: unknown[]) => {
      const timestamp = Date.now();
      forwardLog(level, args, timestamp);
      original(...args);
    };
  }

  window.__grafana_console_structured_forwarding__ = true;
}
