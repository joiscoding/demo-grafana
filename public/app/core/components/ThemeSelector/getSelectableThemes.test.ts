import { config } from '@grafana/runtime';

import { getSelectableThemes } from './getSelectableThemes';

describe('getSelectableThemes', () => {
  const originalGrafanaconThemes = config.featureToggles.grafanaconThemes;

  afterEach(() => {
    config.featureToggles.grafanaconThemes = originalGrafanaconThemes;
  });

  it('includes 90s mode when grafanacon themes are enabled', () => {
    config.featureToggles.grafanaconThemes = true;

    expect(getSelectableThemes().map((theme) => theme.id)).toContain('90smode');
  });

  it('hides 90s mode when grafanacon themes are disabled', () => {
    config.featureToggles.grafanaconThemes = false;

    expect(getSelectableThemes().map((theme) => theme.id)).not.toContain('90smode');
  });
});
