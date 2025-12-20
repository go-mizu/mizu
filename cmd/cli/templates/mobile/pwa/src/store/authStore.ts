import { create } from 'zustand';
import { MizuRuntime } from '../runtime/MizuRuntime';
import { AuthToken } from '../runtime/tokenStore';

interface AuthState {
  isAuthenticated: boolean;
  token: AuthToken | null;
  setIsAuthenticated: (value: boolean) => void;
  setToken: (token: AuthToken | null) => void;
  signOut: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  token: null,
  setIsAuthenticated: (value) => set({ isAuthenticated: value }),
  setToken: (token) => set({ token, isAuthenticated: token !== null }),
  signOut: async () => {
    await MizuRuntime.shared.tokenStore.clearToken();
    set({ isAuthenticated: false, token: null });
  },
}));
