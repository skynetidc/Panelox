package monitoring

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Metrics struct {
	CPUPercent float64   `json:"cpu_percent"`
	MemUsed    int64     `json:"mem_used"`
	MemTotal   int64     `json:"mem_total"`
	DiskUsed   int64     `json:"disk_used"`
	DiskTotal  int64     `json:"disk_total"`
	NetIn      int64     `json:"net_in"`
	NetOut     int64     `json:"net_out"`
	Load1      float64   `json:"load_1"`
	Load5      float64   `json:"load_5"`
	Load15     float64   `json:"load_15"`
	Uptime     string    `json:"uptime"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	Kernel     string    `json:"kernel"`
	CPUCores   int       `json:"cpu_cores"`
	CreatedAt  time.Time `json:"created_at"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/monitoring", func(r chi.Router) {
		r.Get("/current", h.Current)
		r.Get("/history", h.History)
	})
}

func (h *Handler) Current(w http.ResponseWriter, r *http.Request) {
	m := collectMetrics()
	response.JSON(w, http.StatusOK, m)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 || hours > 720 { // max 30 days
		hours = 24
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT cpu_percent, mem_used, mem_total, disk_used, disk_total, net_in, net_out, load_1, load_5, load_15, created_at
		 FROM metrics WHERE created_at > NOW() - ($1 || ' hours')::interval
		 ORDER BY created_at`, fmt.Sprintf("%d", hours))
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	defer rows.Close()

	var metrics []Metrics
	for rows.Next() {
		var m Metrics
		rows.Scan(&m.CPUPercent, &m.MemUsed, &m.MemTotal, &m.DiskUsed, &m.DiskTotal, &m.NetIn, &m.NetOut, &m.Load1, &m.Load5, &m.Load15, &m.CreatedAt)
		metrics = append(metrics, m)
	}
	response.JSON(w, http.StatusOK, metrics)
}

// Collector runs in background, collecting metrics every minute.
type Collector struct {
	pool *pgxpool.Pool
	stop chan struct{}
}

func NewCollector(pool *pgxpool.Pool) *Collector {
	return &Collector{pool: pool, stop: make(chan struct{})}
}

func (c *Collector) Start() {
	go func() {
		// Collect immediately
		c.collect()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.stop:
				return
			}
		}
	}()
	log.Println("Metrics collector started")
}

func (c *Collector) Stop() {
	close(c.stop)
}

func (c *Collector) collect() {
	m := collectMetrics()
	ctx := context.Background()

	c.pool.Exec(ctx,
		`INSERT INTO metrics (cpu_percent, mem_used, mem_total, disk_used, disk_total, net_in, net_out, load_1, load_5, load_15)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		m.CPUPercent, m.MemUsed, m.MemTotal, m.DiskUsed, m.DiskTotal, m.NetIn, m.NetOut, m.Load1, m.Load5, m.Load15)

	// Cleanup old metrics (keep 30 days)
	c.pool.Exec(ctx, "DELETE FROM metrics WHERE created_at < NOW() - INTERVAL '30 days'")
}

func collectMetrics() *Metrics {
	m := &Metrics{
		CPUCores: runtime.NumCPU(),
		OS:       runtime.GOOS,
	}

	m.Hostname, _ = os.Hostname()

	// Memory from /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			val, _ := strconv.ParseInt(parts[1], 10, 64)
			val *= 1024 // KB to bytes
			switch parts[0] {
			case "MemTotal:":
				m.MemTotal = val
			case "MemAvailable:":
				m.MemUsed = m.MemTotal - val
			}
		}
	}

	// Disk from df
	if out, err := os.ReadFile("/proc/mounts"); err == nil {
		_ = out // Use df for disk stats
	}
	// Simple df parsing for root partition
	if out, err := readCommand("df", "-B1", "/"); err == nil {
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				m.DiskTotal, _ = strconv.ParseInt(fields[1], 10, 64)
				m.DiskUsed, _ = strconv.ParseInt(fields[2], 10, 64)
			}
		}
	}

	// Load average from /proc/loadavg
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			m.Load1, _ = strconv.ParseFloat(parts[0], 64)
			m.Load5, _ = strconv.ParseFloat(parts[1], 64)
			m.Load15, _ = strconv.ParseFloat(parts[2], 64)
		}
	}

	// CPU usage approximation from load average
	if m.CPUCores > 0 {
		m.CPUPercent = (m.Load1 / float64(m.CPUCores)) * 100
		if m.CPUPercent > 100 {
			m.CPUPercent = 100
		}
	}

	// Uptime
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			secs, _ := strconv.ParseFloat(parts[0], 64)
			d := time.Duration(secs) * time.Second
			m.Uptime = fmt.Sprintf("%dd %dh %dm", int(d.Hours())/24, int(d.Hours())%24, int(d.Minutes())%60)
		}
	}

	// Kernel
	if out, err := readCommand("uname", "-r"); err == nil {
		m.Kernel = strings.TrimSpace(out)
	}

	// Network from /proc/net/dev
	if data, err := os.ReadFile("/proc/net/dev"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "eth0:") || strings.HasPrefix(line, "ens") || strings.HasPrefix(line, "enp") {
				fields := strings.Fields(line)
				if len(fields) >= 10 {
					rx, _ := strconv.ParseInt(fields[1], 10, 64)
					tx, _ := strconv.ParseInt(fields[9], 10, 64)
					m.NetIn += rx
					m.NetOut += tx
				}
			}
		}
	}

	return m
}

func readCommand(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}
