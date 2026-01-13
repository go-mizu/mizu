import React from 'react'
import ReactDOM from 'react-dom/client'
import { MantineProvider, createTheme, virtualColor } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '@mantine/code-highlight/styles.css'
import '@mantine/dates/styles.css'

// Cloudflare Dashboard theme - matching official color palette
// Source: https://www.designpieces.com/palette/cloudflare-color-palette-hex-and-rgb/
// Primary: #f38020 (Cloudflare Tango), #faae40 (Yellow Orange), #404041 (Ship Gray)
const theme = createTheme({
  primaryColor: 'orange',
  colors: {
    // Cloudflare brand orange palette (#f38020 as primary)
    orange: [
      '#fff8f1',  // 0: Lightest
      '#ffedd5',  // 1
      '#fed7aa',  // 2
      '#fdba74',  // 3
      '#fb923c',  // 4
      '#f38020',  // 5: Cloudflare Tango (primary)
      '#ea580c',  // 6
      '#c2410c',  // 7
      '#9a3412',  // 8
      '#7c2d12',  // 9: Darkest
    ],
    // Cloudflare light orange / yellow (#faae40)
    yellow: [
      '#fffbeb',
      '#fef3c7',
      '#fde68a',
      '#fcd34d',
      '#fbbf24',
      '#faae40',  // 5: Cloudflare Yellow Orange
      '#d97706',
      '#b45309',
      '#92400e',
      '#78350f',
    ],
    // Gray palette based on Cloudflare Ship Gray (#404041)
    gray: [
      '#fafafa',  // 0: Light mode backgrounds
      '#f5f5f5',  // 1: Light mode secondary bg
      '#e5e5e5',  // 2: Borders light
      '#d4d4d4',  // 3
      '#a3a3a3',  // 4
      '#737373',  // 5: Mid gray
      '#525252',  // 6
      '#404041',  // 7: Cloudflare Ship Gray
      '#262626',  // 8
      '#1d1d1d',  // 9: Dark mode background (from Cloudflare blog)
    ],
    // Blue for links and info states
    blue: [
      '#eff6ff',
      '#dbeafe',
      '#bfdbfe',
      '#93c5fd',
      '#60a5fa',
      '#3b82f6',  // 5: Primary blue
      '#2563eb',
      '#1d4ed8',
      '#1e40af',
      '#1e3a8a',
    ],
    // Green for success states
    green: [
      '#f0fdf4',
      '#dcfce7',
      '#bbf7d0',
      '#86efac',
      '#4ade80',
      '#22c55e',  // 5: Success green
      '#16a34a',
      '#15803d',
      '#166534',
      '#14532d',
    ],
    // Red for error states
    red: [
      '#fef2f2',
      '#fee2e2',
      '#fecaca',
      '#fca5a5',
      '#f87171',
      '#ef4444',  // 5: Error red
      '#dc2626',
      '#b91c1c',
      '#991b1b',
      '#7f1d1d',
    ],
    // Virtual color for primary that adapts to color scheme
    primary: virtualColor({
      name: 'primary',
      dark: 'orange',
      light: 'orange',
    }),
  },
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
  headings: {
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    fontWeight: '600',
  },
  defaultRadius: 'md',
  white: '#ffffff',
  black: '#1d1d1d',
  // Light theme specific overrides
  other: {
    // Semantic colors for consistent usage
    headerBg: 'var(--mantine-color-body)',
    sidebarBg: 'var(--mantine-color-body)',
    cardBg: 'var(--mantine-color-body)',
  },
  components: {
    Button: {
      defaultProps: {
        fw: 600,
      },
    },
    NavLink: {
      styles: {
        root: {
          borderRadius: 'var(--mantine-radius-md)',
        },
      },
    },
    Paper: {
      defaultProps: {
        shadow: 'xs',
      },
    },
    Card: {
      defaultProps: {
        shadow: 'xs',
      },
    },
  },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="light">
      <Notifications position="top-right" />
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </MantineProvider>
  </React.StrictMode>,
)
