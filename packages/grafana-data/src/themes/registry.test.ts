import { getBuiltInThemes, getThemeById } from './registry';

describe('theme registry', () => {
  it('registers 90s mode as an extra built-in theme', () => {
    const theme = getThemeById('90smode');
    const themeOption = getBuiltInThemes(['90smode']).find((item) => item.id === '90smode');

    expect(theme.name).toBe('90s mode');
    expect(theme.isLight).toBe(true);
    expect(theme.colors.background.canvas).toBe('#008080');
    expect(theme.shape.radius.default).toBe('0px');
    expect(theme.typography.fontFamily).toBe('"MS Sans Serif", "Arial", sans-serif');
    expect(themeOption).toMatchObject({
      id: '90smode',
      name: '90s mode',
      isExtra: true,
    });
  });
});
