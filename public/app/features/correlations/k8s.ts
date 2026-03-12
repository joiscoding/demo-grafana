import {
  Correlation as CorrelationK8s,
  CorrelationSpec,
  CorrelationTransformationSpec,
} from '@grafana/api-clients/rtkq/correlations/v0alpha1';
import { SupportedTransformationType } from '@grafana/data';
import { CorrelationExternal, CorrelationQuery, getDataSourceSrv } from '@grafana/runtime';
import { getAPIBaseURL } from 'app/api/utils';

import { Correlation, CreateCorrelationParams, UpdateCorrelationParams } from './types';

export const CORRELATIONS_API_BASE_URL = getAPIBaseURL('correlations.grafana.app', 'v0alpha1');

type CorrelationInput = CreateCorrelationParams | UpdateCorrelationParams;

function getDataSourceRef(uid: string) {
  const dataSource = getDataSourceSrv().getInstanceSettings(uid);
  if (!dataSource?.uid || !dataSource.type) {
    throw new Error(`Data source "${uid}" is unavailable`);
  }

  return {
    group: dataSource.type,
    name: dataSource.uid,
  };
}

function normalizeTransformations(input: CorrelationInput) {
  return input.config.transformations?.map((transformation): CorrelationTransformationSpec => ({
    ...transformation,
    type:
      transformation.type === SupportedTransformationType.Logfmt
        ? SupportedTransformationType.Logfmt
        : SupportedTransformationType.Regex,
    expression: transformation.expression ?? '',
    field: transformation.field ?? '',
    mapValue: transformation.mapValue ?? '',
  }));
}

export function toCorrelationSpec(input: CorrelationInput): CorrelationSpec {
  const spec: CorrelationSpec = {
    type: input.type,
    label: input.label ?? '',
    description: input.description,
    source: getDataSourceRef(input.sourceUID),
    config: {
      field: input.config.field,
      target: input.config.target,
      transformations: normalizeTransformations(input),
    },
  };

  if (input.type === 'query' && 'targetUID' in input && input.targetUID) {
    spec.target = getDataSourceRef(input.targetUID);
  }

  return spec;
}

export function toCreateCorrelationResource(input: CreateCorrelationParams): CorrelationK8s {
  const resource: CorrelationK8s = {
    apiVersion: 'correlations.grafana.app/v0alpha1',
    kind: 'Correlation',
    metadata: {
      generateName: 'c',
    },
    spec: toCorrelationSpec(input),
  };

  if (input.type === 'query' && !resource.spec.target) {
    throw new Error('Query correlations require a target data source');
  }

  return resource;
}

export function toUpdateCorrelationPatch(input: UpdateCorrelationParams) {
  return {
    spec: toCorrelationSpec(input),
  };
}

export function fromK8sCorrelation(resource: CorrelationK8s): Correlation | undefined {
  const sourceDataSource = getDataSourceSrv().getInstanceSettings({
    type: resource.spec.source.group,
    uid: resource.spec.source.name,
  });

  if (!sourceDataSource?.uid || !resource.metadata.name) {
    return undefined;
  }

  const transformations = resource.spec.config.transformations?.map((transformation) => ({
    ...transformation,
    type:
      transformation.type === SupportedTransformationType.Logfmt
        ? SupportedTransformationType.Logfmt
        : SupportedTransformationType.Regex,
  }));

  const baseCorrelation = {
    uid: resource.metadata.name,
    sourceUID: sourceDataSource.uid,
    label: resource.spec.label,
    description: resource.spec.description,
    provisioned: false,
    config: {
      field: resource.spec.config.field,
      target: resource.spec.config.target,
      transformations,
    },
  };

  if (resource.spec.type === 'external') {
    const externalTarget = resource.spec.config.target as CorrelationExternal['config']['target'];
    return {
      ...baseCorrelation,
      type: 'external',
      config: {
        ...baseCorrelation.config,
        target: {
          url: externalTarget?.url ?? '',
        },
      },
    };
  }

  const target = resource.spec.target;
  if (!target?.name || !target.group) {
    return undefined;
  }

  const targetDataSource = getDataSourceSrv().getInstanceSettings({
    type: target.group,
    uid: target.name,
  });
  if (!targetDataSource?.uid) {
    return undefined;
  }

  const correlation: CorrelationQuery = {
    ...baseCorrelation,
    type: 'query',
    targetUID: targetDataSource.uid,
    config: {
      ...baseCorrelation.config,
      target: resource.spec.config.target,
    },
  };

  return correlation;
}
