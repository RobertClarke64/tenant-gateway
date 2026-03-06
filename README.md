# Tenant Gateway

A multi-tenant reverse proxy for Grafana Loki (and similar multi-tenant services) that provides authentication, authorization, and tenant isolation.

## Features

- **Multi-tenant access control** - Users and API keys can be granted read/write access to specific tenants
- **Two authentication methods**:
  - **API Keys** - Long-lived credentials tied to users with access to multiple tenants
  - **Ephemeral Tokens** - Short-lived, scoped tokens for temporary access to a single tenant
- **Secure credential storage** - API keys and tokens are hashed with bcrypt
- **Configurable endpoint permissions** - Define which HTTP methods/paths require read vs write access
- **Admin API** - RESTful API for managing users, tenants, API keys, and ephemeral tokens
- **Swagger UI** - Built-in API documentation at `/swagger/`

## Quick Start

### Using Docker Compose

1. Start the services:
   ```bash
   docker compose up -d
   ```

2. Run the setup script to create an admin user and test tenant:
   ```bash
   ./docker/setup.sh
   ```

3. Access the services:
   - Gateway API: http://localhost:8080
   - Swagger UI: http://localhost:8080/swagger/
   - Grafana: http://localhost:3000

### Manual Setup

1. Build the binary:
   ```bash
   go build -o tenant-gateway ./cmd/gateway
   ```

2. Create a config file (see `config.example.yaml`):
   ```yaml
   server:
     listen: ":8080"

   upstream:
     url: "http://localhost:3100"
     timeout: 30s

   database:
     url: "postgres://user:password@localhost:5432/tenant_gateway?sslmode=disable"

   auth:
     token_hash_cost: 10

   endpoints:
     read:
       - "GET /**"
     write:
       - "POST /loki/api/v1/push"
       - "POST /api/v1/push"
       - "PUT /**"
       - "DELETE /**"
   ```

3. Run migrations and bootstrap an admin user:
   ```bash
   ./tenant-gateway -config config.yaml -bootstrap admin
   ```

4. Start the gateway:
   ```bash
   ./tenant-gateway -config config.yaml
   ```

## Configuration

Configuration can be provided via YAML file and/or environment variables.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `LISTEN_ADDR` | Server listen address (default: `:8080`) |
| `UPSTREAM_URL` | Upstream service URL (required) |
| `DATABASE_URL` | PostgreSQL connection string (required) |

### Endpoint Permissions

The `endpoints` configuration defines which requests require read or write permissions:

```yaml
endpoints:
  read:
    - "GET /**"           # All GET requests require read access
  write:
    - "POST /loki/api/v1/push"  # Push endpoint requires write access
    - "PUT /**"
    - "DELETE /**"
```

Patterns support glob syntax with `**` matching any path segment.

## Authentication

All requests (except `/health` and `/swagger/*`) require authentication via Bearer token:

```
Authorization: Bearer <api_key_or_ephemeral_token>
```

### API Keys

API keys are tied to users and inherit the user's tenant permissions. They can optionally have an expiration date.

### Ephemeral Tokens

Ephemeral tokens are scoped to a single tenant with explicit read/write permissions. They always have an expiration time and are useful for granting temporary access.

## Usage

### Proxying Requests

The gateway proxies all non-admin requests to the upstream service. Include the `X-Scope-OrgID` header to specify the tenant:

```bash
# Query logs from a tenant
curl http://localhost:8080/loki/api/v1/query \
  -H "Authorization: Bearer <api_key>" \
  -H "X-Scope-OrgID: my-tenant" \
  -G --data-urlencode 'query={app="myapp"}'

# Push logs to a tenant
curl -X POST http://localhost:8080/loki/api/v1/push \
  -H "Authorization: Bearer <api_key>" \
  -H "X-Scope-OrgID: my-tenant" \
  -H "Content-Type: application/json" \
  -d '{"streams": [{"stream": {"app": "myapp"}, "values": [["1234567890000000000", "log message"]]}]}'
```

### Admin API

The admin API is available at `/admin/*` and requires an admin user's API key.

#### Users

```bash
# Create a user
curl -X POST http://localhost:8080/admin/users \
  -H "Authorization: Bearer <admin_key>" \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "is_admin": false}'

# List users
curl http://localhost:8080/admin/users \
  -H "Authorization: Bearer <admin_key>"

# Delete a user
curl -X DELETE http://localhost:8080/admin/users/<user_id> \
  -H "Authorization: Bearer <admin_key>"
```

#### Tenants

```bash
# Create a tenant
curl -X POST http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer <admin_key>" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-tenant"}'

# List tenants
curl http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer <admin_key>"
```

#### Tenant Access

```bash
# Grant user access to a tenant
curl -X POST http://localhost:8080/admin/users/<user_id>/tenants \
  -H "Authorization: Bearer <admin_key>" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "<tenant_id>", "can_read": true, "can_write": false}'

# Update access permissions
curl -X PUT http://localhost:8080/admin/users/<user_id>/tenants/<tenant_id> \
  -H "Authorization: Bearer <admin_key>" \
  -H "Content-Type: application/json" \
  -d '{"can_read": true, "can_write": true}'

# Revoke access
curl -X DELETE http://localhost:8080/admin/users/<user_id>/tenants/<tenant_id> \
  -H "Authorization: Bearer <admin_key>"
```

#### API Keys

```bash
# Create an API key for a user
curl -X POST http://localhost:8080/admin/users/<user_id>/api-keys \
  -H "Authorization: Bearer <admin_key>" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-key"}'

# List API keys (hashes are not returned)
curl http://localhost:8080/admin/users/<user_id>/api-keys \
  -H "Authorization: Bearer <admin_key>"

# Revoke an API key
curl -X DELETE http://localhost:8080/admin/users/<user_id>/api-keys/<key_id> \
  -H "Authorization: Bearer <admin_key>"
```

#### Ephemeral Tokens

```bash
# Create an ephemeral token
curl -X POST http://localhost:8080/admin/ephemeral-tokens \
  -H "Authorization: Bearer <admin_key>" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "<tenant_id>",
    "can_read": true,
    "can_write": false,
    "expires_in": "24h"
  }'
```

## Architecture

```
┌─────────────┐      ┌─────────────────┐      ┌──────────────┐
│   Client    │─────▶│  Tenant Gateway │─────▶│  Loki/Other  │
└─────────────┘      └─────────────────┘      └──────────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │   PostgreSQL    │
                     └─────────────────┘
```

The gateway:
1. Authenticates requests using API keys or ephemeral tokens
2. Validates the user/token has access to the requested tenant (`X-Scope-OrgID`)
3. Checks the endpoint requires read or write permission based on configuration
4. Proxies authorized requests to the upstream service

## Development

### Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Docker and Docker Compose (optional)

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o tenant-gateway ./cmd/gateway
```
