import { EchoEventType, InteractionEchoEvent } from '@grafana/runtime';

import { GA4EchoBackend } from './GA4Backend';

jest.mock('../../utils', () => ({
  loadScript: jest.fn(),
}));

describe('GA4EchoBackend', () => {
  beforeEach(() => {
    window.dataLayer = [];
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('forwards interaction properties without a hardcoded staging category', () => {
    const backend = new GA4EchoBackend({
      googleAnalyticsId: 'GA-123',
      googleAnalytics4SendManualPageViews: false,
    });
    window.gtag = jest.fn();

    backend.addEvent({
      type: EchoEventType.Interaction,
      payload: {
        interactionName: 'grafana_dashboard_saved',
        properties: { source: 'toolbar', success: true },
      },
      meta: {} as InteractionEchoEvent['meta'],
    });

    expect(window.gtag).toHaveBeenCalledWith('event', 'grafana_dashboard_saved', {
      source: 'toolbar',
      success: true,
    });
  });

  it('does not emit page views for interaction events', () => {
    const backend = new GA4EchoBackend({
      googleAnalyticsId: 'GA-123',
      googleAnalytics4SendManualPageViews: true,
    });
    window.gtag = jest.fn();

    backend.addEvent({
      type: EchoEventType.Interaction,
      payload: {
        interactionName: 'grafana_dashboard_saved',
        properties: { source: 'toolbar' },
      },
      meta: {} as InteractionEchoEvent['meta'],
    });

    expect(window.gtag).not.toHaveBeenCalledWith('event', 'page_view', expect.anything());
    expect(window.gtag).toHaveBeenCalledWith('event', 'grafana_dashboard_saved', { source: 'toolbar' });
  });
});
