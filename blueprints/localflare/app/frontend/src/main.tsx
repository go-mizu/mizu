import React from 'react'
import ReactDOM from 'react-dom/client'
import { MantineProvider, createTheme } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'
import '@mantine/code-highlight/styles.css'
import '@mantine/dates/styles.css'

const theme = createTheme({
  primaryColor: 'orange',
  colors: {
    orange: [
      '#fff4e6',
      '#ffe8cc',
      '#ffd8a8',
      '#ffc078',
      '#ffa94d',
      '#f6821f', // Cloudflare orange
      '#e8590c',
      '#d9480f',
      '#c92a2a',
      '#a61e4d',
    ],
  },
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif',
  defaultRadius: 'md',
  black: '#1a1a2e',
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <Notifications position="top-right" />
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </MantineProvider>
  </React.StrictMode>,
)
