import { useColorScheme } from 'react-native'

export const lightTheme = {
  bg: '#ffffff',
  text: '#0f1419',
  secondary: '#536471',
  blue: '#1d9bf0',
  border: '#eff3f4',
  bgHover: '#f7f9f9',
  like: '#f91880',
  retweet: '#00ba7c',
  card: '#ffffff',
  searchBg: '#eff3f4',
  barBg: '#ffffff',
  tabActive: '#0f1419',
  tabInactive: '#536471',
}

export const darkTheme = {
  bg: '#000000',
  text: '#e7e9ea',
  secondary: '#71767b',
  blue: '#1d9bf0',
  border: '#2f3336',
  bgHover: '#080808',
  like: '#f91880',
  retweet: '#00ba7c',
  card: '#000000',
  searchBg: '#202327',
  barBg: '#000000',
  tabActive: '#e7e9ea',
  tabInactive: '#71767b',
}

export type Theme = typeof lightTheme

export function useTheme(): Theme {
  const scheme = useColorScheme()
  return scheme === 'dark' ? darkTheme : lightTheme
}
