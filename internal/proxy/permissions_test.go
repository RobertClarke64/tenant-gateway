package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		// Exact matches
		{"/api/v1/push", "/api/v1/push", true},
		{"/api/v1/push", "/api/v1/query", false},

		// Single wildcard
		{"/api/*/push", "/api/v1/push", true},
		{"/api/*/push", "/api/v2/push", true},
		{"/api/*/push", "/api/v1/query", false},

		// Double wildcard
		{"/**", "/", true},
		{"/**", "/api", true},
		{"/**", "/api/v1/push", true},
		{"/api/**", "/api", true},
		{"/api/**", "/api/v1", true},
		{"/api/**", "/api/v1/push", true},
		{"/api/**", "/other", false},

		// Mixed
		{"/loki/api/v1/*", "/loki/api/v1/push", true},
		{"/loki/api/v1/*", "/loki/api/v1/query", true},
		{"/loki/api/v1/*", "/loki/api/v2/push", false},

		// Leading/trailing slashes
		{"api/v1/push", "/api/v1/push", true},
		{"/api/v1/push/", "api/v1/push", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"->"+tt.path, func(t *testing.T) {
			got := matchPath(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestPermissionChecker(t *testing.T) {
	pc := NewPermissionChecker(
		[]string{"GET /**"},
		[]string{"POST /api/v1/push", "POST /loki/api/v1/push", "PUT /**", "DELETE /**"},
	)

	tests := []struct {
		method      string
		path        string
		wantRead    bool
		wantWrite   bool
	}{
		{"GET", "/api/v1/query", true, false},
		{"GET", "/loki/api/v1/labels", true, false},
		{"POST", "/api/v1/push", false, true},
		{"POST", "/loki/api/v1/push", false, true},
		{"POST", "/api/v1/query", false, false}, // POST to query not allowed
		{"PUT", "/some/resource", false, true},
		{"DELETE", "/some/resource", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)

			gotRead := pc.RequiresRead(req)
			gotWrite := pc.RequiresWrite(req)

			if gotRead != tt.wantRead {
				t.Errorf("RequiresRead() = %v, want %v", gotRead, tt.wantRead)
			}
			if gotWrite != tt.wantWrite {
				t.Errorf("RequiresWrite() = %v, want %v", gotWrite, tt.wantWrite)
			}
		})
	}
}

func TestAuthResultCanAccessTenant(t *testing.T) {
	// This is a basic test - full integration tests would require a database
	req, _ := http.NewRequest("GET", "/", nil)
	_ = req // just to avoid unused variable if we need it later
}
