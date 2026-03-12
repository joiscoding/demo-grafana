import { EchoEventType, PageviewEchoEvent } from '@grafana/runtime';

import { BrowserConsoleBackend } from './BrowseConsoleBackend';

jest.mock('@grafana/runtime', () => {
  const actual = jest.requireActual('@grafana/runtime');
  const mockMonitoringLogger = {
    logInfo: jest.fn(),
    logWarning: jest.fn(),
    logError: jest.fn(),
    logDebug: jest.fn(),
    logMeasurement: jest.fn(),
  };

  return {
    ...actual,
    __mockMonitoringLogger: mockMonitoringLogger,
    createMonitoringLogger: jest.fn(() => mockMonitoringLogger),
  };
});

describe('BrowserConsoleBackend', () => {
  const { __mockMonitoringLogger: mockMonitoringLogger } = jest.requireMock('@grafana/runtime') as {
    __mockMonitoringLogger: {
      logInfo: jest.Mock;
      logWarning: jest.Mock;
      logError: jest.Mock;
      logDebug: jest.Mock;
      logMeasurement: jest.Mock;
    };
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('logs pageview events as structured info logs', () => {
    const backend = new BrowserConsoleBackend();

    backend.addEvent({
      type: EchoEventType.Pageview,
      payload: {
        page: '/explore',
      },
    } as PageviewEchoEvent);

    expect(mockMonitoringLogger.logInfo).toHaveBeenCalledWith('echo pageview event', {
      eventType: EchoEventType.Pageview,
      page: '/explore',
    });
  });

  it('logs interaction events and warns on invalid property types', () => {
    const backend = new BrowserConsoleBackend();

    backend.addEvent({
      type: EchoEventType.Interaction,
      payload: {
        interactionName: 'panel_click',
        properties: {
          valid: 'ok',
          invalid: {
            nested: true,
          },
        },
      },
    } as unknown as PageviewEchoEvent);

    expect(mockMonitoringLogger.logInfo).toHaveBeenCalledWith('echo interaction event', {
      eventType: EchoEventType.Interaction,
      interactionName: 'panel_click',
      properties: JSON.stringify({
        valid: 'ok',
        invalid: {
          nested: true,
        },
      }),
    });

    expect(mockMonitoringLogger.logWarning).toHaveBeenCalledWith('echo interaction event has invalid property types', {
      eventType: EchoEventType.Interaction,
      interactionName: 'panel_click',
      invalidProperties: JSON.stringify({
        invalid: {
          nested: true,
        },
      }),
    });
  });

  it('logs experiment events as structured info logs', () => {
    const backend = new BrowserConsoleBackend();

    backend.addEvent({
      type: EchoEventType.ExperimentView,
      payload: {
        experimentId: 'abc',
        experimentGroup: 'group_a',
        experimentVariant: 'variant_1',
      },
    } as unknown as PageviewEchoEvent);

    expect(mockMonitoringLogger.logInfo).toHaveBeenCalledWith('echo experiment event', {
      eventType: EchoEventType.ExperimentView,
      payload: JSON.stringify({
        experimentId: 'abc',
        experimentGroup: 'group_a',
        experimentVariant: 'variant_1',
      }),
    });
  });
});
