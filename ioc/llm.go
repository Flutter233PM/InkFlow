package ioc

import (
	"github.com/KNICEX/InkFlow/internal/ai"
	"github.com/spf13/viper"
	"strings"
)

func InitLLMConfig() ai.LLMConfig {
	var cfg ai.LLMConfig
	if err := viper.UnmarshalKey("llm", &cfg); err != nil {
		panic(err)
	}

	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	cfg.Gemini.Key = trimEmpty(cfg.Gemini.Key)
	cfg.OpenAI.Key = trimEmpty(cfg.OpenAI.Key)
	cfg.Claude.Key = trimEmpty(cfg.Claude.Key)

	if cfg.Provider == "" {
		switch {
		case len(cfg.Gemini.Key) > 0:
			cfg.Provider = ai.LLMProviderGemini
		case len(cfg.OpenAI.Key) > 0:
			cfg.Provider = ai.LLMProviderOpenAI
		case len(cfg.Claude.Key) > 0:
			cfg.Provider = ai.LLMProviderClaude
		}
	}

	return cfg
}

func trimEmpty(keys []string) []string {
	res := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		res = append(res, k)
	}
	return res
}
