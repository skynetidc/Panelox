package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Handler struct {
	pool      *pgxpool.Pool
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(pool *pgxpool.Pool, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{pool: pool, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/2fa", func(r chi.Router) {
		r.Post("/setup", h.Setup)
		r.Post("/enable", h.Enable)
		r.Post("/disable", h.Disable)
		r.Get("/status", h.Status)
	})
}

// GenerateSecret creates a new random TOTP secret.
func GenerateSecret() string {
	bytes := make([]byte, 20)
	rand.Read(bytes)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
}

// GenerateCode produces the current TOTP code for a secret.
func GenerateCode(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}

	counter := uint64(math.Floor(float64(t.Unix()) / 30))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0F
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	otp := code % 1000000

	return fmt.Sprintf("%06d", otp), nil
}

// ValidateCode checks if a TOTP code is valid (allows ±1 time step).
func ValidateCode(secret, code string) bool {
	now := time.Now()
	for _, offset := range []time.Duration{-30 * time.Second, 0, 30 * time.Second} {
		expected, err := GenerateCode(secret, now.Add(offset))
		if err != nil {
			continue
		}
		if hmac.Equal([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

// Setup generates a new TOTP secret and returns the provisioning URI.
func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	secret := GenerateSecret()

	// Store secret (not yet enabled)
	h.pool.Exec(r.Context(),
		"UPDATE users SET totp_secret = $1 WHERE id = $2", secret, c.UserID)

	// Build otpauth URI for QR code
	uri := fmt.Sprintf("otpauth://totp/Panelox:%s?secret=%s&issuer=Panelox&digits=6&period=30", c.Email, secret)

	response.JSON(w, http.StatusOK, map[string]string{
		"secret": secret,
		"uri":    uri,
	})
}

// Enable activates 2FA after verifying a code.
func (h *Handler) Enable(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	var secret string
	h.pool.QueryRow(r.Context(), "SELECT COALESCE(totp_secret, '') FROM users WHERE id = $1", c.UserID).Scan(&secret)
	if secret == "" {
		response.BadRequest(w, "run /2fa/setup first")
		return
	}

	if !ValidateCode(secret, body.Code) {
		response.BadRequest(w, "invalid code")
		return
	}

	h.pool.Exec(r.Context(), "UPDATE users SET totp_enabled = true WHERE id = $1", c.UserID)
	response.JSON(w, http.StatusOK, map[string]string{"message": "2FA enabled"})
}

// Disable turns off 2FA.
func (h *Handler) Disable(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	var secret string
	h.pool.QueryRow(r.Context(), "SELECT COALESCE(totp_secret, '') FROM users WHERE id = $1", c.UserID).Scan(&secret)

	if !ValidateCode(secret, body.Code) {
		response.BadRequest(w, "invalid code")
		return
	}

	h.pool.Exec(r.Context(), "UPDATE users SET totp_enabled = false, totp_secret = NULL WHERE id = $1", c.UserID)
	response.JSON(w, http.StatusOK, map[string]string{"message": "2FA disabled"})
}

// Status returns whether 2FA is enabled for the current user.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var enabled bool
	h.pool.QueryRow(r.Context(), "SELECT totp_enabled FROM users WHERE id = $1", c.UserID).Scan(&enabled)
	response.JSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
}

// CheckLogin verifies 2FA code during login. Returns true if 2FA is not enabled or code is valid.
func CheckLogin(pool *pgxpool.Pool, userID int64, code string) (bool, error) {
	var enabled bool
	var secret string
	err := pool.QueryRow(nil,
		"SELECT totp_enabled, COALESCE(totp_secret, '') FROM users WHERE id = $1", userID,
	).Scan(&enabled, &secret)
	if err != nil {
		return false, err
	}
	if !enabled {
		return true, nil
	}
	return ValidateCode(secret, code), nil
}

// Compile check
var _ = strconv.Itoa
