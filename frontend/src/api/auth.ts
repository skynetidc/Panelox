import api from './client';

export interface User {
  id: number;
  email: string;
  name: string;
  role: string;
  status: string;
  language: string;
  created_at: string;
  updated_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface LoginResponse {
  user: User;
  tokens: TokenPair;
}

export const authApi = {
  login: (email: string, password: string) =>
    api.post<{ data: LoginResponse }>('/auth/login', { email, password }),

  register: (email: string, password: string, name: string) =>
    api.post<{ data: User }>('/auth/register', { email, password, name }),

  me: () => api.get<{ data: User }>('/auth/me'),

  logout: (refreshToken: string) =>
    api.post('/auth/logout', { refresh_token: refreshToken }),
};
