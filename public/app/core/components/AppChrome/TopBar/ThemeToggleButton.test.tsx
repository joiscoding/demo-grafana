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

  it('shows a dark mode label when current mode is light', () => {
    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();
  });

  it('shows a light mode label when current mode is dark', () => {
    config.theme2 = { ...config.theme2, colors: { ...config.theme2.colors, mode: 'dark' } };

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
  });

  it('updates the label when theme mode changes', async () => {
    render(<ThemeToggleButton />);
    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();

    act(() => {
      appEvents.publish(
        new ThemeChangedEvent({ ...config.theme2, colors: { ...config.theme2.colors, mode: 'dark' } })
      );
    });

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
    });
  });

  it('toggles persisted user theme when clicked', async () => {
    render(<ThemeToggleButton />);
    await userEvent.click(screen.getByRole('button', { name: 'Switch to dark mode' }));

    expect(toggleTheme).toHaveBeenCalledWith(false);
  });
});
