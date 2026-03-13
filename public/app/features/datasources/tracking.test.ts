import { reportInteraction } from '@grafana/runtime';

import { trackDataSourceTested } from './tracking';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  reportInteraction: jest.fn(),
}));

describe('trackDataSourceTested', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('includes the success property in the interaction payload', () => {
    const props = {
      datasource_uid: 'test-uid',
      plugin_id: 'prometheus',
      plugin_version: '1.2.3',
      grafana_version: '11.0.0',
      success: true,
      path: '/datasources/edit/prometheus',
    };

    trackDataSourceTested(props);

    expect(reportInteraction).toHaveBeenCalledWith('grafana_ds_test_datasource_clicked', props);
  });
});
