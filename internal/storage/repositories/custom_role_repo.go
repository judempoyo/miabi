// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// CustomRoleRepository accesses admin-defined roles (Enterprise; empty in CE).
type CustomRoleRepository struct{ db *gorm.DB }

func NewCustomRoleRepository(db *gorm.DB) *CustomRoleRepository {
	return &CustomRoleRepository{db: db}
}

func (r *CustomRoleRepository) Create(cr *models.CustomRole) error { return r.db.Create(cr).Error }
func (r *CustomRoleRepository) Update(cr *models.CustomRole) error { return r.db.Save(cr).Error }
func (r *CustomRoleRepository) Delete(id uint) error {
	return r.db.Delete(&models.CustomRole{}, id).Error
}

// ListByWorkspace returns the workspace's custom roles (plus org-wide templates,
// which have a nil workspace_id).
func (r *CustomRoleRepository) ListByWorkspace(workspaceID uint) ([]models.CustomRole, error) {
	var roles []models.CustomRole
	err := r.db.Where("workspace_id = ? OR workspace_id IS NULL", workspaceID).
		Order("name ASC").Find(&roles).Error
	return roles, err
}

// FindInWorkspace returns a role by id, scoped to the workspace (or an org-wide
// template). Enforces tenant isolation in the repo layer.
func (r *CustomRoleRepository) FindInWorkspace(workspaceID, id uint) (*models.CustomRole, error) {
	var cr models.CustomRole
	if err := r.db.Where("id = ? AND (workspace_id = ? OR workspace_id IS NULL)", id, workspaceID).
		First(&cr).Error; err != nil {
		return nil, err
	}
	return &cr, nil
}

func (r *CustomRoleRepository) FindByID(id uint) (*models.CustomRole, error) {
	var cr models.CustomRole
	if err := r.db.First(&cr, id).Error; err != nil {
		return nil, err
	}
	return &cr, nil
}

// CountMembersUsing returns how many members are assigned this custom role (the
// delete guard).
func (r *CustomRoleRepository) CountMembersUsing(roleID uint) (int64, error) {
	var n int64
	err := r.db.Model(&models.WorkspaceMember{}).Where("custom_role_id = ?", roleID).Count(&n).Error
	return n, err
}
