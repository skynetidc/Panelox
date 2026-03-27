import api from './client';

export interface FirewallRule {
  id: number;
  chain: string;
  protocol: string;
  src_ip?: string;
  dst_port: string;
  action: string;
  description?: string;
  position: number;
  created_at: string;
}

export interface FirewallStatus {
  enabled: boolean;
  backend: string;
}

export const firewallApi = {
  status: () => api.get<{ data: FirewallStatus }>('/firewall/status'),
  listRules: () => api.get<{ data: FirewallRule[] }>('/firewall/rules'),
  createRule: (data: { protocol: string; dst_port: string; action?: string; src_ip?: string; chain?: string; description?: string }) =>
    api.post<{ data: FirewallRule }>('/firewall/rules', data),
  deleteRule: (id: number) => api.delete(`/firewall/rules/${id}`),
  reapply: () => api.post('/firewall/reapply'),
};
