import { useEffect, useState, useRef, useCallback } from 'react';
import { filesApi, type FileEntry, type SearchResult, type RecycleBinEntry } from '../../api/files';
import Editor from '@monaco-editor/react';

type Tab = 'files' | 'search' | 'recycle-bin';
type ViewMode = 'list' | 'grid';

const fileIcons: Record<string, { icon: string; color: string }> = {
  folder: { icon: '📁', color: 'text-yellow-500' },
  php: { icon: '🐘', color: 'text-indigo-500' },
  js: { icon: '⚡', color: 'text-yellow-500' },
  ts: { icon: '🔷', color: 'text-blue-500' },
  jsx: { icon: '⚛️', color: 'text-cyan-500' },
  tsx: { icon: '⚛️', color: 'text-blue-500' },
  html: { icon: '🌐', color: 'text-orange-500' },
  css: { icon: '🎨', color: 'text-blue-400' },
  json: { icon: '📋', color: 'text-green-600' },
  yml: { icon: '⚙️', color: 'text-red-400' },
  yaml: { icon: '⚙️', color: 'text-red-400' },
  md: { icon: '📝', color: 'text-gray-600' },
  py: { icon: '🐍', color: 'text-green-500' },
  go: { icon: '🔵', color: 'text-cyan-600' },
  rs: { icon: '🦀', color: 'text-orange-600' },
  sh: { icon: '💻', color: 'text-green-700' },
  sql: { icon: '🗃️', color: 'text-blue-600' },
  env: { icon: '🔐', color: 'text-yellow-600' },
  conf: { icon: '⚙️', color: 'text-gray-500' },
  nginx: { icon: '🟢', color: 'text-green-500' },
  log: { icon: '📄', color: 'text-gray-400' },
  xml: { icon: '📰', color: 'text-orange-400' },
  svg: { icon: '🖼️', color: 'text-purple-500' },
  jpg: { icon: '🖼️', color: 'text-pink-500' },
  jpeg: { icon: '🖼️', color: 'text-pink-500' },
  png: { icon: '🖼️', color: 'text-pink-500' },
  gif: { icon: '🖼️', color: 'text-pink-500' },
  webp: { icon: '🖼️', color: 'text-pink-500' },
  zip: { icon: '📦', color: 'text-amber-600' },
  'tar.gz': { icon: '📦', color: 'text-amber-600' },
  gz: { icon: '📦', color: 'text-amber-600' },
  txt: { icon: '📄', color: 'text-gray-500' },
};

function getFileIcon(entry: FileEntry) {
  if (entry.is_dir) return fileIcons.folder;
  const ext = entry.name.split('.').pop()?.toLowerCase() || '';
  return fileIcons[ext] || { icon: '📄', color: 'text-gray-400' };
}

function formatSize(bytes: number) {
  if (bytes === 0) return '—';
  const units = ['B', 'KB', 'MB', 'GB'];
  let i = 0, size = bytes;
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++; }
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

function timeAgo(date: string) {
  const diff = Date.now() - new Date(date).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(date).toLocaleDateString();
}

const getLanguage = (path: string) => {
  const ext = path.split('.').pop()?.toLowerCase() || '';
  const map: Record<string, string> = {
    js: 'javascript', ts: 'typescript', jsx: 'javascript', tsx: 'typescript',
    py: 'python', rb: 'ruby', go: 'go', rs: 'rust', php: 'php',
    html: 'html', css: 'css', scss: 'scss', json: 'json', yaml: 'yaml', yml: 'yaml',
    xml: 'xml', sql: 'sql', sh: 'shell', bash: 'shell', md: 'markdown',
    conf: 'ini', ini: 'ini', env: 'ini', nginx: 'nginx',
  };
  return map[ext] || 'plaintext';
};

const isArchive = (name: string) => /\.(zip|tar\.gz|tgz|tar\.bz2|tar|gz)$/i.test(name);
const isImage = (name: string) => /\.(jpg|jpeg|png|gif|svg|webp|ico|bmp)$/i.test(name);

export default function FileManager() {
  const [tab, setTab] = useState<Tab>('files');
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [currentPath, setCurrentPath] = useState('/www');
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  // Editor
  const [editing, setEditing] = useState<{ path: string; content: string; modified: boolean } | null>(null);
  const [saving, setSaving] = useState(false);

  // New folder / new file
  const [showNew, setShowNew] = useState<'folder' | 'file' | null>(null);
  const [newName, setNewName] = useState('');

  // Rename
  const [renaming, setRenaming] = useState<string | null>(null);
  const [renameName, setRenameName] = useState('');

  // Chmod
  const [chmodTarget, setChmodTarget] = useState<FileEntry | null>(null);
  const [chmodValue, setChmodValue] = useState('');

  // Multi-select
  const [selected, setSelected] = useState<Set<string>>(new Set());

  // Context menu
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; entry: FileEntry } | null>(null);

  // Search
  const [searchQuery, setSearchQuery] = useState('');
  const [searchExt, setSearchExt] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);

  // Recycle bin
  const [recycleBin, setRecycleBin] = useState<RecycleBinEntry[]>([]);

  // Upload
  const [dragOver, setDragOver] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Clipboard (cut/copy)
  const [clipboard, setClipboard] = useState<{ paths: string[]; action: 'copy' | 'cut' } | null>(null);

  const flash = (msg: string) => { setMessage(msg); setTimeout(() => setMessage(''), 3000); };

  const fetchEntries = useCallback(async (path: string) => {
    setLoading(true);
    setError('');
    try {
      const res = await filesApi.list(path);
      const data = res.data.data || [];
      data.sort((a, b) => {
        if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
        return a.name.localeCompare(b.name);
      });
      setEntries(data);
      setCurrentPath(path);
      setSelected(new Set());
    } catch (err: any) {
      setError(err.response?.data?.error?.message || 'Failed to list directory');
      setEntries([]);
    } finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchEntries(currentPath); }, []);
  useEffect(() => { const h = () => setContextMenu(null); document.addEventListener('click', h); return () => document.removeEventListener('click', h); }, []);

  const navigate = (path: string) => { setEditing(null); fetchEntries(path); };
  const goUp = () => navigate(currentPath.split('/').slice(0, -1).join('/') || '/');

  const openFile = async (entry: FileEntry) => {
    if (entry.is_dir) { navigate(entry.path); return; }
    if (isImage(entry.name)) { window.open(`/api/v1/files/download?path=${encodeURIComponent(entry.path)}`, '_blank'); return; }
    try {
      const res = await filesApi.read(entry.path);
      setEditing({ path: entry.path, content: res.data.data.content, modified: false });
    } catch (err: any) { setError(err.response?.data?.error?.message || 'Cannot open file (binary?)'); }
  };

  const saveFile = async () => {
    if (!editing) return;
    setSaving(true);
    try { await filesApi.write(editing.path, editing.content); setEditing(e => e ? { ...e, modified: false } : null); flash('File saved'); }
    catch { flash('Save failed'); }
    setSaving(false);
  };

  // Keyboard shortcuts
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === 's' && editing) { e.preventDefault(); saveFile(); }
    };
    document.addEventListener('keydown', h);
    return () => document.removeEventListener('keydown', h);
  }, [editing]);

  // Upload
  const uploadFiles = async (files: FileList) => {
    setUploading(true);
    const total = files.length;
    let done = 0;
    for (const file of Array.from(files)) {
      await filesApi.upload(currentPath, file).catch(() => {});
      done++;
      setUploadProgress(Math.round((done / total) * 100));
    }
    setUploading(false);
    setUploadProgress(0);
    fetchEntries(currentPath);
    flash(`${total} file${total > 1 ? 's' : ''} uploaded`);
  };

  const handleDrop = (e: React.DragEvent) => { e.preventDefault(); setDragOver(false); if (e.dataTransfer.files.length > 0) uploadFiles(e.dataTransfer.files); };

  // Selection
  const toggleSelect = (path: string, e?: React.MouseEvent) => {
    if (e?.shiftKey && selected.size > 0) {
      // Range select
      const paths = entries.map(x => x.path);
      const lastSelected = [...selected].pop()!;
      const start = paths.indexOf(lastSelected);
      const end = paths.indexOf(path);
      const range = paths.slice(Math.min(start, end), Math.max(start, end) + 1);
      setSelected(new Set([...selected, ...range]));
    } else {
      setSelected(prev => {
        const next = new Set(prev);
        if (next.has(path)) next.delete(path); else next.add(path);
        return next;
      });
    }
  };

  // Context menu actions
  const ctxRename = (entry: FileEntry) => { setRenaming(entry.path); setRenameName(entry.name); };
  const ctxChmod = (entry: FileEntry) => { setChmodTarget(entry); setChmodValue(entry.mode?.slice(1) || '644'); };
  const ctxCopy = (entry: FileEntry) => { setClipboard({ paths: [entry.path], action: 'copy' }); flash('Copied to clipboard'); };
  const ctxCut = (entry: FileEntry) => { setClipboard({ paths: [entry.path], action: 'cut' }); flash('Cut to clipboard'); };
  const ctxDelete = async (entry: FileEntry) => { if (confirm(`Delete "${entry.name}"?`)) { await filesApi.delete(entry.path).catch(() => {}); fetchEntries(currentPath); } };
  const ctxDeleteToBin = async (entry: FileEntry) => { await filesApi.batchDelete([entry.path], true).catch(() => {}); fetchEntries(currentPath); flash('Moved to recycle bin'); };

  const handlePaste = async () => {
    if (!clipboard) return;
    if (clipboard.action === 'cut') {
      await filesApi.batchMove(clipboard.paths, currentPath).catch(() => {});
    } else {
      for (const p of clipboard.paths) {
        const name = p.split('/').pop()!;
        // Use copy via compress/decompress or just shell
        await filesApi.rename(p, currentPath + '/' + name).catch(() => {});
      }
    }
    setClipboard(null);
    fetchEntries(currentPath);
  };

  const handleRename = async () => {
    if (!renaming || !renameName) return;
    const dir = renaming.split('/').slice(0, -1).join('/');
    await filesApi.rename(renaming, dir + '/' + renameName).catch(() => {});
    setRenaming(null);
    fetchEntries(currentPath);
  };

  const handleChmod = async () => {
    if (!chmodTarget || !chmodValue) return;
    await filesApi.chmod(chmodTarget.path, chmodValue).catch(() => {});
    setChmodTarget(null);
    fetchEntries(currentPath);
    flash('Permissions updated');
  };

  const handleNew = async () => {
    if (!newName) return;
    const path = `${currentPath}/${newName}`;
    if (showNew === 'folder') await filesApi.mkdir(path).catch(() => {});
    else await filesApi.write(path, '').catch(() => {});
    setNewName(''); setShowNew(null); fetchEntries(currentPath);
  };

  // Batch operations
  const batchDelete = async (toBin: boolean) => {
    if (selected.size === 0) return;
    if (!confirm(`${toBin ? 'Move' : 'Delete'} ${selected.size} items?`)) return;
    await filesApi.batchDelete(Array.from(selected), toBin).catch(() => {});
    fetchEntries(currentPath);
  };

  const batchCompress = async () => {
    const name = prompt('Archive name:', 'archive.tar.gz');
    if (!name || selected.size === 0) return;
    await filesApi.compress(Array.from(selected), currentPath + '/' + name).catch(() => {});
    fetchEntries(currentPath);
  };

  const batchChmod = async () => {
    const mode = prompt('Set permissions:', '644');
    if (!mode || selected.size === 0) return;
    await filesApi.batchChmod(Array.from(selected), mode).catch(() => {});
    fetchEntries(currentPath);
    flash('Permissions updated');
  };

  // Search
  const handleSearch = async () => {
    if (!searchQuery) return;
    setSearching(true);
    try { const res = await filesApi.search(currentPath, searchQuery, searchExt); setSearchResults(res.data.data || []); }
    catch { setSearchResults([]); }
    setSearching(false);
  };

  const fetchRecycleBin = () => { filesApi.recycleBin().then(r => setRecycleBin(r.data.data || [])).catch(() => {}); };

  const pathParts = currentPath.split('/').filter(Boolean);
  const totalSize = entries.reduce((sum, e) => sum + (e.is_dir ? 0 : e.size), 0);
  const dirCount = entries.filter(e => e.is_dir).length;
  const fileCount = entries.filter(e => !e.is_dir).length;

  return (
    <div className="h-[calc(100vh-100px)] flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <h1 className="text-2xl font-bold text-gray-800">File Manager</h1>
        <div className="flex items-center gap-1 bg-gray-100 p-0.5 rounded-lg">
          {([['files', 'Files'], ['search', 'Search'], ['recycle-bin', 'Bin']] as [Tab, string][]).map(([k, l]) => (
            <button key={k} onClick={() => { setTab(k); if (k === 'recycle-bin') fetchRecycleBin(); }}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-all ${tab === k ? 'bg-white shadow-sm text-gray-800' : 'text-gray-500 hover:text-gray-700'}`}>{l}</button>
          ))}
        </div>
      </div>

      {/* Toast message */}
      {message && (
        <div className="fixed bottom-4 right-4 z-50 bg-gray-800 text-white text-sm px-4 py-2.5 rounded-lg shadow-lg animate-fade-in flex items-center gap-2">
          <svg className="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
          {message}
        </div>
      )}

      {tab === 'files' && (
        <div className="flex-1 flex flex-col min-h-0"
          onDragOver={e => { e.preventDefault(); setDragOver(true); }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}>

          {/* Toolbar */}
          <div className="flex items-center gap-2 mb-3">
            <button onClick={goUp} className="p-2 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500" title="Go up">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" /></svg>
            </button>

            {/* Breadcrumb */}
            <div className="flex-1 flex items-center gap-0.5 text-sm bg-white border border-gray-200 rounded-lg px-3 py-2 overflow-x-auto">
              <button onClick={() => navigate('/')} className="text-blue-600 hover:text-blue-800 font-medium">/</button>
              {pathParts.map((part, i) => (
                <span key={i} className="flex items-center gap-0.5">
                  <span className="text-gray-300">/</span>
                  <button onClick={() => navigate('/' + pathParts.slice(0, i + 1).join('/'))}
                    className={`hover:text-blue-600 ${i === pathParts.length - 1 ? 'text-gray-800 font-medium' : 'text-blue-600'}`}>{part}</button>
                </span>
              ))}
            </div>

            {/* Action buttons */}
            <div className="flex items-center gap-1">
              <button onClick={() => fetchEntries(currentPath)} className="p-2 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500" title="Refresh">
                <svg className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
              </button>
              <button onClick={() => setShowNew('folder')} className="p-2 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500" title="New folder">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m3-3H9m4.06-7.19l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z" /></svg>
              </button>
              <button onClick={() => setShowNew('file')} className="p-2 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500" title="New file">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.5v15m7.5-7.5h-15" /></svg>
              </button>
              <button onClick={() => fileInputRef.current?.click()} className="p-2 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500" title="Upload">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" /></svg>
              </button>
              <input ref={fileInputRef} type="file" multiple className="hidden" onChange={e => { if (e.target.files) uploadFiles(e.target.files); e.target.value = ''; }} />
              {clipboard && (
                <button onClick={handlePaste} className="p-2 rounded-lg border border-blue-300 bg-blue-50 text-blue-600" title={`Paste (${clipboard.paths.length} items)`}>
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" /></svg>
                </button>
              )}
              <div className="w-px h-6 bg-gray-200 mx-1" />
              <button onClick={() => setViewMode(viewMode === 'list' ? 'grid' : 'list')}
                className="p-2 rounded-lg border border-gray-200 hover:bg-gray-50 text-gray-500" title={viewMode === 'list' ? 'Grid view' : 'List view'}>
                {viewMode === 'list'
                  ? <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" /></svg>
                  : <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z" /></svg>}
              </button>
            </div>
          </div>

          {/* Multi-select bar */}
          {selected.size > 0 && (
            <div className="flex items-center gap-2 mb-3 bg-blue-50 border border-blue-200 rounded-lg px-4 py-2">
              <span className="text-sm font-medium text-blue-700">{selected.size} selected</span>
              <button onClick={() => { if (selected.size === entries.length) setSelected(new Set()); else setSelected(new Set(entries.map(e => e.path))); }}
                className="text-xs text-blue-600 hover:underline">{selected.size === entries.length ? 'Deselect' : 'Select all'}</button>
              <div className="flex-1" />
              <button onClick={() => { setClipboard({ paths: Array.from(selected), action: 'copy' }); flash('Copied'); }} className="px-2.5 py-1 text-xs font-medium rounded bg-white border border-gray-200 hover:bg-gray-50">Copy</button>
              <button onClick={() => { setClipboard({ paths: Array.from(selected), action: 'cut' }); flash('Cut'); }} className="px-2.5 py-1 text-xs font-medium rounded bg-white border border-gray-200 hover:bg-gray-50">Cut</button>
              <button onClick={batchCompress} className="px-2.5 py-1 text-xs font-medium rounded bg-purple-500 text-white hover:bg-purple-600">Compress</button>
              <button onClick={batchChmod} className="px-2.5 py-1 text-xs font-medium rounded bg-gray-500 text-white hover:bg-gray-600">Chmod</button>
              <button onClick={() => batchDelete(true)} className="px-2.5 py-1 text-xs font-medium rounded bg-amber-500 text-white hover:bg-amber-600">To Bin</button>
              <button onClick={() => batchDelete(false)} className="px-2.5 py-1 text-xs font-medium rounded bg-red-500 text-white hover:bg-red-600">Delete</button>
              <button onClick={() => setSelected(new Set())} className="p-1 text-gray-400 hover:text-gray-600">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
          )}

          {/* New folder/file input */}
          {showNew && (
            <div className="flex gap-2 mb-3">
              <div className="flex items-center gap-2 flex-1 bg-white border border-blue-300 rounded-lg px-3 py-2">
                <span className="text-sm">{showNew === 'folder' ? '📁' : '📄'}</span>
                <input type="text" value={newName} onChange={e => setNewName(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') handleNew(); if (e.key === 'Escape') setShowNew(null); }}
                  placeholder={showNew === 'folder' ? 'Folder name' : 'File name (e.g. index.html)'} autoFocus className="flex-1 text-sm outline-none" />
              </div>
              <button onClick={handleNew} className="px-3 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium">Create</button>
              <button onClick={() => setShowNew(null)} className="px-3 py-2 border border-gray-200 rounded-lg text-sm">Cancel</button>
            </div>
          )}

          {/* Upload progress */}
          {uploading && (
            <div className="mb-3 bg-blue-50 border border-blue-200 rounded-lg p-3">
              <div className="flex items-center justify-between text-xs text-blue-700 mb-1">
                <span>Uploading...</span><span>{uploadProgress}%</span>
              </div>
              <div className="w-full bg-blue-100 rounded-full h-1.5">
                <div className="bg-blue-600 h-1.5 rounded-full transition-all" style={{ width: `${uploadProgress}%` }} />
              </div>
            </div>
          )}

          {/* Drag overlay */}
          {dragOver && (
            <div className="mb-3 border-2 border-dashed border-blue-400 rounded-xl p-8 text-center bg-blue-50/50">
              <svg className="w-8 h-8 text-blue-400 mx-auto mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" /></svg>
              <p className="text-sm text-blue-600 font-medium">Drop files to upload to {currentPath}</p>
            </div>
          )}

          {error && <div className="bg-red-50 text-red-600 text-sm px-4 py-3 rounded-lg mb-3">{error}</div>}

          {/* Main content */}
          <div className="flex-1 flex gap-3 min-h-0 overflow-hidden">
            {/* File list/grid */}
            <div className={`bg-white rounded-xl border border-gray-200 overflow-auto ${editing ? 'w-[45%]' : 'w-full'}`}>
              {loading ? (
                <div className="flex items-center justify-center py-20">
                  <div className="w-6 h-6 border-2 border-gray-200 border-t-blue-600 rounded-full animate-spin" />
                </div>
              ) : entries.length === 0 ? (
                <div className="py-16 text-center">
                  <div className="text-3xl mb-2">📂</div>
                  <p className="text-gray-400 text-sm">Empty directory</p>
                  <p className="text-gray-300 text-xs mt-1">Drop files here or use the toolbar to create</p>
                </div>
              ) : viewMode === 'grid' ? (
                <div className="p-3 grid grid-cols-4 xl:grid-cols-6 2xl:grid-cols-8 gap-2">
                  {entries.map(entry => {
                    const fi = getFileIcon(entry);
                    const sel = selected.has(entry.path);
                    return (
                      <div key={entry.path}
                        onClick={e => e.ctrlKey || e.metaKey ? toggleSelect(entry.path) : openFile(entry)}
                        onContextMenu={e => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, entry }); }}
                        className={`p-3 rounded-lg text-center cursor-pointer transition-all hover:bg-gray-50 ${sel ? 'bg-blue-50 ring-1 ring-blue-300' : ''}`}>
                        <div className="text-2xl mb-1">{fi.icon}</div>
                        <p className="text-xs text-gray-700 truncate" title={entry.name}>{entry.name}</p>
                        {!entry.is_dir && <p className="text-[10px] text-gray-400">{formatSize(entry.size)}</p>}
                      </div>
                    );
                  })}
                </div>
              ) : (
                <table className="w-full">
                  <thead className="sticky top-0 bg-gray-50/95 backdrop-blur z-10">
                    <tr className="border-b border-gray-100 text-left text-[10px] text-gray-500 uppercase tracking-wider">
                      <th className="px-2 py-2.5 w-8"><input type="checkbox" checked={selected.size === entries.length && entries.length > 0} onChange={() => { if (selected.size === entries.length) setSelected(new Set()); else setSelected(new Set(entries.map(e => e.path))); }} className="rounded border-gray-300" /></th>
                      <th className="px-3 py-2.5">Name</th>
                      <th className="px-3 py-2.5 w-20 text-right">Size</th>
                      <th className="px-3 py-2.5 w-24">Modified</th>
                      <th className="px-3 py-2.5 w-20">Perms</th>
                    </tr>
                  </thead>
                  <tbody>
                    {entries.map(entry => {
                      const fi = getFileIcon(entry);
                      const sel = selected.has(entry.path);
                      return (
                        <tr key={entry.path}
                          onContextMenu={e => { e.preventDefault(); setContextMenu({ x: e.clientX, y: e.clientY, entry }); }}
                          className={`border-b border-gray-50 hover:bg-gray-50/80 transition-colors ${sel ? 'bg-blue-50/60' : ''}`}>
                          <td className="px-2 py-1.5"><input type="checkbox" checked={sel} onChange={e => toggleSelect(entry.path, e as any)} className="rounded border-gray-300" /></td>
                          <td className="px-3 py-1.5">
                            {renaming === entry.path ? (
                              <div className="flex items-center gap-2">
                                <span className="text-base">{fi.icon}</span>
                                <input type="text" value={renameName} onChange={e => setRenameName(e.target.value)}
                                  onKeyDown={e => { if (e.key === 'Enter') handleRename(); if (e.key === 'Escape') setRenaming(null); }}
                                  autoFocus className="flex-1 py-0.5 px-2 border border-blue-300 rounded text-sm outline-none" />
                                <button onClick={handleRename} className="text-xs text-blue-600">Save</button>
                              </div>
                            ) : (
                              <button onClick={() => openFile(entry)} onDoubleClick={() => openFile(entry)}
                                className="flex items-center gap-2 text-sm hover:text-blue-600 w-full text-left group">
                                <span className="text-base shrink-0">{fi.icon}</span>
                                <span className={`truncate ${entry.is_dir ? 'font-medium' : ''}`}>{entry.name}</span>
                                {entry.is_symlink && <span className="text-[10px] text-gray-300 shrink-0">-&gt; {entry.link_target}</span>}
                              </button>
                            )}
                          </td>
                          <td className="px-3 py-1.5 text-xs text-gray-400 text-right tabular-nums">{entry.is_dir ? '—' : formatSize(entry.size)}</td>
                          <td className="px-3 py-1.5 text-xs text-gray-400">{timeAgo(entry.mod_time)}</td>
                          <td className="px-3 py-1.5 text-[10px] text-gray-400 font-mono">{entry.mode?.slice(1) || ''}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              )}

              {/* Status bar */}
              {!loading && entries.length > 0 && (
                <div className="sticky bottom-0 bg-gray-50/95 backdrop-blur border-t border-gray-100 px-4 py-1.5 flex items-center gap-4 text-[10px] text-gray-400">
                  <span>{dirCount} folder{dirCount !== 1 ? 's' : ''}, {fileCount} file{fileCount !== 1 ? 's' : ''}</span>
                  <span>{formatSize(totalSize)}</span>
                  {selected.size > 0 && <span className="text-blue-500">{selected.size} selected</span>}
                </div>
              )}
            </div>

            {/* Editor panel */}
            {editing && (
              <div className="w-[55%] bg-white rounded-xl border border-gray-200 flex flex-col overflow-hidden">
                <div className="flex items-center justify-between px-3 py-2 border-b border-gray-100 bg-gray-50/50">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="text-base">{getFileIcon({ name: editing.path, path: editing.path, is_dir: false, size: 0, mode: '', mod_time: '', is_symlink: false }).icon}</span>
                    <span className="text-xs font-mono text-gray-600 truncate">{editing.path.split('/').pop()}</span>
                    {editing.modified && <span className="w-2 h-2 rounded-full bg-amber-400" title="Unsaved changes" />}
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="text-[10px] text-gray-400 px-1.5 py-0.5 bg-gray-100 rounded">{getLanguage(editing.path)}</span>
                    <span className="text-[10px] text-gray-400">Ctrl+S</span>
                    <button onClick={saveFile} disabled={saving} className="flex items-center gap-1 px-2.5 py-1 bg-blue-600 text-white rounded text-xs hover:bg-blue-700 disabled:opacity-50">
                      <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" /></svg>
                      {saving ? '...' : 'Save'}
                    </button>
                    <button onClick={() => setEditing(null)} className="p-1 text-gray-400 hover:text-gray-600 rounded hover:bg-gray-100">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
                    </button>
                  </div>
                </div>
                <Editor
                  height="100%"
                  language={getLanguage(editing.path)}
                  value={editing.content}
                  onChange={v => setEditing(prev => prev ? { ...prev, content: v || '', modified: true } : null)}
                  theme="vs-dark"
                  options={{ minimap: { enabled: false }, fontSize: 13, wordWrap: 'on', scrollBeyondLastLine: false, padding: { top: 8 }, smoothScrolling: true }}
                />
              </div>
            )}
          </div>
        </div>
      )}

      {/* Context menu */}
      {contextMenu && (
        <div className="fixed z-50 bg-white border border-gray-200 rounded-lg shadow-xl py-1 min-w-[160px]" style={{ left: contextMenu.x, top: contextMenu.y }}>
          {!contextMenu.entry.is_dir && (
            <button onClick={() => { openFile(contextMenu.entry); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2">
              <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" /></svg>
              Edit
            </button>
          )}
          <button onClick={() => { ctxRename(contextMenu.entry); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2">
            <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
            Rename
          </button>
          <button onClick={() => { ctxCopy(contextMenu.entry); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2">
            <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>
            Copy
          </button>
          <button onClick={() => { ctxCut(contextMenu.entry); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2">
            <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.121 14.121L19 19m-7-7l7-7m-7 7l-2.879 2.879M12 12L9.121 9.121m0 5.758a3 3 0 10-4.243 4.243 3 3 0 004.243-4.243zm0-5.758a3 3 0 10-4.243-4.243 3 3 0 004.243 4.243z" /></svg>
            Cut
          </button>
          <button onClick={() => { ctxChmod(contextMenu.entry); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2">
            <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" /></svg>
            Permissions
          </button>
          {isArchive(contextMenu.entry.name) && (
            <button onClick={() => { filesApi.decompress(contextMenu.entry.path).then(() => fetchEntries(currentPath)); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2">
              <svg className="w-3.5 h-3.5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>
              Extract
            </button>
          )}
          <div className="border-t border-gray-100 my-1" />
          <button onClick={() => { ctxDeleteToBin(contextMenu.entry); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2 text-amber-600">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" /></svg>
            Move to Bin
          </button>
          <button onClick={() => { ctxDelete(contextMenu.entry); setContextMenu(null); }} className="w-full px-3 py-1.5 text-left text-sm hover:bg-gray-50 flex items-center gap-2 text-red-600">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
            Delete
          </button>
        </div>
      )}

      {/* Chmod modal */}
      {chmodTarget && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setChmodTarget(null)}>
          <div className="bg-white rounded-xl p-5 w-[320px] space-y-3" onClick={e => e.stopPropagation()}>
            <h3 className="font-semibold text-sm">Set Permissions</h3>
            <p className="text-xs text-gray-500 font-mono">{chmodTarget.path}</p>
            <input type="text" value={chmodValue} onChange={e => setChmodValue(e.target.value)} placeholder="644" maxLength={4}
              className="w-full py-2 px-3 border rounded-lg text-sm font-mono text-center text-lg tracking-widest" autoFocus />
            <div className="flex gap-2 text-xs">
              {['644', '755', '600', '777', '750'].map(v => (
                <button key={v} onClick={() => setChmodValue(v)} className={`px-2 py-1 rounded border ${chmodValue === v ? 'bg-blue-50 border-blue-300 text-blue-700' : 'border-gray-200 hover:bg-gray-50'}`}>{v}</button>
              ))}
            </div>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setChmodTarget(null)} className="py-2 px-4 text-sm rounded-lg border hover:bg-gray-50">Cancel</button>
              <button onClick={handleChmod} className="py-2 px-4 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700">Apply</button>
            </div>
          </div>
        </div>
      )}

      {/* SEARCH TAB */}
      {tab === 'search' && (
        <div className="space-y-3">
          <div className="bg-white rounded-xl border border-gray-200 p-4 flex gap-3 items-end">
            <div className="flex-1">
              <label className="text-[10px] text-gray-400 uppercase tracking-wider">Search in: {currentPath}</label>
              <input type="text" value={searchQuery} onChange={e => setSearchQuery(e.target.value)} onKeyDown={e => e.key === 'Enter' && handleSearch()}
                placeholder="Search file contents..." className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none focus:ring-2 focus:ring-blue-500/20" />
            </div>
            <div className="w-24">
              <label className="text-[10px] text-gray-400 uppercase tracking-wider">Type</label>
              <input type="text" value={searchExt} onChange={e => setSearchExt(e.target.value)} placeholder="php, js..." className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm outline-none" />
            </div>
            <button onClick={handleSearch} disabled={searching} className="flex items-center gap-1.5 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium disabled:opacity-50">
              {searching ? <div className="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" /> : <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" /></svg>}
              Search
            </button>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            {searchResults.length === 0 ? (
              <div className="py-16 text-center text-gray-400 text-sm">{searching ? 'Searching...' : 'No results'}</div>
            ) : (
              <div className="divide-y divide-gray-50">
                {searchResults.map((r, i) => (
                  <button key={i} onClick={() => { filesApi.read(r.file).then(res => { setEditing({ path: r.file, content: res.data.data.content, modified: false }); setTab('files'); }); }}
                    className="w-full px-4 py-2.5 hover:bg-gray-50 text-left transition-colors">
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-blue-600 font-mono">{r.file}</span>
                      <span className="text-[10px] text-gray-300">:{r.line}</span>
                    </div>
                    <pre className="text-xs text-gray-500 mt-0.5 truncate font-mono">{r.content}</pre>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* RECYCLE BIN TAB */}
      {tab === 'recycle-bin' && (
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-500">{recycleBin.length} items</span>
            <button onClick={() => { if (confirm('Empty recycle bin permanently?')) filesApi.emptyRecycleBin().then(fetchRecycleBin); }}
              className="flex items-center gap-1.5 px-3 py-1.5 border border-red-200 text-red-600 rounded-lg text-xs font-medium hover:bg-red-50">
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" /></svg>
              Empty Bin
            </button>
          </div>
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            {recycleBin.length === 0 ? (
              <div className="py-16 text-center">
                <div className="text-3xl mb-2">🗑️</div>
                <p className="text-gray-400 text-sm">Recycle bin is empty</p>
              </div>
            ) : (
              <table className="w-full">
                <thead><tr className="border-b text-left text-[10px] text-gray-400 uppercase tracking-wider">
                  <th className="px-4 py-2.5">Name</th><th className="px-4 py-2.5">Original Path</th><th className="px-4 py-2.5 w-20">Size</th><th className="px-4 py-2.5 w-28">Deleted</th><th className="px-4 py-2.5 w-20"></th>
                </tr></thead>
                <tbody>
                  {recycleBin.map(item => (
                    <tr key={item.name} className="border-b border-gray-50 hover:bg-gray-50">
                      <td className="px-4 py-2.5 text-sm">{item.name}</td>
                      <td className="px-4 py-2.5 text-xs text-gray-400 font-mono">{item.original_path}</td>
                      <td className="px-4 py-2.5 text-xs text-gray-400">{formatSize(item.size)}</td>
                      <td className="px-4 py-2.5 text-xs text-gray-400">{item.deleted_at ? timeAgo(item.deleted_at) : '—'}</td>
                      <td className="px-4 py-2.5">
                        <button onClick={() => filesApi.restoreFromBin(item.name).then(fetchRecycleBin)}
                          className="text-xs text-blue-600 hover:underline">Restore</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
