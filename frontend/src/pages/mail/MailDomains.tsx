import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { mailApi, type MailDomain, type Mailbox } from '../../api/mail';
import { Plus, Trash2, Mail, ChevronDown, ChevronRight } from 'lucide-react';

export default function MailDomains() {
  const { t } = useTranslation();
  const [domains, setDomains] = useState<MailDomain[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newDomain, setNewDomain] = useState('');
  const [expandedId, setExpandedId] = useState<number | null>(null);
  const [mailboxes, setMailboxes] = useState<Mailbox[]>([]);
  const [newMb, setNewMb] = useState({ address: '', password: '', quota: 1024 });

  const fetch = async () => {
    setLoading(true);
    try {
      const res = await mailApi.listDomains({ page: 1, size: 50 });
      setDomains(res.data.data || []);
    } catch { setDomains([]); }
    finally { setLoading(false); }
  };

  useEffect(() => { fetch(); }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newDomain) return;
    await mailApi.createDomain(newDomain).catch(() => {});
    setNewDomain(''); setShowCreate(false); fetch();
  };

  const handleDeleteDomain = async (id: number) => {
    if (!confirm('Delete this mail domain and all mailboxes?')) return;
    await mailApi.deleteDomain(id).catch(() => {}); fetch();
  };

  const toggleExpand = async (id: number) => {
    if (expandedId === id) { setExpandedId(null); return; }
    setExpandedId(id);
    try {
      const res = await mailApi.listMailboxes(id);
      setMailboxes(res.data.data || []);
    } catch { setMailboxes([]); }
  };

  const handleAddMailbox = async (domainId: number) => {
    if (!newMb.address || !newMb.password) return;
    await mailApi.createMailbox(domainId, newMb.address, newMb.password, newMb.quota).catch(() => {});
    setNewMb({ address: '', password: '', quota: 1024 });
    const res = await mailApi.listMailboxes(domainId);
    setMailboxes(res.data.data || []);
  };

  const handleDeleteMailbox = async (id: number, domainId: number) => {
    await mailApi.deleteMailbox(id).catch(() => {});
    const res = await mailApi.listMailboxes(domainId);
    setMailboxes(res.data.data || []);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{t('nav.mail')}</h1>
        <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
          <Plus size={18} /> Add Domain
        </button>
      </div>

      {showCreate && (
        <form onSubmit={handleCreate} className="bg-white rounded-xl shadow-sm border p-4 mb-4 flex gap-3 items-end">
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">Mail Domain</label>
            <input type="text" value={newDomain} onChange={e => setNewDomain(e.target.value)} placeholder="example.com" className="w-full px-3 py-2 border rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500" />
          </div>
          <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Create</button>
          <button type="button" onClick={() => setShowCreate(false)} className="px-4 py-2 border rounded-lg text-sm">Cancel</button>
        </form>
      )}

      <div className="space-y-3">
        {loading ? (
          <div className="bg-white rounded-xl p-12 text-center text-gray-400">{t('common.loading')}</div>
        ) : domains.length === 0 ? (
          <div className="bg-white rounded-xl p-12 text-center">
            <Mail size={48} className="mx-auto mb-4 text-gray-300" />
            <p className="text-gray-400">No mail domains yet</p>
          </div>
        ) : domains.map(d => (
          <div key={d.id} className="bg-white rounded-xl shadow-sm border border-gray-100">
            <div className="flex items-center justify-between px-6 py-4 cursor-pointer" onClick={() => toggleExpand(d.id)}>
              <div className="flex items-center gap-3">
                {expandedId === d.id ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                <Mail size={20} className="text-orange-500" />
                <div>
                  <p className="font-medium">{d.domain}</p>
                  <p className="text-xs text-gray-500">{d.status}</p>
                </div>
              </div>
              <button onClick={(e) => { e.stopPropagation(); handleDeleteDomain(d.id); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
            </div>

            {expandedId === d.id && (
              <div className="border-t border-gray-100 px-6 py-4">
                <h3 className="text-sm font-semibold text-gray-700 mb-2">Mailboxes</h3>
                <div className="space-y-1 mb-3">
                  {mailboxes.map(m => (
                    <div key={m.id} className="flex items-center justify-between text-sm bg-gray-50 px-3 py-2 rounded">
                      <div>
                        <span className="font-mono">{m.address}@{d.domain}</span>
                        <span className="text-xs text-gray-400 ml-2">{m.quota_mb} MB</span>
                      </div>
                      <button onClick={() => handleDeleteMailbox(m.id, d.id)} className="text-gray-400 hover:text-red-600"><Trash2 size={12} /></button>
                    </div>
                  ))}
                  {mailboxes.length === 0 && <p className="text-xs text-gray-400">No mailboxes</p>}
                </div>
                <div className="flex gap-2">
                  <input type="text" value={newMb.address} onChange={e => setNewMb(m => ({ ...m, address: e.target.value }))} placeholder="user" className="flex-1 px-2 py-1 border rounded text-sm outline-none" />
                  <span className="py-1 text-sm text-gray-400">@{d.domain}</span>
                  <input type="password" value={newMb.password} onChange={e => setNewMb(m => ({ ...m, password: e.target.value }))} placeholder="password" className="flex-1 px-2 py-1 border rounded text-sm outline-none" />
                  <button onClick={() => handleAddMailbox(d.id)} className="px-3 py-1 bg-blue-600 text-white rounded text-xs">Add</button>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
