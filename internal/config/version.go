// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

// Build information, overridable at link time via -ldflags.
var (
	Version  = "dev"
	CommitID = "unknown"
)
