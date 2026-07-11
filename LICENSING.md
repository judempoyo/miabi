# Licensing

Miabi is **open-core**, split across two licenses.

## Miabi Community — AGPL-3.0-or-later

Miabi core is free and open source under the **GNU Affero General Public License,
version 3 or (at your option) any later version (AGPL-3.0-or-later)**. See
[LICENSE](./LICENSE) and [NOTICE](./NOTICE). This covers the entire codebase
except the Enterprise Edition files described below — including the Community
build (everything compiled into the default, tag-free binary).

The AGPL is a strong copyleft license with a **network clause** (section 13): if
you run a modified version of Miabi and let users interact with it over a network
(e.g. you host it as a service), you must offer those users the **complete
corresponding source** of your modified version, also under the AGPL. Running
Miabi unmodified, or for your own internal use, imposes no such obligation beyond
the usual AGPL terms.

If the AGPL's obligations don't fit your use — for example you want to offer a
modified Miabi as a hosted service without publishing your changes, or embed it in
a proprietary product — a **commercial license is available** (see below).

## Miabi Enterprise — Commercial License

Enterprise features are available under a commercial **Miabi Enterprise License**
(Enterprise Edition). The Enterprise files are the sources under
[`internal/enterprise/`](./internal/enterprise) that are built with the
`enterprise` build tag. They are **not** AGPL and require a valid commercial
license to use; the full terms are defined in
[`internal/enterprise/LICENSE.md`](./internal/enterprise/LICENSE.md).

## Dual licensing & the AGPL + Enterprise combination

Jonas Kaninda holds the copyright to the Miabi core, and therefore offers it under
**both** the AGPL (to everyone) and, separately, a **commercial license** to those
who cannot or do not wish to comply with the AGPL. The copyright holder is not
bound by the AGPL for its own code, so it may combine the AGPL-licensed core with
the proprietary Enterprise Edition in a single binary and license that combined
work commercially. This is the basis on which the `enterprise`-tagged build is
distributed.

Contributions to the AGPL core are accepted under a Contributor License Agreement
(see [CONTRIBUTING.md](./CONTRIBUTING.md)) precisely so this dual-licensing —
AGPL to the public, plus a proprietary Enterprise Edition — remains lawful as the
project grows beyond a single author.

For commercial licensing, contact the Miabi project maintainers.

## Per-file markers (SPDX)

Every source file declares its license inline with an [SPDX](https://spdx.dev)
identifier, so the AGPL/Enterprise boundary is explicit and machine-readable:

- Community core: `SPDX-License-Identifier: AGPL-3.0-or-later`
- Enterprise: `SPDX-License-Identifier: LicenseRef-Miabi-Enterprise`

`LicenseRef-Miabi-Enterprise` is the custom (non-SPDX-registered) Miabi Enterprise
License whose full text lives in
[`internal/enterprise/LICENSE.md`](./internal/enterprise/LICENSE.md). Only the
files built with the `enterprise` tag carry it (the SAML, LDAP/AD, and SCIM
providers plus the license-verifying `enterprise.go`); everything else is
AGPL-3.0-or-later. The community stub (`ce_stub.go`), the shared `EE` interface
(`ee.go`), and the always-compiled license-token verification
(`internal/enterprise/license/`) ship in the open-source build and are
AGPL-3.0-or-later.
