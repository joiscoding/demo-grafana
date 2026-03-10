import { config, getBackendSrv } from '@grafana/runtime';

const featureFlagSortCollator = new Intl.Collator('en');

type FeatureFlagEvaluationResponse = {
  flags: FeatureFlagEvaluation[];
};

type FeatureFlagEvaluation = {
  key: string;
  value: boolean;
  variant?: string;
};

export interface FeatureFlagRow {
  key: string;
  enabled: boolean;
  hasRuntimeOverride: boolean;
}

export async function loadFeatureFlags(): Promise<FeatureFlagRow[]> {
  const response = await getBackendSrv().post<FeatureFlagEvaluationResponse>(
    `/apis/features.grafana.app/v0alpha1/namespaces/${config.namespace}/ofrep/v1/evaluate/flags`,
    {
      context: {
        targetingKey: config.namespace,
        namespace: config.namespace,
        ...config.openFeatureContext,
      },
    }
  );

  return response.flags
    .map((flag) => {
      const evaluatedValue = Boolean(flag.value);
      const runtimeValue = Reflect.get(config.featureToggles, flag.key);
      const runtimeOverrideValue = typeof runtimeValue === 'boolean' ? runtimeValue : undefined;
      const hasRuntimeOverride = runtimeOverrideValue !== undefined && runtimeOverrideValue !== evaluatedValue;

      return {
        key: flag.key,
        enabled: runtimeOverrideValue ?? evaluatedValue,
        hasRuntimeOverride,
      };
    })
    .sort(
      (left, right) =>
        Number(right.enabled) - Number(left.enabled) || featureFlagSortCollator.compare(left.key, right.key)
    );
}
