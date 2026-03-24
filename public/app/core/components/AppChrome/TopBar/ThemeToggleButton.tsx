import { memo } from 'react';

import { t } from '@grafana/i18n';
import { useTheme2, ToolbarButton } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleButton = memo(function ThemeToggleButton() {
  const theme = useTheme2();
  const isDark = theme.isDark;

  return (
    <ToolbarButton
      iconOnly
      icon={isDark ? 'sun' : 'moon'}
      aria-label={
        isDark
          ? t('navigation.theme-toggle.switch-to-light', 'Switch to light mode')
          : t('navigation.theme-toggle.switch-to-dark', 'Switch to dark mode')
      }
      tooltip={
        isDark
          ? t('navigation.theme-toggle.light-tooltip', 'Switch to light mode')
          : t('navigation.theme-toggle.dark-tooltip', 'Switch to dark mode')
      }
      onClick={() => toggleTheme(false)}
    />
  );
});
