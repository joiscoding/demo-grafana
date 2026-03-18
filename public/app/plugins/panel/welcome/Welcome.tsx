import { css } from '@emotion/css';

import { GrafanaTheme2 } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { IconButton, TextLink, useStyles2, useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

const helpOptions = [
  { value: 0, label: 'Documentation', href: 'https://grafana.com/docs/grafana/latest' },
  { value: 1, label: 'Tutorials', href: 'https://grafana.com/tutorials' },
  { value: 2, label: 'Community', href: 'https://community.grafana.com' },
  { value: 3, label: 'Public Slack', href: 'http://slack.grafana.com' },
];

export const WelcomeBanner = () => {
  const styles = useStyles2(getStyles);
  const theme = useTheme2();
  const themeToggleLabel = theme.isDark
    ? t('welcome.welcome-banner.switch-to-light-theme', 'Switch to light theme')
    : t('welcome.welcome-banner.switch-to-dark-theme', 'Switch to dark theme');

  return (
    <div className={styles.container}>
      <div className={styles.titleRow}>
        <h1 className={styles.title}>
          <Trans i18nKey="welcome.welcome-banner.welcome-to-grafana">Welcome to Grafana</Trans>
        </h1>
        <IconButton
          name="adjust-circle"
          tooltip={themeToggleLabel}
          size="xl"
          variant="secondary"
          type="button"
          onClick={() => {
            void toggleTheme(false);
          }}
        />
      </div>
      <div className={styles.help}>
        <h2 className={styles.helpText}>
          <Trans i18nKey="welcome.welcome-banner.need-help">Need help?</Trans>
        </h2>
        <div className={styles.helpLinks}>
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
        </div>
      </div>
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
    titleRow: css({
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(1),
      flex: '1 1 auto',
      minWidth: 0,
      flexWrap: 'wrap',

      [theme.breakpoints.down('lg')]: {
        marginBottom: theme.spacing(1),
      },
    }),
    title: css({
      marginBottom: 0,
      flex: '1 1 auto',
      minWidth: 0,

      [theme.breakpoints.down('md')]: {
        fontSize: theme.typography.h2.fontSize,
      },
      [theme.breakpoints.down('sm')]: {
        fontSize: theme.typography.h3.fontSize,
      },
    }),
    help: css({
      display: 'flex',
      alignItems: 'baseline',
    }),
    helpText: css({
      ...theme.typography.h3,
      marginRight: theme.spacing(2),
      marginBottom: 0,

      [theme.breakpoints.down('md')]: {
        fontSize: theme.typography.h4.fontSize,
      },

      [theme.breakpoints.down('sm')]: {
        display: 'none',
      },
    }),
    helpLinks: css({
      display: 'flex',
      flexWrap: 'wrap',
      gap: theme.spacing(2),
      textWrap: 'nowrap',

      [theme.breakpoints.down('sm')]: {
        gap: theme.spacing(1),
      },
    }),
  };
};
