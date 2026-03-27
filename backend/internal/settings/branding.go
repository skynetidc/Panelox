package settings

import (
	"encoding/json"
	"net/http"

	"github.com/jakub/panelox/internal/license"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Branding struct {
	PanelName   string `json:"panel_name"`
	LogoURL     string `json:"logo_url"`
	FaviconURL  string `json:"favicon_url"`
	PrimaryColor string `json:"primary_color"`
	LoginTitle  string `json:"login_title"`
	LoginSubtitle string `json:"login_subtitle"`
	FooterText  string `json:"footer_text"`
	HidePoweredBy bool `json:"hide_powered_by"`
}

var defaultBranding = Branding{
	PanelName:    "Panelox",
	PrimaryColor: "#3b82f6",
	LoginTitle:   "Server management,",
	LoginSubtitle: "simplified.",
	FooterText:   "",
	HidePoweredBy: false,
}

func (h *Handler) GetBranding(w http.ResponseWriter, r *http.Request) {
	var val string
	h.pool.QueryRow(r.Context(), "SELECT value FROM settings WHERE key = 'branding'").Scan(&val)

	branding := defaultBranding
	if val != "" {
		json.Unmarshal([]byte(val), &branding)
	}

	// If not PRO, always return defaults (can't use white-label)
	tier := license.GetCurrentTier(r.Context(), h.pool)
	if tier == license.TierFree {
		branding = defaultBranding
	}

	response.JSON(w, http.StatusOK, branding)
}

func (h *Handler) UpdateBranding(w http.ResponseWriter, r *http.Request) {
	tier := license.GetCurrentTier(r.Context(), h.pool)
	if tier == license.TierFree {
		response.JSON(w, http.StatusForbidden, map[string]interface{}{
			"error": map[string]interface{}{"message": "White-label branding requires a Pro license.", "code": "LICENSE_REQUIRED"},
		})
		return
	}

	var branding Branding
	json.NewDecoder(r.Body).Decode(&branding)

	data, _ := json.Marshal(branding)
	h.pool.Exec(r.Context(),
		"INSERT INTO settings (key, value) VALUES ('branding', $1) ON CONFLICT (key) DO UPDATE SET value = $1, updated_at = NOW()",
		string(data))

	response.JSON(w, http.StatusOK, branding)
}

func RegisterBrandingRoutes(r func(pattern string, handler http.HandlerFunc), rGet func(pattern string, handler http.HandlerFunc)) {
	// This is called from RegisterRoutes in handler.go
}
