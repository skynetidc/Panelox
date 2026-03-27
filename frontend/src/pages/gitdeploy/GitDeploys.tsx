import { useEffect, useState } from 'react';
import { gitDeployApi, type GitDeploy, type DeployHistory } from '../../api/wave3';
import { Plus, Trash2, Play, RotateCw, ChevronDown, GitBranch } from 'lucide-react';

export default function GitDeploys() {
  const [deploys, setDeploys] = useState<GitDeploy[]>([]);
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [history, setHistory] = useState<DeployHistory[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ website_id: 0, repo_url: '', branch: 'main', build_command: '', auto_deploy: true });

  const fetch = () => { gitDeployApi.list().then(r => setDeploys(r.data.data || [])).catch(() => {}); };
  useEffect(() => { fetch(); }, []);

  const toggleExpand = async (id: number) => {
    if (expandedId === id) { setExpandedId(null); return; }
    setExpandedId(id);
    const r = await gitDeployApi.history(id);
    setHistory(r.data.data || []);
  };

  const statusColors: Record<string, string> = { success: 'bg-green-100 text-green-700', failed: 'bg-red-100 text-red-700', deploying: 'bg-yellow-100 text-yellow-700' };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Git Deploy</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 text-sm"><Plus size={16} /> New Deploy</button>
      </div>

      {showCreate && (
        <div className="bg-white rounded-xl shadow-sm border p-6 mb-4 space-y-4 max-w-2xl">
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium text-gray-700">Website ID *</label><input type="number" value={form.website_id || ''} onChange={e => setForm(f => ({ ...f, website_id: Number(e.target.value) }))} className="w-full px-3 py-2 border rounded-lg text-sm outline-none" /></div>
            <div><label className="text-sm font-medium text-gray-700">Branch</label><input type="text" value={form.branch} onChange={e => setForm(f => ({ ...f, branch: e.target.value }))} className="w-full px-3 py-2 border rounded-lg text-sm outline-none" /></div>
          </div>
          <div><label className="text-sm font-medium text-gray-700">Repository URL *</label><input type="text" value={form.repo_url} onChange={e => setForm(f => ({ ...f, repo_url: e.target.value }))} placeholder="https://github.com/user/repo.git" className="w-full px-3 py-2 border rounded-lg text-sm outline-none" /></div>
          <div><label className="text-sm font-medium text-gray-700">Build Command</label><input type="text" value={form.build_command} onChange={e => setForm(f => ({ ...f, build_command: e.target.value }))} placeholder="npm install && npm run build" className="w-full px-3 py-2 border rounded-lg text-sm font-mono outline-none" /></div>
          <label className="flex items-center gap-2"><input type="checkbox" checked={form.auto_deploy} onChange={e => setForm(f => ({ ...f, auto_deploy: e.target.checked }))} /><span className="text-sm">Auto-deploy on git push</span></label>
          <div className="flex gap-2">
            <button onClick={() => { gitDeployApi.create(form).then(() => { setShowCreate(false); fetch(); }); }} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Create</button>
            <button onClick={() => setShowCreate(false)} className="px-4 py-2 border rounded-lg text-sm">Cancel</button>
          </div>
        </div>
      )}

      <div className="space-y-3">
        {deploys.length === 0 ? (
          <div className="bg-white rounded-xl p-12 text-center"><GitBranch size={48} className="mx-auto mb-4 text-gray-300" /><p className="text-gray-400">No git deploys configured</p></div>
        ) : deploys.map(d => (
          <div key={d.id} className="bg-white rounded-xl shadow-sm border">
            <div className="px-6 py-4 flex items-center justify-between cursor-pointer" onClick={() => toggleExpand(d.id)}>
              <div className="flex items-center gap-3">
                <ChevronDown size={16} className={expandedId === d.id ? '' : '-rotate-90'} />
                <GitBranch size={18} className="text-purple-500" />
                <div>
                  <p className="font-medium text-sm">{d.repo_url.split('/').slice(-1)[0]?.replace('.git', '')}</p>
                  <p className="text-xs text-gray-500">{d.repo_url} &middot; {d.branch} {d.auto_deploy && '&middot; auto-deploy'}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                {d.last_status && <span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColors[d.last_status] || 'bg-gray-100'}`}>{d.last_status}</span>}
                {d.last_commit && <span className="text-xs font-mono text-gray-400">{d.last_commit.slice(0, 8)}</span>}
                <div className="flex gap-1" onClick={e => e.stopPropagation()}>
                  <button onClick={() => { gitDeployApi.deploy(d.id); fetch(); }} className="p-1 text-gray-400 hover:text-green-600" title="Deploy now"><Play size={14} /></button>
                  <button onClick={() => { gitDeployApi.rollback(d.id); fetch(); }} className="p-1 text-gray-400 hover:text-orange-600" title="Rollback"><RotateCw size={14} /></button>
                  <button onClick={() => { gitDeployApi.delete(d.id); fetch(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                </div>
              </div>
            </div>

            {expandedId === d.id && (
              <div className="border-t px-6 py-4">
                <div className="mb-3">
                  <p className="text-xs text-gray-500">Webhook URL:</p>
                  <code className="text-xs bg-gray-100 px-2 py-1 rounded">/api/v1/webhook/git/{d.id}</code>
                  <p className="text-xs text-gray-400 mt-1">Secret: {d.webhook_secret}</p>
                </div>
                {d.last_log && (
                  <div className="mb-3"><p className="text-xs text-gray-500 mb-1">Last deploy log:</p>
                  <pre className="bg-gray-900 text-green-400 text-xs p-3 rounded max-h-32 overflow-y-auto whitespace-pre-wrap">{d.last_log}</pre></div>
                )}
                <h4 className="text-xs font-semibold text-gray-500 mb-2">Deploy History</h4>
                <div className="space-y-1">
                  {history.map(h => (
                    <div key={h.id} className="flex items-center justify-between text-xs bg-gray-50 px-3 py-2 rounded">
                      <div className="flex items-center gap-2">
                        <span className={`px-1.5 py-0.5 rounded ${statusColors[h.status] || ''}`}>{h.status}</span>
                        <span className="font-mono">{h.commit_hash.slice(0, 8)}</span>
                        <span className="text-gray-500">{h.commit_msg}</span>
                      </div>
                      <span className="text-gray-400">{h.duration_ms}ms &middot; {new Date(h.created_at).toLocaleString()}</span>
                    </div>
                  ))}
                  {history.length === 0 && <p className="text-xs text-gray-400">No deploys yet</p>}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
