const DEVICE_ID_KEY = 'mizu_device_id';

export interface DeviceInfoData {
  deviceId: string;
  appVersion: string;
  appBuild: string;
  browser: string;
  browserVersion: string;
  os: string;
  osVersion: string;
  timezone: string;
  locale: string;
  pwaMode: 'browser' | 'standalone' | 'fullscreen' | 'minimal-ui';
  screenWidth: number;
  screenHeight: number;
  devicePixelRatio: number;
}

let cachedInfo: DeviceInfoData | null = null;

export class DeviceInfo {
  /**
   * Collects device/browser information
   */
  static async collect(): Promise<DeviceInfoData> {
    if (cachedInfo) return cachedInfo;

    const deviceId = this.getOrCreateDeviceId();
    const ua = this.parseUserAgent();
    const pwaMode = this.getPWAMode();

    cachedInfo = {
      deviceId,
      appVersion: import.meta.env.VITE_APP_VERSION ?? '1.0.0',
      appBuild: import.meta.env.VITE_APP_BUILD ?? '1',
      browser: ua.browser,
      browserVersion: ua.browserVersion,
      os: ua.os,
      osVersion: ua.osVersion,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      locale: navigator.language,
      pwaMode,
      screenWidth: window.screen.width,
      screenHeight: window.screen.height,
      devicePixelRatio: window.devicePixelRatio,
    };

    return cachedInfo;
  }

  private static getOrCreateDeviceId(): string {
    let deviceId = localStorage.getItem(DEVICE_ID_KEY);
    if (!deviceId) {
      deviceId = crypto.randomUUID();
      localStorage.setItem(DEVICE_ID_KEY, deviceId);
    }
    return deviceId;
  }

  private static getPWAMode(): 'browser' | 'standalone' | 'fullscreen' | 'minimal-ui' {
    if (window.matchMedia('(display-mode: fullscreen)').matches) {
      return 'fullscreen';
    }
    if (window.matchMedia('(display-mode: standalone)').matches) {
      return 'standalone';
    }
    if (window.matchMedia('(display-mode: minimal-ui)').matches) {
      return 'minimal-ui';
    }
    return 'browser';
  }

  private static parseUserAgent(): {
    browser: string;
    browserVersion: string;
    os: string;
    osVersion: string;
  } {
    const ua = navigator.userAgent;

    // Browser detection
    let browser = 'Unknown';
    let browserVersion = '';
    if (ua.includes('Firefox/')) {
      browser = 'Firefox';
      browserVersion = ua.match(/Firefox\/([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Edg/')) {
      browser = 'Edge';
      browserVersion = ua.match(/Edg\/([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Chrome/')) {
      browser = 'Chrome';
      browserVersion = ua.match(/Chrome\/([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Safari/') && !ua.includes('Chrome')) {
      browser = 'Safari';
      browserVersion = ua.match(/Version\/([\d.]+)/)?.[1] ?? '';
    }

    // OS detection
    let os = 'Unknown';
    let osVersion = '';
    if (ua.includes('Windows')) {
      os = 'Windows';
      osVersion = ua.match(/Windows NT ([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('Mac OS X')) {
      os = 'macOS';
      osVersion = ua.match(/Mac OS X ([\d_]+)/)?.[1]?.replace(/_/g, '.') ?? '';
    } else if (ua.includes('Android')) {
      os = 'Android';
      osVersion = ua.match(/Android ([\d.]+)/)?.[1] ?? '';
    } else if (ua.includes('iPhone') || ua.includes('iPad')) {
      os = 'iOS';
      osVersion = ua.match(/OS ([\d_]+)/)?.[1]?.replace(/_/g, '.') ?? '';
    } else if (ua.includes('Linux')) {
      os = 'Linux';
    }

    return { browser, browserVersion, os, osVersion };
  }

  /** Clears cached info (useful for testing) */
  static clearCache(): void {
    cachedInfo = null;
  }
}
