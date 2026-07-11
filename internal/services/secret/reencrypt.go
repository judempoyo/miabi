// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package secret

import (
	"context"

	"github.com/miabi-io/miabi/internal/services/crypto"
)

// Reencrypt re-encrypts this owner's workspace secrets under the workspace's
// active DEK (key rotation). Idempotent; returns the number rewritten.
func (s *Service) Reencrypt(ctx context.Context, workspaceID uint) (int, error) {
	rows, err := s.repo.ListByWorkspace(workspaceID)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range rows {
		v, changed, rerr := crypto.Reencrypt(workspaceID, rows[i].ValueEnc)
		if rerr != nil {
			return n, rerr
		}
		if changed {
			rows[i].ValueEnc = v
			if uerr := s.repo.Update(&rows[i]); uerr != nil {
				return n, uerr
			}
			n++
		}
	}
	return n, nil
}
