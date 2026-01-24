import React from 'react'
import ReactDOM from 'react-dom/client'
import { MantineProvider, createTheme } from '@mantine/core'
import '@mantine/core/styles.css'
import './styles/globals.css'
import App from './App'

const theme = createTheme({
  fontFamily: 'Inter, system-ui, sans-serif',
  primaryColor: 'blue',
  colors: {
    blue: [
      '#e7f0ff',
      '#ccdeff',
      '#99bcff',
      '#6699ff',
      '#4285f4',
      '#3377e8',
      '#2266dc',
      '#1155d0',
      '#0044c4',
      '#0033b8',
    ],
  },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider theme={theme}>
      <App />
    </MantineProvider>
  </React.StrictMode>,
)
