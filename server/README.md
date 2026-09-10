# Agora Agent Service

Gin-based Agora Conversational AI backend using the official Go SDK: `github.com/AgoraIO/agora-agents-go/v2`.

## Quick Start

Recommended from the repo root.

Repo setup:

```bash
make setup
```

Setup preserves a configured `server/.env` and copies a legacy `server/.env.local` when needed. The completion message reflects the credential state. Setup and `doctor-local` replace an untouched example file with configured legacy credentials.

Agora credentials:

```bash
agora quickstart env write .
```

Run the app:

```bash
make dev
```

This assumes the Agora CLI is installed and logged in. The command uses the project selected in your Agora CLI context, which is usually your default account project.

`make setup` installs or refreshes both the Go backend dependencies and the frontend workspace dependencies. `make dev` runs the Gin backend and the Next.js frontend together, with the frontend proxying `/api/*` requests to the backend.

If you are not using the Agora CLI, create the env file manually and fill in your project values:

```bash
cp server/.env.example server/.env
```

Backend-only workflow from `server/`:

```bash
cp .env.example .env
go mod tidy
gofmt -w *.go cmd/fake-server/*.go
go test ./...
go build -o ./bin/agent-quickstart-go .
go run .
```

Backend-only Agora CLI env write from `server/`:

```bash
agora quickstart env write ..
```

Required env vars:

- `AGORA_APP_ID`
- `AGORA_APP_CERTIFICATE`

Optional:

- `PORT`

Example:

```bash
AGORA_APP_ID=your_agora_app_id
AGORA_APP_CERTIFICATE=your_agora_app_certificate
PORT=8000
```

`.env.example` is only the reference template. The recommended setup flow is to let the Agora CLI write the real values into `.env`.

If you still need to authenticate with the CLI:

```bash
agora login
```

To select a specific existing project before writing env values, run this from the repo root:

```bash
agora project use <project-id-or-name>
agora quickstart env write .
```

To create a new project instead of using your default project:

```bash
agora project create my-first-voice-agent --feature rtc --feature convoai
agora project use my-first-voice-agent
agora quickstart env write .
```

## API Endpoints

- `GET /get_config`
- `POST /startAgent`
- `POST /stopAgent`

`/get_config` returns a one-hour combined RTC + RTM token. The backend uses the same managed provider chain as the web deployment path: Deepgram `nova-3`, OpenAI `gpt-4o-mini`, and MiniMax `speech_2_6_turbo`.

Manual checks:

```bash
curl 'http://127.0.0.1:8000/get_config?uid=1234&channel=manual-check'
curl -X POST 'http://127.0.0.1:8000/startAgent' \
  -H 'Content-Type: application/json' \
  -d '{"channelName":"manual-check","rtcUid":9999,"userUid":1234}'
curl -X POST 'http://127.0.0.1:8000/stopAgent' \
  -H 'Content-Type: application/json' \
  -d '{"agentId":"fake-agent-9999"}'
```

## Verification

```bash
gofmt -w *.go cmd/fake-server/*.go
go test ./...
go build -o ./bin/agent-quickstart-go .
make verify-backend
```

From the repo root, the real local smoke check is:

```bash
make doctor-local
make verify-local-go
```

## Repo Fit

- Local full-stack mode: Next rewrites proxy here through `AGENT_BACKEND_URL`
- Deployment mode: deploy this service separately and point the Next app's `AGENT_BACKEND_URL` at it
- This backend uses Gin plus the official Go Agent Server SDK, not a hand-written direct REST client
