const TIMER_MISSING_MESSAGE = 'Structured timer missing start';
const GROUP_END_MESSAGE = 'Structured group end';
const DIR_MESSAGE = 'Structured dir';
const DEFAULT_MESSAGE = 'Structured log';

function normalizeError(error) {
  return {
    name: error.name,
    message: error.message,
    stack: error.stack,
  };
}

function normalizeValue(value, seen = new WeakSet()) {
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
    const normalized = {};

    for (const [key, nestedValue] of Object.entries(value)) {
      normalized[key] = normalizeValue(nestedValue, seen);
    }

    seen.delete(value);
    return normalized;
  }

  return String(value);
}

function getMessage(args, fallback) {
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

function getError(args) {
  const error = args.find((arg) => arg instanceof Error);
  return error ? normalizeError(error) : undefined;
}

function writeStructuredLog(entry) {
  const serializedEntry = JSON.stringify(entry);
  const stream = entry.level === 'warn' || entry.level === 'error' ? process.stderr : process.stdout;
  stream.write(`${serializedEntry}\n`);
}

function emitStructuredLog(source, level, args, defaultContext, fallbackMessage = DEFAULT_MESSAGE) {
  const normalizedArgs = args.map((arg) => normalizeValue(arg));

  writeStructuredLog({
    timestamp: new Date().toISOString(),
    level,
    source,
    message: getMessage(args, fallbackMessage),
    error: getError(args),
    arguments: normalizedArgs.length > 0 ? normalizedArgs : undefined,
    context: defaultContext,
  });
}

function createStructuredLogger(source, defaultContext) {
  const timers = new Map();

  return {
    trace: (...args) => emitStructuredLog(source, 'trace', args, defaultContext),
    debug: (...args) => emitStructuredLog(source, 'debug', args, defaultContext),
    info: (...args) => emitStructuredLog(source, 'info', args, defaultContext),
    log: (...args) => emitStructuredLog(source, 'info', args, defaultContext),
    warn: (...args) => emitStructuredLog(source, 'warn', args, defaultContext),
    error: (...args) => emitStructuredLog(source, 'error', args, defaultContext),
    dir: (value, options) => emitStructuredLog(source, 'debug', [DIR_MESSAGE, { value, options }], defaultContext, DIR_MESSAGE),
    groupCollapsed: (...args) =>
      emitStructuredLog(source, 'info', args, { ...defaultContext, group: 'start', collapsed: true }),
    groupEnd: (...args) =>
      emitStructuredLog(source, 'info', args, { ...defaultContext, group: 'end' }, GROUP_END_MESSAGE),
    time: (label, ...args) => {
      timers.set(label, Date.now());
      emitStructuredLog(source, 'debug', [`Structured timer start: ${label}`, ...args], { ...defaultContext, label });
    },
    timeEnd: (label, ...args) => {
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

module.exports = {
  createStructuredLogger,
};
