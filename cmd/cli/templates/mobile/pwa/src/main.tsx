import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { registerServiceWorker } from './pwa/serviceWorker';
import { initInstallPrompt } from './pwa/installPrompt';
import './index.css';

// Initialize PWA features
initInstallPrompt();

registerServiceWorker({
  onUpdate: (registration) => {
    console.log('[App] New version available', registration);
  },
  onSuccess: (registration) => {
    console.log('[App] Content cached for offline use', registration);
  },
  onOffline: () => {
    console.log('[App] You are offline');
  },
  onOnline: () => {
    console.log('[App] You are online');
  },
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
