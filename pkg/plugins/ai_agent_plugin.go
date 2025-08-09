package plugins

import (
    "context"
    "fmt"
    "time"

    "github.com/tcmartin/flowlib"
    "github.com/tcmartin/flowrunner/pkg/utils"
)

// AIAgentPlugin registers a simplified YAML abstraction for an AI agent node.
// Implemented directly against flowlib + utils to avoid package cycles.
//
// Node type: "ai.agent"
//
// Params: provider, api_key, model, prompt or messages, temperature, max_tokens, tools
// Optionally uses shared.input.question if provided by the flow.

type AIAgentPlugin struct{}

func (p *AIAgentPlugin) Name() string { return "ai.agent" }
func (p *AIAgentPlugin) Description() string { return "Simplified AI agent (LLM wrapper) with optional tools" }
func (p *AIAgentPlugin) Version() string { return "0.1.0" }

// llmCompleter abstracts LLM client for testability
type llmCompleter interface {
    Complete(ctx context.Context, req utils.LLMRequest) (*utils.LLMResponse, error)
}

// newLLMCompleter is a hook for tests; defaults to real utils.NewLLMClient
var newLLMCompleter = func(provider utils.LLMProvider, apiKey string, options map[string]interface{}) llmCompleter {
    return utils.NewLLMClient(provider, apiKey, options)
}

func (p *AIAgentPlugin) CreateNode(params map[string]interface{}) (flowlib.Node, error) {
    if params == nil {
        return nil, fmt.Errorf("ai.agent requires params")
    }

    // Build a retriable node
    n := flowlib.NewNode(3, 5*time.Second)
    // Stash params so loader can augment with node_id/node_type later
    n.SetParams(params)

    // Prep: pass through the shared context
    n.SetPrepFn(func(shared any) (any, error) { return shared, nil })

    // Exec: perform a single LLM call with optional tools
    n.SetExecFn(func(shared any) (any, error) {
        // Access static params (possibly augmented by loader)
        pmap := n.Params()

        // Provider
        providerStr, _ := pmap["provider"].(string)
        if providerStr == "" {
            providerStr = "openai"
        }
        var provider utils.LLMProvider
        switch providerStr {
        case "openai":
            provider = utils.OpenAI
        case "anthropic":
            provider = utils.Anthropic
        default:
            provider = utils.Generic
        }

        // Required fields
        apiKey, _ := pmap["api_key"].(string)
        model, _ := pmap["model"].(string)
        if apiKey == "" || model == "" {
            return nil, fmt.Errorf("ai.agent requires api_key and model")
        }

        // Build messages
        var messages []utils.Message
        if sharedMap, ok := shared.(map[string]interface{}); ok {
            if q, ok := sharedMap["question"].(string); ok && q != "" {
                messages = []utils.Message{{Role: "system", Content: "You are a helpful assistant."}, {Role: "user", Content: q}}
            }
        }
        if len(messages) == 0 {
            if prompt, ok := pmap["prompt"].(string); ok && prompt != "" {
                messages = []utils.Message{{Role: "user", Content: prompt}}
            } else if msgs, ok := pmap["messages"].([]any); ok {
                for _, m := range msgs {
                    if mm, ok := m.(map[string]any); ok {
                        role, _ := mm["role"].(string)
                        content, _ := mm["content"].(string)
                        messages = append(messages, utils.Message{Role: role, Content: content})
                    }
                }
            } else {
                return nil, fmt.Errorf("ai.agent requires prompt, messages, or shared.question")
            }
        }

        // Tools
        var tools []utils.ToolDefinition
        if tlist, ok := pmap["tools"].([]any); ok {
            for _, ti := range tlist {
                if tm, ok := ti.(map[string]any); ok {
                    ttype, _ := tm["type"].(string)
                    var fdef map[string]any
                    if fm, ok := tm["function"].(map[string]any); ok {
                        fdef = fm
                    }
                    if fdef != nil {
                        name, _ := fdef["name"].(string)
                        desc, _ := fdef["description"].(string)
                        var params map[string]any
                        if pm, ok := fdef["parameters"].(map[string]any); ok {
                            params = pm
                        }
                        tools = append(tools, utils.ToolDefinition{Type: ttype, Function: utils.FunctionDefinition{Name: name, Description: desc, Parameters: params}})
                    }
                }
            }
        }

        // Options
        temperature := 0.7
        if v, ok := pmap["temperature"].(float64); ok {
            temperature = v
        }
        maxTokens := 0
        if v, ok := pmap["max_tokens"].(int); ok {
            maxTokens = v
        }

        client := newLLMCompleter(provider, apiKey, nil)
        req := utils.LLMRequest{Model: model, Messages: messages, Temperature: temperature, MaxTokens: maxTokens, Tools: tools}
        ctx := context.Background()
        resp, err := client.Complete(ctx, req)
        if err != nil {
            return nil, fmt.Errorf("LLM request failed: %w", err)
        }
        if resp.Error != nil {
            return nil, fmt.Errorf("LLM API error: %s", resp.Error.Message)
        }
        if len(resp.Choices) == 0 {
            return nil, fmt.Errorf("no choices returned from LLM")
        }
        // Extract tool_calls if any
        msg := resp.Choices[0].Message
        hasTools := len(msg.ToolCalls) > 0
        return map[string]any{
            "id":            resp.ID,
            "model":         resp.Model,
            "choices":       resp.Choices,
            "usage":         resp.Usage,
            "content":       msg.Content,
            "finish_reason": resp.Choices[0].FinishReason,
            "raw_response":  resp.RawResponse,
            "has_tool_calls": hasTools,
            "tool_calls":    msg.ToolCalls,
        }, nil
    })

    return n, nil
}
