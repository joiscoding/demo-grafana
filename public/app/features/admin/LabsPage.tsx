import { css } from '@emotion/css';
import { useMemo, useState } from 'react';
import { useAsync } from 'react-use';

import { GrafanaTheme2 } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { getBackendSrv } from '@grafana/runtime';
import { Alert, Badge, CellProps, Column, FilterInput, InteractiveTable, Spinner, Stack, useStyles2 } from '@grafana/ui';
import { Page } from 'app/core/components/Page/Page';

interface FeatureToggle {
  name: string;
  description?: string;
  stage: string;
  enabled: boolean;
  frontendOnly?: boolean;
  requiresRestart?: boolean;
}

interface FeatureTogglesResponse {
  toggles: FeatureToggle[];
}

type Cell<T extends keyof FeatureToggle = keyof FeatureToggle> = CellProps<FeatureToggle, FeatureToggle[T]>;
const EMPTY_TOGGLES: FeatureToggle[] = [];

const getFeatureToggles = async (): Promise<FeatureTogglesResponse | null> => {
  return getBackendSrv().get('api/admin/feature-toggles');
};

export default function LabsPage() {
  const styles = useStyles2(getStyles);
  const [query, setQuery] = useState('');
  const { loading, error, value } = useAsync(getFeatureToggles, []);

  const toggles = value?.toggles ?? EMPTY_TOGGLES;
  const enabledCount = useMemo(() => toggles.filter((toggle) => toggle.enabled).length, [toggles]);

  const filteredToggles = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    if (!normalizedQuery) {
      return toggles;
    }

    return toggles.filter((toggle) => {
      return (
        toggle.name.toLowerCase().includes(normalizedQuery) ||
        toggle.description?.toLowerCase().includes(normalizedQuery) ||
        toggle.stage.toLowerCase().includes(normalizedQuery)
      );
    });
  }, [query, toggles]);

  const columns = useMemo<Array<Column<FeatureToggle>>>(
    () => [
      {
        id: 'name',
        header: t('admin.labs-page.flag-column', 'Feature flag'),
        sortType: 'string',
        cell: ({ row }: Cell) => (
          <Stack direction="column" gap={0.5}>
            <div className={styles.flagName}>{row.original.name}</div>
            {row.original.description && <div className={styles.description}>{row.original.description}</div>}
          </Stack>
        ),
      },
      {
        id: 'enabled',
        header: t('admin.labs-page.status-column', 'Status'),
        disableGrow: true,
        cell: ({ cell: { value } }: Cell<'enabled'>) => (
          <Badge
            text={
              value
                ? t('admin.labs-page.status-enabled', 'Enabled')
                : t('admin.labs-page.status-disabled', 'Disabled')
            }
            color={value ? 'green' : 'orange'}
          />
        ),
      },
      {
        id: 'stage',
        header: t('admin.labs-page.stage-column', 'Stage'),
        sortType: 'string',
        disableGrow: true,
      },
      {
        id: 'frontendOnly',
        header: t('admin.labs-page.frontend-only-column', 'Frontend only'),
        disableGrow: true,
        cell: ({ cell: { value } }: Cell<'frontendOnly'>) => (
          <>{value ? t('admin.labs-page.yes', 'Yes') : t('admin.labs-page.no', 'No')}</>
        ),
      },
      {
        id: 'requiresRestart',
        header: t('admin.labs-page.restart-required-column', 'Restart required'),
        disableGrow: true,
        cell: ({ cell: { value } }: Cell<'requiresRestart'>) => (
          <>{value ? t('admin.labs-page.yes', 'Yes') : t('admin.labs-page.no', 'No')}</>
        ),
      },
    ],
    [styles]
  );

  return (
    <Page navId="labs">
      <Page.Contents>
        <Stack direction="column" gap={2}>
          <Alert severity="info" title="">
            <Trans i18nKey="admin.labs-page.summary" values={{ total: toggles.length, enabled: enabledCount }}>
              This page lists {{ total: toggles.length }} feature flags for the current Grafana instance. {{
                enabled: enabledCount
              }} are enabled.
            </Trans>
          </Alert>

          <div className={styles.actionBar}>
            <FilterInput
              placeholder={t(
                'admin.labs-page.search-placeholder',
                'Search feature flags by name, stage, or description'
              )}
              value={query}
              onChange={setQuery}
            />
          </div>

          {loading && (
            <div className={styles.loadingState}>
              <Spinner />
            </div>
          )}

          {!loading && error && (
            <Alert severity="error" title={t('admin.labs-page.error-title', 'Failed to load feature flags')}>
              {error.message}
            </Alert>
          )}

          {!loading && !error && filteredToggles.length === 0 && (
            <Alert severity="warning" title={t('admin.labs-page.empty-title', 'No feature flags found')}>
              <Trans i18nKey="admin.labs-page.empty-body">Try adjusting the current filter.</Trans>
            </Alert>
          )}

          {!loading && !error && filteredToggles.length > 0 && (
            <div className={styles.tableWrapper}>
              <InteractiveTable columns={columns} data={filteredToggles} getRowId={(row) => row.name} />
            </div>
          )}
        </Stack>
      </Page.Contents>
    </Page>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  actionBar: css({
    maxWidth: '480px',
  }),
  tableWrapper: css({
    minWidth: 0,
  }),
  flagName: css({
    fontWeight: theme.typography.fontWeightMedium,
  }),
  description: css({
    color: theme.colors.text.secondary,
  }),
  loadingState: css({
    display: 'flex',
    justifyContent: 'center',
    padding: theme.spacing(4, 0),
  }),
});
