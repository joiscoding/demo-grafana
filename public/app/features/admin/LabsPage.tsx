import { css } from '@emotion/css';
import { useMemo, useState } from 'react';
import { useAsync } from 'react-use';

import { GrafanaTheme2 } from '@grafana/data';
import { t } from '@grafana/i18n';
import { getBackendSrv } from '@grafana/runtime';
import {
  Alert,
  Badge,
  CellProps,
  Column,
  EmptyState,
  FilterInput,
  InteractiveTable,
  LoadingPlaceholder,
  Stack,
  useStyles2,
} from '@grafana/ui';
import { Page } from 'app/core/components/Page/Page';

interface FeatureToggleStatus {
  name: string;
  description: string;
  enabled: boolean;
  stage: string;
  warning?: string;
}

interface ResolvedToggleState {
  toggles: FeatureToggleStatus[];
}

type Cell<T extends keyof FeatureToggleStatus = keyof FeatureToggleStatus> = CellProps<
  FeatureToggleStatus,
  FeatureToggleStatus[T]
>;

function LabsPage() {
  const styles = useStyles2(getStyles);
  const [query, setQuery] = useState('');
  const { loading, error, value } = useAsync(
    () => getBackendSrv().get<ResolvedToggleState>('/api/admin/feature-toggles'),
    []
  );

  const toggles = useMemo(() => value?.toggles ?? [], [value]);
  const normalizedQuery = query.trim().toLowerCase();
  const filteredToggles = useMemo(() => {
    if (!normalizedQuery) {
      return toggles;
    }

    return toggles.filter((toggle) =>
      [toggle.name, toggle.description, toggle.stage, toggle.warning].some((value) =>
        value?.toLowerCase().includes(normalizedQuery)
      )
    );
  }, [normalizedQuery, toggles]);

  const columns: Array<Column<FeatureToggleStatus>> = useMemo(
    () => [
      {
        id: 'name',
        header: t('admin.labs.table.header-name', 'Feature flag'),
        accessorFn: (toggle) => toggle.name,
        sortType: 'string',
      },
      {
        id: 'enabled',
        header: t('admin.labs.table.header-enabled', 'Status'),
        accessorFn: (toggle) => toggle.enabled,
        cell: ({ cell: { value } }: Cell<'enabled'>) => (
          <Badge
            text={value ? t('admin.labs.enabled', 'Enabled') : t('admin.labs.disabled', 'Disabled')}
            color={value ? 'green' : 'orange'}
          />
        ),
      },
      {
        id: 'stage',
        header: t('admin.labs.table.header-stage', 'Stage'),
        accessorFn: (toggle) => toggle.stage || 'unknown',
        sortType: 'string',
        cell: ({ cell: { value } }: Cell<'stage'>) => <Badge text={value || 'unknown'} color={getStageColor(value)} />,
      },
      {
        id: 'description',
        header: t('admin.labs.table.header-description', 'Description'),
        accessorFn: (toggle) => toggle.description,
        cell: ({ row: { original } }) => (
          <div className={styles.descriptionCell}>
            <span>{original.description || t('admin.labs.no-description', 'No description available')}</span>
            {original.warning && <Badge text={original.warning} color="red" />}
          </div>
        ),
      },
    ],
    [styles.descriptionCell]
  );

  const enabledCount = toggles.filter((toggle) => toggle.enabled).length;

  return (
    <Page navId="labs">
      <Page.Contents>
        <Stack direction="column" gap={2}>
          <Alert severity="info" title="">
            {t(
              'admin.labs.info',
              'Resolved feature flags combine the registered toggle list with the runtime state currently enabled on this Grafana instance.'
            )}
          </Alert>

          <div className={styles.actionBar}>
            <div className={styles.row}>
              <FilterInput
                placeholder={t('admin.labs.search-placeholder', 'Search feature flags by name, stage, or description')}
                autoFocus={true}
                value={query}
                onChange={setQuery}
              />
            </div>
            {!loading && !error && (
              <Stack direction="row" gap={1}>
                <Badge
                  text={t('admin.labs.summary-total', '{{count}} total', { count: toggles.length })}
                  color="blue"
                />
                <Badge
                  text={t('admin.labs.summary-enabled', '{{count}} enabled', { count: enabledCount })}
                  color="green"
                />
                <Badge
                  text={t('admin.labs.summary-disabled', '{{count}} disabled', {
                    count: toggles.length - enabledCount,
                  })}
                  color="orange"
                />
              </Stack>
            )}
          </div>

          {loading && <LoadingPlaceholder text={t('admin.labs.loading', 'Loading feature flags...')} />}

          {error && (
            <Alert severity="error" title={t('admin.labs.load-error-title', 'Unable to load feature flags')}>
              {t('admin.labs.load-error-description', 'Try refreshing the page or check the Grafana server logs.')}
            </Alert>
          )}

          {!loading && !error && filteredToggles.length > 0 && (
            <InteractiveTable columns={columns} data={filteredToggles} getRowId={(toggle) => toggle.name} />
          )}

          {!loading && !error && filteredToggles.length === 0 && (
            <EmptyState
              variant="not-found"
              message={
                normalizedQuery
                  ? t('admin.labs.empty-filtered', 'No feature flags match your search')
                  : t('admin.labs.empty', 'No feature flags available')
              }
            />
          )}
        </Stack>
      </Page.Contents>
    </Page>
  );
}

function getStageColor(stage?: string) {
  switch (stage) {
    case 'GA':
      return 'green';
    case 'preview':
    case 'publicPreview':
      return 'blue';
    case 'privatePreview':
      return 'purple';
    case 'experimental':
      return 'orange';
    case 'deprecated':
      return 'red';
    default:
      return 'darkgrey';
  }
}

const getStyles = (theme: GrafanaTheme2) => {
  return {
    actionBar: css({
      marginBottom: theme.spacing(2),
      display: 'flex',
      alignItems: 'flex-start',
      justifyContent: 'space-between',
      gap: theme.spacing(2),
      [theme.breakpoints.down('md')]: {
        flexWrap: 'wrap',
      },
    }),
    row: css({
      display: 'flex',
      alignItems: 'flex-start',
      flexGrow: 1,
      [theme.breakpoints.down('md')]: {
        width: '100%',
      },
    }),
    descriptionCell: css({
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(1),
      flexWrap: 'wrap',
    }),
  };
};

export default LabsPage;
