import { useEffect, useState } from 'react';
import { cronApi, type CronJob } from '../../api/cronjobs';
import { Plus, Trash2, Play, ToggleLeft, ToggleRight, Clock, ChevronDown } from 'lucide-react';

export default function CronJobs() {
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newJob, setNewJob] = useState({ name: '', schedule: '@daily', command: '', type: 'shell' });
  const [expandedId, setExpandedId] = useState<number | null>(null);

  const fetch = async () => {
    setLoading(true);
    try { const r = await cronApi.list(); setJobs(r.data.data || []); } catch {}
    finally { setLoading(false); }
  };
  useEffect(() => { fetch(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await cronApi.create(newJob).catch(() => {});
    setNewJob({ name: '', schedule: '@daily', command: '', type: 'shell' });
    setShowCreate(false); fetch();
  };

  const statusColors: Record<string, string> = { success: 'text-green-600', failed: 'text-red-600' };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Cron Jobs</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"><Plus size={18} /> New Job</button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="bg-white rounded-xl shadow-sm border p-6 mb-4 space-y-4 max-w-2xl">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
              <input type="text" value={newJob.name} onChange={e => setNewJob(j => ({ ...j, name: e.target.value }))} required className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Schedule *</label>
              <select value={newJob.schedule} onChange={e => setNewJob(j => ({ ...j, schedule: e.target.value }))} className="w-full px-3 py-2 border rounded-lg text-sm">
                <option value="@every 5m">Every 5 min</option>
                <option value="@every 30m">Every 30 min</option>
                <option value="@hourly">Hourly</option>
                <option value="@daily">Daily</option>
                <option value="@weekly">Weekly</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-4 gap-4">
            <div className="col-span-3">
              <label className="block text-sm font-medium text-gray-700 mb-1">Command *</label>
              <input type="text" value={newJob.command} onChange={e => setNewJob(j => ({ ...j, command: e.target.value }))} required placeholder="/usr/bin/backup.sh" className="w-full px-3 py-2 border rounded-lg text-sm font-mono outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
              <select value={newJob.type} onChange={e => setNewJob(j => ({ ...j, type: e.target.value }))} className="w-full px-3 py-2 border rounded-lg text-sm">
                <option value="shell">Shell</option><option value="curl">cURL</option>
              </select>
            </div>
          </div>
          <div className="flex gap-2">
            <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Create</button>
            <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 border rounded-lg text-sm">Cancel</button>
          </div>
        </form>
      )}

      <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
        {loading ? <div className="p-12 text-center text-gray-400">Loading...</div> : jobs.length === 0 ? (
          <div className="p-12 text-center"><Clock size={48} className="mx-auto mb-4 text-gray-300" /><p className="text-gray-400">No cron jobs</p></div>
        ) : (
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Schedule</th><th className="px-4 py-3">Command</th><th className="px-4 py-3">Last Run</th><th className="px-4 py-3">Actions</th></tr></thead>
            <tbody>
              {jobs.map(j => (
                <>
                  <tr key={j.id} className="border-b border-gray-50 hover:bg-gray-50">
                    <td className="px-4 py-3 font-medium text-sm">{j.name}</td>
                    <td className="px-4 py-3 text-sm font-mono text-gray-600">{j.schedule}</td>
                    <td className="px-4 py-3 text-xs font-mono text-gray-600 max-w-xs truncate">{j.command}</td>
                    <td className="px-4 py-3 text-xs">
                      {j.last_run ? (<span className={statusColors[j.last_status || ''] || 'text-gray-500'}>{j.last_status} - {new Date(j.last_run).toLocaleString()}</span>) : <span className="text-gray-400">Never</span>}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-1">
                        <button onClick={() => { cronApi.toggle(j.id); fetch(); }} className="p-1 text-gray-400 hover:text-blue-600">{j.enabled ? <ToggleRight size={16} className="text-green-500" /> : <ToggleLeft size={16} />}</button>
                        <button onClick={() => { cronApi.run(j.id); fetch(); }} className="p-1 text-gray-400 hover:text-green-600"><Play size={14} /></button>
                        <button onClick={() => setExpandedId(expandedId === j.id ? null : j.id)} className="p-1 text-gray-400 hover:text-purple-600"><ChevronDown size={14} /></button>
                        <button onClick={() => { cronApi.delete(j.id); fetch(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                      </div>
                    </td>
                  </tr>
                  {expandedId === j.id && j.last_output && (
                    <tr key={`${j.id}-out`}><td colSpan={5} className="bg-gray-900 px-4 py-2"><pre className="text-xs text-green-400 font-mono max-h-48 overflow-y-auto whitespace-pre-wrap">{j.last_output}</pre></td></tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
