import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { sslApi, type AcmeAccount } from '../../api/ssl';
import { ArrowLeft } from 'lucide-react';

type Mode = 'letsencrypt' | 'upload';

export default function SSLCreate() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [mode, setMode] = useState<Mode>('letsencrypt');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // ACME accounts
  const [accounts, setAccounts] = useState<AcmeAccount[]>([]);
  const [showNewAccount, setShowNewAccount] = useState(false);
  const [newEmail, setNewEmail] = useState('');

  // Let's Encrypt form
  const [domains, setDomains] = useState('');
  const [accountId, setAccountId] = useState<number>(0);
  const [autoRenew, setAutoRenew] = useState(true);
  const [description, setDescription] = useState('');

  // Upload form
  const [certPEM, setCertPEM] = useState('');
  const [keyPEM, setKeyPEM] = useState('');

  useEffect(() => {
    sslApi.listAccounts().then((r) => {
      const accts = r.data.data || [];
      setAccounts(accts);
      if (accts.length > 0) setAccountId(accts[0].id);
    }).catch(() => {});
  }, []);

  const handleCreateAccount = async () => {
    if (!newEmail) return;
    setError('');
    try {
      const res = await sslApi.createAccount(newEmail);
      const acct = res.data.data;
      setAccounts((prev) => [...prev, acct]);
      setAccountId(acct.id);
      setShowNewAccount(false);
      setNewEmail('');
    } catch (err: any) {
      setError(err.response?.data?.error?.message || 'Failed to create ACME account');
    }
  };

  const handleIssueLetsEncrypt = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!accountId) {
      setError('Please create an ACME account first');
      return;
    }

    const domainList = domains.split(/[\n,]+/).map((d) => d.trim()).filter(Boolean);
    if (domainList.length === 0) {
      setError('At least one domain is required');
      return;
    }

    setLoading(true);
    try {
      await sslApi.issue({
        domains: domainList,
        acme_account_id: accountId,
        auto_renew: autoRenew,
        description,
      });
      navigate('/ssl');
    } catch (err: any) {
      setError(err.response?.data?.error?.message || 'Failed to issue certificate');
    } finally {
      setLoading(false);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!certPEM || !keyPEM) {
      setError('Certificate and private key are required');
      return;
    }

    setLoading(true);
    try {
      await sslApi.upload({ certificate: certPEM, private_key: keyPEM, description });
      navigate('/ssl');
    } catch (err: any) {
      setError(err.response?.data?.error?.message || 'Failed to upload certificate');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl">
      <button
        onClick={() => navigate('/ssl')}
        className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 mb-4"
      >
        <ArrowLeft size={16} />
        {t('common.back')}
      </button>

      <h1 className="text-2xl font-bold text-gray-900 mb-6">New SSL Certificate</h1>

      {/* Mode selector */}
      <div className="flex gap-3 mb-6">
        <button
          onClick={() => setMode('letsencrypt')}
          className={`flex-1 p-4 rounded-xl border text-left transition-colors ${
            mode === 'letsencrypt' ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:border-gray-300'
          }`}
        >
          <p className="font-medium">Let's Encrypt</p>
          <p className="text-xs text-gray-500 mt-1">Free, auto-issued & auto-renewed</p>
        </button>
        <button
          onClick={() => setMode('upload')}
          className={`flex-1 p-4 rounded-xl border text-left transition-colors ${
            mode === 'upload' ? 'border-blue-500 bg-blue-50' : 'border-gray-200 hover:border-gray-300'
          }`}
        >
          <p className="font-medium">Upload Custom</p>
          <p className="text-xs text-gray-500 mt-1">Paste your own certificate & key</p>
        </button>
      </div>

      {error && (
        <div className="bg-red-50 text-red-600 text-sm px-4 py-3 rounded-lg mb-4">{error}</div>
      )}

      {mode === 'letsencrypt' && (
        <form onSubmit={handleIssueLetsEncrypt} className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 space-y-5">
          {/* ACME Account */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">ACME Account</label>
            {accounts.length > 0 ? (
              <div className="flex gap-2">
                <select
                  value={accountId}
                  onChange={(e) => setAccountId(Number(e.target.value))}
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
                >
                  {accounts.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.email} ({a.provider})
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  onClick={() => setShowNewAccount(!showNewAccount)}
                  className="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50"
                >
                  + New
                </button>
              </div>
            ) : (
              <p className="text-sm text-gray-500 mb-2">No ACME accounts yet. Create one to get started.</p>
            )}

            {(showNewAccount || accounts.length === 0) && (
              <div className="flex gap-2 mt-2">
                <input
                  type="email"
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  placeholder="your@email.com"
                  className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
                />
                <button
                  type="button"
                  onClick={handleCreateAccount}
                  className="px-4 py-2 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700"
                >
                  Register
                </button>
              </div>
            )}
          </div>

          {/* Domains */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Domains *</label>
            <textarea
              value={domains}
              onChange={(e) => setDomains(e.target.value)}
              rows={3}
              placeholder={"example.com\nwww.example.com"}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500 resize-y"
            />
            <p className="text-xs text-gray-400 mt-1">One domain per line. Wildcard (*.example.com) requires DNS validation.</p>
          </div>

          {/* Auto-renew */}
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={autoRenew}
              onChange={(e) => setAutoRenew(e.target.checked)}
              className="rounded"
            />
            <span className="text-sm text-gray-700">Auto-renew before expiry</span>
          </label>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t">
            <button type="button" onClick={() => navigate('/ssl')} className="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50">
              {t('common.cancel')}
            </button>
            <button type="submit" disabled={loading || !accountId} className="px-6 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
              {loading ? 'Issuing...' : 'Issue Certificate'}
            </button>
          </div>
        </form>
      )}

      {mode === 'upload' && (
        <form onSubmit={handleUpload} className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 space-y-5">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Certificate (PEM) *</label>
            <textarea
              value={certPEM}
              onChange={(e) => setCertPEM(e.target.value)}
              rows={6}
              placeholder="-----BEGIN CERTIFICATE-----&#10;..."
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono outline-none focus:ring-2 focus:ring-blue-500 resize-y"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Private Key (PEM) *</label>
            <textarea
              value={keyPEM}
              onChange={(e) => setKeyPEM(e.target.value)}
              rows={6}
              placeholder="-----BEGIN PRIVATE KEY-----&#10;..."
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm font-mono outline-none focus:ring-2 focus:ring-blue-500 resize-y"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t">
            <button type="button" onClick={() => navigate('/ssl')} className="px-4 py-2 text-sm text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50">
              {t('common.cancel')}
            </button>
            <button type="submit" disabled={loading} className="px-6 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
              {loading ? 'Uploading...' : 'Upload Certificate'}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
