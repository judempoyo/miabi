// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/jkaninda/logger"
	"github.com/miabi-io/miabi/internal/services/platformbackup"
)

// PlatformBackupHandler runs platform (control-plane) backup tasks.
type PlatformBackupHandler struct {
	svc *platformbackup.Service
}

func NewPlatformBackupHandler(svc *platformbackup.Service) *PlatformBackupHandler {
	return &PlatformBackupHandler{svc: svc}
}

// ProcessTask implements asynq.Handler for the platform-backup task.
func (h *PlatformBackupHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var p PlatformBackupPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return fmt.Errorf("bad platform backup payload: %w", err)
	}
	if err := h.svc.RunBackup(ctx, p.PlatformBackupID); err != nil {
		logger.Error("platform backup failed", "backup", p.PlatformBackupID, "error", err)
		return err
	}
	return nil
}
