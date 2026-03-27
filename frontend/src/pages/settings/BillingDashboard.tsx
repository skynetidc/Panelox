import { useEffect, useState } from 'react';
import { quotaApi, type UserBilling, type Package } from '../../api/quota';
import { Plus, Trash2, Ban, CheckCircle, Package as PackageIcon, Users } from 'lucide-react';

type Tab = 'users' | 'packages';

export default function BillingDashboard() {
  const [tab, setTab] = useState<Tab>('users');
  const [users, setUsers] = useState<UserBilling[]>([]);
  const [packages, setPackages] = useState<Package[]>([]);
  const [showCreatePkg, setShowCreatePkg] = useState(false);
  const [newPkg, setNewPkg] = useState<Partial<Package>>({ name: '', max_websites: 10, max_databases: 5, max_ftp_users: 5, max_mail_domains: 5, max_disk_mb: 10240, max_bandwidth_mb: 102400, max_users: 10, price_monthly: 0 });

  const fetchUsers = () => quotaApi.billingDashboard().then(r => setUsers(r.data.data || [])).catch(() => {});
  const fetchPkgs = () => quotaApi.listPackages().then(r => setPackages(r.data.data || [])).catch(() => {});

  useEffect(() => { fetchUsers(); fetchPkgs(); }, []);

  const handleSuspend = async (userId: number) => {
    const reason = prompt('Suspension reason:');
    if (reason === null) return;
    await quotaApi.suspendUser(userId, reason); fetchUsers();
  };

  const handleUnsuspend = async (userId: number) => {
    await quotaApi.unsuspendUser(userId); fetchUsers();
  };

  const handleAssignPkg = async (userId: number) => {
    const pkgId = prompt('Package ID to assign:');
    if (!pkgId) return;
    await quotaApi.assignPackage(Number(pkgId), userId); fetchUsers();
  };

  const usageBar = (used: number, max: number) => {
    if (max === 0) return <span className="text-xs text-gray-400">{used} / ∞</span>;
    const pct = Math.min(100, (used / max) * 100);
    const color = pct > 90 ? 'bg-red-500' : pct > 70 ? 'bg-yellow-500' : 'bg-green-500';
    return (
      <div className="flex items-center gap-2">
        <div className="w-20 h-2 bg-gray-200 rounded-full"><div className={`h-2 rounded-full ${color}`} style={{ width: `${pct}%` }} /></div>
        <span className="text-xs text-gray-500">{used}/{max}</span>
      </div>
    );
  };

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-4">Billing & Quotas</h1>
      <div className="border-b border-gray-200 mb-6">
        <nav className="flex gap-6">
          <button onClick={() => setTab('users')} className={`flex items-center gap-1 pb-3 text-sm font-medium border-b-2 ${tab === 'users' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'}`}><Users size={14} /> Users & Quotas</button>
          <button onClick={() => setTab('packages')} className={`flex items-center gap-1 pb-3 text-sm font-medium border-b-2 ${tab === 'packages' ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'}`}><PackageIcon size={14} /> Packages</button>
        </nav>
      </div>

      {/* USERS TAB */}
      {tab === 'users' && (
        <div className="bg-white rounded-xl shadow-sm border overflow-x-auto">
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase">
              <th className="px-4 py-3">User</th><th className="px-4 py-3">Role</th><th className="px-4 py-3">Package</th>
              <th className="px-4 py-3">Websites</th><th className="px-4 py-3">Databases</th><th className="px-4 py-3">Disk</th>
              <th className="px-4 py-3">Status</th><th className="px-4 py-3">Actions</th>
            </tr></thead>
            <tbody>
              {users.map(u => (
                <tr key={u.user_id} className={`border-b border-gray-50 ${u.suspended ? 'bg-red-50' : 'hover:bg-gray-50'}`}>
                  <td className="px-4 py-3"><p className="font-medium text-sm">{u.name}</p><p className="text-xs text-gray-400">{u.email}</p></td>
                  <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${u.role === 'admin' ? 'bg-red-100 text-red-700' : u.role === 'reseller' ? 'bg-purple-100 text-purple-700' : 'bg-blue-100 text-blue-700'}`}>{u.role}</span></td>
                  <td className="px-4 py-3">
                    <button onClick={() => handleAssignPkg(u.user_id)} className="text-xs text-blue-600 hover:underline">{u.package || 'None (assign)'}</button>
                  </td>
                  <td className="px-4 py-3">{u.quota ? usageBar(u.quota.used_websites, u.quota.max_websites) : '—'}</td>
                  <td className="px-4 py-3">{u.quota ? usageBar(u.quota.used_databases, u.quota.max_databases) : '—'}</td>
                  <td className="px-4 py-3">{u.quota ? usageBar(u.quota.used_disk_mb, u.quota.max_disk_mb) : '—'}</td>
                  <td className="px-4 py-3">
                    {u.suspended
                      ? <span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs">Suspended</span>
                      : <span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">{u.status}</span>}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1">
                      {u.suspended
                        ? <button onClick={() => handleUnsuspend(u.user_id)} className="flex items-center gap-1 px-2 py-1 text-xs text-green-600 border border-green-300 rounded hover:bg-green-50"><CheckCircle size={12} /> Unsuspend</button>
                        : <button onClick={() => handleSuspend(u.user_id)} className="flex items-center gap-1 px-2 py-1 text-xs text-red-600 border border-red-300 rounded hover:bg-red-50"><Ban size={12} /> Suspend</button>}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* PACKAGES TAB */}
      {tab === 'packages' && (
        <div className="space-y-4">
          <button onClick={() => setShowCreatePkg(!showCreatePkg)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-blue-700"><Plus size={16} /> New Package</button>

          {showCreatePkg && (
            <div className="bg-white rounded-xl shadow-sm border p-6 space-y-4 max-w-2xl">
              <div className="grid grid-cols-3 gap-4">
                <div><label className="text-xs text-gray-500">Name *</label><input type="text" value={newPkg.name} onChange={e => setNewPkg(p => ({ ...p, name: e.target.value }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Price/month</label><input type="number" value={newPkg.price_monthly} onChange={e => setNewPkg(p => ({ ...p, price_monthly: Number(e.target.value) }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Max Users</label><input type="number" value={newPkg.max_users} onChange={e => setNewPkg(p => ({ ...p, max_users: Number(e.target.value) }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Max Websites</label><input type="number" value={newPkg.max_websites} onChange={e => setNewPkg(p => ({ ...p, max_websites: Number(e.target.value) }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Max Databases</label><input type="number" value={newPkg.max_databases} onChange={e => setNewPkg(p => ({ ...p, max_databases: Number(e.target.value) }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Max Disk (MB)</label><input type="number" value={newPkg.max_disk_mb} onChange={e => setNewPkg(p => ({ ...p, max_disk_mb: Number(e.target.value) }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
              </div>
              <div className="flex gap-2">
                <button onClick={() => { quotaApi.createPackage(newPkg).then(() => { setShowCreatePkg(false); fetchPkgs(); }); }} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Create</button>
                <button onClick={() => setShowCreatePkg(false)} className="px-4 py-2 border rounded-lg text-sm">Cancel</button>
              </div>
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {packages.map(p => (
              <div key={p.id} className="bg-white rounded-xl shadow-sm border p-5">
                <div className="flex items-center justify-between mb-3">
                  <h3 className="font-bold">{p.name}</h3>
                  <span className="text-lg font-bold text-blue-600">${p.price_monthly}/mo</span>
                </div>
                <div className="space-y-1 text-sm text-gray-600">
                  <p>{p.max_websites} websites</p>
                  <p>{p.max_databases} databases</p>
                  <p>{p.max_ftp_users} FTP users</p>
                  <p>{p.max_mail_domains} mail domains</p>
                  <p>{(p.max_disk_mb / 1024).toFixed(1)} GB disk</p>
                  <p>{(p.max_bandwidth_mb / 1024).toFixed(0)} GB bandwidth</p>
                  <p>{p.max_users} users</p>
                </div>
                <div className="flex gap-2 mt-4">
                  <button onClick={() => { quotaApi.deletePackage(p.id); fetchPkgs(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                </div>
              </div>
            ))}
            {packages.length === 0 && <div className="col-span-3 bg-white rounded-xl p-12 text-center text-gray-400">No packages created</div>}
          </div>
        </div>
      )}
    </div>
  );
}
