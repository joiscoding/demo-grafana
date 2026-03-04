import {
  featureToggleOverridesLocalStorageKey,
  parseFeatureToggleOverrides,
  readFeatureToggleOverridesFromLocalStorage,
  stringifyFeatureToggleOverrides,
  writeFeatureToggleOverridesToLocalStorage,
} from './localFeatureToggles';

describe('localFeatureToggles', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  describe('parseFeatureToggleOverrides', () => {
    it('returns empty map for empty values', () => {
      expect(parseFeatureToggleOverrides(null)).toEqual({});
      expect(parseFeatureToggleOverrides('')).toEqual({});
      expect(parseFeatureToggleOverrides('   ')).toEqual({});
    });

    it('parses true and false values', () => {
      expect(parseFeatureToggleOverrides('dashboardScene=1,newLogsPanel=true,flagX=false')).toEqual({
        dashboardScene: true,
        newLogsPanel: true,
        flagX: false,
      });
    });

    it('ignores invalid key entries', () => {
      expect(parseFeatureToggleOverrides(', ,=1,=false,valid=1')).toEqual({
        valid: true,
      });
    });
  });

  describe('stringifyFeatureToggleOverrides', () => {
    it('serializes entries with stable ordering', () => {
      expect(
        stringifyFeatureToggleOverrides({
          zFlag: false,
          aFlag: true,
        })
      ).toBe('aFlag=true,zFlag=false');
    });
  });

  describe('localStorage helpers', () => {
    it('writes and reads overrides', () => {
      writeFeatureToggleOverridesToLocalStorage({
        dashboardScene: true,
        panelGroupBy: false,
      });

      expect(window.localStorage.getItem(featureToggleOverridesLocalStorageKey)).toBe(
        'dashboardScene=true,panelGroupBy=false'
      );
      expect(readFeatureToggleOverridesFromLocalStorage()).toEqual({
        dashboardScene: true,
        panelGroupBy: false,
      });
    });

    it('clears the storage key for empty overrides', () => {
      window.localStorage.setItem(featureToggleOverridesLocalStorageKey, 'something=true');

      writeFeatureToggleOverridesToLocalStorage({});

      expect(window.localStorage.getItem(featureToggleOverridesLocalStorageKey)).toBeNull();
    });
  });
});
