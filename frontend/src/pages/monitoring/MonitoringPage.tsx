import { useEffect, useState } from 'react';
import { monitoringApi, type Metrics } from '../../api/monitoring';
import { monitorAdvApi, type ContainerMetrics, type NetworkInterface, type DiskIO, type WebsiteBandwidth } from '../../api/monitoring-adv';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import { Cpu, Activity, HardDrive, Wifi, Box, Globe, RefreshCw } from 'lucide-react';

type Tab = 'overview' | 'containers' | 'network' | 'disk' | 'bandwidth';

const formatBytes = (b: number) => {
  if (!b) return '0 B';
  const u = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0, s = b;
  while (s >= 1024 && i < u.length - 1) { s /= 1024; i++; }
  return `${s.toFixed(1)} ${u[i]}`;
};

export default function MonitoringPage() {
  const [tab, setTab] = useState<Tab>('overview');
  const [current, setCurrent] = useState<Metrics | null>(null);
  const [history, setHistory] = useState<Metrics[]>([]);
  const [hours, setHours] = useState(24);
  const [containers, setContainers] = useState<ContainerMetrics[]>([]);
  const [network, setNetwork] = useState<NetworkInterface[]>([]);
  const [diskIO, setDiskIO] = useState<DiskIO[]>([]);
  const [bandwidth, setBandwidth] = useState<WebsiteBandwidth[]>([]);

  const fetch = () => {
    monitoringApi.current().then(r => setCurrent(r.data.data)).catch(() => {});
    monitoringApi.history(hours).then(r => setHistory(r.data.data || [])).catch(() => {});
  };

  useEffect(() => { fetch(); }, [hours]);
  useEffect(() => {
    if (tab === 'containers') monitorAdvApi.containers().then(r => setContainers(r.data.data || [])).catch(() => {});
    if (tab === 'network') monitorAdvApi.network().then(r => setNetwork(r.data.data || [])).catch(() => {});
    if (tab === 'disk') monitorAdvApi.diskIO().then(r => setDiskIO(r.data.data || [])).catch(() => {});
    if (tab === 'bandwidth') monitorAdvApi.bandwidth().then(r => setBandwidth(r.data.data || [])).catch(() => {});
  }, [tab]);

  const chartData = history.map(m => ({
    time: m.created_at ? new Date(m.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '',
    cpu: Math.round(m.cpu_percent * 10) / 10,
    mem: m.mem_total > 0 ? Math.round((m.mem_used / m.mem_total) * 1000) / 10 : 0,
    disk: m.disk_total > 0 ? Math.round((m.disk_used / m.disk_total) * 1000) / 10 : 0,
    net_in: m.net_in / 1024 / 1024,
    net_out: m.net_out / 1024 / 1024,
  }));

  const tabs: { key: Tab; label: string; icon: any }[] = [
    { key: 'overview', label: 'Overview', icon: Activity },
    { key: 'containers', label: 'Containers', icon: Box },
    { key: 'network', label: 'Network', icon: Wifi },
    { key: 'disk', label: 'Disk I/O', icon: HardDrive },
    { key: 'bandwidth', label: 'Bandwidth', icon: Globe },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold text-gray-900">Monitoring</h1>
        <div className="flex items-center gap-2">
          <select value={hours} onChange={e => setHours(Number(e.target.value))} className="px-3 py-1.5 border rounded-lg text-sm">
            <option value={6}>6 hours</option><option value={24}>24 hours</option><option value={72}>3 days</option>
            <option value={168}>7 days</option><option value={720}>30 days</option>
          </select>
          <button onClick={fetch} className="p-2 border rounded-lg hover:bg-gray-50"><RefreshCw size={14} /></button>
        </div>
      </div>

      <div className="border-b border-gray-200 mb-6">
        <nav className="flex gap-4">{tabs.map(t => (
          <button key={t.key} onClick={() => setTab(t.key)} className={`flex items-center gap-1 pb-3 text-sm font-medium border-b-2 ${tab === t.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'}`}>
            <t.icon size={14} /> {t.label}
          </button>
        ))}</nav>
      </div>

      {/* OVERVIEW */}
      {tab === 'overview' && (
        <div className="space-y-6">
          {/* Metric cards */}
          {current && (
            <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <div className="flex items-center gap-2 mb-2"><Cpu size={16} className="text-blue-500" /><span className="text-sm font-semibold text-gray-600">CPU</span></div>
                <p className="text-3xl font-bold">{current.cpu_percent.toFixed(1)}%</p>
                <p className="text-xs text-gray-400">{current.cpu_cores} cores &middot; Load: {current.load_1?.toFixed(2)}</p>
              </div>
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <div className="flex items-center gap-2 mb-2"><Activity size={16} className="text-green-500" /><span className="text-sm font-semibold text-gray-600">Memory</span></div>
                <p className="text-3xl font-bold">{current.mem_total > 0 ? ((current.mem_used / current.mem_total) * 100).toFixed(1) : 0}%</p>
                <p className="text-xs text-gray-400">{formatBytes(current.mem_used)} / {formatBytes(current.mem_total)}</p>
              </div>
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <div className="flex items-center gap-2 mb-2"><HardDrive size={16} className="text-purple-500" /><span className="text-sm font-semibold text-gray-600">Disk</span></div>
                <p className="text-3xl font-bold">{current.disk_total > 0 ? ((current.disk_used / current.disk_total) * 100).toFixed(1) : 0}%</p>
                <p className="text-xs text-gray-400">{formatBytes(current.disk_used)} / {formatBytes(current.disk_total)}</p>
              </div>
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <div className="flex items-center gap-2 mb-2"><Wifi size={16} className="text-cyan-500" /><span className="text-sm font-semibold text-gray-600">Network</span></div>
                <p className="text-lg font-bold">↓{formatBytes(current.net_in)}</p>
                <p className="text-lg font-bold">↑{formatBytes(current.net_out)}</p>
              </div>
            </div>
          )}

          {/* Charts */}
          {chartData.length > 1 && (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <h3 className="text-sm font-semibold text-gray-700 mb-4">CPU Usage</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={chartData}>
                    <XAxis dataKey="time" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                    <YAxis domain={[0, 100]} tick={{ fontSize: 10 }} unit="%" />
                    <Tooltip />
                    <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fill="#3b82f6" fillOpacity={0.1} strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <h3 className="text-sm font-semibold text-gray-700 mb-4">Memory Usage</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={chartData}>
                    <XAxis dataKey="time" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                    <YAxis domain={[0, 100]} tick={{ fontSize: 10 }} unit="%" />
                    <Tooltip />
                    <Area type="monotone" dataKey="mem" stroke="#10b981" fill="#10b981" fillOpacity={0.1} strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <h3 className="text-sm font-semibold text-gray-700 mb-4">Disk Usage</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={chartData}>
                    <XAxis dataKey="time" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                    <YAxis domain={[0, 100]} tick={{ fontSize: 10 }} unit="%" />
                    <Tooltip />
                    <Area type="monotone" dataKey="disk" stroke="#8b5cf6" fill="#8b5cf6" fillOpacity={0.1} strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
              <div className="bg-white rounded-xl p-5 shadow-sm border">
                <h3 className="text-sm font-semibold text-gray-700 mb-4">Network (MB)</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={chartData}>
                    <XAxis dataKey="time" tick={{ fontSize: 10 }} interval="preserveStartEnd" />
                    <YAxis tick={{ fontSize: 10 }} />
                    <Tooltip />
                    <Area type="monotone" dataKey="net_in" stroke="#06b6d4" fill="#06b6d4" fillOpacity={0.1} strokeWidth={2} name="RX" />
                    <Area type="monotone" dataKey="net_out" stroke="#f59e0b" fill="#f59e0b" fillOpacity={0.1} strokeWidth={2} name="TX" />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}
        </div>
      )}

      {/* CONTAINERS */}
      {tab === 'containers' && (
        <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Container</th><th className="px-4 py-3">CPU</th><th className="px-4 py-3">Memory</th><th className="px-4 py-3">Net I/O</th><th className="px-4 py-3">Block I/O</th><th className="px-4 py-3">PIDs</th></tr></thead>
            <tbody>
              {containers.map(c => (
                <tr key={c.id} className="border-b border-gray-50">
                  <td className="px-4 py-3"><p className="font-medium text-sm">{c.name}</p><p className="text-xs text-gray-400 font-mono">{c.id}</p></td>
                  <td className="px-4 py-3 text-sm font-mono">{c.cpu_percent}</td>
                  <td className="px-4 py-3 text-sm font-mono">{c.mem_usage}</td>
                  <td className="px-4 py-3 text-xs font-mono">{c.net_io}</td>
                  <td className="px-4 py-3 text-xs font-mono">{c.block_io}</td>
                  <td className="px-4 py-3 text-sm">{c.pids}</td>
                </tr>
              ))}
              {containers.length === 0 && <tr><td colSpan={6} className="px-6 py-8 text-center text-gray-400">No running containers</td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {/* NETWORK */}
      {tab === 'network' && (
        <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Interface</th><th className="px-4 py-3">RX (received)</th><th className="px-4 py-3">TX (sent)</th><th className="px-4 py-3">RX Packets</th><th className="px-4 py-3">TX Packets</th></tr></thead>
            <tbody>
              {network.map(n => (
                <tr key={n.name} className="border-b border-gray-50">
                  <td className="px-4 py-3 font-mono font-medium">{n.name}</td>
                  <td className="px-4 py-3 text-sm">{formatBytes(n.rx_bytes)}</td>
                  <td className="px-4 py-3 text-sm">{formatBytes(n.tx_bytes)}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">{n.rx_packets.toLocaleString()}</td>
                  <td className="px-4 py-3 text-sm text-gray-500">{n.tx_packets.toLocaleString()}</td>
                </tr>
              ))}
              {network.length === 0 && <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-400">No network data</td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {/* DISK I/O */}
      {tab === 'disk' && (
        <div className="space-y-4">
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Device</th><th className="px-4 py-3">Read</th><th className="px-4 py-3">Written</th><th className="px-4 py-3">Read Ops</th><th className="px-4 py-3">Write Ops</th></tr></thead>
              <tbody>
                {diskIO.map(d => (
                  <tr key={d.device} className="border-b border-gray-50">
                    <td className="px-4 py-3 font-mono font-medium">{d.device}</td>
                    <td className="px-4 py-3 text-sm">{formatBytes(d.read_bytes)}</td>
                    <td className="px-4 py-3 text-sm">{formatBytes(d.write_bytes)}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{d.read_ops.toLocaleString()}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{d.write_ops.toLocaleString()}</td>
                  </tr>
                ))}
                {diskIO.length === 0 && <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-400">No disk data</td></tr>}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* BANDWIDTH per website */}
      {tab === 'bandwidth' && (
        <div className="space-y-4">
          {bandwidth.length > 0 && (
            <div className="bg-white rounded-xl p-5 shadow-sm border">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">Bandwidth per Website</h3>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={bandwidth.map(b => ({ name: b.domain, bytes: b.bytes / 1024 / 1024 }))}>
                  <XAxis dataKey="name" tick={{ fontSize: 10 }} angle={-30} textAnchor="end" height={60} />
                  <YAxis tick={{ fontSize: 10 }} unit=" MB" />
                  <Tooltip formatter={(v: any) => `${Number(v).toFixed(1)} MB`} />
                  <Bar dataKey="bytes" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          )}
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Website</th><th className="px-4 py-3">Domain</th><th className="px-4 py-3">Bandwidth</th><th className="px-4 py-3">Requests</th></tr></thead>
              <tbody>
                {bandwidth.map(b => (
                  <tr key={b.website_id} className="border-b border-gray-50">
                    <td className="px-4 py-3 font-mono text-sm">{b.alias}</td>
                    <td className="px-4 py-3 text-sm">{b.domain}</td>
                    <td className="px-4 py-3 text-sm font-medium">{formatBytes(b.bytes)}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{b.requests.toLocaleString()}</td>
                  </tr>
                ))}
                {bandwidth.length === 0 && <tr><td colSpan={4} className="px-6 py-8 text-center text-gray-400">No bandwidth data</td></tr>}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
