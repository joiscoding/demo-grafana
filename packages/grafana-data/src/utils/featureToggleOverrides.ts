/**
 * Shared parsing and serialization for the `grafana.featureToggles` localStorage value.
 * Format: `name=value,name=value` where value is `true` or `false` (or `1` for true when parsing).
 * Used by both the runtime config and the Labs page to avoid divergence.
 */
export type FeatureToggleMap = Record<string, boolean>;

const FEATURE_TOGGLE_NAME_SORTER = new Intl.Collator('en');

export function parseFeatureToggleOverrides(rawValue: string | null | undefined): FeatureToggleMap {
  if (!rawValue) {
    return {};
  }

  const overrides: FeatureToggleMap = {};

  for (const featureItem of rawValue.split(',')) {
    const [featureName, featureValue] = featureItem.split('=');
    if (!featureName) {
      continue;
    }

    overrides[featureName] = featureValue === 'true' || featureValue === '1';
  }

  return overrides;
}

export function serializeFeatureToggleOverrides(overrides: FeatureToggleMap): string {
  return Object.entries(overrides)
    .sort(([leftName], [rightName]) => FEATURE_TOGGLE_NAME_SORTER.compare(leftName, rightName))
    .map(([featureName, isEnabled]) => `${featureName}=${isEnabled}`)
    .join(',');
}
