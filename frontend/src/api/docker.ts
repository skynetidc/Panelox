import api from './client';

// Compose
export interface ComposeProject { name: string; status: string; config_dir: string; services: number; }

// Networks
export interface DockerNetwork { id: string; name: string; driver: string; scope: string; }

// Volumes
export interface DockerVolume { name: string; driver: string; mountpoint: string; size: string; }

// Images
export interface DockerImage { id: string; repository: string; tag: string; size: string; created: string; }

// Redis
export interface RedisInfo { version: string; mode: string; connected_clients: number; used_memory: string; used_memory_peak: string; total_keys: number; uptime_seconds: number; }

export const dockerApi = {
  // Compose
  listCompose: () => api.get<{ data: ComposeProject[] }>('/compose'),
  createCompose: (data: { name: string; content: string; env_file?: string }) => api.post('/compose', data),
  composeUp: (name: string) => api.post(`/compose/${name}/up`),
  composeDown: (name: string) => api.post(`/compose/${name}/down`),
  composeRestart: (name: string) => api.post(`/compose/${name}/restart`),
  composeLogs: (name: string) => api.get<{ data: { logs: string } }>(`/compose/${name}/logs`),
  composeConfig: (name: string) => api.get<{ data: { compose: string; env: string } }>(`/compose/${name}/config`),
  composeUpdateConfig: (name: string, data: { content?: string; env_file?: string }) => api.put(`/compose/${name}/config`, data),
  composeDelete: (name: string) => api.delete(`/compose/${name}`),

  // Networks
  listNetworks: () => api.get<{ data: DockerNetwork[] }>('/docker/networks'),
  createNetwork: (name: string, driver?: string) => api.post('/docker/networks', { name, driver }),
  deleteNetwork: (id: string) => api.delete(`/docker/networks/${id}`),

  // Volumes
  listVolumes: () => api.get<{ data: DockerVolume[] }>('/docker/volumes'),
  createVolume: (name: string, driver?: string) => api.post('/docker/volumes', { name, driver }),
  deleteVolume: (name: string) => api.delete(`/docker/volumes/${name}`),
  pruneVolumes: () => api.post('/docker/volumes/prune'),

  // Images
  listImages: () => api.get<{ data: DockerImage[] }>('/docker/images'),
  pullImage: (image: string) => api.post('/docker/images/pull', { image }),
  deleteImage: (id: string) => api.delete(`/docker/images/${id}`),
  pruneImages: () => api.post('/docker/images/prune'),

  // Redis
  listRedis: () => api.get<{ data: any[] }>('/databases/redis/servers'),
  createRedis: (name: string, password?: string, port?: number) => api.post('/databases/redis/servers', { name, password, port }),
  deleteRedis: (id: number) => api.delete(`/databases/redis/servers/${id}`),
  operateRedis: (id: number, action: string) => api.post(`/databases/redis/servers/${id}/operate`, { action }),
  redisInfo: (id: number) => api.get<{ data: RedisInfo }>(`/databases/redis/servers/${id}/info`),
  redisCLI: (id: number, command: string) => api.post<{ data: { output: string } }>(`/databases/redis/servers/${id}/cli`, { command }),
};
