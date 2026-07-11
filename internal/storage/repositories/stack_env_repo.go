// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type StackEnvVarRepository struct {
	db *gorm.DB
}

func NewStackEnvVarRepository(db *gorm.DB) *StackEnvVarRepository {
	return &StackEnvVarRepository{db: db}
}

func (r *StackEnvVarRepository) ListByStack(stackID uint) ([]models.StackEnvVar, error) {
	var vars []models.StackEnvVar
	err := r.db.Where("stack_id = ?", stackID).Order("key ASC").Find(&vars).Error
	return vars, err
}

func (r *StackEnvVarRepository) Upsert(v *models.StackEnvVar) error {
	var existing models.StackEnvVar
	err := r.db.Where("stack_id = ? AND key = ?", v.StackID, v.Key).First(&existing).Error
	if err == nil {
		existing.Value = v.Value
		existing.IsSecret = v.IsSecret
		return r.db.Save(&existing).Error
	}
	return r.db.Create(v).Error
}

func (r *StackEnvVarRepository) Delete(stackID uint, key string) error {
	return r.db.Where("stack_id = ? AND key = ?", stackID, key).Delete(&models.StackEnvVar{}).Error
}
