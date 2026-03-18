import { memo } from 'react';

import { t } from '@grafana/i18n';
import { config } from '@grafana/runtime';
import { ToolbarButton, useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleButton = memo(function ThemeToggleButton() {
  const theme = useTheme2();
  const isDark = theme.isDark;

  const label = isDark
    ? t('navigation.theme-toggle.switch-to-light', 'Switch to light theme')
    : t('navigation.theme-toggle.switch-to-dark', 'Switch to dark theme');

  if (!config.bootData?.assets) {
    return null;
  }

  return (
    <ToolbarButton
      iconOnly
      icon={isDark ? 'sun' : 'moon'}
      aria-label={label}
      tooltip={label}
      onClick={() => toggleTheme(false)}
    />
  );
});
