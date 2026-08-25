# Deferred migrations

Files here are **not** embedded into the binary (`db/embed.go` only embeds `migrations/*.json`).

## 000010_hostport_configs_idx

`hostport_configs.appID` unique index — temporarily parked so test-env deploys do not apply it.

Before production (or when ready for test env): move the three files back into `db/migrations/` as `000010_…` (or the next free version number if that slot is taken).
