import { config, createMonitoringLogger, registerEchoBackend, setEchoSrv } from '@grafana/runtime';
import { reportMetricPerformanceMark } from 'app/core/utils/metrics';

import { contextSrv } from '../context_srv';

import { Echo } from './Echo';

const echoInitLogger = createMonitoringLogger('EchoSrv.init');

function toError(error: unknown, backendName: string): Error {
  if (error instanceof Error) {
    return error;
  }

  return new Error(`Error initializing EchoSrv ${backendName} backend`);
}

function toErrorString(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  if (typeof error === 'string') {
    return error;
  }

  return String(error);
}

// Initialise EchoSrv backends, calls during frontend app startup
export async function initEchoSrv() {
  setEchoSrv(new Echo({ debug: process.env.NODE_ENV === 'development' }));

  window.addEventListener('load', (e) => {
    const loadMetricName = 'frontend_boot_load_time_seconds';
    // Metrics below are marked in public/views/index.html
    const jsLoadMetricName = 'frontend_boot_js_done_time_seconds';
    const cssLoadMetricName = 'frontend_boot_css_time_seconds';

    if (performance) {
      performance.mark(loadMetricName);
      reportMetricPerformanceMark('first-paint', 'frontend_boot_', '_time_seconds');
      reportMetricPerformanceMark('first-contentful-paint', 'frontend_boot_', '_time_seconds');
      reportMetricPerformanceMark(loadMetricName);
      reportMetricPerformanceMark(jsLoadMetricName);
      reportMetricPerformanceMark(cssLoadMetricName);
    }
  });

  try {
    await initPerformanceBackend();
  } catch (error) {
    const err = toError(error, 'Performance');
    console.error('[EchoSrv] Failed to init Performance backend:', err);
    echoInitLogger.logError(err, { backend: 'Performance', error: toErrorString(error) });
  }

  try {
    await initFaroBackend();
  } catch (error) {
    const err = toError(error, 'Faro');
    console.error('[EchoSrv] Failed to init Faro backend:', err);
    echoInitLogger.logError(err, { backend: 'Faro', error: toErrorString(error) });
  }

  try {
    await initGoogleAnalyticsBackend();
  } catch (error) {
    const err = toError(error, 'GoogleAnalytics');
    console.error('[EchoSrv] Failed to init GoogleAnalytics backend:', err);
    echoInitLogger.logError(err, {
      backend: 'GoogleAnalytics',
      error: toErrorString(error),
    });
  }

  try {
    await initGoogleAnalaytics4Backend();
  } catch (error) {
    const err = toError(error, 'GoogleAnalytics4');
    console.error('[EchoSrv] Failed to init GoogleAnalytics4 backend:', err);
    echoInitLogger.logError(err, {
      backend: 'GoogleAnalytics4',
      error: toErrorString(error),
    });
  }

  try {
    await initRudderstackBackend();
  } catch (error) {
    const err = toError(error, 'Rudderstack');
    console.error('[EchoSrv] Failed to init Rudderstack backend:', err);
    echoInitLogger.logError(err, { backend: 'Rudderstack', error: toErrorString(error) });
  }

  try {
    await initAzureAppInsightsBackend();
  } catch (error) {
    const err = toError(error, 'AzureAppInsights');
    console.error('[EchoSrv] Failed to init AzureAppInsights backend:', err);
    echoInitLogger.logError(err, {
      backend: 'AzureAppInsights',
      error: toErrorString(error),
    });
  }

  try {
    await initConsoleBackend();
  } catch (error) {
    const err = toError(error, 'Console');
    console.error('[EchoSrv] Failed to init Console backend:', err);
    echoInitLogger.logError(err, { backend: 'Console', error: toErrorString(error) });
  }
}

async function initPerformanceBackend() {
  if (contextSrv.user.orgRole === '') {
    return;
  }

  const { PerformanceBackend } = await import('./backends/PerformanceBackend');
  registerEchoBackend(new PerformanceBackend({}));
}

async function initFaroBackend() {
  if (!config.grafanaJavascriptAgent.enabled) {
    return;
  }

  // Ignore Rudderstack URLs
  const rudderstackUrls = [
    config.rudderstackConfigUrl,
    config.rudderstackDataPlaneUrl,
    config.rudderstackIntegrationsUrl,
  ]
    .filter(Boolean)
    .map((url) => new RegExp(`${url}.*.`));

  const { GrafanaJavascriptAgentBackend } = await import(
    './backends/grafana-javascript-agent/GrafanaJavascriptAgentBackend'
  );

  registerEchoBackend(
    new GrafanaJavascriptAgentBackend({
      buildInfo: config.buildInfo,
      userIdentifier: contextSrv.user.analytics.identifier,
      ignoreUrls: rudderstackUrls,

      apiKey: config.grafanaJavascriptAgent.apiKey,
      customEndpoint: config.grafanaJavascriptAgent.customEndpoint,
      consoleInstrumentalizationEnabled: config.grafanaJavascriptAgent.consoleInstrumentalizationEnabled,
      performanceInstrumentalizationEnabled: config.grafanaJavascriptAgent.performanceInstrumentalizationEnabled,
      cspInstrumentalizationEnabled: config.grafanaJavascriptAgent.cspInstrumentalizationEnabled,
      tracingInstrumentalizationEnabled: config.grafanaJavascriptAgent.tracingInstrumentalizationEnabled,
      webVitalsAttribution: config.grafanaJavascriptAgent.webVitalsAttribution,
      internalLoggerLevel: config.grafanaJavascriptAgent.internalLoggerLevel,
      botFilterEnabled: config.grafanaJavascriptAgent.botFilterEnabled,
    })
  );
}

async function initGoogleAnalyticsBackend() {
  if (!config.googleAnalyticsId) {
    return;
  }

  const { GAEchoBackend } = await import('./backends/analytics/GABackend');
  registerEchoBackend(
    new GAEchoBackend({
      googleAnalyticsId: config.googleAnalyticsId,
    })
  );
}

async function initGoogleAnalaytics4Backend() {
  if (!config.googleAnalytics4Id) {
    return;
  }

  const { GA4EchoBackend } = await import('./backends/analytics/GA4Backend');
  registerEchoBackend(
    new GA4EchoBackend({
      googleAnalyticsId: config.googleAnalytics4Id,
      googleAnalytics4SendManualPageViews: config.googleAnalytics4SendManualPageViews,
    })
  );
}

async function initRudderstackBackend() {
  if (!(config.rudderstackWriteKey && config.rudderstackDataPlaneUrl)) {
    return;
  }

  // Logic: if only one of the sdk urls is provided, use respective code
  // otherwise defer to the feature toggle.

  const hasOldSdkUrl = Boolean(config.rudderstackSdkUrl);
  const hasNewSdkUrl = Boolean(config.rudderstackV3SdkUrl);
  const onlyOneSdkUrlSet = hasOldSdkUrl !== hasNewSdkUrl;
  const useNewRudderstack = onlyOneSdkUrlSet ? hasNewSdkUrl : config.featureToggles.rudderstackUpgrade;

  const sdkUrl = useNewRudderstack ? config.rudderstackV3SdkUrl : config.rudderstackSdkUrl;

  const modulePromise = useNewRudderstack
    ? import('./backends/analytics/RudderstackV3Backend')
    : import('./backends/analytics/RudderstackBackend');

  const { RudderstackBackend } = await modulePromise;
  registerEchoBackend(
    new RudderstackBackend({
      writeKey: config.rudderstackWriteKey,
      dataPlaneUrl: config.rudderstackDataPlaneUrl,
      user: contextSrv.user,
      sdkUrl,
      configUrl: config.rudderstackConfigUrl,
      integrationsUrl: config.rudderstackIntegrationsUrl,
      buildInfo: config.buildInfo,
    })
  );
}

async function initAzureAppInsightsBackend() {
  if (!config.applicationInsightsConnectionString) {
    return;
  }

  const { ApplicationInsightsBackend } = await import('./backends/analytics/ApplicationInsightsBackend');
  registerEchoBackend(
    new ApplicationInsightsBackend({
      connectionString: config.applicationInsightsConnectionString,
      endpointUrl: config.applicationInsightsEndpointUrl,
      autoRouteTracking: config.applicationInsightsAutoRouteTracking,
    })
  );
}

async function initConsoleBackend() {
  if (!config.analyticsConsoleReporting) {
    return;
  }

  const { BrowserConsoleBackend } = await import('./backends/analytics/BrowseConsoleBackend');
  registerEchoBackend(new BrowserConsoleBackend());
}
