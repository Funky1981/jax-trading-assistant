import { describe, expect, it } from 'vitest';
import { createAppTheme } from '../../styles/theme';
import { tokens } from '../../styles/tokens';

describe('theme', () => {
  it('uses semantic CSS variables by default', () => {
    const theme = createAppTheme('dark');
    expect(theme.palette.background.default).toBe(tokens.colors.bg);
    expect(theme.palette.background.default).toBe('hsl(var(--background))');
    expect(theme.palette.primary.main).toBe(tokens.colors.accent);
    expect(theme.palette.primary.main).toBe('hsl(var(--accent))');
  });

  it('keeps light mode on the same semantic token contract', () => {
    const theme = createAppTheme('light');
    expect(theme.palette.background.default).toBe(tokens.colors.bg);
    expect(theme.palette.text.secondary).toBe(tokens.colors.textMuted);
    expect(theme.palette.text.secondary).toBe('hsl(var(--muted-foreground))');
  });
});
