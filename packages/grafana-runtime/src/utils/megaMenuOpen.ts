import { createStructuredLogger } from '@grafana/data';
const structuredLogger = createStructuredLogger('packages/grafana-runtime/src/utils/megaMenuOpen');

type MegaMenuOpenHook = () => Readonly<[boolean, (open: boolean, persist?: boolean) => void]>;

let megaMenuOpenHook: MegaMenuOpenHook | undefined = undefined;

export const setMegaMenuOpenHook = (hook: MegaMenuOpenHook) => {
  megaMenuOpenHook = hook;
};

/**
 * Guidelines:
 * - Should only be used in very specific circumstances where the mega menu needs to be opened or closed programmatically.
 */
export const useMegaMenuOpen: MegaMenuOpenHook = () => {
  if (!megaMenuOpenHook) {
    if (process.env.NODE_ENV !== 'production') {
      throw new Error('useMegaMenuOpen hook not found in @grafana/runtime');
    }
    return [false, () => structuredLogger.error('MegaMenuOpen hook not found')];
  }

  return megaMenuOpenHook();
};
