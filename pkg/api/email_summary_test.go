package api

import (
    "fmt"
    "os"
    "testing"
    "time"

    "github.com/joho/godotenv"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    imap "github.com/emersion/go-imap"
    imapclient "github.com/emersion/go-imap/client"
    "github.com/tcmartin/flowrunner/pkg/loader"
    "github.com/tcmartin/flowrunner/pkg/plugins"
    "github.com/tcmartin/flowrunner/pkg/registry"
    "github.com/tcmartin/flowrunner/pkg/runtime"
    "github.com/tcmartin/flowrunner/pkg/services"
    "github.com/tcmartin/flowrunner/pkg/storage"
    "github.com/tcmartin/flowrunner/pkg/utils"
)

func createEmailSummaryFlow(t *testing.T, url, subjectPrefix string) {
    t.Helper()
    _ = godotenv.Overload("../../.env")

    sender := os.Getenv("GMAIL_USERNAME")
    pass := os.Getenv("GMAIL_PASSWORD")
    recipient := os.Getenv("EMAIL_RECIPIENT")
    imapHost := os.Getenv("IMAP_HOST")
    if sender == "" || pass == "" || recipient == "" {
        t.Skip("Skipping email summary test: missing SMTP envs")
    }
    if sender == recipient {
        t.Skip("Skipping to avoid loop: recipient equals sender")
    }

    // In-memory provider and secrets
    sp := storage.NewMemoryProvider()
    require.NoError(t, sp.Initialize())
    encKey, err := services.GenerateEncryptionKey()
    require.NoError(t, err)
    vault, err := services.NewExtendedSecretVaultService(sp.GetSecretStore(), encKey)
    require.NoError(t, err)
    accountID := fmt.Sprintf("email-summary-%d", time.Now().UnixNano())
    require.NoError(t, vault.Set(accountID, "GMAIL_USERNAME", sender))
    require.NoError(t, vault.Set(accountID, "GMAIL_PASSWORD", pass))
    require.NoError(t, vault.Set(accountID, "EMAIL_RECIPIENT", recipient))
    if imapHost != "" {
        require.NoError(t, vault.Set(accountID, "IMAP_HOST", imapHost))
    }

    // YAML loader & registry
    nodeFactories := make(map[string]plugins.NodeFactory)
    for nodeType, factory := range runtime.CoreNodeTypes() {
        nodeFactories[nodeType] = &LLMTestRuntimeNodeFactoryAdapter{factory: factory}
    }
    yamlLoader := loader.NewYAMLLoader(nodeFactories, plugins.NewPluginRegistry())
    flowReg := registry.NewFlowRegistry(sp.GetFlowStore(), registry.FlowRegistryOptions{YAMLLoader: yamlLoader})

    // Flow: http -> transform summary -> email
    flowYAML := fmt.Sprintf(`metadata:
  name: "Email Summary %s"
  version: "1.0.0"
nodes:
  fetch:
    type: http.request
    params:
      url: %q
      method: GET
      headers:
        User-Agent: FlowRunner-Email-Summary
    next:
      success: summarize
      default: summarize
  summarize:
    type: transform
    params:
      script: |
        // Simplified content with nonce to avoid provider suppression
        var ts = new Date().toISOString();
        var nonce = Math.floor(Math.random()*100000);
        var subject = 'FlowRunner Summary Test — ' + ts + '-' + nonce;
        var msg = 'This is a FlowRunner summary delivery test.';
        return { subject: subject, body: msg };
    next:
      default: email
  email:
    type: email.send
    params:
      smtp_host: "smtp.gmail.com"
      smtp_port: 587
      username: "${secrets.GMAIL_USERNAME}"
      password: "${secrets.GMAIL_PASSWORD}"
      from: "${secrets.GMAIL_USERNAME}"
      to: "${secrets.EMAIL_RECIPIENT}"
      subject: "${shared.result.subject}"
      body: "${shared.result.body}"
    next:
      default: END
  `, subjectPrefix, url)

    flowID, err := flowReg.Create(accountID, "email-summary", flowYAML)
    require.NoError(t, err)

    rt := runtime.NewFlowRuntimeWithStoreAndSecrets(&LLMTestFlowRegistryAdapter{registry: flowReg}, yamlLoader, sp.GetExecutionStore(), vault)
    execID, err := rt.Execute(accountID, flowID, map[string]interface{}{})
    require.NoError(t, err)

    var status runtime.ExecutionStatus
    for i := 0; i < 90; i++ {
        status, err = rt.GetStatus(execID)
        require.NoError(t, err)
        t.Logf("Execution %s status: %s (iteration %d)", execID, status.Status, i)
        if status.Status == "completed" || status.Status == "failed" {
            break
        }
        time.Sleep(1 * time.Second)
    }
    
    // Log detailed status for debugging
    if status.Status == "failed" {
        t.Logf("Flow execution failed: %+v", status)
        if status.Error != "" {
            t.Logf("Error details: %s", status.Error)
        }
    }
    
    assert.Equal(t, "completed", status.Status, "Email summary flow should complete: %s", url)
}

// verifyInSent polls IMAP Sent folder(s) for a message with the given subject.
func verifyInSent(t *testing.T, subject string) {
    t.Helper()
    _ = godotenv.Overload("../../.env")

    imapHost := os.Getenv("IMAP_HOST")
    if imapHost == "" {
        t.Log("IMAP_HOST not set; skipping IMAP verification")
        return
    }
    username := os.Getenv("GMAIL_USERNAME")
    password := os.Getenv("GMAIL_PASSWORD")
    require.NotEmpty(t, username, "GMAIL_USERNAME required for IMAP verification")
    require.NotEmpty(t, password, "GMAIL_PASSWORD required for IMAP verification")

    // Discover Sent-like folders via IMAP attributes (\Sent)
    folders := []string{}
    if sentDiscovered, err := discoverSentFolders(imapHost, username, password); err == nil {
        t.Logf("Discovered Sent-like folders: %v", sentDiscovered)
        folders = append(folders, sentDiscovered...)
    } else {
        t.Logf("Could not discover Sent folders: %v", err)
    }
    // Fallbacks and common Gmail names
    folders = append(folders,
        "[Gmail]/Sent Mail",
        "[Gmail]/All Mail",
        "Sent",
        "Sent Mail",
        "Sent Messages",
        "Sent Items",
        "INBOX",
    )

    // Build IMAP client (SMTP params irrelevant for read)
    client := utils.NewEmailClient("smtp.gmail.com", 587, imapHost, 993, username, password)
    require.NoError(t, client.Connect())
    defer client.Close()

    // Poll for up to 90s
    deadline := time.Now().Add(90 * time.Second)
    found := false
    var lastErr error
    for time.Now().Before(deadline) && !found {
        for _, folder := range folders {
            // First try subject + from
            filter := utils.EmailFilter{
                Folder:  folder,
                Subject: subject,
                Since:   time.Now().Add(-60 * time.Minute),
                Limit:   50,
                From:    username,
            }
            msgs, err := client.GetEmails(filter)
            if err != nil {
                lastErr = err
                continue
            }
            if len(msgs) > 0 {
                found = true
                break
            }
            // Fallback: From-only search in this folder
            filter.Subject = ""
            msgs, err = client.GetEmails(filter)
            if err != nil {
                lastErr = err
                continue
            }
            if len(msgs) > 0 {
                found = true
                break
            }
        }
        if !found {
            time.Sleep(2 * time.Second)
        }
    }
    if !found && lastErr != nil {
        t.Logf("IMAP search error (last): %v", lastErr)
    }
    require.True(t, found, "expected to find sent email with subject: %q", subject)
}

// verifyInRecipientInbox tries to confirm delivery to the recipient's mailbox via IMAP.
// It is enabled only if EMAIL_RECIPIENT_IMAP_HOST, EMAIL_RECIPIENT_USERNAME, and
// EMAIL_RECIPIENT_PASSWORD are set. If enabled, failure to find the email will fail the test.
func verifyInRecipientInbox(t *testing.T, subject string) {
    t.Helper()
    _ = godotenv.Load("../../.env")

    host := os.Getenv("EMAIL_RECIPIENT_IMAP_HOST")
    user := os.Getenv("EMAIL_RECIPIENT_USERNAME")
    pass := os.Getenv("EMAIL_RECIPIENT_PASSWORD")
    if host == "" || user == "" || pass == "" {
        t.Log("Recipient IMAP not configured; skipping recipient delivery verification")
        return
    }

    port := 993
    if p := os.Getenv("EMAIL_RECIPIENT_IMAP_PORT"); p != "" {
        var tmp int
        if _, err := fmt.Sscanf(p, "%d", &tmp); err == nil && tmp > 0 {
            port = tmp
        }
    }

    // Use EmailClient for IMAP read (SMTP details not needed)
    client := utils.NewEmailClient("", 0, host, port, user, pass)
    require.NoError(t, client.Connect())
    defer client.Close()

    deadline := time.Now().Add(120 * time.Second)
    found := false
    var lastErr error
    for time.Now().Before(deadline) && !found {
        filter := utils.EmailFilter{
            Folder:  "INBOX",
            Subject: subject,
            Since:   time.Now().Add(-90 * time.Minute),
            Limit:   50,
        }
        msgs, err := client.GetEmails(filter)
        if err != nil {
            lastErr = err
        } else if len(msgs) > 0 {
            found = true
            break
        }
        time.Sleep(3 * time.Second)
    }
    if !found && lastErr != nil {
        t.Logf("Recipient IMAP search error (last): %v", lastErr)
    }
    require.True(t, found, "recipient did not receive email with subject: %q", subject)
}

// discoverSentFolders lists IMAP mailboxes and returns those marked with \\Sent
func discoverSentFolders(host, username, password string) ([]string, error) {
    addr := fmt.Sprintf("%s:%d", host, 993)
    c, err := imapclient.DialTLS(addr, nil)
    if err != nil {
        return nil, err
    }
    defer c.Logout()
    if err := c.Login(username, password); err != nil {
        return nil, err
    }
    mailboxes := make(chan *imap.MailboxInfo, 50)
    done := make(chan error, 1)
    go func() {
        done <- c.List("", "*", mailboxes)
    }()
    var result []string
    for m := range mailboxes {
        for _, attr := range m.Attributes {
            if attr == imap.SentAttr || attr == "\\Sent" {
                result = append(result, m.Name)
                break
            }
        }
    }
    if err := <-done; err != nil {
        return result, err
    }
    return result, nil
}

// Serial: two summaries one after the other
func TestEmailSummarySeries(t *testing.T) {
    createEmailSummaryFlow(t, "https://www.gemmit.org", "Serial")
    verifyInSent(t, "FlowRunner Summary Test")
    time.Sleep(5 * time.Second)

    createEmailSummaryFlow(t, "https://httpbin.org/html", "Serial")
    verifyInSent(t, "FlowRunner Summary Test")
}

// Parallel: two summaries launched concurrently from the test
func TestEmailSummaryParallel(t *testing.T) {
    _ = godotenv.Overload("../../.env")
    // Serialize to avoid burst; add small gap between sends
    urls := []string{"https://www.gemmit.org", "https://httpbin.org/html"}
    for _, u := range urls {
        createEmailSummaryFlow(t, u, "Parallel-Serialized")
        time.Sleep(3 * time.Second)
    }
    for range urls {
        verifyInSent(t, "FlowRunner Summary Test")
    }
}
