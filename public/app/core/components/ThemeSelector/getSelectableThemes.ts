import { getBuiltInThemes } from '@grafana/data';

export function getSelectableThemes() {
  const allowedExtraThemes = ['desertbloom', 'gildedgrove', 'sapphiredusk', 'tron', 'gloom'];

  return getBuiltInThemes(allowedExtraThemes);
}
