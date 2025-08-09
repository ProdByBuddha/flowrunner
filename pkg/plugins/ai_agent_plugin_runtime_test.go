package plugins

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "github.com/tcmartin/flowrunner/pkg/utils"
    "github.com/joho/godotenv"
)

// fakeCompleter fakes LLM completion and records the request it received
type fakeCompleter struct {
	called   bool
	lastReq  utils.LLMRequest
	resp     *utils.LLMResponse
}

func (f *fakeCompleter) Complete(ctx context.Context, req utils.LLMRequest) (*utils.LLMResponse, error) {
	f.called = true
	f.lastReq = req
	if f.resp != nil {
		return f.resp, nil
	}
	// Default deterministic response with a tool call
	return &utils.LLMResponse{
		ID:    "fake-123",
		Model: req.Model,
		Choices: []utils.Choice{{
			Index: 0,
			Message: utils.Message{
				Role:    "assistant",
				Content: "OK",
				ToolCalls: []utils.ToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: struct {
						Name      string "json:\"name\""
						Arguments string "json:\"arguments\""
					}{Name: "search_web", Arguments: "{\"query\":\"test\"}"},
				}},
			},
			FinishReason: "tool_calls",
		}},
		Usage: utils.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
	}, nil
}

func TestAIAgentPlugin_Runtime_Fake(t *testing.T) {
	// Override the completer for the duration of this test
	orig := newLLMCompleter
	fc := &fakeCompleter{}
	newLLMCompleter = func(provider utils.LLMProvider, apiKey string, options map[string]interface{}) llmCompleter {
		return fc
	}
	defer func() { newLLMCompleter = orig }()

    // Create node directly via plugin
    node, err := (&AIAgentPlugin{}).CreateNode(map[string]interface{}{
        "provider":    "openai",
        "api_key":     "sk-TEST",
        "model":       "gpt-4.1-mini",
        "prompt":      "Use tools if needed",
        "temperature": 0.2,
        "tools": []any{
            map[string]any{
                "type": "function",
                "function": map[string]any{
                    "name":        "search_web",
                    "description": "Search the web",
                    "parameters": map[string]any{
                        "type": "object",
                        "properties": map[string]any{
                            "query": map[string]any{"type": "string"},
                        },
                        "required": []any{"query"},
                    },
                },
            },
        },
    })
    require.NoError(t, err)
    shared := map[string]interface{}{"question": "What is the weather?"}
    _, err = node.Run(shared)
    require.NoError(t, err)

    // Validate the completer received expected request
	require.True(t, fc.called, "fake completer should be called")
    require.Equal(t, "gpt-4.1-mini", fc.lastReq.Model)
    require.GreaterOrEqual(t, len(fc.lastReq.Messages), 1)
    // Log details for visibility
    t.Logf("FAKE: model=%s messages=%d temperature=%.2f", fc.lastReq.Model, len(fc.lastReq.Messages), fc.lastReq.Temperature)
    for i, m := range fc.lastReq.Messages {
        if i > 2 { break }
        t.Logf("FAKE message[%d]: role=%s content-preview=%.60q", i, m.Role, m.Content)
    }
    // Ensure tools wired through
	require.Equal(t, 1, len(fc.lastReq.Tools))
	require.Equal(t, "search_web", fc.lastReq.Tools[0].Function.Name)
    t.Logf("FAKE tools: %d (first=%s)", len(fc.lastReq.Tools), fc.lastReq.Tools[0].Function.Name)
}

// Proxy completer that records the request then forwards to real client
type recordingCompleter struct {
	real  *utils.LLMClient
	req   utils.LLMRequest
}

func (r *recordingCompleter) Complete(ctx context.Context, req utils.LLMRequest) (*utils.LLMResponse, error) {
	r.req = req
	return r.real.Complete(ctx, req)
}

func TestAIAgentPlugin_Runtime_Real(t *testing.T) {
    // Try to load env from common locations to pick up OPENAI_API_KEY
    _ = godotenv.Load(".env")
    _ = godotenv.Load("../.env")
    _ = godotenv.Load("../../.env")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping real integration: OPENAI_API_KEY not set")
	}
	orig := newLLMCompleter
	var rec *recordingCompleter
	newLLMCompleter = func(provider utils.LLMProvider, apiKey string, options map[string]interface{}) llmCompleter {
		real := utils.NewLLMClient(provider, apiKey, options)
		rec = &recordingCompleter{real: real}
		return rec
	}
	defer func() { newLLMCompleter = orig }()

    node, err := (&AIAgentPlugin{}).CreateNode(map[string]interface{}{
        "provider":    "openai",
        "api_key":     apiKey,
        "model":       "gpt-4.1-mini",
        "prompt":      "Return just the word OK",
        "temperature": 0.0,
    })
    require.NoError(t, err)
    shared := map[string]interface{}{"question": "Say OK"}

	// Give a short timeout window to be safe
	done := make(chan error, 1)
    go func() {
        _, e := node.Run(shared)
        done <- e
    }()
    select {
    case err := <-done:
        require.NoError(t, err)
    case <-time.After(60 * time.Second):
        t.Fatal("timed out waiting for real LLM call")
    }

	require.NotNil(t, rec)
    require.Equal(t, "gpt-4.1-mini", rec.req.Model)
    require.GreaterOrEqual(t, len(rec.req.Messages), 1)
    // Log details for visibility
    t.Logf("REAL: model=%s messages=%d temperature=%.2f", rec.req.Model, len(rec.req.Messages), rec.req.Temperature)
    for i, m := range rec.req.Messages {
        if i > 2 { break }
        t.Logf("REAL message[%d]: role=%s content-preview=%.60q", i, m.Role, m.Content)
    }
}
