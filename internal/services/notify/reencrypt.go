// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"context"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/crypto"
)

// Reencrypt re-encrypts the secret values in each notification channel's config
// (bot token, webhook URL) under the workspace's active DEK. Idempotent.
func (s *Service) Reencrypt(ctx context.Context, workspaceID uint) (int, error) {
	chs, err := s.repo.ListByWorkspace(workspaceID)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range chs {
		ch := &chs[i]
		if ch.Config == nil {
			continue
		}
		changed := false
		for _, k := range []string{models.ConfigBotToken, models.ConfigWebhookURL} {
			cur, ok := ch.Config[k]
			if !ok || cur == "" {
				continue
			}
			v, c, rerr := crypto.Reencrypt(workspaceID, cur)
			if rerr != nil {
				return n, rerr
			}
			if c {
				ch.Config[k] = v
				changed = true
			}
		}
		if changed {
			if err := s.repo.Update(ch); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}
