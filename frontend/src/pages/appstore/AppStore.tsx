import { useEffect, useState, useRef, useCallback } from 'react';
import { appApi, type App, type AppInstall } from '../../api/appstore';
import { sslApi } from '../../api/ssl';
import { websiteApi } from '../../api/websites';
import api from '../../api/client';

type Tab = 'catalog' | 'installed' | 'repos';

interface AppRepo { id: number; name: string; url: string; enabled: boolean; }

const categoryMeta: Record<string, { icon: string; color: string; bg: string }> = {
  CMS:        { icon: '📝', color: 'text-blue-700',   bg: 'bg-blue-50 border-blue-200' },
  DevOps:     { icon: '⚙️', color: 'text-indigo-700', bg: 'bg-indigo-50 border-indigo-200' },
  Database:   { icon: '💾', color: 'text-purple-700',  bg: 'bg-purple-50 border-purple-200' },
  Networking: { icon: '🌐', color: 'text-cyan-700',    bg: 'bg-cyan-50 border-cyan-200' },
  Monitoring: { icon: '📊', color: 'text-green-700',   bg: 'bg-green-50 border-green-200' },
  Storage:    { icon: '🗄️', color: 'text-amber-700',   bg: 'bg-amber-50 border-amber-200' },
};

const defaultCat = { icon: '📦', color: 'text-gray-700', bg: 'bg-gray-50 border-gray-200' };
const getCat = (c: string) => categoryMeta[c] || defaultCat;

const appIcons: Record<string, string> = {
  wordpress: '🔵', gitea: '🟠', redis: '🔴', 'nginx-proxy-manager': '🟢',
  portainer: '🐳', mariadb: '🐬', 'uptime-kuma': '💚', minio: '🪣',
};

// Which port var is the "main" web UI port for each app
const mainPortVar: Record<string, string> = {
  wordpress: 'PORT', gitea: 'PORT', redis: 'PORT', 'nginx-proxy-manager': 'ADMIN_PORT',
  portainer: 'PORT', mariadb: 'PORT', 'uptime-kuma': 'PORT', minio: 'CONSOLE_PORT',
};

// Apps that have a web UI (vs TCP-only like redis/mariadb)
const hasWebUI = new Set(['wordpress', 'gitea', 'nginx-proxy-manager', 'portainer', 'uptime-kuma', 'minio']);

function getEnvMap(install: AppInstall): Record<string, string> {
  if (!install.env_values) return {};
  if (typeof install.env_values === 'string') {
    try { return JSON.parse(install.env_values); } catch { return {}; }
  }
  return install.env_values as unknown as Record<string, string>;
}

function getMainPort(install: AppInstall): string | null {
  const env = getEnvMap(install);
  const varName = mainPortVar[install.app_key];
  if (varName && env[varName]) return env[varName];
  // fallback: find first PORT var
  for (const [k, v] of Object.entries(env)) {
    if (k.toUpperCase().includes('PORT')) return v;
  }
  return null;
}

function getPortEntries(install: AppInstall): [string, string][] {
  const env = getEnvMap(install);
  return Object.entries(env).filter(([k]) => k.toUpperCase().includes('PORT'));
}

function getCredentials(install: AppInstall): [string, string][] {
  const env = getEnvMap(install);
  return Object.entries(env).filter(([k]) => {
    const up = k.toUpperCase();
    return up.includes('PASSWORD') || up.includes('SECRET') || up.includes('USER');
  });
}

export default function AppStore() {
  const [tab, setTab] = useState<Tab>('catalog');
  const [catalog, setCatalog] = useState<App[]>([]);
  const [installed, setInstalled] = useState<AppInstall[]>([]);
  const [repos, setRepos] = useState<AppRepo[]>([]);
  const [loading, setLoading] = useState(true);
  const [installing, setInstalling] = useState<string | null>(null);
  const [filter, setFilter] = useState('');
  const [category, setCategory] = useState('');
  const [newRepo, setNewRepo] = useState({ name: '', url: '' });
  const [healthResults, setHealthResults] = useState<Record<string, string>>({});
  const [expandedCreds, setExpandedCreds] = useState<Set<number>>(new Set());
  const [autoUpdate, setAutoUpdate] = useState(false);
  const [updatingAll, setUpdatingAll] = useState(false);
  const [domainModal, setDomainModal] = useState<AppInstall | null>(null);
  const [domainInput, setDomainInput] = useState('');
  const [domainEmail, setDomainEmail] = useState('');
  const [domainSSL, setDomainSSL] = useState(true);
  const [setupRunning, setSetupRunning] = useState(false);
  const [setupSteps, setSetupSteps] = useState<{ label: string; status: 'pending' | 'running' | 'done' | 'error'; detail?: string }[]>([]);
  const [setupDone, setSetupDone] = useState(false);
  const [setupError, setSetupError] = useState('');
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchCatalog = async () => { setLoading(true); try { const r = await appApi.catalog(); setCatalog(r.data.data || []); } catch {} finally { setLoading(false); } };
  const fetchInstalled = async () => { try { const r = await appApi.installed(); setInstalled(r.data.data || []); } catch {} };
  const fetchRepos = async () => { try { const r = await api.get<{ data: AppRepo[] }>('/apps/repos'); setRepos(r.data.data || []); } catch {} };

  useEffect(() => {
    fetchCatalog(); fetchInstalled();
    api.get('/apps/auto-update').then(r => setAutoUpdate(r.data.data?.enabled || false)).catch(() => {});
  }, []);
  useEffect(() => { if (tab === 'repos') fetchRepos(); }, [tab]);

  const handleInstall = async (app: App) => {
    setInstalling(app.key);
    try {
      await appApi.install({ app_key: app.key, name: '', version: 'latest' });
      await fetchInstalled();
      setTab('installed');
    } catch {}
    setInstalling(null);
  };

  const handleUpgrade = async (id: number) => {
    const version = prompt('Upgrade to version:', 'latest');
    if (!version) return;
    await api.post(`/apps/installed/${id}/upgrade`, { version }).catch(() => {});
    fetchInstalled();
  };

  const handleHealthCheck = async (id: number, name: string) => {
    try {
      const r = await api.get<{ data: { status: string } }>(`/apps/installed/${id}/health`);
      setHealthResults(prev => ({ ...prev, [name]: r.data.data.status }));
    } catch { setHealthResults(prev => ({ ...prev, [name]: 'error' })); }
  };

  const handleHealthCheckAll = async () => {
    try {
      const r = await api.post<{ data: Record<string, string> }>('/apps/health-check-all');
      setHealthResults(r.data.data);
    } catch {}
    fetchInstalled();
  };

  const handleOperate = async (id: number, action: 'start' | 'stop' | 'restart') => { await appApi.operate(id, action).catch(() => {}); fetchInstalled(); };
  const handleUninstall = async (id: number, name: string) => { if (!confirm(`Uninstall "${name}"? All data will be deleted.`)) return; await appApi.uninstall(id).catch(() => {}); fetchInstalled(); };

  const toggleCreds = (id: number) => {
    setExpandedCreds(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const updateStep = useCallback((idx: number, status: 'running' | 'done' | 'error', detail?: string) => {
    setSetupSteps(prev => prev.map((s, i) => i === idx ? { ...s, status, detail: detail || s.detail } : s));
  }, []);

  const handleAddDomain = async () => {
    if (!domainModal || !domainInput.trim()) return;
    if (domainSSL && !domainEmail.trim()) return;

    const domain = domainInput.trim();
    const port = getMainPort(domainModal);
    const withSSL = domainSSL;

    const steps = [
      { label: 'Creating reverse proxy website', status: 'pending' as const },
      ...(withSSL ? [
        { label: 'Setting up ACME account', status: 'pending' as const },
        { label: 'Issuing Let\'s Encrypt certificate', status: 'pending' as const },
        { label: 'Applying SSL certificate', status: 'pending' as const },
      ] : []),
    ];
    setSetupSteps(steps);
    setSetupRunning(true);
    setSetupDone(false);
    setSetupError('');

    let stepIdx = 0;

    try {
      // Step 1: Create proxy website
      updateStep(stepIdx, 'running');
      const siteRes = await websiteApi.create({
        domain,
        type: 'proxy',
        proxy_target: `http://127.0.0.1:${port}`,
        remark: `App: ${domainModal.name} (${domainModal.app_key})`,
      });
      const websiteId = siteRes.data.data.id;
      updateStep(stepIdx, 'done', `Website created: ${domain}`);
      stepIdx++;

      if (withSSL) {
        // Step 2: Ensure ACME account
        updateStep(stepIdx, 'running');
        let acmeAccountId: number;
        const accountsRes = await sslApi.listAccounts();
        const accounts = accountsRes.data.data || [];
        if (accounts.length > 0) {
          acmeAccountId = accounts[0].id;
          updateStep(stepIdx, 'done', `Using account: ${accounts[0].email}`);
        } else {
          const newAcc = await sslApi.createAccount(domainEmail.trim());
          acmeAccountId = newAcc.data.data.id;
          updateStep(stepIdx, 'done', `Created account: ${domainEmail.trim()}`);
        }
        stepIdx++;

        // Step 3: Issue certificate
        updateStep(stepIdx, 'running', 'Requesting certificate from Let\'s Encrypt...');
        const certRes = await sslApi.issue({
          domains: [domain],
          acme_account_id: acmeAccountId,
          auto_renew: true,
          http_config: 'HTTPToHTTPS',
          description: `Auto: ${domainModal.name}`,
        });
        const certId = certRes.data.data.id;

        // Poll for cert status
        const sslStepIdx = stepIdx;
        await new Promise<void>((resolve, reject) => {
          let attempts = 0;
          const maxAttempts = 60; // 2 minutes
          pollRef.current = setInterval(async () => {
            attempts++;
            try {
              const check = await sslApi.get(certId);
              const st = check.data.data.status;
              if (st === 'issued') {
                clearInterval(pollRef.current!);
                pollRef.current = null;
                updateStep(sslStepIdx, 'done', 'Certificate issued successfully');
                resolve();
              } else if (st === 'failed') {
                clearInterval(pollRef.current!);
                pollRef.current = null;
                reject(new Error(check.data.data.message || 'Certificate issuance failed'));
              } else {
                updateStep(sslStepIdx, 'running', `Verifying domain ownership... (${attempts}s)`);
              }
            } catch {
              // ignore polling errors
            }
            if (attempts >= maxAttempts) {
              clearInterval(pollRef.current!);
              pollRef.current = null;
              reject(new Error('Certificate issuance timed out (2 min)'));
            }
          }, 2000);
        });
        stepIdx++;

        // Step 4: Apply SSL to website
        updateStep(stepIdx, 'running');
        await sslApi.apply(certId, websiteId, 'HTTPToHTTPS');
        updateStep(stepIdx, 'done', 'HTTPS enabled with automatic redirect');
        stepIdx++;
      }

      setSetupDone(true);
    } catch (e: any) {
      if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null; }
      const msg = e.response?.data?.error?.message || e.message || 'Setup failed';
      setSetupError(msg);
      updateStep(stepIdx, 'error', msg);
    }
    setSetupRunning(false);
  };

  const isInstalled = (key: string) => installed.some(i => i.app_key === key);
  const categories = [...new Set(catalog.map(a => a.category))].sort();
  const filtered = catalog.filter(a => {
    if (category && a.category !== category) return false;
    if (filter && !a.name.toLowerCase().includes(filter.toLowerCase()) && !a.description.toLowerCase().includes(filter.toLowerCase())) return false;
    return true;
  });

  const statusConfig: Record<string, { dot: string; bg: string; text: string; label: string }> = {
    running:    { dot: 'bg-green-500',  bg: 'bg-green-50',  text: 'text-green-700',  label: 'Running' },
    stopped:    { dot: 'bg-gray-400',   bg: 'bg-gray-50',   text: 'text-gray-600',   label: 'Stopped' },
    installing: { dot: 'bg-yellow-500', bg: 'bg-yellow-50', text: 'text-yellow-700', label: 'Installing' },
    error:      { dot: 'bg-red-500',    bg: 'bg-red-50',    text: 'text-red-700',    label: 'Error' },
  };
  const defaultStatus = { dot: 'bg-gray-400', bg: 'bg-gray-50', text: 'text-gray-600', label: 'Unknown' };

  const healthConfig: Record<string, { color: string; bg: string }> = {
    healthy:   { color: 'text-green-700', bg: 'bg-green-50' },
    unhealthy: { color: 'text-red-700',   bg: 'bg-red-50' },
  };

  const tabItems: { key: Tab; label: string; count?: number; icon: string }[] = [
    { key: 'catalog', label: 'Catalog', count: catalog.length, icon: '🏪' },
    { key: 'installed', label: 'Installed', count: installed.length, icon: '📥' },
    { key: 'repos', label: 'Repositories', icon: '🔗' },
  ];

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">App Store</h1>
          <p className="text-sm text-gray-400 mt-1">Deploy and manage containerized applications</p>
        </div>
        {tab === 'installed' && installed.length > 0 && (
          <div className="flex items-center gap-2">
            {/* Auto-update toggle */}
            <label className="inline-flex items-center gap-2 py-2 px-3 text-sm rounded-lg border border-gray-200 cursor-pointer hover:bg-gray-50">
              <div className={`relative w-8 h-4 rounded-full transition-colors ${autoUpdate ? 'bg-blue-500' : 'bg-gray-300'}`}
                onClick={async () => {
                  const next = !autoUpdate;
                  setAutoUpdate(next);
                  await api.put('/apps/auto-update', { enabled: next }).catch(() => setAutoUpdate(!next));
                }}>
                <div className={`absolute top-0.5 w-3 h-3 bg-white rounded-full shadow transition-transform ${autoUpdate ? 'translate-x-4' : 'translate-x-0.5'}`} />
              </div>
              <span className="text-xs text-gray-600">Auto-update</span>
            </label>
            <button onClick={async () => {
              setUpdatingAll(true);
              await api.post('/apps/auto-update/run').catch(() => {});
              setUpdatingAll(false);
              fetchInstalled();
            }} disabled={updatingAll} className="inline-flex items-center gap-2 py-2 px-4 text-sm font-medium rounded-lg border border-gray-200 hover:bg-gray-50 disabled:opacity-50 transition-all">
              <svg className={`w-4 h-4 text-gray-500 ${updatingAll ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
              {updatingAll ? 'Updating...' : 'Update All'}
            </button>
            <button onClick={handleHealthCheckAll} className="inline-flex items-center gap-2 py-2 px-4 text-sm font-medium rounded-lg border border-gray-200 hover:bg-gray-50 hover:shadow-sm transition-all">
              <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" /></svg>
              Health Check
            </button>
          </div>
        )}
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 bg-gray-100 p-1 rounded-xl w-fit">
        {tabItems.map(t => (
          <button key={t.key} onClick={() => setTab(t.key)}
            className={`inline-flex items-center gap-2 py-2 px-4 text-sm font-medium rounded-lg transition-all ${
              tab === t.key ? 'bg-white text-gray-800 shadow-sm' : 'text-gray-500 hover:text-gray-700'
            }`}>
            <span className="text-base">{t.icon}</span>
            {t.label}
            {t.count !== undefined && (
              <span className={`ml-1 py-0.5 px-2 text-[11px] font-semibold rounded-full ${
                tab === t.key ? 'bg-blue-100 text-blue-700' : 'bg-gray-200 text-gray-500'
              }`}>{t.count}</span>
            )}
          </button>
        ))}
      </div>

      {/* ───────────────── CATALOG ───────────────── */}
      {tab === 'catalog' && (
        <div>
          <div className="flex flex-col gap-3 mb-6">
            <div className="relative max-w-md">
              <div className="absolute inset-y-0 start-0 flex items-center ps-3.5">
                <svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>
              </div>
              <input type="text" value={filter} onChange={e => setFilter(e.target.value)} placeholder="Search apps..."
                className="py-2.5 ps-10 pe-4 block w-full border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all" />
            </div>
            <div className="flex flex-wrap gap-2">
              <button onClick={() => setCategory('')}
                className={`py-1.5 px-3 text-xs font-medium rounded-full border transition-all ${
                  !category ? 'bg-gray-800 text-white border-gray-800' : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                }`}>All</button>
              {categories.map(c => {
                const meta = getCat(c);
                return (
                  <button key={c} onClick={() => setCategory(category === c ? '' : c)}
                    className={`inline-flex items-center gap-1.5 py-1.5 px-3 text-xs font-medium rounded-full border transition-all ${
                      category === c ? `${meta.bg} ${meta.color} border-current` : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                    }`}>
                    <span>{meta.icon}</span>{c}
                  </button>
                );
              })}
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-20">
              <div className="flex flex-col items-center gap-3">
                <div className="w-8 h-8 border-2 border-gray-200 border-t-blue-600 rounded-full animate-spin" />
                <span className="text-sm text-gray-400">Loading catalog...</span>
              </div>
            </div>
          ) : filtered.length === 0 ? (
            <div className="bg-white border border-gray-200 rounded-xl p-16 text-center">
              <div className="text-4xl mb-3">🔍</div>
              <p className="text-gray-500 font-medium">No apps found</p>
              <p className="text-sm text-gray-400 mt-1">Try adjusting your search or filters</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {filtered.map(app => {
                const cat = getCat(app.category);
                const done = isInstalled(app.key);
                return (
                  <div key={app.key}
                    className="group bg-white border border-gray-200 rounded-xl p-5 flex flex-col hover:shadow-md hover:-translate-y-0.5 transition-all relative overflow-hidden">
                    {done && (
                      <div className="absolute top-3 right-3">
                        <div className="w-6 h-6 rounded-full bg-green-100 flex items-center justify-center">
                          <svg className="w-3.5 h-3.5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" /></svg>
                        </div>
                      </div>
                    )}
                    <div className="flex items-start gap-3 mb-3">
                      <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-gray-50 to-gray-100 border border-gray-100 flex items-center justify-center text-2xl shrink-0 group-hover:scale-105 transition-transform">
                        {appIcons[app.key] || cat.icon}
                      </div>
                      <div className="min-w-0 pt-0.5">
                        <h3 className="font-semibold text-sm text-gray-800 truncate">{app.name}</h3>
                        <span className={`inline-flex items-center gap-1 mt-1 py-0.5 px-2 text-[10px] font-semibold rounded-full border ${cat.bg} ${cat.color}`}>{app.category}</span>
                      </div>
                    </div>
                    <p className="text-xs text-gray-500 leading-relaxed flex-1 mb-4">{app.description}</p>
                    {app.versions?.length > 0 && (
                      <div className="flex flex-wrap gap-1 mb-4">
                        {app.versions.slice(0, 4).map((v: string) => (
                          <span key={v} className="py-0.5 px-2 text-[10px] font-mono bg-gray-50 text-gray-500 rounded-md border border-gray-100">{v}</span>
                        ))}
                        {app.versions.length > 4 && <span className="py-0.5 px-2 text-[10px] text-gray-400">+{app.versions.length - 4}</span>}
                      </div>
                    )}
                    <button onClick={() => handleInstall(app)} disabled={installing === app.key || done}
                      className={`w-full py-2.5 rounded-lg text-sm font-medium transition-all ${
                        done ? 'bg-green-50 text-green-600 border border-green-200 cursor-default'
                          : installing === app.key ? 'bg-blue-50 text-blue-500 border border-blue-200'
                          : 'bg-blue-600 text-white hover:bg-blue-700 shadow-sm hover:shadow'
                      }`}>
                      {done ? 'Installed' : installing === app.key ? (
                        <span className="inline-flex items-center gap-2">
                          <div className="w-3.5 h-3.5 border-2 border-blue-300 border-t-blue-600 rounded-full animate-spin" />
                          Installing...
                        </span>
                      ) : 'Install'}
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ───────────────── INSTALLED ───────────────── */}
      {tab === 'installed' && (
        <div>
          {installed.length === 0 ? (
            <div className="bg-white border border-gray-200 rounded-xl p-16 text-center">
              <div className="text-4xl mb-3">📦</div>
              <p className="text-gray-500 font-medium">No apps installed yet</p>
              <p className="text-sm text-gray-400 mt-1 mb-4">Browse the catalog and install your first app</p>
              <button onClick={() => setTab('catalog')} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">Browse Catalog</button>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {installed.map(app => {
                const st = statusConfig[app.status] || defaultStatus;
                const health = healthResults[app.name];
                const hc = health ? healthConfig[health] : null;
                const catalogApp = catalog.find(c => c.key === app.app_key);
                const cat = catalogApp ? getCat(catalogApp.category) : defaultCat;
                const mainPort = getMainPort(app);
                const ports = getPortEntries(app);
                const creds = getCredentials(app);
                const showCreds = expandedCreds.has(app.id);
                const isWeb = hasWebUI.has(app.app_key);

                return (
                  <div key={app.id} className="bg-white border border-gray-200 rounded-xl overflow-hidden hover:shadow-md transition-all">
                    {/* Header */}
                    <div className="p-5 pb-4">
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3">
                          <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-gray-50 to-gray-100 border border-gray-100 flex items-center justify-center text-xl shrink-0">
                            {appIcons[app.app_key] || cat.icon}
                          </div>
                          <div>
                            <h3 className="font-semibold text-sm text-gray-800">{app.name}</h3>
                            <p className="text-xs text-gray-400 font-mono">{app.app_key}</p>
                          </div>
                        </div>
                        <div className={`inline-flex items-center gap-1.5 py-1 px-2.5 rounded-full text-xs font-medium ${st.bg} ${st.text}`}>
                          <span className={`w-1.5 h-1.5 rounded-full ${st.dot} ${app.status === 'installing' ? 'animate-pulse' : ''}`} />
                          {st.label}
                        </div>
                      </div>

                      {/* Version + health row */}
                      <div className="flex items-center gap-3 text-xs mb-3">
                        <span className="inline-flex items-center gap-1 py-1 px-2 bg-gray-50 text-gray-500 rounded-md font-mono border border-gray-100">v{app.version}</span>
                        {hc ? (
                          <span className={`inline-flex items-center gap-1 py-1 px-2 rounded-md font-medium ${hc.bg} ${hc.color}`}>
                            {health === 'healthy'
                              ? <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" /></svg>
                              : <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>}
                            {health}
                          </span>
                        ) : (
                          <button onClick={() => handleHealthCheck(app.id, app.name)} className="text-xs text-blue-500 hover:text-blue-700 hover:underline">Check health</button>
                        )}
                        <span className="text-gray-300 ml-auto text-[10px]">{new Date(app.created_at).toLocaleDateString()}</span>
                      </div>

                      {/* Access info */}
                      {ports.length > 0 && (
                        <div className="bg-gray-50 rounded-lg p-3 space-y-2">
                          {/* Main access link */}
                          {mainPort && isWeb && app.status === 'running' && (
                            <div className="flex items-center gap-2">
                              <svg className="w-3.5 h-3.5 text-blue-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
                              <a href={`${window.location.protocol}//${window.location.hostname}:${mainPort}`} target="_blank" rel="noopener noreferrer"
                                className="text-xs text-blue-600 hover:text-blue-800 font-medium hover:underline truncate">
                                {window.location.hostname}:{mainPort}
                              </a>
                            </div>
                          )}
                          {/* All ports */}
                          <div className="flex flex-wrap gap-1.5">
                            {ports.map(([k, v]) => (
                              <span key={k} className="inline-flex items-center gap-1 py-0.5 px-2 text-[10px] rounded-md bg-white border border-gray-200 font-mono">
                                <span className="text-gray-400">{k.replace(/_/g, ' ').toLowerCase()}:</span>
                                <span className="text-gray-700 font-semibold">{v}</span>
                              </span>
                            ))}
                          </div>

                          {/* Credentials toggle */}
                          {creds.length > 0 && (
                            <div>
                              <button onClick={() => toggleCreds(app.id)}
                                className="inline-flex items-center gap-1 text-[10px] text-gray-400 hover:text-gray-600 transition-colors">
                                <svg className={`w-3 h-3 transition-transform ${showCreds ? 'rotate-90' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" /></svg>
                                {showCreds ? 'Hide credentials' : 'Show credentials'}
                              </button>
                              {showCreds && (
                                <div className="mt-1.5 space-y-1">
                                  {creds.map(([k, v]) => (
                                    <div key={k} className="flex items-center gap-2 text-[10px]">
                                      <span className="text-gray-400 min-w-[80px]">{k.replace(/_/g, ' ').toLowerCase()}</span>
                                      <code className="bg-white border border-gray-200 rounded px-1.5 py-0.5 text-gray-700 font-mono select-all">{v}</code>
                                    </div>
                                  ))}
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-1 px-4 py-3 bg-gray-50/50 border-t border-gray-100">
                      {app.status !== 'running' && (
                        <button onClick={() => handleOperate(app.id, 'start')} title="Start"
                          className="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-lg text-xs font-medium text-green-700 hover:bg-green-100 transition-colors">
                          <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z" /></svg>Start
                        </button>
                      )}
                      {app.status === 'running' && (
                        <>
                          <button onClick={() => handleOperate(app.id, 'stop')} title="Stop"
                            className="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-lg text-xs font-medium text-amber-700 hover:bg-amber-100 transition-colors">
                            <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24"><path d="M6 4h4v16H6zM14 4h4v16h-4z" /></svg>Stop
                          </button>
                          <button onClick={() => handleOperate(app.id, 'restart')} title="Restart"
                            className="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-lg text-xs font-medium text-blue-700 hover:bg-blue-100 transition-colors">
                            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg>Restart
                          </button>
                        </>
                      )}
                      <div className="flex-1" />
                      {mainPort && isWeb && (
                        <button onClick={() => { setDomainModal(app); setDomainInput(''); setSetupError(''); setSetupSteps([]); setSetupDone(false); }} title="Add Domain"
                          className="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-lg text-xs font-medium text-purple-700 hover:bg-purple-100 transition-colors">
                          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" /></svg>
                          Domain
                        </button>
                      )}
                      <button onClick={() => handleUpgrade(app.id)} title="Upgrade"
                        className="p-1.5 rounded-lg text-gray-400 hover:text-blue-600 hover:bg-blue-50 transition-colors">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>
                      </button>
                      <button onClick={() => handleUninstall(app.id, app.name)} title="Uninstall"
                        className="p-1.5 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50 transition-colors">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ───────────────── DOMAIN MODAL ───────────────── */}
      {domainModal && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => !setupRunning && setDomainModal(null)}>
          <div className="bg-white rounded-xl p-6 w-[480px] space-y-4" onClick={e => e.stopPropagation()}>
            {/* Header */}
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-purple-50 flex items-center justify-center text-xl">
                {appIcons[domainModal.app_key] || '📦'}
              </div>
              <div>
                <h3 className="font-semibold text-sm">Connect Domain to {domainModal.name}</h3>
                <p className="text-xs text-gray-400">
                  Reverse proxy to port {getMainPort(domainModal)} {domainSSL ? '+ free SSL certificate' : ''}
                </p>
              </div>
            </div>

            {/* Form (hidden during setup) */}
            {!setupRunning && !setupDone && (
              <>
                <div>
                  <label className="block text-xs font-medium text-gray-500 mb-1">Domain</label>
                  <input type="text" value={domainInput} onChange={e => setDomainInput(e.target.value)}
                    placeholder="app.example.com"
                    className="w-full py-2.5 px-3 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-purple-500/20 focus:border-purple-400 transition-all"
                    autoFocus />
                </div>

                {/* SSL toggle */}
                <label className="flex items-center justify-between p-3 bg-gray-50 rounded-lg cursor-pointer group">
                  <div className="flex items-center gap-3">
                    <div className={`w-8 h-8 rounded-lg flex items-center justify-center text-sm ${domainSSL ? 'bg-green-100' : 'bg-gray-200'}`}>
                      {domainSSL ? '🔒' : '🔓'}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-700">Free SSL Certificate</p>
                      <p className="text-[10px] text-gray-400">Auto-issue Let's Encrypt + HTTPS redirect</p>
                    </div>
                  </div>
                  <div className={`relative w-10 h-5 rounded-full transition-colors ${domainSSL ? 'bg-green-500' : 'bg-gray-300'}`}
                    onClick={() => setDomainSSL(!domainSSL)}>
                    <div className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${domainSSL ? 'translate-x-5' : 'translate-x-0.5'}`} />
                  </div>
                </label>

                {/* Email for ACME (only if SSL enabled) */}
                {domainSSL && (
                  <div>
                    <label className="block text-xs font-medium text-gray-500 mb-1">Email (for Let's Encrypt)</label>
                    <input type="email" value={domainEmail} onChange={e => setDomainEmail(e.target.value)}
                      placeholder="admin@example.com"
                      className="w-full py-2.5 px-3 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-purple-500/20 focus:border-purple-400 transition-all" />
                    <p className="text-[10px] text-gray-400 mt-1">Required for certificate issuance. Skipped if account already exists.</p>
                  </div>
                )}

                {/* DNS hint */}
                {domainInput && (
                  <div className="bg-blue-50 border border-blue-100 rounded-lg p-3 text-xs space-y-1.5">
                    <p className="font-semibold text-blue-800">DNS Setup Required</p>
                    <p className="text-blue-600">Point your domain to this server before proceeding:</p>
                    <div className="flex items-center gap-2 bg-white/70 rounded px-2 py-1.5 font-mono text-blue-800">
                      <span className="text-blue-400">A</span>
                      <span>{domainInput}</span>
                      <span className="text-blue-400 mx-1">&rarr;</span>
                      <span className="font-semibold">{window.location.hostname}</span>
                    </div>
                  </div>
                )}

                {setupError && (
                  <div className="bg-red-50 border border-red-200 text-red-700 text-xs px-3 py-2 rounded-lg">{setupError}</div>
                )}

                <div className="flex gap-2 justify-end">
                  <button onClick={() => { setDomainModal(null); setSetupError(''); }}
                    className="py-2 px-4 text-sm rounded-lg border border-gray-200 hover:bg-gray-50">Cancel</button>
                  <button onClick={handleAddDomain}
                    disabled={!domainInput.trim() || (domainSSL && !domainEmail.trim())}
                    className="py-2 px-4 text-sm font-medium rounded-lg bg-purple-600 text-white hover:bg-purple-700 disabled:opacity-50 transition-all">
                    Deploy
                  </button>
                </div>
              </>
            )}

            {/* Progress steps */}
            {(setupRunning || setupDone) && (
              <>
                <div className="space-y-2">
                  {setupSteps.map((step, i) => (
                    <div key={i} className={`flex items-center gap-3 p-3 rounded-lg transition-all ${
                      step.status === 'running' ? 'bg-blue-50 border border-blue-100' :
                      step.status === 'done' ? 'bg-green-50 border border-green-100' :
                      step.status === 'error' ? 'bg-red-50 border border-red-100' :
                      'bg-gray-50 border border-transparent'
                    }`}>
                      {/* Status icon */}
                      <div className="shrink-0">
                        {step.status === 'pending' && <div className="w-5 h-5 rounded-full border-2 border-gray-200" />}
                        {step.status === 'running' && <div className="w-5 h-5 border-2 border-blue-200 border-t-blue-600 rounded-full animate-spin" />}
                        {step.status === 'done' && (
                          <div className="w-5 h-5 rounded-full bg-green-500 flex items-center justify-center">
                            <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" /></svg>
                          </div>
                        )}
                        {step.status === 'error' && (
                          <div className="w-5 h-5 rounded-full bg-red-500 flex items-center justify-center">
                            <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M6 18L18 6M6 6l12 12" /></svg>
                          </div>
                        )}
                      </div>
                      <div className="min-w-0">
                        <p className={`text-sm font-medium ${
                          step.status === 'done' ? 'text-green-700' :
                          step.status === 'error' ? 'text-red-700' :
                          step.status === 'running' ? 'text-blue-700' : 'text-gray-400'
                        }`}>{step.label}</p>
                        {step.detail && <p className="text-[10px] text-gray-500 truncate">{step.detail}</p>}
                      </div>
                    </div>
                  ))}
                </div>

                {/* Success card */}
                {setupDone && (
                  <div className="bg-gradient-to-r from-green-50 to-emerald-50 border border-green-200 rounded-lg p-4 text-center space-y-2">
                    <div className="text-2xl">🎉</div>
                    <p className="font-semibold text-green-800 text-sm">Your app is live!</p>
                    <a href={`${domainSSL ? 'https' : 'http'}://${domainInput}`} target="_blank" rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 text-sm font-medium text-green-700 hover:text-green-900 hover:underline">
                      {domainSSL ? 'https' : 'http'}://{domainInput}
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
                    </a>
                  </div>
                )}

                {/* Error retry */}
                {setupError && !setupRunning && (
                  <div className="flex gap-2 justify-end">
                    <button onClick={() => { setSetupSteps([]); setSetupDone(false); setSetupError(''); }}
                      className="py-2 px-4 text-sm rounded-lg border border-gray-200 hover:bg-gray-50">Back</button>
                  </div>
                )}

                {/* Done close */}
                {setupDone && (
                  <div className="flex justify-end">
                    <button onClick={() => { setDomainModal(null); setSetupSteps([]); setSetupDone(false); setSetupError(''); setDomainInput(''); setDomainEmail(''); }}
                      className="py-2 px-4 text-sm font-medium rounded-lg bg-green-600 text-white hover:bg-green-700">Done</button>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}

      {/* ───────────────── REPOS ───────────────── */}
      {tab === 'repos' && (
        <div className="space-y-4">
          <div className="bg-white border border-gray-200 rounded-xl p-5 max-w-lg">
            <h3 className="font-semibold text-sm mb-3">Add Repository</h3>
            <div className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">Name</label>
                <input type="text" value={newRepo.name} onChange={e => setNewRepo(r => ({ ...r, name: e.target.value }))} placeholder="My Repo"
                  className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all" />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 mb-1">URL</label>
                <input type="text" value={newRepo.url} onChange={e => setNewRepo(r => ({ ...r, url: e.target.value }))} placeholder="https://repo.example.com/apps.json"
                  className="w-full py-2 px-3 border border-gray-200 rounded-lg text-sm font-mono focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all" />
              </div>
              <button onClick={async () => { await api.post('/apps/repos', newRepo); setNewRepo({ name: '', url: '' }); fetchRepos(); }}
                className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 shadow-sm hover:shadow transition-all">Add Repository</button>
            </div>
          </div>

          <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
            {repos.length === 0 ? (
              <div className="p-12 text-center">
                <div className="text-3xl mb-3">🔗</div>
                <p className="text-gray-500 font-medium">No custom repositories</p>
                <p className="text-sm text-gray-400 mt-1">Using built-in catalog only</p>
              </div>
            ) : (
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50"><tr>
                  <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Name</th>
                  <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">URL</th>
                  <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Status</th>
                  <th className="px-6 py-3 text-end"></th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200">
                  {repos.map(r => (
                    <tr key={r.id} className="hover:bg-gray-50">
                      <td className="px-6 py-3 text-sm font-medium">{r.name}</td>
                      <td className="px-6 py-3 text-sm text-gray-500 font-mono">{r.url}</td>
                      <td className="px-6 py-3">
                        <span className={`inline-flex items-center gap-1.5 py-1 px-2.5 text-xs font-medium rounded-full ${r.enabled ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                          <span className={`w-1.5 h-1.5 rounded-full ${r.enabled ? 'bg-green-500' : 'bg-gray-400'}`} />
                          {r.enabled ? 'Active' : 'Disabled'}
                        </span>
                      </td>
                      <td className="px-6 py-3 text-end">
                        <button onClick={async () => { await api.delete(`/apps/repos/${r.id}`); fetchRepos(); }}
                          className="py-1 px-3 text-xs font-medium rounded-lg text-red-600 hover:bg-red-50 transition-colors">Remove</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
