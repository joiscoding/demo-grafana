import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render } from 'test/test-utils';

import { config } from '@grafana/runtime';

import { toggleTheme } from '../../../services/theme';

import { ThemeToggleButton } from './ThemeToggleButton';

jest.mock('../../../services/theme', () => ({
  toggleTheme: jest.fn(),
}));

describe('ThemeToggleButton', () => {
  const originalTheme = config.theme2;

  afterEach(() => {
    config.theme2 = originalTheme;
    jest.clearAllMocks();
  });

  it('shows a dark mode label when current mode is light', () => {
    config.theme2 = { ...config.theme2, isDark: false };

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();
  });

  it('shows a light mode label when current mode is dark', () => {
    config.theme2 = { ...config.theme2, isDark: true };

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
  });

  it('toggles persisted user theme when clicked', async () => {
    config.theme2 = { ...config.theme2, isDark: false };

    render(<ThemeToggleButton />);
    await userEvent.click(screen.getByRole('button', { name: 'Switch to dark mode' }));

    expect(toggleTheme).toHaveBeenCalledWith(false);
  });
});
