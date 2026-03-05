import { getAppRoutes } from './routes';

jest.mock('app/features/plugins/routes', () => ({
  getAppPluginRoutes: () => [],
}));

describe('getAppRoutes', () => {
  it('includes the Labs route', () => {
    const routes = getAppRoutes();
    const labsRoute = routes.find((route) => route.path === '/labs');

    expect(labsRoute).toBeDefined();
  });
});
