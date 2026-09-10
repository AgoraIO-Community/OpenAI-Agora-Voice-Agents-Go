# 06 Interfaces

> Boundary contracts: Gin routes, Next rewrites, environment variables, and managed agent payload.

## Go Backend Routes

`server/main.go` registers these on a Gin router:

| Path           | Method | Request                                                         | Success (200)                                                                                | Errors                                                  |
| -------------- | ------ | --------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `/get_config`  | GET    | Query: optional `channel`, optional `uid`                       | `{ "code": 0, "msg": "success", "data": { app_id, token, uid, channel_name, agent_uid } }`   | `400` invalid uid; `500` `service == nil`; `toHTTPError` |
| `/startAgent`  | POST   | JSON `{ channelName, rtcUid, userUid }` (`startAgentRequest`)   | `{ "code": 0, "msg": "success", "data": { agent_id, channel_name, status } }`                | `400` invalid JSON or validation error; `500` other service errors |
| `/stopAgent`   | POST   | JSON `{ agentId }` (`stopAgentRequest`)                          | `{ "code": 0, "msg": "success" }`                                                            | `400` invalid JSON or validation error; `500` other service errors |

All routes go through `cors.New(...)` with `AllowAllOrigins: true` and methods `GET`/`POST`/`OPTIONS`.

`generateConfig` treats missing, zero, and negative UIDs as "generate a random user UID" and returns the generated value. This keeps the single RTC+RTM token usable for RTM, where `0` is not a valid login subject.

Error responses use `{ "detail": "..." }`, not the success envelope.

## Next.js Rewrites

`client/next.config.ts` registers these only when `AGENT_BACKEND_URL` is set:

| Source              | Destination                                  |
| ------------------- | -------------------------------------------- |
| `/api/get_config`   | `${AGENT_BACKEND_URL}/get_config`             |
| `/api/startAgent`   | `${AGENT_BACKEND_URL}/startAgent`             |
| `/api/stopAgent`    | `${AGENT_BACKEND_URL}/stopAgent`              |

`verify-api-contracts.ts` asserts that no `client/app/api/**/route.ts` files exist. Adding one would create a competing handler in front of the rewrite — don't.

## Environment Variables

| Scope                | Variable                                  |
| -------------------- | ----------------------------------------- |
| Go server (required) | `AGORA_APP_ID`, `AGORA_APP_CERTIFICATE`   |
| Go server (required) | `OPENAI_API_KEY`                          |
| Go server (optional) | `AGENT_GREETING`, `AGENT_INSTRUCTIONS`, `AGENT_PRIOR_MESSAGES`, `PORT` |
| Next build           | `AGENT_BACKEND_URL`                       |
| Browser              | `NEXT_PUBLIC_AGENT_UID` (optional override) |

`AGENT_BACKEND_URL` is a Next **server**-time env var (used inside `next.config.ts`), not a `NEXT_PUBLIC_*` value — do not prefix it.

## Token Shape

`generateConfig` returns:

```json
{
  "app_id": "string",
  "token": "string",                // built by agentkit.GenerateConvoAIToken
  "uid": "string",                  // serialized number
  "channel_name": "string",
  "agent_uid": "string"
}
```

Token expiry is `agentkit.ExpiresInHours(1)`. The same token grants RTC and RTM privileges.

## Managed Agent Payload

The Go server does not POST a hand-written JSON payload to Agora — it uses the SDK builder chain in `server/agent.go`:

```go
agent := agentkit.NewAgent(
    /* prompt, VAD, greeting, maxHistory via WithName/WithInstructions/etc. */
    agentkit.WithAdvancedFeatures(&agentkit.AdvancedFeatures{
        EnableRtm:   &enableRTM,
        EnableTools: &enableTools,
    }),
    agentkit.WithParameters(&agentkit.SessionParams{
        DataChannel:        &dataChannel,        // "rtm"
        EnableErrorMessage: &enableErrorMessage, // true
        EnableMetrics:      &enableMetrics,      // true
    }),
).
    WithMllm(vendors.NewOpenAIGPTLive(...))

session := agent.CreateSession(s.sessionClient, agentkit.CreateSessionOptions{
    Channel:         channelName,
    AgentUID:        strconv.Itoa(agentUID),
    RemoteUIDs:      []string{strconv.Itoa(userUID)},
    EnableStringUID: &enableStringUID,
    IdleTimeout:     &idleTimeout,
    ExpiresIn:       expiresIn,
})
agentID, err := session.Start(ctx)
```

## RTM Event Shapes (Client-Side)

`AgoraVoiceAI` emits the same toolkit events as the Next.js quickstart:

- `TRANSCRIPT_UPDATED` — `{ uid, text, status, timestamp }[]`
- `AGENT_STATE_CHANGED` — `AgentState`
- `AGENT_METRICS` — `{ type, name, value, timestamp }`
- `MESSAGE_ERROR` — `{ module, code, message, send_ts }`
- `MESSAGE_SAL_STATUS` — `{ status, timestamp }`
- `AGENT_ERROR` — SDK error info

`ConversationComponent.tsx` also attaches a raw RTM `message` listener as a fallback for the same `message.error` / `message.sal_status` JSON payloads.

## Internal Types

| Type                          | Lives in                                       | Notes                                            |
| ----------------------------- | ---------------------------------------------- | ------------------------------------------------ |
| `startAgentRequest`           | `server/main.go`                               | `channelName`, `rtcUid`, `userUid`               |
| `stopAgentRequest`            | `server/main.go`                               | `agentId`                                        |
| `configData`                  | `server/agent.go`                              | snake_case JSON tags                              |
| `startAgentResult`            | `server/agent.go`                              | `agent_id`, `channel_name`, `status`             |
| `AgoraTokenData`              | `client/src/types/conversation.ts`             | Used by `LandingPage` + `ConversationComponent`  |
| `AgoraRenewalTokens`          | `client/src/types/conversation.ts`             | Renewal handler payload                          |
| `ConversationComponentProps`  | `client/src/types/conversation.ts`             | Includes RTM client + data                        |

## Related Deep Dives

- [Managed Agent Config](L2/managed_agent_config.md) — Detailed field reference.
- [Verification Scripts](L2/verification_scripts.md) — How the contracts above are enforced in CI.

### GPT Live v3 model contract

The MLLM request uses `params.model: gpt-live-1-diamond-alpha`, `params.alpha_selector: quicksilver=v3`, `params.voice: cedar`, and `params.prompt` from `AGENT_INSTRUCTIONS`. Without an override, the prompt and greeting use the built-in Ada developer advocate persona. The SDK supplies `/v1/live/sessions`; the explicit selector ensures the worker sends the required v3 OpenAI alpha header. Browser API contracts are unchanged.

`AGENT_PRIOR_MESSAGES` is parsed as a JSON array of user/assistant text turns and sent as `mllm.messages`. It is independent of `AGENT_INSTRUCTIONS`; invalid history prevents backend startup.
