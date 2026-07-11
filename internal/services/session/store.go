// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package session maintains a Redis-backed revocation list for JWT sessions,
// keyed by the token's jti claim.
package session

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const revokedPrefix = "session:revoked:"

type Store struct {
	redis *redis.Client
}

func NewStore(client *redis.Client) *Store { return &Store{redis: client} }

// MarkRevoked blacklists a jti until its natural expiry.
func (s *Store) MarkRevoked(ctx context.Context, jti string, expiresAt time.Time) {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return
	}
	s.redis.Set(ctx, revokedPrefix+jti, "1", ttl)
}

// IsRevoked reports whether a jti is blacklisted. Fails open on Redis errors.
func (s *Store) IsRevoked(ctx context.Context, jti string) bool {
	n, err := s.redis.Exists(ctx, revokedPrefix+jti).Result()
	if err != nil {
		return false
	}
	return n > 0
}
