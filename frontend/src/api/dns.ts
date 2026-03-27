import api from './client';

export interface Zone {
  id: number;
  user_id: number;
  name: string;
  status: string;
  serial: number;
  records?: Record[];
  created_at: string;
  updated_at: string;
}

export interface Record {
  id: number;
  zone_id: number;
  name: string;
  type: string;
  content: string;
  ttl: number;
  priority?: number;
  created_at: string;
  updated_at: string;
}

export const dnsApi = {
  listZones: (params?: { page?: number; size?: number }) =>
    api.get<{ data: Zone[]; meta: { page: number; size: number; total: number } }>('/dns/zones', { params }),

  getZone: (id: number) =>
    api.get<{ data: Zone }>(`/dns/zones/${id}`),

  createZone: (name: string) =>
    api.post<{ data: Zone }>('/dns/zones', { name }),

  deleteZone: (id: number) =>
    api.delete(`/dns/zones/${id}`),

  listRecords: (zoneId: number) =>
    api.get<{ data: Record[] }>(`/dns/zones/${zoneId}/records`),

  createRecord: (zoneId: number, data: { name: string; type: string; content: string; ttl?: number; priority?: number }) =>
    api.post<{ data: Record }>(`/dns/zones/${zoneId}/records`, data),

  updateRecord: (id: number, data: { name?: string; type?: string; content?: string; ttl?: number; priority?: number }) =>
    api.put(`/dns/records/${id}`, data),

  deleteRecord: (id: number) =>
    api.delete(`/dns/records/${id}`),

  exportZone: (id: number) =>
    api.get(`/dns/zones/${id}/export`, { responseType: 'text' }),
};
