# Agora Conversational AI Demo — Architecture

The UI stays the same across environments. Next owns the browser-facing `/api/*` paths as rewrites; the Go service owns the backend handlers.

## Local Go-Backed Development

```text
Browser
  ↓
Next.js app on :3000
  ↓
/api/* rewrites through AGENT_BACKEND_URL
  ↓
Gin service on :8000
  ↓
Agora Cloud Services
```

- `client` owns the browser UI and the web-facing routes
- `server` owns token generation and agent lifecycle for local development
- `make dev` starts both services together

## Deployed Rewrite Mode

```text
Browser
  ↓
Next.js app
  ↓
/api/* rewrites through AGENT_BACKEND_URL
  ↓
Reachable Gin service
  ↓
Agora Cloud Services
```

- `client` handles the UI and the rewrite facade
- `server` remains required unless the architecture intentionally adds Next route handlers
- `AGENT_BACKEND_URL` must point to the deployed Go service

## Shared Flow

1. `GET /api/get_config` returns app ID, token, channel, user UID, and agent UID.
2. `POST /api/startAgent` creates a managed agent session scoped to the requesting user.
3. The agent joins RTC, publishes audio, and emits transcript events over RTM.
4. `POST /api/stopAgent` removes the process-local retained session and calls `session.Stop`; unknown or repeated IDs are idempotent success.

## API Contract

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/get_config` | GET | Generate RTC + RTM config |
| `/startAgent` | POST | Start an agent session |
| `/stopAgent` | POST | Stop an agent session |

The frontend always calls `/api/*`. `client/next.config.ts` rewrites those paths to `AGENT_BACKEND_URL`; no `client/app/api/**/route.ts` handlers exist in this repo.

## Authentication

The Go backend uses app credentials mode with the official Agora Agent Server SDK for Go. Combined RTC + RTM tokens are generated from `AGORA_APP_ID` and `AGORA_APP_CERTIFICATE`, and REST auth is handled by the SDK.

Agent sessions are process-local. Multi-instance deployments must keep start and stop on the same process or provide shared lifecycle state; the service does not issue standalone stop requests.

## References

- [docs/ai/L1/02_architecture.md](./docs/ai/L1/02_architecture.md) — detailed client ↔ server flow
- [docs/ai/L1/03_code_map.md](./docs/ai/L1/03_code_map.md) — where code lives under `client/` and `server/`
- [README.md](./README.md)
