import api from './client';

export interface DashboardOverview {
  users: number;
  websites: number;
  ssl_certificates: number;
  dns_zones: number;
  databases: number;
  mail_domains: number;
  firewall_rules: number;
}

export interface SystemStats {
  go_routines: number;
  memory_alloc: number;
  memory_sys: number;
  num_cpu: number;
  go_version: string;
}

export const dashboardApi = {
  overview: () => api.get<{ data: DashboardOverview }>('/dashboard/overview'),
  stats: () => api.get<{ data: SystemStats }>('/dashboard/stats'),
};
