import api from './client';

export interface Website {
  id: number;
  user_id: number;
  domain: string;
  alias: string;
  type: string;
  status: string;
  document_root: string;
  proxy_target?: string;
  php_version?: string;
  ssl_enabled: boolean;
  ipv6: boolean;
  access_log: boolean;
  error_log: boolean;
  default_server: boolean;
  remark?: string;
  created_at: string;
  updated_at: string;
  domains?: WebsiteDomain[];
}

export interface WebsiteDomain {
  id: number;
  website_id: number;
  domain: string;
  port: number;
  ssl: boolean;
  created_at: string;
}

export interface CreateWebsiteRequest {
  domain: string;
  alias?: string;
  type: string;
  proxy_target?: string;
  php_version?: string;
  ipv6?: boolean;
  remark?: string;
  domains?: { domain: string; port: number }[];
}

export interface UpdateWebsiteRequest {
  domain?: string;
  proxy_target?: string;
  php_version?: string;
  ipv6?: boolean;
  default_server?: boolean;
  access_log?: boolean;
  error_log?: boolean;
  remark?: string;
}

export interface ListResponse<T> {
  data: T[];
  meta: { page: number; size: number; total: number };
}

export const websiteApi = {
  list: (params?: { page?: number; size?: number; search?: string; type?: string; status?: string }) =>
    api.get<ListResponse<Website>>('/websites', { params }),

  get: (id: number) =>
    api.get<{ data: Website }>(`/websites/${id}`),

  create: (data: CreateWebsiteRequest) =>
    api.post<{ data: Website }>('/websites', data),

  update: (id: number, data: UpdateWebsiteRequest) =>
    api.put(`/websites/${id}`, data),

  delete: (id: number) =>
    api.delete(`/websites/${id}`),

  operate: (id: number, action: 'start' | 'stop') =>
    api.post(`/websites/${id}/operate`, { action }),

  getNginxConfig: (id: number) =>
    api.get<{ data: { config: string } }>(`/websites/${id}/nginx`),

  updateNginxConfig: (id: number, config: string) =>
    api.put(`/websites/${id}/nginx`, { config }),

  listDomains: (id: number) =>
    api.get<{ data: WebsiteDomain[] }>(`/websites/${id}/domains`),

  addDomain: (id: number, domain: string, port: number) =>
    api.post<{ data: WebsiteDomain }>(`/websites/${id}/domains`, { domain, port }),

  deleteDomain: (websiteId: number, domainId: number) =>
    api.delete(`/websites/${websiteId}/domains/${domainId}`),
};
