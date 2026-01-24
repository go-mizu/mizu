import { createTheme, MantineColorsTuple, CSSVariablesResolver, rem } from '@mantine/core'

// =============================================================================
// MODERN COLOR PALETTE (shadcn-inspired with functional colors)
// =============================================================================

// Brand Blue - Primary color for actions (shadcn blue)
const brandBlue: MantineColorsTuple = [
  '#eff6ff', // 0 - lightest
  '#dbeafe', // 1
  '#bfdbfe', // 2
  '#93c5fd', // 3
  '#2563eb', // 4 - PRIMARY (shadcn primary blue)
  '#1d4ed8', // 5
  '#1e40af', // 6
  '#1e3a8a', // 7
  '#172554', // 8
  '#0f172a', // 9 - darkest
]

// Success Green - For aggregations, metrics, success (shadcn green)
const summarizeGreen: MantineColorsTuple = [
  '#f0fdf4', // 0
  '#dcfce7', // 1
  '#bbf7d0', // 2
  '#86efac', // 3
  '#22c55e', // 4 - PRIMARY (shadcn success green)
  '#16a34a', // 5
  '#15803d', // 6
  '#166534', // 7
  '#14532d', // 8
  '#052e16', // 9
]

// Violet - For filters, constraints (shadcn violet)
const filterPurple: MantineColorsTuple = [
  '#f5f3ff', // 0
  '#ede9fe', // 1
  '#ddd6fe', // 2
  '#c4b5fd', // 3
  '#8b5cf6', // 4 - PRIMARY (shadcn violet)
  '#7c3aed', // 5
  '#6d28d9', // 6
  '#5b21b6', // 7
  '#4c1d95', // 8
  '#2e1065', // 9
]

// Warning Amber (shadcn amber)
const warningYellow: MantineColorsTuple = [
  '#fffbeb', // 0
  '#fef3c7', // 1
  '#fde68a', // 2
  '#fcd34d', // 3
  '#f59e0b', // 4 - PRIMARY (shadcn amber)
  '#d97706', // 5
  '#b45309', // 6
  '#92400e', // 7
  '#78350f', // 8
  '#451a03', // 9
]

// Error Red - For errors, destructive actions (shadcn red)
const errorRed: MantineColorsTuple = [
  '#fef2f2', // 0
  '#fee2e2', // 1
  '#fecaca', // 2
  '#fca5a5', // 3
  '#ef4444', // 4 - PRIMARY (shadcn red)
  '#dc2626', // 5
  '#b91c1c', // 6
  '#991b1b', // 7
  '#7f1d1d', // 8
  '#450a0a', // 9
]

// Neutral gray scale (shadcn zinc)
const textGray: MantineColorsTuple = [
  '#ffffff', // 0 - white
  '#fafafa', // 1 - bg light
  '#f4f4f5', // 2 - bg medium
  '#e4e4e7', // 3 - border
  '#a1a1aa', // 4 - text tertiary
  '#71717a', // 5 - text secondary
  '#52525b', // 6 - text primary
  '#3f3f46', // 7
  '#27272a', // 8 - sidebar bg
  '#18181b', // 9 - darkest
]

// Chart colors - Metabase exact 8-color palette
export const chartColors = [
  '#509EE3', // 1. Brand Blue
  '#84BB4C', // 2. Green (adjusted)
  '#7172AD', // 3. Purple (filter purple)
  '#F2A86F', // 4. Orange
  '#98D9D9', // 5. Teal
  '#ED6E6E', // 6. Coral/Red
  '#50C9BA', // 7. Mint
  '#F9D45C', // 8. Yellow
  // Extended palette
  '#7C79B8', // Darker purple
  '#B5D6A3', // Light green
  '#FFB366', // Light orange
  '#6CD4D4', // Bright teal
  '#F28B8B', // Light coral
  '#6EDBC8', // Light mint
  '#FFE380', // Light yellow
  '#76B4E8', // Light blue
  '#5C9449', // Dark green
  '#555A8C', // Dark purple
  '#CC7A42', // Dark orange
  '#5AADAD', // Dark teal
  '#C55555', // Dark coral
  '#3DAD9A', // Dark mint
  '#CCA847', // Dark yellow
  '#3B7DBF', // Dark blue
]

// =============================================================================
// THEME CONFIGURATION
// =============================================================================

export const theme = createTheme({
  primaryColor: 'brand',
  colors: {
    brand: brandBlue,
    summarize: summarizeGreen,
    filter: filterPurple,
    warning: warningYellow,
    error: errorRed,
    text: textGray,
  },

  // Typography - Metabase uses Lato
  fontFamily: 'Lato, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  fontFamilyMonospace: 'Monaco, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',

  headings: {
    fontFamily: 'Lato, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    fontWeight: '700',
    sizes: {
      h1: { fontSize: rem(32), lineHeight: '1.3' },
      h2: { fontSize: rem(24), lineHeight: '1.35' },
      h3: { fontSize: rem(20), lineHeight: '1.4' },
      h4: { fontSize: rem(16), lineHeight: '1.45' },
      h5: { fontSize: rem(14), lineHeight: '1.5' },
      h6: { fontSize: rem(12), lineHeight: '1.5' },
    },
  },

  fontSizes: {
    xs: rem(11),
    sm: rem(12),
    md: rem(14),
    lg: rem(16),
    xl: rem(20),
  },

  lineHeights: {
    xs: '1.4',
    sm: '1.45',
    md: '1.5',
    lg: '1.55',
    xl: '1.6',
  },

  // Spacing - 8px grid
  spacing: {
    xs: rem(4),
    sm: rem(8),
    md: rem(16),
    lg: rem(24),
    xl: rem(32),
  },

  // Border radius - Metabase style
  defaultRadius: 'sm',
  radius: {
    xs: rem(2),
    sm: rem(4),
    md: rem(8),
    lg: rem(12),
    xl: rem(16),
  },

  // Shadows - Metabase style (softer)
  shadows: {
    xs: '0 1px 2px rgba(0, 0, 0, 0.06)',
    sm: '0 1px 3px rgba(0, 0, 0, 0.08)',
    md: '0 4px 12px rgba(0, 0, 0, 0.08)',
    lg: '0 8px 24px rgba(0, 0, 0, 0.10)',
    xl: '0 16px 48px rgba(0, 0, 0, 0.12)',
  },

  // Other theme values
  other: {
    // Sidebar theme (dark)
    sidebarBg: '#2E353B',
    sidebarText: 'rgba(255, 255, 255, 0.7)',
    sidebarTextHover: '#ffffff',
    sidebarActive: '#509EE3',
    sidebarBorder: 'rgba(255, 255, 255, 0.1)',
    sidebarInputBg: 'rgba(255, 255, 255, 0.1)',

    // Background colors
    bgWhite: '#FFFFFF',
    bgLight: '#F9FBFC',
    bgMedium: '#F0F0F0',
    bgDark: '#2E353B',

    // Text colors
    textPrimary: '#4C5773',
    textSecondary: '#696E7B',
    textTertiary: '#949AAB',
    textWhite: '#FFFFFF',

    // Border colors
    borderLight: '#F0F0F0',
    borderMedium: '#EEECEC',
    borderDark: '#E0E0E0',

    // Functional colors
    brand: '#509EE3',
    summarize: '#84BB4C',
    filter: '#7172AD',
    success: '#84BB4C',
    error: '#ED6E6E',
    warning: '#F9D45C',
  },

  // Component defaults (shadcn-inspired)
  components: {
    Button: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        root: {
          fontWeight: 500,
          transition: 'all 120ms cubic-bezier(0.4, 0, 0.2, 1)',
          '&:active': {
            transform: 'scale(0.98)',
          },
        },
      },
    },

    Card: {
      defaultProps: {
        radius: 'lg',
        shadow: 'none',
      },
      styles: {
        root: {
          border: '1px solid var(--color-border)',
          transition: 'all 150ms cubic-bezier(0.4, 0, 0.2, 1)',
          '&:hover': {
            boxShadow: 'var(--shadow-md)',
          },
        },
      },
    },

    Paper: {
      defaultProps: {
        radius: 'lg',
      },
      styles: {
        root: {
          border: '1px solid var(--color-border)',
        },
      },
    },

    Modal: {
      defaultProps: {
        radius: 'lg',
        centered: true,
        overlayProps: {
          backgroundOpacity: 0.5,
          blur: 4,
        },
      },
      styles: {
        title: {
          fontWeight: 600,
          fontSize: rem(18),
          color: 'var(--color-foreground)',
          letterSpacing: '-0.02em',
        },
        header: {
          paddingBottom: rem(16),
        },
        content: {
          boxShadow: 'var(--shadow-xl)',
        },
      },
    },

    TextInput: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        input: {
          borderColor: 'var(--color-border)',
          backgroundColor: 'var(--color-background)',
          transition: 'all 120ms ease',
          '&:focus': {
            borderColor: 'var(--color-primary)',
            boxShadow: '0 0 0 2px rgba(37, 99, 235, 0.15)',
          },
        },
        label: {
          fontWeight: 500,
          fontSize: rem(14),
          color: 'var(--color-foreground)',
          marginBottom: rem(6),
        },
      },
    },

    Select: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        input: {
          borderColor: 'var(--color-border)',
          '&:focus': {
            borderColor: 'var(--color-primary)',
            boxShadow: '0 0 0 2px rgba(37, 99, 235, 0.15)',
          },
        },
      },
    },

    Textarea: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        input: {
          borderColor: 'var(--color-border)',
          '&:focus': {
            borderColor: 'var(--color-primary)',
            boxShadow: '0 0 0 2px rgba(37, 99, 235, 0.15)',
          },
        },
      },
    },

    SegmentedControl: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        root: {
          backgroundColor: 'var(--color-background-subtle)',
          border: '1px solid var(--color-border)',
        },
      },
    },

    Tabs: {
      styles: {
        tab: {
          fontWeight: 500,
          color: 'var(--color-foreground-muted)',
          transition: 'all 120ms ease',
          '&[data-active]': {
            color: 'var(--color-foreground)',
          },
          '&:hover': {
            color: 'var(--color-foreground)',
            backgroundColor: 'var(--color-background-subtle)',
          },
        },
        tabLabel: {
          fontWeight: 500,
        },
      },
    },

    Badge: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        root: {
          fontWeight: 500,
          textTransform: 'none',
          fontSize: rem(12),
        },
      },
    },

    ActionIcon: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        root: {
          transition: 'all 120ms ease',
        },
      },
    },

    Tooltip: {
      defaultProps: {
        withArrow: true,
        arrowSize: 6,
        radius: 'md',
      },
      styles: {
        tooltip: {
          fontSize: rem(12),
          fontWeight: 500,
          backgroundColor: 'var(--color-foreground)',
          color: 'var(--color-background)',
          boxShadow: 'var(--shadow-lg)',
        },
      },
    },

    Menu: {
      defaultProps: {
        radius: 'md',
      },
      styles: {
        dropdown: {
          boxShadow: 'var(--shadow-lg)',
          border: '1px solid var(--color-border)',
          borderRadius: rem(12),
          padding: rem(4),
        },
        item: {
          fontSize: rem(14),
          padding: `${rem(8)} ${rem(12)}`,
          borderRadius: rem(8),
          '&[data-hovered]': {
            backgroundColor: 'var(--color-background-subtle)',
          },
        },
      },
    },

    NavLink: {
      styles: {
        root: {
          borderRadius: rem(8),
          fontWeight: 500,
          transition: 'all 120ms ease',
        },
      },
    },

    Table: {
      styles: {
        thead: {
          backgroundColor: 'var(--color-background-subtle)',
        },
        th: {
          fontWeight: 500,
          fontSize: rem(12),
          color: 'var(--color-foreground-muted)',
          textTransform: 'uppercase',
          letterSpacing: '0.04em',
          borderBottom: '1px solid var(--color-border)',
        },
        td: {
          fontSize: rem(14),
          color: 'var(--color-foreground)',
          borderBottom: '1px solid var(--color-border-light)',
        },
        tr: {
          transition: 'background-color 120ms ease',
          '&:hover': {
            backgroundColor: 'var(--color-background-subtle)',
          },
        },
      },
    },

    Notification: {
      defaultProps: {
        radius: 'lg',
      },
      styles: {
        root: {
          boxShadow: 'var(--shadow-lg)',
          border: '1px solid var(--color-border)',
        },
      },
    },

    Divider: {
      styles: {
        root: {
          borderColor: 'var(--color-border)',
        },
      },
    },

    Switch: {
      styles: {
        track: {
          borderColor: 'var(--color-border)',
        },
      },
    },

    ScrollArea: {
      styles: {
        scrollbar: {
          '&[data-orientation="vertical"]': {
            width: rem(8),
          },
          '&[data-orientation="horizontal"]': {
            height: rem(8),
          },
        },
        thumb: {
          backgroundColor: 'rgba(0, 0, 0, 0.12)',
          borderRadius: rem(4),
          '&:hover': {
            backgroundColor: 'rgba(0, 0, 0, 0.2)',
          },
        },
      },
    },

    Skeleton: {
      styles: {
        root: {
          '&::after': {
            background: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.5), transparent)',
          },
        },
      },
    },
  },
})

// =============================================================================
// CSS VARIABLES RESOLVER
// =============================================================================

export const cssVariablesResolver: CSSVariablesResolver = (_theme) => ({
  variables: {
    // Brand colors
    '--mb-color-brand': '#509EE3',
    '--mb-color-brand-light': '#E6F2FF',
    '--mb-color-brand-lighter': '#F0F6FC',

    // Semantic colors
    '--mb-color-summarize': '#84BB4C',
    '--mb-color-summarize-light': '#EDF7E4',
    '--mb-color-filter': '#7172AD',
    '--mb-color-filter-light': '#EFEEF5',

    // Status colors
    '--mb-color-success': '#84BB4C',
    '--mb-color-error': '#ED6E6E',
    '--mb-color-warning': '#F9D45C',

    // Text colors
    '--mb-color-text-primary': '#4C5773',
    '--mb-color-text-secondary': '#696E7B',
    '--mb-color-text-tertiary': '#949AAB',
    '--mb-color-text-white': '#FFFFFF',

    // Background colors
    '--mb-color-bg-white': '#FFFFFF',
    '--mb-color-bg-light': '#F9FBFC',
    '--mb-color-bg-medium': '#F0F0F0',
    '--mb-color-bg-dark': '#2E353B',

    // Border colors
    '--mb-color-border': '#EEECEC',
    '--mb-color-border-light': '#F0F0F0',

    // Shadows
    '--mb-shadow-sm': '0 1px 2px rgba(0, 0, 0, 0.06)',
    '--mb-shadow-md': '0 4px 12px rgba(0, 0, 0, 0.08)',
    '--mb-shadow-lg': '0 8px 24px rgba(0, 0, 0, 0.10)',
    '--mb-shadow-hover': '0 4px 16px rgba(0, 0, 0, 0.10)',

    // Sidebar
    '--mb-sidebar-bg': '#2E353B',
    '--mb-sidebar-text': 'rgba(255, 255, 255, 0.7)',
    '--mb-sidebar-text-hover': '#FFFFFF',
    '--mb-sidebar-active': '#509EE3',
    '--mb-sidebar-border': 'rgba(255, 255, 255, 0.1)',

    // Transition
    '--mb-transition-fast': '0.15s ease',
    '--mb-transition-normal': '0.2s ease',
  },
  light: {},
  dark: {},
})

// =============================================================================
// SIDEBAR THEME TOKENS (Light Theme - Metabase Style)
// =============================================================================

export const sidebarTheme = {
  // Light theme (Metabase default)
  bg: '#FFFFFF',
  bgHover: '#F9FBFC',
  bgActive: '#EEF6FC',
  text: '#4C5773',
  textSecondary: '#696E7B',
  textHover: '#4C5773',
  textActive: '#509EE3',
  active: '#509EE3',
  border: '#EEECEC',
  inputBg: '#F9FBFC',
  inputBorder: '#EEECEC',
  inputText: '#4C5773',
  inputPlaceholder: '#949AAB',
  // Icon colors
  iconDefault: '#696E7B',
  iconHover: '#4C5773',
  iconActive: '#509EE3',
  // Section headers
  sectionTitle: '#949AAB',
  // Accent colors for specific icons
  newQuestion: '#509EE3',
  newDashboard: '#84BB4C',
  newCollection: '#F9D45C',
}

// =============================================================================
// SEMANTIC COLORS EXPORT
// =============================================================================

export const semanticColors = {
  // Brand
  brand: '#509EE3',
  brandLight: '#E6F2FF',
  brandLighter: '#F0F6FC',
  brandHover: '#4285C9',
  brandDark: '#326597',

  // Functional - Metabase exact
  summarize: '#84BB4C',
  summarizeLight: '#EDF7E4',
  filter: '#7172AD',
  filterLight: '#EFEEF5',

  // Status
  success: '#84BB4C',
  successLight: '#EDF7E4',
  warning: '#F9D45C',
  warningLight: '#FFFCEB',
  error: '#ED6E6E',
  errorLight: '#FEF0F0',
  info: '#509EE3',
  infoLight: '#E6F2FF',

  // Text
  textPrimary: '#4C5773',
  textSecondary: '#696E7B',
  textTertiary: '#949AAB',
  textWhite: '#FFFFFF',

  // Background
  bgWhite: '#FFFFFF',
  bgLight: '#F9FBFC',
  bgMedium: '#F0F0F0',
  bgDark: '#2E353B',

  // Borders
  borderLight: '#F0F0F0',
  borderMedium: '#EEECEC',
  borderDark: '#E0E0E0',
}

// =============================================================================
// LOADING MESSAGES
// =============================================================================

export const loadingMessages = [
  'Doing science...',
  'Running query...',
  'Loading results...',
  'Crunching numbers...',
  'Fetching data...',
]

// =============================================================================
// VISUALIZATION DEFAULTS
// =============================================================================

export const vizDefaults = {
  showLegend: true,
  showDataLabels: false,
  showGoalLine: false,
  stacked: false,
  normalized: false,
  showTrendLine: false,
}
