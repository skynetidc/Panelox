import { useEffect, useState } from 'react';
import api from '../../api/client';
import { filesApi, type FileEntry } from '../../api/files';
import { Plus, Trash2, ToggleLeft, ToggleRight, HardDrive, Folder, ChevronRight, X } from 'lucide-react';

interface FTPUser { id: number; website_id?: number; username: string; home_dir: string; status: string; created_at: string; }

export default function FTPUsers() {
  const [users, setUsers] = useState<FTPUser[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ username: '', password: '', home_dir: '/www/sites/' });
  const [showBrowser, setShowBrowser] = useState(false);
  const [browserPath, setBrowserPath] = useState('/www/sites');
  const [browserEntries, setBrowserEntries] = useState<FileEntry[]>([]);

  const fetch = () => { api.get<{ data: FTPUser[] }>('/ftp/users').then(r => setUsers(r.data.data || [])).catch(() => {}); };
  useEffect(() => { fetch(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await api.post('/ftp/users', form).catch(() => {});
    setForm({ username: '', password: '', home_dir: '/www/sites/' });
    setShowCreate(false);
    fetch();
  };

  const openBrowser = (path: string) => {
    setBrowserPath(path);
    setShowBrowser(true);
    filesApi.list(path).then(r => {
      const dirs = (r.data.data || []).filter(e => e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
      setBrowserEntries(dirs);
    }).catch(() => setBrowserEntries([]));
  };

  const navigateBrowser = (path: string) => {
    setBrowserPath(path);
    filesApi.list(path).then(r => {
      const dirs = (r.data.data || []).filter(e => e.is_dir).sort((a, b) => a.name.localeCompare(b.name));
      setBrowserEntries(dirs);
    }).catch(() => setBrowserEntries([]));
  };

  const selectDir = () => {
    setForm(f => ({ ...f, home_dir: browserPath }));
    setShowBrowser(false);
  };

  const pathParts = browserPath.split('/').filter(Boolean);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">FTP Accounts</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold text-white" style={{ background: 'var(--accent-gradient)' }}>
          <Plus size={16} /> New FTP User
        </button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="bg-white rounded-xl border p-6 mb-6 space-y-4 max-w-lg" style={{ borderColor: 'var(--border)' }}>
          <div>
            <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--text-secondary)' }}>Username *</label>
            <input type="text" value={form.username} onChange={e => setForm(f => ({ ...f, username: e.target.value }))} required placeholder="ftpuser" className="w-full px-3 py-2 text-sm" />
          </div>
          <div>
            <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--text-secondary)' }}>Password *</label>
            <input type="password" value={form.password} onChange={e => setForm(f => ({ ...f, password: e.target.value }))} required className="w-full px-3 py-2 text-sm" />
          </div>
          <div>
            <label className="block text-xs font-semibold mb-1" style={{ color: 'var(--text-secondary)' }}>Home Directory</label>
            <div className="flex gap-2">
              <input type="text" value={form.home_dir} onChange={e => setForm(f => ({ ...f, home_dir: e.target.value }))} placeholder="/www/sites/mysite/index" className="flex-1 px-3 py-2 text-sm" />
              <button type="button" onClick={() => openBrowser(form.home_dir || '/www/sites')} className="flex items-center gap-1 px-3 py-2 rounded-xl text-sm border" style={{ borderColor: 'var(--border)', color: 'var(--text-secondary)' }}>
                <Folder size={14} /> Browse
              </button>
            </div>
          </div>
          <div className="flex gap-2">
            <button type="submit" className="px-4 py-2 rounded-xl text-sm font-semibold text-white" style={{ background: 'var(--accent-gradient)' }}>Create</button>
            <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-xl text-sm border" style={{ borderColor: 'var(--border)' }}>Cancel</button>
          </div>
        </form>
      )}

      {/* Directory Browser Modal */}
      {showBrowser && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setShowBrowser(false)}>
          <div className="bg-white rounded-2xl shadow-xl w-[500px] max-h-[500px] flex flex-col" onClick={e => e.stopPropagation()} style={{ border: '1px solid var(--border)' }}>
            <div className="flex items-center justify-between px-4 py-3" style={{ borderBottom: '1px solid var(--border)' }}>
              <h3 className="font-semibold text-sm">Select Directory</h3>
              <button onClick={() => setShowBrowser(false)} style={{ color: 'var(--text-muted)' }}><X size={18} /></button>
            </div>

            {/* Breadcrumbs */}
            <div className="flex items-center gap-1 px-4 py-2 text-xs overflow-x-auto" style={{ background: '#faf9f6', borderBottom: '1px solid var(--border)' }}>
              <button onClick={() => navigateBrowser('/')} className="hover:underline" style={{ color: 'var(--accent)' }}>/</button>
              {pathParts.map((part, i) => (
                <span key={i} className="flex items-center gap-1">
                  <ChevronRight size={10} style={{ color: 'var(--text-muted)' }} />
                  <button onClick={() => navigateBrowser('/' + pathParts.slice(0, i + 1).join('/'))} className="hover:underline" style={{ color: 'var(--accent)' }}>{part}</button>
                </span>
              ))}
            </div>

            {/* Directory list */}
            <div className="flex-1 overflow-y-auto p-2">
              {/* Go up */}
              {browserPath !== '/' && (
                <button onClick={() => navigateBrowser(browserPath.split('/').slice(0, -1).join('/') || '/')} className="flex items-center gap-2 w-full px-3 py-2 rounded-lg text-sm hover:bg-gray-50">
                  <Folder size={16} style={{ color: 'var(--text-muted)' }} />
                  <span style={{ color: 'var(--text-muted)' }}>..</span>
                </button>
              )}
              {browserEntries.map(entry => (
                <button key={entry.path} onClick={() => navigateBrowser(entry.path)} className="flex items-center gap-2 w-full px-3 py-2 rounded-lg text-sm hover:bg-gray-50 text-left">
                  <Folder size={16} style={{ color: '#f59e0b' }} />
                  <span>{entry.name}</span>
                </button>
              ))}
              {browserEntries.length === 0 && browserPath !== '/' && (
                <p className="text-center py-8 text-sm" style={{ color: 'var(--text-muted)' }}>Empty directory</p>
              )}
            </div>

            {/* Footer */}
            <div className="flex items-center justify-between px-4 py-3" style={{ borderTop: '1px solid var(--border)' }}>
              <span className="text-xs font-mono" style={{ color: 'var(--text-muted)' }}>{browserPath}</span>
              <button onClick={selectDir} className="px-4 py-1.5 rounded-lg text-sm font-semibold text-white" style={{ background: 'var(--accent-gradient)' }}>
                Select This Directory
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Users list */}
      <div className="bg-white rounded-xl border overflow-hidden" style={{ borderColor: 'var(--border)' }}>
        {users.length === 0 ? (
          <div className="p-12 text-center">
            <HardDrive size={48} className="mx-auto mb-4" style={{ color: 'var(--text-muted)' }} />
            <p style={{ color: 'var(--text-muted)' }}>No FTP accounts</p>
          </div>
        ) : (
          <table className="w-full">
            <thead><tr><th>Username</th><th>Home Dir</th><th>Status</th><th>Created</th><th></th></tr></thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id}>
                  <td className="font-mono font-medium">{u.username}</td>
                  <td className="font-mono text-xs">{u.home_dir}</td>
                  <td><span className={`px-2 py-0.5 rounded-full text-xs font-medium ${u.status === 'active' ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}`}>{u.status}</span></td>
                  <td className="text-xs">{new Date(u.created_at).toLocaleDateString()}</td>
                  <td>
                    <div className="flex gap-1">
                      <button onClick={() => { api.post(`/ftp/users/${u.id}/toggle`); fetch(); }} className="p-1" style={{ color: 'var(--text-muted)' }}>{u.status === 'active' ? <ToggleRight size={16} className="text-green-500" /> : <ToggleLeft size={16} />}</button>
                      <button onClick={() => { if (confirm(`Delete FTP user ${u.username}?`)) { api.delete(`/ftp/users/${u.id}`); fetch(); } }} className="p-1" style={{ color: 'var(--text-muted)' }}><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
