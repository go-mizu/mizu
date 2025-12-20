import Alpine from 'alpinejs'
import { createApp } from './app'
import './styles/index.css'

// Register the main app component
Alpine.data('app', createApp)

// Start Alpine
Alpine.start()

// Make Alpine available globally for debugging
declare global {
  interface Window {
    Alpine: typeof Alpine
  }
}
window.Alpine = Alpine
