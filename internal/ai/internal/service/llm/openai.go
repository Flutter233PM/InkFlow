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

const defaultOpenAIMaxTokens = 1024

type OpenAIService struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	maxTokens  int
}

type OpenAISession struct {
	svc      *OpenAIService
	messages []openAIMessage
}

type OpenAIOption func(*OpenAIService)

func WithOpenAIMaxTokens(n int) OpenAIOption {
	return func(s *OpenAIService) {
		s.maxTokens = n
	}
}

func NewOpenAIService(httpClient *http.Client, baseURL, apiKey, model string, opts ...OpenAIOption) service.LLMService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	svc := &OpenAIService{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
		maxTokens:  defaultOpenAIMaxTokens,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

func (svc *OpenAIService) AskOnce(ctx context.Context, question string) (domain.Resp, error) {
	resp, _, err := svc.chatCompletion(ctx, []openAIMessage{
		{Role: "user", Content: question},
	})
	return resp, err
}

func (svc *OpenAIService) BeginChat(ctx context.Context) (service.LLMSession, error) {
	return &OpenAISession{
		svc:      svc,
		messages: make([]openAIMessage, 0, 8),
	}, nil
}

func (s *OpenAISession) Ask(ctx context.Context, question string) (domain.Resp, error) {
	s.messages = append(s.messages, openAIMessage{Role: "user", Content: question})
	resp, assistantMsg, err := s.svc.chatCompletion(ctx, s.messages)
	if err != nil {
		return domain.Resp{}, err
	}
	s.messages = append(s.messages, assistantMsg)
	return resp, nil
}

func (s *OpenAISession) Close() error {
	return nil
}

type openAIChatCompletionRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int64 `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (svc *OpenAIService) chatCompletion(ctx context.Context, messages []openAIMessage) (domain.Resp, openAIMessage, error) {
	if strings.TrimSpace(svc.apiKey) == "" {
		return domain.Resp{}, openAIMessage{}, errors.New("openai api key is empty")
	}
	if strings.TrimSpace(svc.model) == "" {
		return domain.Resp{}, openAIMessage{}, errors.New("openai model is empty")
	}

	reqBody, err := json.Marshal(openAIChatCompletionRequest{
		Model:     svc.model,
		Messages:  messages,
		MaxTokens: svc.maxTokens,
	})
	if err != nil {
		return domain.Resp{}, openAIMessage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(svc.baseURL, "/chat/completions"), bytes.NewReader(reqBody))
	if err != nil {
		return domain.Resp{}, openAIMessage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+svc.apiKey)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := svc.httpClient.Do(req)
	if err != nil {
		return domain.Resp{}, openAIMessage{}, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResp.Body, 4<<20))
	if err != nil {
		return domain.Resp{}, openAIMessage{}, err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return domain.Resp{}, openAIMessage{}, fmt.Errorf("openai http status=%d body=%s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var resp openAIChatCompletionResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return domain.Resp{}, openAIMessage{}, err
	}
	if resp.Error != nil {
		return domain.Resp{}, openAIMessage{}, fmt.Errorf("openai error: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return domain.Resp{}, openAIMessage{}, errors.New("openai: empty choices")
	}

	content := resp.Choices[0].Message.Content
	assistantMsg := openAIMessage{Role: "assistant", Content: content}
	return domain.Resp{
		Content: content,
		Token:   resp.Usage.TotalTokens,
	}, assistantMsg, nil
}

func joinURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	path = strings.TrimLeft(path, "/")
	return base + "/" + path
}

