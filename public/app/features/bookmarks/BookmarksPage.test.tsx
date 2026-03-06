import { render, screen } from 'test/test-utils';

import { NavModelItem } from '@grafana/data';

import { usePinnedItems } from 'app/core/components/AppChrome/MegaMenu/hooks';

import { BookmarksPage } from './BookmarksPage';

jest.mock('app/core/components/AppChrome/MegaMenu/hooks', () => ({
  ...jest.requireActual('app/core/components/AppChrome/MegaMenu/hooks'),
  usePinnedItems: jest.fn(),
}));

const mockedUsePinnedItems = jest.mocked(usePinnedItems);

const navTree: NavModelItem[] = [
  {
    id: 'cfg',
    text: 'Administration',
    url: '/admin',
    subTitle: 'Manage Grafana instance',
  },
];

describe('BookmarksPage', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it('shows empty state when there are no pinned items', async () => {
    mockedUsePinnedItems.mockReturnValue([]);

    render(<BookmarksPage />, {
      preloadedState: {
        navBarTree: navTree,
        navIndex: {
          bookmarks: {
            id: 'bookmarks',
            text: 'Bookmarks',
            url: '/bookmarks',
          },
        },
      },
    });

    expect(
      await screen.findByText('It looks like you haven’t created any bookmarks yet')
    ).toBeInTheDocument();
  });

  it('renders pinned bookmark cards for valid items', async () => {
    mockedUsePinnedItems.mockReturnValue(['/admin']);

    render(<BookmarksPage />, {
      preloadedState: {
        navBarTree: navTree,
        navIndex: {
          bookmarks: {
            id: 'bookmarks',
            text: 'Bookmarks',
            url: '/bookmarks',
          },
        },
      },
    });

    expect(await screen.findByRole('link', { name: 'Administration' })).toHaveAttribute('href', '/admin');
    expect(screen.queryByText('It looks like you haven’t created any bookmarks yet')).not.toBeInTheDocument();
  });
});
