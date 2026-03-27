import { useEffect, useState } from 'react';
import { securityApi, type LoginEntry, type F2BRule, type BannedIP, type Channel, type AlertRule, type Snapshot } from '../../api/security';
import api from '../../api/client';
import {
  Shield, Clock, Key, Bell, AlertTriangle, Camera, Plus, Trash2,
  ToggleLeft, ToggleRight, Download, RotateCw, Send, Bug,
} from 'lucide-react';

type Tab = 'login-log' | '2fa' | 'fail2ban' | 'notifications' | 'alerts' | 'snapshots' | 'malware';

interface ScanResult { path: string; infected: number; scanned: number; threats: string[]; output: string; }
interface ClamStatus { installed: boolean; version: string; }

export default function Security() {
  const [tab, setTab] = useState<Tab>('login-log');
  const tabs: { key: Tab; label: string; icon: any }[] = [
    { key: 'login-log', label: 'Login Log', icon: Clock },
    { key: '2fa', label: '2FA', icon: Key },
    { key: 'fail2ban', label: 'Fail2Ban', icon: Shield },
    { key: 'notifications', label: 'Notifications', icon: Bell },
    { key: 'alerts', label: 'Alerts', icon: AlertTriangle },
    { key: 'snapshots', label: 'Snapshots', icon: Camera },
    { key: 'malware', label: 'Malware Scanner', icon: Bug },
  ];

  // State for each tab
  const [loginLog, setLoginLog] = useState<LoginEntry[]>([]);
  const [totpEnabled, setTotpEnabled] = useState(false);
  const [totpUri, setTotpUri] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [f2bRules, setF2bRules] = useState<F2BRule[]>([]);
  const [banned, setBanned] = useState<BannedIP[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [alertRules, setAlertRules] = useState<AlertRule[]>([]);
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);

  // Forms
  const [newF2b, setNewF2b] = useState({ name: '', port: '', max_retry: 5, ban_time: 3600 });
  const [newChannel, setNewChannel] = useState({ name: '', type: 'telegram', config: {} as any });
  const [newAlert, setNewAlert] = useState({ name: '', metric: 'cpu', operator: '>', threshold: 90, duration_sec: 300 });

  // Malware scanner
  const [clamStatus, setClamStatus] = useState<ClamStatus | null>(null);
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [scanning, setScanning] = useState(false);
  const [scanPath, setScanPath] = useState('/www/sites');
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    switch (tab) {
      case 'login-log': securityApi.loginLog({ page: 1, size: 50 }).then(r => setLoginLog(r.data.data || [])).catch(() => {}); break;
      case '2fa': securityApi.totpStatus().then(r => setTotpEnabled(r.data.data.enabled)).catch(() => {}); break;
      case 'fail2ban':
        securityApi.f2bRules().then(r => setF2bRules(r.data.data || [])).catch(() => {});
        securityApi.f2bBanned().then(r => setBanned(r.data.data || [])).catch(() => {}); break;
      case 'notifications': securityApi.channels().then(r => setChannels(r.data.data || [])).catch(() => {}); break;
      case 'alerts': securityApi.alertRules().then(r => setAlertRules(r.data.data || [])).catch(() => {}); break;
      case 'snapshots': securityApi.snapshots().then(r => setSnapshots(r.data.data || [])).catch(() => {}); break;
      case 'malware': api.get('/clamav/status').then(r => setClamStatus(r.data.data)).catch(() => {}); break;
    }
  }, [tab]);

  const formatSize = (b: number) => { if (!b) return '—'; const u = ['B','KB','MB','GB']; let i=0,s=b; while(s>=1024&&i<u.length-1){s/=1024;i++;} return `${s.toFixed(1)} ${u[i]}`; };

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-4">Security & Monitoring</h1>

      <div className="border-b border-gray-200 mb-6">
        <nav className="flex gap-4 overflow-x-auto">
          {tabs.map(t => (
            <button key={t.key} onClick={() => setTab(t.key)}
              className={`flex items-center gap-1 pb-3 text-sm font-medium border-b-2 whitespace-nowrap ${tab === t.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'}`}>
              <t.icon size={14} /> {t.label}
            </button>
          ))}
        </nav>
      </div>

      {/* LOGIN LOG */}
      {tab === 'login-log' && (
        <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Time</th><th className="px-4 py-3">Email</th><th className="px-4 py-3">IP</th><th className="px-4 py-3">Result</th></tr></thead>
            <tbody>
              {loginLog.map(e => (
                <tr key={e.id} className="border-b border-gray-50"><td className="px-4 py-2 text-xs text-gray-500">{new Date(e.created_at).toLocaleString()}</td><td className="px-4 py-2 text-sm">{e.email}</td><td className="px-4 py-2 text-xs font-mono">{e.ip_address}</td>
                  <td className="px-4 py-2"><span className={`px-2 py-0.5 rounded text-xs ${e.success ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{e.success ? 'Success' : 'Failed'}</span></td></tr>
              ))}
              {loginLog.length === 0 && <tr><td colSpan={4} className="px-6 py-8 text-center text-gray-400">No login attempts</td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {/* 2FA */}
      {tab === '2fa' && (
        <div className="bg-white rounded-xl shadow-sm border p-6 max-w-md">
          <h3 className="font-semibold mb-4">Two-Factor Authentication</h3>
          <p className="text-sm text-gray-500 mb-4">Status: <span className={totpEnabled ? 'text-green-600 font-medium' : 'text-gray-400'}>{totpEnabled ? 'Enabled' : 'Disabled'}</span></p>
          {!totpEnabled ? (
            <div className="space-y-3">
              {totpUri ? (
                <>
                  <p className="text-sm">Scan this URI in your authenticator app:</p>
                  <code className="block bg-gray-100 p-3 rounded text-xs break-all">{totpUri}</code>
                  <input type="text" value={totpCode} onChange={e => setTotpCode(e.target.value)} placeholder="Enter 6-digit code" className="w-full px-3 py-2 border rounded-lg text-sm outline-none" />
                  <button onClick={() => securityApi.totpEnable(totpCode).then(() => { setTotpEnabled(true); setTotpUri(''); setTotpCode(''); })} className="px-4 py-2 bg-green-600 text-white rounded-lg text-sm">Verify & Enable</button>
                </>
              ) : (
                <button onClick={() => securityApi.totpSetup().then(r => setTotpUri(r.data.data.uri))} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Setup 2FA</button>
              )}
            </div>
          ) : (
            <div className="space-y-3">
              <input type="text" value={totpCode} onChange={e => setTotpCode(e.target.value)} placeholder="Enter code to disable" className="w-full px-3 py-2 border rounded-lg text-sm outline-none" />
              <button onClick={() => securityApi.totpDisable(totpCode).then(() => { setTotpEnabled(false); setTotpCode(''); })} className="px-4 py-2 bg-red-600 text-white rounded-lg text-sm">Disable 2FA</button>
            </div>
          )}
        </div>
      )}

      {/* FAIL2BAN */}
      {tab === 'fail2ban' && (
        <div className="space-y-4">
          <form onSubmit={e => { e.preventDefault(); securityApi.f2bCreateRule(newF2b).then(() => { setNewF2b({ name: '', port: '', max_retry: 5, ban_time: 3600 }); securityApi.f2bRules().then(r => setF2bRules(r.data.data || [])); }); }} className="bg-white rounded-xl shadow-sm border p-4 flex gap-3 items-end">
            <div><label className="text-xs text-gray-500">Name</label><input type="text" value={newF2b.name} onChange={e => setNewF2b(f => ({ ...f, name: e.target.value }))} placeholder="SSH" className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <div><label className="text-xs text-gray-500">Port</label><input type="text" value={newF2b.port} onChange={e => setNewF2b(f => ({ ...f, port: e.target.value }))} placeholder="22" className="w-24 px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <div><label className="text-xs text-gray-500">Max Retries</label><input type="number" value={newF2b.max_retry} onChange={e => setNewF2b(f => ({ ...f, max_retry: Number(e.target.value) }))} className="w-20 px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <div><label className="text-xs text-gray-500">Ban (sec)</label><input type="number" value={newF2b.ban_time} onChange={e => setNewF2b(f => ({ ...f, ban_time: Number(e.target.value) }))} className="w-24 px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <button type="submit" className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm"><Plus size={14} /></button>
          </form>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Port</th><th className="px-4 py-3">Max Retry</th><th className="px-4 py-3">Ban Time</th><th className="px-4 py-3">Actions</th></tr></thead>
              <tbody>
                {f2bRules.map(r => (
                  <tr key={r.id} className="border-b border-gray-50"><td className="px-4 py-3 font-medium">{r.name}</td><td className="px-4 py-3 font-mono text-sm">{r.port}</td><td className="px-4 py-3 text-sm">{r.max_retry}</td><td className="px-4 py-3 text-sm">{r.ban_time}s</td>
                    <td className="px-4 py-3 flex gap-1">
                      <button onClick={() => { securityApi.f2bToggleRule(r.id); securityApi.f2bRules().then(res => setF2bRules(res.data.data || [])); }}>{r.enabled ? <ToggleRight size={16} className="text-green-500" /> : <ToggleLeft size={16} />}</button>
                      <button onClick={() => { securityApi.f2bDeleteRule(r.id); securityApi.f2bRules().then(res => setF2bRules(res.data.data || [])); }} className="text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </td></tr>
                ))}
              </tbody>
            </table>
          </div>
          {banned.length > 0 && (
            <div className="bg-white rounded-xl shadow-sm border p-4">
              <h3 className="text-sm font-semibold mb-2">Banned IPs</h3>
              {banned.map(b => (
                <div key={b.id} className="flex items-center justify-between text-sm py-1">
                  <span className="font-mono">{b.ip_address}</span>
                  <span className="text-xs text-gray-400">until {new Date(b.expires_at).toLocaleString()}</span>
                  <button onClick={() => { securityApi.f2bUnban(b.ip_address); securityApi.f2bBanned().then(r => setBanned(r.data.data || [])); }} className="text-xs text-blue-600">Unban</button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* NOTIFICATIONS */}
      {tab === 'notifications' && (
        <div className="space-y-4">
          <div className="bg-white rounded-xl shadow-sm border p-4">
            <h3 className="text-sm font-semibold mb-3">Add Channel</h3>
            <div className="flex gap-3 items-end">
              <div><label className="text-xs text-gray-500">Name</label><input type="text" value={newChannel.name} onChange={e => setNewChannel(c => ({ ...c, name: e.target.value }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
              <div><label className="text-xs text-gray-500">Type</label>
                <select value={newChannel.type} onChange={e => setNewChannel(c => ({ ...c, type: e.target.value }))} className="px-2 py-1.5 border rounded text-sm">
                  <option value="telegram">Telegram</option><option value="discord">Discord</option><option value="email">Email</option><option value="webhook">Webhook</option>
                </select>
              </div>
              <div className="flex-1">
                <label className="text-xs text-gray-500">{newChannel.type === 'telegram' ? 'Bot Token' : newChannel.type === 'discord' ? 'Webhook URL' : newChannel.type === 'webhook' ? 'URL' : 'SMTP Host'}</label>
                <input type="text" onChange={e => {
                  const key = newChannel.type === 'telegram' ? 'bot_token' : newChannel.type === 'discord' ? 'webhook_url' : newChannel.type === 'webhook' ? 'url' : 'smtp_host';
                  setNewChannel(c => ({ ...c, config: { ...c.config, [key]: e.target.value } }));
                }} className="w-full px-2 py-1.5 border rounded text-sm outline-none" />
              </div>
              <button onClick={() => { securityApi.createChannel(newChannel).then(() => securityApi.channels().then(r => setChannels(r.data.data || []))); }} className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Add</button>
            </div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Type</th><th className="px-4 py-3">Status</th><th className="px-4 py-3">Actions</th></tr></thead>
              <tbody>
                {channels.map(ch => (
                  <tr key={ch.id} className="border-b border-gray-50"><td className="px-4 py-3 font-medium">{ch.name}</td><td className="px-4 py-3 text-sm">{ch.type}</td>
                    <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${ch.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100'}`}>{ch.enabled ? 'Active' : 'Disabled'}</span></td>
                    <td className="px-4 py-3 flex gap-1">
                      <button onClick={() => securityApi.testChannel(ch.id)} className="p-1 text-gray-400 hover:text-blue-600"><Send size={14} /></button>
                      <button onClick={() => { securityApi.toggleChannel(ch.id); securityApi.channels().then(r => setChannels(r.data.data || [])); }}>{ch.enabled ? <ToggleRight size={16} className="text-green-500" /> : <ToggleLeft size={16} />}</button>
                      <button onClick={() => { securityApi.deleteChannel(ch.id); securityApi.channels().then(r => setChannels(r.data.data || [])); }} className="text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </td></tr>
                ))}
                {channels.length === 0 && <tr><td colSpan={4} className="px-6 py-8 text-center text-gray-400">No notification channels</td></tr>}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ALERTS */}
      {tab === 'alerts' && (
        <div className="space-y-4">
          <form onSubmit={e => { e.preventDefault(); securityApi.createAlert(newAlert).then(() => { setNewAlert({ name: '', metric: 'cpu', operator: '>', threshold: 90, duration_sec: 300 }); securityApi.alertRules().then(r => setAlertRules(r.data.data || [])); }); }} className="bg-white rounded-xl shadow-sm border p-4 flex gap-3 items-end flex-wrap">
            <div><label className="text-xs text-gray-500">Name</label><input type="text" value={newAlert.name} onChange={e => setNewAlert(a => ({ ...a, name: e.target.value }))} placeholder="High CPU" className="w-36 px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <div><label className="text-xs text-gray-500">Metric</label>
              <select value={newAlert.metric} onChange={e => setNewAlert(a => ({ ...a, metric: e.target.value }))} className="px-2 py-1.5 border rounded text-sm">
                <option value="cpu">CPU %</option><option value="memory">Memory %</option><option value="disk">Disk %</option>
              </select>
            </div>
            <div><label className="text-xs text-gray-500">Op</label>
              <select value={newAlert.operator} onChange={e => setNewAlert(a => ({ ...a, operator: e.target.value }))} className="w-16 px-2 py-1.5 border rounded text-sm">
                <option value=">">&gt;</option><option value=">=">&gt;=</option><option value="<">&lt;</option>
              </select>
            </div>
            <div><label className="text-xs text-gray-500">Threshold %</label><input type="number" value={newAlert.threshold} onChange={e => setNewAlert(a => ({ ...a, threshold: Number(e.target.value) }))} className="w-20 px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <div><label className="text-xs text-gray-500">Duration (sec)</label><input type="number" value={newAlert.duration_sec} onChange={e => setNewAlert(a => ({ ...a, duration_sec: Number(e.target.value) }))} className="w-24 px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <button type="submit" className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Add Rule</button>
          </form>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Condition</th><th className="px-4 py-3">Status</th><th className="px-4 py-3">Last Fired</th><th className="px-4 py-3">Actions</th></tr></thead>
              <tbody>
                {alertRules.map(a => (
                  <tr key={a.id} className="border-b border-gray-50">
                    <td className="px-4 py-3 font-medium">{a.name}</td>
                    <td className="px-4 py-3 text-sm font-mono">{a.metric} {a.operator} {a.threshold}%</td>
                    <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${a.status === 'firing' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>{a.status}</span></td>
                    <td className="px-4 py-3 text-xs text-gray-500">{a.last_triggered ? new Date(a.last_triggered).toLocaleString() : 'Never'}</td>
                    <td className="px-4 py-3 flex gap-1">
                      <button onClick={() => { securityApi.toggleAlert(a.id); securityApi.alertRules().then(r => setAlertRules(r.data.data || [])); }}>{a.enabled ? <ToggleRight size={16} className="text-green-500" /> : <ToggleLeft size={16} />}</button>
                      <button onClick={() => { securityApi.deleteAlert(a.id); securityApi.alertRules().then(r => setAlertRules(r.data.data || [])); }} className="text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </td>
                  </tr>
                ))}
                {alertRules.length === 0 && <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-400">No alert rules</td></tr>}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* SNAPSHOTS */}
      {tab === 'snapshots' && (
        <div className="space-y-4">
          <button onClick={() => securityApi.createSnapshot({}).then(() => securityApi.snapshots().then(r => setSnapshots(r.data.data || [])))} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 text-sm"><Camera size={16} /> Create Snapshot</button>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Includes</th><th className="px-4 py-3">Size</th><th className="px-4 py-3">Status</th><th className="px-4 py-3">Created</th><th className="px-4 py-3">Actions</th></tr></thead>
              <tbody>
                {snapshots.map(s => (
                  <tr key={s.id} className="border-b border-gray-50">
                    <td className="px-4 py-3 font-medium text-sm">{s.name}</td>
                    <td className="px-4 py-3 text-xs">{s.includes?.join(', ')}</td>
                    <td className="px-4 py-3 text-sm">{formatSize(s.file_size)}</td>
                    <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${s.status === 'completed' ? 'bg-green-100 text-green-700' : s.status === 'creating' ? 'bg-yellow-100 text-yellow-700' : 'bg-red-100 text-red-700'}`}>{s.status}</span></td>
                    <td className="px-4 py-3 text-xs text-gray-500">{new Date(s.created_at).toLocaleString()}</td>
                    <td className="px-4 py-3 flex gap-1">
                      {s.status === 'completed' && <a href={securityApi.downloadSnapshot(s.id)} className="p-1 text-gray-400 hover:text-blue-600"><Download size={14} /></a>}
                      {s.status === 'completed' && <button onClick={() => { if (confirm('Rollback to this snapshot?')) securityApi.rollback(s.id); }} className="p-1 text-gray-400 hover:text-green-600"><RotateCw size={14} /></button>}
                      <button onClick={() => { securityApi.deleteSnapshot(s.id); securityApi.snapshots().then(r => setSnapshots(r.data.data || [])); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </td>
                  </tr>
                ))}
                {snapshots.length === 0 && <tr><td colSpan={6} className="px-6 py-8 text-center text-gray-400">No snapshots</td></tr>}
              </tbody>
            </table>
          </div>
        </div>
      )}
      {/* MALWARE SCANNER TAB */}
      {tab === 'malware' && (
        <div className="space-y-4">
          {/* Status card */}
          <div className="bg-white rounded-xl shadow-sm border p-5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${clamStatus?.installed ? 'bg-green-50' : 'bg-red-50'}`}>
                  <Bug size={20} className={clamStatus?.installed ? 'text-green-600' : 'text-red-500'} />
                </div>
                <div>
                  <h3 className="font-semibold text-sm">ClamAV Scanner</h3>
                  <p className="text-xs text-gray-400">
                    {clamStatus?.installed ? `Version: ${clamStatus.version}` : 'Not installed - run: apt install clamav'}
                  </p>
                </div>
                <span className={`ml-2 py-0.5 px-2 text-[10px] font-bold rounded-full ${clamStatus?.installed ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                  {clamStatus?.installed ? 'READY' : 'NOT INSTALLED'}
                </span>
                <span className="ml-1 py-0.5 px-2 text-[10px] font-bold rounded-full bg-blue-100 text-blue-700">PRO</span>
              </div>
              <button onClick={async () => { setUpdating(true); await api.post('/clamav/update').catch(() => {}); setUpdating(false); }}
                disabled={!clamStatus?.installed || updating}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium border rounded-lg hover:bg-gray-50 disabled:opacity-50">
                <RotateCw size={12} className={updating ? 'animate-spin' : ''} /> {updating ? 'Updating...' : 'Update DB'}
              </button>
            </div>
          </div>

          {/* Scan form */}
          <div className="bg-white rounded-xl shadow-sm border p-5">
            <h3 className="font-semibold text-sm mb-3">Scan Directory</h3>
            <div className="flex gap-2">
              <input type="text" value={scanPath} onChange={e => setScanPath(e.target.value)} placeholder="/www/sites"
                className="flex-1 py-2 px-3 border rounded-lg text-sm font-mono" />
              <button onClick={async () => {
                setScanning(true); setScanResult(null);
                try { const r = await api.post<{ data: ScanResult }>('/clamav/scan', { path: scanPath }); setScanResult(r.data.data); }
                catch { setScanResult({ path: scanPath, infected: -1, scanned: 0, threats: [], output: 'Scan failed' }); }
                setScanning(false);
              }} disabled={scanning || !clamStatus?.installed}
                className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
                {scanning ? <><div className="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" /> Scanning...</>
                  : <><Shield size={14} /> Start Scan</>}
              </button>
            </div>
            <div className="flex gap-2 mt-2">
              {['/www/sites', '/tmp', '/var/www', '/home'].map(p => (
                <button key={p} onClick={() => setScanPath(p)} className={`px-2 py-1 text-[10px] rounded border ${scanPath === p ? 'bg-blue-50 border-blue-300 text-blue-700' : 'border-gray-200 hover:bg-gray-50'}`}>{p}</button>
              ))}
            </div>
          </div>

          {/* Results */}
          {scanResult && (
            <div className={`rounded-xl shadow-sm border p-5 ${scanResult.infected > 0 ? 'bg-red-50 border-red-200' : 'bg-green-50 border-green-200'}`}>
              <div className="flex items-center gap-3 mb-3">
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${scanResult.infected > 0 ? 'bg-red-100' : 'bg-green-100'}`}>
                  {scanResult.infected > 0 ? <AlertTriangle size={20} className="text-red-600" /> : <Shield size={20} className="text-green-600" />}
                </div>
                <div>
                  <h3 className={`font-semibold text-sm ${scanResult.infected > 0 ? 'text-red-800' : 'text-green-800'}`}>
                    {scanResult.infected > 0 ? `${scanResult.infected} Threats Found!` : 'No Threats Found'}
                  </h3>
                  <p className="text-xs text-gray-500">Scanned {scanResult.scanned} files in {scanResult.path}</p>
                </div>
              </div>

              {scanResult.threats.length > 0 && (
                <div className="bg-white/70 rounded-lg p-3 mb-3 space-y-1">
                  <p className="text-xs font-semibold text-red-700 mb-2">Infected Files:</p>
                  {scanResult.threats.map((t, i) => (
                    <div key={i} className="text-xs font-mono text-red-600 flex items-center gap-2">
                      <AlertTriangle size={10} /> {t}
                    </div>
                  ))}
                </div>
              )}

              <details className="text-xs">
                <summary className="cursor-pointer text-gray-500 hover:text-gray-700">Raw output</summary>
                <pre className="mt-2 bg-gray-900 text-green-400 p-3 rounded-lg text-[10px] font-mono whitespace-pre-wrap max-h-60 overflow-auto">{scanResult.output}</pre>
              </details>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
