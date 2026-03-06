import { screen } from '@testing-library/react';
import { render } from 'test/test-utils';

import { BookmarksPage } from './BookmarksPage';

jest.mock('app/core/components/AppChrome/MegaMenu/hooks', () => ({
  usePinnedItems: jest.fn(),
}));

jest.mock('app/core/components/AppChrome/MegaMenu/utils', () => ({
  findByUrl: jest.fn(),
}));

jest.mock('app/types/store', () => ({
  ...jest.requireActual('app/types/store'),
  useSelector: jest.fn(),
}));

const mockUsePinnedItems = jest.mocked(require('app/core/components/AppChrome/MegaMenu/hooks').usePinnedItems);
const mockFindByUrl = jest.mocked(require('app/core/components/AppChrome/MegaMenu/utils').findByUrl);
const mockUseSelector = jest.mocked(require('app/types/store').useSelector);

describe('BookmarksPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseSelector.mockReturnValue([]);
  });

  it('renders empty state when no bookmarks are pinned', () => {
    mockUsePinnedItems.mockReturnValue([]);

    render(<BookmarksPage />);

    expect(screen.getByText('It looks like you haven’t created any bookmarks yet')).toBeInTheDocument();
  });

  it('renders bookmarked items when valid bookmarks are pinned', () => {
    mockUsePinnedItems.mockReturnValue(['/dashboards']);
    mockFindByUrl.mockReturnValue({
      id: 'dashboards',
      text: 'Dashboards',
      url: '/dashboards',
      subTitle: 'Browse dashboards',
    });

    render(<BookmarksPage />);

    expect(screen.getByRole('heading', { name: 'Dashboards' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Dashboards' })).toHaveAttribute('href', '/dashboards');
  });
});
