import { css } from '@emotion/css';
import { ChangeEvent, useMemo, useState } from 'react';

import { GrafanaTheme2, store } from '@grafana/data';
import { t } from '@grafana/i18n';
import { config } from '@grafana/runtime';
import { Alert, Badge, Button, Input, Stack, Switch, Text, useStyles2 } from '@grafana/ui';
import { Page } from 'app/core/components/Page/Page';

const FEATURE_TOGGLE_STORAGE_KEY = 'grafana.featureToggles';
const FEATURE_TOGGLE_NAME_SORTER = new Intl.Collator('en');
const FEATURE_MANAGEMENT_WRITE_PERMISSION = 'featuremgmt.write';

type FeatureToggleMap = Record<string, boolean>;

function parseFeatureToggleOverrides(rawValue: string | undefined): FeatureToggleMap {
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

function serializeFeatureToggleOverrides(overrides: FeatureToggleMap): string {
  return Object.entries(overrides)
    .sort(([leftName], [rightName]) => FEATURE_TOGGLE_NAME_SORTER.compare(leftName, rightName))
    .map(([featureName, isEnabled]) => `${featureName}=${isEnabled}`)
    .join(',');
}

function buildInitialFeatureToggleState(): FeatureToggleMap {
  const runtimeFeatureToggles: FeatureToggleMap = {};

  for (const [featureName, featureValue] of Object.entries(config.featureToggles)) {
    if (typeof featureValue === 'boolean') {
      runtimeFeatureToggles[featureName] = featureValue;
    }
  }

  const runtimeFeatureToggleNames = new Set(Object.keys(runtimeFeatureToggles));
  const overrideFeatureToggles = parseFeatureToggleOverrides(store.get(FEATURE_TOGGLE_STORAGE_KEY));
  const filteredOverrides = Object.fromEntries(
    Object.entries(overrideFeatureToggles).filter(([featureName]) => runtimeFeatureToggleNames.has(featureName))
  );

  return { ...runtimeFeatureToggles, ...filteredOverrides };
}

export default function LabsPage() {
  const styles = useStyles2(getStyles);
  const [search, setSearch] = useState('');
  const [featureToggles, setFeatureToggles] = useState<FeatureToggleMap>(() => buildInitialFeatureToggleState());
  const canWriteFeatureFlags = Boolean(config.bootData?.user?.permissions?.[FEATURE_MANAGEMENT_WRITE_PERMISSION]);

  const enabledCount = useMemo(
    () => Object.values(featureToggles).filter((isEnabled) => isEnabled).length,
    [featureToggles]
  );

  const featureNames = useMemo(() => {
    const searchValue = search.trim().toLowerCase();
    return Object.keys(featureToggles)
      .sort((leftName, rightName) => FEATURE_TOGGLE_NAME_SORTER.compare(leftName, rightName))
      .filter((featureName) => featureName.toLowerCase().includes(searchValue));
  }, [featureToggles, search]);

  const onToggleChange = (featureName: string, event: ChangeEvent<HTMLInputElement>) => {
    if (!canWriteFeatureFlags) {
      return;
    }

    const enabled = event.currentTarget.checked;

    setFeatureToggles((prev) => ({ ...prev, [featureName]: enabled }));
    const localStorageFeatureToggles = parseFeatureToggleOverrides(store.get(FEATURE_TOGGLE_STORAGE_KEY));
    localStorageFeatureToggles[featureName] = enabled;
    store.set(FEATURE_TOGGLE_STORAGE_KEY, serializeFeatureToggleOverrides(localStorageFeatureToggles));
    config.featureToggles = { ...config.featureToggles, [featureName]: enabled };
  };

  return (
    <Page
      navId="labs"
      pageNav={{
        text: t('nav.labs.title', 'Labs'),
        active: true,
      }}
    >
      <Page.Contents>
        <div className={styles.header}>
          <Text variant="h2">{t('labs.feature-flags.title', 'Feature flags')}</Text>
          <Text color="secondary">
            {t(
              'labs.feature-flags.description',
              'View enabled feature flags and override them for your browser session.'
            )}
          </Text>
          <Stack direction="row" gap={1}>
            <Badge text={t('labs.feature-flags.enabled-count', '{{count}} enabled', { count: enabledCount })} color="green" />
            <Badge
              text={t('labs.feature-flags.total-count', '{{count}} visible', { count: featureNames.length })}
              color="blue"
            />
          </Stack>
          <Input
            value={search}
            onChange={(event) => setSearch(event.currentTarget.value)}
            placeholder={t('labs.feature-flags.search-placeholder', 'Search feature flags')}
            width={32}
            prefix={<span className={styles.searchPrefix}>{t('labs.feature-flags.search-prefix', 'Search')}</span>}
          />
        </div>

        <Alert title={t('labs.feature-flags.restart-note-title', 'Local browser override')} severity="info">
          {t(
            'labs.feature-flags.restart-note',
            'Changes are stored in localStorage and can require a page reload to fully apply.'
          )}
        </Alert>
        {!canWriteFeatureFlags && (
          <Alert title={t('labs.feature-flags.read-only-title', 'Read-only access')} severity="warning">
            {t(
              'labs.feature-flags.read-only',
              'You can view feature flags, but you do not have permission to update them.'
            )}
          </Alert>
        )}

        <Button
          className={styles.reloadButton}
          variant="secondary"
          onClick={() => window.location.reload()}
          icon="sync"
        >
          {t('labs.feature-flags.reload-button', 'Reload app')}
        </Button>

        <div className={styles.featureList}>
          {featureNames.length === 0 && (
            <Text color="secondary">{t('labs.feature-flags.no-results', 'No feature flags match your search.')}</Text>
          )}
          {featureNames.map((featureName) => {
            const switchId = `labs-feature-toggle-${featureName}`;
            const enabled = Boolean(featureToggles[featureName]);

            return (
              <div key={featureName} className={styles.featureRow}>
                <label className={styles.featureLabel} htmlFor={switchId}>
                  {featureName}
                </label>
                <Stack direction="row" alignItems="center" gap={1}>
                  <Badge
                    text={enabled ? t('labs.feature-flags.on', 'On') : t('labs.feature-flags.off', 'Off')}
                    color={enabled ? 'green' : 'blue'}
                  />
                  <Switch
                    id={switchId}
                    value={enabled}
                    onChange={(event) => onToggleChange(featureName, event)}
                    data-testid={`labs-feature-toggle-${featureName}`}
                    disabled={!canWriteFeatureFlags}
                  />
                </Stack>
              </div>
            );
          })}
        </div>
      </Page.Contents>
    </Page>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  header: css({
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1),
    marginBottom: theme.spacing(2),
  }),
  searchPrefix: css({
    color: theme.colors.text.secondary,
    fontSize: theme.typography.bodySmall.fontSize,
  }),
  reloadButton: css({
    marginBottom: theme.spacing(2),
  }),
  featureList: css({
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1),
  }),
  featureRow: css({
    alignItems: 'center',
    border: `1px solid ${theme.colors.border.weak}`,
    borderRadius: theme.shape.radius.default,
    display: 'flex',
    justifyContent: 'space-between',
    padding: theme.spacing(1.5, 2),
  }),
  featureLabel: css({
    fontFamily: theme.typography.fontFamilyMonospace,
    marginBottom: 0,
  }),
});
