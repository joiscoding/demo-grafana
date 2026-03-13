import { CurrentUserDTO } from '@grafana/data';
import {
  EchoBackend,
  EchoEventType,
  InteractionEchoEvent,
  isInteractionEvent,
  isPageviewEvent,
  PageviewEchoEvent,
} from '@grafana/runtime';

import { loadScript } from '../../utils';

declare global {
  interface Window {
    dataLayer: unknown[];
  }
}

export interface GA4EchoBackendOptions {
  googleAnalyticsId: string;
  googleAnalytics4SendManualPageViews: boolean;
  user?: CurrentUserDTO;
}

export class GA4EchoBackend implements EchoBackend<PageviewEchoEvent | InteractionEchoEvent, GA4EchoBackendOptions> {
  supportedEvents = [EchoEventType.Pageview, EchoEventType.Interaction];
  googleAnalytics4SendManualPageViews = false;

  constructor(public options: GA4EchoBackendOptions) {
    const url = `https://www.googletagmanager.com/gtag/js?id=${options.googleAnalyticsId}`;
    loadScript(url, true);

    window.dataLayer = window.dataLayer || [];
    window.gtag = function gtag() {
      window.dataLayer.push(arguments);
    };
    window.gtag('js', new Date());

    const configOptions: Gtag.CustomParams = {
      page_path: window.location.pathname,
    };

    if (options.user) {
      configOptions.user_id = options.user.analytics.identifier;
    }

    this.googleAnalytics4SendManualPageViews = options.googleAnalytics4SendManualPageViews;
    window.gtag('config', options.googleAnalyticsId, configOptions);
  }

  addEvent = (e: PageviewEchoEvent | InteractionEchoEvent) => {
    if (!window.gtag) {
      return;
    }

    // this should prevent duplicate events in case enhanced tracking is enabled
    if (isPageviewEvent(e) && this.googleAnalytics4SendManualPageViews) {
      window.gtag('event', 'page_view', { page_path: e.payload.page });
    }

    if (isInteractionEvent(e)) {
      window.gtag('event', e.payload.interactionName, e.payload.properties);
    }
  };

  // Not using Echo buffering, addEvent above sends events to GA as soon as they appear
  flush = () => {};
}
