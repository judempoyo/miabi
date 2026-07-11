// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// DNSRecordRepository persists the ledger of DNS records Miabi manages.
type DNSRecordRepository struct {
	db *gorm.DB
}

func NewDNSRecordRepository(db *gorm.DB) *DNSRecordRepository {
	return &DNSRecordRepository{db: db}
}

// Upsert creates or updates the ledger entry keyed by (domain, name, type).
func (r *DNSRecordRepository) Upsert(rec *models.DNSRecord) error {
	var existing models.DNSRecord
	err := r.db.Where("domain_id = ? AND name = ? AND type = ?", rec.DomainID, rec.Name, rec.Type).
		First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(rec).Error
	}
	if err != nil {
		return err
	}
	existing.Value = rec.Value
	existing.Purpose = rec.Purpose
	existing.AppID = rec.AppID
	return r.db.Save(&existing).Error
}

// Find returns the ledger entry for a (domain, name, type), or ErrRecordNotFound.
func (r *DNSRecordRepository) Find(domainID uint, name, typ string) (*models.DNSRecord, error) {
	var rec models.DNSRecord
	if err := r.db.Where("domain_id = ? AND name = ? AND type = ?", domainID, name, typ).
		First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *DNSRecordRepository) ListByDomain(domainID uint) ([]models.DNSRecord, error) {
	var recs []models.DNSRecord
	err := r.db.Where("domain_id = ?", domainID).Order("name ASC").Find(&recs).Error
	return recs, err
}

func (r *DNSRecordRepository) ListByApp(appID uint) ([]models.DNSRecord, error) {
	var recs []models.DNSRecord
	err := r.db.Where("app_id = ?", appID).Find(&recs).Error
	return recs, err
}

// All returns every ledgered record (used by the reconcile cron).
func (r *DNSRecordRepository) All() ([]models.DNSRecord, error) {
	var recs []models.DNSRecord
	err := r.db.Order("domain_id ASC, name ASC").Find(&recs).Error
	return recs, err
}

func (r *DNSRecordRepository) Delete(id uint) error {
	return r.db.Delete(&models.DNSRecord{}, id).Error
}

// DeleteByDomain removes all ledger entries for a domain (used when a domain is
// deleted).
func (r *DNSRecordRepository) DeleteByDomain(domainID uint) error {
	return r.db.Where("domain_id = ?", domainID).Delete(&models.DNSRecord{}).Error
}
