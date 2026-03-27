import axios from 'axios';
import { useAuthStore } from '../stores/auth';

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config;
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true;
      const { refreshToken, setTokens, logout } = useAuthStore.getState();
      if (refreshToken) {
        try {
          const res = await axios.post('/api/v1/auth/refresh', {
            refresh_token: refreshToken,
          });
          const tokens = res.data.data;
          setTokens(tokens.access_token, tokens.refresh_token);
          original.headers.Authorization = `Bearer ${tokens.access_token}`;
          return api(original);
        } catch {
          logout();
        }
      } else {
        logout();
      }
    }
    // License limit error - show upgrade prompt
    if (error.response?.status === 403 && error.response?.data?.error?.code === 'LICENSE_LIMIT') {
      const msg = error.response.data.error.message;
      const tier = error.response.data.error.tier;
      if (tier === 'free') {
        const goToLicense = window.confirm(`${msg}\n\nGo to License page to upgrade?`);
        if (goToLicense) {
          window.location.href = '/license';
        }
      } else {
        alert(msg);
      }
    }

    return Promise.reject(error);
  }
);

export default api;
