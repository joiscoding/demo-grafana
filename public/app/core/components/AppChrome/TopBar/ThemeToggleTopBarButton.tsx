import { memo, useCallback, useState } from 'react';

import { selectors } from '@grafana/e2e-selectors';
import { t } from '@grafana/i18n';
import { reportInteraction } from '@grafana/runtime';
import { ToolbarButton, useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleTopBarButton = memo(function ThemeToggleTopBarButton() {
  const theme = useTheme2();
  const [busy, setBusy] = useState(false);

  const onToggle = useCallback(() => {
    if (busy) {
      return;
    }
    setBusy(true);
    const nextMode = theme.isDark ? 'light' : 'dark';
    reportInteraction('grafana_nav_toolbar_theme_toggle', {
      fromMode: theme.colors.mode,
      toMode: nextMode,
    });
    void toggleTheme(false).finally(() => setBusy(false));
  }, [busy, theme.colors.mode, theme.isDark]);

  const label = theme.isDark
    ? t('navigation.theme-toggle.switch-to-light', 'Switch to light theme')
    : t('navigation.theme-toggle.switch-to-dark', 'Switch to dark theme');

  const icon = theme.isDark ? 'monitor' : 'star';

  return (
    <ToolbarButton
      type="button"
      narrow
      iconOnly
      icon={icon}
      aria-label={label}
      tooltip={label}
      disabled={busy}
      onClick={onToggle}
      data-testid={selectors.components.NavToolbar.themeToggle}
    />
  );
});
