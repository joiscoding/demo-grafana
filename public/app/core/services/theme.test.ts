const darkTheme = {
  colors: { mode: 'dark' },
  isDark: true,
  isLight: false,
  v1: {},
} as const;

const lightTheme = {
  colors: { mode: 'light' },
  isDark: false,
  isLight: true,
  v1: {},
} as const;

const mockGetThemeById = jest.fn((id: string) => (id === 'light' ? lightTheme : darkTheme));
const mockAppEventsPublish = jest.fn();

const mockConfig = {
  theme2: darkTheme,
  bootData: {
    assets: {
      dark: '/public/build/custom.dark.css',
      light: '/public/build/custom.light.css',
    },
  },
};

jest.mock('@grafana/data/internal', () => ({
  getThemeById: (id: string) => mockGetThemeById(id),
}));

jest.mock('@grafana/runtime', () => ({
  config: mockConfig,
  ThemeChangedEvent: class ThemeChangedEvent {
    constructor(public payload: unknown) {}
  },
}));

jest.mock('../app_events', () => ({
  appEvents: {
    publish: (...args: unknown[]) => mockAppEventsPublish(...args),
  },
}));

jest.mock('../services/context_srv', () => ({
  contextSrv: {
    isSignedIn: false,
  },
}));

jest.mock('./PreferencesService', () => ({
  PreferencesService: jest.fn().mockImplementation(() => ({
    patch: jest.fn(),
  })),
}));

import { config } from '@grafana/runtime';

import { changeTheme } from './theme';

describe('changeTheme', () => {
  const originalBodyClassName = document.body.className;

  beforeEach(() => {
    config.theme2 = darkTheme;
    config.bootData.assets.dark = '/public/build/custom.dark.css';
    config.bootData.assets.light = '/public/build/custom.light.css';
    document.body.className = 'theme-dark';
    document.querySelectorAll('link').forEach((link) => link.remove());
    mockGetThemeById.mockClear();
    mockAppEventsPublish.mockClear();
  });

  afterEach(() => {
    document.body.className = originalBodyClassName;
    document.querySelectorAll('link').forEach((link) => link.remove());
  });

  it('updates body class and removes the old configured stylesheet after loading the new one', async () => {
    const oldCssLink = document.createElement('link');
    oldCssLink.rel = 'stylesheet';
    oldCssLink.href = config.bootData.assets.dark;
    document.head.appendChild(oldCssLink);

    await changeTheme('light', true);

    expect(document.body.classList.contains('theme-light')).toBe(true);
    expect(document.body.classList.contains('theme-dark')).toBe(false);

    const newCssHref = new URL(config.bootData.assets.light, window.location.href).href;
    const newCssLink = Array.from(document.querySelectorAll('link')).find((link) => link.href === newCssHref);

    expect(newCssLink).toBeDefined();

    newCssLink!.dispatchEvent(new Event('load'));

    expect(document.head.contains(oldCssLink)).toBe(false);
  });

  it('keeps legacy stylesheet matching as a fallback when asset names are customized', async () => {
    config.bootData.assets.dark = '/public/build/custom-dark-asset.css';
    config.bootData.assets.light = '/public/build/custom-light-asset.css';

    const oldCssLink = document.createElement('link');
    oldCssLink.rel = 'stylesheet';
    oldCssLink.href = '/public/build/grafana.dark.legacy.css';
    document.head.appendChild(oldCssLink);

    await changeTheme('light', true);

    const newCssHref = new URL(config.bootData.assets.light, window.location.href).href;
    const newCssLink = Array.from(document.querySelectorAll('link')).find((link) => link.href === newCssHref);

    expect(newCssLink).toBeDefined();

    newCssLink!.dispatchEvent(new Event('load'));

    expect(document.head.contains(oldCssLink)).toBe(false);
  });
});
