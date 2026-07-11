// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package routes

import (
	"fmt"

	cronpkg "github.com/miabi-io/miabi/internal/cron"
	"github.com/miabi-io/miabi/internal/services/pipeline"
)

// pipelineCronScheduler adapts the generic cron Manager to the pipeline
// service's Scheduler port: each scheduled pipeline's `on.schedule` cron fires a
// run via TriggerScheduled. Keyed "pipeline:<id>" so it shares the one cron loop.
type pipelineCronScheduler struct {
	cron *cronpkg.Manager
	svc  *pipeline.Service
}

func (p pipelineCronScheduler) Schedule(pipelineID uint, cronExpr string) error {
	return p.cron.RegisterTask("pipeline", pipelineID, fmt.Sprintf("Pipeline #%d schedule", pipelineID), cronExpr, func() error {
		_, err := p.svc.TriggerScheduled(pipelineID)
		return err
	})
}

func (p pipelineCronScheduler) Unschedule(pipelineID uint) {
	p.cron.UnregisterTask("pipeline", pipelineID)
}
