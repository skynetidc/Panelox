import api from './client';

export interface ContainerMetrics { id: string; name: string; cpu_percent: string; mem_usage: string; mem_limit: string; mem_percent: string; net_io: string; block_io: string; pids: string; }
export interface NetworkInterface { name: string; rx_bytes: number; tx_bytes: number; rx_packets: number; tx_packets: number; }
export interface DiskIO { device: string; read_bytes: number; write_bytes: number; read_ops: number; write_ops: number; }
export interface WebsiteBandwidth { website_id: number; alias: string; domain: string; bytes: number; requests: number; }

export const monitorAdvApi = {
  containers: () => api.get<{ data: ContainerMetrics[] }>('/monitoring/containers'),
  network: () => api.get<{ data: NetworkInterface[] }>('/monitoring/network'),
  diskIO: () => api.get<{ data: DiskIO[] }>('/monitoring/disk-io'),
  bandwidth: () => api.get<{ data: WebsiteBandwidth[] }>('/monitoring/bandwidth'),
};
