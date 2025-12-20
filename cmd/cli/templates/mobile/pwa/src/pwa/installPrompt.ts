interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[];
  readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
  prompt(): Promise<void>;
}

export type InstallState = 'not-available' | 'available' | 'installed' | 'dismissed';

let deferredPrompt: BeforeInstallPromptEvent | null = null;
let installState: InstallState = 'not-available';
let stateListeners: Set<(state: InstallState) => void> = new Set();

/**
 * Initializes install prompt handling
 */
export function initInstallPrompt(): void {
  // Check if already installed
  if (window.matchMedia('(display-mode: standalone)').matches) {
    installState = 'installed';
    return;
  }

  // Listen for install prompt event
  window.addEventListener('beforeinstallprompt', (e: Event) => {
    e.preventDefault();
    deferredPrompt = e as BeforeInstallPromptEvent;
    installState = 'available';
    notifyListeners();
  });

  // Listen for successful installation
  window.addEventListener('appinstalled', () => {
    installState = 'installed';
    deferredPrompt = null;
    notifyListeners();
  });
}

/**
 * Gets current install state
 */
export function getInstallState(): InstallState {
  return installState;
}

/**
 * Shows the install prompt
 */
export async function showInstallPrompt(): Promise<boolean> {
  if (!deferredPrompt) return false;

  await deferredPrompt.prompt();
  const { outcome } = await deferredPrompt.userChoice;

  if (outcome === 'accepted') {
    installState = 'installed';
  } else {
    installState = 'dismissed';
  }

  deferredPrompt = null;
  notifyListeners();

  return outcome === 'accepted';
}

/**
 * Dismisses the install prompt
 */
export function dismissInstallPrompt(): void {
  installState = 'dismissed';
  deferredPrompt = null;
  notifyListeners();
}

/**
 * Subscribes to install state changes
 */
export function onInstallStateChange(listener: (state: InstallState) => void): () => void {
  stateListeners.add(listener);
  return () => stateListeners.delete(listener);
}

function notifyListeners(): void {
  stateListeners.forEach((listener) => listener(installState));
}
