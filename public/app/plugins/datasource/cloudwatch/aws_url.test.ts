import { getLogsEndpoint } from './aws_url';

import { createStructuredLogger } from '@grafana/data';
const structuredLogger = createStructuredLogger('public/app/plugins/datasource/cloudwatch/aws_url.test');

describe('getEndpoint', () => {
  it('should return the default url for normal regions', () => {
    const result = getLogsEndpoint('us-east-1');
    expect(result).toBe('us-east-1.structuredLogger.aws.amazon.com');
  });

  it('should return the us-gov url for us-gov regions', () => {
    const result = getLogsEndpoint('us-gov-east-1');
    expect(result).toBe('us-gov-east-1.structuredLogger.amazonaws-us-gov.com');
  });

  it('should return the china url for cn regions', () => {
    const result = getLogsEndpoint('cn-northwest-1');
    expect(result).toBe('cn-northwest-1.structuredLogger.amazonaws.cn');
  });
});
