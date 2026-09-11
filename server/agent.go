package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AgoraIO/agora-agents-go/v2/agentkit"
	"github.com/AgoraIO/agora-agents-go/v2/agentkit/vendors"
	"github.com/AgoraIO/agora-agents-go/v2/option"
)

const (
	defaultGreeting     = "Hi there! I'm Ada, your virtual assistant from Agora. How can I help?"
	defaultInstructions = `You are **Ada**, an agentic developer advocate from **Agora**. You help developers understand and build with Agora's Conversational AI platform.

# What Agora Actually Is
Agora is a real-time communications company. The product you represent is the **Agora Conversational AI Engine**. It lets developers add voice AI agents to apps over Agora's SD-RTN (Software Defined Real-Time Network). Key facts:
- The product is called the **Conversational AI Engine** (not "Chorus", not "Harmony", or any other name you might invent)
- It supports both cascaded and multimodal model pipelines
- A cascaded pipeline connects separate ASR, LLM, and TTS providers
- An MLLM pipeline uses a multimodal large language model that handles audio input, reasoning, and audio output as one end-to-end stage
- This demo uses OpenAI GPT Live as an MLLM, so GPT Live receives the caller's audio directly and returns spoken audio without separate ASR or TTS stages
- MLLM providers use the mllm configuration; prompt supplies persistent instructions, greeting_message supplies the opening line, and messages seeds prior conversation
- Enabling MLLM disables separate ASR, LLM, and TTS stages because the multimodal model owns the end-to-end voice path
- Agora supports MLLM integrations for OpenAI Realtime, Azure OpenAI Realtime, Google Gemini Live, Gemini Live on Vertex AI, and xAI Grok
- It supports Deepgram, Microsoft, and others for ASR; OpenAI, Anthropic, and others for LLM; ElevenLabs, Microsoft, and others for TTS
- Agora's SD-RTN is its global real-time network infrastructure — not "SDRTN"
- MCP in this context means **Model Context Protocol** (Anthropic's open standard for connecting AI models to tools/data), not "multi-channel processing"
- Agora does not have a product called Chorus, Harmony, or any similar name — do not invent product names

# What You Are Running
- You are an MLLM voice agent powered by OpenAI GPT Live through Agora's Conversational AI Engine
- Your Agora MLLM vendor is openai_gpt_live, your model is gpt-live-1-diamond-alpha, and your configured voice is Cedar
- The caller's RTC audio travels through Agora to GPT Live; GPT Live understands the audio and produces spoken audio directly; Agora returns that audio to the caller
- You do not use a separate speech recognizer, text-only language model, or text-to-speech provider for your replies
- Your prompt defines your persistent behavior, your greeting is the opening line requested when the session starts, and messages can seed prior user and assistant turns
- This app enables RTM so the browser can receive transcript, agent state, metric, and error events alongside the RTC audio conversation
- If asked how you work, describe this MLLM path accurately and do not claim that you run a cascaded ASR, LLM, and TTS pipeline

# Honesty Rule
If you don't know a specific fact about Agora, say so plainly and suggest checking docs.agora.io. Never invent product names, feature names, or capabilities.

# Persona & Tone
- Friendly, technically credible, concise. You're a peer who builds things, not a support agent.
- Plain English. No marketing fluff.

# Core Behavior Guidelines
- **Default to brief**: This is a voice conversation. Keep most replies to 1–2 sentences. Only go longer if the user explicitly asks for detail or the answer genuinely requires it.
- **Never list or enumerate**: No bullet points, no numbered steps. Say the single most important thing.
- **Clarify before answering**: For anything complex, ask one focused question first.
- **Ask at most one question per turn**: Never stack questions.
- **Guide, don't lecture**: Unlock the next step, not everything at once.`
)

type agentService struct {
	appID         string
	certificate   string
	greeting      string
	instructions  string
	priorMessages []map[string]interface{}
	openAIAPIKey  string
	sessionClient *agentkit.AgoraClient

	mu       sync.Mutex
	sessions map[string]sessionStopper
}

type sessionStopper interface {
	Stop(ctx context.Context) error
}

type configData struct {
	AppID       string `json:"app_id"`
	Token       string `json:"token"`
	UID         string `json:"uid"`
	ChannelName string `json:"channel_name"`
	AgentUID    string `json:"agent_uid"`
}

type startAgentResult struct {
	AgentID     string `json:"agent_id"`
	ChannelName string `json:"channel_name"`
	Status      string `json:"status"`
}

func newAgentService() (*agentService, error) {
	appID := strings.TrimSpace(os.Getenv("AGORA_APP_ID"))
	certificate := strings.TrimSpace(os.Getenv("AGORA_APP_CERTIFICATE"))
	if appID == "" || certificate == "" {
		return nil, errors.New("AGORA_APP_ID and AGORA_APP_CERTIFICATE are required")
	}
	openAIAPIKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if openAIAPIKey == "" {
		return nil, errors.New("OPENAI_API_KEY is required for OpenAI GPT Live")
	}

	priorMessages, err := readPriorMessages(os.Getenv("AGENT_PRIOR_MESSAGES"))
	if err != nil {
		return nil, err
	}
	agoraClient := agentkit.NewAgoraClient(agentkit.AgoraClientOptions{
		Area:           option.AreaUS,
		AppID:          appID,
		AppCertificate: certificate,
	})

	return &agentService{
		appID:       appID,
		certificate: certificate,
		greeting: firstNonEmpty(
			strings.TrimSpace(os.Getenv("AGENT_GREETING")),
			defaultGreeting,
		),
		instructions: firstNonEmpty(
			strings.TrimSpace(os.Getenv("AGENT_INSTRUCTIONS")),
			defaultInstructions,
		),
		priorMessages: priorMessages,
		openAIAPIKey:  openAIAPIKey,
		sessionClient: agoraClient,
		sessions:      make(map[string]sessionStopper),
	}, nil
}

func (s *agentService) generateConfig(channel string, uid int) (*configData, error) {
	userUID := uid
	if userUID <= 0 {
		userUID = randomInt(1000, 9999999)
	}

	channelName := strings.TrimSpace(channel)
	if channelName == "" {
		channelName = generateChannelName()
	}

	agentUID := randomInt(10000000, 99999999)
	expiry, err := agentkit.ExpiresInHours(1)
	if err != nil {
		return nil, fmt.Errorf("resolve token expiry: %w", err)
	}

	token, err := agentkit.GenerateConvoAIToken(agentkit.GenerateConvoAITokenOptions{
		AppID:          s.appID,
		AppCertificate: s.certificate,
		ChannelName:    channelName,
		UID:            userUID,
		TokenExpire:    expiry,
	})
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &configData{
		AppID:       s.appID,
		Token:       token,
		UID:         strconv.Itoa(userUID),
		ChannelName: channelName,
		AgentUID:    strconv.Itoa(agentUID),
	}, nil
}

func (s *agentService) start(channelName string, agentUID, userUID int) (*startAgentResult, error) {
	channelName = strings.TrimSpace(channelName)
	if channelName == "" {
		return nil, errors.New("channel_name is required and cannot be empty")
	}
	if agentUID <= 0 {
		return nil, errors.New("agent_uid is required and cannot be empty")
	}
	if userUID <= 0 {
		return nil, errors.New("user_uid is required and cannot be empty")
	}

	expiresIn, err := agentkit.ExpiresInHours(1)
	if err != nil {
		return nil, fmt.Errorf("resolve session expiry: %w", err)
	}

	enableRTM := true
	enableTools := false
	enableErrorMessage := true
	enableMetrics := true
	dataChannel := agentkit.ParametersDataChannel("rtm")
	idleTimeout := 30

	agent := agentkit.NewAgent(
		s.sessionClient,
		agentkit.WithAdvancedFeatures(&agentkit.AdvancedFeatures{
			EnableRtm:   &enableRTM,
			EnableTools: &enableTools,
		}),
		agentkit.WithParameters(&agentkit.SessionParams{
			DataChannel:        &dataChannel,
			EnableErrorMessage: &enableErrorMessage,
			EnableMetrics:      &enableMetrics,
		}),
		// web client → ultra-low-latency chorus profile
		agentkit.WithAudioScenario(agentkit.ParametersAudioScenario("chorus")),
	).
		WithMllm(vendors.NewOpenAIGPTLive(vendors.OpenAIGPTLiveOptions{
			APIKey:          s.openAIAPIKey,
			Model:           "gpt-live-1-diamond-alpha",
			Voice:           "cedar",
			Prompt:          s.instructions,
			GreetingMessage: s.greeting,
			Messages:        s.priorMessages,
		}))

	session := agent.CreateSession(agentkit.CreateSessionOptions{
		Channel:     channelName,
		AgentUID:    strconv.Itoa(agentUID),
		RemoteUIDs:  []string{strconv.Itoa(userUID)},
		IdleTimeout: &idleTimeout,
		ExpiresIn:   expiresIn,
		Debug:       true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agentID, err := session.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("start agent: %w", err)
	}

	s.mu.Lock()
	s.sessions[agentID] = session
	s.mu.Unlock()

	return &startAgentResult{
		AgentID:     agentID,
		ChannelName: channelName,
		Status:      "started",
	}, nil
}

func (s *agentService) stop(agentID string) error {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return errors.New("agent_id is required and cannot be empty")
	}

	s.mu.Lock()
	session, ok := s.sessions[agentID]
	if ok {
		delete(s.sessions, agentID)
	}
	s.mu.Unlock()

	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := session.Stop(ctx); err != nil {
		return fmt.Errorf("stop agent session: %w", err)
	}

	return nil
}

func readPriorMessages(raw string) ([]map[string]interface{}, error) {
	messages := make([]map[string]interface{}, 0)
	if strings.TrimSpace(raw) == "" {
		return messages, nil
	}
	if err := json.Unmarshal([]byte(raw), &messages); err != nil {
		return nil, errors.New("AGENT_PRIOR_MESSAGES must be a JSON array of user/assistant messages with string content")
	}
	parsed := make([]map[string]interface{}, 0, len(messages))
	for _, message := range messages {
		role, roleOK := message["role"].(string)
		content, contentOK := message["content"].(string)
		if !roleOK || (role != "user" && role != "assistant") || !contentOK {
			return nil, errors.New("AGENT_PRIOR_MESSAGES must be a JSON array of user/assistant messages with string content")
		}
		parsed = append(parsed, map[string]interface{}{"role": role, "content": content})
	}
	return parsed, nil
}

func generateChannelName() string {
	return fmt.Sprintf("ai-conversation-%d-%d", time.Now().Unix(), randomInt(1000, 9999))
}

func randomInt(minInclusive, maxInclusive int) int {
	if maxInclusive <= minInclusive {
		return minInclusive
	}
	return minInclusive + rand.Intn(maxInclusive-minInclusive+1)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
