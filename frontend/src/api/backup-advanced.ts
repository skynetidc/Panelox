import api from './client';

export interface BackupAccount { id: number; name: string; type: string; config: any; is_default: boolean; created_at: string; }
export interface BackupSchedule { id: number; name: string; type: string; account_id?: number; schedule: string; retention_days: number; retention_count: number; enabled: boolean; last_run?: string; last_status?: string; created_at: string; }

export const backupAdvApi = {
  // Accounts
  listAccounts: () => api.get<{ data: BackupAccount[] }>('/backup-accounts'),
  createAccount: (data: { name: string; type: string; config: any }) => api.post<{ data: BackupAccount }>('/backup-accounts', data),
  deleteAccount: (id: number) => api.delete(`/backup-accounts/${id}`),
  testAccount: (id: number) => api.post<{ data: { status: string; message?: string } }>(`/backup-accounts/${id}/test`),

  // Schedules
  listSchedules: () => api.get<{ data: BackupSchedule[] }>('/backup-schedules'),
  createSchedule: (data: { name: string; type: string; account_id?: number; schedule?: string; retention_days?: number; retention_count?: number }) =>
    api.post<{ data: BackupSchedule }>('/backup-schedules', data),
  deleteSchedule: (id: number) => api.delete(`/backup-schedules/${id}`),
  toggleSchedule: (id: number) => api.post(`/backup-schedules/${id}/toggle`),
  runSchedule: (id: number) => api.post(`/backup-schedules/${id}/run`),
};
