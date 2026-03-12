import { Correlation as CorrelationK8s } from '@grafana/api-clients/rtkq/correlations/v0alpha1';
import { SupportedTransformationType } from '@grafana/data';
import { getDataSourceSrv } from '@grafana/runtime';

import { CreateCorrelationParams } from './types';
import { fromK8sCorrelation, toCreateCorrelationResource } from './k8s';
import { buildCorrelationFieldSelector } from './utils';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getDataSourceSrv: jest.fn(),
}));

describe('correlations k8s mappings', () => {
  const getInstanceSettings = jest.fn();

  beforeEach(() => {
    getInstanceSettings.mockReset();
    (getDataSourceSrv as jest.Mock).mockReturnValue({
      getInstanceSettings,
    });
  });

  it('builds create payload using k8s resource spec', () => {
    getInstanceSettings.mockImplementation((value) => {
      if (value === 'source-uid') {
        return { uid: 'source-uid', type: 'loki' };
      }
      if (value === 'target-uid') {
        return { uid: 'target-uid', type: 'prometheus' };
      }
      return undefined;
    });

    const correlation: CreateCorrelationParams = {
      sourceUID: 'source-uid',
      targetUID: 'target-uid',
      label: 'logs to metrics',
      description: 'desc',
      type: 'query',
      config: {
        field: 'traceID',
        target: { expr: 'sum(rate(...))' },
        transformations: [
          {
            type: SupportedTransformationType.Logfmt,
          },
        ],
      },
    };

    const resource = toCreateCorrelationResource(correlation);

    expect(resource.apiVersion).toBe('correlations.grafana.app/v0alpha1');
    expect(resource.metadata.generateName).toBe('c');
    expect(resource.spec.source).toEqual({ group: 'loki', name: 'source-uid' });
    expect(resource.spec.target).toEqual({ group: 'prometheus', name: 'target-uid' });
    expect(resource.spec.config.transformations).toEqual([
      {
        type: 'logfmt',
        expression: '',
        field: '',
        mapValue: '',
      },
    ]);
  });

  it('converts k8s resource into correlation DTO', () => {
    getInstanceSettings.mockImplementation((value) => {
      if (value?.uid === 'source-uid') {
        return { uid: 'source-uid', type: 'loki' };
      }
      if (value?.uid === 'target-uid') {
        return { uid: 'target-uid', type: 'prometheus' };
      }
      return undefined;
    });

    const resource: CorrelationK8s = {
      apiVersion: 'correlations.grafana.app/v0alpha1',
      kind: 'Correlation',
      metadata: { name: 'corr-1' },
      spec: {
        type: 'query',
        label: 'logs to metrics',
        description: 'desc',
        source: { group: 'loki', name: 'source-uid' },
        target: { group: 'prometheus', name: 'target-uid' },
        config: {
          field: 'traceID',
          target: { expr: 'sum(rate(...))' },
        },
      },
    };

    expect(fromK8sCorrelation(resource)).toEqual({
      uid: 'corr-1',
      sourceUID: 'source-uid',
      targetUID: 'target-uid',
      label: 'logs to metrics',
      description: 'desc',
      type: 'query',
      provisioned: false,
      config: {
        field: 'traceID',
        target: { expr: 'sum(rate(...))' },
        transformations: undefined,
      },
    });
  });
});

describe('buildCorrelationFieldSelector', () => {
  it('returns undefined without source uids', () => {
    expect(buildCorrelationFieldSelector([])).toBeUndefined();
  });

  it('builds equals selector for one source uid', () => {
    expect(buildCorrelationFieldSelector(['source-uid'])).toBe('spec.datasource.name=source-uid');
  });

  it('builds in selector for many source uids', () => {
    expect(buildCorrelationFieldSelector(['uid-a', 'uid-b'])).toBe('spec.datasource.name in (uid-a;uid-b)');
  });
});
