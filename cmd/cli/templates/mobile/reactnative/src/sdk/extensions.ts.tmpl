import { MizuRuntime } from '../runtime/MizuRuntime';
import { createAuthToken } from '../runtime/tokenStore';
import { AuthResponse } from './types';

/**
 * Store an auth response token in the runtime
 */
export async function storeAuthToken(
  runtime: MizuRuntime,
  response: AuthResponse
): Promise<void> {
  const expiresAt = new Date(Date.now() + response.token.expiresIn * 1000);
  await runtime.tokenStore.setToken(
    createAuthToken({
      accessToken: response.token.accessToken,
      refreshToken: response.token.refreshToken,
      expiresAt,
    })
  );
}
