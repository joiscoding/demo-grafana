import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render } from 'test/test-utils';

import { toggleTheme } from 'app/core/services/theme';

import { ThemeToggleButton } from './ThemeToggleButton';

jest.mock('app/core/services/theme', () => ({
  toggleTheme: jest.fn(),
}));

describe('ThemeToggleButton', () => {
  it('renders a button with an accessible label', () => {
    render(<ThemeToggleButton />);
    expect(screen.getByRole('button', { name: /switch to (light|dark) theme/i })).toBeInTheDocument();
  });

  it('calls toggleTheme when clicked', async () => {
    const user = userEvent.setup();
    render(<ThemeToggleButton />);

    await user.click(screen.getByRole('button', { name: /switch to (light|dark) theme/i }));

    expect(toggleTheme).toHaveBeenCalledWith(false);
  });
});
