// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// UpgradeStep records an applied, versioned data-upgrade step so it runs at
// most once. Schema changes use GORM AutoMigrate; data backfills use steps.
type UpgradeStep struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Version   string    `json:"version" gorm:"index;not null"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	AppliedAt time.Time `json:"applied_at"`
}
