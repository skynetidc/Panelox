import api from './client';

export interface PHPVersion { version: string; image: string; installed: boolean; }
export interface PHPRuntime { id: number; version: string; container_id?: string; container_name: string; status: string; port: number; extensions: string[]; created_at: string; }
export interface Redirect { id: number; website_id: number; type: string; source: string; target: string; enabled: boolean; created_at: string; }

export const phpApi = {
  versions: () => api.get<{ data: PHPVersion[] }>('/php/versions'),
  runtimes: () => api.get<{ data: PHPRuntime[] }>('/php/runtimes'),
  install: (version: string, extensions?: string[]) => api.post<{ data: PHPRuntime }>('/php/runtimes', { version, extensions }),
  remove: (id: number) => api.delete(`/php/runtimes/${id}`),
  operate: (id: number, action: string) => api.post(`/php/runtimes/${id}/operate`, { action }),
};

export const websiteAdvancedApi = {
  accessLog: (id: number, lines?: number) => api.get<{ data: { logs: string; path: string } }>(`/websites/${id}/logs/access`, { params: { lines: lines || 100 } }),
  errorLog: (id: number, lines?: number) => api.get<{ data: { logs: string; path: string } }>(`/websites/${id}/logs/error`, { params: { lines: lines || 100 } }),
  listRedirects: (id: number) => api.get<{ data: Redirect[] }>(`/websites/${id}/redirects`),
  createRedirect: (id: number, data: { type: string; source: string; target: string }) => api.post(`/websites/${id}/redirects`, data),
  deleteRedirect: (websiteId: number, redirectId: number) => api.delete(`/websites/${websiteId}/redirects/${redirectId}`),
  updateSecurity: (id: number, data: any) => api.put(`/websites/${id}/security`, data),
  updateProxyConfig: (id: number, data: any) => api.put(`/websites/${id}/proxy-config`, data),
};
