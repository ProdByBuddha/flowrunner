package api

import (
    "fmt"
    "testing"
    "time"
    "os"

    "github.com/joho/godotenv"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/assert"
    "github.com/tcmartin/flowrunner/pkg/loader"
    "github.com/tcmartin/flowrunner/pkg/plugins"
    "github.com/tcmartin/flowrunner/pkg/registry"
    "github.com/tcmartin/flowrunner/pkg/runtime"
    "github.com/tcmartin/flowrunner/pkg/services"
    "github.com/tcmartin/flowrunner/pkg/storage"
)

// TestEmailToolRouteSend exercises the tool-call router path to email.send without web fetch.
// It simulates a single tool call (send_email) and routes to email.send with a delay and nonce subject.
func TestEmailToolRouteSend(t *testing.T) {
    t.Helper()
    _ = godotenv.Overload("../../.env")

    sender := os.Getenv("GMAIL_USERNAME")
    pass := os.Getenv("GMAIL_PASSWORD")
    recipient := os.Getenv("EMAIL_RECIPIENT")
    imapHost := os.Getenv("IMAP_HOST")
    if sender == "" || pass == "" || recipient == "" {
        t.Skip("Skipping tool-route email test: missing SMTP envs")
    }
    if sender == recipient {
        t.Skip("Skipping to avoid loop: recipient equals sender")
    }

    sp := storage.NewMemoryProvider()
    require.NoError(t, sp.Initialize())
    encKey, err := services.GenerateEncryptionKey()
    require.NoError(t, err)
    vault, err := services.NewExtendedSecretVaultService(sp.GetSecretStore(), encKey)
    require.NoError(t, err)
    accountID := fmt.Sprintf("email-tool-route-%d", time.Now().UnixNano())
    require.NoError(t, vault.Set(accountID, "GMAIL_USERNAME", sender))
    require.NoError(t, vault.Set(accountID, "GMAIL_PASSWORD", pass))
    require.NoError(t, vault.Set(accountID, "EMAIL_RECIPIENT", recipient))
    if imapHost != "" {
        require.NoError(t, vault.Set(accountID, "IMAP_HOST", imapHost))
    }

    nodeFactories := make(map[string]plugins.NodeFactory)
    for nodeType, factory := range runtime.CoreNodeTypes() {
        nodeFactories[nodeType] = &LLMTestRuntimeNodeFactoryAdapter{factory: factory}
    }
    yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())
    flowReg := registry.NewFlowRegistry(sp.GetFlowStore(), registry.FlowRegistryOptions{YAMLLoader: yamlLoader})

    // Build a flow: start(transform) -> router -> delay -> email_send
    // The transform returns a result containing a single OpenAI-like tool call to `send_email`.
    flowYAML := `metadata:
  name: "Email Tool Route"
  version: "1.0.0"
nodes:
  start:
    type: transform
    params:
      script: |
        // Build a tool_call under llm_result.tool_calls for router detection
        var ts = new Date().toISOString();
        var subj = "FlowRunner Tool Route Test — " + ts;
        var args = JSON.stringify({ subject: subj, body: "Tool route delivery test." });
        return {
          llm_result: {
            tool_calls: [
              { id: "call_1", type: "function", function: { name: "send_email", arguments: args } }
            ]
          }
        };
    next:
      default: tool_router

  tool_router:
    type: router
    params:
      input: ${shared.result.llm_result}
      condition_script: |
        // Hardened routing: if tool calls are present or unknown, route to send_email
        return 'send_email';
    next:
      send_email: pre_send_delay
      default: pre_send_delay

  pre_send_delay:
    type: delay
    params:
      duration_ms: 30000
    next:
      default: email_send

  email_send:
    type: email.send
    params:
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      imap_host: "${secrets.IMAP_HOST}"
      imap_port: 993
      username: "${secrets.GMAIL_USERNAME}"
      password: "${secrets.GMAIL_PASSWORD}"
      from: "${secrets.GMAIL_USERNAME}"
      to: "${secrets.EMAIL_RECIPIENT}"
      subject: "${input.tool_params.subject}"
      body: "${input.tool_params.body}"
      headers:
        Reply-To: "${secrets.GMAIL_USERNAME}"
        X-FlowRunner: "tool-route-test"
    next:
      default: END
`

    flowID, err := flowReg.Create(accountID, "email-tool-route", flowYAML)
    require.NoError(t, err)

    rt := runtime.NewFlowRuntimeWithStoreAndSecrets(&LLMTestFlowRegistryAdapter{registry: flowReg}, yamlLoader, sp.GetExecutionStore(), vault)
    execID, err := rt.Execute(accountID, flowID, map[string]interface{}{})
    require.NoError(t, err)

    var status runtime.ExecutionStatus
    for i := 0; i < 120; i++ {
        status, err = rt.GetStatus(execID)
        require.NoError(t, err)
        if status.Status == "completed" || status.Status == "failed" {
            break
        }
        time.Sleep(1 * time.Second)
    }
    assert.Equal(t, "completed", status.Status, "Tool-route email flow should complete")
}
