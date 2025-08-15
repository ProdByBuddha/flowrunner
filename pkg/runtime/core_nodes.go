// Package runtime provides functionality for executing flows.
package runtime

import (
	"fmt"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/tcmartin/flowlib"
	"github.com/tcmartin/flowrunner/pkg/utils"
)

// CoreNodeTypes returns a map of built-in node types
func CoreNodeTypes() map[string]NodeFactory {
	return map[string]NodeFactory{
		"http.request":  NewHTTPRequestNodeWrapper,
		"store":         NewStoreNodeWrapper,
		"transform":     NewTransformNodeWrapper,
		"condition":     NewConditionNodeWrapper,
		"router":        NewRouterNodeWrapper,        // Enhanced condition node with tool call support
		"delay":         NewDelayNodeWrapper,
		"wait":          NewWaitNodeWrapper,
		"cron":          NewCronNodeWrapper,
		"llm":           NewLLMNodeWrapper,
		"email.send":    NewSMTPNodeWrapper,
		"email.receive": NewIMAPNodeWrapper,
		"agent":         NewAgentNodeWrapper,
		"webhook":       NewWebhookNodeWrapper,
		"dynamodb":      NewDynamoDBNodeWrapper,
		"postgres":      NewPostgresNodeWrapper,
		"format":        NewResponseFormatterNodeWrapper, // Response formatting for tool results
		"split":         NewSplitNodeWrapper,             // Split execution for parallel processing
		"join":          NewJoinNodeWrapper,              // Join parallel execution results
	}
}

// NodeFactory is a function that creates a node
type NodeFactory func(params map[string]interface{}) (flowlib.Node, error)

// NewTransformNodeWrapper creates a new transform node wrapper
func NewTransformNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var nodeParams map[string]interface{}
			var flowInput map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if paramsField, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := paramsField.(map[string]interface{}); ok {
						nodeParams = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
					
					// Extract flow input
					if inputField, hasInput := combinedInput["input"]; hasInput {
						if inputMap, ok := inputField.(map[string]interface{}); ok {
							flowInput = inputMap
						}
					}
				} else {
					// Old format: direct params (backwards compatibility)
					nodeParams = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Extract the JavaScript script
			script, ok := nodeParams["script"].(string)
			if !ok {
				return nil, fmt.Errorf("script parameter is required and must be a string")
			}

			// Create JavaScript engine using Otto
			vm := otto.New()

			// Set up console.log for debugging
			vm.Set("console", map[string]interface{}{
				"log": func(args ...interface{}) {
					fmt.Printf("[Transform Script] %v\n", args...)
				},
			})

			// Set up the context
			// If we have flow input, add it as 'input' context
			if flowInput != nil {
				vm.Set("input", flowInput)
			} else {
				// For backwards compatibility, if no flow input, use the node params as input
				vm.Set("input", nodeParams)
			}

			// Execute the transform script
			// Wrap the script in a function to allow return statements
			wrappedScript := "(function() {\n" + script + "\n})()"
			result, err := vm.Run(wrappedScript)
			if err != nil {
				return nil, fmt.Errorf("failed to execute transform script: %w", err)
			}

			// Convert result to Go value
			goValue, err := result.Export()
			if err != nil {
				return nil, fmt.Errorf("failed to export JavaScript result: %w", err)
			}

			return goValue, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// This is intentionally left empty to remove the duplicate declaration

// NewSMTPNodeWrapper creates a new SMTP node wrapper
func NewSMTPNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Extract SMTP server parameters
			smtpHost, ok := params["smtp_host"].(string)
			if !ok {
				return nil, fmt.Errorf("smtp_host parameter is required")
			}

			smtpPort := 587 // Default SMTP port
			if portParam, ok := params["smtp_port"].(int); ok {
				smtpPort = portParam
			}

			// Extract IMAP server parameters (for connection sharing)
			imapHost := smtpHost // Default to same as SMTP
			if hostParam, ok := params["imap_host"].(string); ok {
				imapHost = hostParam
			}

			imapPort := 993 // Default IMAP port
			if portParam, ok := params["imap_port"].(int); ok {
				imapPort = portParam
			}

			// Extract authentication parameters
			username, ok := params["username"].(string)
			if !ok {
				return nil, fmt.Errorf("username parameter is required")
			}

			password, ok := params["password"].(string)
			if !ok {
				return nil, fmt.Errorf("password parameter is required")
			}

			// Extract email parameters
			from, ok := params["from"].(string)
			if !ok {
				from = username // Default to username
			}

			var to []string
			if toParam, ok := params["to"].(string); ok {
				to = []string{toParam}
			} else if toArray, ok := params["to"].([]interface{}); ok {
				for _, recipient := range toArray {
					if recipientStr, ok := recipient.(string); ok {
						to = append(to, recipientStr)
					}
				}
			} else {
				return nil, fmt.Errorf("to parameter is required")
			}

			subject, ok := params["subject"].(string)
			if !ok {
				return nil, fmt.Errorf("subject parameter is required")
			}

			var body string
			if bodyParam, ok := params["body"].(string); ok {
				body = bodyParam
			}

			var html string
			if htmlParam, ok := params["html"].(string); ok {
				html = htmlParam
			}

			// Extract CC and BCC recipients
			var cc []string
			if ccParam, ok := params["cc"].(string); ok {
				cc = []string{ccParam}
			} else if ccArray, ok := params["cc"].([]interface{}); ok {
				for _, recipient := range ccArray {
					if recipientStr, ok := recipient.(string); ok {
						cc = append(cc, recipientStr)
					}
				}
			}

			var bcc []string
			if bccParam, ok := params["bcc"].(string); ok {
				bcc = []string{bccParam}
			} else if bccArray, ok := params["bcc"].([]interface{}); ok {
				for _, recipient := range bccArray {
					if recipientStr, ok := recipient.(string); ok {
						bcc = append(bcc, recipientStr)
					}
				}
			}

			// Extract attachments
			var attachments []utils.EmailAttachment
			if attachmentsArray, ok := params["attachments"].([]interface{}); ok {
				for _, attachmentParam := range attachmentsArray {
					if attachmentMap, ok := attachmentParam.(map[string]interface{}); ok {
						filename, _ := attachmentMap["filename"].(string)
						contentType, _ := attachmentMap["content_type"].(string)

						var content []byte
						if contentStr, ok := attachmentMap["content"].(string); ok {
							content = []byte(contentStr)
						} else if contentBytes, ok := attachmentMap["content"].([]byte); ok {
							content = contentBytes
						}

						if filename != "" && len(content) > 0 {
							attachments = append(attachments, utils.EmailAttachment{
								Filename:    filename,
								ContentType: contentType,
								Content:     content,
							})
						}
					}
				}
			}

			// Extract headers
			headers := make(map[string]string)
			if headersMap, ok := params["headers"].(map[string]interface{}); ok {
				for key, value := range headersMap {
					if valueStr, ok := value.(string); ok {
						headers[key] = valueStr
					}
				}
			}

			// Create email client
			client := utils.NewEmailClient(smtpHost, smtpPort, imapHost, imapPort, username, password)

			// Connect to the server
			if err := client.Connect(); err != nil {
				return nil, fmt.Errorf("failed to connect to email server: %w", err)
			}
			defer client.Close()

			// Create email message
			message := utils.EmailMessage{
				From:        from,
				To:          to,
				Cc:          cc,
				Bcc:         bcc,
				Subject:     subject,
				Body:        body,
				HTML:        html,
				Attachments: attachments,
				Headers:     headers,
			}

			// Send the email
			if err := client.SendEmail(message); err != nil {
				return nil, fmt.Errorf("failed to send email: %w", err)
			}

			// Return success
			return map[string]interface{}{
				"status":  "sent",
				"from":    from,
				"to":      to,
				"cc":      cc,
				"bcc":     bcc,
				"subject": subject,
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewIMAPNodeWrapper creates a new IMAP node wrapper
func NewIMAPNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// Extract IMAP server parameters
			imapHost, ok := params["imap_host"].(string)
			if !ok {
				return nil, fmt.Errorf("imap_host parameter is required")
			}

			imapPort := 993 // Default IMAP port
			if portParam, ok := params["imap_port"].(int); ok {
				imapPort = portParam
			}

			// Extract SMTP server parameters (for connection sharing)
			smtpHost := imapHost // Default to same as IMAP
			if hostParam, ok := params["smtp_host"].(string); ok {
				smtpHost = hostParam
			}

			smtpPort := 587 // Default SMTP port
			if portParam, ok := params["smtp_port"].(int); ok {
				smtpPort = portParam
			}

			// Extract authentication parameters
			username, ok := params["username"].(string)
			if !ok {
				return nil, fmt.Errorf("username parameter is required")
			}

			password, ok := params["password"].(string)
			if !ok {
				return nil, fmt.Errorf("password parameter is required")
			}

			// Create email filter
			filter := utils.EmailFilter{
				Folder: "INBOX", // Default folder
			}

			// Extract folder
			if folderParam, ok := params["folder"].(string); ok {
				filter.Folder = folderParam
			}

			// Extract since parameter
			if sinceParam, ok := params["since"].(string); ok {
				if since, err := time.Parse(time.RFC3339, sinceParam); err == nil {
					filter.Since = since
				}
			}

			// Extract before parameter
			if beforeParam, ok := params["before"].(string); ok {
				if before, err := time.Parse(time.RFC3339, beforeParam); err == nil {
					filter.Before = before
				}
			}

			// Extract from parameter
			if fromParam, ok := params["from"].(string); ok {
				filter.From = fromParam
			}

			// Extract to parameter
			if toParam, ok := params["to"].(string); ok {
				filter.To = toParam
			}

			// Extract subject parameter
			if subjectParam, ok := params["subject"].(string); ok {
				filter.Subject = subjectParam
			}

			// Extract unseen parameter
			if unseenParam, ok := params["unseen"].(bool); ok {
				filter.Unseen = unseenParam
			}

			// Extract limit parameter
			if limitParam, ok := params["limit"].(int); ok {
				filter.Limit = uint32(limitParam)
			}

			// Extract mark_as_read parameter
			if markAsReadParam, ok := params["mark_as_read"].(bool); ok {
				filter.MarkAsRead = markAsReadParam
			}

			// Extract with_body parameter
			if withBodyParam, ok := params["with_body"].(bool); ok {
				filter.WithBody = withBodyParam
			}

			// Extract body_preview parameter
			if bodyPreviewParam, ok := params["body_preview"].(int); ok {
				filter.BodyPreview = uint32(bodyPreviewParam)
			}

			// Create email client
			client := utils.NewEmailClient(smtpHost, smtpPort, imapHost, imapPort, username, password)

			// Connect to the server
			if err := client.Connect(); err != nil {
				return nil, fmt.Errorf("failed to connect to email server: %w", err)
			}
			defer client.Close()

			// Get emails
			emails, err := client.GetEmails(filter)
			if err != nil {
				return nil, fmt.Errorf("failed to get emails: %w", err)
			}

			// Convert emails to map
			result := make([]map[string]interface{}, len(emails))
			for i, email := range emails {
				emailMap := map[string]interface{}{
					"subject":   email.Subject,
					"from":      email.From,
					"to":        email.To,
					"cc":        email.Cc,
					"date":      email.Date,
					"body":      email.Body,
					"html":      email.HTML,
					"headers":   email.Headers,
					"messageId": email.MessageID,
					"metadata":  email.Metadata,
				}
				result[i] = emailMap
			}

			return result, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// The actual implementation of NewAgentNodeWrapper is in agent_node.go

// NewWebhookNodeWrapper creates a new webhook node wrapper
func NewWebhookNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(3, 1*time.Second)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Handle both old format (direct params) and new format (combined input)
			var params map[string]interface{}
			
			if combinedInput, ok := input.(map[string]interface{}); ok {
				if nodeParams, hasParams := combinedInput["params"]; hasParams {
					// New format: combined input with params and input
					if paramsMap, ok := nodeParams.(map[string]interface{}); ok {
						params = paramsMap
					} else {
						return nil, fmt.Errorf("expected params to be map[string]interface{}")
					}
				} else {
					// Old format: direct params (backwards compatibility)
					params = combinedInput
				}
			} else {
				return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
			}

			// This is a placeholder - in a real implementation, this would send
			// a webhook to a configured URL using the params
			_ = params // Suppress unused variable warning in placeholder implementation
			return map[string]interface{}{
				"status": "sent",
			}, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}
// NewSplitNodeWrapper creates a new split node wrapper for parallel execution
func NewSplitNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Split node simply passes through the input to enable parallel execution
			// The actual parallel execution is handled by the flow runtime
			return input, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}

// NewJoinNodeWrapper creates a new join node wrapper for collecting parallel execution results
func NewJoinNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
	// Create the base node
	baseNode := flowlib.NewNode(1, 0)

	// Create the wrapper
	wrapper := &NodeWrapper{
		node: baseNode,
		exec: func(input interface{}) (interface{}, error) {
			// Join node collects results from parallel execution branches
			// In a real implementation, this would wait for all parallel branches to complete
			// and merge their results. For now, we'll pass through the input.
			
			// If input is a map, we can collect results from different branches
			if inputMap, ok := input.(map[string]interface{}); ok {
				// Create a combined result from all branches
				result := make(map[string]interface{})
				
				// Copy all input data to the result
				for key, value := range inputMap {
					result[key] = value
				}
				
				// Add a marker that this came from a join operation
				result["_join_operation"] = true
				
				return result, nil
			}
			
			// For non-map inputs, just pass through
			return input, nil
		},
	}

	// Set the parameters
	wrapper.SetParams(params)

	return wrapper, nil
}