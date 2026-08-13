package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/model"
)

func TestCapabilityProbe(t *testing.T) {
	const secret = "probe-secret"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" || request.Header.Get("Authorization") != "Bearer "+secret {
			t.Fatalf("unexpected request %s auth=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "test-model" || body["tool_choice"] == nil || body["tools"] == nil {
			t.Fatalf("missing probe requirements: %#v", body)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"type":"function","function":{"name":"cyberpilot_capability_probe","arguments":"{\"value\":\"cyberpilot-ready\"}"}}]}}]}`))
	}))
	defer server.Close()
	provider := Provider{BaseURL: server.URL + "/v1", Model: "test-model", Credential: func(context.Context) (string, error) { return secret, nil }}
	report, err := provider.Probe(context.Background())
	if err != nil || !report.ToolCalling || !report.StructuredOutput {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}

func validProposalJSON() string {
	return `{"id":"018f1f77-2f8e-7cc0-8000-000000000001","session_id":"018f1f77-2f8e-7cc0-8000-000000000002","hypothesis_id":"018f1f77-2f8e-7cc0-8000-000000000003","target":"http://127.0.0.1","purpose":"inspect local fixture","capability":"http.request","arguments":{"method":"GET"},"risk":{"level":"low","traffic_class":"single"},"expected_evidence":["response"],"timeout_seconds":10}`
}

func TestTurnNonStreamingNormalizesProposalAndUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		arguments, _ := json.Marshal(validProposalJSON())
		_, _ = response.Write([]byte(`{"choices":[{"message":{"content":"next","tool_calls":[{"type":"function","function":{"name":"cyberpilot_propose_action","arguments":` + string(arguments) + `}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":11,"completion_tokens":7}}`))
	}))
	defer server.Close()
	result, err := (Provider{BaseURL: server.URL, Model: "test"}).Turn(context.Background(), model.TurnRequest{MaxTokens: 100})
	if err != nil || len(result.Proposals) != 1 || result.Usage.InputTokens != 11 || result.FinishReason != "tool_calls" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if result.Proposals[0].ID.Validate() != nil || result.Proposals[0].Risk.Level != "low" {
		t.Fatalf("invalid normalized proposal %#v", result.Proposals[0])
	}
}

func TestTurnRepairsInvalidProposalExactlyOnce(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		calls++
		var body struct {
			Messages []model.Message `json:"messages"`
		}
		_ = json.NewDecoder(request.Body).Decode(&body)
		arguments := `{"target":"missing-identifiers"}`
		if calls == 2 {
			arguments = validProposalJSON()
			if len(body.Messages) == 0 || !strings.Contains(body.Messages[len(body.Messages)-1].Content, "previous action proposal was invalid") {
				t.Fatal("repair feedback was not sent")
			}
		}
		encoded, _ := json.Marshal(arguments)
		_, _ = response.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"type":"function","function":{"name":"cyberpilot_propose_action","arguments":` + string(encoded) + `}}]}}]}`))
	}))
	defer server.Close()
	result, err := (Provider{BaseURL: server.URL, Model: "test"}).Turn(context.Background(), model.TurnRequest{})
	if err != nil || len(result.Proposals) != 1 || calls != 2 {
		t.Fatalf("result=%#v err=%v calls=%d", result, err, calls)
	}
}

func TestTurnStreaming(t *testing.T) {
	proposal := validProposalJSON()
	cut := len(proposal) / 2
	first, _ := json.Marshal(proposal[:cut])
	second, _ := json.Marshal(proposal[cut:])
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/event-stream")
		_, _ = response.Write([]byte(`data: {"choices":[{"delta":{"content":"working","tool_calls":[{"index":0,"type":"function","function":{"name":"cyberpilot_propose_action","arguments":` + string(first) + `}}]}}]}` + "\n\n"))
		_, _ = response.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":` + string(second) + `}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"completion_tokens":6}}` + "\n\n"))
		_, _ = response.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()
	result, err := (Provider{BaseURL: server.URL, Model: "test"}).Turn(context.Background(), model.TurnRequest{Stream: true})
	if err != nil || len(result.Proposals) != 1 || result.Text != "working" || result.Usage.OutputTokens != 6 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestProposalValidationRequiresDomainIdentifiers(t *testing.T) {
	if err := validateProposal(domain.ActionProposal{}); err == nil {
		t.Fatal("expected invalid proposal")
	}
}

func TestProbeClassifiesAuthenticationWithoutCredentialLeak(t *testing.T) {
	const secret = "must-not-leak"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
		_, _ = response.Write([]byte(`{"error":{"message":"invalid API credential"}}`))
	}))
	defer server.Close()
	provider := Provider{BaseURL: server.URL, Model: "test-model", Credential: func(context.Context) (string, error) { return secret, nil }}
	_, err := provider.Probe(context.Background())
	if err == nil || !strings.Contains(err.Error(), "authentication") || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe or unclassified error: %v", err)
	}
}

func TestProbeRejectsMissingTypedOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"choices":[{"message":{"content":"ready"}}]}`))
	}))
	defer server.Close()
	_, err := (Provider{BaseURL: server.URL, Model: "test-model"}).Probe(context.Background())
	if err == nil || !strings.Contains(err.Error(), "typed tool call") {
		t.Fatalf("unexpected error %v", err)
	}
}
