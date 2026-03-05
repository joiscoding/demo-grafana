import { t, Trans } from '@grafana/i18n';
import { config } from '@grafana/runtime';
import { Alert, Badge, Card, ScrollContainer, Stack, Text } from '@grafana/ui';
import { Page } from 'app/core/components/Page/Page';

function LabsFeatureFlagsPage() {
  const featureToggles = (config.featureToggles || {}) as Record<string, boolean>;
  const entries = Object.entries(featureToggles).sort(([a], [b]) => a.localeCompare(b));

  return (
    <Page navId="labs/feature-toggles">
      <Page.Contents>
        <Stack direction="column" gap={2}>
          <Alert severity="info" title="">
            <Trans i18nKey="labs.feature-toggles.info-description">
              This page shows the feature flags currently enabled in this Grafana instance. To change feature flags
              instance-wide, edit the <code>[feature_toggles]</code> section in your Grafana configuration file (
              <code>custom.ini</code> or <code>grafana.ini</code>) and restart Grafana.
            </Trans>
          </Alert>

          <Card>
            <Card.Heading>
              <Trans i18nKey="labs.feature-toggles.table-heading">Feature toggles</Trans>
            </Card.Heading>
            <Card.Description>
              <Trans i18nKey="labs.feature-toggles.table-description" count={entries.length}>
                {'{{count}}'} feature flags configured for this instance
              </Trans>
            </Card.Description>
            <ScrollContainer overflowY="visible" overflowX="auto" width="100%">
              <table className="filter-table">
                  <thead>
                    <tr>
                      <th>
                        <Trans i18nKey="labs.feature-toggles.column-name">Name</Trans>
                      </th>
                      <th>
                        <Trans i18nKey="labs.feature-toggles.column-status">Status</Trans>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {entries.length === 0 ? (
                      <tr>
                        <td colSpan={2}>
                          <Text color="secondary">
                            <Trans i18nKey="labs.feature-toggles.empty">
                              No feature toggles are configured. Add entries under [feature_toggles] in your config file.
                            </Trans>
                          </Text>
                        </td>
                      </tr>
                    ) : (
                      entries.map(([name, enabled]) => (
                        <tr key={name}>
                          <td>
                            <Text>{name}</Text>
                          </td>
                          <td>
                            <Badge
                              text={enabled ? t('labs.feature-toggles.status-enabled', 'Enabled') : t('labs.feature-toggles.status-disabled', 'Disabled')}
                              color={enabled ? 'green' : 'darkgrey'}
                            />
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
              </table>
            </ScrollContainer>
          </Card>
        </Stack>
      </Page.Contents>
    </Page>
  );
}

export default LabsFeatureFlagsPage;
