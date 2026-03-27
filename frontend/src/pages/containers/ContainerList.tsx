import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { containerApi, type ContainerInfo } from '../../api/containers';
import { Box, Play, Square, RotateCw, Trash2, RefreshCw, FileText } from 'lucide-react';

export default function ContainerList() {
  const { t } = useTranslation();
  const [containers, setContainers] = useState<ContainerInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [logsId, setLogsId] = useState<string | null>(null);
  const [logsContent, setLogsContent] = useState('');

  const fetch = async () => {
    setLoading(true);
    try {
      const res = await containerApi.list();
      setContainers(res.data.data || []);
    } catch { setContainers([]); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetch(); }, []);

  const handleAction = async (id: string, action: 'start' | 'stop' | 'restart') => {
    await containerApi[action](id).catch(() => {});
    fetch();
  };

  const handleRemove = async (id: string, name: string) => {
    if (!confirm(`Remove container "${name}"?`)) return;
    await containerApi.remove(id).catch(() => {});
    fetch();
  };

  const handleLogs = async (id: string) => {
    if (logsId === id) { setLogsId(null); return; }
    setLogsId(id);
    try {
      const res = await containerApi.logs(id, 200);
      setLogsContent(res.data.data.logs);
    } catch { setLogsContent('Failed to load logs'); }
  };

  const stateColors: { [k: string]: string } = {
    running: 'bg-green-100 text-green-700',
    exited: 'bg-gray-100 text-gray-600',
    created: 'bg-yellow-100 text-yellow-700',
    restarting: 'bg-blue-100 text-blue-700',
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{t('nav.containers')}</h1>
        <button onClick={fetch} className="flex items-center gap-1 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50"><RefreshCw size={14} /> Refresh</button>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        {loading ? (
          <div className="p-12 text-center text-gray-400">{t('common.loading')}</div>
        ) : containers.length === 0 ? (
          <div className="p-12 text-center">
            <Box size={48} className="mx-auto mb-4 text-gray-300" />
            <p className="text-gray-400">No containers found</p>
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-xs text-gray-500 uppercase">
                <th className="px-4 py-3">{t('common.name')}</th><th className="px-4 py-3">Image</th>
                <th className="px-4 py-3">{t('common.status')}</th><th className="px-4 py-3">Ports</th><th className="px-4 py-3">{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {containers.map(c => (
                <>
                  <tr key={c.id} className="border-b border-gray-50 hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <p className="font-medium text-sm">{c.name}</p>
                      <p className="text-xs text-gray-400 font-mono">{c.id}</p>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 font-mono">{c.image}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${stateColors[c.state] || 'bg-gray-100'}`}>{c.state}</span>
                      <p className="text-xs text-gray-400 mt-0.5">{c.status}</p>
                    </td>
                    <td className="px-4 py-3 text-xs font-mono text-gray-600 max-w-xs truncate">{c.ports || '—'}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-1">
                        {c.state !== 'running' && <button onClick={() => handleAction(c.id, 'start')} className="p-1 text-gray-400 hover:text-green-600" title="Start"><Play size={14} /></button>}
                        {c.state === 'running' && <button onClick={() => handleAction(c.id, 'stop')} className="p-1 text-gray-400 hover:text-orange-600" title="Stop"><Square size={14} /></button>}
                        {c.state === 'running' && <button onClick={() => handleAction(c.id, 'restart')} className="p-1 text-gray-400 hover:text-blue-600" title="Restart"><RotateCw size={14} /></button>}
                        <button onClick={() => handleLogs(c.id)} className="p-1 text-gray-400 hover:text-purple-600" title="Logs"><FileText size={14} /></button>
                        <button onClick={() => handleRemove(c.id, c.name)} className="p-1 text-gray-400 hover:text-red-600" title="Remove"><Trash2 size={14} /></button>
                      </div>
                    </td>
                  </tr>
                  {logsId === c.id && (
                    <tr key={`${c.id}-logs`}>
                      <td colSpan={5} className="px-4 py-2 bg-gray-900">
                        <pre className="text-xs text-green-400 font-mono max-h-64 overflow-y-auto whitespace-pre-wrap">{logsContent}</pre>
                      </td>
                    </tr>
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
