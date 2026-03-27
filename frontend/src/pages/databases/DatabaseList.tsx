import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { databaseApi, type DBServer, type Database as DB, type DBUser } from '../../api/databases';
import {
  Database, Plus, Trash2, Play, Square, RotateCw, Users, ChevronDown, ChevronRight,
} from 'lucide-react';

export default function DatabaseList() {
  const { t } = useTranslation();
  const [servers, setServers] = useState<DBServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedServer, setExpandedServer] = useState<number | null>(null);

  // Create server form
  const [showCreate, setShowCreate] = useState(false);
  const [newServer, setNewServer] = useState({ name: '', type: 'mysql', root_password: '', port: 0 });
  const [creating, setCreating] = useState(false);

  // Sub-resources
  const [databases, setDatabases] = useState<DB[]>([]);
  const [dbUsers, setDbUsers] = useState<DBUser[]>([]);
  const [newDbName, setNewDbName] = useState('');
  const [newUser, setNewUser] = useState({ username: '', password: '' });

  const fetchServers = async () => {
    setLoading(true);
    try {
      const res = await databaseApi.listServers({ page: 1, size: 50 });
      setServers(res.data.data || []);
    } catch { setServers([]); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchServers(); }, []);

  const toggleServer = async (id: number) => {
    if (expandedServer === id) {
      setExpandedServer(null);
      return;
    }
    setExpandedServer(id);
    try {
      const [dbRes, userRes] = await Promise.all([
        databaseApi.listDatabases(id),
        databaseApi.listDBUsers(id),
      ]);
      setDatabases(dbRes.data.data || []);
      setDbUsers(userRes.data.data || []);
    } catch { /* ignore */ }
  };

  const handleCreateServer = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);
    try {
      await databaseApi.createServer(newServer);
      setShowCreate(false);
      setNewServer({ name: '', type: 'mysql', root_password: '', port: 0 });
      fetchServers();
    } catch { /* ignore */ }
    finally { setCreating(false); }
  };

  const handleOperate = async (id: number, action: 'start' | 'stop' | 'restart') => {
    await databaseApi.operateServer(id, action).catch(() => {});
    fetchServers();
  };

  const handleDeleteServer = async (id: number, name: string) => {
    if (!confirm(`Delete database server "${name}"? This will stop and remove the container.`)) return;
    await databaseApi.deleteServer(id).catch(() => {});
    fetchServers();
  };

  const handleCreateDb = async (serverId: number) => {
    if (!newDbName) return;
    await databaseApi.createDatabase(serverId, newDbName).catch(() => {});
    setNewDbName('');
    const res = await databaseApi.listDatabases(serverId);
    setDatabases(res.data.data || []);
  };

  const handleDeleteDb = async (id: number, serverId: number) => {
    if (!confirm('Delete this database?')) return;
    await databaseApi.deleteDatabase(id).catch(() => {});
    const res = await databaseApi.listDatabases(serverId);
    setDatabases(res.data.data || []);
  };

  const handleCreateUser = async (serverId: number) => {
    if (!newUser.username || !newUser.password) return;
    await databaseApi.createDBUser(serverId, newUser.username, newUser.password, databases.map(d => d.name)).catch(() => {});
    setNewUser({ username: '', password: '' });
    const res = await databaseApi.listDBUsers(serverId);
    setDbUsers(res.data.data || []);
  };

  const handleDeleteUser = async (id: number, serverId: number) => {
    await databaseApi.deleteDBUser(id).catch(() => {});
    const res = await databaseApi.listDBUsers(serverId);
    setDbUsers(res.data.data || []);
  };

  const statusColors: Record<string, string> = {
    running: 'bg-green-100 text-green-700',
    stopped: 'bg-gray-100 text-gray-600',
    creating: 'bg-yellow-100 text-yellow-700',
    error: 'bg-red-100 text-red-700',
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{t('nav.databases')}</h1>
        <button
          onClick={() => setShowCreate(!showCreate)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
        >
          <Plus size={18} /> New Server
        </button>
      </div>

      {/* Create server form */}
      {showCreate && (
        <form onSubmit={handleCreateServer} className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 mb-6 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
              <input type="text" value={newServer.name} onChange={e => setNewServer(s => ({ ...s, name: e.target.value }))} required placeholder="my-database" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Type *</label>
              <select value={newServer.type} onChange={e => setNewServer(s => ({ ...s, type: e.target.value }))} className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500">
                <option value="mysql">MySQL</option>
                <option value="postgresql">PostgreSQL</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Root Password *</label>
              <input type="password" value={newServer.root_password} onChange={e => setNewServer(s => ({ ...s, root_password: e.target.value }))} required className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Port</label>
              <input type="number" value={newServer.port || ''} onChange={e => setNewServer(s => ({ ...s, port: Number(e.target.value) }))} placeholder={newServer.type === 'mysql' ? '3306' : '5432'} className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 text-sm border rounded-lg">{t('common.cancel')}</button>
            <button type="submit" disabled={creating} className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg disabled:opacity-50">
              {creating ? 'Creating...' : 'Create Server'}
            </button>
          </div>
        </form>
      )}

      {/* Server list */}
      <div className="space-y-3">
        {loading ? (
          <div className="bg-white rounded-xl p-12 text-center text-gray-400">{t('common.loading')}</div>
        ) : servers.length === 0 ? (
          <div className="bg-white rounded-xl p-12 text-center">
            <Database size={48} className="mx-auto mb-4 text-gray-300" />
            <p className="text-gray-400">No database servers yet</p>
          </div>
        ) : servers.map(srv => (
          <div key={srv.id} className="bg-white rounded-xl shadow-sm border border-gray-100">
            {/* Server header */}
            <div className="flex items-center justify-between px-6 py-4 cursor-pointer" onClick={() => toggleServer(srv.id)}>
              <div className="flex items-center gap-3">
                {expandedServer === srv.id ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                <Database size={20} className={srv.type === 'mysql' ? 'text-blue-500' : 'text-cyan-500'} />
                <div>
                  <p className="font-medium">{srv.name}</p>
                  <p className="text-xs text-gray-500">{srv.type} {srv.version} &middot; {srv.host}:{srv.port}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${statusColors[srv.status] || ''}`}>
                  {srv.status}
                </span>
                <div className="flex gap-1" onClick={e => e.stopPropagation()}>
                  {srv.status === 'running' ? (
                    <button onClick={() => handleOperate(srv.id, 'stop')} className="p-1 text-gray-400 hover:text-orange-600"><Square size={14} /></button>
                  ) : srv.status === 'stopped' ? (
                    <button onClick={() => handleOperate(srv.id, 'start')} className="p-1 text-gray-400 hover:text-green-600"><Play size={14} /></button>
                  ) : null}
                  {srv.status === 'running' && (
                    <button onClick={() => handleOperate(srv.id, 'restart')} className="p-1 text-gray-400 hover:text-blue-600"><RotateCw size={14} /></button>
                  )}
                  <button onClick={() => handleDeleteServer(srv.id, srv.name)} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                </div>
              </div>
            </div>

            {/* Expanded: databases + users */}
            {expandedServer === srv.id && (
              <div className="border-t border-gray-100 px-6 py-4 grid grid-cols-2 gap-6">
                {/* Databases */}
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 mb-2 flex items-center gap-1"><Database size={14} /> Databases</h3>
                  <div className="space-y-1 mb-2">
                    {databases.map(db => (
                      <div key={db.id} className="flex items-center justify-between text-sm bg-gray-50 px-3 py-1.5 rounded">
                        <span className="font-mono">{db.name}</span>
                        <button onClick={() => handleDeleteDb(db.id, srv.id)} className="text-gray-400 hover:text-red-600"><Trash2 size={12} /></button>
                      </div>
                    ))}
                    {databases.length === 0 && <p className="text-xs text-gray-400">No databases</p>}
                  </div>
                  <div className="flex gap-2">
                    <input type="text" value={newDbName} onChange={e => setNewDbName(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleCreateDb(srv.id)} placeholder="database_name" className="flex-1 px-2 py-1 border rounded text-sm outline-none" />
                    <button onClick={() => handleCreateDb(srv.id)} className="px-3 py-1 bg-blue-600 text-white rounded text-xs">Add</button>
                  </div>
                </div>

                {/* Users */}
                <div>
                  <h3 className="text-sm font-semibold text-gray-700 mb-2 flex items-center gap-1"><Users size={14} /> Users</h3>
                  <div className="space-y-1 mb-2">
                    {dbUsers.map(u => (
                      <div key={u.id} className="flex items-center justify-between text-sm bg-gray-50 px-3 py-1.5 rounded">
                        <span className="font-mono">{u.username}</span>
                        <button onClick={() => handleDeleteUser(u.id, srv.id)} className="text-gray-400 hover:text-red-600"><Trash2 size={12} /></button>
                      </div>
                    ))}
                    {dbUsers.length === 0 && <p className="text-xs text-gray-400">No users</p>}
                  </div>
                  <div className="flex gap-2">
                    <input type="text" value={newUser.username} onChange={e => setNewUser(u => ({ ...u, username: e.target.value }))} placeholder="username" className="flex-1 px-2 py-1 border rounded text-sm outline-none" />
                    <input type="password" value={newUser.password} onChange={e => setNewUser(u => ({ ...u, password: e.target.value }))} placeholder="password" className="flex-1 px-2 py-1 border rounded text-sm outline-none" />
                    <button onClick={() => handleCreateUser(srv.id)} className="px-3 py-1 bg-blue-600 text-white rounded text-xs">Add</button>
                  </div>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
