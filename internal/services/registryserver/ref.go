// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package registryserver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/miabi-io/miabi/internal/config"
)

var (
	// ErrForeignNamespace is returned when an image reference addresses this
	// registry under a namespace that belongs to a different workspace. It is a
	// tenant boundary violation, not a typo: the platform credential used to pull
	// distributed images is authorized for every namespace, so an unchecked
	// reference is a cross-workspace read.
	ErrForeignNamespace = errors.New("image belongs to another workspace's registry namespace")
	// ErrUnknownNamespace is returned when an internal-registry reference names a
	// namespace that resolves to no workspace. It is refused rather than passed
	// through: a namespace that is free today can be claimed by a rename tomorrow,
	// which would silently turn a broken reference into a foreign one.
	ErrUnknownNamespace = errors.New("image names an unknown workspace registry namespace")
	// ErrHostInvalid is returned when the configured registry hostname is not a
	// usable Docker registry host.
	ErrHostInvalid = config.ErrRegistryHostInvalid
)

// NormalizeHost validates a registry hostname and returns its canonical form, or
// "" for an unset one. The rule lives in the config package because boot has to
// apply it before this service exists (an unusable host must not start the
// platform at all); this is the same check, reachable from the registry code
// that depends on it.
func NormalizeHost(raw string) (string, error) { return config.NormalizeRegistryHost(raw) }

// splitInternalRef splits an image reference that addresses this registry into
// its workspace namespace segment and the remainder (image path plus tag or
// digest). internal is false when the reference points somewhere else entirely,
// in which case the reference is not ours to authorize.
//
// An internal reference with no namespace segment (<host>/<image>) returns an
// empty namespace with internal true: it addresses this registry but names no
// workspace, so the caller must refuse it rather than treat it as external.
func splitInternalRef(host, ref string) (namespace, rest string, internal bool) {
	host = strings.TrimSpace(host)
	ref = strings.TrimSpace(ref)
	if host == "" || ref == "" {
		return "", "", false
	}
	body, found := strings.CutPrefix(ref, host+"/")
	if !found {
		return "", "", false
	}
	ns, rest, found := strings.Cut(body, "/")
	if !found || ns == "" || rest == "" {
		return "", body, true
	}
	return ns, rest, true
}

// IsInternalRef reports whether ref addresses this platform's built-in registry.
// It says nothing about who may pull it — use ResolveImageRef for that.
func (s *Service) IsInternalRef(ref string) bool {
	_, _, internal := splitInternalRef(s.RegistryHost(), ref)
	return internal
}

// ResolveImageRef authorizes an image reference for a workspace and returns the
// reference the platform should actually pull.
//
// A reference that does not address the built-in registry is returned unchanged:
// it is an external image, governed by the app's own registry credential.
//
// A reference that DOES address the built-in registry is only allowed when its
// namespace resolves to workspaceID, and the returned reference is rewritten to
// the immutable ws_<id> form. Both halves matter:
//
//   - Without the check, any workspace could deploy <host>/<other-workspace>/<image>
//     and the node would pull it with the platform credential, which is authorized
//     for every namespace — a cross-tenant image read that never touches the
//     per-request authorization in Authorize.
//   - Rewriting to ws_<id> means a stored reference survives a workspace rename,
//     so a rollback to an older deployment keeps pulling the image it recorded.
func (s *Service) ResolveImageRef(workspaceID uint, ref string) (string, error) {
	host := s.RegistryHost()
	ns, rest, internal := splitInternalRef(host, ref)
	if !internal {
		return ref, nil
	}
	if ns == "" {
		return "", fmt.Errorf("%w: %q — expected %s/<workspace>/<image>", ErrUnknownNamespace, ref, host)
	}
	ws, err := s.resolveNamespace(ns)
	if err != nil || ws == nil {
		return "", fmt.Errorf("%w: %q in %q", ErrUnknownNamespace, ns, ref)
	}
	if ws.ID != workspaceID {
		return "", fmt.Errorf("%w: %q is in workspace %q", ErrForeignNamespace, ref, ws.Name)
	}
	return host + "/" + Namespace(ws.ID) + "/" + rest, nil
}

// ValidateImageRef reports whether workspaceID may use ref, without rewriting
// it. It is the write-time guard (app create/update): refusing a foreign
// reference when it is typed gives the user the error where they can act on it,
// instead of at deploy time on a node.
func (s *Service) ValidateImageRef(workspaceID uint, ref string) error {
	_, err := s.ResolveImageRef(workspaceID, ref)
	return err
}
