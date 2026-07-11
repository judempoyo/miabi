// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package application

import (
	"context"

	"github.com/miabi-io/miabi/internal/services/crypto"
)

// Reencrypt re-encrypts every secret env var of the workspace's applications
// under the workspace's active DEK. Non-secret vars (stored plaintext) are
// skipped. Idempotent; returns the number rewritten.
func (s *Service) Reencrypt(ctx context.Context, workspaceID uint) (int, error) {
	apps, err := s.apps.ListByWorkspace(workspaceID)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range apps {
		vars, verr := s.apps.ListEnvVars(apps[i].ID)
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
				if err := s.apps.UpsertEnvVar(&vars[j]); err != nil {
					return n, err
				}
				n++
			}
		}
	}
	return n, nil
}
