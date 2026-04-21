import { css } from '@emotion/css';

import { GrafanaTheme2 } from '@grafana/data';
import { t, Trans } from '@grafana/i18n';
import { Stack, TextLink, useStyles2 } from '@grafana/ui';

export const WelcomeBanner = () => {
  const styles = useStyles2(getStyles);

  const helpOptions = [
    {
      label: t('welcome.welcome-banner.help-options.documentation', 'Documentation'),
      href: 'https://grafana.com/docs/grafana/latest',
    },
    {
      label: t('welcome.welcome-banner.help-options.tutorials', 'Tutorials'),
      href: 'https://grafana.com/tutorials',
    },
    {
      label: t('welcome.welcome-banner.help-options.community', 'Community'),
      href: 'https://community.grafana.com',
    },
    {
      label: t('welcome.welcome-banner.help-options.public-slack', 'Public Slack'),
      href: 'http://slack.grafana.com',
    },
  ];

  return (
    <div className={styles.container}>
      <h1 className={styles.title}>
        <Trans i18nKey="welcome.welcome-banner.welcome-to-grafana">Welcome to Grafana</Trans>
      </h1>
      <Stack direction="row" alignItems="baseline" gap={2}>
        <h2 className={styles.helpText}>
          <Trans i18nKey="welcome.welcome-banner.need-help">Need help?</Trans>
        </h2>
        <Stack direction="row" gap={2} wrap="wrap">
          {helpOptions.map((option, index) => (
            <TextLink
              key={`${option.label}-${index}`}
              href={`${option.href}?utm_source=grafana_gettingstarted`}
              external
              inline={false}
            >
              {option.label}
            </TextLink>
          ))}
        </Stack>
      </Stack>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => {
  return {
    container: css({
      display: 'flex',
      backgroundSize: 'cover',
      height: '100%',
      alignItems: 'center',
      justifyContent: 'space-between',
      padding: theme.spacing(0, 3),

      [theme.breakpoints.down('lg')]: {
        backgroundPosition: '0px',
        flexDirection: 'column',
        alignItems: 'flex-start',
        justifyContent: 'center',
      },

      [theme.breakpoints.down('sm')]: {
        padding: theme.spacing(0, 1),
      },
    }),
    title: css({
      marginBottom: 0,

      [theme.breakpoints.down('lg')]: {
        marginBottom: theme.spacing(1),
      },

      [theme.breakpoints.down('md')]: {
        fontSize: theme.typography.h2.fontSize,
      },
      [theme.breakpoints.down('sm')]: {
        fontSize: theme.typography.h3.fontSize,
      },
    }),
    helpText: css({
      ...theme.typography.h3,
      marginBottom: 0,

      [theme.breakpoints.down('md')]: {
        fontSize: theme.typography.h4.fontSize,
      },

      [theme.breakpoints.down('sm')]: {
        display: 'none',
      },
    }),
  };
};
