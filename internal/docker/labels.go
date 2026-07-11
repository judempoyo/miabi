// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package docker

import (
	"strconv"
	"strings"
)

// Platform Docker label keys. Every container / volume / service Miabi creates
// carries a back-reference label to its owning record under the io.miabi.*
// namespace (reverse-DNS of miabi.io). These are the single source of truth —
// do not hand-write the string literals elsewhere.
const (
	// LabelPrefix namespaces every platform-applied Docker label.
	LabelPrefix = "io.miabi."

	LabelApp         = "io.miabi.app"          // application id
	LabelDeployment  = "io.miabi.deployment"   // deployment id
	LabelDatabase    = "io.miabi.database"     // database instance id
	LabelStack       = "io.miabi.stack"        // stack id
	LabelVolume      = "io.miabi.volume"       // volume id
	LabelJob         = "io.miabi.job"          // one-shot job container (transient)
	LabelRole        = "io.miabi.role"         // platform-infra role (node-gateway, …)
	LabelNode        = "io.miabi.node"         // node slug
	LabelWorkspace   = "io.miabi.workspace"    // owning workspace id
	LabelManaged     = "io.miabi.managed"      // "true" on managed raw resources/services
	LabelPipelineRun = "io.miabi.pipeline-run" // pipeline run id (transient)
	LabelSizeBytes   = "io.miabi.size_bytes"   // volume size hint
)

// ManagedLabel marks resources created by Miabi (kept as a named alias for the
// many call sites that tag raw containers/volumes/services).
const ManagedLabel = LabelManaged

// LabelValue reads a platform label by its io.miabi.* key. ok is false when the
// label is absent.
func LabelValue(labels map[string]string, key string) (value string, ok bool) {
	if labels == nil {
		return "", false
	}
	v, ok := labels[key]
	return v, ok
}

// IsManaged reports whether a resource carries any platform label — i.e. it is
// owned by Miabi and must not be a blanket prune/delete target.
func IsManaged(labels map[string]string) bool {
	for k := range labels {
		if strings.HasPrefix(k, LabelPrefix) {
			return true
		}
	}
	return false
}

// IsPlatformInfra reports whether a resource is platform infrastructure (carries
// a role label) — the node's edge gateway / its Redis. Such resources are
// managed through their own pages, never reclaimed or hidden as "someone else's".
func IsPlatformInfra(labels map[string]string) bool {
	_, ok := LabelValue(labels, LabelRole)
	return ok
}

// reservedUserLabelPrefixes are label namespaces a user may never write on their
// own containers: the platform's own keys (ownership / workspace scoping /
// housekeeping all read them, so a spoofed io.miabi.workspace could break
// isolation) and the Docker Compose grouping keys stackLabels manages.
var reservedUserLabelPrefixes = []string{LabelPrefix, "com.docker."}

// IsReservedLabelKey reports whether key is platform-reserved — i.e. a user is
// not allowed to set it as a custom container label.
func IsReservedLabelKey(key string) bool {
	for _, p := range reservedUserLabelPrefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// SanitizeUserLabels returns a copy of in with every reserved key removed, so
// user-supplied container labels can never set or spoof a platform label. Keys
// with an empty name are dropped. nil in → nil out.
func SanitizeUserLabels(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if k == "" || IsReservedLabelKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}

// WorkspaceID returns the owning workspace id encoded in a resource's labels.
// ok is false when there is no (valid) workspace label — e.g. raw/system
// containers or platform infrastructure.
func WorkspaceID(labels map[string]string) (id uint, ok bool) {
	v, present := LabelValue(labels, LabelWorkspace)
	if !present {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}
