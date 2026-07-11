// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package registryserver

import (
	"context"
	"errors"
	"sort"
	"strings"
)

var (
	// ErrNotFound is returned when a repository, tag, or manifest is absent.
	ErrNotFound = errors.New("registry: not found")
	// ErrDeleteDisabled is returned when the registry's delete API is off
	// (REGISTRY_STORAGE_DELETE_ENABLED=false / the DeleteEnabled setting).
	ErrDeleteDisabled = errors.New("registry: delete is not enabled")
)

// Repository is a workspace repository and its tags, with the immutable ws_<id>
// storage namespace stripped so callers see the user-facing image name.
type Repository struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// ListRepositories returns the workspace's repositories (its ws_<id> namespace,
// filtered from the registry catalog) with their tags.
func (s *Service) ListRepositories(ctx context.Context, workspaceID uint) ([]Repository, error) {
	prefix := Namespace(workspaceID) + "/"
	all, err := s.reg.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Repository, 0)
	for _, repo := range all {
		if !strings.HasPrefix(repo, prefix) {
			continue
		}
		image := strings.TrimPrefix(repo, prefix)
		tags, err := s.reg.Tags(ctx, repo)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		sort.Strings(tags)
		out = append(out, Repository{Name: image, Tags: tags})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// DeleteTag deletes a tag from a workspace repository by resolving it to its
// manifest digest and deleting the manifest (the registry has no delete-by-tag).
// image is the user-facing name (without the ws_<id> namespace).
func (s *Service) DeleteTag(ctx context.Context, workspaceID uint, image, tag string) error {
	repo := Namespace(workspaceID) + "/" + strings.Trim(image, "/")
	digest, err := s.reg.ManifestDigest(ctx, repo, tag)
	if err != nil {
		return err
	}
	return s.reg.DeleteManifest(ctx, repo, digest)
}
