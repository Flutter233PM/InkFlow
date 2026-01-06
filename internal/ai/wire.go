package ai

import (
	"context"
	"fmt"
	"github.com/KNICEX/InkFlow/internal/ai/internal/service"
	"github.com/KNICEX/InkFlow/internal/ai/internal/service/llm"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"net/http"
	"strings"
	"time"
)

const (
	defaultGeminiModel = "gemini-2.0-flash"
	defaultHTTPTimeout = 60 * time.Second
)

func InitLLMService(cfg LLMConfig) LLMService {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		switch {
		case len(cfg.Gemini.Key) > 0:
			provider = LLMProviderGemini
		case len(cfg.OpenAI.Key) > 0:
			provider = LLMProviderOpenAI
		case len(cfg.Claude.Key) > 0:
			provider = LLMProviderClaude
		}
	}

	var svcs []LLMService
	switch provider {
	case LLMProviderGemini:
		svcs = initGemini(cfg.Gemini)
	case LLMProviderOpenAI:
		svcs = initOpenAI(cfg.OpenAI)
	case LLMProviderClaude:
		svcs = initClaude(cfg.Claude)
	default:
		panic(fmt.Errorf("unknown llm.provider=%q", cfg.Provider))
	}

	if len(svcs) == 0 {
		panic(fmt.Errorf("llm provider %q has no configured keys", provider))
	}
	if len(svcs) == 1 {
		return svcs[0]
	}
	return service.NewFailoverService(svcs)
}

func initGemini(cfg GeminiConfig) []LLMService {
	if len(cfg.Key) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultGeminiModel
	}

	var endpointOpt option.ClientOption
	if strings.TrimSpace(cfg.BaseURL) != "" {
		endpointOpt = option.WithEndpoint(normalizeEndpoint(cfg.BaseURL))
	}

	svcs := make([]LLMService, 0, len(cfg.Key))
	for _, key := range cfg.Key {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		opts := []option.ClientOption{option.WithAPIKey(key)}
		if endpointOpt != nil {
			opts = append(opts, endpointOpt)
		}

		cli, err := genai.NewClient(ctx, opts...)
		if err != nil {
			panic(err)
		}

		svcs = append(svcs, llm.NewGeminiService(cli, llm.WithModel(model)))
	}
	return svcs
}

func initOpenAI(cfg OpenAIConfig) []LLMService {
	if len(cfg.Key) == 0 {
		return nil
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		panic(fmt.Errorf("llm.openai.model is empty"))
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)

	httpClient := &http.Client{Timeout: defaultHTTPTimeout}

	svcs := make([]LLMService, 0, len(cfg.Key))
	for _, key := range cfg.Key {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		svcs = append(svcs, llm.NewOpenAIService(httpClient, baseURL, key, model))
	}
	return svcs
}

func initClaude(cfg ClaudeConfig) []LLMService {
	if len(cfg.Key) == 0 {
		return nil
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		panic(fmt.Errorf("llm.claude.model is empty"))
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)

	httpClient := &http.Client{Timeout: defaultHTTPTimeout}

	svcs := make([]LLMService, 0, len(cfg.Key))
	for _, key := range cfg.Key {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		svcs = append(svcs, llm.NewClaudeService(httpClient, baseURL, key, model))
	}
	return svcs
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimRight(endpoint, "/")
	return endpoint + "/"
}
