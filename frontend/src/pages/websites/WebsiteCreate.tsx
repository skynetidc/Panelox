import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { websiteApi, type CreateWebsiteRequest } from '../../api/websites';
import { Card } from 'primereact/card';
import { InputText } from 'primereact/inputtext';
import { Dropdown } from 'primereact/dropdown';
import { Button } from 'primereact/button';
import { Message } from 'primereact/message';

const websiteTypes = [
  { value: 'static', label: 'Static HTML' },
  { value: 'proxy', label: 'Reverse Proxy' },
  { value: 'php', label: 'PHP' },
  { value: 'node', label: 'Node.js' },
];

const phpVersions = [
  { value: '8.3', label: 'PHP 8.3' },
  { value: '8.2', label: 'PHP 8.2' },
  { value: '8.1', label: 'PHP 8.1' },
  { value: '7.4', label: 'PHP 7.4' },
];

export default function WebsiteCreate() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [form, setForm] = useState<CreateWebsiteRequest>({ domain: '', type: 'static', remark: '' });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setLoading(true);
    try { const res = await websiteApi.create(form); navigate(`/websites/${res.data.data.id}`); }
    catch (err: any) { setError(err.response?.data?.error?.message || 'Failed'); }
    finally { setLoading(false); }
  };

  return (
    <div className="max-w-2xl">
      <Button label={t('common.back')} icon="pi pi-arrow-left" text className="mb-4" onClick={() => navigate('/websites')} />
      <h1 className="text-3xl font-bold mb-6">{t('websites.create')}</h1>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-5">
          {error && <Message severity="error" text={error} className="w-full" />}

          <div className="flex flex-col gap-1">
            <label className="text-sm font-semibold">{t('websites.domain')} *</label>
            <InputText value={form.domain} onChange={e => setForm(f => ({ ...f, domain: e.target.value }))} required placeholder="example.com" />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-semibold">Alias</label>
            <InputText value={form.alias || ''} onChange={e => setForm(f => ({ ...f, alias: e.target.value }))} placeholder="auto-generated" />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-sm font-semibold">{t('websites.type')} *</label>
            <Dropdown value={form.type} onChange={e => setForm(f => ({ ...f, type: e.value }))} options={websiteTypes} optionLabel="label" optionValue="value" className="w-full" />
          </div>

          {(form.type === 'proxy' || form.type === 'node') && (
            <div className="flex flex-col gap-1">
              <label className="text-sm font-semibold">Proxy Target *</label>
              <InputText value={form.proxy_target || ''} onChange={e => setForm(f => ({ ...f, proxy_target: e.target.value }))} placeholder="http://127.0.0.1:3000" />
            </div>
          )}

          {form.type === 'php' && (
            <div className="flex flex-col gap-1">
              <label className="text-sm font-semibold">PHP Version</label>
              <Dropdown value={form.php_version || '8.3'} onChange={e => setForm(f => ({ ...f, php_version: e.value }))} options={phpVersions} optionLabel="label" optionValue="value" className="w-full" />
            </div>
          )}

          <div className="flex flex-col gap-1">
            <label className="text-sm font-semibold">Note</label>
            <InputText value={form.remark || ''} onChange={e => setForm(f => ({ ...f, remark: e.target.value }))} placeholder="Optional" />
          </div>

          <div className="flex justify-end gap-2 pt-4" style={{ borderTop: '1px solid #e5e7eb' }}>
            <Button label={t('common.cancel')} severity="secondary" outlined onClick={() => navigate('/websites')} />
            <Button label={loading ? t('common.loading') : t('common.create')} icon={loading ? 'pi pi-spin pi-spinner' : 'pi pi-check'} type="submit" disabled={loading} />
          </div>
        </form>
      </Card>
    </div>
  );
}
