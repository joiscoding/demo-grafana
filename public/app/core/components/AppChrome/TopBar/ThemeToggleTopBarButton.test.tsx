import { screen } from '@testing-library/react';
import { render } from 'test/test-utils';

import { ThemeContext } from '@grafana/data';
import { getThemeById } from '@grafana/data/internal';

import * as themeService from 'app/core/services/theme';

import { ThemeToggleTopBarButton } from './ThemeToggleTopBarButton';

describe('ThemeToggleTopBarButton', () => {
  it('calls toggleTheme when clicked', async () => {
    const toggleSpy = jest.spyOn(themeService, 'toggleTheme').mockResolvedValue(undefined);
    const dark = getThemeById('dark');

    const { user } = render(
      <ThemeContext.Provider value={dark}>
        <ThemeToggleTopBarButton />
      </ThemeContext.Provider>
    );

    await user.click(screen.getByRole('button', { name: /switch to light theme/i }));
    expect(toggleSpy).toHaveBeenCalledWith(false);
    toggleSpy.mockRestore();
  });
});
