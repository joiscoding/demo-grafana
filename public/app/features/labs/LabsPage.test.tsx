import { render, screen } from 'test/test-utils';

import { config } from '@grafana/runtime';

import LabsPage from './LabsPage';

describe('LabsPage', () => {
  let initialFeatureToggles: typeof config.featureToggles;

  beforeEach(() => {
    initialFeatureToggles = { ...config.featureToggles };
    window.localStorage.clear();
  });

  afterEach(() => {
    config.featureToggles = initialFeatureToggles;
    window.localStorage.clear();
  });

  it('shows enabled runtime feature flags', () => {
    config.featureToggles = {
      dashboardScene: true,
      queryServiceFromUI: true,
    };

    render(<LabsPage />);

    expect(screen.getByText('dashboardScene')).toBeInTheDocument();
    expect(screen.getByText('queryServiceFromUI')).toBeInTheDocument();
    expect(screen.getByText('2 enabled')).toBeInTheDocument();
  });

  it('loads and persists feature flag overrides to localStorage', async () => {
    config.featureToggles = {
      queryServiceFromUI: true,
    };
    window.localStorage.setItem('grafana.featureToggles', 'queryServiceFromUI=false');

    const { user } = render(<LabsPage />);
    const toggle = screen.getByTestId('labs-feature-toggle-queryServiceFromUI');
    const isEnabled = () =>
      toggle instanceof HTMLInputElement ? toggle.checked : toggle.getAttribute('aria-checked') === 'true';

    expect(isEnabled()).toBe(false);

    await user.click(toggle);

    expect(isEnabled()).toBe(true);
    expect(window.localStorage.getItem('grafana.featureToggles')).toContain('queryServiceFromUI=true');
  });
});
