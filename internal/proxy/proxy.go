package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"tenant-gateway/internal/auth"
)

const (
	HeaderXScopeOrgID = "X-Scope-OrgID"
)

// Proxy handles reverse proxying to the upstream with tenant validation
type Proxy struct {
	upstream    *url.URL
	reverseProxy *httputil.ReverseProxy
	permissions *PermissionChecker
}

// New creates a new proxy
func New(upstreamURL string, timeout time.Duration, permissions *PermissionChecker) (*Proxy, error) {
	upstream, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("parsing upstream URL: %w", err)
	}

	rp := httputil.NewSingleHostReverseProxy(upstream)

	// Configure transport with timeout
	rp.Transport = &http.Transport{
		ResponseHeaderTimeout: timeout,
	}

	// Custom error handler
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
	}

	return &Proxy{
		upstream:     upstream,
		reverseProxy: rp,
		permissions:  permissions,
	}, nil
}

// Handler returns the HTTP handler for proxying requests
func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get auth from context
		authResult := auth.GetAuthFromContext(r.Context())
		if authResult == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get tenant from X-Scope-OrgID header
		tenantName := r.Header.Get(HeaderXScopeOrgID)
		if tenantName == "" {
			http.Error(w, "X-Scope-OrgID header is required", http.StatusBadRequest)
			return
		}

		// Determine required permissions based on endpoint
		needsRead := p.permissions.RequiresRead(r)
		needsWrite := p.permissions.RequiresWrite(r)

		// If endpoint doesn't match any pattern, deny by default
		if !needsRead && !needsWrite {
			http.Error(w, "Endpoint not allowed", http.StatusForbidden)
			return
		}

		// Check tenant access
		if !authResult.CanAccessTenant(tenantName, needsRead, needsWrite) {
			http.Error(w, "Access denied to tenant", http.StatusForbidden)
			return
		}

		// Proxy the request
		p.reverseProxy.ServeHTTP(w, r)
	})
}
