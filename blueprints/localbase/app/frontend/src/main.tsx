import React from 'react';
import ReactDOM from 'react-dom/client';
import { MantineProvider, createTheme, MantineColorsTuple } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { BrowserRouter } from 'react-router-dom';
import App from './App';

// Mantine styles
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import '@mantine/dropzone/styles.css';
import '@mantine/dates/styles.css';

// Global styles and Localbase design system
import './styles/index.css';
import './styles/supabase-theme.css';

// Custom green color palette based on Supabase brand (#3ECF8E)
const brandGreen: MantineColorsTuple = [
  '#e6fff2',
  '#ccffe6',
  '#99ffcc',
  '#66ffb3',
  '#3ECF8E', // Primary brand color (index 4)
  '#2DB77A', // Hover state (index 5)
  '#1C9B5E', // Active state (index 6)
  '#156B41',
  '#0F4A2D',
  '#082919',
];

// Custom dark color palette for dark mode
const dark: MantineColorsTuple = [
  '#C1C2C5',
  '#A6A7AB',
  '#909296',
  '#5c5f66',
  '#373A40',
  '#2C2E33',
  '#25262b',
  '#1A1B1E',
  '#141517',
  '#101113',
];

// Localbase theme configuration - inspired by Supabase Studio
const theme = createTheme({
  // Brand color
  primaryColor: 'green',
  primaryShade: { light: 4, dark: 4 },

  // Color definitions
  colors: {
    green: brandGreen,
    dark: dark,
  },

  // Typography
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", sans-serif',
  fontFamilyMonospace: '"Source Code Pro", Menlo, Monaco, "Courier New", monospace',

  // Base colors
  black: '#1C1C1C',
  white: '#ffffff',

  // Border radius scale
  defaultRadius: 'md',
  radius: {
    xs: '3px',
    sm: '4px',
    md: '6px',
    lg: '8px',
    xl: '12px',
  },

  // Spacing scale (4px base unit)
  spacing: {
    xs: '4px',
    sm: '8px',
    md: '16px',
    lg: '24px',
    xl: '32px',
  },

  // Shadows
  shadows: {
    xs: '0 1px 2px rgba(0, 0, 0, 0.04)',
    sm: '0 1px 2px rgba(0, 0, 0, 0.05)',
    md: '0 1px 3px rgba(0, 0, 0, 0.06)',
    lg: '0 4px 6px rgba(0, 0, 0, 0.07)',
    xl: '0 10px 15px rgba(0, 0, 0, 0.1)',
  },

  // Headings
  headings: {
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", sans-serif',
    fontWeight: '600',
    sizes: {
      h1: { fontSize: '2rem', lineHeight: '1.25' },
      h2: { fontSize: '1.5rem', lineHeight: '1.3' },
      h3: { fontSize: '1.25rem', lineHeight: '1.35' },
      h4: { fontSize: '1rem', lineHeight: '1.4' },
      h5: { fontSize: '0.875rem', lineHeight: '1.45' },
      h6: { fontSize: '0.75rem', lineHeight: '1.5' },
    },
  },

  // Font sizes
  fontSizes: {
    xs: '0.6875rem',  // 11px
    sm: '0.75rem',    // 12px
    md: '0.875rem',   // 14px
    lg: '1rem',       // 16px
    xl: '1.25rem',    // 20px
  },

  // Line heights
  lineHeights: {
    xs: '1.25',
    sm: '1.35',
    md: '1.5',
    lg: '1.65',
    xl: '1.75',
  },

  // Component defaults
  components: {
    // Buttons
    Button: {
      defaultProps: {
        size: 'sm',
      },
      styles: {
        root: {
          fontWeight: 500,
        },
      },
    },

    // Text Inputs
    TextInput: {
      defaultProps: {
        size: 'sm',
      },
    },

    // Select
    Select: {
      defaultProps: {
        size: 'sm',
      },
    },

    // Password Input
    PasswordInput: {
      defaultProps: {
        size: 'sm',
      },
    },

    // Number Input
    NumberInput: {
      defaultProps: {
        size: 'sm',
      },
    },

    // Textarea
    Textarea: {
      defaultProps: {
        size: 'sm',
      },
    },

    // Modal
    Modal: {
      defaultProps: {
        centered: true,
        overlayProps: {
          backgroundOpacity: 0.55,
          blur: 3,
        },
      },
    },

    // Paper
    Paper: {
      defaultProps: {
        radius: 'md',
      },
    },

    // Card
    Card: {
      defaultProps: {
        radius: 'md',
        withBorder: true,
      },
    },

    // Badge
    Badge: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        root: {
          textTransform: 'none',
          fontWeight: 500,
        },
      },
    },

    // ActionIcon
    ActionIcon: {
      defaultProps: {
        variant: 'subtle',
      },
    },

    // Tabs
    Tabs: {
      styles: {
        tab: {
          fontWeight: 500,
        },
      },
    },

    // Menu
    Menu: {
      defaultProps: {
        shadow: 'md',
        radius: 'md',
      },
    },

    // Tooltip
    Tooltip: {
      defaultProps: {
        withArrow: false,
        transitionProps: {
          transition: 'fade',
          duration: 150,
        },
      },
    },

    // Notification
    Notification: {
      defaultProps: {
        radius: 'md',
      },
    },

    // Divider
    Divider: {
      styles: {
        root: {
          borderColor: 'var(--lb-border-default)',
        },
      },
    },

    // Table
    Table: {
      defaultProps: {
        verticalSpacing: 'sm',
        horizontalSpacing: 'md',
      },
    },

    // Loader
    Loader: {
      defaultProps: {
        type: 'dots',
      },
    },

    // Switch
    Switch: {
      defaultProps: {
        size: 'sm',
      },
    },

    // Checkbox
    Checkbox: {
      defaultProps: {
        size: 'sm',
        radius: 'sm',
      },
    },

    // NavLink
    NavLink: {
      styles: {
        root: {
          borderRadius: 'var(--lb-radius-md)',
        },
      },
    },

    // AppShell
    AppShell: {
      styles: {
        main: {
          backgroundColor: 'var(--lb-bg-secondary)',
        },
      },
    },

    // ScrollArea
    ScrollArea: {
      styles: {
        scrollbar: {
          '&[data-orientation="vertical"]': {
            width: '8px',
          },
          '&[data-orientation="horizontal"]': {
            height: '8px',
          },
        },
        thumb: {
          borderRadius: '4px',
        },
      },
    },

    // Skeleton
    Skeleton: {
      defaultProps: {
        radius: 'md',
      },
    },
  },

  // Other settings
  cursorType: 'pointer',
  focusRing: 'auto',
  respectReducedMotion: true,
  autoContrast: true,
  luminanceThreshold: 0.3,
});

// Get initial color scheme from localStorage or system preference
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
