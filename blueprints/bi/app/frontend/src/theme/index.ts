import { createTheme, MantineColorsTuple } from '@mantine/core'

// Metabase-inspired color palette
const brandBlue: MantineColorsTuple = [
  '#e7f5ff',
  '#d0ebff',
  '#a5d8ff',
  '#74c0fc',
  '#4dabf7',
  '#509EE3', // Primary brand blue
  '#228be6',
  '#1971c2',
  '#1864ab',
  '#145591',
]

const summarizeGreen: MantineColorsTuple = [
  '#e6f9ed',
  '#cef3dc',
  '#9de7b9',
  '#69db93',
  '#88BF4D', // Summarize green
  '#72a843',
  '#5c9239',
  '#467c2f',
  '#306525',
  '#1a4f1b',
]

const filterPurple: MantineColorsTuple = [
  '#f5f0fa',
  '#ebe1f5',
  '#d5c3eb',
  '#bfa5e0',
  '#A989C5', // Filter purple
  '#9371b0',
  '#7d5a9b',
  '#674386',
  '#512c71',
  '#3b155c',
]

const warningYellow: MantineColorsTuple = [
  '#fff9e6',
  '#fff3cc',
  '#ffe799',
  '#ffdb66',
  '#F9CF48', // Warning yellow
  '#d4ad3c',
  '#af8c30',
  '#8a6a24',
  '#654918',
  '#40270c',
]

const errorRed: MantineColorsTuple = [
  '#fff0f0',
  '#ffe0e0',
  '#ffc0c0',
  '#ffa0a0',
  '#EF8C8C', // Error red
  '#c97070',
  '#a35454',
  '#7d3838',
  '#571c1c',
  '#310000',
]

const successGreen: MantineColorsTuple = [
  '#e6f9ed',
  '#cef3dc',
  '#9de7b9',
  '#69db93',
  '#88BF4D',
  '#72a843',
  '#5c9239',
  '#467c2f',
  '#306525',
  '#1a4f1b',
]

// Chart colors for visualizations (24 colors)
export const chartColors = [
  '#509EE3', '#88BF4D', '#A989C5', '#F9CF48',
  '#EF8C8C', '#98D9D9', '#F2A86F', '#7172AD',
  '#4C5773', '#77C9D4', '#ED6D47', '#DC75CD',
  '#76DFCA', '#B4C2C8', '#FFB86C', '#8BE9FD',
  '#50FA7B', '#FF79C6', '#BD93F9', '#F1FA8C',
  '#6272A4', '#44475A', '#282A36', '#FF5555',
]

export const theme = createTheme({
  primaryColor: 'brand',
  colors: {
    brand: brandBlue,
    summarize: summarizeGreen,
    filter: filterPurple,
    warning: warningYellow,
    error: errorRed,
    success: successGreen,
  },
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif',
  fontFamilyMonospace: 'SF Mono, Monaco, Inconsolata, "Roboto Mono", monospace',
  headings: {
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif',
  },
  defaultRadius: 'md',
  radius: {
    xs: '2px',
    sm: '4px',
    md: '6px',
    lg: '8px',
    xl: '12px',
  },
  shadows: {
    xs: '0 1px 2px rgba(0, 0, 0, 0.05)',
    sm: '0 1px 3px rgba(0, 0, 0, 0.1)',
    md: '0 2px 8px rgba(0, 0, 0, 0.15)',
    lg: '0 4px 16px rgba(0, 0, 0, 0.2)',
    xl: '0 8px 32px rgba(0, 0, 0, 0.25)',
  },
  other: {
    sidebarBg: '#2e353b',
    sidebarText: 'rgba(255, 255, 255, 0.7)',
    sidebarTextHover: '#ffffff',
    sidebarActive: '#509EE3',
  },
  components: {
    Button: {
      defaultProps: {
        radius: 'md',
      },
    },
    Card: {
      defaultProps: {
        radius: 'md',
        shadow: 'sm',
      },
    },
    Modal: {
      defaultProps: {
        radius: 'lg',
        centered: true,
      },
    },
    TextInput: {
      defaultProps: {
        radius: 'md',
      },
    },
    Select: {
      defaultProps: {
        radius: 'md',
      },
    },
    Tabs: {
      styles: {
        tab: {
          fontWeight: 500,
        },
      },
    },
  },
})

// Sidebar theme tokens
export const sidebarTheme = {
  bg: '#2e353b',
  text: 'rgba(255, 255, 255, 0.7)',
  textHover: '#ffffff',
  active: '#509EE3',
  border: 'rgba(255, 255, 255, 0.1)',
  inputBg: 'rgba(255, 255, 255, 0.1)',
}

// Semantic colors
export const semanticColors = {
  brand: '#509EE3',
  brandHover: '#4285c9',
  brandLight: '#e7f5ff',
  summarize: '#88BF4D',
  filter: '#A989C5',
  success: '#88BF4D',
  warning: '#F9CF48',
  error: '#EF8C8C',
  info: '#509EE3',
}
