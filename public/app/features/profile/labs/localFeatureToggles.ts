import { store } from '@grafana/data';

export const featureToggleOverridesLocalStorageKey = 'grafana.featureToggles';

export function parseFeatureToggleOverrides(value: string | null): Record<string, boolean> {
  if (!value?.trim()) {
    return {};
  }

  const toggles: Record<string, boolean> = {};
  for (const rawFeature of value.split(',')) {
    const feature = rawFeature.trim();
    if (!feature) {
      continue;
    }

    const [rawName, rawValue] = feature.split('=');
    const name = rawName?.trim();
    if (!name) {
      continue;
    }

    toggles[name] = rawValue === 'true' || rawValue === '1';
  }

  return toggles;
}

export function stringifyFeatureToggleOverrides(overrides: Record<string, boolean>): string {
  const keys = Object.keys(overrides).sort();
  return keys.map((key) => `${key}=${overrides[key]}`).join(',');
}

export function readFeatureToggleOverridesFromLocalStorage(): Record<string, boolean> {
  return parseFeatureToggleOverrides(store.get(featureToggleOverridesLocalStorageKey));
}

export function writeFeatureToggleOverridesToLocalStorage(overrides: Record<string, boolean>): void {
  const serialized = stringifyFeatureToggleOverrides(overrides);

  if (!serialized) {
    store.delete(featureToggleOverridesLocalStorageKey);
    return;
  }

  store.set(featureToggleOverridesLocalStorageKey, serialized);
}
