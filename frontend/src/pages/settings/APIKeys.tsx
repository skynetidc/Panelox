import { useEffect, useState } from 'react';
import { apiKeysApi, type APIKey, type APIKeyWithSecret } from '../../api/apikeys';
import { Plus, Trash2, Key, Copy, Check } from 'lucide-react';

export default function APIKeys() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [newKey, setNewKey] = useState({ name: '', permissions: 'full', expires_in: 0 });
  const [createdKey, setCreatedKey] = useState<APIKeyWithSecret | null>(null);
  const [copied, setCopied] = useState(false);

  const fetch = () => { apiKeysApi.list().then(r => setKeys(r.data.data || [])).catch(() => {}); };
  useEffect(() => { fetch(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await apiKeysApi.create(newKey);
      setCreatedKey(res.data.data);
      setNewKey({ name: '', permissions: 'full', expires_in: 0 });
      setShowCreate(false);
      fetch();
    } catch {}
  };

  const handleCopy = (key: string) => {
    navigator.clipboard.writeText(key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this API key? Any applications using it will stop working.')) return;
    await apiKeysApi.delete(id).catch(() => {});
    fetch();
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
          <p className="text-sm text-gray-500 mt-1">Generate keys to access Panelox API from external applications</p>
        </div>
        <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 text-sm">
          <Plus size={16} /> Generate Key
        </button>
      </div>

      {/* Created key banner - show only once */}
      {createdKey && (
        <div className="bg-green-50 border border-green-200 rounded-xl p-4 mb-6">
          <p className="text-sm font-semibold text-green-800 mb-2">API Key created! Copy it now - it won't be shown again.</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-white border border-green-300 rounded-lg px-4 py-2 font-mono text-sm select-all">{createdKey.key}</code>
            <button onClick={() => handleCopy(createdKey.key)} className="flex items-center gap-1 px-3 py-2 bg-green-600 text-white rounded-lg text-sm hover:bg-green-700">
              {copied ? <><Check size={14} /> Copied</> : <><Copy size={14} /> Copy</>}
            </button>
          </div>
          <p className="text-xs text-green-600 mt-2">
            Use as: <code className="bg-white px-1 rounded">Authorization: Bearer {createdKey.key.slice(0, 15)}...</code>
          </p>
          <button onClick={() => setCreatedKey(null)} className="text-xs text-green-700 underline mt-2">Dismiss</button>
        </div>
      )}

      {/* Create form */}
      {showCreate && (
        <form onSubmit={handleCreate} className="bg-white rounded-xl shadow-sm border p-6 mb-6 space-y-4 max-w-lg">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Key Name *</label>
            <input type="text" value={newKey.name} onChange={e => setNewKey(k => ({ ...k, name: e.target.value }))} required placeholder="My App, CI/CD, Monitoring..." className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Permissions</label>
              <select value={newKey.permissions} onChange={e => setNewKey(k => ({ ...k, permissions: e.target.value }))} className="w-full px-3 py-2 border rounded-lg text-sm">
                <option value="full">Full Access (read + write)</option>
                <option value="readonly">Read Only</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Expires In</label>
              <select value={newKey.expires_in} onChange={e => setNewKey(k => ({ ...k, expires_in: Number(e.target.value) }))} className="w-full px-3 py-2 border rounded-lg text-sm">
                <option value={0}>Never</option>
                <option value={30}>30 days</option>
                <option value={90}>90 days</option>
                <option value={365}>1 year</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2">
            <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700">Generate</button>
            <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 border rounded-lg text-sm">Cancel</button>
          </div>
        </form>
      )}

      {/* Usage example */}
      <div className="bg-gray-50 border rounded-xl p-4 mb-6">
        <p className="text-sm font-semibold text-gray-700 mb-2">Usage</p>
        <pre className="text-xs bg-gray-900 text-green-400 p-3 rounded-lg overflow-x-auto">{`# Use API key instead of JWT login
curl https://panelox.devority.cz/api/v1/websites \\
  -H "Authorization: Bearer pnx_your_key_here"

# Works with all API endpoints
curl https://panelox.devority.cz/api/v1/dashboard/overview \\
  -H "Authorization: Bearer pnx_your_key_here"`}</pre>
      </div>

      {/* Keys list */}
      <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
        {keys.length === 0 ? (
          <div className="p-12 text-center">
            <Key size={48} className="mx-auto mb-4 text-gray-300" />
            <p className="text-gray-400">No API keys yet</p>
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b text-left text-xs text-gray-500 uppercase">
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Key</th>
                <th className="px-4 py-3">Permissions</th>
                <th className="px-4 py-3">Expires</th>
                <th className="px-4 py-3">Last Used</th>
                <th className="px-4 py-3">Created</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {keys.map(k => (
                <tr key={k.id} className="border-b border-gray-50 hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium text-sm">{k.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-500">{k.key_prefix}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${k.permissions === 'full' ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-700'}`}>
                      {k.permissions}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-xs text-gray-500">
                    {k.expires_at ? new Date(k.expires_at).toLocaleDateString() : 'Never'}
                  </td>
                  <td className="px-4 py-3 text-xs text-gray-500">
                    {k.last_used ? new Date(k.last_used).toLocaleString() : 'Never'}
                  </td>
                  <td className="px-4 py-3 text-xs text-gray-500">
                    {new Date(k.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3">
                    <button onClick={() => handleDelete(k.id)} className="p-1 text-gray-400 hover:text-red-600">
                      <Trash2 size={14} />
                    </button>
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
