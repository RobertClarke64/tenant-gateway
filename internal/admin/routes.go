package admin

import (
	"github.com/go-chi/chi/v5"

	"tenant-gateway/internal/auth"
)

// Routes returns a chi router with all admin API routes
func Routes(h *Handlers) chi.Router {
	r := chi.NewRouter()

	// All admin routes require admin access
	r.Use(auth.RequireAdmin)

	// Users
	r.Post("/users", h.CreateUser)
	r.Get("/users", h.ListUsers)
	r.Get("/users/{id}", h.GetUser)
	r.Delete("/users/{id}", h.DeleteUser)

	// API Keys (nested under users)
	r.Post("/users/{id}/api-keys", h.CreateAPIKey)
	r.Get("/users/{id}/api-keys", h.ListAPIKeys)
	r.Delete("/users/{id}/api-keys/{keyId}", h.RevokeAPIKey)

	// User-Tenant access
	r.Post("/users/{id}/tenants", h.GrantTenantAccess)
	r.Get("/users/{id}/tenants", h.ListUserTenantAccess)
	r.Put("/users/{id}/tenants/{tenantId}", h.UpdateTenantAccess)
	r.Delete("/users/{id}/tenants/{tenantId}", h.RevokeTenantAccess)

	// Tenants
	r.Post("/tenants", h.CreateTenant)
	r.Get("/tenants", h.ListTenants)
	r.Delete("/tenants/{id}", h.DeleteTenant)

	// Ephemeral tokens
	r.Post("/ephemeral-tokens", h.CreateEphemeralToken)

	return r
}
