import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { selectors as e2eSelectors } from '@grafana/e2e-selectors';

import { createDashboardModelFixture } from '../../state/__fixtures__/dashboardFixtures';

import { ShareSnapshot } from './ShareSnapshot';

const STORAGE_KEY = 'grafana.share.snapshot.latest';
const selectors = e2eSelectors.pages.ShareDashboardModal.SnapshotScene;

const getByDeleteURL = jest.fn();
const getSharingOptions = jest.fn().mockResolvedValue({
  externalEnabled: false,
  externalSnapshotName: 'Publish to snapshots.raintank.io',
  externalSnapshotURL: '',
  snapshotEnabled: true,
});

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: () => ({
    get: getByDeleteURL,
  }),
}));

jest.mock('app/features/dashboard/services/SnapshotSrv', () => ({
  getDashboardSnapshotSrv: () => ({
    getSharingOptions,
    create: jest.fn(),
    getSnapshots: jest.fn(),
    getSnapshot: jest.fn(),
    deleteSnapshot: jest.fn(),
  }),
}));

describe('ShareSnapshot', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    window.sessionStorage.clear();
  });

  it('restores an existing copied snapshot link and can disable it from the modal', async () => {
    const dashboard = createDashboardModelFixture({
      id: 1,
      uid: 'dash-uid',
      title: 'My Dashboard',
    });

    window.sessionStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        storageId: 'dash-uid:all',
        snapshotUrl: 'https://grafana.example/dashboard/snapshot/snapshot-key',
        deleteUrl: '/api/snapshots-delete/snapshot-key',
      })
    );

    render(<ShareSnapshot dashboard={dashboard} onDismiss={jest.fn()} />);

    const snapshotURLInput = await screen.findByTestId(selectors.CopyUrlInput);
    expect(snapshotURLInput).toHaveValue('https://grafana.example/dashboard/snapshot/snapshot-key');
    expect(screen.getByText('Disable link')).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByTestId('snapshot-disable-link-button'));

    await waitFor(() => {
      expect(getByDeleteURL).toHaveBeenCalledWith('/api/snapshots-delete/snapshot-key');
    });

    expect(await screen.findByText(/The snapshot has been deleted/)).toBeInTheDocument();
    expect(window.sessionStorage.getItem(STORAGE_KEY)).toBeNull();
  });
});
