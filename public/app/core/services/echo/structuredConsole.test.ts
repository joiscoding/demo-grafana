const logger = {
  logDebug: jest.fn(),
  logInfo: jest.fn(),
  logWarning: jest.fn(),
  logError: jest.fn(),
  logMeasurement: jest.fn(),
};

jest.mock('@grafana/runtime', () => ({
  createMonitoringLogger: jest.fn(() => logger),
}));

jest.mock('app/core/config', () => ({
  __esModule: true,
  default: {
    grafanaJavascriptAgent: {
      enabled: false,
      consoleInstrumentalizationEnabled: false,
    },
  },
}));

import config from 'app/core/config';

import { enableStructuredConsoleForwarding } from './structuredConsole';

describe('enableStructuredConsoleForwarding', () => {
  const originalConsole = {
    log: console.log,
    info: console.info,
    warn: console.warn,
    error: console.error,
  };

  beforeEach(() => {
    jest.clearAllMocks();
    window.__grafana_console_structured_forwarding__ = false;
    window.__grafana_console_buffer__ = [];
    window.__grafana_console_original__ = undefined;
    (config as { grafanaJavascriptAgent: { enabled: boolean; consoleInstrumentalizationEnabled: boolean } })
      .grafanaJavascriptAgent.enabled = false;
    (config as { grafanaJavascriptAgent: { enabled: boolean; consoleInstrumentalizationEnabled: boolean } })
      .grafanaJavascriptAgent.consoleInstrumentalizationEnabled = false;
  });

  afterEach(() => {
    console.log = originalConsole.log;
    console.info = originalConsole.info;
    console.warn = originalConsole.warn;
    console.error = originalConsole.error;
  });

  it('flushes buffered console entries and forwards new logs', () => {
    const originalWarn = jest.fn();

    window.__grafana_console_buffer__ = [{ level: 'error', args: ['boot failure'], timestamp: 1000 }];
    window.__grafana_console_original__ = {
      log: jest.fn(),
      info: jest.fn(),
      warn: originalWarn,
      error: jest.fn(),
      debug: jest.fn(),
      trace: jest.fn(),
    };

    enableStructuredConsoleForwarding();

    expect(logger.logError).toHaveBeenCalledWith(expect.any(Error), expect.objectContaining({ timestamp: 1000 }));

    console.warn('runtime warning', { id: 42 });

    expect(logger.logWarning).toHaveBeenCalledWith(
      'runtime warning',
      expect.objectContaining({ argumentCount: 2, arguments: ['runtime warning', { id: 42 }] })
    );
    expect(originalWarn).toHaveBeenCalledWith('runtime warning', { id: 42 });
  });

  it('does not replace active console instrumentation when faro instrumentation is enabled', () => {
    const originalInfo = jest.fn();
    const instrumentedInfo = jest.fn();

    window.__grafana_console_buffer__ = [{ level: 'info', args: ['buffered info'], timestamp: 1001 }];
    window.__grafana_console_original__ = {
      log: jest.fn(),
      info: originalInfo,
      warn: jest.fn(),
      error: jest.fn(),
      debug: jest.fn(),
      trace: jest.fn(),
    };
    console.info = instrumentedInfo;

    (config as { grafanaJavascriptAgent: { enabled: boolean; consoleInstrumentalizationEnabled: boolean } })
      .grafanaJavascriptAgent.enabled = true;
    (config as { grafanaJavascriptAgent: { enabled: boolean; consoleInstrumentalizationEnabled: boolean } })
      .grafanaJavascriptAgent.consoleInstrumentalizationEnabled = true;

    enableStructuredConsoleForwarding();
    console.info('live info');

    expect(logger.logInfo).toHaveBeenCalledTimes(1);
    expect(logger.logInfo).toHaveBeenCalledWith('buffered info', expect.objectContaining({ timestamp: 1001 }));
    expect(instrumentedInfo).toHaveBeenCalledWith('live info');
    expect(originalInfo).not.toHaveBeenCalled();
  });
});
