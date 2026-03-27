import api from './client';

export interface Certificate {
  id: number;
  user_id: number;
  acme_account_id?: number;
  domains: string[];
  provider: string;
  key_type: string;
  auto_renew: boolean;
  expires_at?: string;
  issued_at?: string;
  status: string;
  message?: string;
  http_config: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface AcmeAccount {
  id: number;
  email: string;
  provider: string;
  key_type: string;
  created_at: string;
}

export const sslApi = {
  // Certificates
  list: (params?: { page?: number; size?: number }) =>
    api.get<{ data: Certificate[]; meta: { page: number; size: number; total: number } }>('/ssl', { params }),

  get: (id: number) =>
    api.get<{ data: Certificate }>(`/ssl/${id}`),

  issue: (data: {
    domains: string[];
    acme_account_id: number;
    key_type?: string;
    auto_renew?: boolean;
    http_config?: string;
    description?: string;
  }) => api.post<{ data: Certificate }>('/ssl', data),

  upload: (data: { certificate: string; private_key: string; description?: string }) =>
    api.post<{ data: Certificate }>('/ssl/upload', data),

  delete: (id: number) =>
    api.delete(`/ssl/${id}`),

  renew: (id: number) =>
    api.post(`/ssl/${id}/renew`),

  apply: (certificateId: number, websiteId: number, httpConfig?: string) =>
    api.post('/ssl/apply', {
      certificate_id: certificateId,
      website_id: websiteId,
      http_config: httpConfig || 'HTTPToHTTPS',
    }),

  remove: (websiteId: number) =>
    api.post('/ssl/remove', { website_id: websiteId }),

  // ACME Accounts
  listAccounts: () =>
    api.get<{ data: AcmeAccount[] }>('/acme-accounts'),

  createAccount: (email: string, provider?: string) =>
    api.post<{ data: AcmeAccount }>('/acme-accounts', { email, provider: provider || 'letsencrypt' }),

  deleteAccount: (id: number) =>
    api.delete(`/acme-accounts/${id}`),
};
