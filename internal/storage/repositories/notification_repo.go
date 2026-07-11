// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package repositories

import (
	"github.com/miabi-io/miabi/internal/models"
	"gorm.io/gorm"
)

type NotificationChannelRepository struct {
	db *gorm.DB
}

func NewNotificationChannelRepository(db *gorm.DB) *NotificationChannelRepository {
	return &NotificationChannelRepository{db: db}
}

func (r *NotificationChannelRepository) Create(ch *models.NotificationChannel) error {
	return r.db.Create(ch).Error
}
func (r *NotificationChannelRepository) Update(ch *models.NotificationChannel) error {
	return r.db.Save(ch).Error
}
func (r *NotificationChannelRepository) Delete(id uint) error {
	return r.db.Delete(&models.NotificationChannel{}, id).Error
}

// FindByID returns a channel by primary key, regardless of workspace. Used by
// the channel-send worker, which already trusts the enqueued id.
func (r *NotificationChannelRepository) FindByID(id uint) (*models.NotificationChannel, error) {
	var ch models.NotificationChannel
	if err := r.db.First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *NotificationChannelRepository) FindInWorkspace(workspaceID, id uint) (*models.NotificationChannel, error) {
	var ch models.NotificationChannel
	if err := r.db.Where("id = ? AND workspace_id = ?", id, workspaceID).First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *NotificationChannelRepository) ListByWorkspace(workspaceID uint) ([]models.NotificationChannel, error) {
	var chs []models.NotificationChannel
	err := r.db.Where("workspace_id = ?", workspaceID).Order("created_at DESC").Find(&chs).Error
	return chs, err
}

// ListEnabledForEvent returns enabled channels in the workspace that subscribe
// to the given event (membership filtered in Go; see WebhookRepository).
func (r *NotificationChannelRepository) ListEnabledForEvent(workspaceID uint, event string) ([]models.NotificationChannel, error) {
	var chs []models.NotificationChannel
	if err := r.db.Where("workspace_id = ? AND enabled = ?", workspaceID, true).Find(&chs).Error; err != nil {
		return nil, err
	}
	out := chs[:0]
	for _, ch := range chs {
		if containsString(ch.Events, event) {
			out = append(out, ch)
		}
	}
	return out, nil
}
