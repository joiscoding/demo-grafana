import { EchoEventType, PageviewEchoEvent } from '@grafana/runtime';

import { BrowserConsoleBackend } from './BrowseConsoleBackend';

describe('BrowserConsoleBackend', () => {
  beforeEach(() => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('logs pageview events as structured info logs', () => {
    const backend = new BrowserConsoleBackend();

    backend.addEvent({
      type: EchoEventType.Pageview,
      payload: {
        page: '/explore',
      },
    } as PageviewEchoEvent);

    expect(console.log).toHaveBeenCalledWith('[EchoSrv] echo pageview event', {
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

    expect(console.log).toHaveBeenCalledWith('[EchoSrv] echo interaction event', {
      eventType: EchoEventType.Interaction,
      interactionName: 'panel_click',
      properties: '{"valid":"ok","invalid":{"nested":true}}',
    });

    expect(console.warn).toHaveBeenCalledWith('[EchoSrv] echo interaction event has invalid property types', {
      eventType: EchoEventType.Interaction,
      interactionName: 'panel_click',
      invalidProperties: '{"invalid":{"nested":true}}',
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

    expect(console.log).toHaveBeenCalledWith('[EchoSrv] echo experiment event', {
      eventType: EchoEventType.ExperimentView,
      payload: '{"experimentId":"abc","experimentGroup":"group_a","experimentVariant":"variant_1"}',
    });
  });
});
