// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package platformbackup

import (
	"errors"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/backup"
	"github.com/miabi-io/miabi/internal/services/crypto"
	"gorm.io/gorm"
)

// GetSettings returns the platform backup settings, or an empty (unsaved) record
// when none exist yet. The S3 secret is never included; S3SecretSet reports its
// presence so the UI can render "••••• (set)".
func (s *Service) GetSettings() (*models.PlatformBackupSettings, error) {
	st, err := s.getSettings()
	if err != nil {
		return nil, err
	}
	st.S3SecretSet = st.S3SecretKeyEnc != ""
	return st, nil
}

// getSettings loads the single settings row, returning an empty record when none
// exists.
func (s *Service) getSettings() (*models.PlatformBackupSettings, error) {
	st, err := s.settings.Get()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.PlatformBackupSettings{}, nil
	}
	if err != nil {
		return nil, err
	}
	return st, nil
}

// SaveInput carries the desired platform backup settings. S3SecretKey is
// nil/empty to keep the stored secret unchanged (so the UI never round-trips it).
type SaveInput struct {
	S3Enabled        bool
	S3Endpoint       string
	S3Bucket         string
	S3Region         string
	S3AccessKey      string
	S3SecretKey      *string
	S3UseSSL         bool
	S3ForcePathStyle bool

	DatabaseBackupPath string
	VolumeBackupPath   string

	ScheduleEnabled bool
	ScheduleCron    string
	MaxBackups      int
	RetentionDays   int
	Volumes         []string
}

// SaveSettings upserts the platform backup settings, encrypting the S3 secret
// (platform scope) when a new one is supplied and preserving it otherwise.
func (s *Service) SaveSettings(in SaveInput) (*models.PlatformBackupSettings, error) {
	st, err := s.getSettings()
	if err != nil {
		return nil, err
	}

	st.S3Enabled = in.S3Enabled
	st.S3Endpoint = in.S3Endpoint
	st.S3Bucket = in.S3Bucket
	st.S3Region = in.S3Region
	st.S3AccessKey = in.S3AccessKey
	st.S3UseSSL = in.S3UseSSL
	st.S3ForcePathStyle = in.S3ForcePathStyle
	st.DatabaseBackupPath = in.DatabaseBackupPath
	st.VolumeBackupPath = in.VolumeBackupPath
	st.ScheduleEnabled = in.ScheduleEnabled
	st.ScheduleCron = in.ScheduleCron
	st.MaxBackups = in.MaxBackups
	st.RetentionDays = in.RetentionDays
	st.Volumes = in.Volumes

	// Only replace the secret when a non-empty new value is supplied.
	if in.S3SecretKey != nil && *in.S3SecretKey != "" {
		enc, err := crypto.Encrypt(*in.S3SecretKey)
		if err != nil {
			return nil, err
		}
		st.S3SecretKeyEnc = enc
	}

	if err := s.settings.Upsert(st); err != nil {
		return nil, err
	}
	st.S3SecretSet = st.S3SecretKeyEnc != ""
	return st, nil
}

// s3Config returns the decrypted S3 config for the platform target, or nil when
// S3 is not enabled/configured. The Path is left empty; callers append the
// database/volume prefix.
func (s *Service) s3Config(st *models.PlatformBackupSettings) (*backup.S3Config, error) {
	if !st.S3Enabled || st.S3Bucket == "" {
		return nil, nil
	}
	secret, err := crypto.Decrypt(st.S3SecretKeyEnc)
	if err != nil {
		return nil, err
	}
	return &backup.S3Config{
		Endpoint:       st.S3Endpoint,
		Bucket:         st.S3Bucket,
		Region:         st.S3Region,
		AccessKey:      st.S3AccessKey,
		SecretKey:      secret,
		UseSSL:         st.S3UseSSL,
		ForcePathStyle: st.S3ForcePathStyle,
	}, nil
}

// S3Configured reports whether a usable S3 target is set (the UI uses it to gate
// volume backups, which have no local destination).
func (s *Service) S3Configured() bool {
	st, err := s.getSettings()
	if err != nil {
		return false
	}
	cfg, err := s.s3Config(st)
	return err == nil && cfg != nil
}
