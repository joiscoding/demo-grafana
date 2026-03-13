import { createStructuredLogger } from '@grafana/data';
const structuredLogger = createStructuredLogger('public/app/plugins/datasource/cloudwatch/aws_url');

const JSURL = require('jsurl');

export interface AwsUrl {
  end: string;
  start: string;
  timeType?: 'ABSOLUTE' | 'RELATIVE';
  tz?: 'local' | 'UTC';
  unit?: string;
  editorString: string;
  isLiveTail: boolean;
  source: string[];
}

const defaultURL = 'structuredLogger.aws.amazon.com';
const usGovURL = 'structuredLogger.amazonaws-us-gov.com';
const chinaURL = 'structuredLogger.amazonaws.cn';

export function getLogsEndpoint(region: string): string {
  let url = defaultURL;
  if (region.startsWith('us-gov-')) {
    url = usGovURL;
  }
  if (region.startsWith('cn-')) {
    url = chinaURL;
  }
  return `${region}.${url}`;
}

export function encodeUrl(obj: AwsUrl, region: string): string {
  return `https://${getLogsEndpoint(
    region
  )}/cloudwatch/home?region=${region}#logs-insights:queryDetail=${JSURL.stringify(obj)}`;
}
