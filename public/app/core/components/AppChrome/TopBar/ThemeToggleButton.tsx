import { memo } from 'react';

import { t } from '@grafana/i18n';
import { ToolbarButton, useTheme2 } from '@grafana/ui';

import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleButton = memo(function ThemeToggleButton() {
  const theme = useTheme2();
  const label = theme.isDark
    ? t('navigation.theme-toggle.switch-to-light', 'Switch to light theme')
    : t('navigation.theme-toggle.switch-to-dark', 'Switch to dark theme');

  const onClick = () => {
    void toggleTheme(false);
  };

  return (
    <ToolbarButton iconOnly icon="adjust-circle" aria-label={label} tooltip={label} onClick={onClick} />
  );
});
