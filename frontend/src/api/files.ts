import api from './client';

export interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mode: string;
  mod_time: string;
  is_symlink: boolean;
  link_target?: string;
}

export interface ReadFileResponse {
  content: string;
  path: string;
  size: number;
  mode: string;
}

export const filesApi = {
  list: (path: string) =>
    api.get<{ data: FileEntry[] }>('/files', { params: { path } }),

  read: (path: string) =>
    api.get<{ data: ReadFileResponse }>('/files/content', { params: { path } }),

  write: (path: string, content: string) =>
    api.put('/files/content', { path, content }),

  upload: (path: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('path', path);
    return api.post('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },

  delete: (path: string) =>
    api.delete('/files', { params: { path } }),

  mkdir: (path: string) =>
    api.post('/files/mkdir', { path }),

  rename: (oldPath: string, newPath: string) =>
    api.post('/files/rename', { old_path: oldPath, new_path: newPath }),

  chmod: (path: string, mode: string) =>
    api.post('/files/chmod', { path, mode }),

  // Advanced
  compress: (paths: string[], destination: string, format?: string) =>
    api.post('/files/compress', { paths, destination, format: format || 'tar.gz' }),

  decompress: (path: string, destination?: string) =>
    api.post('/files/decompress', { path, destination }),

  search: (path: string, query: string, fileExt?: string) =>
    api.post<{ data: SearchResult[] }>('/files/search', { path, query, file_ext: fileExt }),

  batchDelete: (paths: string[], toBin?: boolean) =>
    api.post('/files/batch-delete', { paths, to_bin: toBin }),

  batchMove: (paths: string[], destination: string) =>
    api.post('/files/batch-move', { paths, destination }),

  batchChmod: (paths: string[], mode: string) =>
    api.post('/files/batch-chmod', { paths, mode }),

  recycleBin: () =>
    api.get<{ data: RecycleBinEntry[] }>('/files/recycle-bin'),

  restoreFromBin: (name: string) =>
    api.post('/files/recycle-bin/restore', { name }),

  emptyRecycleBin: () =>
    api.post('/files/recycle-bin/empty'),
};

export interface SearchResult { file: string; line: number; content: string; }
export interface RecycleBinEntry { name: string; original_path: string; size: number; deleted_at: string; }
