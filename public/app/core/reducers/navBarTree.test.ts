import { NavModelItem } from '@grafana/data';

import { addLabsSectionToNav } from './navBarTree';

describe('addLabsSectionToNav', () => {
  it('adds Labs after Connections when signed in', () => {
    const navTree: NavModelItem[] = [
      { id: 'connections', text: 'Connections', url: '/connections' },
      { id: 'cfg', text: 'Administration', url: '/admin' },
    ];

    const result = addLabsSectionToNav(navTree, true, '');

    expect(result.map((item) => item.id)).toEqual(['connections', 'labs', 'cfg']);
    expect(result[1]).toMatchObject({ id: 'labs', url: '/labs', isNew: true });
  });

  it('adds Labs before Administration when Connections is missing', () => {
    const navTree: NavModelItem[] = [{ id: 'cfg', text: 'Administration', url: '/admin' }];

    const result = addLabsSectionToNav(navTree, true, '');

    expect(result.map((item) => item.id)).toEqual(['labs', 'cfg']);
  });

  it('does not add Labs when user is signed out', () => {
    const navTree: NavModelItem[] = [{ id: 'connections', text: 'Connections', url: '/connections' }];

    const result = addLabsSectionToNav(navTree, false, '');

    expect(result).toEqual(navTree);
  });

  it('does not add duplicate Labs section', () => {
    const navTree: NavModelItem[] = [
      { id: 'connections', text: 'Connections', url: '/connections' },
      { id: 'labs', text: 'Labs', url: '/labs' },
    ];

    const result = addLabsSectionToNav(navTree, true, '');

    expect(result).toEqual(navTree);
  });
});
