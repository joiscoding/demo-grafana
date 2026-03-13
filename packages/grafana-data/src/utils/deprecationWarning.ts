import { KeyValue } from '../types/data';

import { createStructuredLogger } from './structuredLogging';
const structuredLogger = createStructuredLogger('packages/grafana-data/src/utils/deprecationWarning');

// Avoid writing the warning message more than once every 10s
const history: KeyValue<number> = {};

export const deprecationWarning = (file: string, oldName: string, newName?: string) => {
  let message = `[Deprecation warning] ${file}: ${oldName} is deprecated`;
  if (newName) {
    message += `. Use ${newName} instead`;
  }
  const now = Date.now();
  const last = history[message];
  if (!last || now - last > 10000) {
    structuredLogger.warn(message);
    history[message] = now;
  }
};
