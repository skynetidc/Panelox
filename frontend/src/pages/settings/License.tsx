import { useEffect, useState } from 'react';
import api from '../../api/client';

interface LicenseInfo {
  id: number;
  tier: string;
  customer_name: string;
  customer_email: string;
  license_key: string;
  status: string;
  activated_at: string;
  expires_at?: string;
}

interface LicenseStatus {
  license: LicenseInfo | null;
  tier: string;
  used_websites: number;
  used_nodes: number;
  used_databases: number;
  used_apps: number;
  max_websites: number;
  max_nodes: number;
  max_databases: number;
  max_apps: number;
  features: string[];
  is_expired: boolean;
}

export default function License() {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [keyInput, setKeyInput] = useState('');
  const [activating, setActivating] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchStatus = async () => {
    try {
      const r = await api.get<{ data: LicenseStatus }>('/license');
      setStatus(r.data.data);
    } catch {}
    setLoading(false);
  };

  useEffect(() => { fetchStatus(); }, []);

  const handleActivate = async () => {
    if (!keyInput.trim()) return;
    setActivating(true);
    setMessage(null);
    try {
      await api.post('/license/activate', { license_key: keyInput.trim() });
      setMessage({ type: 'success', text: 'License activated successfully!' });
      setKeyInput('');
      fetchStatus();
    } catch (e: any) {
      setMessage({ type: 'error', text: e.response?.data?.error?.message || 'Invalid license key' });
    }
    setActivating(false);
  };

  const handleDeactivate = async () => {
    if (!confirm('Deactivate license? You will revert to the free tier.')) return;
    await api.delete('/license');
    setMessage({ type: 'success', text: 'License deactivated.' });
    fetchStatus();
  };

  if (loading) return (
    <div className="flex items-center justify-center py-20">
      <div className="w-8 h-8 border-2 border-gray-200 border-t-blue-600 rounded-full animate-spin" />
    </div>
  );

  if (!status) return null;

  const isPro = status.tier !== 'free';

  const resourceBar = (label: string, used: number, max: number) => {
    const unlimited = max === 0;
    const pct = unlimited ? 0 : Math.min((used / max) * 100, 100);
    const atLimit = !unlimited && used >= max;
    return (
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <span className="text-sm font-medium text-gray-700">{label}</span>
          <span className={`text-sm font-semibold ${atLimit ? 'text-red-600' : 'text-gray-600'}`}>
            {used} / {unlimited ? <span className="text-green-600">Unlimited</span> : max}
          </span>
        </div>
        <div className="w-full bg-gray-100 rounded-full h-2">
          <div className={`h-2 rounded-full transition-all ${atLimit ? 'bg-red-500' : pct > 75 ? 'bg-amber-500' : 'bg-blue-500'}`}
            style={{ width: unlimited ? '0%' : `${pct}%` }} />
        </div>
      </div>
    );
  };

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-800">License</h1>
        <p className="text-sm text-gray-400 mt-1">Manage your Panelox license</p>
      </div>

      {/* Current tier */}
      <div className={`bg-gradient-to-r ${isPro ? 'from-blue-50 to-indigo-50 border-blue-200' : 'from-gray-50 to-gray-100 border-gray-200'} border rounded-xl p-6 mb-6`}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className={`w-14 h-14 rounded-xl flex items-center justify-center text-2xl ${isPro ? 'bg-blue-100' : 'bg-gray-100'}`}>
              {isPro ? '⚡' : '🆓'}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-xl font-bold text-gray-800">Panelox {isPro ? 'Pro' : 'Free'}</h2>
                <span className={`py-0.5 px-2.5 text-xs font-bold rounded-full ${isPro ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-600'}`}>
                  {isPro ? 'PRO' : 'FREE'}
                </span>
              </div>
              {status.license ? (
                <div className="text-sm text-gray-500 mt-1 flex items-center gap-3">
                  {status.license.customer_name && <span>{status.license.customer_name}</span>}
                  <span className="font-mono text-xs text-gray-400">{status.license.license_key}</span>
                </div>
              ) : (
                <p className="text-sm text-gray-500 mt-1">Free tier - limited resources</p>
              )}
            </div>
          </div>
          {isPro && (
            <button onClick={handleDeactivate}
              className="py-2 px-4 text-xs font-medium rounded-lg border border-gray-200 text-gray-500 hover:text-red-600 transition-colors">
              Deactivate
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Usage */}
        <div className="bg-white border border-gray-200 rounded-xl p-6">
          <h3 className="font-semibold text-sm text-gray-800 mb-4">Resource Usage</h3>
          <div className="space-y-4">
            {resourceBar('Websites', status.used_websites, status.max_websites)}
            {resourceBar('Server Nodes', status.used_nodes, status.max_nodes)}
            {resourceBar('Databases', status.used_databases, status.max_databases)}
            {resourceBar('App Installs', status.used_apps, status.max_apps)}
          </div>
        </div>

        {/* Activate */}
        <div className="bg-white border border-gray-200 rounded-xl p-6">
          <h3 className="font-semibold text-sm text-gray-800 mb-2">Activate License</h3>
          <p className="text-xs text-gray-500 mb-4">
            Enter your license key to unlock unlimited resources.
            {' '}<a href="https://panelox.com" target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">Buy a license</a>
          </p>
          <div className="space-y-3">
            <input type="text" value={keyInput} onChange={e => setKeyInput(e.target.value)}
              placeholder="PNLX-XXXX-XXXX-XXXX"
              onKeyDown={e => e.key === 'Enter' && handleActivate()}
              className="w-full py-2.5 px-3 border border-gray-200 rounded-lg text-sm font-mono focus:ring-2 focus:ring-blue-500/20 focus:border-blue-400 transition-all" />
            <button onClick={handleActivate} disabled={activating || !keyInput.trim()}
              className="w-full py-2.5 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 transition-all">
              {activating ? 'Validating...' : 'Activate'}
            </button>
          </div>
          {message && (
            <div className={`mt-3 text-xs px-3 py-2 rounded-lg ${
              message.type === 'success' ? 'bg-green-50 text-green-700 border border-green-200' : 'bg-red-50 text-red-700 border border-red-200'
            }`}>{message.text}</div>
          )}
        </div>
      </div>
    </div>
  );
}
