import { render, screen } from 'test/test-utils';
import userEvent from '@testing-library/user-event';

import { useTheme2 } from '@grafana/ui';
import { toggleTheme } from 'app/core/services/theme';

import { ThemeToggleButton } from './ThemeToggleButton';

jest.mock('@grafana/ui', () => ({
  ...jest.requireActual('@grafana/ui'),
  useTheme2: jest.fn(),
}));

jest.mock('app/core/services/theme', () => ({
  ...jest.requireActual('app/core/services/theme'),
  toggleTheme: jest.fn(),
}));

const mockedUseTheme2 = jest.mocked(useTheme2);
const mockedToggleTheme = jest.mocked(toggleTheme);

describe('ThemeToggleButton', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render dark mode switch label when current theme is light', () => {
    mockedUseTheme2.mockReturnValue({ isDark: false } as ReturnType<typeof useTheme2>);

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to dark mode' })).toBeInTheDocument();
  });

  it('should render light mode switch label when current theme is dark', () => {
    mockedUseTheme2.mockReturnValue({ isDark: true } as ReturnType<typeof useTheme2>);

    render(<ThemeToggleButton />);

    expect(screen.getByRole('button', { name: 'Switch to light mode' })).toBeInTheDocument();
  });

  it('should toggle and persist theme when clicked', async () => {
    mockedUseTheme2.mockReturnValue({ isDark: true } as ReturnType<typeof useTheme2>);
    const user = userEvent.setup();

    render(<ThemeToggleButton />);

    await user.click(screen.getByRole('button', { name: 'Switch to light mode' }));

    expect(mockedToggleTheme).toHaveBeenCalledWith(false);
  });
});
