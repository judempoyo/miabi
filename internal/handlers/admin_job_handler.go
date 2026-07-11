// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"sort"

	"github.com/jkaninda/okapi"
	cronpkg "github.com/miabi-io/miabi/internal/cron"
)

// AdminJobHandler reports the status of scheduled background jobs.
type AdminJobHandler struct {
	cron *cronpkg.Manager
}

func NewAdminJobHandler(cron *cronpkg.Manager) *AdminJobHandler {
	return &AdminJobHandler{cron: cron}
}

// List returns the scheduled jobs as a stable, paginated page. The cron snapshot
// comes from a map (unordered), so it is sorted by kind, then name, then id to
// give pagination a deterministic order across requests.
func (h *AdminJobHandler) List(c *okapi.Context) error {
	page, size, offset := normalizePageParams(queryInt(c, "page", 0), queryInt(c, "size", 20))
	if h.cron == nil {
		return paginated(c, []cronpkg.JobStatus{}, 0, page, size)
	}
	jobs := h.cron.Snapshot()
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].Kind != jobs[j].Kind {
			return jobs[i].Kind < jobs[j].Kind
		}
		if jobs[i].Name != jobs[j].Name {
			return jobs[i].Name < jobs[j].Name
		}
		return jobs[i].ID < jobs[j].ID
	})
	total := int64(len(jobs))
	if offset > len(jobs) {
		offset = len(jobs)
	}
	end := offset + size
	if end > len(jobs) {
		end = len(jobs)
	}
	return paginated(c, jobs[offset:end], total, page, size)
}

// JobStats is the scheduled-jobs dashboard summary computed over every job (not
// just the current page).
type JobStats struct {
	Total   int            `json:"total"`
	Running int            `json:"running"`
	Failed  int            `json:"failed"`
	OK      int            `json:"ok"`
	ByKind  map[string]int `json:"by_kind"`
}

// Stats returns aggregate counts for the jobs dashboard. A job is Failed when its
// last run errored and it is not currently running, Running while in flight, and
// OK otherwise — so the three are disjoint and sum to Total.
func (h *AdminJobHandler) Stats(c *okapi.Context) error {
	stats := JobStats{ByKind: map[string]int{}}
	if h.cron == nil {
		return ok(c, stats)
	}
	for _, j := range h.cron.Snapshot() {
		stats.Total++
		stats.ByKind[j.Kind]++
		switch {
		case j.Running:
			stats.Running++
		case j.LastError != "":
			stats.Failed++
		default:
			stats.OK++
		}
	}
	return ok(c, stats)
}
