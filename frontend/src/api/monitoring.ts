import api from './client';

export interface Metrics {
  cpu_percent: number;
  mem_used: number;
  mem_total: number;
  disk_used: number;
  disk_total: number;
  net_in: number;
  net_out: number;
  load_1: number;
  load_5: number;
  load_15: number;
  uptime?: string;
  hostname?: string;
  os?: string;
  kernel?: string;
  cpu_cores?: number;
  created_at?: string;
}

export const monitoringApi = {
  current: () => api.get<{ data: Metrics }>('/monitoring/current'),
  history: (hours?: number) => api.get<{ data: Metrics[] }>('/monitoring/history', { params: { hours: hours || 24 } }),
};
