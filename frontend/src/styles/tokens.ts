export const tokens = {
  colors: {
    bg: '#0b0f14',
    surface: '#131a22',
    text: '#e8eef4',
    textMuted: '#9db0c3',
    border: '#243041',
    accent: '#3aa0ff',
    positive: '#1db954',
    negative: '#ff5c5c',
    warning: '#f59e0b',
    info: '#3aa0ff',
  },
  colorsLight: {
    bg: '#f4f6f8',
    surface: '#ffffff',
    text: '#0f1b2d',
    textMuted: '#52607a',
    border: '#d9e0ea',
    accent: '#1f6feb',
    positive: '#15803d',
    negative: '#dc2626',
    warning: '#d97706',
    info: '#0284c7',
  },
  typography: {
    fontFamily: '"IBM Plex Sans", "Segoe UI", sans-serif',
    scale: {
      xs: 12,
      sm: 13,
      md: 14,
      lg: 16,
      xl: 20,
      xxl: 28,
    },
    weight: {
      regular: 400,
      medium: 500,
      semibold: 600,
      bold: 700,
    },
    lineHeight: {
      tight: 1.1,
      normal: 1.4,
      relaxed: 1.6,
    },
  },
  spacing: {
    xs: 4,
    sm: 8,
    md: 12,
    lg: 16,
    xl: 24,
    xxl: 32,
  },
  layout: {
    gridRowHeight: 120,
    contentMaxWidth: 1280,
  },
  radius: {
    sm: 4,
    md: 8,
    lg: 12,
  },
  elevation: {
    sm: '0 1px 2px rgba(0, 0, 0, 0.2)',
    md: '0 4px 12px rgba(0, 0, 0, 0.35)',
  },
  chart: {
    up: '#1db954',
    down: '#ff5c5c',
    volume: '#3aa0ff',
    risk: '#f5a623',
  },
};

export type ThemeMode = 'dark' | 'light';
