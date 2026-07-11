// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package stack

import (
	"context"

	"github.com/miabi-io/miabi/internal/services/crypto"
)

// Reencrypt re-encrypts every secret env var of the workspace's stacks under the
// workspace's active DEK. Non-secret vars are skipped. Idempotent.
func (s *Service) Reencrypt(ctx context.Context, workspaceID uint) (int, error) {
	stacks, err := s.repo.ListByWorkspace(workspaceID)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range stacks {
		vars, verr := s.env.ListByStack(stacks[i].ID)
		if verr != nil {
			return n, verr
		}
		for j := range vars {
			if !vars[j].IsSecret {
				continue
			}
			v, changed, rerr := crypto.Reencrypt(workspaceID, vars[j].Value)
			if rerr != nil {
				return n, rerr
			}
			if changed {
				vars[j].Value = v
				if err := s.env.Upsert(&vars[j]); err != nil {
					return n, err
				}
				n++
			}
		}
	}
	return n, nil
}
