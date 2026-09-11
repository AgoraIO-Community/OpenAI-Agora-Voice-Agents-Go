package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/AgoraIO/agora-agents-go/v2/agentkit"
	"github.com/AgoraIO/agora-agents-go/v2/option"
)

type wireRecorder struct {
	req  *http.Request
	body []byte
}

func (w *wireRecorder) Do(req *http.Request) (*http.Response, error) {
	w.req = req
	w.body, _ = io.ReadAll(req.Body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"agent_id":"agent-1"}`)),
		Header:     make(http.Header),
	}, nil
}

func TestOpenAIGPTLiveStandardClientPreviewWireShape(t *testing.T) {
	t.Setenv("AGORA_APP_ID", "00000000000000000000000000000000")
	t.Setenv("AGORA_APP_CERTIFICATE", "11111111111111111111111111111111")
	t.Setenv("OPENAI_API_KEY", "test-openai-api-key")

	svc, err := newAgentService()
	if err != nil {
		t.Fatal(err)
	}
	recorder := &wireRecorder{}
	svc.sessionClient = agentkit.NewAgoraClient(agentkit.AgoraClientOptions{
		Area:           option.AreaUS,
		AppID:          svc.appID,
		AppCertificate: svc.certificate,
		HTTPClient:     recorder,
	})

	if _, err := svc.start("standard-channel", 123456, 100); err != nil {
		t.Fatal(err)
	}
	if recorder.req == nil {
		t.Fatal("no request captured")
	}
	if got := recorder.req.Header.Get("agora-feature"); got != "live-models" {
		t.Fatalf("agora-feature = %q, want live-models", got)
	}
	if got := recorder.req.URL.Host; got != "partner.ai.agora.io" {
		t.Fatalf("request host = %q, want preview gateway", got)
	}

	var payload struct {
		AppID      string         `json:"appid"`
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(recorder.body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.AppID != svc.appID {
		t.Fatalf("appid = %q, want %q", payload.AppID, svc.appID)
	}
	if _, found := payload.Properties["enable_string_uid"]; found {
		t.Fatal("enable_string_uid must be omitted for numeric UIDs")
	}
	for _, field := range []string{"asr", "llm", "tts"} {
		if _, found := payload.Properties[field]; found {
			t.Fatalf("MLLM-only request must not contain %s", field)
		}
	}
	mllm, ok := payload.Properties["mllm"].(map[string]any)
	if !ok {
		t.Fatal("request has no mllm object")
	}
	want := map[string]any{
		"enable":           true,
		"vendor":           "openai_gpt_live",
		"api_key":          "test-openai-api-key",
		"url":              "wss://api.openai.com/v1/live/sessions",
		"greeting_message": defaultGreeting,
		"messages":         []any{},
		"params": map[string]any{
			"model":  "gpt-live-1",
			"voice":  "cedar",
			"prompt": defaultInstructions,
		},
	}
	wantJSON, _ := json.Marshal(want)
	gotJSON, _ := json.Marshal(mllm)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("mllm = %s, want %s", gotJSON, wantJSON)
	}
}
