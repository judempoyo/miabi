# Miabi Templates (embedded offline floor)

This directory is **not** the full catalog. The official Miabi Marketplace lives in its own
repository:

> **https://github.com/miabi-io/marketplace**

Miabi pulls the complete, always-current catalog from the Marketplace's published
[`export.json`](https://github.com/miabi-io/marketplace/releases/latest/download/export.json) at
runtime and merges it on top of what is embedded here. The Marketplace is where templates are
added, versioned, and maintained — contribute new templates there, not in this folder.

## What ships here

Only a **minimal offline floor** is vendored into the binary so a fresh install works out of the
box without network access to the Marketplace — the core data and infra primitives the panel itself
provisions:

| Template | Purpose |
|----------|---------|
| `postgresql` | PostgreSQL database |
| `mysql` | MySQL database |
| `mongodb` | MongoDB database |
| `redis` | Redis cache / queue |
| `libsql` | libSQL (SQLite-compatible) database |
| `nginx` | Static site / reverse-proxy container |
| `minio` | S3-compatible object storage |

Everything else — WordPress, Ghost, Nextcloud, n8n, Grafana, and the rest — is served from the
Marketplace registry. When the registry is reachable it is authoritative: a synced version
overrides the embedded floor of the same slug, and Marketplace-only templates surface alongside
these. When it is unreachable, these seven remain available offline.

## Regenerating the floor

The embedded set is **generated, not hand-maintained**. It is a vendored snapshot of the
Marketplace's `official/` set. Rebuild the slug trees, `index.yaml`, and the `//go:embed` list from
the Marketplace's published bundle:

```sh
go generate ./templates
# or vendor from a specific published bundle:
go run ./gen -source https://github.com/miabi-io/marketplace/releases/latest/download/export.json -out .
```
