import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { settingsApi, type AuditEntry } from '../../api/settings';
import { Save, Clock } from 'lucide-react';

type Tab = 'general' | 'audit' | 'branding';

interface Branding {
  panel_name: string; logo_url: string; favicon_url: string; primary_color: string;
  login_title: string; login_subtitle: string; footer_text: string; hide_powered_by: boolean;
}

export default function General() {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>('general');
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [audit, setAudit] = useState<AuditEntry[]>([]);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditPage, setAuditPage] = useState(1);
  const [branding, setBranding] = useState<Branding>({ panel_name: 'Panelox', logo_url: '', favicon_url: '', primary_color: '#3b82f6', login_title: 'Server management,', login_subtitle: 'simplified.', footer_text: '', hide_powered_by: false });
  const [brandingSaving, setBrandingSaving] = useState(false);

  useEffect(() => {
    settingsApi.getAll().then(r => setSettings(r.data.data || {})).catch(() => {});
  }, []);

  useEffect(() => {
    if (tab === 'branding') {
      settingsApi.getBranding().then(r => setBranding(r.data.data)).catch(() => {});
    }
    if (tab === 'audit') {
      settingsApi.listAudit({ page: auditPage, size: 50 }).then(r => {
        setAudit(r.data.data || []);
        setAuditTotal(r.data.meta?.total || 0);
      }).catch(() => {});
    }
  }, [tab, auditPage]);

  const handleSave = async () => {
    setSaving(true);
    await settingsApi.update(settings).catch(() => {});
    setSaving(false);
  };

  const updateSetting = (key: string, value: string) => {
    setSettings(s => ({ ...s, [key]: value }));
  };

  const tabs: { key: Tab; label: string }[] = [
    { key: 'general', label: t('nav.settings') },
    { key: 'branding', label: 'White-label' },
    { key: 'audit', label: 'Audit Log' },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-4">{t('nav.settings')}</h1>

      <div className="border-b border-gray-200 mb-6">
        <nav className="flex gap-6">
          {tabs.map(tb => (
            <button key={tb.key} onClick={() => setTab(tb.key)}
              className={`pb-3 text-sm font-medium border-b-2 transition-colors ${tab === tb.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}`}>
              {tb.label}
            </button>
          ))}
        </nav>
      </div>

      {tab === 'general' && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 space-y-5 max-w-2xl">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Panel Name</label>
            <input type="text" value={settings.panel_name || ''} onChange={e => updateSetting('panel_name', e.target.value)} placeholder="Panelox" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Server IP</label>
            <input type="text" value={settings.server_ip || ''} onChange={e => updateSetting('server_ip', e.target.value)} placeholder="0.0.0.0" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Backup Path</label>
            <input type="text" value={settings.backup_path || ''} onChange={e => updateSetting('backup_path', e.target.value)} placeholder="/opt/panelox/backups" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Default Language</label>
            <select value={settings.default_language || 'en'} onChange={e => updateSetting('default_language', e.target.value)} className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500">
              <option value="en">English</option>
              <option value="cs">Čeština</option>
            </select>
          </div>
          <div className="pt-4 border-t">
            <button onClick={handleSave} disabled={saving} className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700 disabled:opacity-50">
              <Save size={14} /> {saving ? 'Saving...' : t('common.save')}
            </button>
          </div>
        </div>
      )}

      {tab === 'branding' && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 space-y-5 max-w-2xl">
          <div className="flex items-center gap-2 mb-2">
            <h3 className="font-semibold text-gray-800">White-label Branding</h3>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 font-bold">PRO</span>
          </div>
          <p className="text-xs text-gray-500">Customize the panel appearance. Requires a Pro license.</p>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Panel Name</label>
              <input type="text" value={branding.panel_name} onChange={e => setBranding(b => ({ ...b, panel_name: e.target.value }))} placeholder="Panelox" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Primary Color</label>
              <div className="flex gap-2">
                <input type="color" value={branding.primary_color} onChange={e => setBranding(b => ({ ...b, primary_color: e.target.value }))} className="w-10 h-9 rounded border cursor-pointer" />
                <input type="text" value={branding.primary_color} onChange={e => setBranding(b => ({ ...b, primary_color: e.target.value }))} className="flex-1 px-3 py-2 border rounded-lg text-sm font-mono outline-none focus:ring-2 focus:ring-blue-500" />
              </div>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Logo URL</label>
            <input type="text" value={branding.logo_url} onChange={e => setBranding(b => ({ ...b, logo_url: e.target.value }))} placeholder="https://example.com/logo.png" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Favicon URL</label>
            <input type="text" value={branding.favicon_url} onChange={e => setBranding(b => ({ ...b, favicon_url: e.target.value }))} placeholder="https://example.com/favicon.ico" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Login Title</label>
              <input type="text" value={branding.login_title} onChange={e => setBranding(b => ({ ...b, login_title: e.target.value }))} placeholder="Server management," className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Login Subtitle</label>
              <input type="text" value={branding.login_subtitle} onChange={e => setBranding(b => ({ ...b, login_subtitle: e.target.value }))} placeholder="simplified." className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Footer Text</label>
            <input type="text" value={branding.footer_text} onChange={e => setBranding(b => ({ ...b, footer_text: e.target.value }))} placeholder="© 2026 Your Company" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>

          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={branding.hide_powered_by} onChange={e => setBranding(b => ({ ...b, hide_powered_by: e.target.checked }))} className="rounded border-gray-300" />
            <span className="text-sm text-gray-700">Hide "Powered by Panelox" branding</span>
          </label>

          {branding.logo_url && (
            <div className="bg-gray-50 rounded-lg p-4">
              <p className="text-xs text-gray-500 mb-2">Preview:</p>
              <div className="flex items-center gap-3">
                <img src={branding.logo_url} alt="Logo" className="h-8 rounded" onError={e => (e.target as HTMLImageElement).style.display = 'none'} />
                <span className="font-bold" style={{ color: branding.primary_color }}>{branding.panel_name}</span>
              </div>
            </div>
          )}

          <div className="pt-4 border-t">
            <button onClick={async () => {
              setBrandingSaving(true);
              await settingsApi.updateBranding(branding).catch(() => {});
              setBrandingSaving(false);
            }} disabled={brandingSaving} className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700 disabled:opacity-50">
              <Save size={14} /> {brandingSaving ? 'Saving...' : 'Save Branding'}
            </button>
          </div>
        </div>
      )}

      {tab === 'audit' && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 text-left text-xs text-gray-500 uppercase">
                <th className="px-4 py-3">Time</th>
                <th className="px-4 py-3">User</th>
                <th className="px-4 py-3">Action</th>
                <th className="px-4 py-3">Resource</th>
                <th className="px-4 py-3">IP</th>
              </tr>
            </thead>
            <tbody>
              {audit.map(e => (
                <tr key={e.id} className="border-b border-gray-50 hover:bg-gray-50">
                  <td className="px-4 py-2 text-xs text-gray-500">
                    <div className="flex items-center gap-1">
                      <Clock size={12} />
                      {new Date(e.created_at).toLocaleString()}
                    </div>
                  </td>
                  <td className="px-4 py-2 text-sm">{e.user_id || '—'}</td>
                  <td className="px-4 py-2 text-xs font-mono">{e.action}</td>
                  <td className="px-4 py-2 text-sm text-gray-600">{e.resource}</td>
                  <td className="px-4 py-2 text-xs text-gray-500 font-mono">{e.ip_address}</td>
                </tr>
              ))}
              {audit.length === 0 && (
                <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-400">No audit entries</td></tr>
              )}
            </tbody>
          </table>
          {auditTotal > 50 && (
            <div className="flex items-center justify-between px-6 py-3 border-t border-gray-100">
              <span className="text-sm text-gray-500">{auditTotal} entries</span>
              <div className="flex gap-2">
                <button onClick={() => setAuditPage(p => Math.max(1, p - 1))} disabled={auditPage === 1} className="px-3 py-1 text-sm border rounded disabled:opacity-50">Prev</button>
                <span className="px-3 py-1 text-sm">{auditPage}</span>
                <button onClick={() => setAuditPage(p => p + 1)} disabled={auditPage * 50 >= auditTotal} className="px-3 py-1 text-sm border rounded disabled:opacity-50">Next</button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
