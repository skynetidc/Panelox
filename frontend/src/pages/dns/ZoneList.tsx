import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { dnsApi, type Zone } from '../../api/dns';

export default function ZoneList() {
  const { t } = useTranslation();
  const [zones, setZones] = useState<Zone[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newZone, setNewZone] = useState('');
  const [creating, setCreating] = useState(false);

  const fetchZones = async () => { setLoading(true); try { const res = await dnsApi.listZones({ page: 1, size: 50 }); setZones(res.data.data || []); } catch { setZones([]); } finally { setLoading(false); } };
  useEffect(() => { fetchZones(); }, []);

  const handleCreate = async (e: React.FormEvent) => { e.preventDefault(); if (!newZone) return; setCreating(true); await dnsApi.createZone(newZone).catch(() => {}); setNewZone(''); setShowCreate(false); setCreating(false); fetchZones(); };
  const handleDelete = async (id: number, name: string) => { if (!confirm(`Delete zone "${name}"?`)) return; await dnsApi.deleteZone(id).catch(() => {}); fetchZones(); };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-800">{t('nav.dns')}</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="py-2 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">New Zone</button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="bg-white border border-gray-200 rounded-xl shadow-sm p-4 mb-4 flex gap-3 items-end">
          <div className="flex-1">
            <label className="block text-sm font-medium mb-1 text-gray-700">Domain Name</label>
            <input type="text" value={newZone} onChange={e => setNewZone(e.target.value)} placeholder="example.com" autoFocus className="py-2.5 px-4 block w-full border border-gray-200 rounded-lg text-sm focus:border-blue-500 focus:ring-blue-500" />
          </div>
          <button type="submit" disabled={creating} className="py-2.5 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">{creating ? 'Creating...' : 'Create'}</button>
          <button type="button" onClick={() => setShowCreate(false)} className="py-2.5 px-4 text-sm font-medium rounded-lg border border-gray-200 text-gray-700 hover:bg-gray-50">Cancel</button>
        </form>
      )}

      <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
        {loading ? <div className="p-12 text-center text-gray-400">{t('common.loading')}</div> : zones.length === 0 ? (
          <div className="p-12 text-center"><p className="text-gray-400">No DNS zones yet</p></div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50"><tr>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Zone</th>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Records</th>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Serial</th>
              <th className="px-6 py-3 text-start text-xs font-semibold uppercase tracking-wide text-gray-500">Status</th>
              <th className="px-6 py-3 text-end"></th>
            </tr></thead>
            <tbody className="divide-y divide-gray-200">
              {zones.map(z => (
                <tr key={z.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4"><Link to={`/dns/${z.id}`} className="text-sm font-medium text-blue-600 hover:underline">{z.name}</Link></td>
                  <td className="px-6 py-4"><span className="py-1 px-2 inline-flex text-xs font-medium rounded-full bg-blue-100 text-blue-700">{z.records?.length || 0}</span></td>
                  <td className="px-6 py-4 text-sm text-gray-500 font-mono">{z.serial}</td>
                  <td className="px-6 py-4"><span className="py-1 px-2 inline-flex text-xs font-medium rounded-full bg-green-100 text-green-700">{z.status}</span></td>
                  <td className="px-6 py-4 text-end"><button onClick={() => handleDelete(z.id, z.name)} className="p-1.5 rounded-lg text-gray-400 hover:text-red-600 hover:bg-red-50"><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg></button></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
