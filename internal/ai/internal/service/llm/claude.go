package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/KNICEX/InkFlow/internal/ai/internal/domain"
	"github.com/KNICEX/InkFlow/internal/ai/internal/service"
	"io"
	"net/http"
	"strings"
)

const (
	defaultClaudeMaxTokens = 1024
	claudeAPIVersion       = "2023-06-01"
)

type ClaudeService struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	maxTokens  int
}

type ClaudeSession struct {
	svc      *ClaudeService
	messages []claudeMessage
}

type ClaudeOption func(*ClaudeService)

func WithClaudeMaxTokens(n int) ClaudeOption {
	return func(s *ClaudeService) {
		s.maxTokens = n
	}
}

func NewClaudeService(httpClient *http.Client, baseURL, apiKey, model string, opts ...ClaudeOption) service.LLMService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	svc := &ClaudeService{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
		maxTokens:  defaultClaudeMaxTokens,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

func (svc *ClaudeService) AskOnce(ctx context.Context, question string) (domain.Resp, error) {
	resp, _, err := svc.messages(ctx, []claudeMessage{
		{Role: "user", Content: question},
	})
	return resp, err
}

func (svc *ClaudeService) BeginChat(ctx context.Context) (service.LLMSession, error) {
	return &ClaudeSession{
		svc:      svc,
		messages: make([]claudeMessage, 0, 8),
	}, nil
}

func (s *ClaudeSession) Ask(ctx context.Context, question string) (domain.Resp, error) {
	s.messages = append(s.messages, claudeMessage{Role: "user", Content: question})
	resp, assistantMsg, err := s.svc.messages(ctx, s.messages)
	if err != nil {
		return domain.Resp{}, err
	}
	s.messages = append(s.messages, assistantMsg)
	return resp, nil
}

func (s *ClaudeSession) Close() error {
	return nil
}

type claudeMessagesRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeMessagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (svc *ClaudeService) messages(ctx context.Context, messages []claudeMessage) (domain.Resp, claudeMessage, error) {
	if strings.TrimSpace(svc.apiKey) == "" {
		return domain.Resp{}, claudeMessage{}, errors.New("claude api key is empty")
	}
	if strings.TrimSpace(svc.model) == "" {
		return domain.Resp{}, claudeMessage{}, errors.New("claude model is empty")
	}

	reqBody, err := json.Marshal(claudeMessagesRequest{
		Model:     svc.model,
		MaxTokens: svc.maxTokens,
		Messages:  messages,
	})
	if err != nil {
		return domain.Resp{}, claudeMessage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(svc.baseURL, "/v1/messages"), bytes.NewReader(reqBody))
	if err != nil {
		return domain.Resp{}, claudeMessage{}, err
	}
	req.Header.Set("x-api-key", svc.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := svc.httpClient.Do(req)
	if err != nil {
		return domain.Resp{}, claudeMessage{}, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 4<<20))
	if err != nil {
		return domain.Resp{}, claudeMessage{}, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return domain.Resp{}, claudeMessage{}, fmt.Errorf("claude http status=%d body=%s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var resp claudeMessagesResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return domain.Resp{}, claudeMessage{}, err
	}
	if resp.Error != nil {
		return domain.Resp{}, claudeMessage{}, fmt.Errorf("claude error: %s", resp.Error.Message)
	}

	var content strings.Builder
	for _, blk := range resp.Content {
		if blk.Type != "text" {
			continue
		}
		content.WriteString(blk.Text)
	}
	contentStr := content.String()
	assistantMsg := claudeMessage{Role: "assistant", Content: contentStr}

	return domain.Resp{
		Content: contentStr,
		Token:   resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}, assistantMsg, nil
}

