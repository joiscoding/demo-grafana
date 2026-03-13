/* eslint-disable no-console */
import { createMonitoringLogger, MonitoringLogger } from '@grafana/runtime';

type ConsoleMethod = 'debug' | 'error' | 'info' | 'log' | 'trace' | 'warn';

type SerializableValue = boolean | number | string | null | SerializableValue[] | { [key: string]: SerializableValue };

const monitoringLogger: MonitoringLogger = createMonitoringLogger('frontend.console');
const consoleMethods: ConsoleMethod[] = ['log', 'info', 'warn', 'error', 'debug', 'trace'];

let structuredConsoleInstalled = false;
let forwardingLog = false;

function toSerializableValue(value: unknown): SerializableValue {
  if (value === null || value === undefined) {
    return null;
  }

  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }

  if (typeof value === 'bigint' || typeof value === 'symbol' || typeof value === 'function') {
    return String(value);
  }

  if (value instanceof Error) {
    return {
      name: value.name,
      message: value.message,
      stack: value.stack ?? '',
    };
  }

  if (Array.isArray(value)) {
    return value.map((item) => toSerializableValue(item));
  }

  try {
    return JSON.parse(JSON.stringify(value));
  } catch {
    return String(value);
  }
}

function createLogMessage(method: ConsoleMethod, args: unknown[]): string {
  if (typeof args[0] === 'string' && args[0].length > 0) {
    return args[0];
  }

  if (args[0] instanceof Error && args[0].message.length > 0) {
    return args[0].message;
  }

  return `console.${method}`;
}

function createLogContext(method: ConsoleMethod, args: unknown[]): Record<string, SerializableValue> {
  return {
    consoleMethod: method,
    arguments: args.map((arg) => toSerializableValue(arg)),
  };
}

function toError(args: unknown[], fallbackMessage: string): Error {
  const errorArgument = args.find((arg): arg is Error => arg instanceof Error);
  return errorArgument ?? new Error(fallbackMessage);
}

function forwardToStructuredLogger(method: ConsoleMethod, args: unknown[]) {
  const message = createLogMessage(method, args);
  const context = createLogContext(method, args);

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
