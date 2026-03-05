package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"tenant-gateway/internal/database"
)

type contextKey string

const (
	AuthContextKey contextKey = "auth"
)

// AuthResult represents the result of authentication
type AuthResult struct {
	// For API key authentication
	User         *database.User
	TenantAccess map[string]database.TenantAccess // keyed by tenant name

	// For ephemeral token authentication
	EphemeralToken *database.EphemeralToken
	TenantName     string // tenant name for ephemeral tokens
}

// IsAdmin returns true if authenticated as an admin user
func (a *AuthResult) IsAdmin() bool {
	return a.User != nil && a.User.IsAdmin
}

// CanAccessTenant checks if the authenticated entity can access the given tenant
func (a *AuthResult) CanAccessTenant(tenantName string, needsRead, needsWrite bool) bool {
	// Ephemeral token - only valid for its specific tenant
	if a.EphemeralToken != nil {
		if a.TenantName != tenantName {
			return false
		}
		if needsRead && !a.EphemeralToken.CanRead {
			return false
		}
		if needsWrite && !a.EphemeralToken.CanWrite {
			return false
		}
		return true
	}

	// API key - check user's tenant access
	if a.User != nil && a.TenantAccess != nil {
		access, ok := a.TenantAccess[tenantName]
		if !ok {
			return false
		}
		if needsRead && !access.CanRead {
			return false
		}
		if needsWrite && !access.CanWrite {
			return false
		}
		return true
	}

	return false
}

// Authenticator handles token validation
type Authenticator struct {
	db *database.DB
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(db *database.DB) *Authenticator {
	return &Authenticator{db: db}
}

// Middleware returns an HTTP middleware that authenticates requests
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, err := a.Authenticate(r.Context(), r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), AuthContextKey, result)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Authenticate validates the request and returns auth result
func (a *Authenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthResult, error) {
	token := extractBearerToken(r)
	if token == "" {
		return nil, errors.New("no token provided")
	}

	// Try API key first
	result, err := a.tryAPIKey(ctx, token)
	if err == nil {
		return result, nil
	}

	// Try ephemeral token
	result, err = a.tryEphemeralToken(ctx, token)
	if err == nil {
		return result, nil
	}

	return nil, errors.New("invalid token")
}

func (a *Authenticator) tryAPIKey(ctx context.Context, token string) (*AuthResult, error) {
	prefix := GetTokenPrefix(token)

	keys, err := a.db.ListValidAPIKeysByPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if VerifyToken(token, key.KeyHash) {
			// Found valid key - load user and tenant access
			user, err := a.db.GetUser(ctx, key.UserID)
			if err != nil {
				return nil, err
			}

			tenantAccess, err := a.db.GetUserTenantAccess(ctx, user.ID)
			if err != nil {
				return nil, err
			}

			accessMap := make(map[string]database.TenantAccess)
			for _, ta := range tenantAccess {
				accessMap[ta.Tenant.Name] = ta
			}

			return &AuthResult{
				User:         user,
				TenantAccess: accessMap,
			}, nil
		}
	}

	return nil, errors.New("invalid API key")
}

func (a *Authenticator) tryEphemeralToken(ctx context.Context, token string) (*AuthResult, error) {
	prefix := GetTokenPrefix(token)

	tokens, err := a.db.ListValidEphemeralTokensByPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}

	for _, t := range tokens {
		if VerifyToken(token, t.TokenHash) {
			// Found valid token - get tenant name
			tenantName, err := a.db.GetTenantNameByID(ctx, t.TenantID)
			if err != nil {
				return nil, err
			}

			return &AuthResult{
				EphemeralToken: &t,
				TenantName:     tenantName,
			}, nil
		}
	}

	return nil, errors.New("invalid ephemeral token")
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// GetAuthFromContext retrieves the auth result from context
func GetAuthFromContext(ctx context.Context) *AuthResult {
	result, ok := ctx.Value(AuthContextKey).(*AuthResult)
	if !ok {
		return nil
	}
	return result
}

// RequireAdmin returns middleware that requires admin access
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := GetAuthFromContext(r.Context())
		if auth == nil || !auth.IsAdmin() {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
