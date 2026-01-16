import React from 'react'
import ReactDOM from 'react-dom/client'
import { MantineProvider, createTheme } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import '@mantine/core/styles.css'
import '@mantine/notifications/styles.css'

const theme = createTheme({
  primaryColor: 'green',
  colors: {
    green: [
      '#e6fff2',
      '#ccffe6',
      '#99ffcc',
      '#66ffb3',
      '#3ECF8E',
      '#33a874',
      '#29855c',
      '#1f6346',
      '#144230',
      '#0a211a',
    ],
  },
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  defaultRadius: 'md',
  black: '#1a1a2e',
  white: '#ffffff',
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="dark">
      <Notifications position="top-right" />
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </MantineProvider>
  </React.StrictMode>
)
