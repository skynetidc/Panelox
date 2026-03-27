import api from './client';

export interface MailDomain {
  id: number;
  user_id: number;
  domain: string;
  container_id?: string;
  dkim_public_key?: string;
  status: string;
  created_at: string;
}

export interface Mailbox {
  id: number;
  mail_domain_id: number;
  address: string;
  quota_mb: number;
  status: string;
  created_at: string;
}

export const mailApi = {
  listDomains: (params?: { page?: number; size?: number }) =>
    api.get<{ data: MailDomain[]; meta: { page: number; size: number; total: number } }>('/mail/domains', { params }),
  createDomain: (domain: string) => api.post<{ data: MailDomain }>('/mail/domains', { domain }),
  deleteDomain: (id: number) => api.delete(`/mail/domains/${id}`),
  listMailboxes: (domainId: number) => api.get<{ data: Mailbox[] }>(`/mail/domains/${domainId}/mailboxes`),
  createMailbox: (domainId: number, address: string, password: string, quotaMb?: number) =>
    api.post<{ data: Mailbox }>(`/mail/domains/${domainId}/mailboxes`, { address, password, quota_mb: quotaMb || 1024 }),
  deleteMailbox: (id: number) => api.delete(`/mail/mailboxes/${id}`),
};
