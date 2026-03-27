import api from './client';

export interface Node {
  id: number;
  name: string;
  host: string;
  port: number;
  ssh_user: string;
  auth_mode: string;
  agent_port: number;
  status: string;
  is_local: boolean;
  node_group: string;
  region: string;
  last_ping?: string;
  created_at: string;
}

export const nodeApi = {
  list: () => api.get<{ data: Node[] }>('/nodes'),
  create: (data: { name: string; host: string; port?: number; ssh_user?: string; auth_mode: string; password?: string; private_key?: string; agent_port?: number }) =>
    api.post<{ data: Node }>('/nodes', data),
  delete: (id: number) => api.delete(`/nodes/${id}`),
  test: (id: number) => api.post<{ data: { status: string; message: string } }>(`/nodes/${id}/test`),
  installAgent: (id: number) => api.post<{ data: { message: string } }>(`/nodes/${id}/install-agent`),
  pingAll: () => api.post<{ data: Record<string, string> }>('/nodes/ping-all'),
};
