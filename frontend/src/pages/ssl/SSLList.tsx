import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { sslApi, type Certificate } from '../../api/ssl';

export default function SSLList() {
  const { t } = useTranslation();
  const [certs, setCerts] = useState<Certificate[]>([]);
  const [loading, setLoading] = useState(true);

  const fetch = async () => { setLoading(true); try { const res = await sslApi.list({ page: 1, size: 50 }); setCerts(res.data.data || []); } catch { setCerts([]); } finally { setLoading(false); } };
  useEffect(() => { fetch(); }, []);

  const handleDelete = async (id: number) => { if (!confirm('Delete this certificate?')) return; await sslApi.delete(id).catch(() => {}); fetch(); };
  const handleRenew = async (id: number) => { await sslApi.renew(id).catch(() => {}); fetch(); };

  const statusColors: Record<string, string> = { issued: 'bg-green-100 text-green-700', issuing: 'bg-yellow-100 text-yellow-700', pending: 'bg-gray-100 text-gray-700', failed: 'bg-red-100 text-red-700' };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-800">{t('nav.ssl')}</h1>
        <Link to="/ssl/new" className="py-2 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">New Certificate</Link>
      </div>

      <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
        {loading ? <div className="p-12 text-center text-gray-400">{t('common.loading')}</div> : certs.length === 0 ? (
          <div className="p-12 text-center"><p className="text-gray-400">No SSL certificates</p></div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50"><tr>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Domains</th>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Provider</th>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Status</th>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Expires</th>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Auto-Renew</th>
              <th className="px-6 py-3 text-end"></th>
            </tr></thead>
            <tbody className="divide-y divide-gray-200">
              {certs.map(c => (
                <tr key={c.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4"><span className="text-sm font-medium">{c.domains?.join(', ') || '—'}</span>{c.description && <p className="text-xs text-gray-400 mt-0.5">{c.description}</p>}</td>
                  <td className="px-6 py-4"><span className={`py-1 px-2 inline-flex text-xs font-medium rounded-full ${c.provider === 'letsencrypt' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}`}>{c.provider === 'letsencrypt' ? "Let's Encrypt" : 'Custom'}</span></td>
                  <td className="px-6 py-4"><span className={`py-1 px-2 inline-flex text-xs font-medium rounded-full ${statusColors[c.status] || 'bg-gray-100'}`}>{c.status}</span></td>
                  <td className="px-6 py-4 text-sm text-gray-500">{c.expires_at ? new Date(c.expires_at).toLocaleDateString() : '—'}</td>
                  <td className="px-6 py-4">{c.auto_renew ? <span className="text-green-600 text-sm">Yes</span> : <span className="text-gray-400 text-sm">No</span>}</td>
                  <td className="px-6 py-4 text-end">
                    <div className="inline-flex gap-1">
                      {c.provider === 'letsencrypt' && c.status === 'issued' && <button onClick={() => handleRenew(c.id)} className="p-1.5 rounded-lg text-gray-400 hover:text-blue-600 hover:bg-blue-50"><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182" /></svg></button>}
                      <button onClick={() => handleDelete(c.id)} className="p-1.5 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50"><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
