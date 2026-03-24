import { memo, useCallback, useState } from 'react';

import { selectors } from '@grafana/e2e-selectors';
import { t } from '@grafana/i18n';
import { reportInteraction } from '@grafana/runtime';
import { ToolbarButton, useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleTopBarButton = memo(function ThemeToggleTopBarButton() {
  const theme = useTheme2();
  const [isToggling, setIsToggling] = useState(false);

  const onToggle = useCallback(async () => {
    if (isToggling) {
      return;
    }
    setIsToggling(true);
    try {
      reportInteraction('grafana_top_bar_theme_toggle_clicked', {
        from: theme.isDark ? 'dark' : 'light',
      });
      await toggleTheme(false);
    } finally {
      setIsToggling(false);
    }
  }, [isToggling, theme.isDark]);

  const nextIsDark = !theme.isDark;

  return (
    <ToolbarButton
      data-testid={selectors.components.NavToolbar.themeToggle}
      iconOnly
      icon={theme.isDark ? 'sun' : 'moon'}
      disabled={isToggling}
      aria-label={
        nextIsDark
          ? t('navigation.theme-toggle.switch-to-dark-aria', 'Switch to dark mode')
          : t('navigation.theme-toggle.switch-to-light-aria', 'Switch to light mode')
      }
      tooltip={
        nextIsDark
          ? t('navigation.theme-toggle.switch-to-dark-tooltip', 'Switch to dark mode')
          : t('navigation.theme-toggle.switch-to-light-tooltip', 'Switch to light mode')
      }
      onClick={onToggle}
    />
  );
});
