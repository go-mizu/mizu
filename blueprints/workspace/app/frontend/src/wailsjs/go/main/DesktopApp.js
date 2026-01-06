// Wails generated bindings - stub for browser mode
// These will be replaced by Wails at build time

// Check if we're in Wails runtime
const isWails = typeof window !== 'undefined' && window.go !== undefined;

function wailsCall(method, ...args) {
  if (isWails && window.go && window.go.main && window.go.main.DesktopApp) {
    return window.go.main.DesktopApp[method](...args);
  }
  // Return mock values for browser development
  return Promise.resolve(null);
}

export function GetDataDir() {
  return wailsCall('GetDataDir');
}

export function GetBackendURL() {
  if (isWails) {
    return wailsCall('GetBackendURL');
  }
  // In browser mode, return the current origin or localhost
  return Promise.resolve(window.location.origin || 'http://localhost:8080');
}

export function GetVersion() {
  if (isWails) {
    return wailsCall('GetVersion');
  }
  return Promise.resolve({ version: 'dev', commit: 'browser', buildTime: new Date().toISOString() });
}

export function ShowNotification(title, message) {
  if (isWails) {
    return wailsCall('ShowNotification', title, message);
  }
  // Fallback to browser notifications
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification(title, { body: message });
  }
  return Promise.resolve();
}

export function OpenExternal(url) {
  if (isWails) {
    return wailsCall('OpenExternal', url);
  }
  window.open(url, '_blank');
  return Promise.resolve();
}

export function ShowOpenDialog(title, filters) {
  if (isWails) {
    return wailsCall('ShowOpenDialog', title, filters);
  }
  console.warn('ShowOpenDialog not available in browser mode');
  return Promise.resolve([]);
}

export function ShowSaveDialog(title, defaultFilename) {
  if (isWails) {
    return wailsCall('ShowSaveDialog', title, defaultFilename);
  }
  console.warn('ShowSaveDialog not available in browser mode');
  return Promise.resolve('');
}

export function ShowDirectoryDialog(title) {
  if (isWails) {
    return wailsCall('ShowDirectoryDialog', title);
  }
  console.warn('ShowDirectoryDialog not available in browser mode');
  return Promise.resolve('');
}

export function ReadFile(path) {
  if (isWails) {
    return wailsCall('ReadFile', path);
  }
  console.warn('ReadFile not available in browser mode');
  return Promise.resolve([]);
}

export function WriteFile(path, data) {
  if (isWails) {
    return wailsCall('WriteFile', path, data);
  }
  console.warn('WriteFile not available in browser mode');
  return Promise.resolve();
}

export function FileExists(path) {
  if (isWails) {
    return wailsCall('FileExists', path);
  }
  return Promise.resolve(false);
}

export function GetHomeDir() {
  if (isWails) {
    return wailsCall('GetHomeDir');
  }
  return Promise.resolve('/');
}

export function Minimize() {
  if (isWails) {
    return wailsCall('Minimize');
  }
  return Promise.resolve();
}

export function Maximize() {
  if (isWails) {
    return wailsCall('Maximize');
  }
  return Promise.resolve();
}

export function Close() {
  if (isWails) {
    return wailsCall('Close');
  }
  window.close();
  return Promise.resolve();
}

export function SetTitle(title) {
  if (isWails) {
    return wailsCall('SetTitle', title);
  }
  document.title = title;
  return Promise.resolve();
}

export function ToggleFullscreen() {
  if (isWails) {
    return wailsCall('ToggleFullscreen');
  }
  if (document.fullscreenElement) {
    document.exitFullscreen();
  } else {
    document.documentElement.requestFullscreen();
  }
  return Promise.resolve();
}
