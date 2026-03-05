package database

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserTenant struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	CanRead  bool      `json:"can_read"`
	CanWrite bool      `json:"can_write"`
}

type TenantAccess struct {
	Tenant   Tenant `json:"tenant"`
	CanRead  bool   `json:"can_read"`
	CanWrite bool   `json:"can_write"`
}

type APIKey struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	KeyHash   string     `json:"-"`
	KeyPrefix string     `json:"key_prefix"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Revoked   bool       `json:"revoked"`
}

type EphemeralToken struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	TokenHash       string    `json:"-"`
	TokenPrefix     string    `json:"token_prefix"`
	CanRead         bool      `json:"can_read"`
	CanWrite        bool      `json:"can_write"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedByUserID uuid.UUID `json:"created_by_user_id"`
}

// AuthenticatedUser represents a user authenticated via API key
type AuthenticatedUser struct {
	User          User
	TenantAccess  map[string]TenantAccess // keyed by tenant name
}

// AuthenticatedToken represents authentication via ephemeral token
type AuthenticatedToken struct {
	Token      EphemeralToken
	TenantName string
}
