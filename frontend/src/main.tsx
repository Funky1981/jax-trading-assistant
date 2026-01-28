import React from 'react';
import { createRoot } from 'react-dom/client';
import { CssBaseline, ThemeProvider } from '@mui/material';
import App from './app/App';
import './styles/tokens.css';
import { theme } from './styles/theme';
import { DomainProvider } from './domain/store';

const root = document.getElementById('root');
if (!root) {
  throw new Error('Root element not found');
}

createRoot(root).render(
  <React.StrictMode>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <DomainProvider>
        <App />
      </DomainProvider>
    </ThemeProvider>
  </React.StrictMode>
);
