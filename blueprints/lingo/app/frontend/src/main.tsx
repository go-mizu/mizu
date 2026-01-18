import React from 'react'
import ReactDOM from 'react-dom/client'
import { MantineProvider, createTheme } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import './styles/global.css'

const theme = createTheme({
  primaryColor: 'green',
  colors: {
    green: [
      '#e6f9ed',
      '#c1f0d4',
      '#9be7ba',
      '#76dea1',
      '#58cc02', // Duolingo green
      '#4cb302',
      '#3f9a02',
      '#338102',
      '#266801',
      '#1a4f01',
    ],
    blue: [
      '#e5f6ff',
      '#bfe9ff',
      '#99dcff',
      '#73cfff',
      '#1cb0f6', // Duolingo blue
      '#189cd9',
      '#1489bc',
      '#10759f',
      '#0c6282',
      '#084e65',
    ],
    yellow: [
      '#fff9e5',
      '#fff0bf',
      '#ffe799',
      '#ffde73',
      '#ffc800', // Duolingo gold
      '#e5b400',
      '#cca000',
      '#b28c00',
      '#997800',
      '#806400',
    ],
  },
  fontFamily: 'Nunito, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif',
  headings: {
    fontFamily: 'Nunito, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif',
    fontWeight: '800',
  },
  radius: {
    xs: '0.5rem',
    sm: '0.75rem',
    md: '1rem',
    lg: '1.5rem',
    xl: '2rem',
  },
  defaultRadius: 'lg',
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
