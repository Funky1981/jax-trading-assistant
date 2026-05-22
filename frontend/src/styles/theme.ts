import { createTheme } from '@mui/material/styles';
import { tokens, type ThemeMode } from './tokens';

function getPalette(mode: ThemeMode = 'dark') {
  return {
    mode,
    background: {
      default: tokens.colors.bg,
      paper: tokens.colors.surface,
    },
    text: {
      primary: tokens.colors.text,
      secondary: tokens.colors.textMuted,
    },
    primary: {
      main: tokens.colors.accent,
    },
    success: {
      main: tokens.colors.positive,
    },
    error: {
      main: tokens.colors.negative,
    },
    warning: {
      main: tokens.colors.warning,
    },
    info: {
      main: tokens.colors.info,
    },
    divider: tokens.colors.border,
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
      h6: {
        fontSize: tokens.typography.scale.lg,
        fontWeight: tokens.typography.weight.semibold,
        lineHeight: tokens.typography.lineHeight.tight,
      },
      subtitle1: {
        fontSize: tokens.typography.scale.md,
        fontWeight: tokens.typography.weight.medium,
        lineHeight: tokens.typography.lineHeight.normal,
      },
      subtitle2: {
        fontSize: tokens.typography.scale.sm,
        fontWeight: tokens.typography.weight.medium,
        lineHeight: tokens.typography.lineHeight.normal,
      },
      body1: {
        fontSize: tokens.typography.scale.md,
        lineHeight: tokens.typography.lineHeight.relaxed,
      },
      body2: {
        fontSize: tokens.typography.scale.sm,
        lineHeight: tokens.typography.lineHeight.relaxed,
      },
      overline: {
        fontSize: tokens.typography.scale.xs,
        letterSpacing: 2,
        fontWeight: tokens.typography.weight.semibold,
      },
      caption: {
        fontSize: tokens.typography.scale.xs,
        lineHeight: tokens.typography.lineHeight.normal,
      },
    },
    shape: {
      borderRadius: tokens.radius.md,
    },
    spacing: 8, // Base spacing unit
    components: {
      MuiCard: {
        styleOverrides: {
          root: {
            backgroundImage: 'none',
          },
        },
      },
      MuiPaper: {
        styleOverrides: {
          root: {
            backgroundImage: 'none',
          },
        },
      },
      MuiChip: {
        styleOverrides: {
          root: {
            borderRadius: tokens.radius.sm,
            fontWeight: tokens.typography.weight.medium,
          },
        },
      },
      MuiLinearProgress: {
        styleOverrides: {
          root: {
            borderRadius: tokens.radius.sm,
          },
        },
      },
      MuiTableCell: {
        styleOverrides: {
          root: {
            borderBottom: '1px solid',
            borderBottomColor: tokens.colors.border,
          },
          head: {
            fontWeight: tokens.typography.weight.semibold,
            backgroundColor: tokens.colors.surface,
          },
        },
      },
      MuiAlert: {
        styleOverrides: {
          root: {
            borderRadius: tokens.radius.md,
          },
        },
      },
    },
  });
}

export const theme = createAppTheme('dark');
