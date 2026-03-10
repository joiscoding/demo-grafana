import { createMonitoringLogger } from '@grafana/runtime';

const frontendConsoleLogger = createMonitoringLogger('frontend-console');

export function logStructuredInfo(source: string, ...args: unknown[]): void {
  const [firstArg, ...remainingArgs] = args;
  const message = typeof firstArg === 'string' ? firstArg : 'console.log';
  const contextArgs = typeof firstArg === 'string' ? remainingArgs : args;

  frontendConsoleLogger.logInfo(message, {
    source,
    args: contextArgs,
  });
}
