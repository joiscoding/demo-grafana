import { css } from '@emotion/css';
import { useCallback, useMemo, useState } from 'react';
import { useAsync } from 'react-use';

import type { GrafanaTheme2 } from '@grafana/data';
import { t, Trans } from '@grafana/i18n';
import { config } from '@grafana/runtime';
import { Alert, Badge, Button, Input, Spinner, Stack, Switch, useStyles2 } from '@grafana/ui';
import { ScopedResourceClient } from 'app/features/apiserver/client';
import { Resource } from 'app/features/apiserver/types';

import {
  readFeatureToggleOverridesFromLocalStorage,
  writeFeatureToggleOverridesToLocalStorage,
} from './localFeatureToggles';

const featureResource = {
  group: 'featuretoggle.grafana.app',
  version: 'v0alpha1',
  resource: 'features',
};

interface FeatureSpec {
  description?: string;
  stage?: string;
  frontend?: boolean;
  requiresRestart?: boolean;
}

interface FeatureDefinition {
  description?: string;
  stage?: string;
  frontendOnly: boolean;
  requiresRestart: boolean;
}

export function LabsSettings() {
  const styles = useStyles2(getStyles);
  const [query, setQuery] = useState('');
  const [overrides, setOverrides] = useState<Record<string, boolean>>(() => readFeatureToggleOverridesFromLocalStorage());

  const { loading, error, value: featureDefinitions = {} } = useAsync(loadFeatureDefinitions, []);

  const availableToggleNames = useMemo(() => {
    const names = new Set<string>([...Object.keys(config.featureToggles), ...Object.keys(overrides)]);
    Object.keys(featureDefinitions).forEach((name) => names.add(name));
    return Array.from(names).sort();
  }, [featureDefinitions, overrides]);

  const filteredToggleNames = useMemo(() => {
    const search = query.trim().toLowerCase();
    if (!search) {
      return availableToggleNames;
    }

    return availableToggleNames.filter((name) => {
      const description = featureDefinitions[name]?.description?.toLowerCase() ?? '';
      return name.toLowerCase().includes(search) || description.includes(search);
    });
  }, [availableToggleNames, featureDefinitions, query]);

  const runtimeToggleValues = useMemo(() => {
    return new Map(
      Object.entries(config.featureToggles).map(([name, enabled]) => [name, Boolean(enabled)] as const)
    );
  }, []);

  const onToggleChange = useCallback((name: string, enabled: boolean) => {
    setOverrides((currentOverrides) => {
      const nextOverrides = {
        ...currentOverrides,
        [name]: enabled,
      };

      writeFeatureToggleOverridesToLocalStorage(nextOverrides);

      return nextOverrides;
    });
  }, []);

  const onResetAndReload = useCallback(() => {
    writeFeatureToggleOverridesToLocalStorage({});
    window.location.reload();
  }, []);

  const onReload = useCallback(() => {
    window.location.reload();
  }, []);

  const hasOverrides = Object.keys(overrides).length > 0;

  return (
    <Stack direction="column" gap={2} data-testid="profile-labs-settings">
      <Alert severity="info" title={t('profile.labs.info-title', 'Labs feature toggles')}>
        <Trans i18nKey="profile.labs.info-description">
          Opt in to experimental features for this browser only. Reload Grafana after changing toggles for consistent
          behavior.
        </Trans>
      </Alert>

      <Stack direction="row" gap={1} alignItems="center">
        <Input
          aria-label={t('profile.labs.filter-label', 'Filter lab feature toggles')}
          value={query}
          placeholder={t('profile.labs.filter-placeholder', 'Filter toggles by name or description')}
          onChange={(event) => setQuery(event.currentTarget.value)}
          className={styles.filterInput}
        />
        <Button variant="secondary" onClick={onReload}>
          <Trans i18nKey="profile.labs.reload-button">Reload</Trans>
        </Button>
        <Button variant="destructive" onClick={onResetAndReload} disabled={!hasOverrides}>
          <Trans i18nKey="profile.labs.reset-button">Reset overrides</Trans>
        </Button>
      </Stack>

      {error && (
        <Alert severity="warning" title={t('profile.labs.metadata-error-title', 'Unable to load toggle metadata')}>
          <Trans i18nKey="profile.labs.metadata-error-description">
            Showing toggle names without descriptions because feature metadata is unavailable.
          </Trans>
        </Alert>
      )}

      {loading && (
        <Stack direction="row" alignItems="center" gap={1}>
          <Spinner />
          <span>
            <Trans i18nKey="profile.labs.loading">Loading feature metadata…</Trans>
          </span>
        </Stack>
      )}

      {filteredToggleNames.length === 0 && !loading ? (
        <Alert severity="info" title={t('profile.labs.empty-state-title', 'No matching toggles')}>
          <Trans i18nKey="profile.labs.empty-state-description">Try a different filter.</Trans>
        </Alert>
      ) : (
        <Stack direction="column" gap={1}>
          {filteredToggleNames.map((name) => {
            const definition = featureDefinitions[name];
            const isEnabled = overrides[name] ?? runtimeToggleValues.get(name) ?? false;
            const isOverridden = Object.prototype.hasOwnProperty.call(overrides, name);

            return (
              <div className={styles.toggleRow} key={name} data-testid={`labs-toggle-row-${name}`}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" gap={2}>
                  <div>
                    <div className={styles.toggleName}>{name}</div>
                    <Stack direction="row" gap={0.5}>
                      {definition?.stage && <Badge text={definition.stage} color="blue" />}
                      {definition?.frontendOnly && (
                        <Badge text={t('profile.labs.badge.frontend', 'Frontend')} color="green" />
                      )}
                      {definition?.requiresRestart && (
                        <Badge text={t('profile.labs.badge.restart', 'Requires restart')} color="orange" />
                      )}
                      {isOverridden && (
                        <Badge text={t('profile.labs.badge.override', 'Browser override')} color="purple" />
                      )}
                    </Stack>
                  </div>

                  <Stack direction="row" gap={1} alignItems="center">
                    <span className={styles.toggleState}>
                      {isEnabled ? t('profile.labs.toggle-state.on', 'On') : t('profile.labs.toggle-state.off', 'Off')}
                    </span>
                    <Switch
                      id={`labs-toggle-${name}`}
                      value={isEnabled}
                      data-testid={`labs-toggle-switch-${name}`}
                      onChange={(event) => onToggleChange(name, event.currentTarget.checked)}
                    />
                  </Stack>
                </Stack>

                {definition?.description && <p className={styles.description}>{definition.description}</p>}
              </div>
            );
          })}
        </Stack>
      )}
    </Stack>
  );
}

async function loadFeatureDefinitions(): Promise<Record<string, FeatureDefinition>> {
  const client = new ScopedResourceClient<FeatureSpec, object, 'Feature'>(featureResource, false);
  const featureList = await client.list();
  const items = Array.isArray(featureList.items) ? featureList.items : [];

  return items.reduce<Record<string, FeatureDefinition>>((acc, feature: Resource<FeatureSpec>) => {
    const name = feature.metadata.name;
    if (!name) {
      return acc;
    }
    const spec = feature.spec ?? {};

    acc[name] = {
      description: spec.description,
      stage: spec.stage,
      frontendOnly: Boolean(spec.frontend),
      requiresRestart: Boolean(spec.requiresRestart),
    };

    return acc;
  }, {});
}

const getStyles = (theme: GrafanaTheme2) => ({
  filterInput: css({
    flex: 1,
  }),
  toggleRow: css({
    border: `1px solid ${theme.colors.border.weak}`,
    borderRadius: theme.shape.radius.default,
    padding: theme.spacing(1.5),
  }),
  toggleName: css({
    fontFamily: theme.typography.fontFamilyMonospace,
    marginBottom: theme.spacing(0.5),
  }),
  toggleState: css({
    color: theme.colors.text.secondary,
    minWidth: theme.spacing(3),
    textAlign: 'right',
  }),
  description: css({
    margin: `${theme.spacing(1)} 0 0`,
    color: theme.colors.text.secondary,
  }),
});
