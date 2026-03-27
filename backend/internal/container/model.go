package container

type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Ports   string            `json:"ports"`
	Created string            `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type ContainerStats struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemUsage   string  `json:"mem_usage"`
	MemLimit   string  `json:"mem_limit"`
	NetIO      string  `json:"net_io"`
	BlockIO    string  `json:"block_io"`
}
