package ai

const (
	LLMProviderGemini = "gemini"
	LLMProviderOpenAI = "openai"
	LLMProviderClaude = "claude"
)

type LLMConfig struct {
	Provider string       `mapstructure:"provider"`
	Gemini   GeminiConfig `mapstructure:"gemini"`
	OpenAI   OpenAIConfig `mapstructure:"openai"`
	Claude   ClaudeConfig `mapstructure:"claude"`
}

type GeminiConfig struct {
	BaseURL string   `mapstructure:"base_url"`
	Key     []string `mapstructure:"key"`
	Model   string   `mapstructure:"model"`
}

type OpenAIConfig struct {
	BaseURL string   `mapstructure:"base_url"`
	Key     []string `mapstructure:"key"`
	Model   string   `mapstructure:"model"`
}

type ClaudeConfig struct {
	BaseURL string   `mapstructure:"base_url"`
	Key     []string `mapstructure:"key"`
	Model   string   `mapstructure:"model"`
}

