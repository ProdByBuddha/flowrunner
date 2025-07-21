package plugins

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPPlugin_CreateNode_Success(t *testing.T) {
	plugin := &MCPPlugin{}
	params := map[string]interface{}{
		"connectionType": "cmd",
		"operation":      "listTools",
		"command":        "echo",
		"args":           "hello",
	}

	node, err := plugin.CreateNode(params)

	require.NoError(t, err)
	assert.NotNil(t, node)
}

func TestMCPPlugin_CreateNode_MissingOperation(t *testing.T) {
	plugin := &MCPPlugin{}
	params := map[string]interface{}{
		"connectionType": "cmd",
	}

	_, err := plugin.CreateNode(params)

	require.Error(t, err)
	assert.Equal(t, "mcp: 'operation' parameter is required", err.Error())
}

func TestMCPPlugin_ExecuteTool_CMD_Success(t *testing.T) {
	plugin := &MCPPlugin{}
	params := map[string]interface{}{
		"connectionType": "cmd",
		"operation":      "executeTool",
		"command":        "echo",
		"args":           `{"name": "test"}`,
		"toolName":       "testTool",
	}

	node, err := plugin.CreateNode(params)
	require.NoError(t, err)

	// The Run method in the provided flowlib returns (flowlib.Action, error)
	// The first return value is an Action, which is a string.
	// The result of the execution is embedded in the node itself or handled by post-execution logic,
	// which is not fully implemented in the provided plugin stub.
	// For this test, we will just check if the node runs without error.
	_, err = node.Run(nil)
	require.NoError(t, err)
}

func TestMCPPlugin_ExecuteTool_HTTP_Success(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"result": "success"}`)
	}))
	defer server.Close()

	plugin := &MCPPlugin{}
	params := map[string]interface{}{
		"connectionType": "http",
		"operation":      "executeTool",
		"url":            server.URL,
		"toolName":       "testTool",
	}

	node, err := plugin.CreateNode(params)
	require.NoError(t, err)

	_, err = node.Run(nil)
	require.NoError(t, err)
}
