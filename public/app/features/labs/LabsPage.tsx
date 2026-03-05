import { css } from '@emotion/css';
import { useMemo } from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import config from 'app/core/config';
import { Page } from 'app/core/components/Page/Page';
import { useNavModel } from 'app/core/hooks/useNavModel';
import { useStyles2 } from '@grafana/ui';

interface FeatureToggleItem {
  enabled: boolean;
  name: string;
}

function getStyles(theme: GrafanaTheme2) {
  return {
    pageDescription: css({
      color: theme.colors.text.secondary,
      marginBottom: theme.spacing(2),
    }),
    table: css({
      borderCollapse: 'collapse',
      width: '100%',
    }),
    tableCell: css({
      borderBottom: `1px solid ${theme.colors.border.weak}`,
      padding: `${theme.spacing(1)} ${theme.spacing(1.5)}`,
      textAlign: 'left',
    }),
    tableHeaderCell: css({
      borderBottom: `1px solid ${theme.colors.border.medium}`,
      color: theme.colors.text.secondary,
      fontSize: theme.typography.bodySmall.fontSize,
      fontWeight: theme.typography.fontWeightMedium,
      padding: `${theme.spacing(1)} ${theme.spacing(1.5)}`,
      textAlign: 'left',
      textTransform: 'uppercase',
    }),
  };
}

export default function LabsPage() {
  const navModel = useNavModel('labs');
  const styles = useStyles2(getStyles);

  const featureToggles = useMemo<FeatureToggleItem[]>(() => {
    return Object.entries(config.featureToggles ?? {})
      .map(([name, value]) => ({
        enabled: value === true,
        name,
      }))
      .sort((left, right) => left.name.localeCompare(right.name));
  }, []);

  return (
    <Page navModel={navModel}>
      <Page.Contents>
        <p className={styles.pageDescription}>Read-only view of feature flags currently active in this Grafana instance.</p>
        {featureToggles.length === 0 ? (
          <p>No feature flags are enabled.</p>
        ) : (
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.tableHeaderCell}>Feature flag</th>
                <th className={styles.tableHeaderCell}>Enabled</th>
              </tr>
            </thead>
            <tbody>
              {featureToggles.map((featureToggle) => (
                <tr key={featureToggle.name}>
                  <td className={styles.tableCell}>
                    <code>{featureToggle.name}</code>
                  </td>
                  <td className={styles.tableCell}>{featureToggle.enabled ? 'Yes' : 'No'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Page.Contents>
    </Page>
  );
}
