import { config, createMonitoringLogger } from '@grafana/runtime';

const frontendConsoleLogger = createMonitoringLogger('frontend-console');

function serializeArgs(args: unknown[]): string {
  try {
    return JSON.stringify(args);
  } catch {
    return String(args);
  }
}

function logStructured(
  level: 'info' | 'warn' | 'error' | 'debug',
  source: string,
  ...args: unknown[]
): void {
  const [firstArg, ...remainingArgs] = args;
  const message = typeof firstArg === 'string' ? firstArg : 'console.log';
  const contextArgs = typeof firstArg === 'string' ? remainingArgs : args;
  const context: Record<string, string> = {
    source,
    args: serializeArgs(contextArgs),
  };

  if (config.grafanaJavascriptAgent.enabled) {
    switch (level) {
      case 'info':
        frontendConsoleLogger.logInfo(message, context);
        break;
      case 'warn':
        frontendConsoleLogger.logWarning(message, context);
        break;
      case 'error': {
        const err = contextArgs.find((a): a is Error => a instanceof Error) ?? new Error(message);
        frontendConsoleLogger.logError(err, context);
        break;
      }
      case 'debug':
        frontendConsoleLogger.logDebug(message, context);
        break;
    }
  } else {
    const consoleArgs = [message, ...contextArgs];
    switch (level) {
      case 'info':
        console.log(...consoleArgs);
        break;
      case 'warn':
        console.warn(...consoleArgs);
        break;
      case 'error':
        console.error(...consoleArgs);
        break;
      case 'debug':
        console.debug(...consoleArgs);
        break;
    }
  }
}

export function logStructuredInfo(source: string, ...args: unknown[]): void {
  logStructured('info', source, ...args);
}

export function logStructuredWarn(source: string, ...args: unknown[]): void {
  logStructured('warn', source, ...args);
}

export function logStructuredError(source: string, ...args: unknown[]): void {
  logStructured('error', source, ...args);
}

export function logStructuredDebug(source: string, ...args: unknown[]): void {
  logStructured('debug', source, ...args);
}
