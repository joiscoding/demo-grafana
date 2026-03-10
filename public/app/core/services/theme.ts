import { getThemeById } from '@grafana/data/internal';
import { config, ThemeChangedEvent } from '@grafana/runtime';

import { appEvents } from '../app_events';
import { contextSrv } from '../services/context_srv';

import { PreferencesService } from './PreferencesService';

const THEME_BODY_CLASSES = ['theme-dark', 'theme-light', 'theme-system'];

function setBodyThemeClass(mode: 'light' | 'dark') {
  document.body.classList.remove(...THEME_BODY_CLASSES);
  document.body.classList.add(`theme-${mode}`);
}

function isThemeStylesheetLink(link: HTMLLinkElement, mode: 'light' | 'dark') {
  if (!link.href) {
    return false;
  }

  const configuredThemeStylesheet = config.bootData.assets[mode];
  if (configuredThemeStylesheet) {
    const configuredThemeHref = new URL(configuredThemeStylesheet, window.location.href).href;

    if (link.href === configuredThemeHref) {
      return true;
    }
  }

  // Keep the old matcher as a fallback to avoid regressions.
  return link.href.includes(`build/grafana.${mode}`);
}

export async function changeTheme(themeId: string, runtimeOnly?: boolean) {
  const oldTheme = config.theme2;

  const newTheme = getThemeById(themeId);

  setBodyThemeClass(newTheme.colors.mode);
  appEvents.publish(new ThemeChangedEvent(newTheme));

  // Add css file for new theme
  if (oldTheme.colors.mode !== newTheme.colors.mode) {
    const newCssLink = document.createElement('link');
    newCssLink.rel = 'stylesheet';
    newCssLink.href = config.bootData.assets[newTheme.colors.mode];
    newCssLink.onload = () => {
      // Remove old css file
      const bodyLinks = document.getElementsByTagName('link');
      for (let i = 0; i < bodyLinks.length; i++) {
        const link = bodyLinks[i];

        if (link !== newCssLink && isThemeStylesheetLink(link, oldTheme.colors.mode)) {
          // Remove existing link once the new css has loaded to avoid flickering
          // If we add new css at the same time we remove current one the page will be rendered without css
          // As the new css file is loading
          link.remove();
        }
      }
    };
    document.head.insertBefore(newCssLink, document.head.firstChild);
  }

  if (runtimeOnly) {
    return;
  }

  if (!contextSrv.isSignedIn) {
    return;
  }

  // Persist new theme
  const service = new PreferencesService('user');
  await service.patch({
    theme: themeId,
  });
}

export async function toggleTheme(runtimeOnly: boolean) {
  const currentTheme = config.theme2;
  changeTheme(currentTheme.isDark ? 'light' : 'dark', runtimeOnly);
}
