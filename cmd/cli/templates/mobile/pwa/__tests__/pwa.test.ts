import { describe, test, expect, vi, beforeEach } from 'vitest';

describe('Install Prompt', () => {
  beforeEach(() => {
    vi.resetModules();
  });

  test('initial state is not-available', async () => {
    const { getInstallState } = await import('../src/pwa/installPrompt');
    expect(getInstallState()).toBe('not-available');
  });
});

describe('Push Manager', () => {
  test('isSupported returns false when APIs not available', async () => {
    const { PushManager } = await import('../src/pwa/pushManager');
    expect(PushManager.isSupported()).toBe(false);
  });
});

describe('Sync Manager', () => {
  test('isSupported returns false when APIs not available', async () => {
    const { SyncManager } = await import('../src/pwa/syncManager');
    expect(SyncManager.isSupported()).toBe(false);
  });
});
