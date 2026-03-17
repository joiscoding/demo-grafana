import { memo, useEffect, useState } from 'react';

import { t } from '@grafana/i18n';
import { ThemeChangedEvent, config } from '@grafana/runtime';
import { ToolbarButton } from '@grafana/ui';
import { appEvents } from 'app/core/app_events';
import { toggleTheme } from 'app/core/services/theme';

export const ThemeToggleButton = memo(function ThemeToggleButton() {
  const [mode, setMode] = useState(config.theme2.colors.mode);

  useEffect(() => {
    const sub = appEvents.subscribe(ThemeChangedEvent, (event) => {
      setMode(event.payload.colors.mode);
    });

    return () => sub.unsubscribe();
  }, []);

  const isDarkMode = mode === 'dark';
  const label = isDarkMode
    ? t('navigation.theme.toggle-to-light.aria-label', 'Switch to light mode')
    : t('navigation.theme.toggle-to-dark.aria-label', 'Switch to dark mode');

  return (
    <ToolbarButton
      key={isDarkMode ? 'dark' : 'light'}
      iconOnly
      icon={isDarkMode ? 'toggle-on' : 'toggle-off'}
      aria-label={label}
      tooltip={label}
      variant={isDarkMode ? 'active' : 'default'}
      onClick={() => void toggleTheme(false)}
    />
  );
});
