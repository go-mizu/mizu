import { useState, useCallback, useEffect } from 'react';
import { PushManager, PushPermissionState } from '../pwa/pushManager';
import { AppConfig } from '../config/config';

interface UsePushNotificationsReturn {
  isSupported: boolean;
  permission: PushPermissionState;
  isSubscribed: boolean;
  subscribe: () => Promise<boolean>;
  unsubscribe: () => Promise<boolean>;
}

export function usePushNotifications(): UsePushNotificationsReturn {
  const [permission, setPermission] = useState<PushPermissionState>('prompt');
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [pushManager] = useState(() => new PushManager({
    vapidPublicKey: AppConfig.vapidPublicKey,
  }));

  useEffect(() => {
    async function checkStatus() {
      const perm = await pushManager.getPermissionState();
      setPermission(perm);

      const sub = await pushManager.getSubscription();
      setIsSubscribed(sub !== null);
    }
    checkStatus();
  }, [pushManager]);

  const subscribe = useCallback(async () => {
    const hasPermission = await pushManager.requestPermission();
    if (!hasPermission) {
      setPermission('denied');
      return false;
    }

    setPermission('granted');
    const subscription = await pushManager.subscribe();
    if (subscription) {
      setIsSubscribed(true);
      return true;
    }
    return false;
  }, [pushManager]);

  const unsubscribe = useCallback(async () => {
    const success = await pushManager.unsubscribe();
    if (success) {
      setIsSubscribed(false);
    }
    return success;
  }, [pushManager]);

  return {
    isSupported: PushManager.isSupported(),
    permission,
    isSubscribed,
    subscribe,
    unsubscribe,
  };
}
