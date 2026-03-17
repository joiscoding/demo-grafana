import { act, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render } from 'test/test-utils';

import { ThemeChangedEvent, config } from '@grafana/runtime';
import { appEvents } from 'app/core/app_events';

import { toggleTheme } from '../../../services/theme';

import { ThemeToggleButton } from './ThemeToggleButton';

jest.mock('../../../services/theme', () => ({
  toggleTheme: jest.fn(),
}));

describe('ThemeToggleButton', () => {
  const originalTheme = config.theme2;

  beforeEach(() => {
    config.theme2 = { ...config.theme2, colors: { ...config.theme2.colors, mode: 'light' } };
  });

  afterEach(() => {
    config.theme2 = originalTheme;
    jest.clearAllMocks();
  });

  it('renders light mode state when current mode is light', () => {
    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Toggle dark and light mode' })).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByTestId('icon-toggle-off')).toBeInTheDocument();
  });

  it('renders dark mode state when current mode is dark', () => {
    config.theme2 = { ...config.theme2, colors: { ...config.theme2.colors, mode: 'dark' } };

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Toggle dark and light mode' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByTestId('icon-toggle-on')).toBeInTheDocument();
  });

  it('updates the button state when theme mode changes', async () => {
    render(<ThemeToggleButton />);
    expect(screen.getByTestId('icon-toggle-off')).toBeInTheDocument();

    act(() => {
      appEvents.publish(
        new ThemeChangedEvent({ ...config.theme2, colors: { ...config.theme2.colors, mode: 'dark' } })
      );
    });

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Toggle dark and light mode' })).toHaveAttribute('aria-pressed', 'true');
      expect(screen.getByTestId('icon-toggle-on')).toBeInTheDocument();
    });
  });

  it('toggles persisted user theme when clicked', async () => {
    render(<ThemeToggleButton />);
    await userEvent.click(screen.getByRole('button', { name: 'Toggle dark and light mode' }));

    expect(toggleTheme).toHaveBeenCalledWith(false);
  });
});
