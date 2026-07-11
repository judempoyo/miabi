// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "time"

// WorkspaceBackupSettings holds a workspace's shared backup target: one
// S3-compatible bucket + credentials, plus the path prefixes under which
// database and volume backups are stored. Both database and volume backups read
// this single config. The secret key is encrypted at rest and never returned.
type WorkspaceBackupSettings struct {
	ID          uint `json:"id" gorm:"primaryKey"`
	WorkspaceID uint `json:"workspace_id" gorm:"uniqueIndex;not null"`

	S3Enabled        bool   `json:"s3_enabled"`
	S3Endpoint       string `json:"s3_endpoint,omitempty"`
	S3Bucket         string `json:"s3_bucket,omitempty"`
	S3Region         string `json:"s3_region,omitempty"`
	S3AccessKey      string `json:"s3_access_key,omitempty"`
	S3SecretKeyEnc   string `json:"-" gorm:"column:s3_secret_key_enc"` // encrypted at rest
	S3UseSSL         bool   `json:"s3_use_ssl"`
	S3ForcePathStyle bool   `json:"s3_force_path_style"`

	// Path prefixes within the bucket (one shared bucket, two prefixes).
	DatabaseBackupPath string `json:"database_backup_path,omitempty"`
	VolumeBackupPath   string `json:"volume_backup_path,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// S3SecretSet reports whether a secret key is stored, without exposing it.
	// Not persisted; populated on read so the UI can render "••••• (set)".
	S3SecretSet bool `json:"s3_secret_set" gorm:"-"`
}
