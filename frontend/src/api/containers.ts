import api from './client';

export interface ContainerInfo {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  ports: string;
  created: string;
}

export const containerApi = {
  list: () => api.get<{ data: ContainerInfo[] }>('/containers'),
  start: (id: string) => api.post(`/containers/${id}/start`),
  stop: (id: string) => api.post(`/containers/${id}/stop`),
  restart: (id: string) => api.post(`/containers/${id}/restart`),
  remove: (id: string) => api.delete(`/containers/${id}`),
  logs: (id: string, tail?: number) => api.get<{ data: { logs: string } }>(`/containers/${id}/logs`, { params: { tail: tail || 100 } }),
};
