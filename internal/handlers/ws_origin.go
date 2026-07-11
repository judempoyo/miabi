// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"net/url"
	"strings"
)

// allowWSOrigin builds a websocket CheckOrigin function that blocks cross-site
// WebSocket hijacking (CSWSH) while still admitting non-browser clients. It
// permits:
//   - requests with no Origin header — non-browser clients (node agents, runners,
//     CLIs; Go WebSocket dialers do not set Origin), and
//   - browser requests whose Origin is same-origin with the request Host, or is
//     present in the configured allowlist (the CORS origins + web UI URL).
//
// A cross-site browser Origin is rejected. A "*" entry in allowed disables the
// check entirely (dev, where CORS is also "*").
func allowWSOrigin(allowed []string) func(*http.Request) bool {
	allowSet := make(map[string]struct{}, len(allowed))
	wildcard := false
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if a == "*" {
			wildcard = true
			continue
		}
		if a != "" {
			allowSet[normOrigin(a)] = struct{}{}
		}
	}
	return func(r *http.Request) bool {
		if wildcard {
			return true
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // non-browser client (no ambient credentials to hijack)
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		if strings.EqualFold(u.Host, r.Host) {
			return true // same-origin
		}
		_, ok := allowSet[normOrigin(origin)]
		return ok
	}
}

// normOrigin lower-cases an origin and strips a trailing slash for comparison.
func normOrigin(o string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(o), "/"))
}
