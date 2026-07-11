// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package platformbackup is the admin-only disaster-recovery feature for Miabi's
// own control plane: it backs up and restores the platform database and
// platform/system Docker volumes. It reuses the per-workspace backup primitives
// (the jkaninda/*-bkup one-shot images, backup.S3Env, the docker one-shot runner,
// BackupStatus) but draws its database connection from control-plane config —
// there is no managed Database row for Miabi's own DB — and runs on the manager
// node where the control plane and system volumes live. Artifact encryption is
// intentionally not handled here.
package platformbackup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/jkaninda/logger"
	"github.com/miabi-io/miabi/internal/docker"
	"github.com/miabi-io/miabi/internal/logstore"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/backup"
	"github.com/miabi-io/miabi/internal/services/platformimage"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

var (
	// ErrS3NotConfigured is returned when an operation needs an S3 target that is
	// not set up (volume backups have no local destination, like volume-bkup).
	ErrS3NotConfigured = errors.New("platform S3 backup settings are not configured")
	// ErrNoArtifact is returned when restoring/downloading a backup with no file.
	ErrNoArtifact = errors.New("platform backup has no artifact")
	// ErrDownloadRemote is returned when downloading a non-local backup.
	ErrDownloadRemote = errors.New("download is only available for local backups; fetch S3 backups from your bucket")
	// ErrUnknownSubject is returned for an unrecognized backup subject.
	ErrUnknownSubject = errors.New("unknown platform backup subject")

	// pg-bkup emits "<db>_YYYYMMDD_...sql.gz"; volume-bkup emits "<name>_...tar.gz".
	dbArtifactRe  = regexp.MustCompile(`[\w.\-]+\.sql\.gz`)
	volArtifactRe = regexp.MustCompile(`[\w.\-]+\.tar\.gz`)
)

const (
	dbMount     = "/backup"
	volumeMount = "/data"

	// platformBackupVolume is the fixed local volume holding platform DB backup
	// artifacts when the destination is "local".
	platformBackupVolume = "mb-platform-backups"

	defaultPgImage  = "jkaninda/pg-bkup:latest"
	defaultVolImage = "jkaninda/volume-bkup:latest"
)

// DBConn carries the resolved control-plane Postgres connection parameters the
// platform DB backup runner feeds to pg-bkup.
type DBConn struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// NodeDocker resolves the Docker client for a node id (0 = local manager node).
type NodeDocker interface {
	For(serverID uint) (docker.Client, error)
	LocalID() uint
}

// ImageResolver resolves a deployment-config catalog key to an image ref.
type ImageResolver interface {
	Ref(key string) string
}

// Enqueuer schedules a platform backup to run in the background worker. Satisfied
// by worker.Producer. When unset, backups run synchronously (tests / no-redis).
type Enqueuer interface {
	EnqueuePlatformBackup(backupID uint) error
}

// Service backs up and restores the platform's own database and volumes.
type Service struct {
	repo     *repositories.PlatformBackupRepository
	settings *repositories.PlatformBackupSettingsRepository
	clients  NodeDocker
	db       DBConn
	network  string // proxy network attached so pg-bkup can reach a managed DB by name
	images   ImageResolver
	enqueuer Enqueuer
	logs     *logstore.Store
}

// NewService builds the platform backup service. db is the control-plane DB
// connection; network is the proxy network the DB-backup container joins so it
// can reach a Compose/managed Postgres by service name.
func NewService(repo *repositories.PlatformBackupRepository, settings *repositories.PlatformBackupSettingsRepository, clients NodeDocker, db DBConn, network string) *Service {
	return &Service{repo: repo, settings: settings, clients: clients, db: db, network: network}
}

// SetImageResolver wires the deployment-config resolver for the backup tool images.
func (s *Service) SetImageResolver(r ImageResolver) { s.images = r }

// SetLogStore wires the shared execution-log store. When set, a platform-backup
// run's full output is externalized to the store on terminal state and the DB
// row keeps only a bounded tail + a reference. nil keeps DB-tail-only. Platform
// backups have no workspace; their objects live under an admin-only prefix.
func (s *Service) SetLogStore(store *logstore.Store) { s.logs = store }

// externalizeLog moves a terminal platform-backup's full output into the shared
// log store and trims the row to a bounded tail + a reference. No-op when the
// store is disabled or already externalized; on any error the full log stays in
// the DB tail.
func (s *Service) externalizeLog(b *models.PlatformBackup) {
	if !s.logs.Enabled() || b.LogRef != "" {
		return
	}
	ref := logstore.PlatformBackupRef(b.ID)
	res, err := s.logs.Externalize(ref, b.Logs)
	if err != nil {
		logger.Error("log store: externalize platform backup log failed", "platform_backup", b.ID, "error", err)
		return
	}
	if err := s.repo.SetLogMeta(b.ID, res.Ref, res.Tail, res.Bytes, res.Lines, res.Truncated); err != nil {
		logger.Error("log store: record platform backup log ref failed", "platform_backup", b.ID, "error", err)
		return
	}
	b.LogRef, b.Logs = res.Ref, res.Tail
	b.LogBytes, b.LogLines, b.LogTruncated = res.Bytes, res.Lines, res.Truncated
}

// SetEnqueuer wires the background worker producer.
func (s *Service) SetEnqueuer(e Enqueuer) { s.enqueuer = e }

func (s *Service) pgImage() string {
	if s.images != nil {
		if r := s.images.Ref(platformimage.KeyBackupPostgres); r != "" {
			return r
		}
	}
	return defaultPgImage
}

func (s *Service) volImage() string {
	if s.images != nil {
		if r := s.images.Ref(platformimage.KeyBackupVolume); r != "" {
			return r
		}
	}
	return defaultVolImage
}

// docker resolves the manager-node Docker client (the control plane + system
// volumes live on the local node).
func (s *Service) docker() (docker.Client, error) { return s.clients.For(0) }

// List returns all platform backups, newest first.
func (s *Service) List() ([]models.PlatformBackup, error) { return s.repo.List() }

// ListPaged returns a page of platform backups plus the total count.
func (s *Service) ListPaged(limit, offset int) ([]models.PlatformBackup, int64, error) {
	return s.repo.ListPaged(limit, offset)
}

// Get returns a single platform backup.
func (s *Service) Get(id uint) (*models.PlatformBackup, error) { return s.repo.FindByID(id) }

// Create records a pending platform backup and enqueues it for the background
// worker, returning the pending record immediately. With no enqueuer wired it
// runs synchronously. Volume backups require an S3 target (volume-bkup has no
// local destination).
func (s *Service) Create(ctx context.Context, subject models.PlatformBackupSubject, volumeName, trigger string) (*models.PlatformBackup, error) {
	st, err := s.getSettings()
	if err != nil {
		return nil, err
	}
	cfg, err := s.s3Config(st)
	if err != nil {
		return nil, err
	}
	dest := "local"
	if cfg != nil {
		dest = "s3"
	}
	switch subject {
	case models.PlatformBackupDatabase:
		// local or s3 — both supported by pg-bkup
	case models.PlatformBackupVolume:
		if volumeName == "" {
			return nil, errors.New("volume name is required for a volume backup")
		}
		if cfg == nil {
			return nil, ErrS3NotConfigured // volume-bkup is S3-only
		}
	default:
		return nil, ErrUnknownSubject
	}

	b := &models.PlatformBackup{
		Subject: subject, VolumeName: volumeName,
		Status: models.BackupPending, Trigger: trigger, Destination: dest,
	}
	if cfg != nil {
		b.S3Bucket = cfg.Bucket
		if subject == models.PlatformBackupDatabase {
			b.S3Path = st.DatabaseBackupPath
		} else {
			b.S3Path = st.VolumeBackupPath
		}
	}
	if err := s.repo.Create(b); err != nil {
		return nil, err
	}
	if s.enqueuer == nil {
		_ = s.RunBackup(ctx, b.ID)
		return s.repo.FindByID(b.ID)
	}
	if err := s.enqueuer.EnqueuePlatformBackup(b.ID); err != nil {
		return s.fail(b, fmt.Errorf("enqueue backup: %w", err)), nil
	}
	return b, nil
}

// RunBackup executes a pending platform backup (the worker entry point),
// dispatching by subject.
func (s *Service) RunBackup(ctx context.Context, backupID uint) error {
	b, err := s.repo.FindByID(backupID)
	if err != nil {
		return fmt.Errorf("platform backup %d not found: %w", backupID, err)
	}
	if b.Status == models.BackupCompleted || b.Status == models.BackupFailed {
		return nil // already processed
	}
	st, err := s.getSettings()
	if err != nil {
		s.fail(b, err)
		return nil
	}
	switch b.Subject {
	case models.PlatformBackupDatabase:
		return s.runDBBackup(ctx, b, st)
	case models.PlatformBackupVolume:
		return s.runVolumeBackup(ctx, b, st)
	default:
		s.fail(b, ErrUnknownSubject)
		return nil
	}
}

func (s *Service) runDBBackup(ctx context.Context, b *models.PlatformBackup, st *models.PlatformBackupSettings) error {
	dc, err := s.docker()
	if err != nil {
		s.fail(b, err)
		return nil
	}
	cfg, err := s.s3Config(st)
	if err != nil {
		s.fail(b, err)
		return nil
	}

	now := time.Now()
	b.Status = models.BackupRunning
	b.StartedAt = &now
	_ = s.repo.Update(b)

	env := s.dbEnv()
	cmd := []string{"backup", "-d", s.db.Name}
	var mounts map[string]string
	var nets []string

	if cfg != nil {
		env = append(env, backup.S3Env(cfg)...)
		cmd = []string{"backup", "--storage", "s3", "-d", s.db.Name}
		if st.DatabaseBackupPath != "" {
			cmd = append(cmd, "--path", st.DatabaseBackupPath)
		}
	} else {
		if _, err := dc.CreateVolume(ctx, platformBackupVolume, map[string]string{docker.LabelManaged: "true"}, 0); err != nil {
			s.fail(b, fmt.Errorf("create backup volume: %w", err))
			return nil
		}
		mounts = map[string]string{platformBackupVolume: dbMount}
	}
	if s.network != "" {
		if _, err := dc.EnsureNetwork(ctx, s.network); err == nil {
			nets = []string{s.network}
		}
	}

	image := s.pgImage()
	if err := dc.PullImage(ctx, image, nil); err != nil {
		s.fail(b, fmt.Errorf("pull backup image: %w", err))
		return nil
	}
	exit, out, err := dc.RunOneShot(ctx, docker.RunSpec{
		Name:     fmt.Sprintf("mb-platform-dbbkup-%d", b.ID),
		Image:    image,
		Env:      env,
		Cmd:      cmd,
		Networks: nets,
		Mounts:   mounts,
		Labels:   map[string]string{docker.LabelManaged: "true"},
	})
	b.Logs = out
	if err != nil || exit != 0 {
		s.fail(b, fmt.Errorf("backup exited with code %d: %w", exit, err))
		return nil
	}
	b.Filename = dbArtifactRe.FindString(out)
	fin := time.Now()
	b.Status = models.BackupCompleted
	b.FinishedAt = &fin
	if err := s.repo.Update(b); err != nil {
		return err
	}
	s.externalizeLog(b)
	logger.Info("platform database backup completed", "backup", b.ID, "destination", b.Destination, "file", b.Filename)
	return nil
}

func (s *Service) runVolumeBackup(ctx context.Context, b *models.PlatformBackup, st *models.PlatformBackupSettings) error {
	dc, err := s.docker()
	if err != nil {
		s.fail(b, err)
		return nil
	}
	cfg, err := s.s3Config(st)
	if err != nil {
		s.fail(b, err)
		return nil
	}
	if cfg == nil {
		s.fail(b, ErrS3NotConfigured)
		return nil
	}

	now := time.Now()
	b.Status = models.BackupRunning
	b.StartedAt = &now
	_ = s.repo.Update(b)

	image := s.volImage()
	if err := dc.PullImage(ctx, image, nil); err != nil {
		s.fail(b, fmt.Errorf("pull image: %w", err))
		return nil
	}
	exit, out, err := dc.RunOneShot(ctx, docker.RunSpec{
		Name:   fmt.Sprintf("mb-platform-volbkup-%d", b.ID),
		Image:  image,
		Env:    backup.S3Env(cfg),
		Cmd:    []string{"backup", "--storage", "s3", "--remote-path", st.VolumeBackupPath, "--name", volumeArchiveName(b.VolumeName)},
		Mounts: map[string]string{b.VolumeName: volumeMount},
		Labels: map[string]string{docker.LabelManaged: "true"},
	})
	b.Logs = out
	if err != nil || exit != 0 {
		s.fail(b, fmt.Errorf("volume backup exited with code %d: %w", exit, err))
		return nil
	}
	b.Filename = volArtifactRe.FindString(out)
	fin := time.Now()
	b.Status = models.BackupCompleted
	b.FinishedAt = &fin
	if err := s.repo.Update(b); err != nil {
		return err
	}
	s.externalizeLog(b)
	logger.Info("platform volume backup completed", "backup", b.ID, "volume", b.VolumeName, "file", b.Filename)
	return nil
}

// Restore restores a completed platform backup. It runs synchronously (the admin
// confirms a destructive, maintenance-mode operation and waits for the result).
// A DB restore overwrites the control-plane database in place; a volume restore
// overwrites the target volume.
func (s *Service) Restore(ctx context.Context, b *models.PlatformBackup) error {
	if b.Filename == "" {
		return ErrNoArtifact
	}
	st, err := s.getSettings()
	if err != nil {
		return err
	}
	switch b.Subject {
	case models.PlatformBackupDatabase:
		return s.restoreDB(ctx, b, st)
	case models.PlatformBackupVolume:
		return s.restoreVolume(ctx, b, st)
	default:
		return ErrUnknownSubject
	}
}

func (s *Service) restoreDB(ctx context.Context, b *models.PlatformBackup, st *models.PlatformBackupSettings) error {
	dc, err := s.docker()
	if err != nil {
		return err
	}
	env := s.dbEnv()
	cmd := []string{"restore", "-d", s.db.Name, "-f", b.Filename}
	var mounts map[string]string
	var nets []string

	if b.Destination == "s3" {
		cfg, err := s.s3Config(st)
		if err != nil {
			return err
		}
		if cfg == nil {
			return ErrS3NotConfigured
		}
		env = append(env, backup.S3Env(cfg)...)
		cmd = []string{"restore", "--storage", "s3", "-d", s.db.Name, "-f", b.Filename}
		if b.S3Path != "" {
			cmd = append(cmd, "--path", b.S3Path)
		}
	} else {
		mounts = map[string]string{platformBackupVolume: dbMount}
	}
	if s.network != "" {
		if _, err := dc.EnsureNetwork(ctx, s.network); err == nil {
			nets = []string{s.network}
		}
	}

	image := s.pgImage()
	if err := dc.PullImage(ctx, image, nil); err != nil {
		return fmt.Errorf("pull backup image: %w", err)
	}
	exit, out, err := dc.RunOneShot(ctx, docker.RunSpec{
		Name:     fmt.Sprintf("mb-platform-dbrestore-%d", b.ID),
		Image:    image,
		Env:      env,
		Cmd:      cmd,
		Networks: nets,
		Mounts:   mounts,
		Labels:   map[string]string{docker.LabelManaged: "true"},
	})
	if err != nil || exit != 0 {
		return fmt.Errorf("restore exited with code %d: %s", exit, out)
	}
	logger.Info("platform database restore completed", "backup", b.ID)
	return nil
}

func (s *Service) restoreVolume(ctx context.Context, b *models.PlatformBackup, st *models.PlatformBackupSettings) error {
	if b.VolumeName == "" {
		return errors.New("volume backup has no target volume")
	}
	cfg, err := s.s3Config(st)
	if err != nil {
		return err
	}
	if cfg == nil {
		return ErrS3NotConfigured
	}
	dc, err := s.docker()
	if err != nil {
		return err
	}
	image := s.volImage()
	if err := dc.PullImage(ctx, image, nil); err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	exit, out, err := dc.RunOneShot(ctx, docker.RunSpec{
		Name:   fmt.Sprintf("mb-platform-volrestore-%d", b.ID),
		Image:  image,
		Env:    backup.S3Env(cfg),
		Cmd:    []string{"restore", "--storage", "s3", "--remote-path", b.S3Path, "--file", b.Filename},
		Mounts: map[string]string{b.VolumeName: volumeMount},
		Labels: map[string]string{docker.LabelManaged: "true"},
	})
	if err != nil || exit != 0 {
		return fmt.Errorf("volume restore exited with code %d: %s", exit, out)
	}
	logger.Info("platform volume restore completed", "backup", b.ID, "volume", b.VolumeName)
	return nil
}

// Delete removes a backup record and, for local DB backups, its artifact file
// from the platform backup volume (best-effort). S3 artifacts are left in place.
func (s *Service) Delete(ctx context.Context, b *models.PlatformBackup) error {
	if b.Destination == "local" && b.Subject == models.PlatformBackupDatabase && b.Filename != "" {
		if dc, err := s.docker(); err == nil {
			image := s.pgImage()
			if _, out, err := dc.RunOneShot(ctx, docker.RunSpec{
				Name:       fmt.Sprintf("mb-platform-bkup-rm-%d", b.ID),
				Image:      image,
				Entrypoint: []string{"/bin/sh", "-c"},
				Cmd:        []string{"rm -f " + dbMount + "/" + b.Filename},
				Mounts:     map[string]string{platformBackupVolume: dbMount},
			}); err != nil {
				logger.Error("remove platform backup artifact", "backup", b.ID, "error", err, "out", out)
			}
		}
	}
	return s.repo.Delete(b.ID)
}

// Download streams a local DB backup artifact. The returned reader must be
// closed by the caller. S3 backups are not downloadable through Miabi.
func (s *Service) Download(ctx context.Context, b *models.PlatformBackup) (io.ReadCloser, int64, string, error) {
	if b.Destination != "local" || b.Subject != models.PlatformBackupDatabase {
		return nil, 0, "", ErrDownloadRemote
	}
	if b.Filename == "" {
		return nil, 0, "", ErrNoArtifact
	}
	dc, err := s.docker()
	if err != nil {
		return nil, 0, "", err
	}
	rc, size, err := dc.CopyFileFromVolume(ctx, platformBackupVolume, s.pgImage(), b.Filename)
	if err != nil {
		return nil, 0, "", err
	}
	return rc, size, b.Filename, nil
}

// Prune enforces the retention policy on platform backups: keep at most
// maxBackups most-recent, and delete any older than retentionDays. A zero bound
// is ignored. Returns the number removed.
func (s *Service) Prune(ctx context.Context, maxBackups, retentionDays int) (int, error) {
	if maxBackups <= 0 && retentionDays <= 0 {
		return 0, nil
	}
	backups, err := s.repo.List() // newest-first
	if err != nil {
		return 0, err
	}
	var cutoff time.Time
	if retentionDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -retentionDays)
	}
	removed := 0
	for i := range backups {
		b := &backups[i]
		overCount := maxBackups > 0 && i >= maxBackups
		tooOld := retentionDays > 0 && b.CreatedAt.Before(cutoff)
		if overCount || tooOld {
			if err := s.Delete(ctx, b); err != nil {
				logger.Error("prune platform backup", "backup", b.ID, "error", err)
				continue
			}
			removed++
		}
	}
	if removed > 0 {
		logger.Info("pruned platform backups", "removed", removed)
	}
	return removed, nil
}

// RunScheduled is the cron entry point: it backs up the database plus every
// selected platform volume, then prunes per retention. Errors on individual
// volumes are logged but do not abort the run.
func (s *Service) RunScheduled(ctx context.Context) error {
	st, err := s.getSettings()
	if err != nil {
		return err
	}
	if _, err := s.Create(ctx, models.PlatformBackupDatabase, "", "scheduled"); err != nil {
		logger.Error("scheduled platform database backup failed", "error", err)
	}
	for _, v := range st.Volumes {
		if _, err := s.Create(ctx, models.PlatformBackupVolume, v, "scheduled"); err != nil {
			logger.Error("scheduled platform volume backup failed", "volume", v, "error", err)
		}
	}
	if st.MaxBackups > 0 || st.RetentionDays > 0 {
		_, _ = s.Prune(ctx, st.MaxBackups, st.RetentionDays)
	}
	return nil
}

// PlatformVolume is a candidate platform/system volume the admin may include in
// backups. Role is the io.miabi.role label value when the volume is infra.
type PlatformVolume struct {
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

// DiscoverVolumes lists candidate platform/system volumes on the manager node:
// every Miabi-managed volume except the per-workspace and platform backup
// volumes (which must never back themselves up).
func (s *Service) DiscoverVolumes(ctx context.Context) ([]PlatformVolume, error) {
	dc, err := s.docker()
	if err != nil {
		return nil, err
	}
	vols, err := dc.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PlatformVolume, 0, len(vols))
	for _, v := range vols {
		if v.Name == platformBackupVolume || strings.HasPrefix(v.Name, "mb-backups-") {
			continue // exclude backup volumes
		}
		// Limit to Miabi-managed/named volumes; ignore unrelated host volumes.
		if !docker.IsManaged(v.Labels) && !strings.HasPrefix(v.Name, "mb-") {
			continue
		}
		role, _ := docker.LabelValue(v.Labels, docker.LabelRole)
		out = append(out, PlatformVolume{Name: v.Name, Role: role})
	}
	return out, nil
}

func (s *Service) dbEnv() []string {
	return []string{
		"DB_HOST=" + s.db.Host,
		fmt.Sprintf("DB_PORT=%d", s.db.Port),
		"DB_NAME=" + s.db.Name,
		"DB_USERNAME=" + s.db.User,
		"DB_PASSWORD=" + s.db.Password,
	}
}

func (s *Service) fail(b *models.PlatformBackup, cause error) *models.PlatformBackup {
	fin := time.Now()
	b.Status = models.BackupFailed
	b.Error = cause.Error()
	b.FinishedAt = &fin
	_ = s.repo.Update(b)
	s.externalizeLog(b)
	logger.Error("platform backup failed", "backup", b.ID, "subject", b.Subject, "error", cause)
	return b
}

// volumeArchiveName sanitizes a docker volume name into a volume-bkup --name
// (the archive base name): keep alphanumerics, dash, underscore.
func volumeArchiveName(volume string) string {
	var b strings.Builder
	for _, r := range volume {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := b.String()
	if name == "" {
		name = "platform-volume"
	}
	return name
}
