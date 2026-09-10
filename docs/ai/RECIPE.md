# Recipe card: OpenAI GPT Live with the Agora Go SDK

Use this recipe when your Go backend should run an end-to-end GPT Live voice agent in an Agora channel.

| Item | Value |
| --- | --- |
| SDK | `github.com/AgoraIO/agora-agents-go/v2@v2.8.0` |
| Provider | `openai_gpt_live` |
| Model | `gpt-live-1-diamond-alpha` |
| Voice | `cedar` |
| Data channel | RTM |
| Agent pipeline | MLLM only |

## Install

```bash
go get github.com/AgoraIO/agora-agents-go/v2@v2.8.0
```

## Configure credentials

```dotenv
AGORA_APP_ID=your_agora_app_id
AGORA_APP_CERTIFICATE=your_agora_app_certificate
OPENAI_API_KEY=your_openai_api_key
```

Keep all three values on the server.

## Create the agent

```go
client := agentkit.NewAgoraClient(agentkit.AgoraClientOptions{
	Area:           option.AreaUS,
	AppID:          os.Getenv("AGORA_APP_ID"),
	AppCertificate: os.Getenv("AGORA_APP_CERTIFICATE"),
})

enableRTM := true
enableErrors := true
enableMetrics := true
enableTools := false
dataChannel := agentkit.ParametersDataChannel("rtm")

agent := agentkit.NewAgent(
	client,
	agentkit.WithAdvancedFeatures(&agentkit.AdvancedFeatures{
		EnableRtm:   &enableRTM,
		EnableTools: &enableTools,
	}),
	agentkit.WithParameters(&agentkit.SessionParams{
		DataChannel:        &dataChannel,
		EnableErrorMessage: &enableErrors,
		EnableMetrics:      &enableMetrics,
	}),
	agentkit.WithAudioScenario(agentkit.ParametersAudioScenario("chorus")),
).WithMllm(vendors.NewOpenAIGPTLive(vendors.OpenAIGPTLiveOptions{
	APIKey:          os.Getenv("OPENAI_API_KEY"),
	Model:           "gpt-live-1-diamond-alpha",
	AlphaSelector:   "quicksilver=v3",
	Voice:           "cedar",
	Prompt:          "You are a concise and helpful voice assistant.",
	GreetingMessage: "Hello! How can I help?",
	Messages: []map[string]interface{}{
		{"role": "user", "content": "My name is Arlene."},
		{"role": "assistant", "content": "Nice to meet you, Arlene."},
	},
}))

idleTimeout := 30
session := agent.CreateSession(agentkit.CreateSessionOptions{
	Channel:     "your-channel",
	AgentUID:    "123456",
	RemoteUIDs:  []string{"your-user-uid"},
	IdleTimeout: &idleTimeout,
})

agentID, err := session.Start(context.Background())
```

The snippet uses `context`, `os`, `agentkit`, `agentkit/vendors`, and `option`. Handle the returned error before using `agentID`.

Generate the RTC+RTM token before starting the session. The browser and agent must join the same channel with different UIDs. Set `RemoteUIDs` to the browser user's UID so the agent processes that user's audio.

## Parameter map

| Go option | Request field | Purpose |
| --- | --- | --- |
| `Prompt` | `mllm.params.prompt` | Persistent system instructions |
| `GreetingMessage` | `mllm.greeting_message` | Opening line |
| `Messages` | `mllm.messages` | Prior user and assistant turns |
| `Voice` | `mllm.params.voice` | Output voice |
| `AlphaSelector` | `mllm.params.alpha_selector` | Selects the GPT Live v3 contract |

Use `Messages` for prior conversation. Keep system behavior in `Prompt`.

## Stop the session

Retain the session object with its returned agent ID, then stop it through the same object:

```go
err := session.Stop(context.Background())
```

For a multi-instance backend, store lifecycle ownership in shared state or route start and stop requests to the same instance.

## Try the complete sample

Return to the [project README](../../README.md) for credential setup, local run commands, browser UI, and troubleshooting.
