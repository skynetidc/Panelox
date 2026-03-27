import api from './client';

export interface APIKey {
  id: number;
  user_id: number;
  name: string;
  key_prefix: string;
  permissions: string;
  expires_at?: string;
  last_used?: string;
  created_at: string;
}

export interface APIKeyWithSecret extends APIKey {
  key: string;
}

export const apiKeysApi = {
  list: () => api.get<{ data: APIKey[] }>('/api-keys'),
  create: (data: { name: string; permissions?: string; expires_in?: number }) =>
    api.post<{ data: APIKeyWithSecret }>('/api-keys', data),
  delete: (id: number) => api.delete(`/api-keys/${id}`),
};
