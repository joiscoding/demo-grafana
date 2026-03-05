import { render, screen } from '@testing-library/react';
import { TestProvider } from 'test/helpers/TestProvider';

import config from 'app/core/config';

import LabsPage from './LabsPage';

describe('LabsPage', () => {
  const originalFeatureToggles = config.featureToggles;
  const originalNavTree = config.bootData.navTree;

  beforeEach(() => {
    config.bootData.navTree = [{ id: 'labs', text: 'Labs', url: '/labs' }];
  });

  afterEach(() => {
    config.featureToggles = originalFeatureToggles;
    config.bootData.navTree = originalNavTree;
  });

  it('renders feature flags from config in sorted order', () => {
    config.featureToggles = {
      panelTitleSearch: true,
      featureHighlights: false,
    };

    const { container } = render(
      <TestProvider>
        <LabsPage />
      </TestProvider>
    );

    expect(screen.getByRole('heading', { name: 'Labs' })).toBeInTheDocument();
    expect(screen.getByText('featureHighlights')).toBeInTheDocument();
    expect(screen.getByText('panelTitleSearch')).toBeInTheDocument();
    expect(container.textContent?.indexOf('featureHighlights')).toBeLessThan(
      container.textContent?.indexOf('panelTitleSearch') ?? -1
    );
  });

  it('shows empty state when no feature flags are enabled', () => {
    config.featureToggles = {};

    render(
      <TestProvider>
        <LabsPage />
      </TestProvider>
    );

    expect(screen.getByText('No feature flags are enabled.')).toBeInTheDocument();
  });
});
