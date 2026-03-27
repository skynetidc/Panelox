import { Outlet, Navigate, useLocation } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuthStore } from '../../stores/auth';
import { authApi } from '../../api/auth';
import Sidebar from './Sidebar';
import Header from './Header';

export default function AppLayout() {
  const { isAuthenticated, user, setUser } = useAuthStore();
  const location = useLocation();

  useEffect(() => { if (isAuthenticated && !user) { authApi.me().then(res => setUser(res.data.data)).catch(() => {}); } }, [isAuthenticated, user, setUser]);
  useEffect(() => { const w = window as any; if (w.HSStaticMethods) w.HSStaticMethods.autoInit(); }, [location.pathname]);

  if (!isAuthenticated) return <Navigate to="/login" replace />;

  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
