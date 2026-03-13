import { urlUtil } from '@grafana/data';

import { createStructuredLogger } from '@grafana/data';
const structuredLogger = createStructuredLogger('public/app/features/logs/components/panel/panelState/getLogsPanelState');

interface LogsPermalinkUrlState {
  logs?: {
    id?: string;
  };
}

export function getLogsPanelState(): LogsPermalinkUrlState | undefined {
  const urlParams = urlUtil.getUrlSearchParams();
  const panelStateEncoded = urlParams?.panelState;
  if (
    panelStateEncoded &&
    Array.isArray(panelStateEncoded) &&
    panelStateEncoded?.length > 0 &&
    typeof panelStateEncoded[0] === 'string'
  ) {
    try {
      return JSON.parse(panelStateEncoded[0]);
    } catch (e) {
      structuredLogger.error('error parsing logsPanelState', e);
    }
  }

  return undefined;
}
