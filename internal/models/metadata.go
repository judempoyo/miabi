// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"strconv"
	"strings"
)

// Metadata is a free-form set of key/value labels attached to a resource
// (Application, Stack, Database, Volume, Route, Secret). It backs provenance,
// grouping, and the declarative/GitOps features.
//
// Keys under MetadataReservedPrefix are platform-managed ("built-in") and are
// protected from user modification; all other keys are user-defined and freely
// editable. Persisted as JSON (gorm serializer).
type Metadata = map[string]string

// MetadataReservedPrefix marks built-in, platform-managed metadata keys.
const MetadataReservedPrefix = "miabi.io/"

// Built-in metadata keys (all under MetadataReservedPrefix).
const (
	MetaManagedBy       = "miabi.io/managed-by"       // origin: see ManagedBy* values
	MetaTemplate        = "miabi.io/template"         // marketplace template slug
	MetaTemplateVersion = "miabi.io/template-version" // marketplace template version
	MetaTemplateInstall = "miabi.io/template-install" // template install id
	MetaStack           = "miabi.io/stack"            // owning stack docker name
	MetaGitOpsSource    = "miabi.io/gitops-source"    // id of the GitOps project that created the resource
	// MetaRuntimeAutoService marks an app whose service runtime was auto-defaulted
	// by cluster mode (not chosen by the caller). It is a one-shot marker: the first
	// deploy re-evaluates the choice — downgrading to a container if the app turns
	// out to hold node-local state — then clears it, so a later explicit runtime
	// choice is always respected. Value: "true".
	MetaRuntimeAutoService = "miabi.io/runtime-auto-service"
	// MetaDeclarativeName records the declarative/manifest resource name a logical
	// database was provisioned for. A manifest Database may live as a dedicated
	// instance or as a logical database sharing another instance, so this tag is
	// how a reconcile maps the manifest name back to the exact logical database it
	// owns (instead of guessing by instance name / first-database).
	MetaDeclarativeName = "miabi.io/declarative-name"

	// Owner reference: what entity this resource belongs to / whose lifecycle it
	// follows. Distinct from managed-by (the *mechanism* of creation); owner is
	// the *parent*, stored as kind+id+name so the UI can link to it without a
	// join. See OwnerKind* values and SetOwner/Owner.
	MetaOwnerKind = "miabi.io/owner-kind" // one of OwnerKind* values
	MetaOwnerID   = "miabi.io/owner-id"   // numeric id of the owner (omitted when 0)
	MetaOwnerName = "miabi.io/owner-name" // owner display name (for rendering + linking)
)

// ManagedBy values record how a resource came to exist.
const (
	ManagedByUser        = "user"         // created by hand
	ManagedByMarketplace = "marketplace"  // installed from a marketplace template
	ManagedByStack       = "stack"        // created as part of a stack
	ManagedByStackImport = "stack-import" // created by importing a compose file
)

// OwnerKind values classify what a resource belongs to (the MetaOwnerKind value).
const (
	OwnerUser     = "user"     // a person created it for their own use (owner id = user id)
	OwnerApp      = "app"      // it backs an application
	OwnerDatabase = "database" // it backs a database instance
	OwnerStack    = "stack"    // it belongs to a stack
)

// OwnerRef is the parsed owner reference attached to a resource's metadata.
type OwnerRef struct {
	Kind string `json:"kind"`           // one of OwnerKind*
	ID   uint   `json:"id,omitempty"`   // owning resource id (0 = none/not applicable)
	Name string `json:"name,omitempty"` // display name
}

// SetOwner records the owner reference on m (creating the map if needed) and
// returns it. id is omitted when 0 and name when empty, so callers can record a
// partial reference (e.g. kind+id with the name resolved later).
func SetOwner(m Metadata, kind string, id uint, name string) Metadata {
	if m == nil {
		m = Metadata{}
	}
	m[MetaOwnerKind] = kind
	if id > 0 {
		m[MetaOwnerID] = strconv.FormatUint(uint64(id), 10)
	} else {
		delete(m, MetaOwnerID)
	}
	if name != "" {
		m[MetaOwnerName] = name
	} else {
		delete(m, MetaOwnerName)
	}
	return m
}

// Owner parses the owner reference from m. ok is false when no owner kind is set.
func Owner(m Metadata) (OwnerRef, bool) {
	if m == nil || m[MetaOwnerKind] == "" {
		return OwnerRef{}, false
	}
	id, _ := strconv.ParseUint(m[MetaOwnerID], 10, 64)
	return OwnerRef{Kind: m[MetaOwnerKind], ID: uint(id), Name: m[MetaOwnerName]}, true
}

// DefaultOwner sets the owner reference only when one is not already present, so
// higher-level callers (marketplace, stacks, apply) that record a richer owner
// win over a creation-path default. Returns the (possibly updated) map.
func DefaultOwner(m Metadata, kind string, id uint, name string) Metadata {
	if _, ok := Owner(m); ok {
		return m
	}
	return SetOwner(m, kind, id, name)
}

// IsReservedMetadataKey reports whether key is platform-managed (built-in).
func IsReservedMetadataKey(key string) bool {
	return strings.HasPrefix(key, MetadataReservedPrefix)
}

// SanitizeUserMetadata returns a copy of in with all reserved (built-in) keys
// removed, so user-supplied metadata can never set or spoof platform-managed
// keys. nil in → nil out.
func SanitizeUserMetadata(in Metadata) Metadata {
	if in == nil {
		return nil
	}
	out := make(Metadata, len(in))
	for k, v := range in {
		if IsReservedMetadataKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}

// MergeUserMetadata applies a user-supplied overlay onto the current metadata
// while protecting built-in keys: reserved keys from current are preserved and
// cannot be overridden or removed by the overlay; non-reserved keys are replaced
// wholesale by the overlay (so users can add/remove their own labels). Use on
// update with the user's desired user-metadata as overlay.
func MergeUserMetadata(current, overlay Metadata) Metadata {
	out := make(Metadata, len(current)+len(overlay))
	// Keep only the protected (built-in) keys from the current value.
	for k, v := range current {
		if IsReservedMetadataKey(k) {
			out[k] = v
		}
	}
	// Apply the user's keys, ignoring any reserved keys they try to set.
	for k, v := range overlay {
		if IsReservedMetadataKey(k) {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DefaultManagedBy ensures the managed-by built-in is set, defaulting to value
// when absent, and returns the (possibly newly created) map. Used at creation so
// every resource records its origin.
func DefaultManagedBy(m Metadata, value string) Metadata {
	if m == nil {
		m = Metadata{}
	}
	if m[MetaManagedBy] == "" {
		m[MetaManagedBy] = value
	}
	return m
}

// SetBuiltin sets one or more reserved (built-in) key/value pairs on m, creating
// the map if needed, and returns it. Reserved keys are authoritative, so this
// always wins. Pairs are passed as key, value, key, value, …
func SetBuiltin(m Metadata, kv ...string) Metadata {
	if m == nil {
		m = Metadata{}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}
