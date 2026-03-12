import { useAsyncFn } from 'react-use';
import { lastValueFrom } from 'rxjs';

import { Correlation as CorrelationK8s } from '@grafana/api-clients/rtkq/correlations/v0alpha1';
import { config, CorrelationData, CorrelationsData, FetchResponse, getDataSourceSrv } from '@grafana/runtime';
import { useGrafana } from 'app/core/context/GrafanaContext';

import {
  Correlation,
  CreateCorrelationParams,
  CreateCorrelationResponse,
  GetCorrelationsParams,
  RemoveCorrelationParams,
  RemoveCorrelationResponse,
  UpdateCorrelationParams,
  UpdateCorrelationResponse,
} from './types';
import { correlationsLogger } from './utils';
import {
  CORRELATIONS_API_BASE_URL,
  fromK8sCorrelation,
  toCreateCorrelationResource,
  toUpdateCorrelationPatch,
} from './k8s';

export interface CorrelationsResponse {
  correlations: Correlation[];
  page: number;
  limit: number;
  totalCount: number;
}

export const toEnrichedCorrelationData = ({ sourceUID, ...correlation }: Correlation): CorrelationData | undefined => {
  const sourceDatasource = getDataSourceSrv().getInstanceSettings(sourceUID);
  const targetDatasource =
    correlation.type === 'query' ? getDataSourceSrv().getInstanceSettings(correlation.targetUID) : undefined;

  // According to #72258 we will remove logic to handle orgId=0/null as global correlations.
  // This logging is to check if there are any customers who did not migrate existing correlations.
  // See Deprecation Notice in https://github.com/grafana/grafana/pull/72258 for more details
  if (correlation?.orgId === undefined || correlation?.orgId === null || correlation?.orgId === 0) {
    correlationsLogger.logWarning('Invalid correlation config: Missing org id.');
  }

  if (
    sourceDatasource &&
    sourceDatasource?.uid !== undefined &&
    targetDatasource?.uid !== undefined &&
    correlation.type === 'query'
  ) {
    return {
      ...correlation,
      source: sourceDatasource,
      target: targetDatasource,
    };
  }

  if (
    sourceDatasource &&
    sourceDatasource?.uid !== undefined &&
    targetDatasource?.uid === undefined &&
    correlation.type === 'external'
  ) {
    return {
      ...correlation,
      source: sourceDatasource,
    };
  }

  correlationsLogger.logWarning(`Invalid correlation config: Missing source or target.`, {
    source: JSON.stringify(sourceDatasource),
    target: JSON.stringify(targetDatasource),
  });
  return undefined;
};

const validSourceFilter = (correlation: CorrelationData | undefined): correlation is CorrelationData => !!correlation;

export const toEnrichedCorrelationsData = (correlationsResponse: CorrelationsResponse): CorrelationsData => {
  return {
    ...correlationsResponse,
    correlations: correlationsResponse.correlations.map(toEnrichedCorrelationData).filter(validSourceFilter),
  };
};

export function getData<T>(response: FetchResponse<T>) {
  return response.data;
}

/**
 * hook for managing correlations data.
 * TODO: ideally this hook shouldn't have any side effect like showing notifications on error
 * and let consumers handle them. It works nicely with the correlations settings page, but when we'll
 * expose this we'll have to remove those side effects.
 */
export const useCorrelations = () => {
  const { backend } = useGrafana();
  const useKubernetesCorrelations = config.featureToggles.kubernetesCorrelations;

  const [getInfo, get] = useAsyncFn<(params: GetCorrelationsParams) => Promise<CorrelationsData>>(
    async (params) => {
      return lastValueFrom(
        backend.fetch<CorrelationsResponse>({
          url: '/api/datasources/correlations',
          params: { page: params.page },
          method: 'GET',
          showErrorAlert: false,
        })
      )
        .then(getData)
        .then(toEnrichedCorrelationsData);
    },

    [backend]
  );

  const [createInfo, create] = useAsyncFn<(params: CreateCorrelationParams) => Promise<CorrelationData>>(
    async ({ sourceUID, ...correlation }) => {
      if (!useKubernetesCorrelations) {
        return backend
          .post<CreateCorrelationResponse>(`/api/datasources/uid/${sourceUID}/correlations`, correlation)
          .then((response) => {
            const enrichedCorrelation = toEnrichedCorrelationData(response.result);
            if (enrichedCorrelation !== undefined) {
              return enrichedCorrelation;
            } else {
              throw new Error('invalid sourceUID');
            }
          });
      }

      const resource = toCreateCorrelationResource({ sourceUID, ...correlation });
      const response = await lastValueFrom(
        backend.fetch<CorrelationK8s>({
          url: `${CORRELATIONS_API_BASE_URL}/correlations`,
          method: 'POST',
          data: resource,
          showErrorAlert: false,
        })
      );
      const createdCorrelation = fromK8sCorrelation(response.data as CorrelationK8s);
      if (createdCorrelation === undefined) {
        throw new Error('invalid sourceUID');
      }

      const enrichedCorrelation = toEnrichedCorrelationData(createdCorrelation);
      if (enrichedCorrelation === undefined) {
        throw new Error('invalid sourceUID');
      }

      return enrichedCorrelation;
    },
    [backend, useKubernetesCorrelations]
  );

  const [removeInfo, remove] = useAsyncFn<(params: RemoveCorrelationParams) => Promise<{ message: string }>>(
    async ({ sourceUID, uid }) => {
      if (!useKubernetesCorrelations) {
        return backend.delete<RemoveCorrelationResponse>(`/api/datasources/uid/${sourceUID}/correlations/${uid}`);
      }

      await lastValueFrom(
        backend.fetch({
          url: `${CORRELATIONS_API_BASE_URL}/correlations/${uid}`,
          method: 'DELETE',
          showErrorAlert: false,
        })
      );

      return { message: 'Correlation deleted' };
    },
    [backend, useKubernetesCorrelations]
  );

  const [updateInfo, update] = useAsyncFn<(params: UpdateCorrelationParams) => Promise<CorrelationData>>(
    async ({ sourceUID, uid, ...correlation }) => {
      if (!useKubernetesCorrelations) {
        return backend
          .patch<UpdateCorrelationResponse>(`/api/datasources/uid/${sourceUID}/correlations/${uid}`, correlation)
          .then((response) => {
            const enrichedCorrelation = toEnrichedCorrelationData(response.result);
            if (enrichedCorrelation !== undefined) {
              return enrichedCorrelation;
            } else {
              throw new Error('invalid sourceUID');
            }
          });
      }

      const response = await lastValueFrom(
        backend.fetch<CorrelationK8s>({
          url: `${CORRELATIONS_API_BASE_URL}/correlations/${uid}`,
          method: 'PATCH',
          data: toUpdateCorrelationPatch({ sourceUID, uid, ...correlation }),
          showErrorAlert: false,
          headers: {
            'Content-Type': 'application/strategic-merge-patch+json',
          },
        })
      );
      const updatedCorrelation = fromK8sCorrelation(response.data as CorrelationK8s);
      if (updatedCorrelation === undefined) {
        throw new Error('invalid sourceUID');
      }

      const enrichedCorrelation = toEnrichedCorrelationData(updatedCorrelation);
      if (enrichedCorrelation === undefined) {
        throw new Error('invalid sourceUID');
      }

      return enrichedCorrelation;
    },
    [backend, useKubernetesCorrelations]
  );

  return {
    create: {
      execute: create,
      ...createInfo,
    },
    update: {
      execute: update,
      ...updateInfo,
    },
    get: {
      execute: get,
      ...getInfo,
    },
    remove: {
      execute: remove,
      ...removeInfo,
    },
  };
};
