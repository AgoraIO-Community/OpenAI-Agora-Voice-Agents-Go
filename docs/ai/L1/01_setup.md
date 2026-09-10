# 01 Setup

> Local environment for the two-process Go + Next.js quickstart: prerequisites, env vars, and verification commands wired through the `Makefile`, root `package.json`, and `client/scripts/`.

## Prerequisites

- **Go** 1.23.* / 1.24.* / 1.25.* / 1.26.* / 2.* (matched by `make doctor-local`).
- **Node**: version 22 or newer with **pnpm 11** recommended. The repository does not force pnpm's automatic version switching, which can require registry access before local scripts run.
- **pnpm** 10.23.0 (enforced by the workspaces config; the root `package.json` is a pnpm workspace).
- Agora project with App ID + App Certificate.

## Install

```bash
make setup
# or:
pnpm install
cd server && go mod tidy
```

`make setup` chains `setup-env`, `setup-backend`, and `setup-frontend`. Setup preserves a configured `server/.env`, copies a legacy `server/.env.local` to `server/.env` when needed, or seeds `server/.env` from `server/.env.example`. If the seeded file has both example values and a CLI version writes valid credentials to the legacy path, `setup-env` and `doctor-local` copy those credentials into `server/.env`. The completion message omits the credential-writing command when the resulting file has non-placeholder Agora credentials. Backend setup runs `go mod tidy`; frontend setup runs `pnpm install` when `node_modules/` is missing.

## Environment Variables

`server/.env.example`:

```
AGORA_APP_ID=your_agora_app_id
AGORA_APP_CERTIFICATE=your_agora_app_certificate
PORT=8000
```

| Variable                 | Process     | Required | Notes                                                                 |
| ------------------------ | ----------- | -------- | --------------------------------------------------------------------- |
| `AGORA_APP_ID`           | Go (server) | Yes      | Loaded in `newAgentService`.                                          |
| `AGORA_APP_CERTIFICATE`  | Go (server) | Yes      | Loaded in `newAgentService`; never exposed to the browser.            |
| `OPENAI_API_KEY`         | Go (server) | Yes      | Credential for the OpenAI GPT Live preview provider.                  |
| `AGENT_INSTRUCTIONS`     | Go (server) | No       | Persistent GPT Live behavior guidance; has a built-in default.        |
| `AGENT_PRIOR_MESSAGES`   | Go (server) | No       | JSON array of prior user/assistant text turns; defaults to `[]`.      |
| `PORT`                   | Go (server) | No       | Default `8000` (`main.go`).                                            |
| `AGENT_BACKEND_URL`      | Next build  | Yes for rewrites | Empty/missing means no `/api/*` rewrites are registered.       |
| `NEXT_PUBLIC_AGENT_UID`  | Browser     | No       | Optional override read in `ConversationComponent.tsx`.                |

## Quick Commands

```bash
make setup            # one-time bootstrap
make doctor           # pnpm + node_modules presence
make doctor-local     # adds Go + server/.env + Agora credential presence
make dev              # spawns Gin + Next dev with AGENT_BACKEND_URL set
make fmt              # gofmt server *.go and cmd/fake-server/*.go
make build            # build-backend + build-web
make test             # alias for test-backend → verify-backend → go test ./...
make verify           # alias for verify-web (doctor + verify-web-api + verify-web-build)
make verify-local     # doctor-local + backend + local-go + web-proxy + web-build
make clean            # remove backend bin, node_modules, .next, client/dist
```

Useful narrower targets:

```bash
make verify-web-api      # client/scripts/verify-api-contracts.ts
make verify-web-proxy    # client/scripts/verify-local-proxy.ts
make verify-local-go     # builds cmd/fake-server, runs verify-local-go.ts
make verify-backend      # cd server && go test ./...
make build-backend       # produces server/bin/agent-quickstart-go
```

The root `package.json` exposes the same workflows under `pnpm run setup`, `pnpm run dev`, `pnpm run verify`, etc. — the Makefile remains the canonical entry point.

## Verification Safety

| Command                | Live Agora? | Notes                                                |
| ---------------------- | ----------- | ---------------------------------------------------- |
| `make doctor`          | No          | pnpm + node_modules sanity                           |
| `make doctor-local`    | No          | Adds Go + env presence                               |
| `make verify-web-api`  | No          | Contract harness with mocked SDK                     |
| `make verify-web-proxy`| No          | Static fake-server smoke                             |
| `make verify-local-go` | No          | Boots `cmd/fake-server`, exercises web → proxy → Go  |
| `make verify-backend`  | No          | `go test ./...`                                      |
| `make verify-web-build`| No          | `pnpm build`                                         |
| `make dev`             | Yes (for use) | Often blocked in sandboxes due to port binding     |

## Common Setup Failures

- `make doctor-local` reports **"Go version not supported"** → install Go 1.23+.
- Doctor fails on missing or placeholder credentials in `server/.env` → run `agora quickstart env write .`.
- `make verify-web-api` fails on a new route → extend `client/scripts/verify-api-contracts.ts` to cover it.
- `make dev` exits with port-in-use → either Gin or Next is already running; check ports 8000 and 3000.

## Related Deep Dives

- [Verification Scripts](L2/verification_scripts.md) — How the four contract checks differ and when to run each.
