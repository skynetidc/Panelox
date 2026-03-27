import api from './client';

export interface AuditEntry {
  id: number;
  user_id?: number;
  action: string;
  resource: string;
  details?: string;
  ip_address: string;
  created_at: string;
}

export const settingsApi = {
  getAll: () => api.get<{ data: Record<string, string> }>('/settings'),
  update: (settings: Record<string, string>) => api.put('/settings', settings),
  listAudit: (params?: { page?: number; size?: number }) =>
    api.get<{ data: AuditEntry[]; meta: { page: number; size: number; total: number } }>('/audit', { params }),
  getBranding: () => api.get<{ data: any }>('/branding'),
  updateBranding: (data: any) => api.put('/branding', data),
};
