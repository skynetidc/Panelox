import api from './client';

export interface App {
  id: number;
  key: string;
  name: string;
  description: string;
  icon: string;
  category: string;
  versions: string[];
  env_schema: any[];
}

export interface AppInstall {
  id: number;
  app_key: string;
  name: string;
  version: string;
  status: string;
  env_values: Record<string, string>;
  created_at: string;
}

export const appApi = {
  catalog: () => api.get<{ data: App[] }>('/apps/catalog'),
  installed: () => api.get<{ data: AppInstall[] }>('/apps/installed'),
  install: (data: { app_key: string; name: string; version?: string; env_values?: Record<string, string> }) =>
    api.post<{ data: AppInstall }>('/apps/install', data),
  uninstall: (id: number) => api.delete(`/apps/installed/${id}`),
  operate: (id: number, action: 'start' | 'stop' | 'restart') =>
    api.post(`/apps/installed/${id}/operate`, { action }),
};
