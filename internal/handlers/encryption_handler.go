// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"github.com/jkaninda/okapi"
	"github.com/miabi-io/miabi/internal/services/crypto"
)

// EncryptionInfo is the platform encryption posture shown read-only in admin
// Settings. Per-workspace keys + auto-rotation are operator-configured (env), so
// this is informational, not editable.
type EncryptionInfo struct {
	EncryptionEnabled bool `json:"encryption_enabled"` // a master key is configured
	PerWorkspaceKeys  bool `json:"per_workspace_keys"` // the keyring is wired
	AutoRotate        bool `json:"auto_rotate"`
	RotateMonths      int  `json:"rotate_months"`
	// GatewayConfigEncryption reports whether GOMA_CONFIG_ENCRYPTION_KEY is set, so
	// the config Miabi hands to Goma Gateway (middleware rules + TLS material) is
	// encrypted at rest in transit between the two.
	GatewayConfigEncryption bool `json:"gateway_config_encryption"`
}

// NewEncryptionInfo returns an admin handler reporting the encryption posture.
func NewEncryptionInfo(autoRotate bool, rotateMonths int, gatewayConfigEncryption bool) okapi.HandlerFunc {
	return func(c *okapi.Context) error {
		return ok(c, EncryptionInfo{
			EncryptionEnabled:       crypto.Enabled(),
			PerWorkspaceKeys:        crypto.CurrentKeyring() != nil,
			AutoRotate:              autoRotate,
			RotateMonths:            rotateMonths,
			GatewayConfigEncryption: gatewayConfigEncryption,
		})
	}
}
