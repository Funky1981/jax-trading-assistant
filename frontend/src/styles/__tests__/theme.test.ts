import { describe, expect, it } from 'vitest';
import { createAppTheme } from '../../styles/theme';
import { tokens } from '../../styles/tokens';

describe('theme', () => {
  it('uses dark tokens by default', () => {
    const theme = createAppTheme('dark');
    expect(theme.palette.background.default).toBe(tokens.colors.bg);
    expect(theme.palette.primary.main).toBe(tokens.colors.accent);
  });

  it('uses light tokens when requested', () => {
    const theme = createAppTheme('light');
    expect(theme.palette.background.default).toBe(tokens.colorsLight.bg);
    expect(theme.palette.text.secondary).toBe(tokens.colorsLight.textMuted);
  });
});
