# AI-16: Multi-AI Provider Support

## Overview

Support multiple AI providers (Claude, GPT, Gemini, Ollama) with fallback and cost optimization.

## User Stories

1. As a user, I want to choose my preferred AI provider
2. As a user, I want automatic fallback if one provider fails
3. As a user, I want to use local AI (Ollama) for privacy
4. As a user, I want to see AI cost estimates

## Technical Requirements

### Service Layer

Create `internal/services/aiprovider.go`:

```go
package services

type AIProviderService interface {
    // GetProvider returns the configured AI provider
    GetProvider(ctx context.Context) (AIProvider, error)

    // SetProvider changes the active provider
    SetProvider(ctx context.Context, provider string) error

    // ListProviders returns available providers
    ListProviders(ctx context.Context) ([]ProviderInfo, error)

    // Chat sends a message to the AI
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

    // GetUsage returns usage statistics
    GetUsage(ctx context.Context, provider string, period string) (*UsageStats, error)
}

type AIProvider interface {
    Name() string
    Chat(ctx context.Context, messages []Message, opts ChatOptions) (*ChatResponse, error)
    IsAvailable(ctx context.Context) bool
    EstimateCost(tokens int) float64
}

type ProviderInfo struct {
    Name        string
    Type        ProviderType
    IsAvailable bool
    IsLocal     bool
    CostPerToken float64
    Models      []string
}

type ProviderType string

const (
    ProviderClaude   ProviderType = "claude"
    ProviderOpenAI   ProviderType = "openai"
    ProviderGemini   ProviderType = "gemini"
    ProviderOllama   ProviderType = "ollama"
    ProviderOpenRouter ProviderType = "openrouter"
)

type ChatRequest struct {
    Messages   []Message
    Provider   string  // Optional, uses default if empty
    Model      string  // Optional, uses provider default
    MaxTokens  int
    Temperature float64
}

type ChatResponse struct {
    Content    string
    Provider   string
    Model      string
    TokensUsed int
    Cost       float64
    Latency    time.Duration
}

type UsageStats struct {
    Provider     string
    Period       string
    TotalTokens  int
    TotalCost    float64
    RequestCount int
    AvgLatency   time.Duration
}
```

### Provider Implementations

```go
// Claude provider
type ClaudeProvider struct {
    apiKey string
    model  string
}

func (p *ClaudeProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*ChatResponse, error) {
    // Use existing Claude integration via stdin/stdout
}

// Ollama provider (local)
type OllamaProvider struct {
    url   string
    model string
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*ChatResponse, error) {
    // HTTP POST to http://localhost:11434/api/chat
}

// OpenRouter provider (multi-model)
type OpenRouterProvider struct {
    apiKey string
    model  string
}

func (p *OpenRouterProvider) Chat(ctx context.Context, messages []Message, opts ChatOptions) (*ChatResponse, error) {
    // HTTP POST to https://openrouter.ai/api/v1/chat/completions
}
```

### Database Schema

```sql
CREATE TABLE ai_usage (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    provider TEXT NOT NULL,
    model TEXT,
    tokens_in INTEGER,
    tokens_out INTEGER,
    cost REAL,
    latency_ms INTEGER,
    feature TEXT,  -- "summarize", "categorize", etc.
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ai_usage_provider ON ai_usage(provider);
CREATE INDEX idx_ai_usage_date ON ai_usage(created_at);
```

### Fallback Logic

```go
func (s *AIProviderService) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    providers := s.getProviderOrder(req.Provider)

    var lastErr error
    for _, providerName := range providers {
        provider, err := s.getProvider(providerName)
        if err != nil {
            continue
        }

        if !provider.IsAvailable(ctx) {
            continue
        }

        resp, err := provider.Chat(ctx, req.Messages, ChatOptions{
            Model:      req.Model,
            MaxTokens:  req.MaxTokens,
            Temperature: req.Temperature,
        })
        if err != nil {
            lastErr = err
            continue
        }

        // Track usage
        s.recordUsage(ctx, resp)
        return resp, nil
    }

    return nil, fmt.Errorf("all providers failed: %w", lastErr)
}
```

## UI/UX

### TUI
- Provider indicator in AI panel
- Switch provider with shortcut

```
┌─ AI Assistant (Claude) ───────────────────────────────────────────┐
│ > Summarize this email                                            │
│                                                                   │
│ This email is about...                                            │
├───────────────────────────────────────────────────────────────────┤
│ Tokens: 150  Cost: $0.002  Latency: 1.2s                         │
│ [C]laude [G]PT [O]llama [Tab] Switch provider                    │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Provider selector in settings
- Usage dashboard
- Cost tracking charts
- Model comparison

## Configuration

```yaml
# config.yaml
ai:
  default_provider: "claude"
  fallback_order: ["claude", "openai", "ollama"]
  providers:
    claude:
      enabled: true
      model: "claude-3-sonnet"
    openai:
      enabled: false
      api_key: "${OPENAI_API_KEY}"
      model: "gpt-4"
    ollama:
      enabled: true
      url: "http://localhost:11434"
      model: "llama3"
    openrouter:
      enabled: false
      api_key: "${OPENROUTER_API_KEY}"
      model: "anthropic/claude-3-sonnet"
```

## Testing

1. Test each provider implementation
2. Test fallback behavior
3. Test cost calculation
4. Test availability checks
5. Test usage tracking
6. Test with Ollama (local)

## Acceptance Criteria

- [ ] Claude provider works (existing)
- [ ] OpenAI provider integration
- [ ] Ollama (local) provider works
- [ ] Automatic fallback on failure
- [ ] Usage tracking and cost display
- [ ] Can switch providers easily
- [ ] Provider-specific settings

## Estimated Complexity

High - Multiple API integrations plus orchestration
