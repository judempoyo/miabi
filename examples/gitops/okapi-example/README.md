# GitOps example — Okapi Example

The [`okapi-example` marketplace template](https://github.com/jkaninda/okapi-example)
expressed as a GitOps desired state: a single
[Okapi](https://github.com/jkaninda/okapi) reference app (middleware, routing,
SSE, OpenAPI docs at `/docs`), reconciled from Git instead of one-click installed.

```
okapi-example/
└── stack.yaml    # one Application, exposed over HTTPS
```

## Wire it up

Point a **GitSource** at this folder; Miabi converges the workspace to it on every
push / sweep. Set `BASE`, `TOKEN`, and `WS` (your workspace **name** — the
`/workspaces/{ws}` path accepts the name, its uid, or the numeric id):

```bash
BASE=https://miabi.example.com; WS=acme; TOKEN=mb_xxx
curl -X POST "$BASE/api/v1/workspaces/$WS/gitops" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{
        "name": "okapi-example",
        "git_repository_id": 1,
        "ref": "main",
        "path": "miabi/examples/gitops/okapi-example",
        "sync_policy": "auto",
        "prune": true,
        "self_heal": true
      }'
```

See [../README.md](../README.md) for the full GitSource / webhook / promotion flow.

## Before applying

- **Set `JWT_SECRET`** in `stack.yaml` to a long random value (it's listed under
  `secretEnv`, so Miabi stores it encrypted at rest). The marketplace template
  auto-generates this; in GitOps you provide it.
- `externalAccess: true` publishes the app at `okapi.<platform-base-domain>` —
  requires a platform base domain. For a custom hostname instead, use the
  commented `Domain` + `Route` block at the bottom of `stack.yaml`.

> Marketplace-only fields (`inputs`, `healthcheck`) don't appear here — they're
> handled by the marketplace installer, not the declarative engine.
