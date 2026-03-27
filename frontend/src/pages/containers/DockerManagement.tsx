import { useEffect, useState } from 'react';
import { dockerApi, type ComposeProject, type DockerNetwork, type DockerVolume, type DockerImage } from '../../api/docker';
import { containerApi, type ContainerInfo } from '../../api/containers';
import {
  Box, Layers, Network, HardDrive, Image, Plus, Trash2, Play, Square, RotateCw,
  FileText, Download, RefreshCw,
} from 'lucide-react';

type Tab = 'containers' | 'compose' | 'networks' | 'volumes' | 'images';

export default function DockerManagement() {
  const [tab, setTab] = useState<Tab>('containers');
  const [containers, setContainers] = useState<ContainerInfo[]>([]);
  const [compose, setCompose] = useState<ComposeProject[]>([]);
  const [networks, setNetworks] = useState<DockerNetwork[]>([]);
  const [volumes, setVolumes] = useState<DockerVolume[]>([]);
  const [images, setImages] = useState<DockerImage[]>([]);
  const [logsId, setLogsId] = useState<string | null>(null);
  const [logsContent, setLogsContent] = useState('');

  // Forms
  const [showCreate, setShowCreate] = useState(false);
  const [newCompose, setNewCompose] = useState({ name: '', content: '', env_file: '' });
  const [newNetwork, setNewNetwork] = useState('');
  const [newVolume, setNewVolume] = useState('');
  const [pullImage, setPullImage] = useState('');

  const fetch = () => {
    switch (tab) {
      case 'containers': containerApi.list().then(r => setContainers(r.data.data || [])).catch(() => {}); break;
      case 'compose': dockerApi.listCompose().then(r => setCompose(r.data.data || [])).catch(() => {}); break;
      case 'networks': dockerApi.listNetworks().then(r => setNetworks(r.data.data || [])).catch(() => {}); break;
      case 'volumes': dockerApi.listVolumes().then(r => setVolumes(r.data.data || [])).catch(() => {}); break;
      case 'images': dockerApi.listImages().then(r => setImages(r.data.data || [])).catch(() => {}); break;
    }
  };
  useEffect(() => { fetch(); }, [tab]);

  const tabs: { key: Tab; label: string; icon: any }[] = [
    { key: 'containers', label: 'Containers', icon: Box },
    { key: 'compose', label: 'Compose', icon: Layers },
    { key: 'networks', label: 'Networks', icon: Network },
    { key: 'volumes', label: 'Volumes', icon: HardDrive },
    { key: 'images', label: 'Images', icon: Image },
  ];

  const stateColors: Record<string, string> = { running: 'bg-green-100 text-green-700', exited: 'bg-gray-100 text-gray-600', created: 'bg-yellow-100 text-yellow-700' };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold text-gray-900">Docker</h1>
        <button onClick={fetch} className="flex items-center gap-1 px-3 py-2 border rounded-lg text-sm hover:bg-gray-50"><RefreshCw size={14} /> Refresh</button>
      </div>

      <div className="border-b border-gray-200 mb-6">
        <nav className="flex gap-4">{tabs.map(t => (
          <button key={t.key} onClick={() => { setTab(t.key); setShowCreate(false); }} className={`flex items-center gap-1 pb-3 text-sm font-medium border-b-2 ${tab === t.key ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500'}`}>
            <t.icon size={14} /> {t.label}
          </button>
        ))}</nav>
      </div>

      {/* CONTAINERS */}
      {tab === 'containers' && (
        <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
          <table className="w-full">
            <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Image</th><th className="px-4 py-3">Status</th><th className="px-4 py-3">Ports</th><th className="px-4 py-3">Actions</th></tr></thead>
            <tbody>
              {containers.map(c => (
                <>
                  <tr key={c.id} className="border-b border-gray-50 hover:bg-gray-50">
                    <td className="px-4 py-3"><p className="font-medium text-sm">{c.name}</p><p className="text-xs text-gray-400 font-mono">{c.id}</p></td>
                    <td className="px-4 py-3 text-sm font-mono text-gray-600">{c.image}</td>
                    <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs font-medium ${stateColors[c.state] || 'bg-gray-100'}`}>{c.state}</span></td>
                    <td className="px-4 py-3 text-xs font-mono text-gray-600 max-w-xs truncate">{c.ports || '—'}</td>
                    <td className="px-4 py-3"><div className="flex gap-1">
                      {c.state !== 'running' && <button onClick={() => { containerApi.start(c.id); fetch(); }} className="p-1 text-gray-400 hover:text-green-600"><Play size={14} /></button>}
                      {c.state === 'running' && <button onClick={() => { containerApi.stop(c.id); fetch(); }} className="p-1 text-gray-400 hover:text-orange-600"><Square size={14} /></button>}
                      <button onClick={() => { if (logsId === c.id) { setLogsId(null); } else { setLogsId(c.id); containerApi.logs(c.id, 200).then(r => setLogsContent(r.data.data.logs)); } }} className="p-1 text-gray-400 hover:text-purple-600"><FileText size={14} /></button>
                      <button onClick={() => { if (confirm(`Remove ${c.name}?`)) { containerApi.remove(c.id); fetch(); } }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
                    </div></td>
                  </tr>
                  {logsId === c.id && <tr><td colSpan={5} className="bg-gray-900 px-4 py-2"><pre className="text-xs text-green-400 font-mono max-h-48 overflow-y-auto whitespace-pre-wrap">{logsContent}</pre></td></tr>}
                </>
              ))}
              {containers.length === 0 && <tr><td colSpan={5} className="px-6 py-8 text-center text-gray-400">No containers</td></tr>}
            </tbody>
          </table>
        </div>
      )}

      {/* COMPOSE */}
      {tab === 'compose' && (
        <div className="space-y-4">
          <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-blue-700"><Plus size={16} /> New Project</button>
          {showCreate && (
            <div className="bg-white rounded-xl shadow-sm border p-4 space-y-3">
              <input type="text" value={newCompose.name} onChange={e => setNewCompose(c => ({ ...c, name: e.target.value }))} placeholder="Project name" className="w-full px-3 py-2 border rounded-lg text-sm outline-none" />
              <textarea value={newCompose.content} onChange={e => setNewCompose(c => ({ ...c, content: e.target.value }))} rows={8} placeholder="docker-compose.yml content..." className="w-full px-3 py-2 border rounded-lg text-sm font-mono outline-none" />
              <div className="flex gap-2">
                <button onClick={() => { dockerApi.createCompose(newCompose).then(() => { setNewCompose({ name: '', content: '', env_file: '' }); setShowCreate(false); fetch(); }); }} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Create</button>
                <button onClick={() => setShowCreate(false)} className="px-4 py-2 border rounded-lg text-sm">Cancel</button>
              </div>
            </div>
          )}
          {compose.map(p => (
            <div key={p.name} className="bg-white rounded-xl shadow-sm border px-6 py-4 flex items-center justify-between">
              <div><p className="font-semibold">{p.name}</p><p className="text-xs text-gray-500">{p.services} services &middot; {p.config_dir}</p></div>
              <div className="flex items-center gap-2">
                <span className={`px-2 py-0.5 rounded text-xs ${p.status === 'running' ? 'bg-green-100 text-green-700' : 'bg-gray-100'}`}>{p.status}</span>
                <button onClick={() => { dockerApi.composeUp(p.name); fetch(); }} className="p-1 text-gray-400 hover:text-green-600"><Play size={14} /></button>
                <button onClick={() => { dockerApi.composeDown(p.name); fetch(); }} className="p-1 text-gray-400 hover:text-orange-600"><Square size={14} /></button>
                <button onClick={() => { dockerApi.composeRestart(p.name); fetch(); }} className="p-1 text-gray-400 hover:text-blue-600"><RotateCw size={14} /></button>
                <button onClick={() => { if (confirm(`Delete project ${p.name}?`)) { dockerApi.composeDelete(p.name); fetch(); } }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button>
              </div>
            </div>
          ))}
          {compose.length === 0 && !showCreate && <div className="bg-white rounded-xl p-12 text-center text-gray-400">No compose projects</div>}
        </div>
      )}

      {/* NETWORKS */}
      {tab === 'networks' && (
        <div className="space-y-4">
          <div className="flex gap-2">
            <input type="text" value={newNetwork} onChange={e => setNewNetwork(e.target.value)} placeholder="Network name" className="px-3 py-2 border rounded-lg text-sm outline-none" />
            <button onClick={() => { if (newNetwork) { dockerApi.createNetwork(newNetwork); setNewNetwork(''); fetch(); } }} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Create</button>
          </div>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Driver</th><th className="px-4 py-3">Scope</th><th className="px-4 py-3">ID</th><th className="px-4 py-3"></th></tr></thead>
              <tbody>{networks.map(n => (
                <tr key={n.id} className="border-b border-gray-50"><td className="px-4 py-3 font-medium">{n.name}</td><td className="px-4 py-3 text-sm">{n.driver}</td><td className="px-4 py-3 text-sm">{n.scope}</td><td className="px-4 py-3 text-xs font-mono text-gray-400">{n.id}</td>
                  <td className="px-4 py-3"><button onClick={() => { dockerApi.deleteNetwork(n.id); fetch(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button></td></tr>
              ))}</tbody>
            </table>
          </div>
        </div>
      )}

      {/* VOLUMES */}
      {tab === 'volumes' && (
        <div className="space-y-4">
          <div className="flex gap-2">
            <input type="text" value={newVolume} onChange={e => setNewVolume(e.target.value)} placeholder="Volume name" className="px-3 py-2 border rounded-lg text-sm outline-none" />
            <button onClick={() => { if (newVolume) { dockerApi.createVolume(newVolume); setNewVolume(''); fetch(); } }} className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm">Create</button>
            <button onClick={() => { if (confirm('Prune all unused volumes?')) { dockerApi.pruneVolumes(); fetch(); } }} className="px-4 py-2 border border-red-300 text-red-600 rounded-lg text-sm">Prune</button>
          </div>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Name</th><th className="px-4 py-3">Driver</th><th className="px-4 py-3">Mountpoint</th><th className="px-4 py-3"></th></tr></thead>
              <tbody>{volumes.map(v => (
                <tr key={v.name} className="border-b border-gray-50"><td className="px-4 py-3 font-medium text-sm">{v.name}</td><td className="px-4 py-3 text-sm">{v.driver}</td><td className="px-4 py-3 text-xs font-mono text-gray-500 max-w-xs truncate">{v.mountpoint}</td>
                  <td className="px-4 py-3"><button onClick={() => { dockerApi.deleteVolume(v.name); fetch(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button></td></tr>
              ))}</tbody>
            </table>
          </div>
        </div>
      )}

      {/* IMAGES */}
      {tab === 'images' && (
        <div className="space-y-4">
          <div className="flex gap-2">
            <input type="text" value={pullImage} onChange={e => setPullImage(e.target.value)} placeholder="nginx:latest" className="flex-1 px-3 py-2 border rounded-lg text-sm outline-none" />
            <button onClick={() => { if (pullImage) { dockerApi.pullImage(pullImage); setPullImage(''); setTimeout(fetch, 3000); } }} className="flex items-center gap-1 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm"><Download size={14} /> Pull</button>
            <button onClick={() => { if (confirm('Prune all unused images?')) { dockerApi.pruneImages(); fetch(); } }} className="px-4 py-2 border border-red-300 text-red-600 rounded-lg text-sm">Prune</button>
          </div>
          <div className="bg-white rounded-xl shadow-sm border overflow-hidden">
            <table className="w-full">
              <thead><tr className="border-b text-left text-xs text-gray-500 uppercase"><th className="px-4 py-3">Repository</th><th className="px-4 py-3">Tag</th><th className="px-4 py-3">Size</th><th className="px-4 py-3">Created</th><th className="px-4 py-3"></th></tr></thead>
              <tbody>{images.map(img => (
                <tr key={img.id} className="border-b border-gray-50"><td className="px-4 py-3 font-medium text-sm">{img.repository}</td><td className="px-4 py-3 text-sm">{img.tag}</td><td className="px-4 py-3 text-sm">{img.size}</td><td className="px-4 py-3 text-xs text-gray-500">{img.created}</td>
                  <td className="px-4 py-3"><button onClick={() => { dockerApi.deleteImage(img.id); fetch(); }} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={14} /></button></td></tr>
              ))}</tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
