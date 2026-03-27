package nginx

import (
	"fmt"
	"strings"
)

// SitePaths returns all paths for a website based on its alias.
// These map to the folder structure under /www/sites/{alias}/
type SitePaths struct {
	Root      string // /www/sites/{alias}
	Index     string // /www/sites/{alias}/index
	Log       string // /www/sites/{alias}/log
	SSL       string // /www/sites/{alias}/ssl
	Proxy     string // /www/sites/{alias}/proxy
	Rewrite   string // /www/sites/{alias}/rewrite
	ConfFile  string // /www/conf.d/{alias}.conf
	AccessLog string // /www/sites/{alias}/log/access.log
	ErrorLog  string // /www/sites/{alias}/log/error.log
}

func GetSitePaths(alias string) SitePaths {
	root := fmt.Sprintf("/www/sites/%s", alias)
	return SitePaths{
		Root:      root,
		Index:     root + "/index",
		Log:       root + "/log",
		SSL:       root + "/ssl",
		Proxy:     root + "/proxy",
		Rewrite:   root + "/rewrite",
		ConfFile:  fmt.Sprintf("/www/conf.d/%s.conf", alias),
		AccessLog: root + "/log/access.log",
		ErrorLog:  root + "/log/error.log",
	}
}

// AllDirs returns all directories that need to be created for a site.
func (p SitePaths) AllDirs() []string {
	return []string{p.Root, p.Index, p.Log, p.SSL, p.Proxy, p.Rewrite}
}

// Domain represents a listen domain with port.
type Domain struct {
	Name string
	Port int
	SSL  bool
}

// Config holds the parameters for nginx config generation.
type Config struct {
	Alias         string
	Domains       []Domain
	Type          string // static, proxy, php, node
	DocumentRoot  string
	ProxyTarget   string
	PHPVersion    string
	PHPPort       int // PHP-FPM port (9000, 9001, etc.)
	IPV6          bool
	AccessLog     bool
	ErrorLog      bool
	DefaultServer bool
	SSLEnabled    bool
	SSLCertPath   string
	SSLKeyPath    string
	CustomConfig  string
}

// GenerateServerBlock generates an nginx server block configuration.
func GenerateServerBlock(cfg Config) string {
	paths := GetSitePaths(cfg.Alias)
	var b strings.Builder

	b.WriteString("server {\n")

	// Listen directives
	ports := uniquePorts(cfg.Domains)
	for _, port := range ports {
		defaultStr := ""
		if cfg.DefaultServer {
			defaultStr = " default_server"
		}
		b.WriteString(fmt.Sprintf("    listen %d%s;\n", port, defaultStr))
		if cfg.IPV6 {
			b.WriteString(fmt.Sprintf("    listen [::]:%d%s;\n", port, defaultStr))
		}
	}

	// SSL listen
	if cfg.SSLEnabled {
		b.WriteString("    listen 443 ssl http2;\n")
		if cfg.IPV6 {
			b.WriteString("    listen [::]:443 ssl http2;\n")
		}
	}

	b.WriteString("\n")

	// Server name
	serverNames := uniqueServerNames(cfg.Domains)
	b.WriteString(fmt.Sprintf("    server_name %s;\n", strings.Join(serverNames, " ")))

	// Root
	root := paths.Index
	if cfg.DocumentRoot != "" {
		root = cfg.DocumentRoot
	}
	b.WriteString(fmt.Sprintf("    root %s;\n", root))
	b.WriteString("    index index.html index.htm index.php;\n")
	b.WriteString("\n")

	// Logging
	if cfg.AccessLog {
		b.WriteString(fmt.Sprintf("    access_log %s;\n", paths.AccessLog))
	} else {
		b.WriteString("    access_log off;\n")
	}
	if cfg.ErrorLog {
		b.WriteString(fmt.Sprintf("    error_log %s;\n", paths.ErrorLog))
	} else {
		b.WriteString("    error_log /dev/null;\n")
	}
	b.WriteString("\n")

	// SSL
	if cfg.SSLEnabled {
		certPath := cfg.SSLCertPath
		keyPath := cfg.SSLKeyPath
		if certPath == "" {
			certPath = paths.SSL + "/fullchain.pem"
		}
		if keyPath == "" {
			keyPath = paths.SSL + "/privkey.pem"
		}
		b.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", certPath))
		b.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", keyPath))
		b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		b.WriteString("    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;\n")
		b.WriteString("    ssl_prefer_server_ciphers on;\n")
		b.WriteString("    ssl_session_cache shared:SSL:10m;\n")
		b.WriteString("    ssl_session_timeout 10m;\n")
		b.WriteString("\n")
	}

	// Type-specific location blocks
	switch cfg.Type {
	case "static":
		b.WriteString("    location / {\n")
		b.WriteString("        try_files $uri $uri/ =404;\n")
		b.WriteString("    }\n")
		b.WriteString("\n")
		b.WriteString("    # Cache static assets\n")
		b.WriteString("    location ~* \\.(jpg|jpeg|png|gif|ico|css|js|woff2?|ttf|svg)$ {\n")
		b.WriteString("        expires 30d;\n")
		b.WriteString("        add_header Cache-Control \"public, immutable\";\n")
		b.WriteString("    }\n")

	case "proxy":
		target := cfg.ProxyTarget
		if target == "" {
			target = "http://127.0.0.1:3000"
		}
		b.WriteString("    location / {\n")
		b.WriteString(fmt.Sprintf("        proxy_pass %s;\n", target))
		b.WriteString("        proxy_set_header Host $host;\n")
		b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection \"upgrade\";\n")
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_read_timeout 600s;\n")
		b.WriteString("        proxy_send_timeout 600s;\n")
		b.WriteString("    }\n")

	case "php":
		b.WriteString("    location / {\n")
		b.WriteString("        try_files $uri $uri/ /index.php$is_args$args;\n")
		b.WriteString("    }\n")
		b.WriteString("\n")
		phpPort := cfg.PHPPort
		if phpPort == 0 {
			phpPort = 9000
		}
		phpSocket := fmt.Sprintf("127.0.0.1:%d", phpPort)
		b.WriteString("    location ~ \\.php$ {\n")
		b.WriteString("        try_files $uri =404;\n")
		b.WriteString("        fastcgi_split_path_info ^(.+\\.php)(/.+)$;\n")
		b.WriteString(fmt.Sprintf("        fastcgi_pass %s;\n", phpSocket))
		b.WriteString("        fastcgi_index index.php;\n")
		b.WriteString("        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
		b.WriteString("        include fastcgi_params;\n")
		b.WriteString("\n")
		// open_basedir: restrict PHP to only this website's directory
		b.WriteString(fmt.Sprintf("        fastcgi_param PHP_VALUE \"open_basedir=%s/:/tmp/:/proc/\";\n", root))
		b.WriteString("    }\n")
		b.WriteString("\n")
		b.WriteString("    # Deny access to .ht files\n")
		b.WriteString("    location ~ /\\.ht {\n")
		b.WriteString("        deny all;\n")
		b.WriteString("    }\n")

	case "node":
		target := cfg.ProxyTarget
		if target == "" {
			target = "http://127.0.0.1:3000"
		}
		b.WriteString("    location / {\n")
		b.WriteString(fmt.Sprintf("        proxy_pass %s;\n", target))
		b.WriteString("        proxy_set_header Host $host;\n")
		b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection \"upgrade\";\n")
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("    }\n")
		b.WriteString("\n")
		b.WriteString("    # Static assets served directly\n")
		b.WriteString("    location /public/ {\n")
		b.WriteString(fmt.Sprintf("        alias %s/public/;\n", root))
		b.WriteString("        expires 30d;\n")
		b.WriteString("    }\n")
	}

	// Custom config
	if cfg.CustomConfig != "" {
		b.WriteString("\n    # Custom configuration\n")
		for _, line := range strings.Split(cfg.CustomConfig, "\n") {
			if strings.TrimSpace(line) != "" {
				b.WriteString("    " + line + "\n")
			}
		}
	}

	// Include proxy configs
	b.WriteString(fmt.Sprintf("\n    include %s/*.conf;\n", GetSitePaths(cfg.Alias).Proxy))

	// Security headers
	b.WriteString("\n    # Security headers\n")
	b.WriteString("    add_header X-Content-Type-Options \"nosniff\" always;\n")
	b.WriteString("    add_header X-Frame-Options \"SAMEORIGIN\" always;\n")
	b.WriteString("    add_header X-XSS-Protection \"1; mode=block\" always;\n")

	b.WriteString("}\n")

	return b.String()
}

// GenerateDefaultIndex generates a default index.html for new websites.
func GenerateDefaultIndex(domain string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to %s</title>
    <style>
        body { font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #f8fafc; color: #334155; }
        .container { text-align: center; }
        h1 { font-size: 2rem; margin-bottom: 0.5rem; }
        p { color: #64748b; }
        .badge { display: inline-block; background: #3b82f6; color: white; padding: 0.25rem 0.75rem; border-radius: 1rem; font-size: 0.875rem; margin-top: 1rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to %s</h1>
        <p>Your website is ready. Upload your files to get started.</p>
        <span class="badge">Powered by Panelox</span>
    </div>
</body>
</html>`, domain, domain)
}

func uniquePorts(domains []Domain) []int {
	seen := map[int]bool{}
	var ports []int
	for _, d := range domains {
		if !seen[d.Port] {
			seen[d.Port] = true
			ports = append(ports, d.Port)
		}
	}
	if len(ports) == 0 {
		return []int{80}
	}
	return ports
}

func uniqueServerNames(domains []Domain) []string {
	seen := map[string]bool{}
	var names []string
	for _, d := range domains {
		if !seen[d.Name] {
			seen[d.Name] = true
			names = append(names, d.Name)
		}
	}
	return names
}
