import { useState } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '../stores/auth';
import { authApi } from '../api/auth';

export default function Login() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isAuthenticated, setAuth } = useAuthStore();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  if (isAuthenticated) return <Navigate to="/" replace />;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault(); setError(''); setLoading(true);
    try { const res = await authApi.login(email, password); const { user, tokens } = res.data.data; setAuth(user, tokens.access_token, tokens.refresh_token); navigate('/'); }
    catch (err: any) { setError(err.response?.data?.error?.message || 'Login failed'); }
    finally { setLoading(false); }
  };

  return (
    <div className="min-h-screen flex">
      <div className="hidden lg:flex flex-1 flex-col justify-between p-12 bg-gray-900">
        <div>
          <div className="flex items-center gap-3 mb-16">
            <span className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-blue-600 text-white text-xl font-bold">P</span>
            <span className="text-white font-bold text-xl">Panelox</span>
          </div>
          <h1 className="text-5xl font-bold text-white leading-tight">Server management,<br /><span className="text-blue-400">simplified.</span></h1>
          <p className="text-gray-400 mt-6 text-lg max-w-lg">Manage websites, databases, Docker, DNS, SSL and more from one beautiful interface.</p>
          <div className="flex flex-wrap gap-2 mt-10">
            {['Multi-Server', 'Git Deploy', 'App Store', 'API Keys', 'Monitoring', '228 APIs'].map(f => (
              <span key={f} className="py-1.5 px-3 inline-flex items-center text-xs font-medium rounded-full border border-gray-700 text-gray-400">{f}</span>
            ))}
          </div>
        </div>
        <div className="flex gap-8 text-gray-600 text-xs uppercase tracking-widest"><span>Open Source</span><span>MIT License</span><span>Go + React</span></div>
      </div>

      <div className="flex-1 flex items-center justify-center p-8 bg-gray-50">
        <div className="w-full max-w-md">
          <div className="lg:hidden flex items-center gap-3 mb-8">
            <span className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-blue-600 text-white font-bold">P</span>
            <span className="font-bold text-xl">Panelox</span>
          </div>
          <div className="bg-white border border-gray-200 rounded-xl shadow-sm p-7">
            <h2 className="text-2xl font-bold text-gray-800">{t('auth.welcomeBack')}</h2>
            <p className="text-sm text-gray-500 mt-1 mb-6">{t('auth.signInToContinue')}</p>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && <div className="bg-red-50 border border-red-200 text-sm text-red-600 rounded-lg p-3">{error}</div>}
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-700">{t('auth.email')}</label>
                <input type="email" value={email} onChange={e => setEmail(e.target.value)} required placeholder="admin@example.com" className="py-2.5 px-4 block w-full border border-gray-200 rounded-lg text-sm focus:border-blue-500 focus:ring-blue-500" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2 text-gray-700">{t('auth.password')}</label>
                <input type="password" value={password} onChange={e => setPassword(e.target.value)} required className="py-2.5 px-4 block w-full border border-gray-200 rounded-lg text-sm focus:border-blue-500 focus:ring-blue-500" />
              </div>
              <button type="submit" disabled={loading} className="w-full py-3 px-4 inline-flex justify-center items-center gap-x-2 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
                {loading ? t('common.loading') : t('auth.login')}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}
