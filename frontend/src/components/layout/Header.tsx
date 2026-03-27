import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../stores/auth';
import { authApi } from '../../api/auth';

export default function Header() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { user, refreshToken, logout } = useAuthStore();
  const handleLogout = async () => { if (refreshToken) { try { await authApi.logout(refreshToken); } catch {} } logout(); };
  const toggleLang = () => { const next = i18n.language === 'en' ? 'cs' : 'en'; i18n.changeLanguage(next); localStorage.setItem('panelox_lang', next); };

  return (
    <header className="h-14 flex items-center justify-between px-6 bg-white border-b border-gray-200">
      <div />
      <div className="flex items-center gap-2">
        <button onClick={toggleLang} className="py-1.5 px-3 inline-flex items-center gap-x-1.5 text-xs font-medium rounded-lg border border-gray-200 bg-white text-gray-500 hover:bg-gray-50">
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3" /></svg>
          {i18n.language.toUpperCase()}
        </button>

        {user && (
          <div className="flex items-center gap-2 pl-3 border-l border-gray-200">
            <span className="inline-flex items-center justify-center w-8 h-8 rounded-full bg-blue-600 text-white text-xs font-bold">{user.name.charAt(0).toUpperCase()}</span>
            <div><p className="text-sm font-medium text-gray-800">{user.name}</p><p className="text-xs text-gray-500">{user.role}</p></div>
          </div>
        )}

        <button onClick={() => navigate('/settings')} className="p-2 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100" title={t('nav.settings')}>
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281zM15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
        </button>

        <button onClick={handleLogout} className="p-2 rounded-lg text-gray-400 hover:text-red-500 hover:bg-red-50" title={t('auth.logout')}>
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" /></svg>
        </button>
      </div>
    </header>
  );
}
