import { config } from '@grafana/runtime';

import { LABS_NAV_ID, withLabsNavItem } from './nav';

describe('withLabsNavItem', () => {
  const originalIsSignedIn = config.bootData.user.isSignedIn;

  afterEach(() => {
    config.bootData.user.isSignedIn = originalIsSignedIn;
  });

  it('adds the Labs nav item for signed-in users before Administration', () => {
    config.bootData.user.isSignedIn = true;

    const navTree = withLabsNavItem([
      { id: 'home', text: 'Home', url: '/' },
      { id: 'cfg', text: 'Administration', url: '/admin' },
    ]);

    expect(navTree.map((item) => item.id)).toEqual(['home', LABS_NAV_ID, 'cfg']);
    expect(navTree[1]).toMatchObject({
      text: 'Labs',
      url: '/labs',
    });
  });

  it('does not add the Labs nav item for signed-out users', () => {
    config.bootData.user.isSignedIn = false;

    const navTree = withLabsNavItem([{ id: 'home', text: 'Home', url: '/' }]);

    expect(navTree).toEqual([{ id: 'home', text: 'Home', url: '/' }]);
  });
});
