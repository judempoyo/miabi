// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package registry manages stored container-registry credentials used to pull
// private images at deploy time. Secrets are encrypted at rest.
package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/crypto"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

var (
	ErrNameRequired   = errors.New("registry name is required")
	ErrSecretRequired = errors.New("registry secret is required")
	ErrNameTaken      = errors.New("a registry with this name already exists")
	ErrNotFound       = errors.New("registry not found")
)

// DefaultServer is the implicit registry when none is given (Docker Hub).
const DefaultServer = "registry-1.docker.io"

type Service struct {
	repo *repositories.RegistryRepository
}

func NewService(repo *repositories.RegistryRepository) *Service { return &Service{repo: repo} }

// Input describes a registry credential to create or update.
type Input struct {
	Name     string
	Server   string
	Username string
	Secret   string // plaintext; encrypted before storage
}

func (s *Service) Create(workspaceID uint, in Input) (*models.Registry, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, ErrNameRequired
	}
	if strings.TrimSpace(in.Secret) == "" {
		return nil, ErrSecretRequired
	}
	taken, err := s.repo.ExistsByName(workspaceID, in.Name)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrNameTaken
	}
	enc, err := crypto.EncryptWS(workspaceID, in.Secret)
	if err != nil {
		return nil, err
	}
	reg := &models.Registry{
		WorkspaceID: workspaceID,
		Name:        in.Name,
		Server:      normalizeServer(in.Server),
		Username:    in.Username,
		Secret:      enc,
	}
	if err := s.repo.Create(reg); err != nil {
		return nil, err
	}
	return strip(reg), nil
}

func (s *Service) Update(workspaceID, id uint, in Input) (*models.Registry, error) {
	reg, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if in.Name != "" {
		reg.Name = in.Name
	}
	reg.Server = normalizeServer(in.Server)
	reg.Username = in.Username
	// Only rotate the secret when a new value is supplied.
	if strings.TrimSpace(in.Secret) != "" {
		enc, err := crypto.EncryptWS(workspaceID, in.Secret)
		if err != nil {
			return nil, err
		}
		reg.Secret = enc
	}
	if err := s.repo.Update(reg); err != nil {
		return nil, err
	}
	return strip(reg), nil
}

func (s *Service) Get(workspaceID, id uint) (*models.Registry, error) {
	reg, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return strip(reg), nil
}

func (s *Service) List(workspaceID uint) ([]models.Registry, error) {
	regs, err := s.repo.ListByWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	for i := range regs {
		strip(&regs[i])
	}
	return regs, nil
}

func (s *Service) Delete(workspaceID, id uint) error {
	reg, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(reg.ID)
}

// TestConnection verifies the stored credential can authenticate to the
// registry, following the Docker Registry v2 token-auth flow.
func (s *Service) TestConnection(ctx context.Context, workspaceID, id uint) error {
	reg, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return ErrNotFound
	}
	secret, err := crypto.Decrypt(reg.Secret)
	if err != nil {
		return fmt.Errorf("decrypt secret: %w", err)
	}
	return checkRegistryAuth(ctx, reg.Server, reg.Username, secret)
}

// strip clears the ciphertext and flags secret presence for safe responses.
func strip(reg *models.Registry) *models.Registry {
	reg.HasSecret = reg.Secret != ""
	reg.Secret = ""
	return reg
}

// normalizeServer trims a registry host to its bare authority, defaulting to
// Docker Hub when empty.
func normalizeServer(server string) string {
	server = strings.TrimSpace(server)
	if server == "" {
		return DefaultServer
	}
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")
	return strings.TrimSuffix(server, "/")
}

// checkRegistryAuth probes the registry's /v2/ endpoint with basic auth and, if
// the registry uses bearer-token auth, completes a token request against the
// advertised realm.
func checkRegistryAuth(ctx context.Context, server, username, password string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	base := "https://" + normalizeServer(server)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v2/", nil)
	if err != nil {
		return err
	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect to registry: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	// Bearer-token flow: parse the realm/service from WWW-Authenticate and
	// exchange basic creds for a token.
	challenge := resp.Header.Get("Www-Authenticate")
	if !strings.HasPrefix(strings.ToLower(challenge), "bearer ") {
		return fmt.Errorf("authentication failed")
	}
	params := parseBearerChallenge(challenge)
	realm := params["realm"]
	if realm == "" {
		return fmt.Errorf("authentication failed")
	}
	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodGet, realm, nil)
	if err != nil {
		return err
	}
	q := tokenReq.URL.Query()
	if svc := params["service"]; svc != "" {
		q.Set("service", svc)
	}
	tokenReq.URL.RawQuery = q.Encode()
	if username != "" {
		tokenReq.SetBasicAuth(username, password)
	}
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return fmt.Errorf("registry token request: %w", err)
	}
	defer func() { _ = tokenResp.Body.Close() }()
	if tokenResp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed (status %d)", tokenResp.StatusCode)
	}
	return nil
}

// parseBearerChallenge extracts key="value" pairs from a WWW-Authenticate header.
func parseBearerChallenge(challenge string) map[string]string {
	out := map[string]string{}
	rest := strings.TrimSpace(challenge[len("Bearer "):])
	for _, part := range strings.Split(rest, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
	}
	return out
}
