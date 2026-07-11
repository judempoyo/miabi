// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jkaninda/logger"
	"github.com/miabi-io/miabi/internal/services/volumebackup"
)

// VolumeBackupHandler runs volume backup tasks.
type VolumeBackupHandler struct {
	svc *volumebackup.Service
}

func NewVolumeBackupHandler(svc *volumebackup.Service) *VolumeBackupHandler {
	return &VolumeBackupHandler{svc: svc}
}

// ProcessTask implements asynq.Handler for the volume-backup task.
func (h *VolumeBackupHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p VolumeBackupPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return fmt.Errorf("bad volume backup payload: %w", err)
	}
	if err := h.svc.RunBackup(ctx, p.VolumeBackupID); err != nil {
		logger.Error("volume backup failed", "backup", p.VolumeBackupID, "error", err)
		return err
	}
	return nil
}
