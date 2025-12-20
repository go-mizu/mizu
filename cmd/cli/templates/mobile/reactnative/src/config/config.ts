import { Platform } from 'react-native';

const isDevelopment = __DEV__;

export const AppConfig = {
  get baseURL(): string {
    if (isDevelopment) {
      // Use 10.0.2.2 for Android emulator, localhost for iOS simulator
      return Platform.select({
        android: 'http://10.0.2.2:3000',
        ios: 'http://localhost:3000',
        default: 'http://localhost:3000',
      }) as string;
    }
    return 'https://api.example.com';
  },

  timeout: 30000,

  debug: isDevelopment,
};
