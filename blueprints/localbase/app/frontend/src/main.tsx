import React from 'react';
import ReactDOM from 'react-dom/client';
import { MantineProvider, createTheme } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { BrowserRouter } from 'react-router-dom';
import App from './App';

// Mantine styles
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import '@mantine/dropzone/styles.css';
import '@mantine/dates/styles.css';

// Global styles and Supabase theme overrides
import './styles/index.css';
import './styles/supabase-theme.css';

// Supabase-inspired theme configuration
const theme = createTheme({
  primaryColor: 'green',
  colors: {
    green: [
      '#e6fff2',
      '#ccffe6',
      '#99ffcc',
      '#66ffb3',
      '#3ECF8E',
      '#33a874',
      '#29855c',
      '#1f6346',
      '#144230',
      '#0a211a',
    ],
  },
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", sans-serif',
  fontFamilyMonospace: '"Source Code Pro", Menlo, Monaco, monospace',
  defaultRadius: 'md',
  black: '#1C1C1C',
  white: '#ffffff',
  components: {
    Button: {
      defaultProps: {
        size: 'sm',
      },
    },
    TextInput: {
      defaultProps: {
        size: 'sm',
      },
    },
    Select: {
      defaultProps: {
        size: 'sm',
      },
    },
    PasswordInput: {
      defaultProps: {
        size: 'sm',
      },
    },
  },
});

// Get initial color scheme from localStorage or default to light
const getInitialColorScheme = (): 'light' | 'dark' => {
  const stored = localStorage.getItem('mantine-color-scheme');
  if (stored === 'dark' || stored === 'light') {
    return stored;
  }
  // Check system preference
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }
  return 'light';
};

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider theme={theme} defaultColorScheme={getInitialColorScheme()}>
      <Notifications position="top-right" />
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </MantineProvider>
  </React.StrictMode>
);
