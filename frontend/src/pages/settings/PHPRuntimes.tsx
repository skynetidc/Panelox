import { useEffect, useState } from 'react';
import { phpApi, type PHPVersion, type PHPRuntime } from '../../api/php';
import { Trash2, Play, Square, RotateCw } from 'lucide-react';

export default function PHPRuntimes() {
  const [versions, setVersions] = useState<PHPVersion[]>([]);
  const [runtimes, setRuntimes] = useState<PHPRuntime[]>([]);
  const [installing, setInstalling] = useState<string | null>(null);

  const fetch = async () => {
    try {
      const [v, r] = await Promise.all([phpApi.versions(), phpApi.runtimes()]);
      setVersions(v.data.data || []);
      setRuntimes(r.data.data || []);
    } catch {}
  };
  useEffect(() => { fetch(); }, []);

  const handleInstall = async (version: string) => {
    setInstalling(version);
    await phpApi.install(version).catch(() => {});
    setInstalling(null); fetch();
  };

  const handleRemove = async (id: number, version: string) => {
    if (!confirm(`Remove PHP ${version}?`)) return;
    await phpApi.remove(id).catch(() => {}); fetch();
  };

  const handleOperate = async (id: number, action: string) => {
    await phpApi.operate(id, action).catch(() => {}); fetch();
  };

  const statusColors: Record<string, string> = { running: 'bg-green-100 text-green-700', stopped: 'bg-gray-100 text-gray-600', installing: 'bg-yellow-100 text-yellow-700', error: 'bg-red-100 text-red-700' };

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">PHP Runtimes</h1>

      {/* Available versions */}
      <div className="grid grid-cols-5 gap-3 mb-6">
        {versions.map(v => (
          <div key={v.version} className="bg-white rounded-xl shadow-sm border p-4 text-center">
            <p className="text-lg font-bold text-gray-900">PHP {v.version}</p>
            <p className="text-xs text-gray-400 mb-3">{v.image}</p>
            <button
              onClick={() => handleInstall(v.version)}
              disabled={v.installed || installing === v.version}
              className={`w-full py-1.5 rounded-lg text-sm font-medium ${v.installed ? 'bg-gray-100 text-gray-400' : 'bg-blue-600 text-white hover:bg-blue-700'} disabled:opacity-50`}
            >
              {v.installed ? 'Installed' : installing === v.version ? 'Installing...' : 'Install'}
            </button>
          </div>
        ))}
      </div>

      {/* Installed runtimes */}
      {runtimes.length > 0 && (
        <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Version</th><th className="px-4 py-3">Container</th><th className="px-4 py-3">Port</th><th className="px-4 py-3">Status</th><th className="px-4 py-3">Actions</th></tr></thead>
            <tbody>
              {runtimes.map(rt => (
                <tr key={rt.id} className="border-b border-gray-50 hover:bg-gray-50">
                  <td className="px-4 py-3 font-bold">PHP {rt.version}</td>
                  <td className="px-4 py-3 text-sm font-mono text-gray-500">{rt.container_name}</td>
                  <td className="px-4 py-3 text-sm">127.0.0.1:{rt.port}</td>
                  <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColors[rt.status] || ''}`}>{rt.status}</span></td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1">
                      {rt.status !== 'running' && <button onClick={() => handleOperate(rt.id, 'start')} className="p-1 text-gray-400 hover:text-green-600"><Play size={14} /></button>}
                      {rt.status === 'running' && <button onClick={() => handleOperate(rt.id, 'stop')} className="p-1 text-gray-400 hover:text-orange-600"><Square size={14} /></button>}
                      {rt.status === 'running' && <button onClick={() => handleOperate(rt.id, 'restart')} className="p-1 text-gray-400 hover:text-blue-600"><RotateCw size={14} /></button>}
                      <button onClick={() => handleRemove(rt.id, rt.version)} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
