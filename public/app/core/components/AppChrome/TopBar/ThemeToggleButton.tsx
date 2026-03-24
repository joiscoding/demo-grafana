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
  const label = t('navigation.theme.toggle.aria-label', 'Toggle dark and light mode');

  return (
    <ToolbarButton
      key={isDarkMode ? 'dark' : 'light'}
      iconOnly
      icon={isDarkMode ? 'toggle-on' : 'toggle-off'}
      aria-label={label}
      aria-pressed={isDarkMode}
      variant={isDarkMode ? 'active' : 'default'}
      onClick={() => void toggleTheme(false)}
    />
  );
});
