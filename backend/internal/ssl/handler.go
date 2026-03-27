package ssl

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jakub/panelox/internal/pkg/claims"
	"github.com/jakub/panelox/internal/pkg/response"
)

type Handler struct {
	svc       *Service
	getClaims func(r *http.Request) *claims.Claims
}

func NewHandler(svc *Service, getClaims func(r *http.Request) *claims.Claims) *Handler {
	return &Handler{svc: svc, getClaims: getClaims}
}

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/ssl", func(r chi.Router) {
		r.Get("/", h.ListCertificates)
		r.Post("/", h.Issue)
		r.Post("/upload", h.Upload)
		r.Get("/{id}", h.GetCertificate)
		r.Delete("/{id}", h.DeleteCertificate)
		r.Post("/{id}/renew", h.Renew)
		r.Post("/apply", h.Apply)
		r.Post("/remove", h.RemoveFromWebsite)
	})

	r.Route("/acme-accounts", func(r chi.Router) {
		r.Get("/", h.ListAcmeAccounts)
		r.Post("/", h.CreateAcmeAccount)
		r.Delete("/{id}", h.DeleteAcmeAccount)
	})
}

func (h *Handler) Issue(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req IssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if len(req.Domains) == 0 {
		response.BadRequest(w, "at least one domain is required")
		return
	}
	if req.AcmeAccountID == 0 {
		response.BadRequest(w, "acme_account_id is required")
		return
	}

	cert, err := h.svc.Issue(r.Context(), c.UserID, req)
	if err != nil {
		handleSSLError(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, cert)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	c := h.getClaims(r)
	if c == nil {
		response.Unauthorized(w, "not authenticated")
		return
	}

	var req UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Certificate == "" || req.PrivateKey == "" {
		response.BadRequest(w, "certificate and private_key are required")
		return
	}

	cert, err := h.svc.Upload(r.Context(), c.UserID, req)
	if err != nil {
		handleSSLError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, cert)
}

func (h *Handler) GetCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid certificate id")
		return
	}

	cert, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleSSLError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, cert)
}

func (h *Handler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	certs, total, err := h.svc.List(r.Context(), page, size)
	if err != nil {
		response.InternalError(w, "failed to list certificates")
		return
	}

	response.JSONList(w, certs, page, size, total)
}

func (h *Handler) DeleteCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid certificate id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleSSLError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "certificate deleted"})
}

func (h *Handler) Renew(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid certificate id")
		return
	}

	if err := h.svc.Renew(r.Context(), id); err != nil {
		handleSSLError(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, map[string]string{"message": "renewal started"})
}

func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.CertificateID == 0 || req.WebsiteID == 0 {
		response.BadRequest(w, "certificate_id and website_id are required")
		return
	}

	if err := h.svc.Apply(r.Context(), req); err != nil {
		handleSSLError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "SSL applied to website"})
}

func (h *Handler) RemoveFromWebsite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WebsiteID int64 `json:"website_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := h.svc.RemoveFromWebsite(r.Context(), body.WebsiteID); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "SSL removed from website"})
}

// ACME Accounts

func (h *Handler) CreateAcmeAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAcmeAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Email == "" {
		response.BadRequest(w, "email is required")
		return
	}

	account, err := h.svc.CreateAcmeAccount(r.Context(), req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, account)
}

func (h *Handler) ListAcmeAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.svc.ListAcmeAccounts(r.Context())
	if err != nil {
		response.InternalError(w, "failed to list accounts")
		return
	}

	response.JSON(w, http.StatusOK, accounts)
}

func (h *Handler) DeleteAcmeAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "invalid account id")
		return
	}

	if err := h.svc.DeleteAcmeAccount(r.Context(), id); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "account deleted"})
}

func handleSSLError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.NotFound(w, "certificate not found")
	case errors.Is(err, ErrAccountNotFound):
		response.NotFound(w, "ACME account not found")
	case errors.Is(err, ErrCertInUse):
		response.BadRequest(w, "certificate is in use by websites, remove it first")
	case errors.Is(err, ErrInvalidPEM):
		response.BadRequest(w, "invalid certificate PEM data")
	default:
		response.InternalError(w, err.Error())
	}
}
