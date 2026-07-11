// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package gitrepo manages stored Git credentials used to clone private
// repositories at build time. Secrets (tokens or SSH keys) are encrypted at
// rest. The package also builds the go-git auth method shared by the test-
// connection check and the deploy worker.
package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/crypto"
	"github.com/miabi-io/miabi/internal/slug"
	"github.com/miabi-io/miabi/internal/storage/repositories"
)

var (
	ErrNameRequired   = errors.New("git repository name is required")
	ErrURLRequired    = errors.New("git repository URL is required")
	ErrSecretRequired = errors.New("git credential secret is required")
	ErrNameTaken      = errors.New("a git repository with this name already exists")
	ErrNotFound       = errors.New("git repository not found")
)

type Service struct {
	repo *repositories.GitRepoRepository
}

func NewService(repo *repositories.GitRepoRepository) *Service { return &Service{repo: repo} }

// Input describes a git credential to create or update. Name is the desired
// unique slug handle; DisplayName is the free-text label (falls back to Name).
type Input struct {
	Name        string
	DisplayName string
	URL         string
	AuthType    models.GitAuthType
	Username    string
	Secret      string // plaintext token or SSH private key; encrypted before storage
}

func (s *Service) Create(workspaceID uint, in Input) (*models.GitRepository, error) {
	name := slug.Make(in.Name, "")
	if name == "" {
		return nil, ErrNameRequired
	}
	displayName := strings.TrimSpace(in.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(in.Name)
	}
	if strings.TrimSpace(in.URL) == "" {
		return nil, ErrURLRequired
	}
	taken, err := s.repo.ExistsByName(workspaceID, name)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrNameTaken
	}
	authType := normalizeAuthType(in.AuthType)
	// Public repos (and a blank secret) are cloned anonymously — no stored secret.
	enc := ""
	if authType != models.GitAuthPublic && strings.TrimSpace(in.Secret) != "" {
		enc, err = crypto.EncryptWS(workspaceID, in.Secret)
		if err != nil {
			return nil, err
		}
	}
	g := &models.GitRepository{
		WorkspaceID: workspaceID,
		Name:        name,
		DisplayName: displayName,
		URL:         normalizeGitURL(in.URL),
		AuthType:    authType,
		Username:    in.Username,
		Secret:      enc,
	}
	if err := s.repo.Create(g); err != nil {
		return nil, err
	}
	return strip(g), nil
}

func (s *Service) Update(workspaceID, id uint, in Input) (*models.GitRepository, error) {
	g, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if name := slug.Make(in.Name, ""); name != "" && name != g.Name {
		taken, err := s.repo.ExistsByName(workspaceID, name)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrNameTaken
		}
		g.Name = name
	}
	if dn := strings.TrimSpace(in.DisplayName); dn != "" {
		g.DisplayName = dn
	}
	if in.URL != "" {
		g.URL = normalizeGitURL(in.URL)
	}
	if in.AuthType != "" {
		g.AuthType = normalizeAuthType(in.AuthType)
	}
	g.Username = in.Username
	switch {
	case g.AuthType == models.GitAuthPublic:
		g.Secret = "" // public repo carries no credential
	case strings.TrimSpace(in.Secret) != "":
		enc, err := crypto.EncryptWS(workspaceID, in.Secret)
		if err != nil {
			return nil, err
		}
		g.Secret = enc
	}
	if err := s.repo.Update(g); err != nil {
		return nil, err
	}
	return strip(g), nil
}

func (s *Service) Get(workspaceID, id uint) (*models.GitRepository, error) {
	g, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return strip(g), nil
}

func (s *Service) List(workspaceID uint) ([]models.GitRepository, error) {
	repos, err := s.repo.ListByWorkspace(workspaceID)
	if err != nil {
		return nil, err
	}
	for i := range repos {
		strip(&repos[i])
	}
	return repos, nil
}

func (s *Service) Delete(workspaceID, id uint) error {
	g, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return ErrNotFound
	}
	return s.repo.Delete(g.ID)
}

// TestConnection performs an authenticated ls-remote against the repository.
func (s *Service) TestConnection(ctx context.Context, workspaceID, id uint) error {
	g, err := s.repo.FindInWorkspace(workspaceID, id)
	if err != nil {
		return ErrNotFound
	}
	auth, err := AuthFor(g)
	if err != nil {
		return err
	}
	rem := gogit.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{Name: "origin", URLs: []string{g.URL}})
	if _, err := rem.ListContext(ctx, &gogit.ListOptions{Auth: auth}); err != nil {
		return fmt.Errorf("git connection failed: %w", err)
	}
	return nil
}

// Checkout clones url into dir and checks out ref, returning the resolved commit
// hash. When ref is empty the cloned HEAD is used. log receives progress lines
// (nil is allowed). It is the shared clone+checkout path used by both the deploy
// worker's git build and the pipeline runner's workspace, so the two can't drift
// in how they resolve a revision to a concrete commit.
func Checkout(ctx context.Context, dir, url, ref string, auth transport.AuthMethod, log func(string)) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", ErrURLRequired
	}
	logf := func(s string) {
		if log != nil {
			log(s)
		}
	}
	logf("cloning " + url)
	repo, err := gogit.PlainCloneContext(ctx, dir, false, &gogit.CloneOptions{URL: url, Auth: auth})
	if err != nil {
		return "", fmt.Errorf("git clone: %w", err)
	}
	if strings.TrimSpace(ref) == "" {
		head, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("resolve HEAD: %w", err)
		}
		return head.Hash().String(), nil
	}
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return "", fmt.Errorf("resolve ref %q: %w", ref, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	if err := wt.Checkout(&gogit.CheckoutOptions{Hash: *hash}); err != nil {
		return "", fmt.Errorf("checkout %q: %w", ref, err)
	}
	logf("checked out " + ref + " (" + hash.String()[:min(7, len(hash.String()))] + ")")
	return hash.String(), nil
}

// AuthFor builds a go-git auth method from a stored credential, decrypting the
// secret. Returns nil auth (anonymous) when g is nil. Shared by the deploy
// worker and the test-connection check.
func AuthFor(g *models.GitRepository) (transport.AuthMethod, error) {
	if g == nil {
		return nil, nil
	}
	// Public repos, or any repo with no stored secret, clone anonymously.
	if normalizeAuthType(g.AuthType) == models.GitAuthPublic || strings.TrimSpace(g.Secret) == "" {
		return nil, nil
	}
	secret, err := crypto.Decrypt(g.Secret)
	if err != nil {
		return nil, fmt.Errorf("decrypt git secret: %w", err)
	}
	switch normalizeAuthType(g.AuthType) {
	case models.GitAuthSSH:
		user := g.Username
		if user == "" {
			user = "git"
		}
		keys, err := gitssh.NewPublicKeys(user, []byte(secret), "")
		if err != nil {
			return nil, fmt.Errorf("parse ssh key: %w", err)
		}
		return keys, nil
	default: // token / HTTPS basic auth
		user := g.Username
		if user == "" {
			user = "x-access-token" // provider-agnostic default for PAT auth
		}
		return &githttp.BasicAuth{Username: user, Password: secret}, nil
	}
}

// ErrSSHUnsupportedOnRunner is returned when a repo authenticates by SSH key but
// the clone must happen on a remote runner (which has no way to receive the key
// via the URL). The user should add an HTTPS token credential instead.
var ErrSSHUnsupportedOnRunner = errors.New("SSH-key git credentials can't be used for runner builds yet; add an HTTPS token credential for this repository")

// CredentialURL returns rawURL with the repository's HTTPS credential embedded
// (https://user:token@host/…), so a remote builder — a runner cloning over the
// network with no local git auth — can clone a private repo. A public repo (or
// one with no stored secret) returns the URL unchanged. An SSH-key credential
// can't be carried in a URL, so it returns ErrSSHUnsupportedOnRunner.
//
// The credential lands only in the runner's ephemeral per-job workspace git
// config (removed when the job ends) and is never logged (the runner treats the
// source URL as opaque/secret).
func CredentialURL(rawURL string, g *models.GitRepository) (string, error) {
	rawURL = normalizeGitURL(rawURL)
	if g == nil || normalizeAuthType(g.AuthType) == models.GitAuthPublic || strings.TrimSpace(g.Secret) == "" {
		return rawURL, nil
	}
	if normalizeAuthType(g.AuthType) == models.GitAuthSSH {
		return "", ErrSSHUnsupportedOnRunner
	}
	secret, err := crypto.Decrypt(g.Secret)
	if err != nil {
		return "", fmt.Errorf("decrypt git secret: %w", err)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse git url %q: %w", rawURL, err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("token auth requires an http(s) repository URL, got scheme %q", u.Scheme)
	}
	user := g.Username
	if user == "" {
		user = "x-access-token" // provider-agnostic default for PAT auth
	}
	u.User = url.UserPassword(user, secret) // url-encodes tokens with special chars
	return u.String(), nil
}

func strip(g *models.GitRepository) *models.GitRepository {
	g.HasSecret = g.Secret != ""
	g.Secret = ""
	return g
}

func normalizeAuthType(t models.GitAuthType) models.GitAuthType {
	switch t {
	case models.GitAuthSSH:
		return models.GitAuthSSH
	case models.GitAuthPublic:
		return models.GitAuthPublic
	default:
		return models.GitAuthToken
	}
}

// normalizeGitURL trims the URL and appends a ".git" suffix when missing, since
// users routinely forget it. SSH (git@…) and HTTPS URLs are both handled; an
// already-".git" URL is left as-is.
func normalizeGitURL(raw string) string {
	u := strings.TrimRight(strings.TrimSpace(raw), "/")
	if u == "" {
		return u
	}
	if !strings.HasSuffix(strings.ToLower(u), ".git") {
		u += ".git"
	}
	return u
}
