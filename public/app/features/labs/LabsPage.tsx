import { css } from '@emotion/css';
import { useEffect, useMemo, useState } from 'react';

import { Column, GrafanaTheme2 } from '@grafana/data';
import { t } from '@grafana/i18n';
import { Alert, Badge, EmptyState, InteractiveTable, Stack, Text, useStyles2 } from '@grafana/ui';
import { Page } from 'app/core/components/Page/Page';

import { FeatureFlagRow, loadFeatureFlags } from './api';

interface LabsPageState {
  error?: Error;
  flags: FeatureFlagRow[];
  isLoading: boolean;
}

export default function LabsPage() {
  const styles = useStyles2(getStyles);
  const [state, setState] = useState<LabsPageState>({ flags: [], isLoading: true });
  const subtitle = t(
    'labs-page.subtitle',
    'Review evaluated feature flags and whether they are enabled in the current runtime.'
  );

  useEffect(() => {
    let isMounted = true;

    const fetchFlags = async () => {
      try {
        const flags = await loadFeatureFlags();

        if (isMounted) {
          setState({ flags, isLoading: false });
        }
      } catch (error) {
        if (isMounted) {
          setState({
            error: error instanceof Error ? error : new Error(t('labs-page.error.unknown', 'Unknown error')),
            flags: [],
            isLoading: false,
          });
        }
      }
    };

    void fetchFlags();

    return () => {
      isMounted = false;
    };
  }, []);

  const columns = useMemo<Array<Column<FeatureFlagRow>>>(
    () => [
      {
        id: 'key',
        header: t('labs-page.columns.flag', 'Feature flag'),
        accessor: 'key',
        sortType: 'string',
      },
      {
        id: 'enabled',
        header: t('labs-page.columns.status', 'Status'),
        accessor: 'enabled',
        cell: ({ row: { original } }) => (
          <Stack alignItems="center" gap={1}>
            <Badge
              color={original.enabled ? 'green' : 'red'}
              text={original.enabled ? t('labs-page.enabled', 'Enabled') : t('labs-page.disabled', 'Disabled')}
            />
            {original.hasRuntimeOverride && (
              <Badge color="blue" text={t('labs-page.runtime-override', 'Runtime override')} />
            )}
          </Stack>
        ),
      },
    ],
    []
  );

  const enabledCount = state.flags.filter((flag) => flag.enabled).length;
  const runtimeOverrideCount = state.flags.filter((flag) => flag.hasRuntimeOverride).length;

  return (
    <Page
      navId="labs"
      pageNav={{ text: t('labs-page.title', 'Labs'), subTitle: subtitle, url: '/labs' }}
      subTitle={subtitle}
    >
      <Page.Contents isLoading={state.isLoading}>
        {state.error && (
          <Alert title={t('labs-page.error.title', 'Unable to load feature flags')} severity="error">
            {state.error.message}
          </Alert>
        )}

        {!state.isLoading && !state.error && (
          <>
            {state.flags.length === 0 ? (
              <EmptyState
                variant="not-found"
                message={t('labs-page.empty.message', 'No feature flags are available')}
              />
            ) : (
              <div className={styles.content}>
                <div className={styles.summaryRow}>
                  <Badge
                    color="green"
                    text={t('labs-page.summary.enabled', '{{count}} enabled', { count: enabledCount })}
                  />
                  <Text color="secondary">
                    {t('labs-page.summary.total', '{{count}} total flags', { count: state.flags.length })}
                  </Text>
                  {runtimeOverrideCount > 0 && (
                    <Badge
                      color="blue"
                      text={t('labs-page.summary.overrides', '{{count}} runtime overrides', {
                        count: runtimeOverrideCount,
                      })}
                    />
                  )}
                </div>

                <InteractiveTable columns={columns} data={state.flags} getRowId={(flag) => flag.key} />
              </div>
            )}
          </>
        )}
      </Page.Contents>
    </Page>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  content: css({
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(2),
  }),
  summaryRow: css({
    display: 'flex',
    flexWrap: 'wrap',
    gap: theme.spacing(2),
    alignItems: 'center',
  }),
});
