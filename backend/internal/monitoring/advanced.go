package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/response"
)

// --- Container Metrics ---

type ContainerMetrics struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	CPUPerc  string  `json:"cpu_percent"`
	MemUsage string  `json:"mem_usage"`
	MemLimit string  `json:"mem_limit"`
	MemPerc  string  `json:"mem_percent"`
	NetIO    string  `json:"net_io"`
	BlockIO  string  `json:"block_io"`
	PIDs     string  `json:"pids"`
}

// --- Network Interface ---

type NetworkInterface struct {
	Name    string `json:"name"`
	RxBytes int64  `json:"rx_bytes"`
	TxBytes int64  `json:"tx_bytes"`
	RxPkts  int64  `json:"rx_packets"`
	TxPkts  int64  `json:"tx_packets"`
	RxErrs  int64  `json:"rx_errors"`
	TxErrs  int64  `json:"tx_errors"`
}

// --- Disk I/O ---

type DiskIO struct {
	Device     string `json:"device"`
	ReadBytes  int64  `json:"read_bytes"`
	WriteBytes int64  `json:"write_bytes"`
	ReadOps    int64  `json:"read_ops"`
	WriteOps   int64  `json:"write_ops"`
}

// --- Website Bandwidth ---

type WebsiteBandwidth struct {
	WebsiteID int64  `json:"website_id"`
	Alias     string `json:"alias"`
	Domain    string `json:"domain"`
	Bytes     int64  `json:"bytes"`
	Requests  int64  `json:"requests"`
}

// RegisterAdvancedRoutes adds detailed monitoring endpoints.
func RegisterAdvancedRoutes(r chi.Router, h *Handler) {
	r.Get("/monitoring/containers", h.ContainerStats)
	r.Get("/monitoring/network", h.NetworkStats)
	r.Get("/monitoring/disk-io", h.DiskIOStats)
	r.Get("/monitoring/bandwidth", h.WebsiteBandwidth)
}

func (h *Handler) ContainerStats(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("docker", "stats", "--no-stream", "--format",
		`{"id":"{{.ID}}","name":"{{.Name}}","cpu_percent":"{{.CPUPerc}}","mem_usage":"{{.MemUsage}}","mem_limit":"{{.MemPerc}}","mem_percent":"{{.MemPerc}}","net_io":"{{.NetIO}}","block_io":"{{.BlockIO}}","pids":"{{.PIDs}}"}`).Output()
	if err != nil {
		response.JSON(w, http.StatusOK, []ContainerMetrics{})
		return
	}

	var metrics []ContainerMetrics
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var m ContainerMetrics
		json.Unmarshal([]byte(line), &m)
		metrics = append(metrics, m)
	}
	response.JSON(w, http.StatusOK, metrics)
}

func (h *Handler) NetworkStats(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		response.JSON(w, http.StatusOK, []NetworkInterface{})
		return
	}

	var interfaces []NetworkInterface
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, ":") || strings.HasPrefix(line, "Inter") || strings.HasPrefix(line, "face") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 11 {
			continue
		}

		name := strings.TrimSuffix(parts[0], ":")
		if name == "lo" {
			continue
		}

		iface := NetworkInterface{Name: name}
		iface.RxBytes, _ = strconv.ParseInt(parts[1], 10, 64)
		iface.RxPkts, _ = strconv.ParseInt(parts[2], 10, 64)
		iface.RxErrs, _ = strconv.ParseInt(parts[3], 10, 64)
		iface.TxBytes, _ = strconv.ParseInt(parts[9], 10, 64)
		iface.TxPkts, _ = strconv.ParseInt(parts[10], 10, 64)
		iface.TxErrs, _ = strconv.ParseInt(parts[11], 10, 64)

		interfaces = append(interfaces, iface)
	}
	response.JSON(w, http.StatusOK, interfaces)
}

func (h *Handler) DiskIOStats(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		response.JSON(w, http.StatusOK, []DiskIO{})
		return
	}

	var disks []DiskIO
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		device := fields[2]
		// Only show real disks (sda, vda, nvme), skip partitions with numbers at end
		if !strings.HasPrefix(device, "sd") && !strings.HasPrefix(device, "vd") && !strings.HasPrefix(device, "nvme") {
			continue
		}
		// Skip partitions (sda1, sda2...)
		lastChar := device[len(device)-1]
		if lastChar >= '0' && lastChar <= '9' && !strings.Contains(device, "nvme") {
			continue
		}

		d := DiskIO{Device: device}
		d.ReadOps, _ = strconv.ParseInt(fields[3], 10, 64)
		readSectors, _ := strconv.ParseInt(fields[5], 10, 64)
		d.ReadBytes = readSectors * 512
		d.WriteOps, _ = strconv.ParseInt(fields[7], 10, 64)
		writeSectors, _ := strconv.ParseInt(fields[9], 10, 64)
		d.WriteBytes = writeSectors * 512

		disks = append(disks, d)
	}
	response.JSON(w, http.StatusOK, disks)
}

func (h *Handler) WebsiteBandwidth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.pool.Query(ctx, "SELECT id, alias, domain FROM websites ORDER BY id")
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var results []WebsiteBandwidth
	for rows.Next() {
		var wb WebsiteBandwidth
		rows.Scan(&wb.WebsiteID, &wb.Alias, &wb.Domain)

		// Parse nginx access log for bandwidth
		logPath := fmt.Sprintf("/www/sites/%s/log/access.log", wb.Alias)
		if data, err := os.ReadFile(logPath); err == nil {
			lines := strings.Split(string(data), "\n")
			wb.Requests = int64(len(lines) - 1)

			for _, line := range lines {
				// nginx combined format: ... "status" bytes ...
				parts := strings.Fields(line)
				if len(parts) >= 10 {
					bytes, _ := strconv.ParseInt(parts[9], 10, 64)
					wb.Bytes += bytes
				}
			}
		}

		results = append(results, wb)
	}
	response.JSON(w, http.StatusOK, results)
}
