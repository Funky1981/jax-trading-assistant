import { CssBaseline, ThemeProvider } from '@mui/material';
import type { Preview } from '@storybook/react';
import { theme } from '../src/styles/theme';
import '../src/index.css';

if (typeof document !== 'undefined') {
  document.documentElement.classList.add('dark');
}

const preview: Preview = {
  parameters: {
    actions: { argTypesRegex: '^on[A-Z].*' },
    controls: { expanded: true },
  },
  decorators: [
    (Story) => (
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <Story />
      </ThemeProvider>
    ),
  ],
};

export default preview;
