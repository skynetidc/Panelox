import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { dnsApi, type Zone, type Record as DnsRecord } from '../../api/dns';
import { ArrowLeft, Plus, Trash2, Download, Save } from 'lucide-react';

const recordTypes = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'SRV', 'NS', 'CAA', 'PTR'];

export default function ZoneDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [zone, setZone] = useState<Zone | null>(null);
  const [records, setRecords] = useState<DnsRecord[]>([]);
  const [loading, setLoading] = useState(true);

  // New record form
  const [showAdd, setShowAdd] = useState(false);
  const [newRec, setNewRec] = useState({ name: '', type: 'A', content: '', ttl: 3600, priority: undefined as number | undefined });

  // Editing
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editData, setEditData] = useState({ name: '', type: '', content: '', ttl: 0, priority: undefined as number | undefined });

  const zoneId = Number(id);

  const fetchZone = async () => {
    setLoading(true);
    try {
      const res = await dnsApi.getZone(zoneId);
      setZone(res.data.data);
      setRecords(res.data.data.records || []);
    } catch { navigate('/dns'); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchZone(); }, [zoneId]);

  const handleAddRecord = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await dnsApi.createRecord(zoneId, {
        name: newRec.name,
        type: newRec.type,
        content: newRec.content,
        ttl: newRec.ttl,
        priority: newRec.priority,
      });
      setNewRec({ name: '', type: 'A', content: '', ttl: 3600, priority: undefined });
      setShowAdd(false);
      fetchZone();
    } catch { /* ignore */ }
  };

  const startEdit = (rec: DnsRecord) => {
    setEditingId(rec.id);
    setEditData({ name: rec.name, type: rec.type, content: rec.content, ttl: rec.ttl, priority: rec.priority });
  };

  const handleSaveEdit = async () => {
    if (!editingId) return;
    try {
      await dnsApi.updateRecord(editingId, editData);
      setEditingId(null);
      fetchZone();
    } catch { /* ignore */ }
  };

  const handleDelete = async (recId: number) => {
    if (!confirm('Delete this record?')) return;
    await dnsApi.deleteRecord(recId).catch(() => {});
    fetchZone();
  };

  const handleExport = async () => {
    try {
      const res = await dnsApi.exportZone(zoneId);
      const blob = new Blob([res.data as string], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${zone?.name || 'zone'}.txt`;
      a.click();
      URL.revokeObjectURL(url);
    } catch { /* ignore */ }
  };

  if (loading || !zone) {
    return <div className="text-center text-gray-400 py-12">{t('common.loading')}</div>;
  }

  const typeColors: { [key: string]: string } = {
    A: 'bg-blue-100 text-blue-700',
    AAAA: 'bg-indigo-100 text-indigo-700',
    CNAME: 'bg-purple-100 text-purple-700',
    MX: 'bg-orange-100 text-orange-700',
    TXT: 'bg-gray-100 text-gray-700',
    NS: 'bg-green-100 text-green-700',
    SRV: 'bg-cyan-100 text-cyan-700',
    CAA: 'bg-yellow-100 text-yellow-700',
    PTR: 'bg-pink-100 text-pink-700',
  };

  return (
    <div>
      <button onClick={() => navigate('/dns')} className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4">
        <ArrowLeft size={16} /> {t('common.back')}
      </button>

      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{zone.name}</h1>
          <p className="text-sm text-gray-500">Serial: {zone.serial} &middot; {records.length} records</p>
        </div>
        <div className="flex gap-2">
          <button onClick={handleExport} className="flex items-center gap-1 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50">
            <Download size={14} /> Export
          </button>
          <button onClick={() => setShowAdd(!showAdd)} className="flex items-center gap-1 px-3 py-2 bg-blue-600 text-white rounded-lg text-sm hover:bg-blue-700">
            <Plus size={14} /> Add Record
          </button>
        </div>
      </div>

      {/* Add record form */}
      {showAdd && (
        <form onSubmit={handleAddRecord} className="bg-white rounded-xl shadow-sm border p-4 mb-4">
          <div className="grid grid-cols-6 gap-3 items-end">
            <div>
              <label className="block text-xs text-gray-500 mb-1">Name</label>
              <input type="text" value={newRec.name} onChange={e => setNewRec(r => ({ ...r, name: e.target.value }))} placeholder="@" className="w-full px-2 py-1.5 border rounded text-sm outline-none" />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">Type</label>
              <select value={newRec.type} onChange={e => setNewRec(r => ({ ...r, type: e.target.value }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none">
                {recordTypes.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div className="col-span-2">
              <label className="block text-xs text-gray-500 mb-1">Content</label>
              <input type="text" value={newRec.content} onChange={e => setNewRec(r => ({ ...r, content: e.target.value }))} placeholder="1.2.3.4" className="w-full px-2 py-1.5 border rounded text-sm outline-none" />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">TTL</label>
              <input type="number" value={newRec.ttl} onChange={e => setNewRec(r => ({ ...r, ttl: Number(e.target.value) }))} className="w-full px-2 py-1.5 border rounded text-sm outline-none" />
            </div>
            <div className="flex gap-2">
              <button type="submit" className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Add</button>
              <button type="button" onClick={() => setShowAdd(false)} className="px-3 py-1.5 border rounded text-sm">Cancel</button>
            </div>
          </div>
        </form>
      )}

      {/* Records table */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-100 text-left text-xs text-gray-500 uppercase">
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Type</th>
              <th className="px-4 py-3">Content</th>
              <th className="px-4 py-3 w-20">TTL</th>
              <th className="px-4 py-3 w-20">Priority</th>
              <th className="px-4 py-3 w-24"></th>
            </tr>
          </thead>
          <tbody>
            {records.map(rec => (
              <tr key={rec.id} className="border-b border-gray-50 hover:bg-gray-50">
                {editingId === rec.id ? (
                  <>
                    <td className="px-4 py-2"><input type="text" value={editData.name} onChange={e => setEditData(d => ({ ...d, name: e.target.value }))} className="w-full px-2 py-1 border rounded text-sm" /></td>
                    <td className="px-4 py-2">
                      <select value={editData.type} onChange={e => setEditData(d => ({ ...d, type: e.target.value }))} className="px-2 py-1 border rounded text-sm">
                        {recordTypes.map(t => <option key={t} value={t}>{t}</option>)}
                      </select>
                    </td>
                    <td className="px-4 py-2"><input type="text" value={editData.content} onChange={e => setEditData(d => ({ ...d, content: e.target.value }))} className="w-full px-2 py-1 border rounded text-sm" /></td>
                    <td className="px-4 py-2"><input type="number" value={editData.ttl} onChange={e => setEditData(d => ({ ...d, ttl: Number(e.target.value) }))} className="w-full px-2 py-1 border rounded text-sm" /></td>
                    <td className="px-4 py-2"><input type="number" value={editData.priority ?? ''} onChange={e => setEditData(d => ({ ...d, priority: e.target.value ? Number(e.target.value) : undefined }))} className="w-full px-2 py-1 border rounded text-sm" /></td>
                    <td className="px-4 py-2 flex gap-1">
                      <button onClick={handleSaveEdit} className="p-1 text-green-600"><Save size={14} /></button>
                      <button onClick={() => setEditingId(null)} className="p-1 text-gray-400">Cancel</button>
                    </td>
                  </>
                ) : (
                  <>
                    <td className="px-4 py-3 font-mono text-sm">{rec.name}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${typeColors[rec.type] || 'bg-gray-100'}`}>{rec.type}</span>
                    </td>
                    <td className="px-4 py-3 font-mono text-sm text-gray-700 max-w-xs truncate">{rec.content}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{rec.ttl}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{rec.priority ?? '—'}</td>
                    <td className="px-4 py-3 flex gap-1">
                      <button onClick={() => startEdit(rec)} className="p-1 text-gray-400 hover:text-blue-600 text-xs">Edit</button>
                      <button onClick={() => handleDelete(rec.id)} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </td>
                  </>
                )}
              </tr>
            ))}
            {records.length === 0 && (
              <tr><td colSpan={6} className="px-6 py-8 text-center text-gray-400">No records</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
