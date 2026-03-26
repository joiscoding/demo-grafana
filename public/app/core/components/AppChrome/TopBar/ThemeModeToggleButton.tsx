import { t } from '@grafana/i18n';
import { reportInteraction } from '@grafana/runtime';
import { ToolbarButton, useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

export function ThemeModeToggleButton() {
  const theme = useTheme2();
  const nextTheme = theme.isDark ? 'light' : 'dark';
  const tooltip = theme.isDark
    ? t('navigation.theme-mode-toggle.light-tooltip', 'Switch to light mode')
    : t('navigation.theme-mode-toggle.dark-tooltip', 'Switch to dark mode');

  const onClick = async () => {
    reportInteraction('grafana_theme_mode_toggled', {
      fromTheme: theme.isDark ? 'dark' : 'light',
      toTheme: nextTheme,
      placement: 'top_bar_right',
    });

    await toggleTheme(false);
  };

  return (
    <ToolbarButton
      iconOnly
      icon="adjust-circle"
      aria-label={tooltip}
      tooltip={tooltip}
      onClick={onClick}
    />
  );
}
