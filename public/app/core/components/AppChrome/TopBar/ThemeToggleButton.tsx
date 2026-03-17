import { memo } from 'react';

import { t } from '@grafana/i18n';
import { ToolbarButton, useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleButton = memo(function ThemeToggleButton() {
  const theme = useTheme2();
  const isDarkMode = theme.isDark;
  const label = isDarkMode
    ? t('navigation.theme.toggle-to-light.aria-label', 'Switch to light mode')
    : t('navigation.theme.toggle-to-dark.aria-label', 'Switch to dark mode');

  return (
    <ToolbarButton
      iconOnly
      icon={isDarkMode ? 'toggle-on' : 'toggle-off'}
      aria-label={label}
      tooltip={label}
      variant={isDarkMode ? 'active' : 'default'}
      onClick={() => void toggleTheme(false)}
    />
  );
});
