package alert

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jakub/panelox/internal/license"
	"github.com/jakub/panelox/internal/notification"
)

// StartProChecker runs additional PRO alert checks (SSL expiry, app crashes)
func (c *Checker) StartProChecks() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.runProChecks()
			case <-c.stop:
				return
			}
		}
	}()
	log.Println("PRO alert checks started (SSL expiry, app health - every 1h)")
}

func (c *Checker) runProChecks() {
	ctx := context.Background()

	tier := license.GetCurrentTier(ctx, c.pool)
	if tier == license.TierFree {
		return
	}

	c.checkSSLExpiry(ctx)
	c.checkAppHealth(ctx)
}

// checkSSLExpiry alerts if any SSL certificate expires within 7 days
func (c *Checker) checkSSLExpiry(ctx context.Context) {
	rows, err := c.pool.Query(ctx,
		"SELECT id, domains, expires_at FROM ssl_certificates WHERE status = 'issued' AND auto_renew = true AND expires_at < NOW() + INTERVAL '7 days'")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var domains []string
		var expiresAt time.Time
		rows.Scan(&id, &domains, &expiresAt)

		daysLeft := int(time.Until(expiresAt).Hours() / 24)
		msg := fmt.Sprintf("SSL certificate for %s expires in %d days (%s)",
			strings.Join(domains, ", "), daysLeft, expiresAt.Format("2006-01-02"))

		if daysLeft <= 0 {
			msg = fmt.Sprintf("SSL certificate for %s has EXPIRED!", strings.Join(domains, ", "))
		}

		notification.SendToAll(ctx, c.pool, "SSL Certificate Expiry Warning", msg)
	}
}

// checkAppHealth alerts if any running app container is down
func (c *Checker) checkAppHealth(ctx context.Context) {
	rows, err := c.pool.Query(ctx, "SELECT id, name, app_key FROM app_installs WHERE status = 'running'")
	if err != nil {
		return
	}
	defer rows.Close()

	type app struct {
		id   int64
		name string
		key  string
	}
	var apps []app
	for rows.Next() {
		var a app
		rows.Scan(&a.id, &a.name, &a.key)
		apps = append(apps, a)
	}

	for _, a := range apps {
		composeFile := filepath.Join("/opt/panelox/apps", a.name, "docker-compose.yml")
		out, err := exec.Command("docker", "compose", "-f", composeFile, "ps", "--format", "json").Output()
		if err != nil {
			continue
		}

		// Check if any container is not running
		output := string(out)
		if strings.Contains(output, "\"State\":\"exited\"") || strings.Contains(output, "\"State\":\"dead\"") {
			msg := fmt.Sprintf("App '%s' (%s) has crashed containers. Check the App Store for details.", a.name, a.key)
			notification.SendToAll(ctx, c.pool, "App Crash Alert", msg)

			// Update status in DB
			c.pool.Exec(ctx, "UPDATE app_installs SET status = 'error', updated_at = NOW() WHERE id = $1", a.id)
		}
	}
}
