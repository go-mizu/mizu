import { describe, test, expect, beforeEach } from 'vitest';
import { InMemoryTokenStore, createAuthToken, isTokenExpired } from '../src/runtime/tokenStore';
import { MizuError, APIError } from '../src/runtime/errors';

describe('InMemoryTokenStore', () => {
  let store: InMemoryTokenStore;

  beforeEach(() => {
    store = new InMemoryTokenStore();
  });

  test('returns null when no token stored', async () => {
    expect(await store.getToken()).toBeNull();
  });

  test('stores and retrieves token', async () => {
    const token = createAuthToken({
      accessToken: 'test123',
      refreshToken: 'refresh456',
    });

    await store.setToken(token);
    const retrieved = await store.getToken();

    expect(retrieved?.accessToken).toBe('test123');
    expect(retrieved?.refreshToken).toBe('refresh456');
  });

  test('clears token', async () => {
    const token = createAuthToken({ accessToken: 'test123' });
    await store.setToken(token);
    await store.clearToken();

    expect(await store.getToken()).toBeNull();
  });

  test('notifies observers on token change', async () => {
    let notifiedToken: unknown = undefined;
    store.onTokenChange((token) => {
      notifiedToken = token;
    });

    const token = createAuthToken({ accessToken: 'test123' });
    await store.setToken(token);

    expect((notifiedToken as { accessToken: string })?.accessToken).toBe('test123');
  });
});

describe('AuthToken', () => {
  test('isTokenExpired returns false when no expiry', () => {
    const token = createAuthToken({ accessToken: 'test' });
    expect(isTokenExpired(token)).toBe(false);
  });

  test('isTokenExpired returns true when expired', () => {
    const token = createAuthToken({
      accessToken: 'test',
      expiresAt: new Date(Date.now() - 1000),
    });
    expect(isTokenExpired(token)).toBe(true);
  });

  test('isTokenExpired returns false when not expired', () => {
    const token = createAuthToken({
      accessToken: 'test',
      expiresAt: new Date(Date.now() + 3600000),
    });
    expect(isTokenExpired(token)).toBe(false);
  });
});

describe('MizuError', () => {
  test('creates network error', () => {
    const error = MizuError.network(new Error('Connection failed'));
    expect(error.isNetwork).toBe(true);
    expect(error.message).toBe('Network error');
  });

  test('creates api error', () => {
    const apiError: APIError = { code: 'test_error', message: 'Test message' };
    const error = MizuError.api(apiError);
    expect(error.isApi).toBe(true);
    expect(error.apiError?.code).toBe('test_error');
  });

  test('creates offline error', () => {
    const error = MizuError.offline();
    expect(error.isOffline).toBe(true);
    expect(error.message).toBe('You are offline');
  });

  test('creates queued error', () => {
    const error = MizuError.queued();
    expect(error.isQueued).toBe(true);
    expect(error.message).toBe('Request queued for later');
  });
});
