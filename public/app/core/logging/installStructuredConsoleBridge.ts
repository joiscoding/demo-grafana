import { config, createMonitoringLogger } from '@grafana/runtime';

type ConsoleLevel = 'log' | 'info' | 'warn' | 'error' | 'debug' | 'trace';
type ConsoleMethod = (...args: unknown[]) => void;
type ConsoleMethods = Record<ConsoleLevel, ConsoleMethod>;

const CONSOLE_LEVELS: ConsoleLevel[] = ['log', 'info', 'warn', 'error', 'debug', 'trace'];
let restoreBridge: (() => void) | undefined;

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
  const shouldEmitStructuredLogs = !config.grafanaJavascriptAgent.consoleInstrumentalizationEnabled;

  const emitStructuredLog = (level: ConsoleLevel, args: unknown[]) => {
    if (!shouldEmitStructuredLogs) {
      return;
    }

    const message = formatLogMessage(level, args);
    const serializedArgs = args.map(serializeForContext);
    const context = { level, args: serializedArgs };

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
