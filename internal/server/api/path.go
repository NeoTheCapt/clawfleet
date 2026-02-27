package api

import "strings"

// splitSubpath splits URL path after a prefix into id + optional subpath.
// Example: path=/api/agents/abc/redeploy, prefix=/api/agents/ => id=abc, sub=redeploy
func splitSubpath(path, prefix string) (id string, sub string) {
	p := strings.TrimPrefix(path, prefix)
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "", ""
	}
	parts := strings.SplitN(p, "/", 2)
	id = parts[0]
	if len(parts) > 1 {
		sub = parts[1]
	}
	return id, sub
}
