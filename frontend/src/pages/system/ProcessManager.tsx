import { useEffect, useState } from 'react';
import api from '../../api/client';

interface Proc { pid: number; user: string; cpu: number; mem: number; vsz: string; rss: string; stat: string; command: string; }

export default function ProcessManager() {
  const [procs, setProcs] = useState<Proc[]>([]);
  const [filter, setFilter] = useState('');
  const [loading, setLoading] = useState(true);

  const fetch = () => { setLoading(true); api.get<{ data: Proc[] }>('/processes').then(r => setProcs(r.data.data || [])).catch(() => {}).finally(() => setLoading(false)); };
  useEffect(() => { fetch(); const i = setInterval(fetch, 5000); return () => clearInterval(i); }, []);

  const kill = (pid: number, signal: string) => { if (confirm(`Send ${signal} to PID ${pid}?`)) { api.post(`/processes/${pid}/kill?signal=${signal}`).then(fetch); } };

  const filtered = procs.filter(p => filter ? p.command.toLowerCase().includes(filter.toLowerCase()) || p.user.includes(filter) : true);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">Process Manager</h1>
          <p className="text-sm text-gray-500 mt-1">{procs.length} processes · auto-refresh 5s</p>
        </div>
        <button onClick={fetch} className="py-2 px-4 text-sm font-medium rounded-lg border border-gray-200 hover:bg-gray-50">Refresh</button>
      </div>

      <div className="relative mb-4">
        <div className="absolute inset-y-0 start-0 flex items-center ps-3.5"><svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg></div>
        <input type="text" value={filter} onChange={e => setFilter(e.target.value)} placeholder="Filter by command or user..." className="py-2.5 ps-10 pe-4 block w-full border border-gray-200 rounded-lg text-sm focus:border-blue-500 focus:ring-blue-500" />
      </div>

      <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
        {loading && procs.length === 0 ? <div className="p-12 text-center text-gray-400">Loading...</div> : (
          <div className="overflow-x-auto" style={{ maxHeight: '70vh' }}>
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50 sticky top-0"><tr>
                <th className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">PID</th>
                <th className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">User</th>
                <th className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">CPU%</th>
                <th className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">MEM%</th>
                <th className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Command</th>
                <th className="px-4 py-3 text-end"></th>
              </tr></thead>
              <tbody className="divide-y divide-gray-200">
                {filtered.slice(0, 100).map(p => (
                  <tr key={p.pid} className="hover:bg-gray-50">
                    <td className="px-4 py-2 text-sm font-mono">{p.pid}</td>
                    <td className="px-4 py-2 text-sm">{p.user}</td>
                    <td className="px-4 py-2 text-sm"><span className={p.cpu > 50 ? 'text-red-600 font-bold' : p.cpu > 10 ? 'text-yellow-600' : ''}>{p.cpu}%</span></td>
                    <td className="px-4 py-2 text-sm"><span className={p.mem > 50 ? 'text-red-600 font-bold' : ''}>{p.mem}%</span></td>
                    <td className="px-4 py-2 text-xs font-mono text-gray-600 max-w-md truncate">{p.command}</td>
                    <td className="px-4 py-2 text-end">
                      <div className="inline-flex gap-1">
                        <button onClick={() => kill(p.pid, 'TERM')} className="py-1 px-2 text-xs rounded border border-yellow-200 text-yellow-600 hover:bg-yellow-50">TERM</button>
                        <button onClick={() => kill(p.pid, 'KILL')} className="py-1 px-2 text-xs rounded border border-red-200 text-red-600 hover:bg-red-50">KILL</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
