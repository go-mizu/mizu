export interface SyncConfig {
  onSyncComplete?: () => void;
  onSyncError?: (error: Error) => void;
}

/**
 * Manages background sync for offline requests
 */
export class SyncManager {
  private config: SyncConfig;

  constructor(config: SyncConfig = {}) {
    this.config = config;
  }

  /**
   * Checks if background sync is supported
   */
  static isSupported(): boolean {
    return 'serviceWorker' in navigator && 'SyncManager' in window;
  }

  /**
   * Registers a sync event
   */
  async registerSync(tag: string = 'background-sync'): Promise<boolean> {
    if (!SyncManager.isSupported()) return false;

    try {
      const registration = await navigator.serviceWorker.ready;
      await (registration as unknown as { sync: { register: (tag: string) => Promise<void> } }).sync.register(tag);
      return true;
    } catch (error) {
      console.error('[Sync] Registration failed:', error);
      return false;
    }
  }

  /**
   * Gets all pending sync tags
   */
  async getTags(): Promise<string[]> {
    if (!SyncManager.isSupported()) return [];

    try {
      const registration = await navigator.serviceWorker.ready;
      return await (registration as unknown as { sync: { getTags: () => Promise<string[]> } }).sync.getTags();
    } catch {
      return [];
    }
  }
}

/**
 * Registers a periodic sync (if supported)
 */
export async function registerPeriodicSync(
  tag: string,
  minInterval: number
): Promise<boolean> {
  if (!('periodicSync' in ServiceWorkerRegistration.prototype)) {
    return false;
  }

  try {
    const status = await navigator.permissions.query({
      name: 'periodic-background-sync' as PermissionName,
    });

    if (status.state !== 'granted') {
      return false;
    }

    const registration = await navigator.serviceWorker.ready;
    await (registration as unknown as {
      periodicSync: { register: (tag: string, options: { minInterval: number }) => Promise<void> }
    }).periodicSync.register(tag, { minInterval });

    return true;
  } catch {
    return false;
  }
}
