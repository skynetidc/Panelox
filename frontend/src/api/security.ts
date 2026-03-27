import api from './client';

// Login Log
export interface LoginEntry { id: number; user_id?: number; email: string; ip_address: string; user_agent: string; success: boolean; message?: string; created_at: string; }

// 2FA
export interface TOTPStatus { enabled: boolean; }
export interface TOTPSetup { secret: string; uri: string; }

// Fail2Ban
export interface F2BRule { id: number; name: string; port: string; max_retry: number; ban_time: number; find_time: number; enabled: boolean; created_at: string; }
export interface BannedIP { id: number; rule_id: number; ip_address: string; reason: string; banned_at: string; expires_at: string; }

// Notifications
export interface Channel { id: number; name: string; type: string; config: any; enabled: boolean; created_at: string; }

// Alerts
export interface AlertRule { id: number; name: string; metric: string; operator: string; threshold: number; duration_sec: number; channel_id?: number; enabled: boolean; last_triggered?: string; status: string; created_at: string; }

// Snapshots
export interface Snapshot { id: number; name: string; description: string; file_path: string; file_size: number; includes: string[]; status: string; message?: string; created_at: string; }

export const securityApi = {
  // Login log
  loginLog: (params?: { page?: number; size?: number }) => api.get<{ data: LoginEntry[]; meta: any }>('/login-log', { params }),

  // 2FA
  totpStatus: () => api.get<{ data: TOTPStatus }>('/2fa/status'),
  totpSetup: () => api.post<{ data: TOTPSetup }>('/2fa/setup'),
  totpEnable: (code: string) => api.post('/2fa/enable', { code }),
  totpDisable: (code: string) => api.post('/2fa/disable', { code }),

  // Fail2Ban
  f2bRules: () => api.get<{ data: F2BRule[] }>('/fail2ban/rules'),
  f2bCreateRule: (data: { name: string; port: string; max_retry?: number; ban_time?: number }) => api.post<{ data: F2BRule }>('/fail2ban/rules', data),
  f2bDeleteRule: (id: number) => api.delete(`/fail2ban/rules/${id}`),
  f2bToggleRule: (id: number) => api.post(`/fail2ban/rules/${id}/toggle`),
  f2bBanned: () => api.get<{ data: BannedIP[] }>('/fail2ban/banned'),
  f2bUnban: (ip: string) => api.post('/fail2ban/unban', { ip }),

  // Notifications
  channels: () => api.get<{ data: Channel[] }>('/notifications/channels'),
  createChannel: (data: { name: string; type: string; config: any }) => api.post<{ data: Channel }>('/notifications/channels', data),
  deleteChannel: (id: number) => api.delete(`/notifications/channels/${id}`),
  toggleChannel: (id: number) => api.post(`/notifications/channels/${id}/toggle`),
  testChannel: (id: number) => api.post<{ data: { status: string; message: string } }>(`/notifications/channels/${id}/test`),

  // Alerts
  alertRules: () => api.get<{ data: AlertRule[] }>('/alerts/rules'),
  createAlert: (data: { name: string; metric: string; operator?: string; threshold: number; duration_sec?: number; channel_id?: number }) => api.post<{ data: AlertRule }>('/alerts/rules', data),
  deleteAlert: (id: number) => api.delete(`/alerts/rules/${id}`),
  toggleAlert: (id: number) => api.post(`/alerts/rules/${id}/toggle`),

  // Snapshots
  snapshots: () => api.get<{ data: Snapshot[] }>('/snapshots'),
  createSnapshot: (data: { name?: string; description?: string; includes?: string[] }) => api.post<{ data: Snapshot }>('/snapshots', data),
  deleteSnapshot: (id: number) => api.delete(`/snapshots/${id}`),
  rollback: (id: number) => api.post(`/snapshots/${id}/rollback`),
  downloadSnapshot: (id: number) => `/api/v1/snapshots/${id}/download`,
};
