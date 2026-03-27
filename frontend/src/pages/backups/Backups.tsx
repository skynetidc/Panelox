import { useEffect, useState } from 'react';
import { backupApi, type Backup } from '../../api/backups';
import { backupAdvApi, type BackupAccount, type BackupSchedule } from '../../api/backup-advanced';
import { Plus, Trash2, Download, RotateCw, Archive, ToggleLeft, ToggleRight, Play, Send } from 'lucide-react';

type Tab = 'backups' | 'accounts' | 'schedules';

export default function Backups() {
  const [tab, setTab] = useState<Tab>('backups');
  const tabs: { key: Tab; label: string }[] = [
    { key: 'backups', label: 'Backups' },
    { key: 'accounts', label: 'Storage Accounts' },
    { key: 'schedules', label: 'Scheduled Backups' },
  ];
  const [backups, setBackups] = useState<Backup[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newBackup, setNewBackup] = useState({ type: 'website', name: '' });
  const [creating, setCreating] = useState(false);

  // Accounts + Schedules state
  const [accounts, setAccounts] = useState<BackupAccount[]>([]);
  const [schedules, setSchedules] = useState<BackupSchedule[]>([]);
  const [newAccount, setNewAccount] = useState({ name: '', type: 'local', config: {} as any });
  const [newSchedule, setNewSchedule] = useState({ name: '', type: 'website', schedule: '@daily', retention_days: 30, retention_count: 10, account_id: undefined as number | undefined });

  const fetch = async () => {
    setLoading(true);
    try { const r = await backupApi.list({ page: 1, size: 50 }); setBackups(r.data.data || []); } catch {}
    finally { setLoading(false); }
  };
  useEffect(() => { fetch(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    await backupApi.create(newBackup.type, newBackup.name || undefined).catch(() => {});
    setNewBackup({ type: 'website', name: '' }); setShowCreate(false); setCreating(false); fetch();
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '—';
    const units = ['B', 'KB', 'MB', 'GB'];
    let i = 0, size = bytes;
    while (size >= 1024 && i < units.length - 1) { size /= 1024; i++; }
    return `${size.toFixed(1)} ${units[i]}`;
  };

  const statusColors: Record<string, string> = { completed: 'bg-green-100 text-green-700', running: 'bg-yellow-100 text-yellow-700', failed: 'bg-red-100 text-red-700', pending: 'bg-gray-100 text-gray-600' };

  const fetchAccounts = () => backupAdvApi.listAccounts().then(r => setAccounts(r.data.data || [])).catch(() => {});
  const fetchSchedules = () => backupAdvApi.listSchedules().then(r => setSchedules(r.data.data || [])).catch(() => {});

  useEffect(() => {
    if (tab === 'accounts') fetchAccounts();
    if (tab === 'schedules') { fetchSchedules(); fetchAccounts(); }
  }, [tab]);

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-4">Backups</h1>
      <div className="border-b border-gray-200 mb-6">
        <nav className="flex gap-6">{tabs.map(t => (
          <button key={t.key} onClick={() => setTab(t.key)} className={`pb-3 text-sm font-medium border-b-2 ${tab === t.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'}`}>{t.label}</button>
        ))}</nav>
      </div>

      {tab === 'backups' && (<div>
      <div className="flex justify-end mb-4">
        <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"><Plus size={18} /> New Backup</button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="bg-white rounded-xl shadow-sm border p-4 mb-4 flex gap-3 items-end">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
            <select value={newBackup.type} onChange={e => setNewBackup(b => ({ ...b, type: e.target.value }))} className="px-3 py-2 border rounded-lg text-sm">
              <option value="website">Websites</option><option value="database">Databases</option><option value="full">Full</option>
            </select>
          </div>
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">Name (optional)</label>
            <input type="text" value={newBackup.name} onChange={e => setNewBackup(b => ({ ...b, name: e.target.value }))} placeholder="Auto-generated" className="w-full px-3 py-2 border rounded-lg text-sm outline-none" />
          </div>
          <button type="submit" disabled={creating} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm disabled:opacity-50">{creating ? 'Creating...' : 'Create'}</button>
          <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 border rounded-lg text-sm">Cancel</button>
        </form>
      )}

      <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
        {loading ? <div className="p-12 text-center text-gray-400">Loading...</div> : backups.length === 0 ? (
          <div className="p-12 text-center"><Archive size={48} className="mx-auto mb-4 text-gray-300" /><p className="text-gray-400">No backups yet</p></div>
        ) : (
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Type</th><th className="px-4 py-3">Size</th><th className="px-4 py-3">Status</th><th className="px-4 py-3">Created</th><th className="px-4 py-3">Actions</th></tr></thead>
            <tbody>
              {backups.map(b => (
                <tr key={b.id} className="border-b border-gray-50 hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium text-sm">{b.name}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">{b.type}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">{formatSize(b.file_size)}</td>
                  <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColors[b.status] || ''}`}>{b.status}</span></td>
                  <td className="px-4 py-3 text-xs text-gray-500">{new Date(b.created_at).toLocaleString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1">
                      {b.status === 'completed' && <a href={backupApi.downloadUrl(b.id)} className="p-1 text-gray-400 hover:text-blue-600"><Download size={14} /></a>}
                      {b.status === 'completed' && <button onClick={() => { if (confirm('Restore from this backup?')) backupApi.restore(b.id); }} className="p-1 text-gray-400 hover:text-green-600"><RotateCw size={14} /></button>}
                      <button onClick={() => { backupApi.delete(b.id); fetch(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      </div>)}

      {/* STORAGE ACCOUNTS */}
      {tab === 'accounts' && (
        <div className="space-y-4">
          <div className="bg-white rounded-xl shadow-sm border p-4">
            <h3 className="text-sm font-semibold mb-3">Add Storage Account</h3>
            <div className="flex gap-3 items-end">
              <div><label className="text-xs text-gray-500">Name</label><input type="text" value={newAccount.name} onChange={e => setNewAccount(a => ({ ...a, name: e.target.value }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
              <div><label className="text-xs text-gray-500">Type</label>
                <select value={newAccount.type} onChange={e => setNewAccount(a => ({ ...a, type: e.target.value, config: {} }))} className="px-2 py-1.5 border rounded text-sm">
                  <option value="local">Local</option><option value="s3">S3 (AWS/MinIO/B2)</option><option value="sftp">SFTP</option>
                </select>
              </div>
              {newAccount.type === 's3' && <>
                <div><label className="text-xs text-gray-500">Bucket</label><input type="text" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, bucket: e.target.value } }))} className="w-32 px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Access Key</label><input type="text" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, access_key: e.target.value } }))} className="w-32 px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Secret Key</label><input type="password" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, secret_key: e.target.value } }))} className="w-32 px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Endpoint (MinIO)</label><input type="text" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, endpoint: e.target.value } }))} placeholder="Optional" className="w-40 px-2 py-1.5 border rounded text-sm outline-none" /></div>
              </>}
              {newAccount.type === 'sftp' && <>
                <div><label className="text-xs text-gray-500">Host</label><input type="text" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, host: e.target.value } }))} className="w-32 px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">User</label><input type="text" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, user: e.target.value } }))} className="w-24 px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Password</label><input type="password" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, password: e.target.value } }))} className="w-24 px-2 py-1.5 border rounded text-sm outline-none" /></div>
                <div><label className="text-xs text-gray-500">Path</label><input type="text" onChange={e => setNewAccount(a => ({ ...a, config: { ...a.config, path: e.target.value } }))} placeholder="/backups" className="w-28 px-2 py-1.5 border rounded text-sm outline-none" /></div>
              </>}
              {newAccount.type === 'local' && <div><label className="text-xs text-gray-500">Path</label><input type="text" onChange={e => setNewAccount(a => ({ ...a, config: { path: e.target.value } }))} placeholder="/opt/panelox/backups" className="w-48 px-2 py-1.5 border rounded text-sm outline-none" /></div>}
              <button onClick={() => { backupAdvApi.createAccount(newAccount).then(() => fetchAccounts()); }} className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Add</button>
            </div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Type</th><th className="px-4 py-3">Default</th><th className="px-4 py-3">Actions</th></tr></thead>
              <tbody>{accounts.map(a => (
                <tr key={a.id} className="border-b border-gray-50"><td className="px-4 py-3 font-medium">{a.name}</td>
                  <td className="px-4 py-3"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{a.type.toUpperCase()}</span></td>
                  <td className="px-4 py-3">{a.is_default ? <span className="text-green-600 text-xs">Default</span> : '—'}</td>
                  <td className="px-4 py-3 flex gap-1">
                    <button onClick={() => backupAdvApi.testAccount(a.id).then(r => alert(r.data.data.status === 'ok' ? 'Connection OK!' : r.data.data.message))} className="p-1 text-gray-400 hover:text-blue-600"><Send size={14} /></button>
                    {!a.is_default && <button onClick={() => { backupAdvApi.deleteAccount(a.id); fetchAccounts(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>}
                  </td></tr>
              ))}{accounts.length === 0 && <tr><td colSpan={4} className="px-6 py-8 text-center text-gray-400">No storage accounts</td></tr>}</tbody>
            </table>
          </div>
        </div>
      )}

      {/* SCHEDULED BACKUPS */}
      {tab === 'schedules' && (
        <div className="space-y-4">
          <div className="bg-white rounded-xl shadow-sm border p-4">
            <h3 className="text-sm font-semibold mb-3">Add Schedule</h3>
            <div className="flex gap-3 items-end flex-wrap">
              <div><label className="text-xs text-gray-500">Name</label><input type="text" value={newSchedule.name} onChange={e => setNewSchedule(s => ({ ...s, name: e.target.value }))} placeholder="Daily websites" className="w-40 px-2 py-1.5 border rounded text-sm outline-none" /></div>
              <div><label className="text-xs text-gray-500">Type</label><select value={newSchedule.type} onChange={e => setNewSchedule(s => ({ ...s, type: e.target.value }))} className="px-2 py-1.5 border rounded text-sm"><option value="website">Websites</option><option value="database">Databases</option><option value="full">Full</option></select></div>
              <div><label className="text-xs text-gray-500">Schedule</label><select value={newSchedule.schedule} onChange={e => setNewSchedule(s => ({ ...s, schedule: e.target.value }))} className="px-2 py-1.5 border rounded text-sm"><option value="@hourly">Hourly</option><option value="@daily">Daily</option><option value="@weekly">Weekly</option></select></div>
              <div><label className="text-xs text-gray-500">Storage</label><select value={newSchedule.account_id || ''} onChange={e => setNewSchedule(s => ({ ...s, account_id: e.target.value ? Number(e.target.value) : undefined }))} className="px-2 py-1.5 border rounded text-sm"><option value="">Default (local)</option>{accounts.map(a => <option key={a.id} value={a.id}>{a.name} ({a.type})</option>)}</select></div>
              <div><label className="text-xs text-gray-500">Keep days</label><input type="number" value={newSchedule.retention_days} onChange={e => setNewSchedule(s => ({ ...s, retention_days: Number(e.target.value) }))} className="w-16 px-2 py-1.5 border rounded text-sm outline-none" /></div>
              <div><label className="text-xs text-gray-500">Keep max</label><input type="number" value={newSchedule.retention_count} onChange={e => setNewSchedule(s => ({ ...s, retention_count: Number(e.target.value) }))} className="w-16 px-2 py-1.5 border rounded text-sm outline-none" /></div>
              <button onClick={() => { backupAdvApi.createSchedule(newSchedule).then(() => fetchSchedules()); }} className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Add</button>
            </div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Type</th><th className="px-4 py-3">Schedule</th><th className="px-4 py-3">Retention</th><th className="px-4 py-3">Last Run</th><th className="px-4 py-3">Actions</th></tr></thead>
              <tbody>{schedules.map(s => (
                <tr key={s.id} className="border-b border-gray-50">
                  <td className="px-4 py-3 font-medium text-sm">{s.name}</td>
                  <td className="px-4 py-3 text-sm">{s.type}</td>
                  <td className="px-4 py-3 text-sm font-mono">{s.schedule}</td>
                  <td className="px-4 py-3 text-xs text-gray-500">{s.retention_days}d / max {s.retention_count}</td>
                  <td className="px-4 py-3 text-xs">{s.last_run ? <span className={s.last_status === 'success' ? 'text-green-600' : 'text-red-600'}>{s.last_status} - {new Date(s.last_run).toLocaleString()}</span> : <span className="text-gray-400">Never</span>}</td>
                  <td className="px-4 py-3 flex gap-1">
                    <button onClick={() => { backupAdvApi.toggleSchedule(s.id); fetchSchedules(); }}>{s.enabled ? <ToggleRight size={16} className="text-green-500" /> : <ToggleLeft size={16} />}</button>
                    <button onClick={() => backupAdvApi.runSchedule(s.id)} className="p-1 text-gray-400 hover:text-green-600"><Play size={14} /></button>
                    <button onClick={() => { backupAdvApi.deleteSchedule(s.id); fetchSchedules(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                  </td>
                </tr>
              ))}{schedules.length === 0 && <tr><td colSpan={6} className="px-6 py-8 text-center text-gray-400">No scheduled backups</td></tr>}</tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
