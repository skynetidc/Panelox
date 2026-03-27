import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { websiteApi, type Website } from '../../api/websites';

export default function WebsiteList() {
  const { t } = useTranslation();
  const [websites, setWebsites] = useState<Website[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  const fetch = async () => { setLoading(true); try { const res = await websiteApi.list({ page: 1, size: 50, search }); setWebsites(res.data.data || []); } catch { setWebsites([]); } finally { setLoading(false); } };
  useEffect(() => { fetch(); }, [search]);

  const handleOperate = async (id: number, action: 'start' | 'stop') => { await websiteApi.operate(id, action).catch(() => {}); fetch(); };
  const handleDelete = async (id: number, domain: string) => { if (!confirm(`Delete website ${domain}?`)) return; await websiteApi.delete(id).catch(() => {}); fetch(); };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-800">{t('websites.title')}</h1>
        <Link to="/websites/new" className="py-2 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">{t('websites.create')}</Link>
      </div>

      <div className="relative mb-4">
        <div className="absolute inset-y-0 start-0 flex items-center ps-3.5 pointer-events-none"><svg className="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg></div>
        <input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder={t('common.search')} className="py-2.5 ps-10 pe-4 block w-full border border-gray-200 rounded-lg text-sm focus:border-blue-500 focus:ring-blue-500" />
      </div>

      <div className="flex flex-col">
        <div className="overflow-x-auto">
          <div className="min-w-full inline-block align-middle">
            <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
              {loading ? <div className="p-12 text-center text-gray-400">{t('common.loading')}</div> : websites.length === 0 ? (
                <div className="p-12 text-center"><p className="text-gray-400">{t('websites.noWebsites')}</p></div>
              ) : (
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">{t('websites.domain')}</th>
                      <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">{t('websites.type')}</th>
                      <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">{t('websites.ssl')}</th>
                      <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">{t('common.status')}</th>
                      <th className="px-6 py-3 text-end text-xs font-semibold uppercase tracking-wide text-gray-500">{t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {websites.map(site => (
                      <tr key={site.id} className="hover:bg-gray-50">
                        <td className="px-6 py-4"><Link to={`/websites/${site.id}`} className="text-sm font-medium text-blue-600 hover:underline">{site.domain}</Link>{site.remark && <p className="text-xs text-gray-400 mt-0.5">{site.remark}</p>}</td>
                        <td className="px-6 py-4"><span className={`py-1 px-2 inline-flex items-center text-xs font-medium rounded-full ${site.type === 'php' ? 'bg-purple-100 text-purple-700' : site.type === 'proxy' ? 'bg-amber-100 text-amber-700' : 'bg-blue-100 text-blue-700'}`}>{site.type}</span></td>
                        <td className="px-6 py-4">{site.ssl_enabled ? <span className="py-1 px-2 inline-flex items-center text-xs font-medium rounded-full bg-green-100 text-green-700">HTTPS</span> : <span className="text-sm text-gray-400">HTTP</span>}</td>
                        <td className="px-6 py-4"><div className="flex items-center gap-2"><span className={`w-2 h-2 rounded-full ${site.status === 'active' ? 'bg-green-500' : 'bg-gray-300'}`} /><span className="text-sm">{site.status === 'active' ? t('common.active') : t('common.inactive')}</span></div></td>
                        <td className="px-6 py-4 text-end">
                          <div className="inline-flex gap-1">
                            <a href={`http://${site.domain}`} target="_blank" className="p-1.5 rounded-lg text-gray-400 hover:text-blue-600 hover:bg-blue-50"><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" /></svg></a>
                            {site.status === 'active' ? <button onClick={() => handleOperate(site.id, 'stop')} className="p-1.5 rounded-lg text-gray-400 hover:text-amber-600 hover:bg-amber-50"><svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M6 4h4v16H6zM14 4h4v16h-4z" /></svg></button>
                            : <button onClick={() => handleOperate(site.id, 'start')} className="p-1.5 rounded-lg text-gray-400 hover:text-green-600 hover:bg-green-50"><svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z" /></svg></button>}
                            <button onClick={() => handleDelete(site.id, site.domain)} className="p-1.5 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50"><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg></button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
