# Build an OpenAI GPT Live voice agent with Go

Use Agora's Go SDK to place an OpenAI GPT Live voice agent in an Agora channel. GPT Live handles speech input, reasoning, and speech output as one MLLM stage. The included browser client publishes microphone audio and displays transcripts, agent state, and latency metrics.

| Item | Value |
| --- | --- |
| SDK | `github.com/AgoraIO/agora-agents-go/v2@v2.8.0` |
| Provider | `openai_gpt_live` |
| Model | `gpt-live-1-diamond-alpha` |
| Voice | `cedar` |
| Backend | Go and Gin |
| Web client | Next.js |

## Prerequisites

- Go 1.23 or newer
- Node.js 22 or newer
- [pnpm](https://pnpm.io/installation)
- GNU Make
- [Agora CLI](https://github.com/AgoraIO/cli)
- An Agora project with an App ID and App Certificate
- An OpenAI API key with GPT Live access

## Run the recipe

Clone the repository and install its dependencies:

```bash
git clone git@github.com:AgoraIO-Community/OpenAI-Agora-Voice-Agents-Go.git
cd OpenAI-Agora-Voice-Agents-Go
make setup
```

Use the Agora CLI to select a project and write its credentials to `server/.env`:

```bash
agora login
agora project use <your-project-name-or-id>
agora project env write server/.env --template standard
```

Add your OpenAI key to `server/.env`:

```dotenv
OPENAI_API_KEY=your_openai_api_key
```

Start the Gin backend and Next.js client:

```bash
make dev
```

Open [http://localhost:3000](http://localhost:3000), allow microphone access, and select **Start conversation**.

## Configure GPT Live

The following function is a complete SDK example. Pass the channel and UIDs used by the browser, retain the returned session, and stop it when the call ends.

```go
package main

import (
	"context"
	"os"

	"github.com/AgoraIO/agora-agents-go/v2/agentkit"
	"github.com/AgoraIO/agora-agents-go/v2/agentkit/vendors"
	"github.com/AgoraIO/agora-agents-go/v2/option"
)

func startAgent(
	ctx context.Context,
	channel string,
	agentUID string,
	userUID string,
) (string, *agentkit.AgentSession, error) {
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
	idleTimeout := 30
	expiresIn, err := agentkit.ExpiresInHours(1)
	if err != nil {
		return "", nil, err
	}

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
		Voice:           "cedar",
		Prompt:          "You are a concise and helpful voice assistant.",
		GreetingMessage: "Hello! How can I help?",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "My name is Arlene."},
			{"role": "assistant", "content": "Nice to meet you, Arlene."},
		},
	}))

	session := agent.CreateSession(agentkit.CreateSessionOptions{
		Channel:     channel,
		AgentUID:    agentUID,
		RemoteUIDs:  []string{userUID},
		IdleTimeout: &idleTimeout,
		ExpiresIn:   expiresIn,
	})
	agentID, err := session.Start(ctx)
	if err != nil {
		return "", nil, err
	}
	return agentID, session, nil
}
```

The complete sample generates an RTC+RTM token before starting the agent. The browser and agent join the same channel with different UIDs, and `RemoteUIDs` identifies the browser user whose audio the agent should process.

## Customize the conversation

| Go option | Request field | Purpose |
| --- | --- | --- |
| `Prompt` | `mllm.params.prompt` | Persistent system instructions |
| `GreetingMessage` | `mllm.greeting_message` | Requested opening line |
| `Messages` | `mllm.messages` | Prior user and assistant turns |
| `Voice` | `mllm.params.voice` | Output voice |

Use `Prompt` for system behavior and `Messages` to continue an earlier conversation. Keep credentials and conversation history on the server.

## Stop the agent

Retain the session returned by `startAgent` and stop it when the call ends:

```go
err := session.Stop(context.Background())
```

For a multi-instance deployment, store lifecycle ownership in shared state or route start and stop requests to the same instance.

## Verify the project

```bash
make test
make verify-web
```

See the [project README](../../README.md) for architecture, deployment, configuration options, and troubleshooting.
