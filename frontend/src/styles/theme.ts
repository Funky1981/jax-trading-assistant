import { createTheme } from '@mui/material/styles';
import { tokens, type ThemeMode } from './tokens';

function getPalette(mode: ThemeMode) {
  const paletteTokens = mode === 'light' ? tokens.colorsLight : tokens.colors;

  return {
    mode,
    background: {
      default: paletteTokens.bg,
      paper: paletteTokens.surface,
    },
    text: {
      primary: paletteTokens.text,
      secondary: paletteTokens.textMuted,
    },
    primary: {
      main: paletteTokens.accent,
    },
    success: {
      main: paletteTokens.positive,
    },
    error: {
      main: paletteTokens.negative,
    },
    warning: {
      main: paletteTokens.warning,
    },
    divider: paletteTokens.border,
  };
}

export function createAppTheme(mode: ThemeMode = 'dark') {
  return createTheme({
    palette: getPalette(mode),
    typography: {
      fontFamily: tokens.typography.fontFamily,
      h4: {
        fontSize: tokens.typography.scale.xxl,
        fontWeight: tokens.typography.weight.semibold,
        lineHeight: tokens.typography.lineHeight.tight,
      },
      body2: {
        fontSize: tokens.typography.scale.sm,
        lineHeight: tokens.typography.lineHeight.relaxed,
      },
      overline: {
        fontSize: tokens.typography.scale.xs,
        letterSpacing: 2,
      },
    },
    shape: {
      borderRadius: tokens.radius.md,
    },
    components: {
      MuiChip: {
        styleOverrides: {
          root: {
            borderRadius: tokens.radius.sm,
          },
        },
      },
    },
  });
}

export const theme = createAppTheme('dark');
