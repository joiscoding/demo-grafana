export type StructuredLogLevel = 'trace' | 'debug' | 'info' | 'warn' | 'error';

export interface StructuredLogError {
  name: string;
  message: string;
  stack?: string;
}

export interface StructuredLogEntry {
  timestamp: string;
  level: StructuredLogLevel;
  source: string;
  message: string;
  context?: Record<string, unknown>;
  arguments?: unknown[];
  error?: StructuredLogError;
}

export interface StructuredLogger {
  trace: (...args: unknown[]) => void;
  debug: (...args: unknown[]) => void;
  info: (...args: unknown[]) => void;
  log: (...args: unknown[]) => void;
  warn: (...args: unknown[]) => void;
  error: (...args: unknown[]) => void;
  dir: (value: unknown, options?: unknown) => void;
  groupCollapsed: (...args: unknown[]) => void;
  groupEnd: (...args: unknown[]) => void;
  time: (label: string, ...args: unknown[]) => void;
  timeEnd: (label: string, ...args: unknown[]) => void;
}

const TIMER_MISSING_MESSAGE = 'Structured timer missing start';
const GROUP_END_MESSAGE = 'Structured group end';
const DIR_MESSAGE = 'Structured dir';
const DEFAULT_MESSAGE = 'Structured log';

function normalizeValue(value: unknown, seen = new WeakSet<object>()): unknown {
  if (value instanceof Error) {
    return normalizeError(value);
  }

  if (
    value === null ||
    value === undefined ||
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean'
  ) {
    return value;
  }

  if (typeof value === 'bigint') {
    return value.toString();
  }

  if (typeof value === 'function') {
    return `[Function ${value.name || 'anonymous'}]`;
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (Array.isArray(value)) {
    return value.map((item) => normalizeValue(item, seen));
  }

  if (typeof value === 'object') {
    if (seen.has(value)) {
      return '[Circular]';
    }

    seen.add(value);
    const normalized: Record<string, unknown> = {};

    for (const [key, nestedValue] of Object.entries(value)) {
      normalized[key] = normalizeValue(nestedValue, seen);
    }

    seen.delete(value);
    return normalized;
  }

  return String(value);
}

function normalizeError(error: Error): StructuredLogError {
  return {
    name: error.name,
    message: error.message,
    stack: error.stack,
  };
}

function getMessage(args: unknown[], fallback: string): string {
  for (const arg of args) {
    if (typeof arg === 'string' && arg.length > 0) {
      return arg;
    }

    if (arg instanceof Error && arg.message.length > 0) {
      return arg.message;
    }
  }

  return fallback;
}

function getError(args: unknown[]): StructuredLogError | undefined {
  const error = args.find((arg): arg is Error => arg instanceof Error);
  return error ? normalizeError(error) : undefined;
}

function writeStructuredLog(entry: StructuredLogEntry) {
  const serializedEntry = JSON.stringify(entry);
  const shouldUseConsole = typeof window !== 'undefined' || process?.env?.NODE_ENV === 'test';

  if (!shouldUseConsole && typeof process !== 'undefined' && process?.stdout?.write && process?.stderr?.write) {
    const stream = entry.level === 'warn' || entry.level === 'error' ? process.stderr : process.stdout;
    stream.write(`${serializedEntry}\n`);
    return;
  }

  const browserConsole = globalThis.console;
  const consoleMethod =
    entry.level === 'trace'
      ? browserConsole?.debug
      : entry.level === 'debug'
        ? browserConsole?.debug
        : entry.level === 'info'
          ? browserConsole?.info
          : entry.level === 'warn'
            ? browserConsole?.warn
            : browserConsole?.error;

  consoleMethod?.(entry);
}

function emitStructuredLog(
  source: string,
  level: StructuredLogLevel,
  args: unknown[],
  defaultContext?: Record<string, unknown>,
  fallbackMessage = DEFAULT_MESSAGE
) {
  const normalizedArgs = args.map((arg) => normalizeValue(arg));
  const entry: StructuredLogEntry = {
    timestamp: new Date().toISOString(),
    level,
    source,
    message: getMessage(args, fallbackMessage),
    error: getError(args),
    arguments: normalizedArgs.length > 0 ? normalizedArgs : undefined,
    context: defaultContext,
  };

  writeStructuredLog(entry);
}

export function createStructuredLogger(source: string, defaultContext?: Record<string, unknown>): StructuredLogger {
  const timers = new Map<string, number>();

  return {
    trace: (...args: unknown[]) => emitStructuredLog(source, 'trace', args, defaultContext),
    debug: (...args: unknown[]) => emitStructuredLog(source, 'debug', args, defaultContext),
    info: (...args: unknown[]) => emitStructuredLog(source, 'info', args, defaultContext),
    log: (...args: unknown[]) => emitStructuredLog(source, 'info', args, defaultContext),
    warn: (...args: unknown[]) => emitStructuredLog(source, 'warn', args, defaultContext),
    error: (...args: unknown[]) => emitStructuredLog(source, 'error', args, defaultContext),
    dir: (value: unknown, options?: unknown) =>
      emitStructuredLog(source, 'debug', [DIR_MESSAGE, { value, options }], defaultContext, DIR_MESSAGE),
    groupCollapsed: (...args: unknown[]) =>
      emitStructuredLog(source, 'info', args, { ...defaultContext, group: 'start', collapsed: true }),
    groupEnd: (...args: unknown[]) =>
      emitStructuredLog(source, 'info', args, { ...defaultContext, group: 'end' }, GROUP_END_MESSAGE),
    time: (label: string, ...args: unknown[]) => {
      timers.set(label, Date.now());
      emitStructuredLog(source, 'debug', [`Structured timer start: ${label}`, ...args], { ...defaultContext, label });
    },
    timeEnd: (label: string, ...args: unknown[]) => {
      const startedAt = timers.get(label);

      if (startedAt === undefined) {
        emitStructuredLog(
          source,
          'warn',
          [TIMER_MISSING_MESSAGE, label, ...args],
          { ...defaultContext, label },
          TIMER_MISSING_MESSAGE
        );
        return;
      }

      timers.delete(label);
      emitStructuredLog(
        source,
        'info',
        [`Structured timer end: ${label}`, ...args],
        { ...defaultContext, label, durationMs: Date.now() - startedAt }
      );
    },
  };
}
