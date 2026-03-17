import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render } from 'test/test-utils';

import { useTheme2 } from '@grafana/ui';

import { toggleTheme } from '../../../services/theme';

import { ThemeToggleButton } from './ThemeToggleButton';

jest.mock('../../../services/theme', () => ({
  toggleTheme: jest.fn(),
}));

jest.mock('@grafana/ui', () => ({
  ...jest.requireActual('@grafana/ui'),
  useTheme2: jest.fn(),
}));

describe('ThemeToggleButton', () => {
  const mockedUseTheme2 = jest.mocked(useTheme2);

  beforeEach(() => {
    mockedUseTheme2.mockReturnValue({ isDark: false } as any);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('shows a dark mode label when current mode is light', () => {
    mockedUseTheme2.mockReturnValue({ isDark: false } as any);

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();
  });

  it('shows a light mode label when current mode is dark', () => {
    mockedUseTheme2.mockReturnValue({ isDark: true } as any);

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
  });

  it('updates the label when theme mode changes', () => {
    const { rerender } = render(<ThemeToggleButton />);
    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();

    mockedUseTheme2.mockReturnValue({ isDark: true } as any);
    rerender(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
  });

  it('toggles persisted user theme when clicked', async () => {
    render(<ThemeToggleButton />);
    await userEvent.click(screen.getByRole('button', { name: 'Switch to dark mode' }));

    expect(toggleTheme).toHaveBeenCalledWith(false);
  });
});
