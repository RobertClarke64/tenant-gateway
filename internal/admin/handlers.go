package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"tenant-gateway/internal/auth"
	"tenant-gateway/internal/database"
)

type Handlers struct {
	db            *database.DB
	tokenHashCost int
}

func NewHandlers(db *database.DB, tokenHashCost int) *Handlers {
	return &Handlers{
		db:            db,
		tokenHashCost: tokenHashCost,
	}
}

// User handlers

type CreateUserRequest struct {
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
}

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	user, err := h.db.CreateUser(r.Context(), req.Username, req.IsAdmin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.db.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUser(r.Context(), id)
	if errors.Is(err, database.ErrNotFound) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Tenant handlers

type CreateTenantRequest struct {
	Name string `json:"name"`
}

func (h *Handlers) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	tenant, err := h.db.CreateTenant(r.Context(), req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, tenant)
}

func (h *Handlers) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.db.ListTenants(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, tenants)
}

func (h *Handlers) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteTenant(r.Context(), id); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Tenant not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// API Key handlers

type CreateAPIKeyRequest struct {
	Name      string `json:"name"`
	ExpiresIn string `json:"expires_in,omitempty"` // e.g., "24h", "720h"
}

type CreateAPIKeyResponse struct {
	*database.APIKey
	Key string `json:"key"` // Only returned on creation
}

func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify user exists
	if _, err := h.db.GetUser(r.Context(), userID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate token
	plaintext, hash, prefix, err := auth.GenerateToken(h.tokenHashCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse expiration
	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			http.Error(w, "Invalid expires_in duration", http.StatusBadRequest)
			return
		}
		t := time.Now().Add(duration)
		expiresAt = &t
	}

	key, err := h.db.CreateAPIKey(r.Context(), userID, hash, prefix, req.Name, expiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, CreateAPIKeyResponse{
		APIKey: key,
		Key:    plaintext,
	})
}

func (h *Handlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	keys, err := h.db.ListAPIKeysByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, keys)
}

func (h *Handlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID, err := uuid.Parse(chi.URLParam(r, "keyId"))
	if err != nil {
		http.Error(w, "Invalid key ID", http.StatusBadRequest)
		return
	}

	if err := h.db.RevokeAPIKey(r.Context(), keyID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "API key not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// User-Tenant access handlers

type GrantTenantAccessRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	CanRead  bool      `json:"can_read"`
	CanWrite bool      `json:"can_write"`
}

type UpdateTenantAccessRequest struct {
	CanRead  bool `json:"can_read"`
	CanWrite bool `json:"can_write"`
}

func (h *Handlers) GrantTenantAccess(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req GrantTenantAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.db.GrantTenantAccess(r.Context(), userID, req.TenantID, req.CanRead, req.CanWrite); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handlers) ListUserTenantAccess(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	access, err := h.db.GetUserTenantAccess(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, access)
}

func (h *Handlers) UpdateTenantAccess(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(chi.URLParam(r, "tenantId"))
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	var req UpdateTenantAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.db.GrantTenantAccess(r.Context(), userID, tenantID, req.CanRead, req.CanWrite); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) RevokeTenantAccess(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(chi.URLParam(r, "tenantId"))
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	if err := h.db.RevokeTenantAccess(r.Context(), userID, tenantID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Access not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Ephemeral token handlers

type CreateEphemeralTokenRequest struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	CanRead   bool      `json:"can_read"`
	CanWrite  bool      `json:"can_write"`
	ExpiresIn string    `json:"expires_in"` // Required, e.g., "1h", "24h"
}

type CreateEphemeralTokenResponse struct {
	*database.EphemeralToken
	Token string `json:"token"` // Only returned on creation
}

func (h *Handlers) CreateEphemeralToken(w http.ResponseWriter, r *http.Request) {
	authResult := auth.GetAuthFromContext(r.Context())
	if authResult == nil || authResult.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateEphemeralTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ExpiresIn == "" {
		http.Error(w, "expires_in is required", http.StatusBadRequest)
		return
	}

	duration, err := time.ParseDuration(req.ExpiresIn)
	if err != nil {
		http.Error(w, "Invalid expires_in duration", http.StatusBadRequest)
		return
	}

	// Verify tenant exists
	if _, err := h.db.GetTenant(r.Context(), req.TenantID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Tenant not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate token
	plaintext, hash, prefix, err := auth.GenerateToken(h.tokenHashCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(duration)

	token, err := h.db.CreateEphemeralToken(
		r.Context(),
		req.TenantID,
		hash,
		prefix,
		req.CanRead,
		req.CanWrite,
		expiresAt,
		authResult.User.ID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, CreateEphemeralTokenResponse{
		EphemeralToken: token,
		Token:          plaintext,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
