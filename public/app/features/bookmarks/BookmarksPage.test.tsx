import { screen } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { render } from 'test/test-utils';

import type { NavModelItem } from '@grafana/data';

import { BookmarksPage } from './BookmarksPage';

const mockedUsePinnedItems = jest.fn<() => string[], []>();
const mockedFindByUrl = jest.fn<(nodes: NavModelItem[], url: string) => NavModelItem | null>();

jest.mock('app/core/components/AppChrome/MegaMenu/hooks', () => ({
  usePinnedItems: () => mockedUsePinnedItems(),
}));

jest.mock('app/core/components/AppChrome/MegaMenu/utils', () => ({
  ...jest.requireActual('app/core/components/AppChrome/MegaMenu/utils'),
  findByUrl: (nodes: NavModelItem[], url: string) => mockedFindByUrl(nodes, url),
}));

jest.mock('app/core/components/Page/Page', () => {
  const PageComponent = ({ children }: PropsWithChildren<{}>) => <div>{children}</div>;
  PageComponent.Contents = ({ children }: PropsWithChildren<{}>) => <div>{children}</div>;
  return { Page: PageComponent };
});

describe('BookmarksPage', () => {
  const adminItem: NavModelItem = {
    id: 'admin',
    text: 'Administration',
    url: '/admin',
    subTitle: 'System settings',
  };

  beforeEach(() => {
    jest.clearAllMocks();
    mockedFindByUrl.mockImplementation((_nodes, url) => (url === '/admin' ? adminItem : null));
  });

  it('shows empty state when no bookmarks are pinned', () => {
    mockedUsePinnedItems.mockReturnValue([]);

    render(<BookmarksPage />);

    expect(screen.getByText('It looks like you haven’t created any bookmarks yet')).toBeInTheDocument();
  });

  it('renders bookmarked items with valid URLs', () => {
    mockedUsePinnedItems.mockReturnValue(['/admin']);

    render(<BookmarksPage />);

    expect(screen.getByRole('link', { name: 'Administration' })).toBeInTheDocument();
    expect(screen.getByText('System settings')).toBeInTheDocument();
  });

  it('does not crash when invalid bookmarks are present', () => {
    mockedUsePinnedItems.mockReturnValue(['/admin', '/missing']);

    render(<BookmarksPage />);

    expect(screen.getByRole('link', { name: 'Administration' })).toBeInTheDocument();
    expect(screen.getByText('System settings')).toBeInTheDocument();
  });
});
