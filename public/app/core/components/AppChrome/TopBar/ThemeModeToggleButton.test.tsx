import { render, screen } from 'test/test-utils';

import { createTheme } from '@grafana/data';
import { reportInteraction } from '@grafana/runtime';
import { toggleTheme } from 'app/core/services/theme';
import { ThemeProvider } from 'app/core/utils/ConfigProvider';

import { ThemeModeToggleButton } from './ThemeModeToggleButton';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  reportInteraction: jest.fn(),
}));

jest.mock('app/core/services/theme', () => ({
  toggleTheme: jest.fn(),
}));

const mockReportInteraction = jest.mocked(reportInteraction);
const mockToggleTheme = jest.mocked(toggleTheme);

describe('ThemeModeToggleButton', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('switches from dark mode to light mode', async () => {
    const darkTheme = createTheme({ colors: { mode: 'dark' } });
    const { user } = render(
      <ThemeProvider value={darkTheme}>
        <ThemeModeToggleButton />
      </ThemeProvider>,
      {
        grafanaContext: {
          config: {
            theme2: darkTheme,
            featureToggles: {},
          },
        },
      }
    );

    const button = screen.getByRole('button', { name: /switch to light mode/i });

    await user.click(button);

    expect(mockReportInteraction).toHaveBeenCalledWith('grafana_theme_mode_toggled', {
      fromTheme: 'dark',
      toTheme: 'light',
      placement: 'top_bar_right',
    });
    expect(mockToggleTheme).toHaveBeenCalledWith(false);
  });

  it('switches from light mode to dark mode', async () => {
    const lightTheme = createTheme({ colors: { mode: 'light' } });
    const { user } = render(
      <ThemeProvider value={lightTheme}>
        <ThemeModeToggleButton />
      </ThemeProvider>,
      {
        grafanaContext: {
          config: {
            theme2: lightTheme,
            featureToggles: {},
          },
        },
      }
    );

    const button = screen.getByRole('button', { name: /switch to dark mode/i });

    await user.click(button);

    expect(mockReportInteraction).toHaveBeenCalledWith('grafana_theme_mode_toggled', {
      fromTheme: 'light',
      toTheme: 'dark',
      placement: 'top_bar_right',
    });
    expect(mockToggleTheme).toHaveBeenCalledWith(false);
  });
});
