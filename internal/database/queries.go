package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")

// User operations

func (db *DB) CreateUser(ctx context.Context, username string, isAdmin bool) (*User, error) {
	user := &User{
		ID:        uuid.New(),
		Username:  username,
		IsAdmin:   isAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO users (id, username, is_admin, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		user.ID, user.Username, user.IsAdmin, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

func (db *DB) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, username, is_admin, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Username, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return user, nil
}

func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user := &User{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, username, is_admin, created_at, updated_at
		 FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return user, nil
}

func (db *DB) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, username, is_admin, created_at, updated_at
		 FROM users ORDER BY username`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
	}

	return users, nil
}

func (db *DB) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result, err := db.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Tenant operations

func (db *DB) CreateTenant(ctx context.Context, name string) (*Tenant, error) {
	tenant := &Tenant{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4)`,
		tenant.ID, tenant.Name, tenant.CreatedAt, tenant.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("creating tenant: %w", err)
	}

	return tenant, nil
}

func (db *DB) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	tenant := &Tenant{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, created_at, updated_at
		 FROM tenants WHERE id = $1`,
		id,
	).Scan(&tenant.ID, &tenant.Name, &tenant.CreatedAt, &tenant.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting tenant: %w", err)
	}

	return tenant, nil
}

func (db *DB) GetTenantByName(ctx context.Context, name string) (*Tenant, error) {
	tenant := &Tenant{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, created_at, updated_at
		 FROM tenants WHERE name = $1`,
		name,
	).Scan(&tenant.ID, &tenant.Name, &tenant.CreatedAt, &tenant.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting tenant: %w", err)
	}

	return tenant, nil
}

func (db *DB) ListTenants(ctx context.Context) ([]Tenant, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, created_at, updated_at
		 FROM tenants ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning tenant: %w", err)
		}
		tenants = append(tenants, t)
	}

	return tenants, nil
}

func (db *DB) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	result, err := db.pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting tenant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// User-Tenant access operations

func (db *DB) GrantTenantAccess(ctx context.Context, userID, tenantID uuid.UUID, canRead, canWrite bool) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO user_tenants (user_id, tenant_id, can_read, can_write)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, tenant_id) DO UPDATE SET can_read = $3, can_write = $4`,
		userID, tenantID, canRead, canWrite,
	)
	if err != nil {
		return fmt.Errorf("granting tenant access: %w", err)
	}
	return nil
}

func (db *DB) RevokeTenantAccess(ctx context.Context, userID, tenantID uuid.UUID) error {
	result, err := db.pool.Exec(ctx,
		"DELETE FROM user_tenants WHERE user_id = $1 AND tenant_id = $2",
		userID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("revoking tenant access: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DB) GetUserTenantAccess(ctx context.Context, userID uuid.UUID) ([]TenantAccess, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT t.id, t.name, t.created_at, t.updated_at, ut.can_read, ut.can_write
		 FROM tenants t
		 JOIN user_tenants ut ON t.id = ut.tenant_id
		 WHERE ut.user_id = $1
		 ORDER BY t.name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting user tenant access: %w", err)
	}
	defer rows.Close()

	var access []TenantAccess
	for rows.Next() {
		var ta TenantAccess
		if err := rows.Scan(&ta.Tenant.ID, &ta.Tenant.Name, &ta.Tenant.CreatedAt, &ta.Tenant.UpdatedAt, &ta.CanRead, &ta.CanWrite); err != nil {
			return nil, fmt.Errorf("scanning tenant access: %w", err)
		}
		access = append(access, ta)
	}

	return access, nil
}

// API Key operations

func (db *DB) CreateAPIKey(ctx context.Context, userID uuid.UUID, keyHash, keyPrefix, name string, expiresAt *time.Time) (*APIKey, error) {
	key := &APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Name:      name,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		Revoked:   false,
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO api_keys (id, user_id, key_hash, key_prefix, name, created_at, expires_at, revoked)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		key.ID, key.UserID, key.KeyHash, key.KeyPrefix, key.Name, key.CreatedAt, key.ExpiresAt, key.Revoked,
	)
	if err != nil {
		return nil, fmt.Errorf("creating API key: %w", err)
	}

	return key, nil
}

func (db *DB) GetAPIKey(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	key := &APIKey{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, key_hash, key_prefix, name, created_at, expires_at, revoked
		 FROM api_keys WHERE id = $1`,
		id,
	).Scan(&key.ID, &key.UserID, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.CreatedAt, &key.ExpiresAt, &key.Revoked)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting API key: %w", err)
	}

	return key, nil
}

func (db *DB) ListAPIKeysByUser(ctx context.Context, userID uuid.UUID) ([]APIKey, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, key_hash, key_prefix, name, created_at, expires_at, revoked
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.CreatedAt, &k.ExpiresAt, &k.Revoked); err != nil {
			return nil, fmt.Errorf("scanning API key: %w", err)
		}
		keys = append(keys, k)
	}

	return keys, nil
}

func (db *DB) ListValidAPIKeysByPrefix(ctx context.Context, prefix string) ([]APIKey, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, key_hash, key_prefix, name, created_at, expires_at, revoked
		 FROM api_keys
		 WHERE key_prefix = $1 AND revoked = FALSE
		   AND (expires_at IS NULL OR expires_at > NOW())`,
		prefix,
	)
	if err != nil {
		return nil, fmt.Errorf("listing API keys by prefix: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.CreatedAt, &k.ExpiresAt, &k.Revoked); err != nil {
			return nil, fmt.Errorf("scanning API key: %w", err)
		}
		keys = append(keys, k)
	}

	return keys, nil
}

func (db *DB) RevokeAPIKey(ctx context.Context, id uuid.UUID) error {
	result, err := db.pool.Exec(ctx,
		"UPDATE api_keys SET revoked = TRUE WHERE id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("revoking API key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Ephemeral Token operations

func (db *DB) CreateEphemeralToken(ctx context.Context, tenantID uuid.UUID, tokenHash, tokenPrefix string, canRead, canWrite bool, expiresAt time.Time, createdByUserID uuid.UUID) (*EphemeralToken, error) {
	token := &EphemeralToken{
		ID:              uuid.New(),
		TenantID:        tenantID,
		TokenHash:       tokenHash,
		TokenPrefix:     tokenPrefix,
		CanRead:         canRead,
		CanWrite:        canWrite,
		CreatedAt:       time.Now(),
		ExpiresAt:       expiresAt,
		CreatedByUserID: createdByUserID,
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO ephemeral_tokens (id, tenant_id, token_hash, token_prefix, can_read, can_write, created_at, expires_at, created_by_user_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		token.ID, token.TenantID, token.TokenHash, token.TokenPrefix, token.CanRead, token.CanWrite, token.CreatedAt, token.ExpiresAt, token.CreatedByUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("creating ephemeral token: %w", err)
	}

	return token, nil
}

func (db *DB) ListValidEphemeralTokensByPrefix(ctx context.Context, prefix string) ([]EphemeralToken, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, tenant_id, token_hash, token_prefix, can_read, can_write, created_at, expires_at, created_by_user_id
		 FROM ephemeral_tokens
		 WHERE token_prefix = $1 AND expires_at > NOW()`,
		prefix,
	)
	if err != nil {
		return nil, fmt.Errorf("listing ephemeral tokens by prefix: %w", err)
	}
	defer rows.Close()

	var tokens []EphemeralToken
	for rows.Next() {
		var t EphemeralToken
		if err := rows.Scan(&t.ID, &t.TenantID, &t.TokenHash, &t.TokenPrefix, &t.CanRead, &t.CanWrite, &t.CreatedAt, &t.ExpiresAt, &t.CreatedByUserID); err != nil {
			return nil, fmt.Errorf("scanning ephemeral token: %w", err)
		}
		tokens = append(tokens, t)
	}

	return tokens, nil
}

func (db *DB) GetTenantNameByID(ctx context.Context, id uuid.UUID) (string, error) {
	var name string
	err := db.pool.QueryRow(ctx, "SELECT name FROM tenants WHERE id = $1", id).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("getting tenant name: %w", err)
	}
	return name, nil
}

func (db *DB) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	result, err := db.pool.Exec(ctx, "DELETE FROM ephemeral_tokens WHERE expires_at < NOW()")
	if err != nil {
		return 0, fmt.Errorf("cleaning up expired tokens: %w", err)
	}
	return result.RowsAffected(), nil
}
