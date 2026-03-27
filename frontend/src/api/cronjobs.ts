import api from './client';

export interface CronJob {
  id: number;
  name: string;
  type: string;
  schedule: string;
  command: string;
  enabled: boolean;
  last_run?: string;
  last_status?: string;
  last_output?: string;
  created_at: string;
}

export const cronApi = {
  list: () => api.get<{ data: CronJob[] }>('/cronjobs'),
  create: (data: { name: string; type?: string; schedule: string; command: string }) =>
    api.post<{ data: CronJob }>('/cronjobs', data),
  update: (id: number, data: Partial<CronJob>) => api.put(`/cronjobs/${id}`, data),
  delete: (id: number) => api.delete(`/cronjobs/${id}`),
  run: (id: number) => api.post(`/cronjobs/${id}/run`),
  toggle: (id: number) => api.post(`/cronjobs/${id}/toggle`),
};
