package runtime

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/tcmartin/flowrunner/pkg/utils"
)

// ToolExecutionHelper provides utilities for mapping LLM tool calls to node executions
type ToolExecutionHelper struct{}

// NewToolExecutionHelper creates a new tool execution helper
func NewToolExecutionHelper() *ToolExecutionHelper {
	return &ToolExecutionHelper{}
}

// ExtractToolCallParameters extracts parameters from a tool call for node execution
func (h *ToolExecutionHelper) ExtractToolCallParameters(toolCall utils.ToolCall, nodeType string) (map[string]interface{}, error) {
	// Parse the tool call arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	log.Printf("[Tool Helper] Extracting parameters for tool '%s' -> node '%s'", toolCall.Function.Name, nodeType)
	log.Printf("[Tool Helper] Raw arguments: %s", toolCall.Function.Arguments)

	// Map tool calls to node parameters based on tool name and node type
	switch toolCall.Function.Name {
	case "get_website", "fetch_url", "scrape_text":
		return h.mapToHTTPRequest(args)
	case "search_web", "google_search", "search_google":
		return h.mapToWebSearch(args)
	case "send_email", "send_email_summary":
		return h.mapToEmailSend(args)
	default:
		// For unknown tools, pass through the arguments as-is
		log.Printf("[Tool Helper] Unknown tool '%s', passing arguments through", toolCall.Function.Name)
		return args, nil
	}
}

// mapToHTTPRequest maps tool arguments to http.request node parameters
func (h *ToolExecutionHelper) mapToHTTPRequest(args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Extract URL from various possible argument names
	if url, ok := args["url"].(string); ok {
		params["url"] = url
	} else if website, ok := args["website"].(string); ok {
		params["url"] = website
	} else {
		return nil, fmt.Errorf("url parameter is required for HTTP request")
	}

	// Default to GET method
	params["method"] = "GET"
	
	// Add user agent to avoid blocking
	params["headers"] = map[string]interface{}{
		"User-Agent": "Mozilla/5.0 (compatible; FlowRunner-Agent/1.0)",
	}

	// Set timeout
	params["timeout"] = 30

	log.Printf("[Tool Helper] Mapped to HTTP request: %v", params)
	return params, nil
}

// mapToWebSearch maps tool arguments to web search (using Google search via HTTP)
func (h *ToolExecutionHelper) mapToWebSearch(args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Extract query
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required for web search")
	}

	// Create Google search URL
	params["url"] = fmt.Sprintf("https://www.google.com/search?q=%s&num=5", query)
	params["method"] = "GET"
	
	// Add headers to avoid blocking
	params["headers"] = map[string]interface{}{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	}

	// Set timeout
	params["timeout"] = 30

	log.Printf("[Tool Helper] Mapped web search '%s' to HTTP request: %v", query, params)
	return params, nil
}

// mapToEmailSend maps tool arguments to email.send node parameters
func (h *ToolExecutionHelper) mapToEmailSend(args map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Required email parameters
	if subject, ok := args["subject"].(string); ok {
		params["subject"] = subject
	} else {
		return nil, fmt.Errorf("subject parameter is required for email")
	}

	if body, ok := args["body"].(string); ok {
		params["body"] = body
	} else {
		return nil, fmt.Errorf("body parameter is required for email")
	}

	// Extract recipient
	if recipient, ok := args["recipient"].(string); ok {
		params["to"] = recipient
	} else if to, ok := args["to"].(string); ok {
		params["to"] = to
	} else {
		return nil, fmt.Errorf("recipient/to parameter is required for email")
	}

	// Default SMTP settings (these should come from secrets in the actual flow)
	params["smtp_host"] = "smtp.gmail.com"
	params["smtp_port"] = 587
	params["imap_host"] = "imap.gmail.com"
	params["imap_port"] = 993
	params["tls"] = true

	// Note: username, password, and from should be set from secrets in the flow YAML
	// These will be populated by the secret resolution system

	log.Printf("[Tool Helper] Mapped to email send: subject='%s', to='%s'", params["subject"], params["to"])
	return params, nil
}

// FormatToolResult formats a node execution result as a tool result for the LLM
func (h *ToolExecutionHelper) FormatToolResult(toolCall utils.ToolCall, nodeResult interface{}, nodeError error) string {
	toolName := toolCall.Function.Name

	if nodeError != nil {
		log.Printf("[Tool Helper] Tool '%s' failed: %v", toolName, nodeError)
		return fmt.Sprintf("Tool '%s' failed: %v", toolName, nodeError)
	}

	log.Printf("[Tool Helper] Formatting result for tool '%s'", toolName)

	switch toolName {
	case "get_website", "fetch_url", "scrape_text":
		return h.formatHTTPResult(nodeResult)
	case "search_web", "google_search", "search_google":
		return h.formatSearchResult(nodeResult)
	case "send_email", "send_email_summary":
		return h.formatEmailResult(nodeResult)
	default:
		// For unknown tools, convert result to JSON
		if resultBytes, err := json.Marshal(nodeResult); err == nil {
			return string(resultBytes)
		}
		return fmt.Sprintf("%v", nodeResult)
	}
}

// formatHTTPResult formats HTTP request results for the LLM
func (h *ToolExecutionHelper) formatHTTPResult(result interface{}) string {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if body, ok := resultMap["body"].(string); ok {
			// Truncate long responses
			if len(body) > 2000 {
				body = body[:2000] + "... [truncated]"
			}
			
			statusCode := 0
			if sc, ok := resultMap["status_code"].(int); ok {
				statusCode = sc
			}
			
			return fmt.Sprintf("HTTP %d: Retrieved content (%d characters):\n%s", statusCode, len(body), body)
		}
	}
	
	return fmt.Sprintf("HTTP request completed: %v", result)
}

// formatSearchResult formats web search results for the LLM
func (h *ToolExecutionHelper) formatSearchResult(result interface{}) string {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if body, ok := resultMap["body"].(string); ok {
			// Extract search results from HTML (simplified)
			// In a real implementation, you'd parse the HTML properly
			return fmt.Sprintf("Search completed successfully. Found search results page (%d characters of HTML data).", len(body))
		}
	}
	
	return fmt.Sprintf("Search completed: %v", result)
}

// formatEmailResult formats email send results for the LLM
func (h *ToolExecutionHelper) formatEmailResult(result interface{}) string {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if status, ok := resultMap["status"].(string); ok && status == "sent" {
			to := "recipient"
			if toField, ok := resultMap["to"].(string); ok {
				to = toField
			} else if toArray, ok := resultMap["to"].([]interface{}); ok && len(toArray) > 0 {
				if toStr, ok := toArray[0].(string); ok {
					to = toStr
				}
			}
			
			subject := "email"
			if subjectField, ok := resultMap["subject"].(string); ok {
				subject = subjectField
			}
			
			return fmt.Sprintf("Email sent successfully to %s with subject '%s'", to, subject)
		}
	}
	
	return fmt.Sprintf("Email operation completed: %v", result)
}