// Design System Tokens - Duolingo-inspired Light Theme
// Based on Duolingo's official design guidelines

export const colors = {
  // Primary Brand Colors
  primary: {
    green: '#58CC02',
    greenHover: '#4CAD02',
    greenPressed: '#4CAD02',
    greenShadow: '#58A700',
    greenLight: '#D7FFB8',
    greenLighter: '#E5F7D3',
  },

  // Secondary Colors
  secondary: {
    blue: '#1CB0F6',
    blueHover: '#1899D6',
    blueShadow: '#1899D6',
    blueLight: '#DDF4FF',
  },

  // Accent Colors
  accent: {
    yellow: '#FFC800',
    yellowLight: '#FFF4CC',
    orange: '#FF9600',
    orangeLight: '#FFE8CC',
    purple: '#CE82FF',
    purpleLight: '#F3E5FF',
    pink: '#FF4B4B',
    pinkLight: '#FFDFE0',
    red: '#FF4B4B',
    redLight: '#FFDFE0',
  },

  // Neutral Colors (Light Theme)
  neutral: {
    white: '#FFFFFF',
    background: '#F7F7F7',
    surface: '#FFFFFF',
    surfaceHover: '#F7F7F7',
    border: '#E5E5E5',
    borderHover: '#AFAFAF',
    divider: '#E5E5E5',
  },

  // Text Colors
  text: {
    primary: '#4B4B4B',
    secondary: '#777777',
    muted: '#AFAFAF',
    disabled: '#CDCDCD',
    onPrimary: '#FFFFFF',
    onDark: '#FFFFFF',
    link: '#1CB0F6',
  },

  // Semantic Colors
  semantic: {
    success: '#58CC02',
    successLight: '#D7FFB8',
    error: '#FF4B4B',
    errorLight: '#FFDFE0',
    warning: '#FFC800',
    warningLight: '#FFF4CC',
    info: '#1CB0F6',
    infoLight: '#DDF4FF',
  },

  // Skill Node Colors
  skill: {
    active: '#58CC02',
    activeShadow: '#58A700',
    completed: '#FFC800',
    completedShadow: '#E5B400',
    legendary: '#FFC800',
    available: '#58CC02',
    locked: '#E5E5E5',
    lockedIcon: '#AFAFAF',
    cracked: '#CE82FF',
  },

  // Progress Colors
  progress: {
    empty: '#E5E5E5',
    fill: '#58CC02',
    fillLegendary: '#FFC800',
  },
} as const

export const typography = {
  fontFamily: {
    primary: '"Nunito", "DIN Round Pro", system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    mono: '"Fira Code", "Source Code Pro", "Consolas", monospace',
  },

  fontSize: {
    xs: '0.75rem',     // 12px
    sm: '0.875rem',    // 14px
    base: '1rem',      // 16px
    lg: '1.125rem',    // 18px
    xl: '1.25rem',     // 20px
    '2xl': '1.5rem',   // 24px
    '3xl': '1.875rem', // 30px
    '4xl': '2.25rem',  // 36px
    '5xl': '3rem',     // 48px
  },

  fontWeight: {
    normal: '400',
    medium: '500',
    semibold: '600',
    bold: '700',
    extrabold: '800',
  },

  lineHeight: {
    tight: '1.2',
    snug: '1.375',
    normal: '1.5',
    relaxed: '1.625',
    loose: '2',
  },

  letterSpacing: {
    tighter: '-0.05em',
    tight: '-0.025em',
    normal: '0',
    wide: '0.025em',
    wider: '0.05em',
    widest: '0.1em',
  },
} as const

export const spacing = {
  0: '0',
  px: '1px',
  0.5: '0.125rem',  // 2px
  1: '0.25rem',     // 4px
  1.5: '0.375rem',  // 6px
  2: '0.5rem',      // 8px
  2.5: '0.625rem',  // 10px
  3: '0.75rem',     // 12px
  3.5: '0.875rem',  // 14px
  4: '1rem',        // 16px
  5: '1.25rem',     // 20px
  6: '1.5rem',      // 24px
  7: '1.75rem',     // 28px
  8: '2rem',        // 32px
  9: '2.25rem',     // 36px
  10: '2.5rem',     // 40px
  11: '2.75rem',    // 44px
  12: '3rem',       // 48px
  14: '3.5rem',     // 56px
  16: '4rem',       // 64px
  20: '5rem',       // 80px
  24: '6rem',       // 96px
} as const

export const layout = {
  sidebarWidth: '256px',
  sidebarCollapsedWidth: '80px',
  rightSidebarWidth: '368px',
  headerHeight: '60px',
  maxContentWidth: '600px',
  lessonMaxWidth: '600px',
  lessonHeaderHeight: '56px',
  lessonFooterHeight: '80px',
} as const

export const borderRadius = {
  none: '0',
  sm: '0.25rem',    // 4px
  base: '0.5rem',   // 8px
  md: '0.75rem',    // 12px
  lg: '1rem',       // 16px
  xl: '1.25rem',    // 20px
  '2xl': '1.5rem',  // 24px
  '3xl': '2rem',    // 32px
  full: '9999px',
} as const

export const shadows = {
  none: 'none',
  sm: '0 1px 2px rgba(0, 0, 0, 0.05)',
  base: '0 1px 3px rgba(0, 0, 0, 0.1), 0 1px 2px rgba(0, 0, 0, 0.06)',
  md: '0 4px 6px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06)',
  lg: '0 10px 15px rgba(0, 0, 0, 0.1), 0 4px 6px rgba(0, 0, 0, 0.05)',
  xl: '0 20px 25px rgba(0, 0, 0, 0.1), 0 10px 10px rgba(0, 0, 0, 0.04)',

  // Duolingo-style button shadows (solid color bottom)
  button: {
    green: '0 4px 0 #58A700',
    greenHover: '0 2px 0 #58A700',
    blue: '0 4px 0 #1899D6',
    blueHover: '0 2px 0 #1899D6',
    gray: '0 4px 0 #E5E5E5',
    grayHover: '0 2px 0 #CDCDCD',
    red: '0 4px 0 #EA2B2B',
    redHover: '0 2px 0 #EA2B2B',
    yellow: '0 4px 0 #E5B400',
    yellowHover: '0 2px 0 #E5B400',
    white: '0 4px 0 #E5E5E5',
    whiteHover: '0 2px 0 #E5E5E5',
  },

  // Skill node shadows
  skill: {
    active: '0 4px 0 #58A700',
    completed: '0 4px 0 #E5B400',
    locked: 'none',
  },

  // Card shadows
  card: '0 2px 4px rgba(0, 0, 0, 0.08)',
  cardHover: '0 4px 8px rgba(0, 0, 0, 0.12)',
} as const

export const transitions = {
  duration: {
    fast: '100ms',
    normal: '200ms',
    slow: '300ms',
    slower: '500ms',
  },

  timing: {
    ease: 'ease',
    easeIn: 'ease-in',
    easeOut: 'ease-out',
    easeInOut: 'ease-in-out',
    linear: 'linear',
    bounce: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
  },
} as const

export const zIndex = {
  hide: -1,
  base: 0,
  dropdown: 1000,
  sticky: 1100,
  fixed: 1200,
  modalBackdrop: 1300,
  modal: 1400,
  popover: 1500,
  tooltip: 1600,
  toast: 1700,
} as const

// Skill node size configurations
export const skillNode = {
  size: {
    sm: '56px',
    md: '72px',
    lg: '88px',
  },
  iconSize: {
    sm: '24px',
    md: '32px',
    lg: '40px',
  },
} as const

// Exercise UI configurations
export const exercise = {
  choiceMinHeight: '56px',
  wordBankPillHeight: '40px',
  progressBarHeight: '16px',
} as const

// Export type for colors
export type Colors = typeof colors
export type Typography = typeof typography
export type Spacing = typeof spacing
