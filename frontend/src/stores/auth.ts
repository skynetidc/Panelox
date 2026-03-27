import { create } from 'zustand';
import type { User } from '../api/auth';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  setAuth: (user: User, accessToken: string, refreshToken: string) => void;
  setTokens: (accessToken: string, refreshToken: string) => void;
  setUser: (user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: localStorage.getItem('panelox_at'),
  refreshToken: localStorage.getItem('panelox_rt'),
  isAuthenticated: !!localStorage.getItem('panelox_at'),

  setAuth: (user, accessToken, refreshToken) => {
    localStorage.setItem('panelox_at', accessToken);
    localStorage.setItem('panelox_rt', refreshToken);
    set({ user, accessToken, refreshToken, isAuthenticated: true });
  },

  setTokens: (accessToken, refreshToken) => {
    localStorage.setItem('panelox_at', accessToken);
    localStorage.setItem('panelox_rt', refreshToken);
    set({ accessToken, refreshToken });
  },

  setUser: (user) => set({ user }),

  logout: () => {
    localStorage.removeItem('panelox_at');
    localStorage.removeItem('panelox_rt');
    set({ user: null, accessToken: null, refreshToken: null, isAuthenticated: false });
  },
}));
