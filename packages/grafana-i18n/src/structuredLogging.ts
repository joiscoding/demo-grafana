type StructuredLogLevel = 'warn' | 'error';

function normalizeValue(value: unknown): unknown {
  if (value instanceof Error) {
    return {
      name: value.name,
      message: value.message,
      stack: value.stack,
    };
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

  if (typeof value === 'object') {
    try {
      return JSON.parse(JSON.stringify(value));
    } catch {
      return String(value);
    }
  }

  return String(value);
}

function writeStructuredLog(level: StructuredLogLevel, source: string, args: unknown[]) {
  const payload = {
    timestamp: new Date().toISOString(),
    level,
    source,
    message: typeof args[0] === 'string' ? args[0] : 'Structured log',
    arguments: args.map(normalizeValue),
  };

  const shouldUseConsole = typeof window !== 'undefined' || process?.env?.NODE_ENV === 'test';
  if (!shouldUseConsole && process?.stdout?.write && process?.stderr?.write) {
    process.stderr.write(`${JSON.stringify(payload)}\n`);
    return;
  }

  const browserConsole = globalThis.console;
  const method = level === 'warn' ? browserConsole?.warn : browserConsole?.error;
  method?.(payload);
}

export function createStructuredLogger(source: string) {
  return {
    warn: (...args: unknown[]) => writeStructuredLog('warn', source, args),
    error: (...args: unknown[]) => writeStructuredLog('error', source, args),
  };
}
