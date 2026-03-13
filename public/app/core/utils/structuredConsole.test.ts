/* eslint-disable no-console */
const mockLogDebug = jest.fn();
const mockLogError = jest.fn();
const mockLogInfo = jest.fn();
const mockLogWarning = jest.fn();
const mockLogMeasurement = jest.fn();
const mockCreateMonitoringLogger = jest.fn(() => ({
  logDebug: mockLogDebug,
  logError: mockLogError,
  logInfo: mockLogInfo,
  logWarning: mockLogWarning,
  logMeasurement: mockLogMeasurement,
}));

jest.mock('@grafana/runtime', () => ({
  createMonitoringLogger: mockCreateMonitoringLogger,
}));

const originalConsole = {
  debug: console.debug,
  error: console.error,
  info: console.info,
  log: console.log,
  trace: console.trace,
  warn: console.warn,
};

describe('installStructuredConsoleLogging', () => {
  beforeEach(() => {
    jest.resetModules();
    jest.clearAllMocks();

    console.debug = originalConsole.debug;
    console.error = originalConsole.error;
    console.info = originalConsole.info;
    console.log = originalConsole.log;
    console.trace = originalConsole.trace;
    console.warn = originalConsole.warn;
  });

  afterAll(() => {
    console.debug = originalConsole.debug;
    console.error = originalConsole.error;
    console.info = originalConsole.info;
    console.log = originalConsole.log;
    console.trace = originalConsole.trace;
    console.warn = originalConsole.warn;
  });

  it('forwards console.warn to structured warning logs', async () => {
    const warnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    const { installStructuredConsoleLogging } = await import('./structuredConsole');

    installStructuredConsoleLogging();
    console.warn('warning message', { requestId: '123' });

    expect(mockLogWarning).toHaveBeenCalledWith(
      'warning message',
      expect.objectContaining({
        consoleMethod: 'warn',
        arguments: ['warning message', { requestId: '123' }],
      })
    );
    expect(warnSpy).toHaveBeenCalledWith('warning message', { requestId: '123' });
  });

  it('forwards console.error and creates an Error from string input', async () => {
    const errorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
    const { installStructuredConsoleLogging } = await import('./structuredConsole');

    installStructuredConsoleLogging();
    console.error('something failed', { path: '/api/health' });

    expect(mockLogError).toHaveBeenCalledTimes(1);
    const [error, context] = mockLogError.mock.calls[0];
    expect(error).toBeInstanceOf(Error);
    expect(error.message).toBe('something failed');
    expect(context).toMatchObject({
      consoleMethod: 'error',
      arguments: ['something failed', { path: '/api/health' }],
    });
    expect(errorSpy).toHaveBeenCalledWith('something failed', { path: '/api/health' });
  });

  it('installs only once and avoids duplicate forwarding', async () => {
    const logSpy = jest.spyOn(console, 'log').mockImplementation(() => {});
    const { installStructuredConsoleLogging } = await import('./structuredConsole');

    installStructuredConsoleLogging();
    installStructuredConsoleLogging();
    console.log('single event');

    expect(mockLogInfo).toHaveBeenCalledTimes(1);
    expect(mockLogInfo).toHaveBeenCalledWith(
      'single event',
      expect.objectContaining({
        consoleMethod: 'log',
      })
    );
    expect(logSpy).toHaveBeenCalledWith('single event');
  });
});
