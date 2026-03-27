import React from 'react';
import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useUIStore } from '../../stores/ui';

const navItems = [
  { to: '/', icon: 'hs-icon hs-icon-home', label: 'nav.dashboard', group: '' },
  { to: '/monitoring', icon: 'hs-icon hs-icon-chart-bar', label: 'nav.monitoring', group: '' },
  { to: '/websites', icon: 'hs-icon hs-icon-globe-alt', label: 'nav.websites', group: 'Hosting' },
  { to: '/ssl', icon: 'hs-icon hs-icon-shield-check', label: 'nav.ssl', group: 'Hosting' },
  { to: '/dns', icon: 'hs-icon hs-icon-server', label: 'nav.dns', group: 'Hosting' },
  { to: '/databases', icon: 'hs-icon hs-icon-circle-stack', label: 'nav.databases', group: 'Hosting' },
  { to: '/mail', icon: 'hs-icon hs-icon-envelope', label: 'nav.mail', group: 'Hosting' },
  { to: '/ftp', icon: 'hs-icon hs-icon-cloud-arrow-up', label: 'FTP', group: 'Hosting' },
  { to: '/php', icon: 'hs-icon hs-icon-code-bracket', label: 'nav.phpRuntimes', group: 'Hosting' },
  { to: '/containers', icon: 'hs-icon hs-icon-cube', label: 'nav.containers', group: 'DevOps' },
  { to: '/apps', icon: 'hs-icon hs-icon-squares-2x2', label: 'nav.appStore', group: 'DevOps' },
  { to: '/git-deploy', icon: 'hs-icon hs-icon-arrow-path', label: 'nav.gitDeploy', group: 'DevOps' },
  { to: '/files', icon: 'hs-icon hs-icon-folder', label: 'nav.files', group: 'Tools' },
  { to: '/terminal', icon: 'hs-icon hs-icon-command-line', label: 'nav.terminal', group: 'Tools' },
  { to: '/cronjobs', icon: 'hs-icon hs-icon-clock', label: 'nav.cronJobs', group: 'Tools' },
  { to: '/backups', icon: 'hs-icon hs-icon-cloud-arrow-down', label: 'nav.backups', group: 'Tools' },
  { to: '/processes', icon: 'hs-icon', label: 'Processes', group: 'System' },
  { to: '/toolbox', icon: 'hs-icon', label: 'Toolbox', group: 'System' },
  { to: '/security', icon: 'hs-icon hs-icon-lock-closed', label: 'nav.security', group: 'Admin' },
  { to: '/firewall', icon: 'hs-icon hs-icon-fire', label: 'nav.firewall', group: 'Admin' },
  { to: '/billing', icon: 'hs-icon hs-icon-banknotes', label: 'nav.billing', group: 'Admin' },
  { to: '/api-keys', icon: 'hs-icon hs-icon-key', label: 'nav.apiKeys', group: 'Admin' },
  { to: '/nodes', icon: 'hs-icon hs-icon-server-stack', label: 'nav.nodes', group: 'Admin' },
  { to: '/license', icon: 'hs-icon hs-icon-sparkles', label: 'License', group: 'Admin' },
];

export default function Sidebar() {
  const { t } = useTranslation();
  const { sidebarCollapsed, toggleSidebar } = useUIStore();
  let lastGroup = '';

  return (
    <div id="hs-sidebar" className={`flex flex-col h-screen transition-all duration-300 sidebar-dark ${sidebarCollapsed ? 'w-[60px]' : 'w-[240px]'}`} style={{ borderRight: '1px solid rgba(255,255,255,0.06)' }}>
      {/* Logo */}
      <div className="flex items-center h-[56px] px-3" style={{ borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
        {!sidebarCollapsed ? (
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center gap-3">
              <span className="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-blue-600 text-white text-sm font-bold">P</span>
              <div>
                <span className="block text-white text-sm font-semibold">Panelox</span>
                <span className="block text-gray-500 text-[10px] uppercase tracking-widest">Panel</span>
              </div>
            </div>
            <button onClick={toggleSidebar} className="text-gray-500 hover:text-gray-300 p-1 rounded">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7" /></svg>
            </button>
          </div>
        ) : (
          <div className="w-full flex justify-center">
            <span className="inline-flex items-center justify-center w-8 h-8 rounded-lg bg-blue-600 text-white text-sm font-bold">P</span>
          </div>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-3 px-2">
        {navItems.map(item => {
          const showGroup = !sidebarCollapsed && item.group && item.group !== lastGroup;
          if (item.group) lastGroup = item.group;
          return (
            <div key={item.to}>
              {showGroup && <p className="px-3 pt-4 pb-1 text-[10px] font-bold uppercase tracking-[0.15em] text-gray-500">{item.group}</p>}
              <NavLink to={item.to} end={item.to === '/' || item.to === '/settings'}>
                {({ isActive }) => (
                  <span className={`flex items-center gap-x-3 py-2 px-3 text-sm rounded-lg mb-0.5 transition-colors ${sidebarCollapsed ? 'justify-center' : ''} ${isActive ? 'bg-blue-600/20 text-blue-400 font-medium' : 'text-gray-400 hover:text-white hover:bg-white/5'}`}>
                    <svg className="shrink-0 w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.5}>
                      {getIconPath(item.to)}
                    </svg>
                    {!sidebarCollapsed && <span className="truncate">{t(item.label)}</span>}
                  </span>
                )}
              </NavLink>
            </div>
          );
        })}
      </nav>

      {sidebarCollapsed && (
        <div className="p-2 border-t border-white/5">
          <button onClick={toggleSidebar} className="w-full p-2 rounded-lg text-gray-500 hover:text-gray-300 hover:bg-white/5 flex justify-center">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 5l7 7-7 7" /></svg>
          </button>
        </div>
      )}
    </div>
  );
}

function getIconPath(to: string): React.ReactNode {
  const p = (d: string) => <path strokeLinecap="round" strokeLinejoin="round" d={d} />;
  const icons: Record<string, React.ReactNode> = {
    '/': p("M2.25 12l8.954-8.955a1.126 1.126 0 011.591 0L21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75M8.25 21h8.25"),
    '/monitoring': p("M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z"),
    '/websites': p("M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3"),
    '/ssl': p("M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z"),
    '/dns': p("M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z"),
    '/databases': p("M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125"),
    '/mail': p("M21.75 6.75v10.5a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V6.75m19.5 0A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25m19.5 0v.243a2.25 2.25 0 01-1.07 1.916l-7.5 4.615a2.25 2.25 0 01-2.36 0L3.32 8.91a2.25 2.25 0 01-1.07-1.916V6.75"),
    '/ftp': p("M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5"),
    '/php': p("M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"),
    '/processes': p("M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25H12"),
    '/toolbox': p("M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085"),
    '/containers': p("M21 7.5l-9-5.25L3 7.5m18 0l-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9"),
    '/apps': p("M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z"),
    '/git-deploy': p("M7.217 10.907a2.25 2.25 0 100 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186l9.566-5.314m-9.566 7.5l9.566 5.314m0 0a2.25 2.25 0 103.935 2.186 2.25 2.25 0 00-3.935-2.186zm0-12.814a2.25 2.25 0 103.933-2.185 2.25 2.25 0 00-3.933 2.185z"),
    '/files': p("M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z"),
    '/terminal': p("M6.75 7.5l3 2.25-3 2.25m4.5 0h3m-9 8.25h13.5A2.25 2.25 0 0021 18V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v12a2.25 2.25 0 002.25 2.25z"),
    '/cronjobs': p("M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z"),
    '/backups': p("M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z"),
    '/security': p("M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z"),
    '/firewall': p("M15.362 5.214A8.252 8.252 0 0112 21 8.25 8.25 0 016.038 7.048 8.287 8.287 0 009 9.6a8.983 8.983 0 013.361-6.867 8.21 8.21 0 003 2.48z"),
    '/billing': p("M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"),
    '/api-keys': p("M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z"),
    '/nodes': p("M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7"),
    '/license': p("M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z"),
    '/settings': p("M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281zM15 12a3 3 0 11-6 0 3 3 0 016 0z"),
  };
  return icons[to] || p("M12 6.75a.75.75 0 110-1.5.75.75 0 010 1.5zM12 12.75a.75.75 0 110-1.5.75.75 0 010 1.5zM12 18.75a.75.75 0 110-1.5.75.75 0 010 1.5z");
}
