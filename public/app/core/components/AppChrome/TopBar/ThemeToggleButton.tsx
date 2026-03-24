import { memo } from 'react';

import { t } from '@grafana/i18n';
import { ToolbarButton, useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleButton = memo(function ThemeToggleButton() {
  const theme = useTheme2();
  const isDarkMode = theme.isDark;
  const toggleLabel = isDarkMode
    ? t('navigation.theme-toggle.switch-to-light', 'Switch to light mode')
    : t('navigation.theme-toggle.switch-to-dark', 'Switch to dark mode');

  return (
    <ToolbarButton
      iconOnly
      icon={isDarkMode ? 'toggle-on' : 'toggle-off'}
      aria-label={toggleLabel}
      tooltip={toggleLabel}
      onClick={() => void toggleTheme(false)}
    />
  );
});
