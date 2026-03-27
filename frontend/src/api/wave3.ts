import api from './client';

// Git Deploy
export interface GitDeploy { id: number; website_id: number; repo_url: string; branch: string; auto_deploy: boolean; webhook_secret: string; build_command: string; deploy_path: string; last_commit?: string; last_deploy?: string; last_status?: string; last_log?: string; created_at: string; }
export interface DeployHistory { id: number; deploy_id: number; commit_hash: string; commit_msg: string; status: string; duration_ms: number; created_at: string; }

// Webhooks
export interface WebhookDef { id: number; name: string; url: string; secret?: string; events: string[]; enabled: boolean; last_sent?: string; last_status?: number; created_at: string; }

// Staging
export interface StagingSite { id: number; website_id: number; subdomain: string; status: string; url: string; created_at: string; }

// Billing
export interface UsageReport { user_id: number; user_name: string; email: string; period: string; disk_bytes: number; bandwidth: number; cpu_minutes: number; websites: number; databases: number; }

// Analytics
export interface WebsiteAnalytics { total_requests: number; total_bandwidth: number; unique_visitors: number; top_pages: { path: string; count: number }[]; status_codes: Record<string, number>; hourly_hits: { hour: string; count: number }[]; }

export const gitDeployApi = {
  list: () => api.get<{ data: GitDeploy[] }>('/git-deploy'),
  get: (id: number) => api.get<{ data: GitDeploy }>(`/git-deploy/${id}`),
  create: (data: { website_id: number; repo_url: string; branch?: string; build_command?: string; auto_deploy?: boolean }) => api.post<{ data: GitDeploy }>('/git-deploy', data),
  delete: (id: number) => api.delete(`/git-deploy/${id}`),
  deploy: (id: number) => api.post(`/git-deploy/${id}/deploy`),
  rollback: (id: number, commitHash?: string) => api.post(`/git-deploy/${id}/rollback`, { commit_hash: commitHash }),
  history: (id: number) => api.get<{ data: DeployHistory[] }>(`/git-deploy/${id}/history`),
};

export const webhookApi = {
  list: () => api.get<{ data: WebhookDef[] }>('/webhooks'),
  events: () => api.get<{ data: string[] }>('/webhooks/events'),
  create: (data: { name: string; url: string; secret?: string; events: string[] }) => api.post<{ data: WebhookDef }>('/webhooks', data),
  delete: (id: number) => api.delete(`/webhooks/${id}`),
  toggle: (id: number) => api.post(`/webhooks/${id}/toggle`),
  test: (id: number) => api.post<{ data: { status: string } }>(`/webhooks/${id}/test`),
};

export const stagingApi = {
  list: (websiteId?: number) => api.get<{ data: StagingSite[] }>('/staging', { params: { website_id: websiteId } }),
  create: (websiteId: number) => api.post<{ data: StagingSite }>('/staging', { website_id: websiteId }),
  delete: (id: number) => api.delete(`/staging/${id}`),
  sync: (id: number) => api.post(`/staging/${id}/sync`),
  promote: (id: number) => api.post(`/staging/${id}/promote`),
};

export const billingApi = {
  usage: () => api.get<{ data: UsageReport[] }>('/billing/usage'),
  history: (userId?: number) => api.get<{ data: UsageReport[] }>('/billing/history', { params: { user_id: userId } }),
  collect: () => api.post('/billing/collect'),
};

export const analyticsApi = {
  stats: (websiteId: number, days?: number) => api.get<{ data: WebsiteAnalytics }>(`/analytics/${websiteId}`, { params: { days } }),
};
