import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { websiteApi, type Website, type WebsiteDomain } from '../../api/websites';
import { websiteAdvancedApi, type Redirect } from '../../api/php';
import { ArrowLeft, Play, Square, Plus, Trash2, Save, RefreshCw } from 'lucide-react';

type Tab = 'overview' | 'domains' | 'nginx' | 'logs' | 'security' | 'redirects' | 'rewrite' | 'loadbalance' | 'snippets' | 'ratelimit' | 'ipaccess';

interface NginxSnippet { id: number; website_id: number; preset_key: string; name: string; category: string; content: string; enabled: boolean; created_at: string; }
interface SnippetPreset { key: string; name: string; category: string; content: string; }
interface RateLimitCfg { enabled: boolean; rps: number; burst: number; nodelay: boolean; }
interface IPRuleEntry { id: number; website_id: number; mode: string; ip_address: string; description: string; created_at: string; }

export default function WebsiteDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [website, setWebsite] = useState<Website | null>(null);
  const [domains, setDomains] = useState<WebsiteDomain[]>([]);
  const [nginxConfig, setNginxConfig] = useState('');
  const [tab, setTab] = useState<Tab>('overview');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  // New domain form
  const [newDomain, setNewDomain] = useState('');
  const [newPort, setNewPort] = useState(80);

  // Logs
  const [accessLog, setAccessLog] = useState('');
  const [errorLog, setErrorLog] = useState('');

  // Redirects
  const [redirects, setRedirects] = useState<Redirect[]>([]);
  const [newRedirect, setNewRedirect] = useState({ type: '301', source: '', target: '' });

  // Security
  const [security, setSecurity] = useState({
    hsts_enabled: false, hsts_max_age: 31536000, hsts_subdomains: false, http3_enabled: false,
    basic_auth_enabled: false, basic_auth_user: '', basic_auth_pass: '',
    anti_hotlink: false, hotlink_extensions: 'jpg,jpeg,png,gif,svg,webp', hotlink_allowed_domains: '',
    cors_enabled: false, cors_origins: '*', cors_methods: 'GET,POST,PUT,DELETE,OPTIONS', cors_headers: 'Content-Type,Authorization',
  });

  // Rewrite
  const [rewriteRules, setRewriteRules] = useState('');
  const [newRewrite, setNewRewrite] = useState({ source: '', target: '', type: 'rewrite', flag: 'last' });

  // Load Balancer
  const [lbAlgorithm, setLbAlgorithm] = useState('round-robin');
  const [lbServers, setLbServers] = useState<string[]>([]);
  const [newLbServer, setNewLbServer] = useState('');

  // Snippets
  const [snippets, setSnippets] = useState<NginxSnippet[]>([]);
  const [presets, setPresets] = useState<SnippetPreset[]>([]);
  const [snippetCategory, setSnippetCategory] = useState('');

  // Rate limit
  const [rateLimit, setRateLimit] = useState<RateLimitCfg>({ enabled: false, rps: 10, burst: 20, nodelay: true });

  // IP Access
  const [ipRules, setIpRules] = useState<IPRuleEntry[]>([]);
  const [ipMode, setIpMode] = useState('off');
  const [newIP, setNewIP] = useState({ ip_address: '', description: '' });

  // Clone
  const [showClone, setShowClone] = useState(false);
  const [cloneDomain, setCloneDomain] = useState('');
  const [cloning, setCloning] = useState(false);

  const websiteId = Number(id);

  const fetchWebsite = async () => {
    setLoading(true);
    try {
      const res = await websiteApi.get(websiteId);
      setWebsite(res.data.data);
      setDomains(res.data.data.domains || []);
    } catch {
      navigate('/websites');
    } finally {
      setLoading(false);
    }
  };

  const fetchNginxConfig = async () => {
    try {
      const res = await websiteApi.getNginxConfig(websiteId);
      setNginxConfig(res.data.data.config);
    } catch { /* ignore */ }
  };

  useEffect(() => {
    fetchWebsite();
  }, [websiteId]);

  useEffect(() => {
    if (tab === 'nginx' && !nginxConfig) fetchNginxConfig();
    if (tab === 'domains') {
      websiteApi.listDomains(websiteId).then((r) => setDomains(r.data.data || [])).catch(() => {});
    }
    if (tab === 'logs') {
      websiteAdvancedApi.accessLog(websiteId, 200).then(r => setAccessLog(r.data.data.logs)).catch(() => {});
      websiteAdvancedApi.errorLog(websiteId, 200).then(r => setErrorLog(r.data.data.logs)).catch(() => {});
    }
    if (tab === 'redirects') {
      websiteAdvancedApi.listRedirects(websiteId).then(r => setRedirects(r.data.data || [])).catch(() => {});
    }
    if (tab === 'rewrite') {
      import('../../api/client').then(({ default: api }) => {
        api.get(`/websites/${websiteId}/rewrite`).then(r => setRewriteRules(r.data.data?.rules || '')).catch(() => {});
      });
    }
    if (tab === 'loadbalance') {
      import('../../api/client').then(({ default: api }) => {
        api.get(`/websites/${websiteId}/loadbalance`).then(r => setRewriteRules(r.data.data?.config || '')).catch(() => {});
      });
    }
    if (tab === 'snippets') {
      import('../../api/client').then(({ default: api }) => {
        api.get(`/websites/${websiteId}/snippets`).then(r => setSnippets(r.data.data || [])).catch(() => {});
        api.get('/websites/snippet-presets').then(r => setPresets(r.data.data || [])).catch(() => {});
      });
    }
    if (tab === 'ratelimit') {
      import('../../api/client').then(({ default: api }) => {
        api.get(`/websites/${websiteId}/rate-limit`).then(r => setRateLimit(r.data.data)).catch(() => {});
      });
    }
    if (tab === 'ipaccess') {
      import('../../api/client').then(({ default: api }) => {
        api.get(`/websites/${websiteId}/ip-rules`).then(r => { setIpRules(r.data.data?.rules || []); setIpMode(r.data.data?.mode || 'off'); }).catch(() => {});
      });
    }
  }, [tab]);

  const handleOperate = async (action: 'start' | 'stop') => {
    await websiteApi.operate(websiteId, action);
    fetchWebsite();
  };

  const handleSaveNginx = async () => {
    setSaving(true);
    try {
      await websiteApi.updateNginxConfig(websiteId, nginxConfig);
    } catch { /* ignore */ }
    setSaving(false);
  };

  const handleAddDomain = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomain) return;
    try {
      await websiteApi.addDomain(websiteId, newDomain, newPort);
      setNewDomain('');
      setNewPort(80);
      const res = await websiteApi.listDomains(websiteId);
      setDomains(res.data.data || []);
    } catch { /* ignore */ }
  };

  const handleDeleteDomain = async (domainId: number) => {
    await websiteApi.deleteDomain(websiteId, domainId);
    const res = await websiteApi.listDomains(websiteId);
    setDomains(res.data.data || []);
  };

  if (loading || !website) {
    return <div className="text-center text-gray-400 py-12">{t('common.loading')}</div>;
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'domains', label: 'Domains' },
    { key: 'nginx', label: 'Nginx Config' },
    { key: 'logs', label: 'Logs' },
    { key: 'security', label: 'Security' },
    { key: 'redirects', label: 'Redirects' },
    { key: 'rewrite', label: 'URL Rewrite' },
    { key: 'loadbalance', label: 'Load Balancer' },
    { key: 'snippets', label: 'Snippets' },
    { key: 'ratelimit', label: 'Rate Limit' },
    { key: 'ipaccess', label: 'IP Access' },
  ];

  return (
    <div>
      <button
        onClick={() => navigate('/websites')}
        className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4"
      >
        <ArrowLeft size={16} />
        {t('common.back')}
      </button>

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{website.domain}</h1>
          <p className="text-sm text-gray-500">
            {website.type} &middot; {website.alias} &middot; {website.document_root}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm ${
            website.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'
          }`}>
            <span className={`w-2 h-2 rounded-full ${
              website.status === 'active' ? 'bg-green-500' : 'bg-gray-400'
            }`} />
            {website.status}
          </span>
          <button onClick={() => setShowClone(true)} className="flex items-center gap-1 px-3 py-1.5 text-sm border border-purple-300 text-purple-600 rounded-lg hover:bg-purple-50">
            <RefreshCw size={14} /> Clone
          </button>
          {website.status === 'active' ? (
            <button
              onClick={() => handleOperate('stop')}
              className="flex items-center gap-1 px-3 py-1.5 text-sm border border-orange-300 text-orange-600 rounded-lg hover:bg-orange-50"
            >
              <Square size={14} /> Stop
            </button>
          ) : (
            <button
              onClick={() => handleOperate('start')}
              className="flex items-center gap-1 px-3 py-1.5 text-sm border border-green-300 text-green-600 rounded-lg hover:bg-green-50"
            >
              <Play size={14} /> Start
            </button>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 mb-6">
        <nav className="flex gap-6">
          {tabs.map((t) => (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`pb-3 text-sm font-medium border-b-2 transition-colors ${
                tab === t.key
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {t.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      {tab === 'overview' && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
          <div className="grid grid-cols-2 gap-6">
            <div>
              <label className="text-sm text-gray-500">Domain</label>
              <p className="font-medium">{website.domain}</p>
            </div>
            <div>
              <label className="text-sm text-gray-500">Type</label>
              <p className="font-medium">{website.type}</p>
            </div>
            <div>
              <label className="text-sm text-gray-500">Document Root</label>
              <p className="font-medium font-mono text-sm">{website.document_root}</p>
            </div>
            {website.proxy_target && (
              <div>
                <label className="text-sm text-gray-500">Proxy Target</label>
                <p className="font-medium font-mono text-sm">{website.proxy_target}</p>
              </div>
            )}
            {website.type === 'php' && (
              <div>
                <label className="text-sm text-gray-500">PHP Version</label>
                <div className="flex items-center gap-2 mt-1">
                  <select
                    value={website.php_version || '8.3'}
                    onChange={async (e) => {
                      const newVer = e.target.value;
                      await websiteApi.update(websiteId, { php_version: newVer });
                      fetchWebsite();
                    }}
                    className="px-3 py-1.5 border rounded-lg text-sm font-medium"
                  >
                    <option value="8.3">PHP 8.3</option>
                    <option value="8.2">PHP 8.2</option>
                    <option value="8.1">PHP 8.1</option>
                    <option value="8.0">PHP 8.0</option>
                    <option value="7.4">PHP 7.4</option>
                  </select>
                  <span className="text-xs text-gray-400">Port: {
                    {'8.3': 9000, '8.2': 9001, '8.1': 9002, '8.0': 9003, '7.4': 9004}[website.php_version || '8.3'] || 9000
                  }</span>
                </div>
              </div>
            )}
            <div>
              <label className="text-sm text-gray-500">SSL</label>
              <p className="font-medium">{website.ssl_enabled ? 'Enabled' : 'Disabled'}</p>
            </div>
            <div>
              <label className="text-sm text-gray-500">IPv6</label>
              <p className="font-medium">{website.ipv6 ? 'Enabled' : 'Disabled'}</p>
            </div>
            <div>
              <label className="text-sm text-gray-500">Created</label>
              <p className="font-medium">{new Date(website.created_at).toLocaleDateString()}</p>
            </div>
          </div>
        </div>
      )}

      {tab === 'domains' && (
        <div className="space-y-4">
          {/* Add domain form */}
          <form onSubmit={handleAddDomain} className="bg-white rounded-xl shadow-sm border border-gray-100 p-4 flex gap-3 items-end">
            <div className="flex-1">
              <label className="block text-sm text-gray-500 mb-1">Domain</label>
              <input
                type="text"
                value={newDomain}
                onChange={(e) => setNewDomain(e.target.value)}
                placeholder="www.example.com"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div className="w-28">
              <label className="block text-sm text-gray-500 mb-1">Port</label>
              <input
                type="number"
                value={newPort}
                onChange={(e) => setNewPort(Number(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <button
              type="submit"
              className="flex items-center gap-1 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700"
            >
              <Plus size={16} /> Add
            </button>
          </form>

          {/* Domain list */}
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 text-left text-sm text-gray-500">
                  <th className="px-6 py-3">Domain</th>
                  <th className="px-6 py-3">Port</th>
                  <th className="px-6 py-3">SSL</th>
                  <th className="px-6 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {domains.map((d) => (
                  <tr key={d.id} className="border-b border-gray-50">
                    <td className="px-6 py-3 font-medium">{d.domain}</td>
                    <td className="px-6 py-3 text-gray-600">{d.port}</td>
                    <td className="px-6 py-3">
                      {d.ssl ? (
                        <span className="text-green-600 text-sm">Yes</span>
                      ) : (
                        <span className="text-gray-400 text-sm">No</span>
                      )}
                    </td>
                    <td className="px-6 py-3">
                      <button
                        onClick={() => handleDeleteDomain(d.id)}
                        className="p-1 text-gray-400 hover:text-red-600"
                      >
                        <Trash2 size={16} />
                      </button>
                    </td>
                  </tr>
                ))}
                {domains.length === 0 && (
                  <tr>
                    <td colSpan={4} className="px-6 py-8 text-center text-gray-400">
                      No domains configured
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'nginx' && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
          <div className="flex items-center justify-between mb-3">
            <label className="text-sm font-medium text-gray-700">Nginx Configuration</label>
            <button
              onClick={handleSaveNginx}
              disabled={saving}
              className="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              <Save size={14} />
              {saving ? 'Saving...' : 'Save & Reload'}
            </button>
          </div>
          <textarea
            value={nginxConfig}
            onChange={(e) => setNginxConfig(e.target.value)}
            className="w-full h-96 font-mono text-sm p-4 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none resize-y"
            spellCheck={false}
          />
          <p className="text-xs text-gray-400 mt-2">
            Saving will validate the config with nginx -t before applying.
          </p>
        </div>
      )}

      {tab === 'logs' && (
        <div className="space-y-4">
          <div className="bg-white rounded-xl shadow-sm border p-4">
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-semibold">Access Log</h3>
              <button onClick={() => websiteAdvancedApi.accessLog(websiteId, 200).then(r => setAccessLog(r.data.data.logs))} className="p-1 text-gray-400 hover:text-blue-600"><RefreshCw size={14} /></button>
            </div>
            <pre className="bg-gray-900 text-green-400 text-xs font-mono p-4 rounded-lg max-h-64 overflow-y-auto whitespace-pre-wrap">{accessLog || 'No logs'}</pre>
          </div>
          <div className="bg-white rounded-xl shadow-sm border p-4">
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-semibold">Error Log</h3>
              <button onClick={() => websiteAdvancedApi.errorLog(websiteId, 200).then(r => setErrorLog(r.data.data.logs))} className="p-1 text-gray-400 hover:text-blue-600"><RefreshCw size={14} /></button>
            </div>
            <pre className="bg-gray-900 text-red-400 text-xs font-mono p-4 rounded-lg max-h-64 overflow-y-auto whitespace-pre-wrap">{errorLog || 'No logs'}</pre>
          </div>
        </div>
      )}

      {tab === 'security' && (
        <div className="bg-white rounded-xl shadow-sm border p-6 space-y-6 max-w-2xl">
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">HSTS (HTTP Strict Transport Security)</h3>
            <label className="flex items-center gap-2 mb-2">
              <input type="checkbox" checked={security.hsts_enabled} onChange={e => setSecurity(s => ({ ...s, hsts_enabled: e.target.checked }))} />
              <span className="text-sm">Enable HSTS</span>
            </label>
            {security.hsts_enabled && (
              <div className="flex gap-4 ml-6">
                <div>
                  <label className="text-xs text-gray-500">Max Age (sec)</label>
                  <input type="number" value={security.hsts_max_age} onChange={e => setSecurity(s => ({ ...s, hsts_max_age: Number(e.target.value) }))} className="w-32 px-2 py-1 border rounded text-sm" />
                </div>
                <label className="flex items-center gap-1 text-sm">
                  <input type="checkbox" checked={security.hsts_subdomains} onChange={e => setSecurity(s => ({ ...s, hsts_subdomains: e.target.checked }))} />
                  includeSubDomains
                </label>
              </div>
            )}
          </div>

          <div>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={security.http3_enabled} onChange={e => setSecurity(s => ({ ...s, http3_enabled: e.target.checked }))} />
              <span className="text-sm">Enable HTTP/3 (QUIC)</span>
            </label>
          </div>

          <hr />
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">HTTP Basic Auth</h3>
            <label className="flex items-center gap-2 mb-2">
              <input type="checkbox" checked={security.basic_auth_enabled} onChange={e => setSecurity(s => ({ ...s, basic_auth_enabled: e.target.checked }))} />
              <span className="text-sm">Enable Basic Auth</span>
            </label>
            {security.basic_auth_enabled && (
              <div className="flex gap-3 ml-6">
                <input type="text" value={security.basic_auth_user} onChange={e => setSecurity(s => ({ ...s, basic_auth_user: e.target.value }))} placeholder="Username" className="px-2 py-1 border rounded text-sm" />
                <input type="password" value={security.basic_auth_pass} onChange={e => setSecurity(s => ({ ...s, basic_auth_pass: e.target.value }))} placeholder="Password" className="px-2 py-1 border rounded text-sm" />
              </div>
            )}
          </div>

          <hr />
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">Anti-Hotlink</h3>
            <label className="flex items-center gap-2 mb-2">
              <input type="checkbox" checked={security.anti_hotlink} onChange={e => setSecurity(s => ({ ...s, anti_hotlink: e.target.checked }))} />
              <span className="text-sm">Enable Hotlink Protection</span>
            </label>
            {security.anti_hotlink && (
              <div className="space-y-2 ml-6">
                <div>
                  <label className="text-xs text-gray-500">Protected extensions</label>
                  <input type="text" value={security.hotlink_extensions} onChange={e => setSecurity(s => ({ ...s, hotlink_extensions: e.target.value }))} className="w-full px-2 py-1 border rounded text-sm" />
                </div>
                <div>
                  <label className="text-xs text-gray-500">Allowed domains (comma-separated)</label>
                  <input type="text" value={security.hotlink_allowed_domains} onChange={e => setSecurity(s => ({ ...s, hotlink_allowed_domains: e.target.value }))} className="w-full px-2 py-1 border rounded text-sm" placeholder="example.com,cdn.example.com" />
                </div>
              </div>
            )}
          </div>

          <hr />
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">CORS</h3>
            <label className="flex items-center gap-2 mb-2">
              <input type="checkbox" checked={security.cors_enabled} onChange={e => setSecurity(s => ({ ...s, cors_enabled: e.target.checked }))} />
              <span className="text-sm">Enable CORS headers</span>
            </label>
            {security.cors_enabled && (
              <div className="space-y-2 ml-6">
                <div><label className="text-xs text-gray-500">Allowed Origins</label><input type="text" value={security.cors_origins} onChange={e => setSecurity(s => ({ ...s, cors_origins: e.target.value }))} className="w-full px-2 py-1 border rounded text-sm" /></div>
                <div><label className="text-xs text-gray-500">Methods</label><input type="text" value={security.cors_methods} onChange={e => setSecurity(s => ({ ...s, cors_methods: e.target.value }))} className="w-full px-2 py-1 border rounded text-sm" /></div>
              </div>
            )}
          </div>

          <button onClick={() => websiteAdvancedApi.updateSecurity(websiteId, security)} className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700">
            <Save size={14} /> Save Security Settings
          </button>
        </div>
      )}

      {tab === 'redirects' && (
        <div className="space-y-4">
          <form onSubmit={e => { e.preventDefault(); if (newRedirect.source && newRedirect.target) { websiteAdvancedApi.createRedirect(websiteId, newRedirect).then(() => { setNewRedirect({ type: '301', source: '', target: '' }); websiteAdvancedApi.listRedirects(websiteId).then(r => setRedirects(r.data.data || [])); }); } }} className="bg-white rounded-xl shadow-sm border p-4 flex gap-3 items-end">
            <div className="w-20"><label className="text-xs text-gray-500">Type</label><select value={newRedirect.type} onChange={e => setNewRedirect(r => ({ ...r, type: e.target.value }))} className="w-full px-2 py-1.5 border rounded text-sm"><option value="301">301</option><option value="302">302</option></select></div>
            <div className="flex-1"><label className="text-xs text-gray-500">Source Path</label><input type="text" value={newRedirect.source} onChange={e => setNewRedirect(r => ({ ...r, source: e.target.value }))} placeholder="/old-page" className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <div className="flex-1"><label className="text-xs text-gray-500">Target URL</label><input type="text" value={newRedirect.target} onChange={e => setNewRedirect(r => ({ ...r, target: e.target.value }))} placeholder="/new-page or https://..." className="w-full px-2 py-1.5 border rounded text-sm outline-none" /></div>
            <button type="submit" className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Add</button>
          </form>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Type</th><th className="px-4 py-3">Source</th><th className="px-4 py-3">Target</th><th className="px-4 py-3"></th></tr></thead>
              <tbody>
                {redirects.map(rd => (
                  <tr key={rd.id} className="border-b border-gray-50">
                    <td className="px-4 py-3"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{rd.type}</span></td>
                    <td className="px-4 py-3 font-mono text-sm">{rd.source}</td>
                    <td className="px-4 py-3 font-mono text-sm text-gray-600">{rd.target}</td>
                    <td className="px-4 py-3"><button onClick={() => { websiteAdvancedApi.deleteRedirect(websiteId, rd.id).then(() => websiteAdvancedApi.listRedirects(websiteId).then(r => setRedirects(r.data.data || []))); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button></td>
                  </tr>
                ))}
                {redirects.length === 0 && <tr><td colSpan={4} className="px-6 py-8 text-center text-gray-400">No redirects</td></tr>}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* URL REWRITE */}
      {tab === 'rewrite' && (
        <div className="space-y-4">
          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <h3 className="font-semibold mb-3">Add Rewrite Rule</h3>
            <div className="grid grid-cols-5 gap-3 items-end">
              <div><label className="text-xs text-gray-500">Source (regex)</label><input type="text" value={newRewrite.source} onChange={e => setNewRewrite(r => ({ ...r, source: e.target.value }))} placeholder="^/old-page$" className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
              <div><label className="text-xs text-gray-500">Target</label><input type="text" value={newRewrite.target} onChange={e => setNewRewrite(r => ({ ...r, target: e.target.value }))} placeholder="/new-page" className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" /></div>
              <div><label className="text-xs text-gray-500">Type</label><select value={newRewrite.type} onChange={e => setNewRewrite(r => ({ ...r, type: e.target.value }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm"><option value="rewrite">Rewrite</option><option value="redirect">Redirect (302)</option><option value="permanent">Permanent (301)</option></select></div>
              <div><label className="text-xs text-gray-500">Flag</label><select value={newRewrite.flag} onChange={e => setNewRewrite(r => ({ ...r, flag: e.target.value }))} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm"><option value="last">last</option><option value="break">break</option><option value="redirect">redirect</option><option value="permanent">permanent</option></select></div>
              <button onClick={async () => {
                const api = (await import('../../api/client')).default;
                await api.put(`/websites/${websiteId}/rewrite`, { rules: [newRewrite] });
                setNewRewrite({ source: '', target: '', type: 'rewrite', flag: 'last' });
                api.get(`/websites/${websiteId}/rewrite`).then(r => setRewriteRules(r.data.data?.rules || ''));
              }} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">Add</button>
            </div>
          </div>

          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-semibold">Current Rules</h3>
              <span className="text-xs text-gray-400 font-mono">/www/sites/{website?.alias}/rewrite/rules.conf</span>
            </div>
            <pre className="bg-gray-900 text-green-400 text-xs font-mono p-4 rounded-lg whitespace-pre-wrap min-h-[100px]">{rewriteRules || '# No rewrite rules configured'}</pre>
          </div>
        </div>
      )}

      {/* LOAD BALANCER */}
      {tab === 'loadbalance' && (
        <div className="space-y-4">
          <div className="bg-white border border-gray-200 rounded-xl p-5 max-w-lg">
            <h3 className="font-semibold mb-4">Upstream Configuration</h3>

            <div className="mb-4">
              <label className="text-sm text-gray-600 font-medium">Algorithm</label>
              <select value={lbAlgorithm} onChange={e => setLbAlgorithm(e.target.value)} className="mt-1 w-full py-2 px-3 border border-gray-200 rounded-lg text-sm">
                <option value="round-robin">Round Robin (default)</option>
                <option value="least_conn">Least Connections</option>
                <option value="ip_hash">IP Hash (sticky sessions)</option>
              </select>
            </div>

            <div className="mb-4">
              <label className="text-sm text-gray-600 font-medium">Backend Servers</label>
              <div className="space-y-2 mt-2">
                {lbServers.map((srv, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <span className="flex-1 py-1.5 px-3 bg-gray-50 rounded-lg text-sm font-mono">{srv}</span>
                    <button onClick={() => setLbServers(s => s.filter((_, j) => j !== i))} className="text-red-500 text-xs hover:underline">Remove</button>
                  </div>
                ))}
              </div>
              <div className="flex gap-2 mt-2">
                <input type="text" value={newLbServer} onChange={e => setNewLbServer(e.target.value)} placeholder="127.0.0.1:3000" className="flex-1 py-2 px-3 border border-gray-200 rounded-lg text-sm font-mono" />
                <button onClick={() => { if (newLbServer) { setLbServers(s => [...s, newLbServer]); setNewLbServer(''); } }} className="py-2 px-3 text-sm rounded-lg border border-gray-200 hover:bg-gray-50">Add</button>
              </div>
            </div>

            <button onClick={async () => {
              const api = (await import('../../api/client')).default;
              await api.put(`/websites/${websiteId}/loadbalance`, { algorithm: lbAlgorithm, servers: lbServers });
              alert('Load balancer saved');
            }} className="w-full py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">
              <Save size={14} className="inline mr-1" /> Save Load Balancer
            </button>
          </div>

          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <h3 className="font-semibold mb-2">Generated Nginx Upstream</h3>
            <pre className="bg-gray-900 text-green-400 text-xs font-mono p-4 rounded-lg whitespace-pre-wrap">{
              lbServers.length > 0
                ? `upstream ${website?.alias}_backend {\n${lbAlgorithm !== 'round-robin' ? `    ${lbAlgorithm};\n` : ''}${lbServers.map(s => `    server ${s};`).join('\n')}\n}`
                : '# No upstream configured'
            }</pre>
          </div>
        </div>
      )}
      {/* ── Snippets Tab ── */}
      {tab === 'snippets' && (
        <div className="space-y-4">
          {/* Active snippets */}
          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <h3 className="font-semibold mb-3">Active Snippets</h3>
            {snippets.length === 0 ? (
              <p className="text-sm text-gray-400">No snippets added. Choose from the library below.</p>
            ) : (
              <div className="space-y-2">
                {snippets.map(s => (
                  <div key={s.id} className={`border rounded-lg p-3 ${s.enabled ? 'border-green-200 bg-green-50/50' : 'border-gray-200 bg-gray-50'}`}>
                    <div className="flex items-center justify-between mb-1">
                      <div className="flex items-center gap-2">
                        <span className={`w-2 h-2 rounded-full ${s.enabled ? 'bg-green-500' : 'bg-gray-400'}`} />
                        <span className="text-sm font-medium">{s.name}</span>
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-500">{s.category}</span>
                      </div>
                      <div className="flex gap-1">
                        <button onClick={async () => { const api = (await import('../../api/client')).default; await api.put(`/websites/${websiteId}/snippets/${s.id}/toggle`); setSnippets(prev => prev.map(x => x.id === s.id ? { ...x, enabled: !x.enabled } : x)); }}
                          className={`px-2 py-1 text-xs rounded ${s.enabled ? 'text-amber-600 hover:bg-amber-50' : 'text-green-600 hover:bg-green-50'}`}>
                          {s.enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button onClick={async () => { const api = (await import('../../api/client')).default; await api.delete(`/websites/${websiteId}/snippets/${s.id}`); setSnippets(prev => prev.filter(x => x.id !== s.id)); }}
                          className="px-2 py-1 text-xs text-red-500 hover:bg-red-50 rounded">Remove</button>
                      </div>
                    </div>
                    <pre className="text-[10px] font-mono text-gray-500 whitespace-pre-wrap max-h-20 overflow-hidden">{s.content}</pre>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Preset library */}
          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <h3 className="font-semibold mb-3">Snippet Library <span className="text-xs font-normal text-blue-600 ml-1">PRO</span></h3>
            <div className="flex gap-2 mb-3 flex-wrap">
              <button onClick={() => setSnippetCategory('')} className={`px-2.5 py-1 text-xs rounded-full ${!snippetCategory ? 'bg-gray-800 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}>All</button>
              {[...new Set(presets.map(p => p.category))].sort().map(c => (
                <button key={c} onClick={() => setSnippetCategory(snippetCategory === c ? '' : c)}
                  className={`px-2.5 py-1 text-xs rounded-full capitalize ${snippetCategory === c ? 'bg-gray-800 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}>{c}</button>
              ))}
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
              {presets.filter(p => !snippetCategory || p.category === snippetCategory).map(p => {
                const added = snippets.some(s => s.preset_key === p.key);
                return (
                  <div key={p.key} className="border border-gray-100 rounded-lg p-3 hover:border-gray-300 transition-colors">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium">{p.name}</span>
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-100 text-gray-500">{p.category}</span>
                    </div>
                    <pre className="text-[10px] font-mono text-gray-400 whitespace-pre-wrap max-h-16 overflow-hidden mb-2">{p.content.substring(0, 150)}...</pre>
                    <button onClick={async () => {
                      if (added) return;
                      const api = (await import('../../api/client')).default;
                      const r = await api.post(`/websites/${websiteId}/snippets`, { preset_key: p.key });
                      setSnippets(prev => [...prev, r.data.data]);
                    }} disabled={added} className={`text-xs px-3 py-1 rounded ${added ? 'bg-green-50 text-green-600' : 'bg-blue-600 text-white hover:bg-blue-700'}`}>
                      {added ? 'Added' : 'Add'}
                    </button>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}

      {/* ── Rate Limit Tab ── */}
      {tab === 'ratelimit' && (
        <div className="bg-white border border-gray-200 rounded-xl p-6 max-w-lg space-y-5">
          <div>
            <h3 className="font-semibold">Rate Limiting <span className="text-xs font-normal text-blue-600 ml-1">PRO</span></h3>
            <p className="text-xs text-gray-500 mt-1">Limit requests per IP to protect against abuse and DDoS.</p>
          </div>

          <label className="flex items-center justify-between p-3 bg-gray-50 rounded-lg cursor-pointer">
            <div>
              <p className="text-sm font-medium text-gray-700">Enable Rate Limiting</p>
              <p className="text-[10px] text-gray-400">Applies to all requests on this website</p>
            </div>
            <div className={`relative w-10 h-5 rounded-full transition-colors ${rateLimit.enabled ? 'bg-blue-500' : 'bg-gray-300'}`}
              onClick={() => setRateLimit(r => ({ ...r, enabled: !r.enabled }))}>
              <div className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${rateLimit.enabled ? 'translate-x-5' : 'translate-x-0.5'}`} />
            </div>
          </label>

          {rateLimit.enabled && (
            <>
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Requests per second</label>
                <input type="number" value={rateLimit.rps} onChange={e => setRateLimit(r => ({ ...r, rps: Number(e.target.value) }))}
                  min={1} max={10000} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Burst (queued requests)</label>
                <input type="number" value={rateLimit.burst} onChange={e => setRateLimit(r => ({ ...r, burst: Number(e.target.value) }))}
                  min={0} max={10000} className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm" />
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={rateLimit.nodelay} onChange={e => setRateLimit(r => ({ ...r, nodelay: e.target.checked }))}
                  className="rounded border-gray-300" />
                <span className="text-sm text-gray-700">No delay (reject excess immediately)</span>
              </label>

              <div className="bg-gray-50 rounded-lg p-3 text-xs text-gray-500 font-mono">
                limit_req_zone $binary_remote_addr zone=rl_{website?.alias}:10m rate={rateLimit.rps}r/s;<br />
                limit_req zone=rl_{website?.alias} burst={rateLimit.burst}{rateLimit.nodelay ? ' nodelay' : ''};
              </div>
            </>
          )}

          <button onClick={async () => {
            const api = (await import('../../api/client')).default;
            await api.put(`/websites/${websiteId}/rate-limit`, rateLimit);
          }} className="w-full py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">
            <Save size={14} className="inline mr-1" /> Save Rate Limit
          </button>
        </div>
      )}

      {/* ── IP Access Tab ── */}
      {tab === 'ipaccess' && (
        <div className="space-y-4 max-w-2xl">
          <div className="bg-white border border-gray-200 rounded-xl p-5">
            <div className="flex items-center gap-2 mb-3">
              <h3 className="font-semibold text-sm">IP Access Control</h3>
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 font-bold">PRO</span>
            </div>
            <p className="text-xs text-gray-500 mb-4">Control which IPs can access this website via Nginx allow/deny directives.</p>

            {/* Mode selector */}
            <div className="flex gap-2 mb-4">
              {[
                { key: 'off', label: 'Off', desc: 'No IP filtering' },
                { key: 'whitelist', label: 'Whitelist', desc: 'Allow only listed IPs' },
                { key: 'blacklist', label: 'Blacklist', desc: 'Block listed IPs' },
              ].map(m => (
                <button key={m.key} onClick={async () => {
                  setIpMode(m.key);
                  const a = (await import('../../api/client')).default;
                  await a.put(`/websites/${websiteId}/ip-access-mode`, { mode: m.key });
                }} className={`flex-1 p-3 rounded-lg border text-center transition-all ${
                  ipMode === m.key
                    ? m.key === 'whitelist' ? 'border-green-300 bg-green-50' : m.key === 'blacklist' ? 'border-red-300 bg-red-50' : 'border-blue-300 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}>
                  <p className="text-sm font-medium">{m.label}</p>
                  <p className="text-[10px] text-gray-400">{m.desc}</p>
                </button>
              ))}
            </div>

            {/* Add IP form */}
            {ipMode !== 'off' && (
              <>
                <div className="flex gap-2 mb-3">
                  <input type="text" value={newIP.ip_address} onChange={e => setNewIP(n => ({ ...n, ip_address: e.target.value }))}
                    placeholder="192.168.1.0/24 or single IP" className="flex-1 py-2 px-3 border border-gray-200 rounded-lg text-sm font-mono" />
                  <input type="text" value={newIP.description} onChange={e => setNewIP(n => ({ ...n, description: e.target.value }))}
                    placeholder="Description (optional)" className="w-48 py-2 px-3 border border-gray-200 rounded-lg text-sm" />
                  <button onClick={async () => {
                    if (!newIP.ip_address) return;
                    const a = (await import('../../api/client')).default;
                    const r = await a.post(`/websites/${websiteId}/ip-rules`, { ...newIP, mode: ipMode === 'whitelist' ? 'allow' : 'deny' });
                    setIpRules(prev => [...prev, r.data.data]);
                    setNewIP({ ip_address: '', description: '' });
                  }} className="px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700">Add</button>
                </div>

                {/* Rules list */}
                {ipRules.length === 0 ? (
                  <p className="text-xs text-gray-400 text-center py-4">No IP rules. Add IPs above to {ipMode === 'whitelist' ? 'allow' : 'block'} them.</p>
                ) : (
                  <div className="space-y-1">
                    {ipRules.map(rule => (
                      <div key={rule.id} className={`flex items-center justify-between p-2.5 rounded-lg border ${
                        rule.mode === 'allow' ? 'border-green-200 bg-green-50/50' : 'border-red-200 bg-red-50/50'
                      }`}>
                        <div className="flex items-center gap-3">
                          <span className={`w-2 h-2 rounded-full ${rule.mode === 'allow' ? 'bg-green-500' : 'bg-red-500'}`} />
                          <span className="text-sm font-mono">{rule.ip_address}</span>
                          {rule.description && <span className="text-xs text-gray-400">- {rule.description}</span>}
                        </div>
                        <button onClick={async () => {
                          const a = (await import('../../api/client')).default;
                          await a.delete(`/websites/${websiteId}/ip-rules/${rule.id}`);
                          setIpRules(prev => prev.filter(r => r.id !== rule.id));
                        }} className="p-1 text-gray-400 hover:text-red-600">
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
                        </button>
                      </div>
                    ))}
                  </div>
                )}

                {ipRules.length > 0 && (
                  <div className="mt-3 bg-gray-50 rounded-lg p-3 text-[10px] font-mono text-gray-500">
                    # Generated Nginx config:<br />
                    {ipRules.map(r => `${r.mode === 'allow' ? 'allow' : 'deny'} ${r.ip_address};`).join('\n')}
                    {ipMode === 'whitelist' && '\ndeny all;'}
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}

      {/* ── Clone Modal ── */}
      {showClone && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setShowClone(false)}>
          <div className="bg-white rounded-xl p-6 w-[400px] space-y-4" onClick={e => e.stopPropagation()}>
            <h3 className="font-semibold">Clone Website <span className="text-xs font-normal text-blue-600 ml-1">PRO</span></h3>
            <p className="text-xs text-gray-500">Creates a full copy of {website?.domain} including files, settings, snippets, and redirects.</p>
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-1">New Domain</label>
              <input type="text" value={cloneDomain} onChange={e => setCloneDomain(e.target.value)}
                placeholder="clone.example.com" autoFocus
                className="w-full py-2.5 px-3 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-purple-500/20 focus:border-purple-400" />
            </div>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setShowClone(false)} className="py-2 px-4 text-sm rounded-lg border border-gray-200 hover:bg-gray-50">Cancel</button>
              <button onClick={async () => {
                if (!cloneDomain.trim()) return;
                setCloning(true);
                try {
                  const api = (await import('../../api/client')).default;
                  const r = await api.post(`/websites/${websiteId}/clone`, { domain: cloneDomain.trim() });
                  navigate(`/websites/${r.data.data.id}`);
                } catch {}
                setCloning(false);
              }} disabled={cloning || !cloneDomain.trim()}
                className="py-2 px-4 text-sm font-medium rounded-lg bg-purple-600 text-white hover:bg-purple-700 disabled:opacity-50">
                {cloning ? 'Cloning...' : 'Clone'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
