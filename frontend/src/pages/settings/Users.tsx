import { useEffect, useState } from 'react';
import api from '../../api/client';

interface QuotaUsage {
  max_websites: number; used_websites: number;
  max_databases: number; used_databases: number;
  max_ftp_users: number; used_ftp_users: number;
  max_mail_domains: number; used_mail_domains: number;
  max_disk_mb: number; used_disk_mb: number;
  max_bandwidth_mb: number;
}

interface Package {
  id: number; name: string;
  max_websites: number; max_databases: number; max_ftp_users: number;
  max_mail_domains: number; max_disk_mb: number; max_bandwidth_mb: number;
  max_users: number; price_monthly: number;
}

interface BillingUser {
  user_id: number; name: string; email: string; role: string; status: string;
  suspended: boolean; package?: string; quota?: QuotaUsage;
}

type Tab = 'users' | 'packages';

const roleConfig: Record<string, { label: string; bg: string; color: string; icon: string }> = {
  admin:    { label: 'Admin',    bg: 'bg-red-50',    color: 'text-red-700',    icon: '🛡️' },
  reseller: { label: 'Reseller', bg: 'bg-purple-50', color: 'text-purple-700', icon: '🏪' },
  user:     { label: 'User',     bg: 'bg-blue-50',   color: 'text-blue-700',   icon: '👤' },
};

const statusConfig: Record<string, { bg: string; color: string; dot: string }> = {
  active:    { bg: 'bg-green-50', color: 'text-green-700', dot: 'bg-green-500' },
  suspended: { bg: 'bg-red-50',   color: 'text-red-700',   dot: 'bg-red-500' },
};

function usageBar(used: number, max: number) {
  const unlimited = max === 0;
  const pct = unlimited ? 0 : Math.min((used / max) * 100, 100);
  const atLimit = !unlimited && used >= max;
  return (
    <div className="flex items-center gap-2">
      <div className="w-16 bg-gray-100 rounded-full h-1.5">
        <div className={`h-1.5 rounded-full ${atLimit ? 'bg-red-500' : pct > 75 ? 'bg-amber-500' : 'bg-blue-500'}`}
          style={{ width: unlimited ? '0%' : `${pct}%` }} />
      </div>
      <span className={`text-xs tabular-nums ${atLimit ? 'text-red-600 font-medium' : 'text-gray-500'}`}>
        {used}/{unlimited ? '∞' : max}
      </span>
    </div>
  );
}

function formatDisk(mb: number) {
  if (mb === 0) return '∞';
  if (mb >= 1024) return `${(mb / 1024).toFixed(0)}GB`;
  return `${mb}MB`;
}

export default function Users() {
  const [tab, setTab] = useState<Tab>('users');
  const [billingUsers, setBillingUsers] = useState<BillingUser[]>([]);
  const [packages, setPackages] = useState<Package[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [showPkgForm, setShowPkgForm] = useState(false);
  const [newUser, setNewUser] = useState({ email: '', name: '', password: '', role: 'user' });
  const [newPkg, setNewPkg] = useState<Partial<Package>>({ name: '', max_websites: 10, max_databases: 10, max_ftp_users: 5, max_mail_domains: 5, max_disk_mb: 5120, max_bandwidth_mb: 0, max_users: 0, price_monthly: 0 });
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [editUser, setEditUser] = useState<BillingUser | null>(null);
  const [editQuota, setEditQuota] = useState<Partial<QuotaUsage>>({});

  const fetchUsers = async () => {
    setLoading(true);
    try { const res = await api.get<{ data: BillingUser[] }>('/users/billing'); setBillingUsers(res.data.data || []); }
    catch { setBillingUsers([]); }
    setLoading(false);
  };

  const fetchPackages = async () => {
    try { const res = await api.get<{ data: Package[] }>('/packages'); setPackages(res.data.data || []); } catch {}
  };

  useEffect(() => { fetchUsers(); fetchPackages(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setCreating(true);
    try { await api.post('/auth/register', newUser); setNewUser({ email: '', name: '', password: '', role: 'user' }); setShowCreate(false); fetchUsers(); }
    catch (err: any) { setError(err.response?.data?.error?.message || 'Failed'); }
    setCreating(false);
  };

  const handleCreatePkg = async (e: React.FormEvent) => {
    e.preventDefault(); setCreating(true);
    try { await api.post('/packages', newPkg); setShowPkgForm(false); fetchPackages(); }
    catch (err: any) { setError(err.response?.data?.error?.message || 'Failed'); }
    setCreating(false);
  };

  const handleSuspend = async (id: number, suspended: boolean) => {
    await api.post(`/users/${id}/${suspended ? 'unsuspend' : 'suspend'}`).catch(() => {});
    fetchUsers();
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Delete user "${name}"? This cannot be undone.`)) return;
    await api.delete(`/users/${id}`).catch(() => {});
    fetchUsers();
  };

  const handleAssignPkg = async (userId: number, pkgId: number) => {
    await api.post(`/packages/${pkgId}/assign/${userId}`).catch(() => {});
    fetchUsers();
  };

  const handleUpdateRole = async (id: number, role: string) => {
    await api.put(`/users/${id}`, { role }).catch(() => {});
    fetchUsers();
  };

  const handleSaveQuota = async () => {
    if (!editUser) return;
    await api.put(`/quotas/user/${editUser.user_id}`, editQuota).catch(() => {});
    setEditUser(null); fetchUsers();
  };

  const filtered = billingUsers.filter(u =>
    !search || u.name.toLowerCase().includes(search.toLowerCase()) || u.email.toLowerCase().includes(search.toLowerCase())
  );

  const stats = {
    total: billingUsers.length,
    active: billingUsers.filter(u => !u.suspended).length,
    suspended: billingUsers.filter(u => u.suspended).length,
    admins: billingUsers.filter(u => u.role === 'admin').length,
  };

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">User Management</h1>
          <p className="text-sm text-gray-400 mt-1">Manage users, packages, and resource quotas</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="inline-flex items-center gap-2 py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.5v15m7.5-7.5h-15" /></svg>
          Add User
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-3 mb-6">
        {[
          { label: 'Total Users', value: stats.total, icon: '👥', color: 'text-gray-700' },
          { label: 'Active', value: stats.active, icon: '✅', color: 'text-green-700' },
          { label: 'Suspended', value: stats.suspended, icon: '🚫', color: 'text-red-700' },
          { label: 'Admins', value: stats.admins, icon: '🛡️', color: 'text-purple-700' },
        ].map(s => (
          <div key={s.label} className="bg-white border border-gray-200 rounded-xl p-4">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-base">{s.icon}</span>
              <span className="text-xs text-gray-400 uppercase tracking-wider">{s.label}</span>
            </div>
            <span className={`text-2xl font-bold ${s.color}`}>{s.value}</span>
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex gap-1 bg-gray-100 p-0.5 rounded-lg">
          {[['users', 'Users'] as const, ['packages', 'Packages'] as const].map(([k, l]) => (
            <button key={k} onClick={() => setTab(k)} className={`px-4 py-1.5 text-sm font-medium rounded-md transition-all ${tab === k ? 'bg-white shadow-sm text-gray-800' : 'text-gray-500 hover:text-gray-700'}`}>{l}</button>
          ))}
        </div>
        {tab === 'users' && (
          <div className="relative">
            <svg className="absolute left-3 top-2.5 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
            <input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder="Search users..."
              className="pl-9 pr-3 py-2 border border-gray-200 rounded-lg text-sm w-64 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400" />
          </div>
        )}
      </div>

      {/* Create user modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <form onSubmit={handleCreate} className="bg-white rounded-xl p-6 w-[480px] space-y-4" onClick={e => e.stopPropagation()}>
            <h3 className="font-semibold text-gray-800">Create User</h3>
            {error && <div className="bg-red-50 text-red-600 text-xs px-3 py-2 rounded-lg">{error}</div>}
            <div className="grid grid-cols-2 gap-3">
              <div><label className="block text-xs font-medium text-gray-500 mb-1">Name</label><input type="text" value={newUser.name} onChange={e => setNewUser(u => ({ ...u, name: e.target.value }))} required className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
              <div><label className="block text-xs font-medium text-gray-500 mb-1">Email</label><input type="email" value={newUser.email} onChange={e => setNewUser(u => ({ ...u, email: e.target.value }))} required className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
              <div><label className="block text-xs font-medium text-gray-500 mb-1">Password</label><input type="password" value={newUser.password} onChange={e => setNewUser(u => ({ ...u, password: e.target.value }))} required className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
              <div><label className="block text-xs font-medium text-gray-500 mb-1">Role</label><select value={newUser.role} onChange={e => setNewUser(u => ({ ...u, role: e.target.value }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm"><option value="user">User</option><option value="reseller">Reseller</option><option value="admin">Admin</option></select></div>
            </div>
            <div className="flex gap-2 justify-end"><button type="button" onClick={() => setShowCreate(false)} className="py-2 px-4 text-sm rounded-lg border hover:bg-gray-50">Cancel</button><button type="submit" disabled={creating} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">{creating ? '...' : 'Create'}</button></div>
          </form>
        </div>
      )}

      {/* USERS TAB */}
      {tab === 'users' && (
        <div className="space-y-3">
          {loading ? (
            <div className="flex items-center justify-center py-20"><div className="w-8 h-8 border-2 border-gray-200 border-t-blue-600 rounded-full animate-spin" /></div>
          ) : filtered.length === 0 ? (
            <div className="bg-white border border-gray-200 rounded-xl p-16 text-center"><div className="text-4xl mb-3">👥</div><p className="text-gray-400">No users found</p></div>
          ) : filtered.map(u => {
            const rc = roleConfig[u.role] || roleConfig.user;
            const sc = u.suspended ? statusConfig.suspended : statusConfig.active;
            const q = u.quota;
            return (
              <div key={u.user_id} className={`bg-white border rounded-xl overflow-hidden transition-all hover:shadow-md ${u.suspended ? 'border-red-200 opacity-75' : 'border-gray-200'}`}>
                <div className="p-5">
                  <div className="flex items-start justify-between">
                    {/* User info */}
                    <div className="flex items-center gap-4">
                      <div className={`w-12 h-12 rounded-xl ${rc.bg} flex items-center justify-center text-xl`}>{rc.icon}</div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-sm text-gray-800">{u.name}</h3>
                          <span className={`py-0.5 px-2 text-[10px] font-bold rounded-full ${rc.bg} ${rc.color}`}>{rc.label}</span>
                          <span className={`inline-flex items-center gap-1 py-0.5 px-2 text-[10px] font-medium rounded-full ${sc.bg} ${sc.color}`}>
                            <span className={`w-1.5 h-1.5 rounded-full ${sc.dot}`} />{u.suspended ? 'Suspended' : 'Active'}
                          </span>
                        </div>
                        <p className="text-xs text-gray-400 mt-0.5">{u.email}</p>
                        {u.package && <p className="text-xs text-purple-600 mt-0.5">Package: {u.package}</p>}
                      </div>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-1">
                      {/* Assign package dropdown */}
                      {packages.length > 0 && (
                        <select value="" onChange={e => { if (e.target.value) handleAssignPkg(u.user_id, Number(e.target.value)); }}
                          className="py-1 px-2 text-xs border border-gray-200 rounded-lg text-gray-500">
                          <option value="">Assign Package</option>
                          {packages.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
                        </select>
                      )}
                      <select value={u.role} onChange={e => handleUpdateRole(u.user_id, e.target.value)}
                        className="py-1 px-2 text-xs border border-gray-200 rounded-lg text-gray-500">
                        <option value="user">User</option><option value="reseller">Reseller</option><option value="admin">Admin</option>
                      </select>
                      <button onClick={() => { setEditUser(u); setEditQuota(u.quota || {}); }} className="py-1 px-2.5 text-xs rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500" title="Edit quota">Quota</button>
                      <button onClick={() => handleSuspend(u.user_id, u.suspended)}
                        className={`py-1 px-2.5 text-xs rounded-lg border ${u.suspended ? 'border-green-200 text-green-600 hover:bg-green-50' : 'border-amber-200 text-amber-600 hover:bg-amber-50'}`}>
                        {u.suspended ? 'Unsuspend' : 'Suspend'}
                      </button>
                      <button onClick={() => handleDelete(u.user_id, u.name)}
                        className="p-1.5 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
                      </button>
                    </div>
                  </div>

                  {/* Quota usage bars */}
                  {q && (
                    <div className="grid grid-cols-5 gap-4 mt-4 pt-4 border-t border-gray-100">
                      <div><span className="text-[10px] text-gray-400 uppercase tracking-wider block mb-1">Websites</span>{usageBar(q.used_websites, q.max_websites)}</div>
                      <div><span className="text-[10px] text-gray-400 uppercase tracking-wider block mb-1">Databases</span>{usageBar(q.used_databases, q.max_databases)}</div>
                      <div><span className="text-[10px] text-gray-400 uppercase tracking-wider block mb-1">FTP Users</span>{usageBar(q.used_ftp_users, q.max_ftp_users)}</div>
                      <div><span className="text-[10px] text-gray-400 uppercase tracking-wider block mb-1">Mail</span>{usageBar(q.used_mail_domains, q.max_mail_domains)}</div>
                      <div><span className="text-[10px] text-gray-400 uppercase tracking-wider block mb-1">Disk</span>{usageBar(q.used_disk_mb, q.max_disk_mb)}</div>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* PACKAGES TAB */}
      {tab === 'packages' && (
        <div>
          <div className="flex justify-end mb-4">
            <button onClick={() => setShowPkgForm(true)} className="inline-flex items-center gap-2 py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.5v15m7.5-7.5h-15" /></svg>
              Create Package
            </button>
          </div>

          {packages.length === 0 ? (
            <div className="bg-white border border-gray-200 rounded-xl p-16 text-center"><div className="text-4xl mb-3">📦</div><p className="text-gray-400">No packages defined</p><p className="text-xs text-gray-300 mt-1">Create packages to assign resource limits to users</p></div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {packages.map(pkg => (
                <div key={pkg.id} className="bg-white border border-gray-200 rounded-xl p-5 hover:shadow-md transition-all">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-bold text-gray-800">{pkg.name}</h3>
                    {pkg.price_monthly > 0 && (
                      <span className="text-lg font-bold text-blue-600">${pkg.price_monthly}<span className="text-xs text-gray-400 font-normal">/mo</span></span>
                    )}
                  </div>
                  <div className="space-y-2 text-xs">
                    {[
                      ['Websites', pkg.max_websites],
                      ['Databases', pkg.max_databases],
                      ['FTP Users', pkg.max_ftp_users],
                      ['Mail Domains', pkg.max_mail_domains],
                      ['Disk', pkg.max_disk_mb ? formatDisk(pkg.max_disk_mb) : '∞'],
                      ['Bandwidth', pkg.max_bandwidth_mb ? formatDisk(pkg.max_bandwidth_mb) : '∞'],
                    ].map(([label, val]) => (
                      <div key={label as string} className="flex items-center justify-between text-gray-600">
                        <span>{label}</span>
                        <span className="font-medium text-gray-800">{val === 0 ? '∞' : val}</span>
                      </div>
                    ))}
                  </div>
                  <div className="flex gap-1 mt-4 pt-3 border-t border-gray-100">
                    <button onClick={async () => { await api.delete(`/packages/${pkg.id}`).catch(() => {}); fetchPackages(); }}
                      className="py-1 px-3 text-xs rounded-lg text-red-600 hover:bg-red-50">Delete</button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Create package modal */}
          {showPkgForm && (
            <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setShowPkgForm(false)}>
              <form onSubmit={handleCreatePkg} className="bg-white rounded-xl p-6 w-[500px] space-y-4" onClick={e => e.stopPropagation()}>
                <h3 className="font-semibold">Create Package</h3>
                <div><label className="block text-xs font-medium text-gray-500 mb-1">Name</label><input type="text" value={newPkg.name} onChange={e => setNewPkg(p => ({ ...p, name: e.target.value }))} required placeholder="Basic Hosting" className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
                <div className="grid grid-cols-2 gap-3">
                  {[
                    ['max_websites', 'Max Websites', 10],
                    ['max_databases', 'Max Databases', 10],
                    ['max_ftp_users', 'Max FTP Users', 5],
                    ['max_mail_domains', 'Max Mail Domains', 5],
                    ['max_disk_mb', 'Max Disk (MB)', 5120],
                    ['max_bandwidth_mb', 'Max Bandwidth (MB)', 0],
                    ['price_monthly', 'Price $/month', 0],
                  ].map(([key, label, def]) => (
                    <div key={key as string}>
                      <label className="block text-xs font-medium text-gray-500 mb-1">{label as string} <span className="text-gray-300">(0 = unlimited)</span></label>
                      <input type="number" value={(newPkg as any)[key] ?? def} onChange={e => setNewPkg(p => ({ ...p, [key as string]: Number(e.target.value) }))}
                        className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" />
                    </div>
                  ))}
                </div>
                <div className="flex gap-2 justify-end"><button type="button" onClick={() => setShowPkgForm(false)} className="py-2 px-4 text-sm rounded-lg border hover:bg-gray-50">Cancel</button><button type="submit" disabled={creating} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">Create</button></div>
              </form>
            </div>
          )}
        </div>
      )}

      {/* Edit quota modal */}
      {editUser && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setEditUser(null)}>
          <div className="bg-white rounded-xl p-6 w-[440px] space-y-4" onClick={e => e.stopPropagation()}>
            <h3 className="font-semibold">Edit Quota: {editUser.name}</h3>
            <p className="text-xs text-gray-500">Set to 0 for unlimited.</p>
            <div className="grid grid-cols-2 gap-3">
              {[
                ['max_websites', 'Websites'],
                ['max_databases', 'Databases'],
                ['max_ftp_users', 'FTP Users'],
                ['max_mail_domains', 'Mail Domains'],
                ['max_disk_mb', 'Disk (MB)'],
                ['max_bandwidth_mb', 'Bandwidth (MB)'],
              ].map(([key, label]) => (
                <div key={key}>
                  <label className="block text-xs font-medium text-gray-500 mb-1">{label}</label>
                  <input type="number" value={(editQuota as any)[key] ?? 0} onChange={e => setEditQuota(q => ({ ...q, [key]: Number(e.target.value) }))}
                    className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" />
                </div>
              ))}
            </div>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setEditUser(null)} className="py-2 px-4 text-sm rounded-lg border hover:bg-gray-50">Cancel</button>
              <button onClick={handleSaveQuota} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">Save Quota</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
