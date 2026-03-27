import api from './client';

export interface Backup {
  id: number;
  name: string;
  type: string;
  source_id?: number;
  file_path: string;
  file_size: number;
  status: string;
  message?: string;
  created_at: string;
}

export const backupApi = {
  list: (params?: { page?: number; size?: number }) =>
    api.get<{ data: Backup[]; meta: { page: number; size: number; total: number } }>('/backups', { params }),
  create: (type: string, name?: string, sourceId?: number) =>
    api.post<{ data: Backup }>('/backups', { type, name, source_id: sourceId }),
  delete: (id: number) => api.delete(`/backups/${id}`),
  restore: (id: number) => api.post(`/backups/${id}/restore`),
  downloadUrl: (id: number) => `/api/v1/backups/${id}/download`,
};
