import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { firewallApi, type FirewallRule, type FirewallStatus } from '../../api/firewall';
import { Plus, Trash2, Shield, RefreshCw } from 'lucide-react';

export default function FirewallRules() {
  const { t } = useTranslation();
  const [rules, setRules] = useState<FirewallRule[]>([]);
  const [status, setStatus] = useState<FirewallStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [newRule, setNewRule] = useState({ protocol: 'tcp', dst_port: '', action: 'accept', src_ip: '', description: '' });

  const fetch = async () => {
    setLoading(true);
    try {
      const [rulesRes, statusRes] = await Promise.all([firewallApi.listRules(), firewallApi.status()]);
      setRules(rulesRes.data.data || []);
      setStatus(statusRes.data.data);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  useEffect(() => { fetch(); }, []);

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newRule.dst_port) return;
    await firewallApi.createRule(newRule).catch(() => {});
    setNewRule({ protocol: 'tcp', dst_port: '', action: 'accept', src_ip: '', description: '' });
    setShowAdd(false);
    fetch();
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this firewall rule?')) return;
    await firewallApi.deleteRule(id).catch(() => {});
    fetch();
  };

  const handleReapply = async () => {
    await firewallApi.reapply().catch(() => {});
    fetch();
  };

  const actionColors: { [k: string]: string } = {
    accept: 'bg-green-100 text-green-700',
    drop: 'bg-red-100 text-red-700',
    reject: 'bg-orange-100 text-orange-700',
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{t('nav.firewall')}</h1>
          {status && <p className="text-sm text-gray-500">Backend: {status.backend} &middot; {status.enabled ? 'Active' : 'Inactive'}</p>}
        </div>
        <div className="flex gap-2">
          <button onClick={handleReapply} className="flex items-center gap-1 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50"><RefreshCw size={14} /> Reapply All</button>
          <button onClick={() => setShowAdd(!showAdd)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"><Plus size={18} /> Add Rule</button>
        </div>
      </div>

      {showAdd && (
        <form onSubmit={handleAdd} className="bg-white rounded-xl shadow-sm border p-4 mb-4">
          <div className="grid grid-cols-6 gap-3 items-end">
            <div>
              <label className="block text-xs text-gray-500 mb-1">Protocol</label>
              <select value={newRule.protocol} onChange={e => setNewRule(r => ({ ...r, protocol: e.target.value }))} className="w-full px-2 py-1.5 border rounded text-sm">
                <option value="tcp">TCP</option><option value="udp">UDP</option><option value="icmp">ICMP</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Port *</label>
              <input type="text" value={newRule.dst_port} onChange={e => setNewRule(r => ({ ...r, dst_port: e.target.value }))} placeholder="80 or 8000-9000" className="w-full px-2 py-1.5 border rounded text-sm outline-none" />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Action</label>
              <select value={newRule.action} onChange={e => setNewRule(r => ({ ...r, action: e.target.value }))} className="w-full px-2 py-1.5 border rounded text-sm">
                <option value="accept">Accept</option><option value="drop">Drop</option><option value="reject">Reject</option>
              </select>
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Source IP</label>
              <input type="text" value={newRule.src_ip} onChange={e => setNewRule(r => ({ ...r, src_ip: e.target.value }))} placeholder="Any" className="w-full px-2 py-1.5 border rounded text-sm outline-none" />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Description</label>
              <input type="text" value={newRule.description} onChange={e => setNewRule(r => ({ ...r, description: e.target.value }))} placeholder="Optional" className="w-full px-2 py-1.5 border rounded text-sm outline-none" />
            </div>
            <div className="flex gap-2">
              <button type="submit" className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Add</button>
              <button type="button" onClick={() => setShowAdd(false)} className="px-3 py-1.5 border rounded text-sm">Cancel</button>
            </div>
          </div>
        </form>
      )}

      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        {loading ? (
          <div className="p-12 text-center text-gray-400">{t('common.loading')}</div>
        ) : rules.length === 0 ? (
          <div className="p-12 text-center">
            <Shield size={48} className="mx-auto mb-4 text-gray-300" />
            <p className="text-gray-400">No firewall rules</p>
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-xs text-gray-500 uppercase">
                <th className="px-4 py-3">#</th><th className="px-4 py-3">Protocol</th><th className="px-4 py-3">Port</th>
                <th className="px-4 py-3">Source</th><th className="px-4 py-3">Action</th><th className="px-4 py-3">Description</th><th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {rules.map(r => (
                <tr key={r.id} className="border-b border-gray-50 hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm text-gray-400">{r.position}</td>
                  <td className="px-4 py-3 text-sm font-mono">{r.protocol.toUpperCase()}</td>
                  <td className="px-4 py-3 text-sm font-mono">{r.dst_port}</td>
                  <td className="px-4 py-3 text-sm font-mono text-gray-600">{r.src_ip || 'Any'}</td>
                  <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs font-medium ${actionColors[r.action] || ''}`}>{r.action.toUpperCase()}</span></td>
                  <td className="px-4 py-3 text-sm text-gray-500">{r.description || '—'}</td>
                  <td className="px-4 py-3"><button onClick={() => handleDelete(r.id)} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
