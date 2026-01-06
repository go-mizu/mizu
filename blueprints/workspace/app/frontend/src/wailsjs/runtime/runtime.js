// Wails Runtime - stub for browser mode
// These will be replaced by Wails at build time

const isWails = typeof window !== 'undefined' && window.runtime !== undefined;

function wailsRuntime(method, ...args) {
  if (isWails && window.runtime && window.runtime[method]) {
    return window.runtime[method](...args);
  }
  return Promise.resolve();
}

// Events
const eventListeners = new Map();

export function EventsOn(eventName, callback) {
  if (isWails) {
    return wailsRuntime('EventsOn', eventName, callback);
  }
  if (!eventListeners.has(eventName)) {
    eventListeners.set(eventName, new Set());
  }
  eventListeners.get(eventName).add(callback);
  return () => {
    eventListeners.get(eventName)?.delete(callback);
  };
}

export function EventsOnce(eventName, callback) {
  if (isWails) {
    return wailsRuntime('EventsOnce', eventName, callback);
  }
  const wrapper = (...args) => {
    callback(...args);
    eventListeners.get(eventName)?.delete(wrapper);
  };
  return EventsOn(eventName, wrapper);
}

export function EventsOnMultiple(eventName, callback, maxCallbacks) {
  if (isWails) {
    return wailsRuntime('EventsOnMultiple', eventName, callback, maxCallbacks);
  }
  let count = 0;
  const wrapper = (...args) => {
    callback(...args);
    count++;
    if (count >= maxCallbacks) {
      eventListeners.get(eventName)?.delete(wrapper);
    }
  };
  return EventsOn(eventName, wrapper);
}

export function EventsOff(eventName, ...additionalEventNames) {
  if (isWails) {
    return wailsRuntime('EventsOff', eventName, ...additionalEventNames);
  }
  [eventName, ...additionalEventNames].forEach(name => {
    eventListeners.delete(name);
  });
}

export function EventsEmit(eventName, ...data) {
  if (isWails) {
    return wailsRuntime('EventsEmit', eventName, ...data);
  }
  const listeners = eventListeners.get(eventName);
  if (listeners) {
    listeners.forEach(cb => cb(...data));
  }
}

// Window
export function WindowReload() {
  if (isWails) return wailsRuntime('WindowReload');
  window.location.reload();
}

export function WindowReloadApp() {
  if (isWails) return wailsRuntime('WindowReloadApp');
  window.location.reload();
}

export function WindowSetAlwaysOnTop(b) {
  return wailsRuntime('WindowSetAlwaysOnTop', b);
}

export function WindowSetSystemDefaultTheme() {
  return wailsRuntime('WindowSetSystemDefaultTheme');
}

export function WindowSetLightTheme() {
  return wailsRuntime('WindowSetLightTheme');
}

export function WindowSetDarkTheme() {
  return wailsRuntime('WindowSetDarkTheme');
}

export function WindowCenter() {
  return wailsRuntime('WindowCenter');
}

export function WindowSetTitle(title) {
  if (isWails) return wailsRuntime('WindowSetTitle', title);
  document.title = title;
}

export function WindowFullscreen() {
  if (isWails) return wailsRuntime('WindowFullscreen');
  document.documentElement.requestFullscreen?.();
}

export function WindowUnfullscreen() {
  if (isWails) return wailsRuntime('WindowUnfullscreen');
  document.exitFullscreen?.();
}

export function WindowIsFullscreen() {
  if (isWails) return wailsRuntime('WindowIsFullscreen');
  return Promise.resolve(!!document.fullscreenElement);
}

export function WindowSetSize(width, height) {
  return wailsRuntime('WindowSetSize', width, height);
}

export function WindowGetSize() {
  if (isWails) return wailsRuntime('WindowGetSize');
  return Promise.resolve({ w: window.innerWidth, h: window.innerHeight });
}

export function WindowSetMaxSize(width, height) {
  return wailsRuntime('WindowSetMaxSize', width, height);
}

export function WindowSetMinSize(width, height) {
  return wailsRuntime('WindowSetMinSize', width, height);
}

export function WindowSetPosition(x, y) {
  return wailsRuntime('WindowSetPosition', x, y);
}

export function WindowGetPosition() {
  if (isWails) return wailsRuntime('WindowGetPosition');
  return Promise.resolve({ x: window.screenX, y: window.screenY });
}

export function WindowHide() {
  return wailsRuntime('WindowHide');
}

export function WindowShow() {
  return wailsRuntime('WindowShow');
}

export function WindowMaximise() {
  return wailsRuntime('WindowMaximise');
}

export function WindowToggleMaximise() {
  return wailsRuntime('WindowToggleMaximise');
}

export function WindowUnmaximise() {
  return wailsRuntime('WindowUnmaximise');
}

export function WindowIsMaximised() {
  if (isWails) return wailsRuntime('WindowIsMaximised');
  return Promise.resolve(false);
}

export function WindowMinimise() {
  return wailsRuntime('WindowMinimise');
}

export function WindowUnminimise() {
  return wailsRuntime('WindowUnminimise');
}

export function WindowIsMinimised() {
  if (isWails) return wailsRuntime('WindowIsMinimised');
  return Promise.resolve(false);
}

export function WindowIsNormal() {
  if (isWails) return wailsRuntime('WindowIsNormal');
  return Promise.resolve(true);
}

export function WindowSetBackgroundColour(R, G, B, A) {
  if (isWails) return wailsRuntime('WindowSetBackgroundColour', R, G, B, A);
  document.body.style.backgroundColor = `rgba(${R}, ${G}, ${B}, ${A})`;
}

// Screen
export function ScreenGetAll() {
  if (isWails) return wailsRuntime('ScreenGetAll');
  return Promise.resolve([{
    isCurrent: true,
    isPrimary: true,
    width: window.screen.width,
    height: window.screen.height
  }]);
}

// Browser
export function BrowserOpenURL(url) {
  if (isWails) return wailsRuntime('BrowserOpenURL', url);
  window.open(url, '_blank');
}

// Clipboard
export function ClipboardGetText() {
  if (isWails) return wailsRuntime('ClipboardGetText');
  return navigator.clipboard.readText();
}

export function ClipboardSetText(text) {
  if (isWails) return wailsRuntime('ClipboardSetText', text);
  return navigator.clipboard.writeText(text).then(() => true);
}

// Log
export function LogPrint(message) {
  console.log(message);
}

export function LogTrace(message) {
  console.trace(message);
}

export function LogDebug(message) {
  console.debug(message);
}

export function LogInfo(message) {
  console.info(message);
}

export function LogWarning(message) {
  console.warn(message);
}

export function LogError(message) {
  console.error(message);
}

export function LogFatal(message) {
  console.error('[FATAL]', message);
}

// Environment
export function Environment() {
  if (isWails) return wailsRuntime('Environment');
  return Promise.resolve({
    buildType: 'browser',
    platform: navigator.platform,
    arch: 'unknown'
  });
}

// Quit
export function Quit() {
  if (isWails) return wailsRuntime('Quit');
  window.close();
}

// Hide
export function Hide() {
  return wailsRuntime('Hide');
}

// Show
export function Show() {
  return wailsRuntime('Show');
}
