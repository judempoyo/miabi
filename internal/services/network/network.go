// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package network manages Docker networks owned by workspaces. The platform
// owns the full lifecycle: each workspace gets a default network on creation,
// and the Docker network is created/removed alongside the database record.
package network

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jkaninda/logger"
	"github.com/miabi-io/miabi/internal/docker"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/netalloc"
	"github.com/miabi-io/miabi/internal/services/quota"
	"github.com/miabi-io/miabi/internal/slug"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

var (
	ErrNameRequired  = errors.New("network name is required")
	ErrNameTaken     = errors.New("a network with this name already exists")
	ErrNotFound      = errors.New("network not found")
	ErrInUse         = errors.New("network is attached to one or more applications")
	ErrIsDefault     = errors.New("the default network cannot be deleted")
	ErrInvalidDriver = errors.New("unsupported network driver")
)

type Service struct {
	repo   *repositories.NetworkRepository
	docker docker.Client
	quota  *quota.Service
	alloc  *netalloc.Service
}

func NewService(repo *repositories.NetworkRepository, dockerClient docker.Client) *Service {
	return &Service{repo: repo, docker: dockerClient}
}

// SetQuota wires the plan/quota enforcer (nil-safe; nil skips checks).
func (s *Service) SetQuota(q *quota.Service) { s.quota = q }

// SetAllocator wires the subnet allocator so managed networks get a Miabi-carved
// subnet instead of Docker's default pool (nil-safe; nil = Docker default pool).
func (s *Service) SetAllocator(a *netalloc.Service) { s.alloc = a }

type Input struct {
	// Name is the desired unique slug handle; it is normalized to canonical slug
	// form. DisplayName is the free-text label (falls back to Name when blank).
	Name        string
	DisplayName string
	Driver      string
	Internal    bool
}

func (s *Service) Create(ctx context.Context, workspaceID uint, in Input) (*models.Network, error) {
	return s.create(ctx, workspaceID, in, false)
}

// EnsureDefault creates the workspace's default network if it has none.
func (s *Service) EnsureDefault(ctx context.Context, workspaceID uint) (*models.Network, error) {
	nets, err := s.repo.ListByWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	for i := range nets {
		if nets[i].IsDefault {
			return &nets[i], nil
		}
	}
	return s.create(ctx, workspaceID, Input{Name: "default"}, true)
}

func (s *Service) create(ctx context.Context, workspaceID uint, in Input, isDefault bool) (*models.Network, error) {
	name := slug.Make(in.Name, "")
	if name == "" {
		return nil, ErrNameRequired
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(in.Name)
	}
	taken, err := s.repo.ExistsByName(workspaceID, name)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrNameTaken
	}
	// The auto-provisioned default network doesn't consume the workspace quota.
	if !isDefault && s.quota.Enabled() {
		n, _ := s.repo.CountByWorkspace(workspaceID)
		if err := s.quota.CheckCreate(workspaceID, quota.ResourceNetworks, int(n)); err != nil {
			return nil, err
		}
	}
	driver := in.Driver
	if driver == "" {
		driver = "bridge"
	}
	switch driver {
	case "bridge", "overlay", "macvlan", "ipvlan":
	default:
		return nil, fmt.Errorf("%w: unsupported network driver %q", ErrInvalidDriver, driver)
	}
	dockerName := fmt.Sprintf("mb-ws%d-%s", workspaceID, randHex(6))
	dockerCreated := false
	if err := s.provisionDockerNetwork(ctx, dockerName, driver, in.Internal); err != nil {
		// The platform-managed default network must always have a record, even when
		// the Docker daemon is unavailable at this instant (e.g. during early
		// bootstrap or in a dev environment). It is (re)ensured on each node at
		// deploy time — see the worker's syncNetworks — so persist the record and
		// let deployment reconcile the Docker side. A user-created network, by
		// contrast, is expected to exist now, so surface the failure.
		if !isDefault {
			return nil, fmt.Errorf("create docker network: %w", err)
		}
		logger.Warn("default network: deferring Docker creation to deploy time",
			"workspace", workspaceID, "docker_name", dockerName, "error", err)
	} else {
		dockerCreated = true
	}
	net := &models.Network{
		WorkspaceID: workspaceID, Name: name, DisplayName: displayName, DockerName: dockerName,
		Driver: driver, Internal: in.Internal, IsDefault: isDefault,
	}
	if err := s.repo.Create(net); err != nil {
		if dockerCreated {
			_ = s.docker.RemoveNetwork(ctx, dockerName) // roll back the docker side
			if s.alloc != nil {
				_ = s.alloc.Release(dockerName) // and free the subnet
			}
		}
		return nil, err
	}
	return net, nil
}

// provisionDockerNetwork creates the Docker network, preferring a Miabi-allocated
// subnet (via the allocator) over Docker's default address pool.
func (s *Service) provisionDockerNetwork(ctx context.Context, dockerName, driver string, internal bool) error {
	spec := docker.NetworkSpec{Name: dockerName, Driver: driver, Internal: internal}
	if s.alloc != nil {
		_, _, err := s.alloc.EnsureManaged(ctx, s.docker, spec, 0, models.NetAllocKindWorkspace)
		return err
	}
	_, err := s.docker.CreateNetworkSpec(ctx, spec)
	return err
}

func (s *Service) Get(workspaceID, id uint) (*models.Network, error) {
	n, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return n, nil
}

func (s *Service) List(workspaceID uint) ([]models.Network, error) {
	return s.repo.ListByWorkspace(workspaceID)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id uint) error {
	n, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return ErrNotFound
	}
	if n.IsDefault {
		return ErrIsDefault
	}
	count, err := s.repo.CountAppsUsing(n.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrInUse
	}
	if err := s.docker.RemoveNetwork(ctx, n.DockerName); err != nil {
		return fmt.Errorf("remove docker network: %w", err)
	}
	if s.alloc != nil {
		_ = s.alloc.Release(n.DockerName) // return the subnet to the pool
	}
	return s.repo.Delete(n.ID)
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "0000000000"[:n*2]
	}
	return hex.EncodeToString(b)
}
