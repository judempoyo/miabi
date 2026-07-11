// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UIDModel is an embeddable mixin giving a resource a stable, globally-unique,
// opaque identifier (UUIDv7) alongside its uint primary key. The PK stays the
// internal handle (foreign keys, Docker labels, performance); the uid is the
// portable, external one — the Terraform resource ID, the GitOps/backup
// cross-install reference, and the non-enumerable handle in public URLs.
//
// Embed it anonymously so `obj.UID` is the string directly and JSON exposes
// "uid". New rows get a UUIDv7 from BeforeCreate (time-ordered); the
// gen_random_uuid() default backfills pre-existing rows when AutoMigrate adds
// the column and is a safety net if the hook ever doesn't run.
type UIDModel struct {
	UID string `json:"uid" gorm:"type:uuid;uniqueIndex;not null;default:gen_random_uuid()"`
}

// BeforeCreate assigns a UUIDv7 when one was not supplied. A caller may preset
// UID (e.g. import/restore keeping an incoming identity); only an empty value is
// generated.
func (m *UIDModel) BeforeCreate(*gorm.DB) error {
	if m.UID == "" {
		v, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.UID = v.String()
	}
	return nil
}
