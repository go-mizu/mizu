export interface PushConfig {
  vapidPublicKey: string;
  onPermissionGranted?: () => void;
  onPermissionDenied?: () => void;
}

export type PushPermissionState = 'granted' | 'denied' | 'prompt';

/**
 * Manages Web Push notifications
 */
export class PushManager {
  private config: PushConfig;
  private registration: ServiceWorkerRegistration | null = null;

  constructor(config: PushConfig) {
    this.config = config;
  }

  /**
   * Checks if push notifications are supported
   */
  static isSupported(): boolean {
    return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
  }

  /**
   * Gets current permission state
   */
  async getPermissionState(): Promise<PushPermissionState> {
    if (!PushManager.isSupported()) return 'denied';

    const permission = await navigator.permissions.query({ name: 'notifications' });
    return permission.state as PushPermissionState;
  }

  /**
   * Requests push notification permission
   */
  async requestPermission(): Promise<boolean> {
    if (!PushManager.isSupported()) return false;

    const result = await Notification.requestPermission();
    if (result === 'granted') {
      this.config.onPermissionGranted?.();
      return true;
    }

    this.config.onPermissionDenied?.();
    return false;
  }

  /**
   * Subscribes to push notifications
   */
  async subscribe(): Promise<PushSubscription | null> {
    if (!PushManager.isSupported()) return null;

    try {
      this.registration = await navigator.serviceWorker.ready;

      const subscription = await this.registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: this.urlBase64ToUint8Array(this.config.vapidPublicKey),
      });

      return subscription;
    } catch (error) {
      console.error('[Push] Subscription failed:', error);
      return null;
    }
  }

  /**
   * Gets current subscription
   */
  async getSubscription(): Promise<PushSubscription | null> {
    if (!PushManager.isSupported()) return null;

    const registration = await navigator.serviceWorker.ready;
    return registration.pushManager.getSubscription();
  }

  /**
   * Unsubscribes from push notifications
   */
  async unsubscribe(): Promise<boolean> {
    const subscription = await this.getSubscription();
    if (!subscription) return false;

    return subscription.unsubscribe();
  }

  private urlBase64ToUint8Array(base64String: string): Uint8Array {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }

    return outputArray;
  }
}
