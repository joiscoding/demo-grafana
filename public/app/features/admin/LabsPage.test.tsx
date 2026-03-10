import { screen, waitFor } from '@testing-library/react';
import { render } from 'test/test-utils';

import LabsPage from './LabsPage';

const getMock = jest.fn();

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: () => ({ get: getMock }),
}));

const renderPage = () =>
  render(<LabsPage />, {
    preloadedState: {
      navIndex: {
        labs: {
          id: 'labs',
          text: 'Labs',
          url: '/admin/labs',
          parentItem: {
            id: 'cfg',
            text: 'Administration',
            url: '/admin',
            children: [{ id: 'labs', text: 'Labs', url: '/admin/labs' }],
          },
        },
      },
    },
  });

describe('LabsPage', () => {
  beforeEach(() => {
    getMock.mockResolvedValue({
      toggles: [
        {
          name: 'cloudWatchCrossAccountQuerying',
          description: 'Enables cross-account querying in CloudWatch datasources',
          enabled: true,
          stage: 'GA',
        },
        {
          name: 'publicDashboardsEmailSharing',
          description: 'Restrict public dashboard sharing to only allowed emails',
          enabled: false,
          stage: 'preview',
        },
      ],
    });
  });

  afterEach(() => {
    getMock.mockReset();
  });

  it('renders current feature flag states and filters them by query', async () => {
    const { user } = renderPage();

    await waitFor(() => expect(getMock).toHaveBeenCalledWith('/api/admin/feature-toggles'));

    expect(await screen.findByText('cloudWatchCrossAccountQuerying')).toBeInTheDocument();
    expect(screen.getByText('publicDashboardsEmailSharing')).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText('Search feature flags by name, stage, or description'), 'cloud');

    await waitFor(() => {
      expect(screen.getByText('cloudWatchCrossAccountQuerying')).toBeInTheDocument();
      expect(screen.queryByText('publicDashboardsEmailSharing')).not.toBeInTheDocument();
    });
  });
});
