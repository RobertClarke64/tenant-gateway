package proxy

import (
	"net/http"
	"strings"
)

// PermissionChecker determines if an endpoint requires read or write permission
type PermissionChecker struct {
	readPatterns  []endpointPattern
	writePatterns []endpointPattern
}

type endpointPattern struct {
	method  string
	pattern string
}

// NewPermissionChecker creates a permission checker from config
func NewPermissionChecker(readEndpoints, writeEndpoints []string) *PermissionChecker {
	pc := &PermissionChecker{}

	for _, ep := range readEndpoints {
		pc.readPatterns = append(pc.readPatterns, parsePattern(ep))
	}
	for _, ep := range writeEndpoints {
		pc.writePatterns = append(pc.writePatterns, parsePattern(ep))
	}

	return pc
}

func parsePattern(s string) endpointPattern {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return endpointPattern{
			method:  strings.ToUpper(parts[0]),
			pattern: parts[1],
		}
	}
	// No method specified - match all methods
	return endpointPattern{
		method:  "*",
		pattern: s,
	}
}

// RequiresRead returns true if the request requires read permission
func (pc *PermissionChecker) RequiresRead(r *http.Request) bool {
	return pc.matchesAny(r, pc.readPatterns)
}

// RequiresWrite returns true if the request requires write permission
func (pc *PermissionChecker) RequiresWrite(r *http.Request) bool {
	return pc.matchesAny(r, pc.writePatterns)
}

func (pc *PermissionChecker) matchesAny(r *http.Request, patterns []endpointPattern) bool {
	for _, p := range patterns {
		if pc.matches(r, p) {
			return true
		}
	}
	return false
}

func (pc *PermissionChecker) matches(r *http.Request, p endpointPattern) bool {
	// Check method
	if p.method != "*" && p.method != r.Method {
		return false
	}

	// Check path pattern
	return matchPath(p.pattern, r.URL.Path)
}

// matchPath checks if a path matches a pattern with glob support
// Supports:
//   - * matches any single path segment
//   - ** matches any number of path segments
func matchPath(pattern, path string) bool {
	// Normalize paths
	pattern = strings.Trim(pattern, "/")
	path = strings.Trim(path, "/")

	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	return matchParts(patternParts, pathParts)
}

func matchParts(pattern, path []string) bool {
	pi := 0 // pattern index
	pa := 0 // path index

	for pi < len(pattern) {
		if pattern[pi] == "**" {
			// ** matches zero or more segments
			if pi == len(pattern)-1 {
				// ** at end matches everything
				return true
			}

			// Try matching the rest of the pattern at each position
			for pa <= len(path) {
				if matchParts(pattern[pi+1:], path[pa:]) {
					return true
				}
				pa++
			}
			return false
		}

		if pa >= len(path) {
			return false
		}

		if pattern[pi] == "*" {
			// * matches exactly one segment
			pi++
			pa++
			continue
		}

		if pattern[pi] != path[pa] {
			return false
		}

		pi++
		pa++
	}

	return pa == len(path)
}
