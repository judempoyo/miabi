// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"strings"

	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/middlewares"
	"github.com/miabi-io/miabi/internal/services/auth"
)

// setSessionCookie stores the JWT in an HttpOnly, SameSite=Strict cookie so a
// browser never exposes it to JavaScript (XSS can't read or exfiltrate it). CLI
// and API clients keep using the token from the response body via the
// Authorization header. Secure is set whenever the request is HTTPS (directly or
// behind a TLS-terminating reverse proxy), so the cookie still works over plain
// HTTP in local dev.
func setSessionCookie(c *okapi.Context, token string) {
	http.SetCookie(c.ResponseWriter(), &http.Cookie{
		Name:     middlewares.SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(auth.TokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   requestIsHTTPS(c),
		SameSite: http.SameSiteStrictMode,
	})
}

// clearSessionCookie expires the session cookie (used on logout).
func clearSessionCookie(c *okapi.Context) {
	http.SetCookie(c.ResponseWriter(), &http.Cookie{
		Name:     middlewares.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   requestIsHTTPS(c),
		SameSite: http.SameSiteStrictMode,
	})
}

func requestIsHTTPS(c *okapi.Context) bool {
	if c.Request().TLS != nil {
		return true
	}
	return strings.EqualFold(c.Header("X-Forwarded-Proto"), "https")
}
