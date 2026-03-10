import { CellProps } from 'react-table';

import { dateTimeFormat } from '@grafana/data';
import { Text } from '@grafana/ui';

import { DashboardsTreeItem } from '../types';

export function LastViewedCell({ row: { original: data } }: CellProps<DashboardsTreeItem, unknown>) {
  const item = data.item;

  if (item.kind === 'ui' || item.kind === 'folder' || !item.lastViewed) {
    return null;
  }

  return (
    <Text variant="bodySmall" color="secondary">
      {dateTimeFormat(item.lastViewed)}
    </Text>
  );
}
