import { render, screen } from 'test/test-utils';

import LabsPage from './LabsPage';

const getMock = jest.fn();

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: () => ({ get: getMock }),
}));

describe('LabsPage', () => {
  beforeEach(() => {
    getMock.mockResolvedValue({
      toggles: [
        {
          name: 'panelTitleSearch',
          description: 'Search dashboards using panel title',
          stage: 'experimental',
          enabled: true,
          frontendOnly: false,
          requiresRestart: false,
        },
        {
          name: 'storage',
          description: 'Configurable storage for dashboards',
          stage: 'preview',
          enabled: false,
          frontendOnly: false,
          requiresRestart: true,
        },
      ],
    });
  });

  afterEach(() => {
    getMock.mockReset();
  });

  it('renders feature flags and their current status', async () => {
    render(<LabsPage />);

    expect(await screen.findByText('panelTitleSearch')).toBeInTheDocument();
    expect(screen.getByText('storage')).toBeInTheDocument();
    expect(screen.getByText(/This page lists 2 feature flags/i)).toBeInTheDocument();
    expect(screen.getByText('Enabled')).toBeInTheDocument();
    expect(screen.getByText('Disabled')).toBeInTheDocument();
    expect(screen.getByText('Yes')).toBeInTheDocument();
  });

  it('filters feature flags by query', async () => {
    const { user } = render(<LabsPage />);

    await screen.findByText('panelTitleSearch');
    await user.type(screen.getByPlaceholderText(/search feature flags/i), 'storage');

    expect(screen.getByText('storage')).toBeInTheDocument();
    expect(screen.queryByText('panelTitleSearch')).not.toBeInTheDocument();
  });
});
