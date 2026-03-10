import { render, screen, waitFor, within } from 'test/test-utils';

import { config } from '@grafana/runtime';

import LabsPage from './LabsPage';

const postMock = jest.fn();

jest.mock('@grafana/runtime', () => {
  const runtime = jest.requireActual('@grafana/runtime');

  return {
    ...runtime,
    getBackendSrv: () => ({
      post: postMock,
    }),
  };
});

describe('LabsPage', () => {
  const originalNamespace = config.namespace;
  const originalOpenFeatureContext = config.openFeatureContext;
  const originalFeatureToggles = config.featureToggles;

  beforeEach(() => {
    postMock.mockReset();
    config.namespace = 'default';
    config.openFeatureContext = { stack: 'dev' };
    config.featureToggles = {};
  });

  afterAll(() => {
    config.namespace = originalNamespace;
    config.openFeatureContext = originalOpenFeatureContext;
    config.featureToggles = originalFeatureToggles;
  });

  it('renders evaluated flags and highlights runtime overrides', async () => {
    postMock.mockResolvedValue({
      flags: [
        { key: 'zetaFeature', value: false },
        { key: 'alphaFeature', value: false },
        { key: 'betaFeature', value: true },
      ],
    });

    Reflect.set(config.featureToggles, 'alphaFeature', true);
    Reflect.set(config.featureToggles, 'betaFeature', false);

    render(<LabsPage />);

    await waitFor(() => {
      expect(postMock).toHaveBeenCalledWith(
        '/apis/features.grafana.app/v0alpha1/namespaces/default/ofrep/v1/evaluate/flags',
        {
          context: {
            targetingKey: 'default',
            namespace: 'default',
            stack: 'dev',
          },
        }
      );
    });

    const table = await screen.findByRole('table');
    const tableBody = within(table).getAllByRole('rowgroup')[1];
    const rows = within(tableBody).getAllByRole('row');

    expect(rows).toHaveLength(3);
    expect(rows[0]).toHaveTextContent('alphaFeature');
    expect(rows[0]).toHaveTextContent('Enabled');
    expect(rows[0]).toHaveTextContent('Runtime override');
    expect(rows[1]).toHaveTextContent('betaFeature');
    expect(rows[1]).toHaveTextContent('Disabled');
    expect(rows[1]).toHaveTextContent('Runtime override');
    expect(rows[2]).toHaveTextContent('zetaFeature');
    expect(rows[2]).toHaveTextContent('Disabled');

    expect(screen.getByText('1 enabled')).toBeInTheDocument();
    expect(screen.getByText('3 total flags')).toBeInTheDocument();
    expect(screen.getByText('2 runtime overrides')).toBeInTheDocument();
  });

  it('shows an error when the flag request fails', async () => {
    postMock.mockRejectedValue(new Error('request failed'));

    render(<LabsPage />);

    expect(await screen.findByText('Unable to load feature flags')).toBeInTheDocument();
    expect(await screen.findByText('request failed')).toBeInTheDocument();
  });
});
