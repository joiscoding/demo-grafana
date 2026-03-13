import type { MonitoringLogger } from './logging';

const mockMonitoringLogger: MonitoringLogger = {
  logDebug: jest.fn(),
  logError: jest.fn(),
  logInfo: jest.fn(),
  logMeasurement: jest.fn(),
  logWarning: jest.fn(),
};

jest.mock('./logging', () => ({
  createMonitoringLogger: jest.fn(() => mockMonitoringLogger),
}));

const { createStructuredConsole, installStructuredConsole } = require('./structuredConsole') as typeof import('./structuredConsole');

describe('structuredConsole', () => {
  const structuredConsoleKey = '__grafanaStructuredConsole';
  let consoleWarnSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.clearAllMocks();
    Reflect.deleteProperty(globalThis, structuredConsoleKey);
    jest.spyOn(console, 'log').mockImplementation(() => undefined);
    jest.spyOn(console, 'info').mockImplementation(() => undefined);
    consoleWarnSpy = jest.spyOn(console, 'warn').mockImplementation(() => undefined);
    jest.spyOn(console, 'error').mockImplementation(() => undefined);
    jest.spyOn(console, 'debug').mockImplementation(() => undefined);
    jest.spyOn(console, 'trace').mockImplementation(() => undefined);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  test('sends warning logs with a structured payload', () => {
    const structuredConsole = createStructuredConsole();

    structuredConsole.warn('warning message', { userId: 7 });

    expect(mockMonitoringLogger.logWarning).toHaveBeenCalledWith(
      'warning message',
      expect.objectContaining({
        console: expect.objectContaining({
          level: 'warn',
          message: 'warning message',
          args: ['warning message', { type: 'Object' }],
          timestamp: expect.any(String),
        }),
      })
    );
  });

  test('converts error arguments into Error instances', () => {
    const structuredConsole = createStructuredConsole();
    const originalError = new Error('boom');

    structuredConsole.error(originalError, { feature: 'alerts' });

    expect(mockMonitoringLogger.logError).toHaveBeenCalledTimes(1);

    const [loggedError, context] = (mockMonitoringLogger.logError as jest.Mock).mock.calls[0];
    expect(loggedError).toBeInstanceOf(Error);
    expect(loggedError.message).toBe('boom');
    expect(context).toEqual(
      expect.objectContaining({
        console: expect.objectContaining({
          level: 'error',
          message: 'boom',
        }),
      })
    );
  });

  test('sanitizes monitoring payloads before sending them to monitoring', () => {
    const structuredConsole = createStructuredConsole();
    const originalError = new Error('session_id=secret-session');

    structuredConsole.error(originalError, { token: 'secret-token' }, 'Bearer abc123');

    const [loggedError, context] = (mockMonitoringLogger.logError as jest.Mock).mock.calls[0];
    expect(loggedError).toBeInstanceOf(Error);
    expect(loggedError.message).toBe('session_id=[REDACTED]');
    expect(context).toEqual(
      expect.objectContaining({
        console: expect.objectContaining({
          level: 'error',
          message: 'session_id=[REDACTED]',
          args: [{ type: 'Error', name: 'Error' }, { type: 'Object' }, 'Bearer [REDACTED]'],
        }),
      })
    );
  });

  test('forwards original arguments to the raw browser console', () => {
    const structuredConsole = createStructuredConsole();
    const details = { userId: 7 };

    structuredConsole.warn('warning message', details);

    expect(consoleWarnSpy).toHaveBeenCalledWith('warning message', details);
  });

  test('installs only once and reuses an existing structured console', () => {
    const first = installStructuredConsole();
    const second = installStructuredConsole({
      log: jest.fn(),
      info: jest.fn(),
      warn: jest.fn(),
      error: jest.fn(),
      debug: jest.fn(),
      trace: jest.fn(),
    });

    expect(second).toBe(first);
    expect(Reflect.get(globalThis, structuredConsoleKey)).toBe(first);
  });
});
