import { config, createMonitoringLogger } from '@grafana/runtime';

import { installStructuredConsoleBridge } from './installStructuredConsoleBridge';

const mockLogger = {
  logDebug: jest.fn(),
  logInfo: jest.fn(),
  logWarning: jest.fn(),
  logError: jest.fn(),
  logMeasurement: jest.fn(),
};

jest.mock('@grafana/runtime', () => ({
  config: {
    grafanaJavascriptAgent: {
      enabled: true,
      consoleInstrumentalizationEnabled: false,
    },
  },
  createMonitoringLogger: jest.fn(() => mockLogger),
}));

describe('installStructuredConsoleBridge', () => {
  let restoreBridge: (() => void) | undefined;
  let logSpy: jest.SpyInstance;
  let warnSpy: jest.SpyInstance;
  let errorSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.clearAllMocks();
    restoreBridge?.();
    restoreBridge = undefined;

    const agentConfig = config.grafanaJavascriptAgent as {
      enabled: boolean;
      consoleInstrumentalizationEnabled: boolean;
    };
    agentConfig.enabled = true;
    agentConfig.consoleInstrumentalizationEnabled = false;

    logSpy = jest.spyOn(console, 'log').mockImplementation(() => {});
    warnSpy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    errorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    restoreBridge?.();
    logSpy.mockRestore();
    warnSpy.mockRestore();
    errorSpy.mockRestore();
  });

  it('forwards console.log to monitoring logger as info', () => {
    restoreBridge = installStructuredConsoleBridge('unit-test.console');
    console.log('hello', { foo: 'bar' });

    expect(createMonitoringLogger).toHaveBeenCalledWith('unit-test.console');
    expect(mockLogger.logInfo).toHaveBeenCalledWith(
      'hello',
      expect.objectContaining({
        level: 'log',
        args: '["hello",{"foo":"bar"}]',
      })
    );
    expect(logSpy).toHaveBeenCalledWith('hello', { foo: 'bar' });
  });

  it('forwards console.warn to monitoring logger as warning', () => {
    restoreBridge = installStructuredConsoleBridge('unit-test.console');
    console.warn('warn-message', { context: true });

    expect(mockLogger.logWarning).toHaveBeenCalledWith(
      'warn-message',
      expect.objectContaining({
        level: 'warn',
      })
    );
    expect(warnSpy).toHaveBeenCalledWith('warn-message', { context: true });
  });

  it('forwards console.error to monitoring logger as error', () => {
    const error = new Error('boom');
    restoreBridge = installStructuredConsoleBridge('unit-test.console');
    console.error(error);

    expect(mockLogger.logError).toHaveBeenCalledWith(
      error,
      expect.objectContaining({
        level: 'error',
      })
    );
    expect(errorSpy).toHaveBeenCalledWith(error);
  });

  it('does not emit duplicate structured logs when console instrumentation is enabled', () => {
    (
      config.grafanaJavascriptAgent as { consoleInstrumentalizationEnabled: boolean }
    ).consoleInstrumentalizationEnabled = true;

    restoreBridge = installStructuredConsoleBridge('unit-test.console');
    console.log('no-duplicate');

    expect(mockLogger.logInfo).not.toHaveBeenCalled();
    expect(logSpy).toHaveBeenCalledWith('no-duplicate');
  });

  it('redacts sensitive values in remote context args', () => {
    restoreBridge = installStructuredConsoleBridge('unit-test.console');
    console.log({ userEmail: 'dev@example.com', token: 'abc', nested: { Authorization: 'Bearer 123' } });

    expect(mockLogger.logInfo).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        args: '[{"userEmail":"[REDACTED]","token":"[REDACTED]","nested":{"Authorization":"[REDACTED]"}}]',
      })
    );
  });

  it('does not emit structured logs when grafana javascript agent is disabled', () => {
    (config.grafanaJavascriptAgent as { enabled: boolean }).enabled = false;

    restoreBridge = installStructuredConsoleBridge('unit-test.console');
    console.log('agent-disabled');

    expect(mockLogger.logInfo).not.toHaveBeenCalled();
    expect(logSpy).toHaveBeenCalledWith('agent-disabled');
  });
});
