// =============================================================================
// DESIGN TOKENS - Modern shadcn-inspired design system
// =============================================================================

// Color palette - HSL values for easy manipulation
export const colors = {
  // Backgrounds
  background: {
    DEFAULT: '#ffffff',
    muted: '#f9fafb',
    subtle: '#f3f4f6',
    accent: '#f1f5f9',
  },

  // Foregrounds (text)
  foreground: {
    DEFAULT: '#4c5773',
    muted: '#6b7280',
    subtle: '#9ca3af',
    inverse: '#ffffff',
  },

  // Primary (Brand Blue)
  primary: {
    DEFAULT: '#509ee3',
    hover: '#4285c9',
    light: '#e6f2ff',
    lighter: '#f0f6fc',
    dark: '#326597',
    foreground: '#ffffff',
  },

  // Success / Summarize (Green)
  success: {
    DEFAULT: '#84bb4c',
    hover: '#6fa83d',
    light: '#edf7e4',
    dark: '#467a1f',
    foreground: '#ffffff',
  },

  // Warning (Yellow)
  warning: {
    DEFAULT: '#f9d45c',
    hover: '#e0b73d',
    light: '#fffceb',
    dark: '#956f1c',
    foreground: '#4c5773',
  },

  // Error (Red)
  error: {
    DEFAULT: '#ed6e6e',
    hover: '#d65555',
    light: '#fef0f0',
    dark: '#a42323',
    foreground: '#ffffff',
  },

  // Info / Filter (Purple)
  info: {
    DEFAULT: '#7172ad',
    hover: '#5f6099',
    light: '#efeef5',
    dark: '#3b3c71',
    foreground: '#ffffff',
  },

  // Borders
  border: {
    DEFAULT: '#e5e7eb',
    muted: '#f3f4f6',
    strong: '#d1d5db',
  },

  // Focus ring
  ring: {
    DEFAULT: '#509ee3',
    offset: '#ffffff',
  },
} as const

// Semantic color aliases
export const semantic = {
  brand: colors.primary.DEFAULT,
  summarize: colors.success.DEFAULT,
  filter: colors.info.DEFAULT,
  positive: colors.success.DEFAULT,
  negative: colors.error.DEFAULT,
  neutral: colors.foreground.muted,
} as const

// Chart colors - Consistent palette for visualizations
export const chartPalette = [
  '#509ee3', // Blue
  '#84bb4c', // Green
  '#7172ad', // Purple
  '#f2a86f', // Orange
  '#98d9d9', // Teal
  '#ed6e6e', // Coral
  '#50c9ba', // Mint
  '#f9d45c', // Yellow
  '#7c79b8', // Dark purple
  '#b5d6a3', // Light green
  '#ffb366', // Light orange
  '#6cd4d4', // Bright teal
] as const

// Spacing scale (rem units, 4px base)
export const spacing = {
  0: '0',
  px: '1px',
  0.5: '0.125rem', // 2px
  1: '0.25rem',    // 4px
  1.5: '0.375rem', // 6px
  2: '0.5rem',     // 8px
  2.5: '0.625rem', // 10px
  3: '0.75rem',    // 12px
  3.5: '0.875rem', // 14px
  4: '1rem',       // 16px
  5: '1.25rem',    // 20px
  6: '1.5rem',     // 24px
  7: '1.75rem',    // 28px
  8: '2rem',       // 32px
  9: '2.25rem',    // 36px
  10: '2.5rem',    // 40px
  11: '2.75rem',   // 44px
  12: '3rem',      // 48px
  14: '3.5rem',    // 56px
  16: '4rem',      // 64px
  20: '5rem',      // 80px
  24: '6rem',      // 96px
} as const

// Border radius
export const radius = {
  none: '0',
  sm: '0.25rem',   // 4px
  DEFAULT: '0.375rem', // 6px
  md: '0.5rem',    // 8px
  lg: '0.75rem',   // 12px
  xl: '1rem',      // 16px
  '2xl': '1.5rem', // 24px
  full: '9999px',
} as const

// Shadows
export const shadows = {
  none: 'none',
  sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
  DEFAULT: '0 1px 3px 0 rgb(0 0 0 / 0.08), 0 1px 2px -1px rgb(0 0 0 / 0.08)',
  md: '0 4px 6px -1px rgb(0 0 0 / 0.08), 0 2px 4px -2px rgb(0 0 0 / 0.08)',
  lg: '0 10px 15px -3px rgb(0 0 0 / 0.08), 0 4px 6px -4px rgb(0 0 0 / 0.08)',
  xl: '0 20px 25px -5px rgb(0 0 0 / 0.08), 0 8px 10px -6px rgb(0 0 0 / 0.08)',
  hover: '0 4px 12px 0 rgb(0 0 0 / 0.1)',
  card: '0 1px 3px 0 rgb(0 0 0 / 0.06)',
  cardHover: '0 4px 12px 0 rgb(0 0 0 / 0.08)',
} as const

// Typography
export const typography = {
  fontFamily: {
    sans: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    mono: 'JetBrains Mono, Monaco, "SF Mono", Consolas, monospace',
  },
  fontSize: {
    xs: ['0.75rem', { lineHeight: '1rem' }],      // 12px
    sm: ['0.8125rem', { lineHeight: '1.25rem' }], // 13px
    base: ['0.875rem', { lineHeight: '1.5rem' }], // 14px
    lg: ['1rem', { lineHeight: '1.75rem' }],      // 16px
    xl: ['1.125rem', { lineHeight: '1.75rem' }],  // 18px
    '2xl': ['1.25rem', { lineHeight: '2rem' }],   // 20px
    '3xl': ['1.5rem', { lineHeight: '2rem' }],    // 24px
    '4xl': ['2rem', { lineHeight: '2.5rem' }],    // 32px
  },
  fontWeight: {
    normal: '400',
    medium: '500',
    semibold: '600',
    bold: '700',
  },
  letterSpacing: {
    tight: '-0.01em',
    normal: '0',
    wide: '0.01em',
    wider: '0.05em',
  },
} as const

// Transitions
export const transitions = {
  fast: '150ms cubic-bezier(0.4, 0, 0.2, 1)',
  normal: '200ms cubic-bezier(0.4, 0, 0.2, 1)',
  slow: '300ms cubic-bezier(0.4, 0, 0.2, 1)',
  colors: 'color 150ms, background-color 150ms, border-color 150ms',
  opacity: 'opacity 150ms',
  transform: 'transform 200ms',
  all: 'all 200ms cubic-bezier(0.4, 0, 0.2, 1)',
} as const

// Z-index scale
export const zIndex = {
  dropdown: 1000,
  sticky: 1020,
  fixed: 1030,
  modalBackdrop: 1040,
  modal: 1050,
  popover: 1060,
  tooltip: 1070,
  toast: 1080,
} as const

// Breakpoints
export const breakpoints = {
  sm: '640px',
  md: '768px',
  lg: '1024px',
  xl: '1280px',
  '2xl': '1536px',
} as const

// =============================================================================
// CSS CUSTOM PROPERTIES (for global.css)
// =============================================================================

export const cssVariables = `
  /* Colors - Background */
  --color-background: ${colors.background.DEFAULT};
  --color-background-muted: ${colors.background.muted};
  --color-background-subtle: ${colors.background.subtle};
  --color-background-accent: ${colors.background.accent};

  /* Colors - Foreground */
  --color-foreground: ${colors.foreground.DEFAULT};
  --color-foreground-muted: ${colors.foreground.muted};
  --color-foreground-subtle: ${colors.foreground.subtle};

  /* Colors - Primary */
  --color-primary: ${colors.primary.DEFAULT};
  --color-primary-hover: ${colors.primary.hover};
  --color-primary-light: ${colors.primary.light};
  --color-primary-foreground: ${colors.primary.foreground};

  /* Colors - Success */
  --color-success: ${colors.success.DEFAULT};
  --color-success-hover: ${colors.success.hover};
  --color-success-light: ${colors.success.light};
  --color-success-foreground: ${colors.success.foreground};

  /* Colors - Warning */
  --color-warning: ${colors.warning.DEFAULT};
  --color-warning-hover: ${colors.warning.hover};
  --color-warning-light: ${colors.warning.light};

  /* Colors - Error */
  --color-error: ${colors.error.DEFAULT};
  --color-error-hover: ${colors.error.hover};
  --color-error-light: ${colors.error.light};

  /* Colors - Info */
  --color-info: ${colors.info.DEFAULT};
  --color-info-hover: ${colors.info.hover};
  --color-info-light: ${colors.info.light};

  /* Colors - Border */
  --color-border: ${colors.border.DEFAULT};
  --color-border-muted: ${colors.border.muted};
  --color-border-strong: ${colors.border.strong};

  /* Colors - Ring */
  --color-ring: ${colors.ring.DEFAULT};

  /* Semantic */
  --color-brand: ${semantic.brand};
  --color-summarize: ${semantic.summarize};
  --color-filter: ${semantic.filter};

  /* Radius */
  --radius-sm: ${radius.sm};
  --radius: ${radius.DEFAULT};
  --radius-md: ${radius.md};
  --radius-lg: ${radius.lg};
  --radius-xl: ${radius.xl};
  --radius-full: ${radius.full};

  /* Shadows */
  --shadow-sm: ${shadows.sm};
  --shadow: ${shadows.DEFAULT};
  --shadow-md: ${shadows.md};
  --shadow-lg: ${shadows.lg};
  --shadow-card: ${shadows.card};
  --shadow-card-hover: ${shadows.cardHover};

  /* Transitions */
  --transition-fast: ${transitions.fast};
  --transition: ${transitions.normal};
  --transition-slow: ${transitions.slow};

  /* Typography */
  --font-sans: ${typography.fontFamily.sans};
  --font-mono: ${typography.fontFamily.mono};
`
