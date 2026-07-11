// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

// ImageRefResolver resolves a platform-image catalog key (e.g. the default
// buildpack builder) to its effective ref (admin override -> config default ->
// built-in). Satisfied by platformimage.Resolver; kept as an interface so the
// worker doesn't import the settings stack. The control plane resolves the
// admin-controlled builder image with it and passes it to the runner in the
// build config — builds themselves run on the runner, never here.
type ImageRefResolver interface {
	Ref(key string) string
}

// BuildResult is what a runner build reports for provenance: the pushed image
// digest, its size when known, and which runner built it.
type BuildResult struct {
	Digest string
	Size   int64
	Runner string
}
