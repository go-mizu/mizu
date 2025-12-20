import { Platform } from 'react-native';
import * as Application from 'expo-application';
import * as Device from 'expo-device';
import * as Localization from 'expo-localization';
import * as SecureStore from 'expo-secure-store';
import 'react-native-get-random-values';
import { v4 as uuidv4 } from 'uuid';

const DEVICE_ID_KEY = 'mizu_device_id';

export interface DeviceInfoData {
  deviceId: string;
  appVersion: string;
  appBuild: string;
  model: string;
  platform: string;
  osVersion: string;
  timezone: string;
  locale: string;
}

let cachedInfo: DeviceInfoData | null = null;

export class DeviceInfo {
  /**
   * Collects device information
   */
  static async collect(): Promise<DeviceInfoData> {
    if (cachedInfo) return cachedInfo;

    const deviceId = await this.getOrCreateDeviceId();

    cachedInfo = {
      deviceId,
      appVersion: Application.nativeApplicationVersion ?? '1.0.0',
      appBuild: Application.nativeBuildVersion ?? '1',
      model: Device.modelName ?? 'Unknown',
      platform: Platform.OS,
      osVersion: Platform.Version?.toString() ?? 'Unknown',
      timezone: Localization.timezone,
      locale: Localization.locale,
    };

    return cachedInfo;
  }

  private static async getOrCreateDeviceId(): Promise<string> {
    let deviceId = await SecureStore.getItemAsync(DEVICE_ID_KEY);
    if (!deviceId) {
      deviceId = uuidv4();
      await SecureStore.setItemAsync(DEVICE_ID_KEY, deviceId);
    }
    return deviceId;
  }

  /** Clears cached info (useful for testing) */
  static clearCache(): void {
    cachedInfo = null;
  }
}
