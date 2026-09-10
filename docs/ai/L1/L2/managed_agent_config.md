# OpenAI GPT Live Agent Config

> **When to Read This:** Load this document when changing the Go preview agent, greeting, credentials, or session options.

The backend integration lives in `server/agent.go`. `newAgentService` reads the Agora credentials and `OPENAI_API_KEY`, then constructs the standard client. It detects the OpenAI GPT Live provider and routes start through the preview endpoint.

```go
client := agentkit.NewAgoraClient(agentkit.AgoraClientOptions{
    Area:           option.AreaUS,
    AppID:          appID,
    AppCertificate: certificate,
})

model, err := vendors.NewOpenAIGPTLive(vendors.OpenAIGPTLiveOptions{
    APIKey:          openAIAPIKey,
    Model:           "gpt-live-1-diamond-alpha",
    AlphaSelector:   "quicksilver=v3",
    Voice:           "cedar",
    Prompt:          instructions,
    GreetingMessage: greeting,
})
if err != nil { return nil, err }

agent := agentkit.NewAgent(
    client,
    agentkit.WithAdvancedFeatures(...),
    agentkit.WithParameters(...),
).WithMllm(model)
```

`NewOpenAIGPTLive` emits `mllm.vendor: "openai_gpt_live"`, `wss://api.openai.com/v1/live/sessions`, `params.alpha_selector: "quicksilver=v3"`, and `greeting_message` for the opening line. The selector makes the required v3 OpenAI alpha handshake explicit. Do not add `WithStt`, `WithLlm`, or `WithTts` to this demo.

Required server environment:

```bash
AGORA_APP_ID=...
AGORA_APP_CERTIFICATE=...
OPENAI_API_KEY=...
```

`AGENT_GREETING` and `AGENT_INSTRUCTIONS` are optional. Instructions are persistent behavior guidance; the greeting is only the first spoken turn. `AGENT_PRIOR_MESSAGES` accepts a JSON array of user/assistant text turns and is sent as `mllm.messages`, separately from the prompt. Session UIDs remain strings on the wire and `RemoteUIDs` remains an array. Started sessions are stored in a process-local map by agent ID. Stop removes and calls that retained session; unknown IDs are idempotent success and there is no standalone `StopAgent` fallback.

Run `GOCACHE=/private/tmp/openai-mllm-demo-gocache go test ./...` in `server/` and `make build` at the repo root after changes.
