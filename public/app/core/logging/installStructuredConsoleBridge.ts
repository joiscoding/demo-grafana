import { config, createMonitoringLogger } from '@grafana/runtime';

type ConsoleLevel = 'log' | 'info' | 'warn' | 'error' | 'debug' | 'trace';
type ConsoleMethod = (...args: unknown[]) => void;
type ConsoleMethods = Record<ConsoleLevel, ConsoleMethod>;

const CONSOLE_LEVELS: ConsoleLevel[] = ['log', 'info', 'warn', 'error', 'debug', 'trace'];
let restoreBridge: (() => void) | undefined;
const SENSITIVE_KEYS =
  /^(password|token|secret|apikey|api_key|authorization|cookie|sessionid|session_id|auth_token|privatekey|private_key)$/i;
const EMAIL_PATTERN = /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g;
const BEARER_TOKEN_PATTERN = /Bearer\s+[^\s]+/gi;

function serializeForContext(value: unknown): unknown {
  if (value instanceof Error) {
    return {
      name: value.name,
      message: value.message,
      stack: value.stack,
    };
  }

  if (typeof value === 'function') {
    return `[Function ${value.name || 'anonymous'}]`;
  }

  if (typeof value === 'symbol') {
    return value.toString();
  }

  if (typeof value === 'bigint') {
    return value.toString();
  }

  if (value === undefined || value === null) {
    return value;
  }

  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }

  try {
    return JSON.parse(JSON.stringify(value));
  } catch {
    return String(value);
  }
}

function formatLogMessage(level: ConsoleLevel, args: unknown[]): string {
  if (args.length === 0) {
    return `console.${level}`;
  }

  const [first] = args;
  if (typeof first === 'string') {
    return first;
  }

  if (first instanceof Error) {
    return first.message;
  }

  const serializedValue = serializeForContext(first);
  return `console.${level}: ${typeof serializedValue === 'string' ? serializedValue : JSON.stringify(serializedValue)}`;
}

function sanitizeForRemoteContext(value: unknown): unknown {
  if (value === undefined || value === null) {
    return value;
  }

  if (typeof value === 'string') {
    return value.replace(EMAIL_PATTERN, '[REDACTED]').replace(BEARER_TOKEN_PATTERN, 'Bearer [REDACTED]');
  }

  if (typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }

  if (Array.isArray(value)) {
    return value.map((item) => sanitizeForRemoteContext(item));
  }

  if (typeof value === 'object') {
    const result: Record<string, unknown> = {};

    for (const [key, nestedValue] of Object.entries(value)) {
      result[key] = SENSITIVE_KEYS.test(key) ? '[REDACTED]' : sanitizeForRemoteContext(nestedValue);
    }

    return result;
  }

  return value;
}

export function installStructuredConsoleBridge(source = 'frontend.console') {
  if (restoreBridge) {
    return restoreBridge;
  }

  const nativeConsole = globalThis.console;
  const logger = createMonitoringLogger(source);
  const originalConsole: ConsoleMethods = {
    log: nativeConsole.log.bind(nativeConsole),
    info: nativeConsole.info.bind(nativeConsole),
    warn: nativeConsole.warn.bind(nativeConsole),
    error: nativeConsole.error.bind(nativeConsole),
    debug: nativeConsole.debug.bind(nativeConsole),
    trace: nativeConsole.trace.bind(nativeConsole),
  };
  const shouldEmitStructuredLogs =
    config.grafanaJavascriptAgent.enabled && !config.grafanaJavascriptAgent.consoleInstrumentalizationEnabled;

  const emitStructuredLog = (level: ConsoleLevel, args: unknown[]) => {
    if (!shouldEmitStructuredLogs) {
      return;
    }

    const message = formatLogMessage(level, args);
    const serializedArgs = args.map(serializeForContext);
    const sanitizedArgs = serializedArgs.map((arg) => sanitizeForRemoteContext(arg));
    const context: Record<string, string> = {
      level,
      args: JSON.stringify(sanitizedArgs),
    };

    if (level === 'error') {
      const [firstArg] = args;
      const error = firstArg instanceof Error ? firstArg : new Error(message);
      logger.logError(error, context);
      return;
    }

    if (level === 'warn') {
      logger.logWarning(message, context);
      return;
    }

    if (level === 'debug' || level === 'trace') {
      logger.logDebug(message, context);
      return;
    }

    logger.logInfo(message, context);
  };

  CONSOLE_LEVELS.forEach((level) => {
    nativeConsole[level] = (...args: unknown[]) => {
      emitStructuredLog(level, args);
      originalConsole[level](...args);
    };
  });

  const restore = () => {
    CONSOLE_LEVELS.forEach((level) => {
      nativeConsole[level] = originalConsole[level];
    });
    restoreBridge = undefined;
  };

  restoreBridge = restore;
  return restore;
}
