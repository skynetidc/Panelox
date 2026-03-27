import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { dashboardApi, type DashboardOverview } from '../api/dashboard';
import { monitoringApi, type Metrics } from '../api/monitoring';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

const fmtB = (b: number) => { if (!b) return '0 B'; const u = ['B','KB','MB','GB','TB']; let i = 0, s = b; while (s >= 1024 && i < u.length - 1) { s /= 1024; i++; } return `${s.toFixed(1)} ${u[i]}`; };

const resourceCards = [
  { key: 'websites', label: 'dashboard.websites', icon: '🌐', color: 'blue', link: '/websites' },
  { key: 'ssl_certificates', label: 'dashboard.sslCerts', icon: '🔒', color: 'green', link: '/ssl' },
  { key: 'dns_zones', label: 'dashboard.dnsZones', icon: '📡', color: 'purple', link: '/dns' },
  { key: 'databases', label: 'dashboard.databases', icon: '💾', color: 'cyan', link: '/databases' },
  { key: 'mail_domains', label: 'dashboard.mailDomains', icon: '📧', color: 'amber', link: '/mail' },
  { key: 'firewall_rules', label: 'dashboard.firewallRules', icon: '🛡️', color: 'red', link: '/firewall' },
  { key: 'users', label: 'dashboard.users', icon: '👥', color: 'indigo', link: '/billing' },
];

export default function Dashboard() {
  const { t } = useTranslation();
  const [ov, setOv] = useState<DashboardOverview | null>(null);
  const [cur, setCur] = useState<Metrics | null>(null);
  const [hist, setHist] = useState<Metrics[]>([]);
  useEffect(() => {
    dashboardApi.overview().then(r => setOv(r.data.data)).catch(() => {});
    monitoringApi.current().then(r => setCur(r.data.data)).catch(() => {});
    monitoringApi.history(6).then(r => setHist(r.data.data || [])).catch(() => {});
  }, []);
  const cd = hist.map(m => ({ time: m.created_at ? new Date(m.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '', cpu: Math.round(m.cpu_percent * 10) / 10, mem: m.mem_total > 0 ? Math.round((m.mem_used / m.mem_total) * 1000) / 10 : 0 }));

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-800">{t('dashboard.title')}</h1>
        <p className="text-sm text-gray-500 mt-1">{cur?.hostname || 'Server'} · {cur?.uptime || '—'} uptime · {cur?.kernel || ''}</p>
      </div>

      {/* System Metrics */}
      <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        {[
          { l: 'CPU', v: `${cur?.cpu_percent?.toFixed(1) || 0}%`, s: `${cur?.cpu_cores || 0} cores`, c: 'blue' },
          { l: 'Memory', v: `${cur && cur.mem_total > 0 ? ((cur.mem_used / cur.mem_total) * 100).toFixed(1) : 0}%`, s: `${fmtB(cur?.mem_used || 0)} / ${fmtB(cur?.mem_total || 0)}`, c: 'green' },
          { l: 'Disk', v: `${cur && cur.disk_total > 0 ? ((cur.disk_used / cur.disk_total) * 100).toFixed(1) : 0}%`, s: `${fmtB(cur?.disk_used || 0)} / ${fmtB(cur?.disk_total || 0)}`, c: 'purple' },
          { l: 'Network', v: `↓${fmtB(cur?.net_in || 0)}`, s: `↑${fmtB(cur?.net_out || 0)}`, c: 'cyan' },
        ].map(m => (
          <div key={m.l} className="bg-white border border-gray-200 rounded-xl shadow-sm p-5">
            <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-2">{m.l}</p>
            <p className={`text-3xl font-bold text-${m.c}-600`}>{m.v}</p>
            <p className="text-xs text-gray-400 mt-1 font-mono">{m.s}</p>
          </div>
        ))}
      </div>

      {/* Charts */}
      {cd.length > 1 && (
        <div className="grid lg:grid-cols-2 gap-4 mb-6">
          {[{ l: 'CPU Usage', k: 'cpu', c: '#3b82f6' }, { l: 'Memory Usage', k: 'mem', c: '#10b981' }].map(ch => (
            <div key={ch.k} className="bg-white border border-gray-200 rounded-xl shadow-sm p-5">
              <p className="text-sm font-semibold text-gray-600 mb-4">{ch.l}</p>
              <ResponsiveContainer width="100%" height={180}>
                <AreaChart data={cd}>
                  <defs><linearGradient id={`g${ch.k}`} x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor={ch.c} stopOpacity={0.15} /><stop offset="100%" stopColor={ch.c} stopOpacity={0} /></linearGradient></defs>
                  <XAxis dataKey="time" tick={{ fontSize: 10 }} stroke="#e5e7eb" />
                  <YAxis domain={[0, 100]} tick={{ fontSize: 10 }} unit="%" stroke="#e5e7eb" />
                  <Tooltip contentStyle={{ borderRadius: 8, fontSize: 12, border: '1px solid #e5e7eb' }} />
                  <Area type="monotone" dataKey={ch.k} stroke={ch.c} fill={`url(#g${ch.k})`} strokeWidth={2} dot={false} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          ))}
        </div>
      )}

      {/* Resources */}
      <p className="text-xs font-bold text-gray-400 uppercase tracking-widest mb-3">Resources</p>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
        {resourceCards.map(c => (
          <Link to={c.link} key={c.key} className="group bg-white border border-gray-200 rounded-xl shadow-sm p-4 hover:shadow-md hover:-translate-y-0.5 transition-all">
            <span className="text-2xl">{c.icon}</span>
            <p className="text-2xl font-bold text-gray-800 mt-2">{ov ? (ov as any)[c.key] ?? 0 : '—'}</p>
            <p className="text-xs text-gray-500 mt-0.5">{t(c.label)}</p>
          </Link>
        ))}
      </div>
    </div>
  );
}
