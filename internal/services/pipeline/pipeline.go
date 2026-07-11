// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"strings"

	"github.com/miabi-io/miabi/internal/declarative"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/slug"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

var (
	ErrNotFound     = errors.New("pipeline not found")
	ErrRunNotFound  = errors.New("pipeline run not found")
	ErrNameTaken    = errors.New("a pipeline with this name already exists")
	ErrNameRequired = errors.New("name is required")
	ErrInvalidSpec  = errors.New("invalid pipeline spec")
	ErrDisabled     = errors.New("pipeline is disabled")
	ErrUnauthorized = errors.New("invalid webhook signature")
)

// Enqueuer hands a created run to the background worker. It is an interface so
// the pipeline service does not import the worker package (avoiding a cycle:
// the worker's runner imports this package).
type Enqueuer interface {
	EnqueuePipelineRun(runID, serverID uint) error
}

// Service manages pipeline definitions and triggers runs.
type Service struct {
	repo      *repositories.PipelineRepository
	enqueuer  Enqueuer
	scheduler Scheduler
}

func NewService(repo *repositories.PipelineRepository, enqueuer Enqueuer) *Service {
	return &Service{repo: repo, enqueuer: enqueuer}
}

// Input is the create/update payload for a pipeline definition. Name is the
// desired unique slug handle; DisplayName is the free-text label (falls back to
// Name when blank).
type Input struct {
	Name        string
	DisplayName string
	// ApplicationID is the deploy-target app. On update it is written only when
	// SetApplicationID is true (partial-update aware): a nil pointer then unbinds,
	// a non-nil binds; when SetApplicationID is false the binding is left as-is.
	ApplicationID    *uint
	SetApplicationID bool
	Spec             string
	// Enabled is a pointer so an update can leave it unchanged (nil). On create,
	// nil is treated as disabled (the zero value), matching the prior behavior.
	Enabled *bool
}

// Create validates the spec and stores a new pipeline definition.
func (s *Service) Create(workspaceID uint, in Input) (*models.PipelineDefinition, error) {
	name := slug.Make(in.Name, "")
	if name == "" {
		return nil, ErrNameRequired
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(in.Name)
	}
	if _, err := ParseSpec([]byte(in.Spec)); err != nil {
		return nil, errors.Join(ErrInvalidSpec, err)
	}
	if taken, _ := s.repo.ExistsByName(workspaceID, name); taken {
		return nil, ErrNameTaken
	}
	p := &models.PipelineDefinition{
		WorkspaceID: workspaceID, Name: name, DisplayName: displayName, ApplicationID: in.ApplicationID,
		Spec: in.Spec, Enabled: in.Enabled != nil && *in.Enabled, WebhookSecret: declarative.RandAlphaNum(40),
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	s.applySchedule(p)
	return p, nil
}

// Update mutates a pipeline definition.
func (s *Service) Update(workspaceID, id uint, in Input) (*models.PipelineDefinition, error) {
	p, err := s.Get(workspaceID, id)
	if err != nil {
		return nil, err
	}
	if in.Spec != "" {
		if _, err := ParseSpec([]byte(in.Spec)); err != nil {
			return nil, errors.Join(ErrInvalidSpec, err)
		}
		p.Spec = in.Spec
	}
	if name := slug.Make(in.Name, ""); name != "" && name != p.Name {
		if taken, _ := s.repo.ExistsByName(workspaceID, name); taken {
			return nil, ErrNameTaken
		}
		p.Name = name
	}
	if dn := strings.TrimSpace(in.DisplayName); dn != "" {
		p.DisplayName = dn
	}
	// Partial update: only touch the app binding / enabled flag when the caller
	// actually supplied them, so a spec-only update can't unbind or disable.
	if in.SetApplicationID {
		p.ApplicationID = in.ApplicationID
	}
	if in.Enabled != nil {
		p.Enabled = *in.Enabled
	}
	// Backfill a webhook secret for pipelines created before push triggers existed.
	if p.WebhookSecret == "" {
		p.WebhookSecret = declarative.RandAlphaNum(40)
	}
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	s.applySchedule(p)
	return p, nil
}

// Get loads a pipeline definition without last-run enrichment. Used by internal
// callers (Update/Delete/Trigger existence checks); client-facing reads that
// want the at-a-glance status should use GetWithLastRun.
func (s *Service) Get(workspaceID, id uint) (*models.PipelineDefinition, error) {
	p, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return p, nil
}

// GetWithLastRun loads a pipeline and attaches its most recent run summary, for
// client-facing reads (the runs page header, etc.).
func (s *Service) GetWithLastRun(workspaceID, id uint) (*models.PipelineDefinition, error) {
	p, err := s.Get(workspaceID, id)
	if err != nil {
		return nil, err
	}
	s.attachLastRuns(p)
	return p, nil
}

func (s *Service) List(workspaceID uint) ([]models.PipelineDefinition, error) {
	return s.repo.ListByWorkspace(workspaceID)
}

// ListPaged returns a page of pipeline definitions plus the total count, each
// enriched with its most recent run.
func (s *Service) ListPaged(workspaceID uint, limit, offset int) ([]models.PipelineDefinition, int64, error) {
	defs, total, err := s.repo.ListByWorkspacePaged(workspaceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	ptrs := make([]*models.PipelineDefinition, len(defs))
	for i := range defs {
		ptrs[i] = &defs[i]
	}
	s.attachLastRuns(ptrs...)
	return defs, total, nil
}

// attachLastRuns fills each definition's LastRun with its newest run, resolved
// in a single batch query. Best-effort: an enrichment failure leaves LastRun nil
// rather than failing the caller.
func (s *Service) attachLastRuns(defs ...*models.PipelineDefinition) {
	if len(defs) == 0 {
		return
	}
	ids := make([]uint, len(defs))
	for i, p := range defs {
		ids[i] = p.ID
	}
	latest, err := s.repo.LatestRunByPipeline(ids)
	if err != nil {
		return
	}
	for _, p := range defs {
		if run, ok := latest[p.ID]; ok {
			p.LastRun = run.Summary()
		}
	}
}

func (s *Service) Delete(workspaceID, id uint) error {
	if _, err := s.Get(workspaceID, id); err != nil {
		return err
	}
	if s.scheduler != nil {
		s.scheduler.Unschedule(id)
	}
	return s.repo.Delete(workspaceID, id)
}

// TriggerInput attributes and contextualizes a run.
type TriggerInput struct {
	Trigger       string // push | manual | schedule | upstream
	Commit        string
	CommitMessage string
	UserID        *uint
	APIKeyID      *uint
}

// Trigger creates a PipelineRun (with its step rows) and enqueues it. The run
// executes on the internal runner unless routed to a remote runner by labels.
func (s *Service) Trigger(workspaceID, pipelineID uint, in TriggerInput) (*models.PipelineRun, error) {
	p, err := s.Get(workspaceID, pipelineID)
	if err != nil {
		return nil, err
	}
	if !p.Enabled {
		return nil, ErrDisabled
	}
	spec, err := ParseSpec([]byte(p.Spec))
	if err != nil {
		return nil, errors.Join(ErrInvalidSpec, err)
	}
	number, err := s.repo.NextRunNumber(p.ID)
	if err != nil {
		return nil, err
	}
	run := &models.PipelineRun{
		WorkspaceID: workspaceID, PipelineID: p.ID, Number: number,
		Status: models.PipelineRunPending, Trigger: in.Trigger,
		Commit: in.Commit, CommitMessage: in.CommitMessage,
		TriggeredByUserID: in.UserID, TriggeredByKeyID: in.APIKeyID,
	}
	if err := s.repo.CreateRun(run); err != nil {
		return nil, err
	}
	for i, st := range spec.Steps {
		step := &models.PipelineStepRun{
			PipelineRunID: run.ID, Ordinal: i, Name: st.Name,
			Status: models.PipelineRunPending, Image: st.Image, Uses: st.Uses, Run: st.Run,
			ContinueOnError: st.ContinueOnError,
		}
		if err := s.repo.CreateStep(step); err != nil {
			return nil, err
		}
	}
	if s.enqueuer != nil {
		if err := s.enqueuer.EnqueuePipelineRun(run.ID, 0); err != nil {
			return nil, err
		}
	}
	return run, nil
}

func (s *Service) GetRun(workspaceID, id uint) (*models.PipelineRun, error) {
	run, err := s.repo.FindRun(workspaceID, id)
	if err != nil {
		return nil, ErrRunNotFound
	}
	return run, nil
}

func (s *Service) ListRuns(workspaceID, pipelineID uint, limit int) ([]models.PipelineRun, error) {
	return s.repo.ListRuns(workspaceID, pipelineID, limit)
}

// ListRunsPaged returns a page of a pipeline's runs plus the total count.
func (s *Service) ListRunsPaged(workspaceID, pipelineID uint, limit, offset int) ([]models.PipelineRun, int64, error) {
	return s.repo.ListRunsPaged(workspaceID, pipelineID, limit, offset)
}

// IDByUID resolves a pipeline's portable uid to its numeric id.
func (s *Service) IDByUID(uid string) (uint, error) { return s.repo.IDByUID(uid) }
