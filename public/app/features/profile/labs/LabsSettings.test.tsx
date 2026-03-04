import { render, screen, userEvent, waitFor, within } from 'test/test-utils';

import { config } from '@grafana/runtime';
import { ScopedResourceClient } from 'app/features/apiserver/client';
import { ResourceList } from 'app/features/apiserver/types';

import { LabsSettings } from './LabsSettings';
import { featureToggleOverridesLocalStorageKey } from './localFeatureToggles';

describe('LabsSettings', () => {
  const originalFeatureToggles = config.featureToggles;

  beforeEach(() => {
    config.featureToggles = {
      dashboardScene: true,
    };
    window.localStorage.clear();

    const featureListResponse: ResourceList<{ description?: string; stage?: string; frontend?: boolean }, object> = {
      apiVersion: 'featuretoggle.grafana.app/v0alpha1',
      kind: 'FeatureList',
      metadata: {
        resourceVersion: '1',
      },
      items: [
        {
          apiVersion: 'featuretoggle.grafana.app/v0alpha1',
          kind: 'Feature',
          metadata: { name: 'dashboardScene' },
          spec: {
            description: 'Use scenes-based dashboards',
            stage: 'preview',
          },
        },
        {
          apiVersion: 'featuretoggle.grafana.app/v0alpha1',
          kind: 'Feature',
          metadata: { name: 'newLogsPanel' },
          spec: {
            description: 'Enable new logs panel rendering',
            stage: 'experimental',
            frontend: true,
          },
        },
      ],
    };

    jest.spyOn(ScopedResourceClient.prototype, 'list').mockResolvedValue(featureListResponse);
  });

  afterEach(() => {
    config.featureToggles = originalFeatureToggles;
    jest.restoreAllMocks();
  });

  it('renders toggle metadata when available', async () => {
    render(<LabsSettings />);

    expect(await screen.findByTestId('labs-toggle-row-dashboardScene')).toBeInTheDocument();
    expect(await screen.findByTestId('labs-toggle-row-newLogsPanel')).toBeInTheDocument();
    expect(await screen.findByText('Use scenes-based dashboards')).toBeInTheDocument();
    expect(await screen.findByText('Enable new logs panel rendering')).toBeInTheDocument();
  });

  it('writes local browser overrides when toggles change', async () => {
    render(<LabsSettings />);

    const dashboardSceneRow = await screen.findByTestId('labs-toggle-row-dashboardScene');
    const dashboardSceneSwitch = within(dashboardSceneRow).getByTestId('labs-toggle-switch-dashboardScene');

    await userEvent.click(dashboardSceneSwitch);

    await waitFor(() => {
      expect(window.localStorage.getItem(featureToggleOverridesLocalStorageKey)).toBe('dashboardScene=false');
    });
  });
});
