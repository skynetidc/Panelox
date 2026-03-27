import { useEffect, useState } from 'react';
import { nodeApi, type Node } from '../../api/nodes';
import api from '../../api/client';

export default function Nodes() {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [editNode, setEditNode] = useState<Node | null>(null);
  const [newNode, setNewNode] = useState({ name: '', host: '', port: 22, ssh_user: 'root', auth_mode: 'password', password: '', private_key: '', agent_port: 9090 });
  const [creating, setCreating] = useState(false);
  const [testing, setTesting] = useState<number | null>(null);
  const [installing, setInstalling] = useState<number | null>(null);
  const [message, setMessage] = useState('');

  const fetchNodes = async () => { try { const r = await nodeApi.list(); setNodes(r.data.data || []); } catch {} };
  useEffect(() => { fetchNodes(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault(); setCreating(true);
    try { await nodeApi.create(newNode); setNewNode({ name: '', host: '', port: 22, ssh_user: 'root', auth_mode: 'password', password: '', private_key: '', agent_port: 9090 }); setShowCreate(false); }
    catch (err: any) { setMessage(err.response?.data?.error?.message || 'Failed'); }
    setCreating(false); fetchNodes();
  };

  const handleTest = async (id: number) => {
    setTesting(id); setMessage('');
    try { const r = await nodeApi.test(id); setMessage(`${r.data.data.status}: ${r.data.data.message}`); }
    catch { setMessage('Connection failed'); }
    setTesting(null); fetchNodes();
  };

  const handleInstall = async (id: number) => {
    setInstalling(id); setMessage('');
    try { const r = await nodeApi.installAgent(id); setMessage(r.data.data.message); }
    catch (e: any) { setMessage(e.response?.data?.error?.message || 'Install failed'); }
    setInstalling(null); fetchNodes();
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Remove node "${name}"? This cannot be undone.`)) return;
    try { await nodeApi.delete(id); }
    catch (e: any) { setMessage(e.response?.data?.error?.message || 'Delete failed'); }
    fetchNodes();
  };

  const handleEdit = async () => {
    if (!editNode) return;
    try {
      await api.put(`/nodes/${editNode.id}/group`, { group: editNode.node_group || 'default', region: editNode.region || '' });
      setEditNode(null); fetchNodes();
    } catch (e: any) { setMessage(e.response?.data?.error?.message || 'Update failed'); }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-800">Server Nodes</h1>
        <div className="flex gap-2">
          <button onClick={() => nodeApi.pingAll().then(fetchNodes)} className="py-2 px-4 text-sm font-medium rounded-lg border border-gray-200 hover:bg-gray-50">Ping All</button>
          <button onClick={() => setShowCreate(!showCreate)} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">Add Node</button>
        </div>
      </div>

      {message && (
        <div className="bg-blue-50 border border-blue-200 text-blue-700 text-sm px-4 py-3 rounded-lg mb-4 flex justify-between">
          <span>{message}</span>
          <button onClick={() => setMessage('')} className="text-blue-500 hover:underline text-xs">Dismiss</button>
        </div>
      )}

      {/* Create form */}
      {showCreate && (
        <form onSubmit={handleCreate} className="bg-white border border-gray-200 rounded-xl p-6 mb-4 space-y-4 max-w-2xl">
          <h3 className="font-semibold">Add Remote Server</h3>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="block text-sm font-medium text-gray-700 mb-1">Name *</label><input type="text" value={newNode.name} onChange={e => setNewNode(n => ({ ...n, name: e.target.value }))} required placeholder="vps-amsterdam" className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
            <div><label className="block text-sm font-medium text-gray-700 mb-1">Host (IP) *</label><input type="text" value={newNode.host} onChange={e => setNewNode(n => ({ ...n, host: e.target.value }))} required placeholder="185.123.45.67" className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
            <div><label className="block text-sm font-medium text-gray-700 mb-1">SSH Port</label><input type="number" value={newNode.port} onChange={e => setNewNode(n => ({ ...n, port: Number(e.target.value) }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
            <div><label className="block text-sm font-medium text-gray-700 mb-1">SSH User</label><input type="text" value={newNode.ssh_user} onChange={e => setNewNode(n => ({ ...n, ssh_user: e.target.value }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
            <div><label className="block text-sm font-medium text-gray-700 mb-1">Auth</label><select value={newNode.auth_mode} onChange={e => setNewNode(n => ({ ...n, auth_mode: e.target.value }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm"><option value="password">Password</option><option value="key">SSH Key</option></select></div>
            <div><label className="block text-sm font-medium text-gray-700 mb-1">Agent Port</label><input type="number" value={newNode.agent_port} onChange={e => setNewNode(n => ({ ...n, agent_port: Number(e.target.value) }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
          </div>
          {newNode.auth_mode === 'password'
            ? <div><label className="block text-sm font-medium text-gray-700 mb-1">Password</label><input type="password" value={newNode.password} onChange={e => setNewNode(n => ({ ...n, password: e.target.value }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
            : <div><label className="block text-sm font-medium text-gray-700 mb-1">Private Key</label><textarea value={newNode.private_key} onChange={e => setNewNode(n => ({ ...n, private_key: e.target.value }))} rows={4} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm font-mono" /></div>
          }
          <div className="flex gap-2">
            <button type="submit" disabled={creating} className="py-2 px-4 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">{creating ? 'Adding...' : 'Add Node'}</button>
            <button type="button" onClick={() => setShowCreate(false)} className="py-2 px-4 text-sm rounded-lg border border-gray-200 hover:bg-gray-50">Cancel</button>
          </div>
        </form>
      )}

      {/* Edit modal */}
      {editNode && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setEditNode(null)}>
          <div className="bg-white rounded-xl p-6 w-[400px] space-y-4" onClick={e => e.stopPropagation()}>
            <h3 className="font-semibold">Edit Node: {editNode.name}</h3>
            <div><label className="text-sm text-gray-600">Group</label><input type="text" value={editNode.node_group || ''} onChange={e => setEditNode({ ...editNode, node_group: e.target.value })} placeholder="default" className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm mt-1" /></div>
            <div><label className="text-sm text-gray-600">Region</label><input type="text" value={editNode.region || ''} onChange={e => setEditNode({ ...editNode, region: e.target.value })} placeholder="EU, US, Asia..." className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm mt-1" /></div>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setEditNode(null)} className="py-2 px-4 text-sm rounded-lg border border-gray-200 hover:bg-gray-50">Cancel</button>
              <button onClick={handleEdit} className="py-2 px-4 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700">Save</button>
            </div>
          </div>
        </div>
      )}

      {/* Node list */}
      <div className="space-y-3">
        {nodes.length === 0 ? (
          <div className="bg-white border border-gray-200 rounded-xl p-12 text-center text-gray-400">No nodes configured</div>
        ) : nodes.map(n => (
          <div key={n.id} className="bg-white border border-gray-200 rounded-xl px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className={`w-3 h-3 rounded-full ${n.status === 'online' ? 'bg-green-500' : 'bg-red-400'}`} />
              <div>
                <div className="flex items-center gap-2">
                  <p className="font-semibold text-sm">{n.name}</p>
                  {n.is_local && <span className="py-0.5 px-2 text-[10px] font-medium rounded-full bg-blue-100 text-blue-700">local</span>}
                </div>
                <p className="text-xs text-gray-500">{n.ssh_user}@{n.host}:{n.port} · Agent :{n.agent_port}</p>
                {n.last_ping && <p className="text-[10px] text-gray-400">Last ping: {new Date(n.last_ping).toLocaleString()}</p>}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span className={`py-1 px-2 text-xs font-medium rounded-full ${n.status === 'online' ? 'bg-green-100 text-green-700' : n.status === 'offline' ? 'bg-red-100 text-red-700' : 'bg-yellow-100 text-yellow-700'}`}>{n.status}</span>
              <button onClick={() => handleTest(n.id)} disabled={testing === n.id} className="py-1 px-3 text-xs font-medium rounded-lg border border-gray-200 hover:bg-gray-50 disabled:opacity-50">{testing === n.id ? '...' : 'Test'}</button>
              <button onClick={() => setEditNode(n)} className="py-1 px-3 text-xs font-medium rounded-lg border border-gray-200 hover:bg-gray-50">Edit</button>
              {!n.is_local && <button onClick={() => handleInstall(n.id)} disabled={installing === n.id} className="py-1 px-3 text-xs font-medium rounded-lg border border-gray-200 hover:bg-gray-50 disabled:opacity-50">{installing === n.id ? '...' : 'Install'}</button>}
              {!n.is_local && <button onClick={() => handleDelete(n.id, n.name)} className="py-1 px-3 text-xs font-medium rounded-lg border border-red-200 text-red-600 hover:bg-red-50">Delete</button>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
