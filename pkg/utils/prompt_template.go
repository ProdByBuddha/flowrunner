// Package utils provides utility functions for the flowrunner.
package utils

import (
	"bytes"
	"fmt"
	"regexp"
	"text/template"
)

// PromptTemplate represents a template for generating prompts with variables
type PromptTemplate struct {
	Template string
	parser   *template.Template
}

// NewPromptTemplate creates a new prompt template
func NewPromptTemplate(templateStr string) (*PromptTemplate, error) {
	// Parse the template
	tmpl, err := template.New("prompt").Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &PromptTemplate{
		Template: templateStr,
		parser:   tmpl,
	}, nil
}

// Render renders the template with the given variables
func (pt *PromptTemplate) Render(variables map[string]any) (string, error) {
	var buf bytes.Buffer
	if err := pt.parser.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}

// ParseVariables extracts variable names from a template string
func ParseVariables(templateStr string) []string {
	// Find all occurrences of {{.VarName}}
	re := regexp.MustCompile(`{{\.([a-zA-Z0-9_]+)}}`)
	matches := re.FindAllStringSubmatch(templateStr, -1)

	// Extract variable names
	vars := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seen[varName] {
				vars = append(vars, varName)
				seen[varName] = true
			}
		}
	}

	return vars
}

// TemplateManager manages a collection of prompt templates
type TemplateManager struct {
	templates map[string]*PromptTemplate
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*PromptTemplate),
	}
}

// AddTemplate adds a template to the manager
func (tm *TemplateManager) AddTemplate(name, templateStr string) error {
	tmpl, err := NewPromptTemplate(templateStr)
	if err != nil {
		return err
	}
	tm.templates[name] = tmpl
	return nil
}

// GetTemplate retrieves a template by name
func (tm *TemplateManager) GetTemplate(name string) (*PromptTemplate, bool) {
	tmpl, ok := tm.templates[name]
	return tmpl, ok
}

// RenderTemplate renders a template by name with the given variables
func (tm *TemplateManager) RenderTemplate(name string, variables map[string]any) (string, error) {
	tmpl, ok := tm.templates[name]
	if !ok {
		return "", fmt.Errorf("template not found: %s", name)
	}
	return tmpl.Render(variables)
}

// ListTemplates returns a list of all template names
func (tm *TemplateManager) ListTemplates() []string {
	names := make([]string, 0, len(tm.templates))
	for name := range tm.templates {
		names = append(names, name)
	}
	return names
}

// RemoveTemplate removes a template from the manager
func (tm *TemplateManager) RemoveTemplate(name string) {
	delete(tm.templates, name)
}

// MessageFromTemplate creates a Message from a template and variables
func MessageFromTemplate(role, templateStr string, variables map[string]any) (Message, error) {
	tmpl, err := NewPromptTemplate(templateStr)
	if err != nil {
		return Message{}, err
	}

	content, err := tmpl.Render(variables)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Role:    role,
		Content: content,
	}, nil
}

// MessagesFromTemplates creates a slice of Messages from templates and variables
func MessagesFromTemplates(templates []struct {
	Role     string
	Template string
}, variables map[string]any) ([]Message, error) {
	messages := make([]Message, 0, len(templates))

	for _, tmpl := range templates {
		msg, err := MessageFromTemplate(tmpl.Role, tmpl.Template, variables)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
