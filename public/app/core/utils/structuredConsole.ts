/* eslint-disable no-console */
import { createMonitoringLogger, MonitoringLogger } from '@grafana/runtime';

type ConsoleMethod = 'debug' | 'error' | 'info' | 'log' | 'trace' | 'warn';

const monitoringLogger: MonitoringLogger = createMonitoringLogger('frontend.console');
const consoleMethods: ConsoleMethod[] = ['log', 'info', 'warn', 'error', 'debug', 'trace'];

let structuredConsoleInstalled = false;
let forwardingLog = false;

function createLogMessage(method: ConsoleMethod, args: unknown[]): string {
  if (typeof args[0] === 'string' && args[0].length > 0) {
    return args[0];
  }

  if (args[0] instanceof Error && args[0].message.length > 0) {
    return args[0].message;
  }

  return `console.${method}`;
}

/** Returns flat Record<string, string> as required by Faro. Does not include raw args to avoid PII leakage. */
function createLogContext(method: ConsoleMethod): Record<string, string> {
  return {
    consoleMethod: method,
  };
}

function toError(args: unknown[], fallbackMessage: string): Error {
  const errorArgument = args.find((arg): arg is Error => arg instanceof Error);
  return errorArgument ?? new Error(fallbackMessage);
}

function forwardToStructuredLogger(method: ConsoleMethod, args: unknown[]) {
  const message = createLogMessage(method, args);
  const context = createLogContext(method);

  switch (method) {
    case 'error':
      monitoringLogger.logError(toError(args, message), context);
      break;
    case 'warn':
      monitoringLogger.logWarning(message, context);
      break;
    case 'debug':
    case 'trace':
      monitoringLogger.logDebug(message, context);
      break;
    case 'info':
    case 'log':
      monitoringLogger.logInfo(message, context);
      break;
  }
}

export function installStructuredConsoleLogging() {
  if (structuredConsoleInstalled) {
    return;
  }

  structuredConsoleInstalled = true;

  for (const method of consoleMethods) {
    const originalMethod = console[method].bind(console);

    console[method] = (...args: unknown[]) => {
      originalMethod(...args);

      if (forwardingLog) {
        return;
      }

      forwardingLog = true;
      try {
        forwardToStructuredLogger(method, args);
      } finally {
        forwardingLog = false;
      }
    };
  }
}
