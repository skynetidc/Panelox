import api from './client';

export interface QuotaUsage {
  user_id: number; max_websites: number; max_databases: number; max_ftp_users: number; max_mail_domains: number; max_disk_mb: number; max_bandwidth_mb: number;
  used_websites: number; used_databases: number; used_ftp_users: number; used_mail_domains: number; used_disk_mb: number;
}
export interface Package { id: number; name: string; max_websites: number; max_databases: number; max_ftp_users: number; max_mail_domains: number; max_disk_mb: number; max_bandwidth_mb: number; max_users: number; price_monthly: number; }
export interface UserBilling { user_id: number; name: string; email: string; role: string; status: string; suspended: boolean; package: string; quota?: QuotaUsage; }

export const quotaApi = {
  myQuota: () => api.get<{ data: QuotaUsage }>('/quotas/my'),
  userQuota: (userId: number) => api.get<{ data: QuotaUsage }>(`/quotas/user/${userId}`),
  setQuota: (userId: number, data: Partial<QuotaUsage>) => api.put(`/quotas/user/${userId}`, data),
  listPackages: () => api.get<{ data: Package[] }>('/packages'),
  createPackage: (data: Partial<Package>) => api.post<{ data: Package }>('/packages', data),
  updatePackage: (id: number, data: Partial<Package>) => api.put(`/packages/${id}`, data),
  deletePackage: (id: number) => api.delete(`/packages/${id}`),
  assignPackage: (pkgId: number, userId: number) => api.post(`/packages/${pkgId}/assign/${userId}`),
  suspendUser: (userId: number, reason?: string) => api.post(`/users/${userId}/suspend`, { reason }),
  unsuspendUser: (userId: number) => api.post(`/users/${userId}/unsuspend`),
  billingDashboard: () => api.get<{ data: UserBilling[] }>('/users/billing'),
};
