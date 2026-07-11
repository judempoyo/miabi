// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package registryserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/miabi-io/miabi/internal/docker"
)

// platformUser is the docker username the build/deploy worker logs in as; it is
// advisory (the platform token is what authorizes), like any registry username.
const platformUser = "_miabi"

// DistributionEnabled reports whether built images can be pushed to and pulled
// from the registry: it must be enabled, have a resolvable host, and have a
// platform token configured. When false the build/deploy flow uses node-local
// images, so single-node installs are unaffected.
func (s *Service) DistributionEnabled() bool {
	return s.DistributionUnavailableReason() == ""
}

// DistributionUnavailableReason returns "" when image distribution is ready, or
// a user-facing sentence naming the specific missing piece (registry disabled,
// no platform token, or no resolvable host). Keeping the diagnosis here means a
// git-source deploy can tell the operator exactly what to configure rather than
// re-suggesting a flag they already set.
func (s *Service) DistributionUnavailableReason() string {
	st, err := s.Get()
	if err != nil {
		return "registry settings could not be read: " + err.Error()
	}
	if !st.Enabled {
		return "the internal registry is disabled (set MIABI_REGISTRY_ENABLED=true)"
	}
	// The platform token is derived from the master key, so it is always present
	// (no operator action needed); only enablement and a resolvable host remain.
	if s.HostFor(st) == "" {
		return "the registry host cannot be resolved (set MIABI_REGISTRY_HOST, or configure the external base domain so it defaults to registry.<domain>)"
	}
	return ""
}

// TagReleaseVersion adds a v<version> tag to an already-pushed build image
// (identified by its digest) so the registry mirrors the release number shown in
// the UI, alongside the immutable build tag. Best-effort and a no-op when
// distribution is off. It talks to the registry directly, so it uses the internal
// ws_<id>/<app-name> storage path (the runner pushed via the gateway, which
// rewrote the workspace name to its id).
func (s *Service) TagReleaseVersion(ctx context.Context, workspaceID uint, appName, digest string, version int) error {
	if !s.DistributionEnabled() {
		return nil
	}
	repo := Namespace(workspaceID) + "/" + appName
	return s.reg.TagManifest(ctx, repo, digest, fmt.Sprintf("v%d", version))
}

// RegistryHost returns the resolved registry host — the explicit setting, else
// registry.<external-base-domain>, else "" when distribution can't be served. It
// is the host a runner must both log into and push to; deriving it here (rather
// than trusting the raw MIABI_REGISTRY_HOST env, which is empty when the host is
// UI-set or domain-derived) keeps the runner's login host and push host identical.
func (s *Service) RegistryHost() string {
	st, err := s.Get()
	if err != nil {
		return ""
	}
	return s.HostFor(st)
}

// BuildRef is the registry reference a built image is distributed under:
// <host>/<workspace-name>/<app-name>:<deploymentID>. Both path segments use the
// human-readable, per-workspace-unique handles a user pushes to (Connect tab), so
// a runner build authenticates and pushes exactly like a user — the gateway
// rewrites the workspace name to the immutable ws_<id> for storage. The workspace
// namespace falls back to the ws_<id> form when the name can't be resolved (no
// workspace finder wired), which still works through the same rewrite.
func (s *Service) BuildRef(workspaceID uint, appName string, deploymentID uint) string {
	st, err := s.Get()
	if err != nil {
		return ""
	}
	host := s.HostFor(st)
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s:%d", host, s.namespaceSegment(workspaceID), appName, deploymentID)
}

// namespaceSegment is the repository's first path segment for a workspace: its
// name when resolvable (matching user pushes), else the ws_<id> id form.
func (s *Service) namespaceSegment(workspaceID uint) string {
	if s.ws != nil {
		if w, err := s.ws.FindByID(workspaceID); err == nil && w.Name != "" {
			return w.Name
		}
	}
	return Namespace(workspaceID)
}

// IsBuildRef reports whether ref is one of this registry's distributed image
// refs (i.e. it lives under the registry host), so the deploy path knows to pull
// it from the internal registry rather than rebuild.
func (s *Service) IsBuildRef(ref string) bool {
	st, err := s.Get()
	if err != nil {
		return false
	}
	host := s.HostFor(st)
	return host != "" && strings.HasPrefix(ref, host+"/")
}

// PushAuth is the credential the worker uses to push/pull built images (the
// platform token). Returns nil when distribution is unconfigured.
func (s *Service) PushAuth() *docker.RegistryAuth {
	st, err := s.Get()
	if err != nil {
		return nil
	}
	host := s.HostFor(st)
	if host == "" {
		return nil
	}
	return &docker.RegistryAuth{Server: host, Username: platformUser, Password: s.platformToken()}
}
