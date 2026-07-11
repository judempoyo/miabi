// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package backupsettings manages a workspace's shared S3 backup target: the
// single bucket + credentials and the database/volume path prefixes that both
// database and volume backups draw from. The S3 secret is encrypted at rest.
package backupsettings

import (
	"errors"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/backup"
	"github.com/miabi-io/miabi/internal/services/crypto"
	"github.com/miabi-io/miabi/internal/storage/repositories"
	"gorm.io/gorm"
)

type Service struct {
	repo *repositories.WorkspaceBackupSettingsRepository
}

func NewService(repo *repositories.WorkspaceBackupSettingsRepository) *Service {
	return &Service{repo: repo}
}

// Get returns the workspace's settings, or an empty (unsaved) record when none
// exist yet. The secret is never included; S3SecretSet reports its presence.
func (s *Service) Get(workspaceID uint) (*models.WorkspaceBackupSettings, error) {
	st, err := s.repo.FindByWorkspace(workspaceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.WorkspaceBackupSettings{WorkspaceID: workspaceID}, nil
	}
	if err != nil {
		return nil, err
	}
	st.S3SecretSet = st.S3SecretKeyEnc != ""
	return st, nil
}

// SaveInput carries the desired settings. S3SecretKey is nil/empty to keep the
// stored secret unchanged (so the UI never needs to round-trip it).
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
}

// Save upserts the workspace's settings, encrypting the secret when a new one is
// supplied and preserving the existing secret otherwise.
func (s *Service) Save(workspaceID uint, in SaveInput) (*models.WorkspaceBackupSettings, error) {
	st, err := s.repo.FindByWorkspace(workspaceID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if st == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		st = &models.WorkspaceBackupSettings{WorkspaceID: workspaceID}
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

	// Only replace the secret when a non-empty new value is supplied.
	if in.S3SecretKey != nil && *in.S3SecretKey != "" {
		enc, err := crypto.EncryptWS(workspaceID, *in.S3SecretKey)
		if err != nil {
			return nil, err
		}
		st.S3SecretKeyEnc = enc
	}

	if err := s.repo.Upsert(st); err != nil {
		return nil, err
	}
	st.S3SecretSet = st.S3SecretKeyEnc != ""
	return st, nil
}

// S3ConfigFor returns the decrypted S3 config for a workspace, or nil when S3 is
// not enabled/configured. The Path field is left empty: callers append the
// database or volume prefix. Consumed by database + volume backups.
func (s *Service) S3ConfigFor(workspaceID uint) (*backup.S3Config, error) {
	st, err := s.repo.FindByWorkspace(workspaceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
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

// VolumeBackupTarget returns the workspace's S3 config plus the volume backup
// path prefix, or (nil, "", nil) when S3 is not enabled/configured. Satisfies
// the volumebackup.S3Provider interface.
func (s *Service) VolumeBackupTarget(workspaceID uint) (*backup.S3Config, string, error) {
	cfg, err := s.S3ConfigFor(workspaceID)
	if err != nil || cfg == nil {
		return nil, "", err
	}
	st, err := s.repo.FindByWorkspace(workspaceID)
	if err != nil {
		return nil, "", err
	}
	return cfg, st.VolumeBackupPath, nil
}

// DatabaseBackupTarget returns the workspace's S3 config plus the database
// backup path prefix, or (nil, "", nil) when S3 is not enabled/configured.
// Database backups use this as their destination when configured.
func (s *Service) DatabaseBackupTarget(workspaceID uint) (*backup.S3Config, string, error) {
	cfg, err := s.S3ConfigFor(workspaceID)
	if err != nil || cfg == nil {
		return nil, "", err
	}
	st, err := s.repo.FindByWorkspace(workspaceID)
	if err != nil {
		return nil, "", err
	}
	return cfg, st.DatabaseBackupPath, nil
}
