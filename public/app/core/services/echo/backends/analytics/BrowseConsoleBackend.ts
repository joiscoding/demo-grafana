import {
  EchoBackend,
  EchoEventType,
  isExperimentViewEvent,
  isInteractionEvent,
  isPageviewEvent,
  PageviewEchoEvent,
  createMonitoringLogger,
} from '@grafana/runtime';

const logger = createMonitoringLogger('EchoSrv.BrowserConsoleBackend');

function toContextValue(value: unknown): string {
  if (value === undefined) {
    return 'undefined';
  }

  return typeof value === 'string' ? value : JSON.stringify(value);
}

export class BrowserConsoleBackend implements EchoBackend<PageviewEchoEvent, unknown> {
  options = {};
  supportedEvents = [EchoEventType.Pageview, EchoEventType.Interaction, EchoEventType.ExperimentView];

  constructor() {}

  addEvent = (e: PageviewEchoEvent) => {
    if (isPageviewEvent(e)) {
      logger.logInfo('echo pageview event', {
        eventType: EchoEventType.Pageview,
        page: e.payload.page,
      });
    }

    if (isInteractionEvent(e)) {
      const eventName = e.payload.interactionName;
      logger.logInfo('echo interaction event', {
        eventType: EchoEventType.Interaction,
        interactionName: eventName,
        properties: toContextValue(e.payload.properties),
      });

      // Warn for non-scalar property values. We're not yet making this a hard a
      const invalidTypeProperties = Object.entries(e.payload.properties ?? {}).filter(([_, value]) => {
        const valueType = typeof value;
        const isValidType =
          valueType === 'string' || valueType === 'number' || valueType === 'boolean' || valueType === 'undefined';
        return !isValidType;
      });

      if (invalidTypeProperties.length > 0) {
        logger.logWarning('echo interaction event has invalid property types', {
          eventType: EchoEventType.Interaction,
          interactionName: eventName,
          invalidProperties: toContextValue(Object.fromEntries(invalidTypeProperties)),
        });
      }
    }

    if (isExperimentViewEvent(e)) {
      logger.logInfo('echo experiment event', {
        eventType: EchoEventType.ExperimentView,
        payload: toContextValue(e.payload),
      });
    }
  };

  flush = () => {};
}
