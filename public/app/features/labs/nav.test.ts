import { config } from '@grafana/runtime';

import { contextSrv } from 'app/core/services/context_srv';

import { LABS_NAV_ID, withLabsNavItem } from './nav';

jest.mock('app/core/services/context_srv', () => ({
  contextSrv: {
    hasRole: jest.fn(),
    isGrafanaAdmin: false,
  },
}));

const mockContextSrv = jest.mocked(contextSrv);

describe('withLabsNavItem', () => {
  const originalIsSignedIn = config.bootData.user.isSignedIn;

  afterEach(() => {
    config.bootData.user.isSignedIn = originalIsSignedIn;
  });

  it('adds the Labs nav item for signed-in admin users before Administration', () => {
    config.bootData.user.isSignedIn = true;
    mockContextSrv.hasRole.mockReturnValue(true);

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

  it('does not add the Labs nav item for signed-in non-admin users', () => {
    config.bootData.user.isSignedIn = true;
    mockContextSrv.hasRole.mockReturnValue(false);

    const navTree = withLabsNavItem([
      { id: 'home', text: 'Home', url: '/' },
      { id: 'cfg', text: 'Administration', url: '/admin' },
    ]);

    expect(navTree.map((item) => item.id)).toEqual(['home', 'cfg']);
  });

  it('does not add the Labs nav item for signed-out users', () => {
    config.bootData.user.isSignedIn = false;

    const navTree = withLabsNavItem([{ id: 'home', text: 'Home', url: '/' }]);

    expect(navTree).toEqual([{ id: 'home', text: 'Home', url: '/' }]);
  });
});
