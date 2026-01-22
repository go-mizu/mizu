import { createTheme, MantineColorsTuple, CSSVariablesResolver, rem } from '@mantine/core'

// =============================================================================
// METABASE EXACT COLOR PALETTE
// =============================================================================

// Brand Blue - Primary color for actions, links, brand
const brandBlue: MantineColorsTuple = [
  '#F0F6FC', // 0 - lightest
  '#E6F2FF', // 1
  '#CCE5FF', // 2
  '#99CCFF', // 3
  '#509EE3', // 4 - PRIMARY (Metabase brand)
  '#4285C9', // 5
  '#3A75B0', // 6
  '#326597', // 7
  '#2A557E', // 8
  '#224565', // 9 - darkest
]

// Summarize Green - For aggregations, metrics, success
const summarizeGreen: MantineColorsTuple = [
  '#EDF7E4', // 0
  '#DBEFC9', // 1
  '#C2E5A0', // 2
  '#A8DB77', // 3
  '#84BB4C', // 4 - PRIMARY (Metabase summarize - adjusted)
  '#6FA83D', // 5
  '#5A912E', // 6
  '#467A1F', // 7
  '#336310', // 8
  '#1F4C00', // 9
]

// Filter Purple - For filters, constraints (FIXED to Metabase exact)
const filterPurple: MantineColorsTuple = [
  '#EFEEF5', // 0
  '#E0DEF0', // 1
  '#C5C2E0', // 2
  '#9995C5', // 3
  '#7172AD', // 4 - PRIMARY (Metabase filter - CORRECTED)
  '#5F6099', // 5
  '#4D4E85', // 6
  '#3B3C71', // 7
  '#292A5D', // 8
  '#171849', // 9
]

// Warning Yellow
const warningYellow: MantineColorsTuple = [
  '#FFFCEB', // 0
  '#FFF9D6', // 1
  '#FFF3AD', // 2
  '#FFED85', // 3
  '#F9D45C', // 4 - PRIMARY (Metabase warning - adjusted)
  '#E0B73D', // 5
  '#C79F32', // 6
  '#AE8727', // 7
  '#956F1C', // 8
  '#7C5711', // 9
]

// Error Red - For errors, destructive actions (FIXED)
const errorRed: MantineColorsTuple = [
  '#FEF0F0', // 0
  '#FCDCDC', // 1
  '#F9B8B8', // 2
  '#F59494', // 3
  '#ED6E6E', // 4 - PRIMARY (Metabase error - CORRECTED)
  '#D65555', // 5
  '#BD3C3C', // 6
  '#A42323', // 7
  '#8B0A0A', // 8
  '#720000', // 9
]

// Text colors as a shade scale
const textGray: MantineColorsTuple = [
  '#FFFFFF', // 0 - white
  '#F9FBFC', // 1 - bg light
  '#F0F0F0', // 2 - bg medium
  '#EEECEC', // 3 - border
  '#949AAB', // 4 - text tertiary
  '#696E7B', // 5 - text secondary
  '#4C5773', // 6 - text primary (Metabase)
  '#3B4357', // 7
  '#2E353B', // 8 - sidebar bg
  '#1C1F24', // 9 - darkest
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

  // Component defaults
  components: {
    Button: {
      defaultProps: {
        radius: 'sm',
      },
      styles: {
        root: {
          fontWeight: 700,
          transition: 'all 0.15s ease',
        },
      },
    },

    Card: {
      defaultProps: {
        radius: 'sm',
        shadow: 'sm',
      },
      styles: {
        root: {
          border: '1px solid #EEECEC',
          transition: 'box-shadow 0.15s ease, transform 0.15s ease',
          '&:hover': {
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)',
          },
        },
      },
    },

    Paper: {
      defaultProps: {
        radius: 'sm',
      },
      styles: {
        root: {
          border: '1px solid #EEECEC',
        },
      },
    },

    Modal: {
      defaultProps: {
        radius: 'md',
        centered: true,
        overlayProps: {
          backgroundOpacity: 0.4,
          blur: 2,
        },
      },
      styles: {
        title: {
          fontWeight: 700,
          fontSize: rem(20),
          color: '#4C5773',
        },
        header: {
          borderBottom: '1px solid #EEECEC',
          paddingBottom: rem(16),
          marginBottom: rem(16),
        },
      },
    },

    TextInput: {
      defaultProps: {
        radius: 'sm',
      },
      styles: {
        input: {
          borderColor: '#EEECEC',
          '&:focus': {
            borderColor: '#509EE3',
            boxShadow: '0 0 0 2px rgba(80, 158, 227, 0.2)',
          },
        },
        label: {
          fontWeight: 600,
          fontSize: rem(14),
          color: '#4C5773',
          marginBottom: rem(4),
        },
      },
    },

    Select: {
      defaultProps: {
        radius: 'sm',
      },
      styles: {
        input: {
          borderColor: '#EEECEC',
          '&:focus': {
            borderColor: '#509EE3',
          },
        },
      },
    },

    Textarea: {
      defaultProps: {
        radius: 'sm',
      },
      styles: {
        input: {
          borderColor: '#EEECEC',
          '&:focus': {
            borderColor: '#509EE3',
          },
        },
      },
    },

    Tabs: {
      styles: {
        tab: {
          fontWeight: 600,
          color: '#696E7B',
          transition: 'color 0.15s ease',
          '&[data-active]': {
            color: '#509EE3',
          },
          '&:hover': {
            color: '#4C5773',
            backgroundColor: 'transparent',
          },
        },
        tabLabel: {
          fontWeight: 600,
        },
      },
    },

    Badge: {
      defaultProps: {
        radius: 'xl',
      },
      styles: {
        root: {
          fontWeight: 700,
          textTransform: 'none',
          fontSize: rem(11),
        },
      },
    },

    ActionIcon: {
      styles: {
        root: {
          transition: 'all 0.15s ease',
        },
      },
    },

    Tooltip: {
      defaultProps: {
        withArrow: true,
        arrowSize: 6,
      },
      styles: {
        tooltip: {
          fontSize: rem(12),
          fontWeight: 500,
          backgroundColor: '#2E353B',
          color: '#FFFFFF',
        },
      },
    },

    Menu: {
      styles: {
        dropdown: {
          boxShadow: '0 4px 16px rgba(0, 0, 0, 0.12)',
          border: '1px solid #EEECEC',
          borderRadius: rem(8),
        },
        item: {
          fontSize: rem(14),
          padding: `${rem(8)} ${rem(12)}`,
          '&[data-hovered]': {
            backgroundColor: '#F9FBFC',
          },
        },
      },
    },

    NavLink: {
      styles: {
        root: {
          borderRadius: rem(6),
          fontWeight: 500,
          transition: 'all 0.15s ease',
        },
      },
    },

    Table: {
      styles: {
        thead: {
          backgroundColor: '#F9FBFC',
        },
        th: {
          fontWeight: 700,
          fontSize: rem(12),
          color: '#696E7B',
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          borderBottom: '2px solid #EEECEC',
        },
        td: {
          fontSize: rem(14),
          color: '#4C5773',
          borderBottom: '1px solid #F0F0F0',
        },
        tr: {
          '&:hover': {
            backgroundColor: '#F9FBFC',
          },
        },
      },
    },

    Notification: {
      styles: {
        root: {
          boxShadow: '0 4px 16px rgba(0, 0, 0, 0.12)',
          border: '1px solid #EEECEC',
        },
      },
    },

    Divider: {
      styles: {
        root: {
          borderColor: '#EEECEC',
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
          backgroundColor: 'rgba(0, 0, 0, 0.15)',
          borderRadius: rem(4),
          '&:hover': {
            backgroundColor: 'rgba(0, 0, 0, 0.25)',
          },
        },
      },
    },

    Skeleton: {
      styles: {
        root: {
          '&::after': {
            background: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.4), transparent)',
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
