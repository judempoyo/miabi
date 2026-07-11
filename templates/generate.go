// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package templates

// The embedded official catalog is a vendored snapshot of the marketplace
// registry's official/ set — the offline floor. It is GENERATED, not hand-kept:
// `go generate ./templates` rebuilds the slug trees, index.yaml, and the
// //go:embed list in embed.go from the marketplace's published export.json.
//
// Source defaults to the sibling repo's committed bundle; for a release, vendor
// from the published bundle instead, e.g.:
//
//	go run ./gen -source https://github.com/miabi-io/marketplace/releases/latest/download/export.json -out .
//
//go:generate go run ./gen -source ../../marketplace/export.json -out .
