package analytics

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/nginx"
	"github.com/jakub/panelox/internal/pkg/response"
)

type WebsiteStats struct {
	TotalRequests int64            `json:"total_requests"`
	TotalBytes    int64            `json:"total_bandwidth"`
	UniqueIPs     int              `json:"unique_visitors"`
	TopPages      []PageStat       `json:"top_pages"`
	TopReferers   []PageStat       `json:"top_referers"`
	StatusCodes   map[string]int64 `json:"status_codes"`
	HourlyHits    []HourlyStat     `json:"hourly_hits"`
}

type PageStat struct {
	Path  string `json:"path"`
	Count int64  `json:"count"`
}

type HourlyStat struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/analytics/{websiteId}", h.GetStats)
}

// Nginx combined log format regex
var logRegex = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "(\S+) (\S+) \S+" (\d{3}) (\d+) "([^"]*)" "([^"]*)"`)

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	websiteID, _ := strconv.ParseInt(chi.URLParam(r, "websiteId"), 10, 64)

	var alias string
	h.pool.QueryRow(r.Context(), "SELECT alias FROM websites WHERE id = $1", websiteID).Scan(&alias)
	if alias == "" {
		response.NotFound(w, "website not found")
		return
	}

	paths := nginx.GetSitePaths(alias)

	// Parse days parameter
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 30 {
		days = 7
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	stats := parseAccessLog(paths.AccessLog, cutoff)
	response.JSON(w, http.StatusOK, stats)
}

func parseAccessLog(logPath string, since time.Time) *WebsiteStats {
	stats := &WebsiteStats{
		StatusCodes: make(map[string]int64),
	}

	f, err := os.Open(logPath)
	if err != nil {
		return stats
	}
	defer f.Close()

	pageCounts := map[string]int64{}
	refererCounts := map[string]int64{}
	ipSet := map[string]bool{}
	hourCounts := map[string]int64{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := logRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		ip := matches[1]
		timeStr := matches[2]
		path := matches[4]
		statusCode := matches[5]
		bytesStr := matches[6]
		referer := matches[7]

		// Parse time
		t, err := time.Parse("02/Jan/2006:15:04:05 -0700", timeStr)
		if err != nil {
			continue
		}
		if t.Before(since) {
			continue
		}

		stats.TotalRequests++

		bytes, _ := strconv.ParseInt(bytesStr, 10, 64)
		stats.TotalBytes += bytes

		ipSet[ip] = true
		pageCounts[path]++
		stats.StatusCodes[statusCode]++

		hour := t.Format("2006-01-02 15:00")
		hourCounts[hour]++

		if referer != "-" && referer != "" {
			refererCounts[referer]++
		}
	}

	stats.UniqueIPs = len(ipSet)

	// Top pages
	stats.TopPages = topN(pageCounts, 20)
	stats.TopReferers = topN(refererCounts, 10)

	// Hourly hits
	for hour, count := range hourCounts {
		stats.HourlyHits = append(stats.HourlyHits, HourlyStat{Hour: hour, Count: count})
	}
	sort.Slice(stats.HourlyHits, func(i, j int) bool {
		return stats.HourlyHits[i].Hour < stats.HourlyHits[j].Hour
	})

	return stats
}

func topN(counts map[string]int64, n int) []PageStat {
	var items []PageStat
	for path, count := range counts {
		items = append(items, PageStat{Path: path, Count: count})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Count > items[j].Count })
	if len(items) > n {
		items = items[:n]
	}
	return items
}

// FormatBytes formats bytes to human readable
func FormatBytes(b int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(b)
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}

// Compile check
var _ = strings.TrimSpace
