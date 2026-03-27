import { createBrowserRouter } from 'react-router-dom';
import AppLayout from './components/layout/AppLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import MonitoringPage from './pages/monitoring/MonitoringPage';
import WebsiteList from './pages/websites/WebsiteList';
import WebsiteCreate from './pages/websites/WebsiteCreate';
import WebsiteDetail from './pages/websites/WebsiteDetail';
import SSLList from './pages/ssl/SSLList';
import SSLCreate from './pages/ssl/SSLCreate';
import DatabaseList from './pages/databases/DatabaseList';
import ZoneList from './pages/dns/ZoneList';
import ZoneDetail from './pages/dns/ZoneDetail';
import FirewallRules from './pages/firewall/FirewallRules';
import MailDomains from './pages/mail/MailDomains';
import DockerManagement from './pages/containers/DockerManagement';
import FileManager from './pages/files/FileManager';
import TerminalPage from './pages/terminal/TerminalPage';
import CronJobs from './pages/cronjobs/CronJobs';
import Backups from './pages/backups/Backups';
import AppStore from './pages/appstore/AppStore';
import PHPRuntimes from './pages/settings/PHPRuntimes';
import FTPUsers from './pages/settings/FTPUsers';
import GitDeploys from './pages/gitdeploy/GitDeploys';
import ProcessManager from './pages/system/ProcessManager';
import Toolbox from './pages/system/Toolbox';
import Security from './pages/security/Security';
import BillingDashboard from './pages/settings/BillingDashboard';
import APIKeys from './pages/settings/APIKeys';
import Nodes from './pages/settings/Nodes';
import General from './pages/settings/General';
import Users from './pages/settings/Users';
import License from './pages/settings/License';

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/',
    element: <AppLayout />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: 'monitoring', element: <MonitoringPage /> },
      { path: 'websites', element: <WebsiteList /> },
      { path: 'websites/new', element: <WebsiteCreate /> },
      { path: 'websites/:id', element: <WebsiteDetail /> },
      { path: 'ssl', element: <SSLList /> },
      { path: 'ssl/new', element: <SSLCreate /> },
      { path: 'dns', element: <ZoneList /> },
      { path: 'dns/:id', element: <ZoneDetail /> },
      { path: 'mail', element: <MailDomains /> },
      { path: 'databases', element: <DatabaseList /> },
      { path: 'firewall', element: <FirewallRules /> },
      { path: 'files', element: <FileManager /> },
      { path: 'containers', element: <DockerManagement /> },
      { path: 'apps', element: <AppStore /> },
      { path: 'cronjobs', element: <CronJobs /> },
      { path: 'backups', element: <Backups /> },
      { path: 'terminal', element: <TerminalPage /> },
      { path: 'processes', element: <ProcessManager /> },
      { path: 'toolbox', element: <Toolbox /> },
      { path: 'php', element: <PHPRuntimes /> },
      { path: 'ftp', element: <FTPUsers /> },
      { path: 'git-deploy', element: <GitDeploys /> },
      { path: 'security', element: <Security /> },
      { path: 'billing', element: <BillingDashboard /> },
      { path: 'api-keys', element: <APIKeys /> },
      { path: 'nodes', element: <Nodes /> },
      { path: 'settings', element: <General /> },
      { path: 'settings/users', element: <Users /> },
      { path: 'license', element: <License /> },
    ],
  },
]);
