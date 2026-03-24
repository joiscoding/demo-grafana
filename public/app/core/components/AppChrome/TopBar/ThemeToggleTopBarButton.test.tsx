import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render } from 'test/test-utils';

import { toggleTheme } from 'app/core/services/theme';

import { ThemeToggleTopBarButton } from './ThemeToggleTopBarButton';

jest.mock('app/core/services/theme', () => ({
  toggleTheme: jest.fn().mockResolvedValue(undefined),
}));

describe('ThemeToggleTopBarButton', () => {
  it('calls toggleTheme when clicked', async () => {
    const user = userEvent.setup();
    render(<ThemeToggleTopBarButton />);

    await user.click(screen.getByRole('button', { name: /switch to (dark|light) mode/i }));

    expect(toggleTheme).toHaveBeenCalledWith(false);
  });
});
