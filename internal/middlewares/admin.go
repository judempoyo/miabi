// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

// RequireSystemAdmin allows only platform super-admins. It resolves the role
// from the database, so it works for both JWT and API-key authentication.
// Must run after Authenticate.
func RequireSystemAdmin(users *repositories.UserRepository) okapi.Middleware {
	return func(c *okapi.Context) error {
		uid := UserID(c)
		if uid == 0 {
			return c.AbortUnauthorized("authentication required")
		}
		user, err := users.FindByID(uid)
		if err != nil || user.Role != models.SystemRoleAdmin {
			return c.AbortForbidden("platform admin required")
		}
		return c.Next()
	}
}
