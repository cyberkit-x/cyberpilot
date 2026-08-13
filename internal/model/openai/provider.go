package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cyberkit-x/cyberpilot/internal/domain"
	"github.com/cyberkit-x/cyberpilot/internal/model"
)

const maxResponseBytes = 4 << 20

type Provider struct {
	BaseURL    string
	Model      string
	Credential func(context.Context) (string, error)
	Client     *http.Client
}

type providerError struct {
	Status int
	Kind   string
	Detail string
}

func (e *providerError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("model provider %s error (HTTP %d)", e.Kind, e.Status)
	}
	return fmt.Sprintf("model provider %s error (HTTP %d): %s", e.Kind, e.Status, e.Detail)
}

func (p Provider) Probe(ctx context.Context) (model.CapabilityReport, error) {
	if strings.TrimSpace(p.Model) == "" {
		return model.CapabilityReport{}, errors.New("model name is required")
	}
	payload := map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "Return the probe tool call with value cyberpilot-ready. Do not answer with plain text."},
			{"role": "user", "content": "Run the capability probe."},
		},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name":        "cyberpilot_capability_probe",
				"description": "Confirm typed tool calling and structured arguments",
				"parameters":  map[string]any{"type": "object", "properties": map[string]any{"value": map[string]string{"type": "string"}}, "required": []string{"value"}, "additionalProperties": false},
				"strict":      true,
			},
		}},
		"tool_choice": map[string]any{"type": "function", "function": map[string]string{"name": "cyberpilot_capability_probe"}},
		"temperature": 0,
		"max_tokens":  128,
	}
	var response chatResponse
	if err := p.request(ctx, "/chat/completions", payload, &response); err != nil {
		return model.CapabilityReport{}, err
	}
	if len(response.Choices) == 0 || len(response.Choices[0].Message.ToolCalls) != 1 {
		return model.CapabilityReport{}, errors.New("selected model did not return the required typed tool call")
	}
	call := response.Choices[0].Message.ToolCalls[0]
	if call.Type != "function" || call.Function.Name != "cyberpilot_capability_probe" {
		return model.CapabilityReport{}, errors.New("selected model returned an unexpected tool call")
	}
	var arguments struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil || arguments.Value != "cyberpilot-ready" {
		return model.CapabilityReport{}, errors.New("selected model returned invalid structured tool arguments")
	}
	return model.CapabilityReport{Model: p.Model, ToolCalling: true, StructuredOutput: true}, nil
}

func (p Provider) Turn(ctx context.Context, turn model.TurnRequest) (model.TurnResult, error) {
	result, raw, err := p.turn(ctx, turn)
	if err == nil {
		return result, nil
	}
	var invalid *invalidProposalError
	if !errors.As(err, &invalid) || len(turn.RepairFrom) != 0 {
		return model.TurnResult{}, err
	}
	turn.RepairFrom = append(json.RawMessage(nil), raw...)
	turn.Messages = append(turn.Messages, model.Message{Role: "system", Content: "The previous action proposal was invalid: " + invalid.Error() + ". Return exactly one corrected cyberpilot_propose_action tool call."})
	result, _, err = p.turn(ctx, turn)
	return result, err
}

func (p Provider) turn(ctx context.Context, turn model.TurnRequest) (model.TurnResult, json.RawMessage, error) {
	if turn.MaxTokens <= 0 {
		turn.MaxTokens = 2048
	}
	payload := map[string]any{
		"model":       p.Model,
		"messages":    turn.Messages,
		"tools":       openAITools(turn.Tools),
		"tool_choice": "auto",
		"max_tokens":  turn.MaxTokens,
		"stream":      turn.Stream,
	}
	if turn.Stream {
		payload["stream_options"] = map[string]bool{"include_usage": true}
		response, err := p.do(ctx, "/chat/completions", payload)
		if err != nil {
			return model.TurnResult{}, nil, err
		}
		defer response.Body.Close()
		return decodeStream(response.Body)
	}
	var response chatResponse
	if err := p.request(ctx, "/chat/completions", payload, &response); err != nil {
		return model.TurnResult{}, nil, err
	}
	return normalizeResponse(response)
}

type invalidProposalError struct{ reason string }

func (e *invalidProposalError) Error() string { return e.reason }

func openAITools(tools []model.Tool) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{"type": "function", "function": map[string]any{"name": tool.Name, "description": tool.Description, "parameters": tool.Schema, "strict": true}})
	}
	return result
}

func normalizeResponse(response chatResponse) (model.TurnResult, json.RawMessage, error) {
	if len(response.Choices) == 0 {
		return model.TurnResult{}, nil, errors.New("model response contained no choices")
	}
	choice := response.Choices[0]
	result := model.TurnResult{Text: choice.Message.Content, FinishReason: choice.FinishReason, Usage: model.Usage{InputTokens: response.Usage.PromptTokens, OutputTokens: response.Usage.CompletionTokens}}
	for _, call := range choice.Message.ToolCalls {
		raw := json.RawMessage(call.Function.Arguments)
		switch call.Function.Name {
		case "cyberpilot_propose_action":
			var proposal domain.ActionProposal
			if err := json.Unmarshal(raw, &proposal); err != nil {
				return model.TurnResult{}, raw, &invalidProposalError{reason: "invalid action proposal JSON: " + err.Error()}
			}
			if err := validateProposal(proposal); err != nil {
				return model.TurnResult{}, raw, &invalidProposalError{reason: err.Error()}
			}
			result.Proposals = append(result.Proposals, proposal)
		case "cyberpilot_report_finding":
			var proposal domain.FindingProposal
			if err := json.Unmarshal(raw, &proposal); err != nil {
				return model.TurnResult{}, raw, &invalidProposalError{reason: "invalid finding proposal JSON: " + err.Error()}
			}
			result.Findings = append(result.Findings, proposal)
		case "cyberpilot_complete":
			var completion struct {
				Reason string `json:"reason"`
			}
			if err := json.Unmarshal(raw, &completion); err != nil || strings.TrimSpace(completion.Reason) == "" {
				return model.TurnResult{}, raw, &invalidProposalError{reason: "completion requires a reason"}
			}
			result.Complete, result.Reason = true, strings.TrimSpace(completion.Reason)
		}
	}
	return result, nil, nil
}

func validateProposal(proposal domain.ActionProposal) error {
	if proposal.ID.Validate() != nil || proposal.SessionID.Validate() != nil || proposal.HypothesisID.Validate() != nil {
		return errors.New("action proposal requires valid action, session, and hypothesis IDs")
	}
	if strings.TrimSpace(proposal.Target) == "" || strings.TrimSpace(proposal.Purpose) == "" || strings.TrimSpace(proposal.Capability) == "" {
		return errors.New("action proposal requires target, purpose, and capability")
	}
	if proposal.TimeoutSeconds <= 0 {
		return errors.New("action proposal requires a positive timeout")
	}
	return nil
}

func decodeStream(body io.Reader) (model.TurnResult, json.RawMessage, error) {
	var assembled chatResponse
	assembled.Choices = make([]chatChoice, 1)
	toolArguments := map[int]*chatToolCall{}
	scanner := bufio.NewScanner(io.LimitReader(body, maxResponseBytes+1))
	scanner.Buffer(make([]byte, 64*1024), maxResponseBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return model.TurnResult{}, nil, fmt.Errorf("decode model stream: %w", err)
		}
		assembled.Usage = chunk.Usage
		for _, choice := range chunk.Choices {
			assembled.Choices[0].Message.Content += choice.Delta.Content
			if choice.FinishReason != "" {
				assembled.Choices[0].FinishReason = choice.FinishReason
			}
			for _, call := range choice.Delta.ToolCalls {
				current := toolArguments[call.Index]
				if current == nil {
					current = &chatToolCall{}
					toolArguments[call.Index] = current
				}
				if call.Type != "" {
					current.Type = call.Type
				}
				if call.Function.Name != "" {
					current.Function.Name += call.Function.Name
				}
				current.Function.Arguments += call.Function.Arguments
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return model.TurnResult{}, nil, err
	}
	for index := 0; index < len(toolArguments); index++ {
		if call := toolArguments[index]; call != nil {
			assembled.Choices[0].Message.ToolCalls = append(assembled.Choices[0].Message.ToolCalls, *call)
		}
	}
	return normalizeResponse(assembled)
}

func (p Provider) request(ctx context.Context, path string, input, output any) error {
	response, err := p.do(ctx, path, input)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxResponseBytes {
		return errors.New("model response exceeds size limit")
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode model response: %w", err)
	}
	return nil
}

func (p Provider) do(ctx context.Context, path string, input any) (*http.Response, error) {
	base, err := url.Parse(p.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("valid model base URL is required")
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	body, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if p.Credential != nil {
		secret, err := p.Credential(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve model credential: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+secret)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("reach model endpoint: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		kind := "request"
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			kind = "authentication"
		case http.StatusNotFound:
			kind = "model-or-endpoint"
		case http.StatusTooManyRequests:
			kind = "rate-limit"
		default:
			if response.StatusCode >= 500 {
				kind = "unavailable"
			}
		}
		return nil, &providerError{Status: response.StatusCode, Kind: kind, Detail: safeProviderDetail(data)}
	}
	return response, nil
}

func safeProviderDetail(data []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &envelope) == nil && envelope.Error.Message != "" {
		if len(envelope.Error.Message) > 512 {
			return envelope.Error.Message[:512]
		}
		return envelope.Error.Message
	}
	return http.StatusText(http.StatusBadRequest)
}

type chatToolCall struct {
	Index    int    `json:"index,omitempty"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatMessage struct {
	Content   string         `json:"content"`
	ToolCalls []chatToolCall `json:"tool_calls"`
}

type chatChoice struct {
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type tokenUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   tokenUsage   `json:"usage"`
}

type streamChunk struct {
	Choices []struct {
		Delta        chatMessage `json:"delta"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage tokenUsage `json:"usage"`
}
