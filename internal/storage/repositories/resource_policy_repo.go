// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

// ResourcePolicyRepository accesses per-resource permission grants (Enterprise;
// empty in CE).
type ResourcePolicyRepository struct{ db *gorm.DB }

func NewResourcePolicyRepository(db *gorm.DB) *ResourcePolicyRepository {
	return &ResourcePolicyRepository{db: db}
}

// Find returns the grant for (workspace, user, resource), or gorm.ErrRecordNotFound.
func (r *ResourcePolicyRepository) Find(workspaceID, userID uint, resourceType string, resourceID uint) (*models.ResourcePolicy, error) {
	var p models.ResourcePolicy
	err := r.db.Where("workspace_id = ? AND user_id = ? AND resource_type = ? AND resource_id = ?",
		workspaceID, userID, resourceType, resourceID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// HasPermission reports whether the user holds a permission on a specific
// resource via a policy. Used on the hot path by RequireResourcePermission.
func (r *ResourcePolicyRepository) HasPermission(workspaceID, userID uint, resourceType string, resourceID uint, perm models.Permission) bool {
	p, err := r.Find(workspaceID, userID, resourceType, resourceID)
	if err != nil {
		return false
	}
	return p.Grants(perm)
}

// Upsert creates or replaces the grant for (workspace, user, resource).
func (r *ResourcePolicyRepository) Upsert(p *models.ResourcePolicy) error {
	existing, err := r.Find(p.WorkspaceID, p.UserID, p.ResourceType, p.ResourceID)
	if err == nil {
		existing.Permissions = p.Permissions
		return r.db.Save(existing).Error
	}
	return r.db.Create(p).Error
}

// ListByResource returns all grants on a resource (with the granted users).
func (r *ResourcePolicyRepository) ListByResource(workspaceID uint, resourceType string, resourceID uint) ([]models.ResourcePolicy, error) {
	var policies []models.ResourcePolicy
	err := r.db.Preload("User").
		Where("workspace_id = ? AND resource_type = ? AND resource_id = ?", workspaceID, resourceType, resourceID).
		Order("created_at ASC").Find(&policies).Error
	return policies, err
}

// DeleteForUser revokes a user's grant on a resource.
func (r *ResourcePolicyRepository) DeleteForUser(workspaceID, userID uint, resourceType string, resourceID uint) error {
	return r.db.Where("workspace_id = ? AND user_id = ? AND resource_type = ? AND resource_id = ?",
		workspaceID, userID, resourceType, resourceID).Delete(&models.ResourcePolicy{}).Error
}

// DeleteForResource removes every grant on a resource (cascade on resource delete).
func (r *ResourcePolicyRepository) DeleteForResource(workspaceID uint, resourceType string, resourceID uint) error {
	return r.db.Where("workspace_id = ? AND resource_type = ? AND resource_id = ?",
		workspaceID, resourceType, resourceID).Delete(&models.ResourcePolicy{}).Error
}
