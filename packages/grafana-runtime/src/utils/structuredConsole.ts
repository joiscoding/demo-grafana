import { LogContext } from '@grafana/faro-web-sdk';

import { createMonitoringLogger } from './logging';

type StructuredConsoleMethod = 'log' | 'info' | 'warn' | 'error' | 'debug' | 'trace';
const STRUCTURED_CONSOLE_METHODS: StructuredConsoleMethod[] = ['log', 'info', 'warn', 'error', 'debug', 'trace'];

type StructuredConsolePayload = {
  level: StructuredConsoleMethod;
  message: string;
  args: unknown[];
  timestamp: string;
};

export type StructuredConsole = Pick<Console, StructuredConsoleMethod>;

const STRUCTURED_CONSOLE_KEY = '__grafanaStructuredConsole';
const monitoringLogger = createMonitoringLogger('core.structured-console');
const rawConsole = console;
const REDACTED_VALUE = '[REDACTED]';
const MAX_MESSAGE_LENGTH = 200;

function sanitizeString(value: string): string {
  const sanitized = value
    .replace(/\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi, REDACTED_VALUE)
    .replace(/(Bearer\s+)[^\s,;]+/gi, `$1${REDACTED_VALUE}`)
    .replace(/([?&](?:access_token|auth_token|refresh_token|token|session(?:_id|id)?)=)[^&\s]+/gi, `$1${REDACTED_VALUE}`)
    .replace(
      /\b((?:access|auth|refresh|api|session)[-_ ]?(?:token|id))\b\s*[:=]\s*["']?[^"',\s}]+["']?/gi,
      `$1=${REDACTED_VALUE}`
    );

  return sanitized.length > MAX_MESSAGE_LENGTH ? `${sanitized.slice(0, MAX_MESSAGE_LENGTH)}...` : sanitized;
}

function summarizeArgument(argument: unknown): unknown {
  if (typeof argument === 'string') {
    return sanitizeString(argument);
  }

  if (argument instanceof Error) {
    return {
      type: 'Error',
      name: argument.name,
    };
  }

  if (Array.isArray(argument)) {
    return {
      type: 'Array',
      length: argument.length,
    };
  }

  if (typeof argument === 'bigint' || typeof argument === 'symbol') {
    return {
      type: typeof argument,
    };
  }

  if (typeof argument === 'function') {
    return {
      type: 'Function',
      name: argument.name || 'anonymous',
    };
  }

  if (argument && typeof argument === 'object') {
    return {
      type: argument.constructor?.name ?? 'Object',
    };
  }

  return argument;
}

function getMessageFromArgs(level: StructuredConsoleMethod, args: unknown[]): string {
  const [first] = args;

  if (typeof first === 'string' && first.length > 0) {
    return sanitizeString(first);
  }

  if (first instanceof Error && first.message.length > 0) {
    return sanitizeString(first.message);
  }

  return `console.${level}`;
}

function toPayload(level: StructuredConsoleMethod, args: unknown[]): StructuredConsolePayload {
  return {
    level,
    message: getMessageFromArgs(level, args),
    args: args.map(summarizeArgument),
    timestamp: new Date().toISOString(),
  };
}

function toContext(payload: StructuredConsolePayload): LogContext {
  return {
    console: payload,
  };
}

function getConsoleError(payload: StructuredConsolePayload, args: unknown[]): Error {
  const firstError = args.find((arg): arg is Error => arg instanceof Error);

  if (firstError) {
    const error = new Error(payload.message || firstError.name || 'console.error');
    error.name = firstError.name || error.name;
    return error;
  }

  return new Error(payload.message);
}

function isStructuredConsole(value: unknown): value is StructuredConsole {
  if (!value || typeof value !== 'object') {
    return false;
  }

  return STRUCTURED_CONSOLE_METHODS.every((method) => typeof Reflect.get(value, method) === 'function');
}

function emitStructuredLog(method: StructuredConsoleMethod, args: unknown[]) {
  const payload = toPayload(method, args);
  const context = toContext(payload);

  switch (method) {
    case 'error':
      monitoringLogger.logError(getConsoleError(payload, args), context);
      break;
    case 'warn':
      monitoringLogger.logWarning(payload.message, context);
      break;
    case 'debug':
    case 'trace':
      monitoringLogger.logDebug(payload.message, context);
      break;
    case 'log':
    case 'info':
      monitoringLogger.logInfo(payload.message, context);
      break;
  }

  rawConsole[method](...args);
}

export function createStructuredConsole(): StructuredConsole {
  return {
    log: (...args: unknown[]) => emitStructuredLog('log', args),
    info: (...args: unknown[]) => emitStructuredLog('info', args),
    warn: (...args: unknown[]) => emitStructuredLog('warn', args),
    error: (...args: unknown[]) => emitStructuredLog('error', args),
    debug: (...args: unknown[]) => emitStructuredLog('debug', args),
    trace: (...args: unknown[]) => emitStructuredLog('trace', args),
  };
}

export function installStructuredConsole(structuredConsole: StructuredConsole = createStructuredConsole()): StructuredConsole {
  const existing = Reflect.get(globalThis, STRUCTURED_CONSOLE_KEY);

  if (isStructuredConsole(existing)) {
    return existing;
  }

  Reflect.set(globalThis, STRUCTURED_CONSOLE_KEY, structuredConsole);
  return structuredConsole;
}
