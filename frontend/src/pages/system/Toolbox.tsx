import { useEffect, useState } from 'react';
import api from '../../api/client';

type Tab = 'cleanup' | 'clamav' | 'ssh' | 'disk';

export default function Toolbox() {
  const [tab, setTab] = useState<Tab>('cleanup');
  const [cleanupInfo, setCleanupInfo] = useState<any>(null);
  const [clamStatus, setClamStatus] = useState<any>(null);
  const [scanResult, setScanResult] = useState<any>(null);
  const [scanning, setScanning] = useState(false);
  const [sshConfig, setSshConfig] = useState<any>(null);
  const [sshKeys, setSshKeys] = useState<any[]>([]);
  const [diskUsage, setDiskUsage] = useState<any[]>([]);
  const [newKey, setNewKey] = useState('');
  const [cleaning, setCleaning] = useState('');

  useEffect(() => {
    if (tab === 'cleanup') api.get('/cleanup/info').then(r => setCleanupInfo(r.data.data)).catch(() => {});
    if (tab === 'clamav') api.get('/clamav/status').then(r => setClamStatus(r.data.data)).catch(() => {});
    if (tab === 'ssh') { api.get('/ssh/config').then(r => setSshConfig(r.data.data)).catch(() => {}); api.get('/ssh/authorized-keys').then(r => setSshKeys(r.data.data || [])).catch(() => {}); }
    if (tab === 'disk') api.get('/disk/usage').then(r => setDiskUsage(r.data.data || [])).catch(() => {});
  }, [tab]);

  const clean = async (type: string) => { setCleaning(type); await api.post(`/cleanup/${type}`).catch(() => {}); setCleaning(''); api.get('/cleanup/info').then(r => setCleanupInfo(r.data.data)); };
  const scan = async (path: string) => { setScanning(true); const r = await api.post('/clamav/scan', { path }).catch(() => null); if (r) setScanResult(r.data.data); setScanning(false); };
  const saveSsh = async () => { await api.put('/ssh/config', sshConfig).catch(() => {}); alert('SSH config saved & reloaded'); };
  const addKey = async () => { if (!newKey) return; await api.post('/ssh/authorized-keys', { key: newKey }); setNewKey(''); api.get('/ssh/authorized-keys').then(r => setSshKeys(r.data.data || [])); };
  const removeKey = async (key: string) => { await api.delete('/ssh/authorized-keys', { data: { key } }); api.get('/ssh/authorized-keys').then(r => setSshKeys(r.data.data || [])); };

  const tabs: { key: Tab; label: string; icon: string }[] = [
    { key: 'cleanup', label: 'System Cleanup', icon: '🧹' },
    { key: 'clamav', label: 'Antivirus', icon: '🛡️' },
    { key: 'ssh', label: 'SSH Config', icon: '🔑' },
    { key: 'disk', label: 'Disk Usage', icon: '💾' },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-800 mb-6">Toolbox</h1>

      <div className="flex gap-2 mb-6">
        {tabs.map(t => (
          <button key={t.key} onClick={() => setTab(t.key)} className={`py-2 px-4 text-sm font-medium rounded-lg transition-colors ${tab === t.key ? 'bg-blue-600 text-white' : 'bg-white border border-gray-200 text-gray-600 hover:bg-gray-50'}`}>
            {t.icon} {t.label}
          </button>
        ))}
      </div>

      {/* CLEANUP */}
      {tab === 'cleanup' && (
        <div className="space-y-4">
          {cleanupInfo && (
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              {[
                { l: 'Logs', v: cleanupInfo.logs || '—', a: 'logs' },
                { l: 'Temp', v: cleanupInfo.tmp || '—', a: 'cache' },
                { l: 'Journal', v: cleanupInfo.journal || '—', a: 'journal' },
                { l: 'Docker', v: 'Prune all', a: 'docker' },
              ].map(i => (
                <div key={i.a} className="bg-white border border-gray-200 rounded-xl p-4">
                  <p className="text-sm text-gray-500">{i.l}</p>
                  <p className="text-lg font-bold mt-1">{i.v}</p>
                  <button onClick={() => clean(i.a)} disabled={cleaning === i.a} className="mt-3 py-1.5 px-3 text-xs font-medium rounded-lg bg-red-50 text-red-600 hover:bg-red-100 disabled:opacity-50 w-full">
                    {cleaning === i.a ? 'Cleaning...' : `Clean ${i.l}`}
                  </button>
                </div>
              ))}
            </div>
          )}
          <button onClick={() => clean('all')} disabled={!!cleaning} className="py-2 px-4 text-sm font-medium rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50">
            {cleaning === 'all' ? 'Cleaning everything...' : '🧹 Clean All'}
          </button>
        </div>
      )}

      {/* CLAMAV */}
      {tab === 'clamav' && (
        <div className="space-y-4">
          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <p className="text-sm text-gray-500">ClamAV Status</p>
            <p className="text-lg font-bold mt-1">{clamStatus?.installed ? `✅ Installed - ${clamStatus.version}` : '❌ Not installed (apt install clamav)'}</p>
          </div>
          <div className="flex gap-3">
            <button onClick={() => scan('/www/sites')} disabled={scanning} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
              {scanning ? '🔄 Scanning...' : '🔍 Scan /www/sites'}
            </button>
            <button onClick={() => api.post('/clamav/update')} className="py-2 px-4 text-sm font-medium rounded-lg border border-gray-200 hover:bg-gray-50">Update DB</button>
          </div>
          {scanResult && (
            <div className="bg-white border border-gray-200 rounded-xl p-5">
              <div className="flex gap-6 mb-3">
                <div><p className="text-xs text-gray-500">Scanned</p><p className="text-xl font-bold">{scanResult.scanned}</p></div>
                <div><p className="text-xs text-gray-500">Infected</p><p className={`text-xl font-bold ${scanResult.infected > 0 ? 'text-red-600' : 'text-green-600'}`}>{scanResult.infected}</p></div>
              </div>
              {scanResult.threats?.length > 0 && (
                <div className="mt-3">
                  <p className="text-xs font-semibold text-red-600 mb-1">Threats found:</p>
                  {scanResult.threats.map((t: string, i: number) => <p key={i} className="text-xs font-mono text-red-500">{t}</p>)}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* SSH */}
      {tab === 'ssh' && sshConfig && (
        <div className="space-y-4">
          <div className="bg-white border border-gray-200 rounded-xl p-5 max-w-lg space-y-3">
            <h3 className="font-semibold mb-2">SSH Daemon Config</h3>
            {[
              { l: 'Port', k: 'port' },
              { l: 'Permit Root Login', k: 'permit_root_login' },
              { l: 'Password Auth', k: 'password_auth' },
              { l: 'Pubkey Auth', k: 'pubkey_auth' },
              { l: 'Max Auth Tries', k: 'max_auth_tries' },
            ].map(f => (
              <div key={f.k} className="flex items-center justify-between">
                <label className="text-sm text-gray-600">{f.l}</label>
                <input type="text" value={(sshConfig as any)[f.k] || ''} onChange={e => setSshConfig((c: any) => ({ ...c, [f.k]: e.target.value }))} className="py-1.5 px-3 w-32 border border-gray-200 rounded-lg text-sm text-right" />
              </div>
            ))}
            <button onClick={saveSsh} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 w-full mt-2">Save & Reload SSH</button>
          </div>

          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <h3 className="font-semibold mb-3">Authorized Keys ({sshKeys.length})</h3>
            <div className="space-y-2 mb-3">
              {sshKeys.map((k, i) => (
                <div key={i} className="flex items-center justify-between bg-gray-50 rounded-lg p-2">
                  <span className="text-xs font-mono truncate flex-1">{k.comment || k.key?.slice(0, 60) + '...'}</span>
                  <button onClick={() => removeKey(k.key)} className="ml-2 text-xs text-red-500 hover:underline">Remove</button>
                </div>
              ))}
            </div>
            <div className="flex gap-2">
              <input type="text" value={newKey} onChange={e => setNewKey(e.target.value)} placeholder="ssh-ed25519 AAAA..." className="flex-1 py-2 px-3 border border-gray-200 rounded-lg text-xs font-mono" />
              <button onClick={addKey} className="py-2 px-4 text-sm rounded-lg bg-blue-600 text-white hover:bg-blue-700">Add</button>
            </div>
          </div>
        </div>
      )}

      {/* DISK */}
      {tab === 'disk' && (
        <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
          {diskUsage.length === 0 ? <div className="p-12 text-center text-gray-400">No data</div> : (
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50"><tr>
                <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Website</th>
                <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Path</th>
                <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Size</th>
              </tr></thead>
              <tbody className="divide-y divide-gray-200">
                {diskUsage.map((d: any) => (
                  <tr key={d.path} className="hover:bg-gray-50">
                    <td className="px-6 py-3 text-sm font-medium">{d.name}</td>
                    <td className="px-6 py-3 text-sm text-gray-500 font-mono">{d.path}</td>
                    <td className="px-6 py-3 text-sm font-bold">{d.size}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}
