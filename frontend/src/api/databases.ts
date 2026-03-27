import api from './client';

export interface DBServer {
  id: number;
  user_id: number;
  name: string;
  type: string;
  version: string;
  host: string;
  port: number;
  container_id?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Database {
  id: number;
  server_id: number;
  name: string;
  charset: string;
  created_at: string;
}

export interface DBUser {
  id: number;
  server_id: number;
  username: string;
  databases?: string[];
  created_at: string;
}

export const databaseApi = {
  // Servers
  listServers: (params?: { page?: number; size?: number }) =>
    api.get<{ data: DBServer[]; meta: { page: number; size: number; total: number } }>('/databases/servers', { params }),

  getServer: (id: number) =>
    api.get<{ data: DBServer }>(`/databases/servers/${id}`),

  createServer: (data: { name: string; type: string; version?: string; root_password: string; port?: number }) =>
    api.post<{ data: DBServer }>('/databases/servers', data),

  deleteServer: (id: number) =>
    api.delete(`/databases/servers/${id}`),

  operateServer: (id: number, action: 'start' | 'stop' | 'restart') =>
    api.post(`/databases/servers/${id}/operate`, { action }),

  // Databases
  listDatabases: (serverId: number) =>
    api.get<{ data: Database[] }>(`/databases/servers/${serverId}/databases`),

  createDatabase: (serverId: number, name: string, charset?: string) =>
    api.post<{ data: Database }>(`/databases/servers/${serverId}/databases`, { name, charset }),

  deleteDatabase: (id: number) =>
    api.delete(`/databases/${id}`),

  // Users
  listDBUsers: (serverId: number) =>
    api.get<{ data: DBUser[] }>(`/databases/servers/${serverId}/users`),

  createDBUser: (serverId: number, username: string, password: string, databases?: string[]) =>
    api.post<{ data: DBUser }>(`/databases/servers/${serverId}/users`, { username, password, databases }),

  deleteDBUser: (id: number) =>
    api.delete(`/databases/users/${id}`),
};
