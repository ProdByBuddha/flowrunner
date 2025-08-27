package api

import (
    "os"
    "testing"
    "time"
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

// TestEmailSmoke sends a deterministic email using the email.send node.
func TestEmailSmoke(t *testing.T) {
    _ = godotenv.Overload("../../.env")

    sender := os.Getenv("GMAIL_USERNAME")
    pass := os.Getenv("GMAIL_PASSWORD")
    recipient := os.Getenv("EMAIL_RECIPIENT")
    imapHost := os.Getenv("IMAP_HOST")

    if sender == "" || pass == "" || recipient == "" {
        t.Skip("Skipping email smoke test: GMAIL_USERNAME/GMAIL_PASSWORD/EMAIL_RECIPIENT not set")
    }
    if sender == recipient {
        t.Skip("Skipping email smoke test to avoid loop: recipient equals sender")
    }

    // In-memory stores and secret vault
    sp := storage.NewMemoryProvider()
    require.NoError(t, sp.Initialize())
    encKey, err := services.GenerateEncryptionKey()
    require.NoError(t, err)
    vault, err := services.NewExtendedSecretVaultService(sp.GetSecretStore(), encKey)
    require.NoError(t, err)

    // Account and secrets
    accountID := "email-smoke-account"
    require.NoError(t, vault.Set(accountID, "GMAIL_USERNAME", sender))
    require.NoError(t, vault.Set(accountID, "GMAIL_PASSWORD", pass))
    require.NoError(t, vault.Set(accountID, "EMAIL_RECIPIENT", recipient))
    if imapHost != "" {
        require.NoError(t, vault.Set(accountID, "IMAP_HOST", imapHost))
    }

    // YAML loader and flow registry
    nodeFactories := make(map[string]plugins.NodeFactory)
    for nodeType, factory := range runtime.CoreNodeTypes() {
        nodeFactories[nodeType] = &LLMTestRuntimeNodeFactoryAdapter{factory: factory}
    }
    yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())
    flowReg := registry.NewFlowRegistry(sp.GetFlowStore(), registry.FlowRegistryOptions{YAMLLoader: yamlLoader})

    // Simple email-only flow
    flowYAML := `metadata:
  name: "Email Smoke Test"
  version: "1.0.0"
nodes:
  send:
    type: "email.send"
    params:
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      imap_host: "${secrets.IMAP_HOST}"
      imap_port: 993
      username: "${secrets.GMAIL_USERNAME}"
      password: "${secrets.GMAIL_PASSWORD}"
      from: "${secrets.GMAIL_USERNAME}"
      to: "${secrets.EMAIL_RECIPIENT}"
      subject: "FlowRunner Email Smoke Test"
      body: "This is a deterministic email from FlowRunner tests."
    next:
      default: END
`

    flowID, err := flowReg.Create(accountID, "email-smoke", flowYAML)
    require.NoError(t, err)

    // Runtime with secrets
    rt := runtime.NewFlowRuntimeWithStoreAndSecrets(&LLMTestFlowRegistryAdapter{registry: flowReg}, yamlLoader, sp.GetExecutionStore(), vault)

    execID, err := rt.Execute(accountID, flowID, map[string]interface{}{})
    require.NoError(t, err)

    // Poll for completion (short)
    var status runtime.ExecutionStatus
    for i := 0; i < 60; i++ {
        status, err = rt.GetStatus(execID)
        require.NoError(t, err)
        if status.Status == "completed" || status.Status == "failed" {
            break
        }
        // Wait a bit before next poll
        // Using a short sleep to allow SMTP to complete
        time.Sleep(1 * time.Second)
    }

    assert.Equal(t, "completed", status.Status, "Email flow should complete successfully")
}
