const isDevelopment = import.meta.env.DEV;

export const AppConfig = {
  get baseURL(): string {
    if (isDevelopment) {
      return 'http://localhost:3000';
    }
    return import.meta.env.VITE_API_URL ?? 'https://api.example.com';
  },

  timeout: 30000,

  debug: isDevelopment,

  vapidPublicKey: import.meta.env.VITE_VAPID_PUBLIC_KEY ?? '',
};
