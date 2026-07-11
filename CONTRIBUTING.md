# Contributing to Miabi

Thanks for your interest in improving Miabi! This document covers how to
contribute and the licensing terms your contributions are made under.

## Development

- The Go application lives in this repository (module `github.com/miabi-io/miabi`).
- Build, run, lint, and test with the targets in the [`Makefile`](./Makefile);
  the web console is under [`web/`](./web).
- Please run `go build ./...`, `go vet ./...`, the relevant tests, and
  `golangci-lint` before opening a pull request, and keep the OpenAPI spec
  generated from code annotations (don't hand-edit it).

## Licensing of contributions

Miabi is **open-core** (see [LICENSING.md](./LICENSING.md)):

- The **Community core** is licensed **AGPL-3.0-or-later**.
- The **Enterprise Edition** (`internal/enterprise/`, `enterprise` build tag) is
  **proprietary**, under a separate commercial license.

To keep this model lawful as the project grows beyond a single author — so the
maintainer can keep offering Miabi under **both** the AGPL and a **commercial
license**, and can combine the AGPL core with the proprietary Enterprise Edition —
**every contribution is accepted under the Miabi Contributor License Agreement
([CLA.md](./CLA.md))**.

By submitting a contribution (a pull request, patch, or any other change) you:

1. **Certify the origin** of your work with a Developer Certificate of Origin
   (DCO) sign-off — add a `Signed-off-by: Your Name <you@example.com>` line to
   each commit (`git commit -s`); and
2. **Agree to the CLA** in [CLA.md](./CLA.md), which grants the maintainer a
   broad copyright and patent license to your contribution, including the right to
   sublicense and relicense it (for example, to distribute it as part of the
   AGPL core **and** under the commercial Enterprise license). You retain the
   copyright to your contribution.

If you are contributing on behalf of an employer, make sure you are authorized to
do so under these terms.

Contributions to files marked `SPDX-License-Identifier: LicenseRef-Miabi-Enterprise`
are only accepted from authorized Enterprise contributors.

## Reporting security issues

Please do **not** open a public issue for security vulnerabilities — follow
[SECURITY.md](./SECURITY.md) instead.
