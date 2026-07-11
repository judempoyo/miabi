// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package web embeds the built frontend (the Vue SPA) into the Go binary so a
// single `miabi` executable serves both the API and the UI — no separate static
// files to ship or MIABI_WEB_DIR to configure.
//
// The `dist/` directory is a build artifact: `make build-ui` runs the Vue build
// and copies its output here, then `go build` bakes it in. Only a placeholder
// (.gitkeep) is committed, so `go build`/`go test` always compile even when the
// UI has not been built; in that case the embedded FS is empty and the server
// serves the API with the UI routes 404-ing. Real release/Docker builds always
// build the UI first, so shipped binaries are self-contained.
package web

import "embed"

// Assets holds the built SPA under a top-level "dist/" directory. Served via
// Okapi's WebFS with WebConfig{Root: "dist"}. The `all:` prefix ensures dotted
// files (e.g. the .gitkeep placeholder, Vite's occasional dotfiles) are included.
//
//go:embed all:dist
var Assets embed.FS
