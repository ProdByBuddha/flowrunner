package runtime

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "time"

    "github.com/r3labs/sse/v2"
    "github.com/tcmartin/flowlib"
    "github.com/tcmartin/flowrunner/pkg/utils"
)

// NewMCPNodeWrapper creates a new MCP client node wrapper
// This node can communicate with MCP servers via three transports:
// - cmd (STDIO): launch a local MCP server command and exchange JSON via stdin/stdout
// - http: POST a JSON request to an HTTP endpoint and parse the JSON response
// - sse: POST the request (optional separate endpoint) and listen for a single SSE message
func NewMCPNodeWrapper(params map[string]interface{}) (flowlib.Node, error) {
    baseNode := flowlib.NewNode(3, 1*time.Second)

    wrapper := &NodeWrapper{
        node: baseNode,
        exec: func(input interface{}) (interface{}, error) {
            // Unpack combined input format used by Flowrunner
            var nodeParams map[string]interface{}
            var flowInput map[string]interface{}
            if combined, ok := input.(map[string]interface{}); ok {
                if p, has := combined["params"]; has {
                    pm, ok := p.(map[string]interface{})
                    if !ok {
                        return nil, fmt.Errorf("expected params to be map[string]interface{}")
                    }
                    nodeParams = pm
                    if in, hasIn := combined["input"]; hasIn {
                        if im, ok := in.(map[string]interface{}); ok {
                            flowInput = im
                        }
                    }
                } else {
                    // Back-compat: direct params
                    nodeParams = combined
                }
            } else {
                return nil, fmt.Errorf("expected map[string]interface{}, got %T", input)
            }

            connectionType := getString(nodeParams, "connectionType", "cmd")
            operation := getString(nodeParams, "operation", "")

            // Build operation payload
            // Build operation payload unless rawBody is provided later
            payload := map[string]interface{}{
                "method": "",
                "params": map[string]interface{}{},
            }
            op := payload["params"].(map[string]interface{})

            // Provide ergonomic aliases for common MCP methods when operation supplied
            // - listTools  => tools/list
            // - executeTool => tools/call (uses toolName/promptName + toolParameters)
            if operation != "" {
                switch operation {
                case "listTools":
                    payload["method"] = "tools/list"
                case "executeTool":
                    payload["method"] = "tools/call"
                default:
                    // Assume fully-qualified MCP method
                    payload["method"] = operation
                }
            }

            // Optional operation fields
            if v := getString(nodeParams, "resourceUri", ""); v != "" {
                op["uri"] = v
            }
            if v := getString(nodeParams, "toolName", ""); v != "" {
                op["name"] = v
            }
            if v := getString(nodeParams, "promptName", ""); v != "" {
                op["name"] = v
            }
            // toolParameters can be JSON string or structured object
            if raw, ok := nodeParams["toolParameters"]; ok {
                switch tv := raw.(type) {
                case string:
                    if strings.TrimSpace(tv) != "" {
                        var parsed any
                        if err := json.Unmarshal([]byte(tv), &parsed); err == nil {
                            op["arguments"] = parsed
                        } else {
                            // keep as string
                            op["arguments"] = tv
                        }
                    }
                default:
                    // pass-through any structured object/array
                    op["arguments"] = tv
                }
            }

            // Allow rawBody override: if provided, post it verbatim and skip operation requirement
            var reqBytes []byte
            // Optional Go-template rendering for raw body
            if rbt, ok := nodeParams["rawBodyTemplate"]; ok && rbt != nil {
                if s, ok := rbt.(string); ok && strings.TrimSpace(s) != "" {
                    vars := map[string]interface{}{"input": flowInput, "params": nodeParams}
                    if pt, err := utils.NewPromptTemplate(s); err == nil {
                        if rendered, err := pt.Render(vars); err == nil {
                            reqBytes = []byte(rendered)
                        } else {
                            return nil, fmt.Errorf("failed to render rawBodyTemplate: %w", err)
                        }
                    } else {
                        return nil, fmt.Errorf("failed to parse rawBodyTemplate: %w", err)
                    }
                }
            }
            if rb, ok := nodeParams["rawBody"]; ok && rb != nil && len(reqBytes) == 0 {
                switch v := rb.(type) {
                case string:
                    reqBytes = []byte(v)
                default:
                    if b, e := json.Marshal(v); e == nil {
                        reqBytes = b
                    } else {
                        return nil, fmt.Errorf("failed to marshal rawBody: %w", e)
                    }
                }
            }
            
            // If no rawBody or rawBodyTemplate was processed, require either an operation alias or method has been set
            if len(reqBytes) == 0 {
                if getString(nodeParams, "operation", "") == "" && payload["method"] == "" {
                    return nil, fmt.Errorf("mcp: 'operation' parameter is required unless 'rawBody' or 'rawBodyTemplate' is provided")
                }
                var err error
                reqBytes, err = json.Marshal(payload)
                if err != nil {
                    return nil, fmt.Errorf("failed to marshal request payload: %w", err)
                }
            }

            // Execute based on transport
            switch connectionType {
            case "cmd":
                command := getString(nodeParams, "command", "")
                if command == "" {
                    return nil, fmt.Errorf("mcp: 'command' parameter is required for 'cmd' connection type")
                }
                origArgs := fieldsOrJSON(getString(nodeParams, "args", ""))
                // Optional extra args appended to the command
                if raw, ok := nodeParams["argsExtra"]; ok {
                    switch tv := raw.(type) {
                    case string:
                        if ex := fieldsOrJSON(tv); len(ex) > 0 {
                            origArgs = append(origArgs, ex...)
                        }
                    case []interface{}:
                        for _, it := range tv {
                            if s, ok := it.(string); ok {
                                origArgs = append(origArgs, s)
                            }
                        }
                    }
                }

                // Optional serverPort support: append a flag and set envs as needed
                if spRaw, ok := nodeParams["serverPort"]; ok {
                    // Determine port string
                    portStr := ""
                    switch v := spRaw.(type) {
                    case string:
                        portStr = strings.TrimSpace(v)
                    case float64:
                        portStr = fmt.Sprintf("%d", int(v))
                    case int:
                        portStr = fmt.Sprintf("%d", v)
                    }
                    if portStr != "" {
                        // Choose port flag (default --port) or allow override via portFlag
                        portFlag := getString(nodeParams, "portFlag", "--port")
                        origArgs = append(origArgs, portFlag, portStr)

                        // Set common env keys unless user provided explicit env already
                        // Merge into envMap after we build it below by adding to explicitEnv
                        // We simulate here by appending to explicitEnv slice
                        // Note: we modify nodeParams["env"] to include keys so parseEnvList picks them up
                        extraEnvs := []string{"PORT=" + portStr, "MCP_PORT=" + portStr, "SERVER_PORT=" + portStr}
                        if pevRaw, ok2 := nodeParams["portEnvKeys"]; ok2 {
                            // Allow override of env key names
                            switch tv := pevRaw.(type) {
                            case string:
                                for _, k := range fieldsOrJSON(tv) {
                                    if k != "" {
                                        extraEnvs = append(extraEnvs, k+"="+portStr)
                                    }
                                }
                            case []interface{}:
                                for _, it := range tv {
                                    if s, ok := it.(string); ok && s != "" {
                                        extraEnvs = append(extraEnvs, s+"="+portStr)
                                    }
                                }
                            }
                        }
                        // Merge into any existing env value in params
                        merged := strings.Join(extraEnvs, "\n")
                        if existing, ok3 := nodeParams["env"].(string); ok3 && strings.TrimSpace(existing) != "" {
                            nodeParams["env"] = existing + "\n" + merged
                        } else {
                            nodeParams["env"] = merged
                        }
                    }
                }

                // Build environment for the child process
                // Start with OS env; explicit env from params overlay on top
                envMap := envSliceToMap(os.Environ())
                explicitEnv := parseEnvList(getString(nodeParams, "env", ""))
                for _, kv := range explicitEnv {
                    if eq := strings.IndexByte(kv, '='); eq > 0 {
                        k := kv[:eq]
                        v := kv[eq+1:]
                        envMap[k] = v
                    }
                }
                env := make([]string, 0, len(envMap))
                for k, v := range envMap {
                    env = append(env, fmt.Sprintf("%s=%s", k, v))
                }

                // Prefer dotenvx for DX unless disabled
                useDotenvx := true
                if v, ok := nodeParams["dotenvx"]; ok {
                    if b, okb := v.(bool); okb {
                        useDotenvx = b
                    }
                }

                if useDotenvx {
                    // Compose a wrapped command using dotenvx
                    dotenvxCmd := getString(nodeParams, "dotenvxCommand", "dotenvx")
                    // Support running via npx if desired
                    useNpx := false
                    if v, ok := nodeParams["dotenvxUseNpx"]; ok {
                        if b, okb := v.(bool); okb {
                            useNpx = b
                        }
                    }
                    pkg := getString(nodeParams, "dotenvxPackage", "dotenvx@latest")
                    // Files to load
                    files := []string{}
                    if raw, ok := nodeParams["dotenvxFiles"]; ok {
                        switch tv := raw.(type) {
                        case string:
                            if arr := fieldsOrJSON(tv); len(arr) > 0 {
                                files = arr
                            } else if strings.TrimSpace(tv) != "" {
                                files = []string{tv}
                            }
                        case []interface{}:
                            for _, it := range tv {
                                if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
                                    files = append(files, s)
                                }
                            }
                        }
                    }
                    extra := []string{}
                    if raw, ok := nodeParams["dotenvxArgs"]; ok {
                        switch tv := raw.(type) {
                        case string:
                            extra = fieldsOrJSON(tv)
                        case []interface{}:
                            for _, it := range tv {
                                if s, ok := it.(string); ok {
                                    extra = append(extra, s)
                                }
                            }
                        }
                    }

                    // Build dotenvx arg list
                    dx := []string{"run"}
                    for _, f := range files {
                        dx = append(dx, "-f", f)
                    }
                    dx = append(dx, extra...)
                    dx = append(dx, "--", command)
                    dx = append(dx, origArgs...)

                    // Wrap with dotenvx (or npx dotenvx@latest)
                    if useNpx || dotenvxCmd == "npx" {
                        command = "npx"
                        args := append([]string{pkg}, dx...)
                        out, err := runCmd(command, args, env, reqBytes)
                        if err != nil {
                            return nil, err
                        }
                        return decodeResult(out), nil
                    }

                    // Use dotenvx binary directly
                    args := dx
                    out, err := runCmd(dotenvxCmd, args, env, reqBytes)
                    if err != nil {
                        return nil, err
                    }
                    return decodeResult(out), nil
                }

                // Fallback: run command directly without dotenvx
                out, err := runCmd(command, origArgs, env, reqBytes)
                if err != nil {
                    return nil, err
                }
                return decodeResult(out), nil

            case "http":
                url := getString(nodeParams, "url", "")
                if url == "" {
                    return nil, fmt.Errorf("mcp: 'url' parameter is required for 'http' connection type")
                }
                headers := toStringMap(nodeParams["headers"]) // optional
                // Optional templated headers via Go templates (e.g., Authorization: "Bearer {{.input.token}}")
                if htRaw, ok := nodeParams["headersTemplate"]; ok && htRaw != nil {
                    if hm, ok := htRaw.(map[string]interface{}); ok {
                        vars := map[string]interface{}{"input": flowInput, "params": nodeParams}
                        for k, v := range hm {
                            if sv, ok := v.(string); ok {
                                if pt, err := utils.NewPromptTemplate(sv); err == nil {
                                    if rendered, err := pt.Render(vars); err == nil {
                                        if headers == nil { headers = map[string]string{} }
                                        headers[k] = rendered
                                    } else {
                                        return nil, fmt.Errorf("failed to render headersTemplate[%s]: %w", k, err)
                                    }
                                } else {
                                    return nil, fmt.Errorf("failed to parse headersTemplate[%s]: %w", k, err)
                                }
                            }
                        }
                    }
                }
                timeout := parseTimeoutMS(nodeParams["timeout"], 60000)

                out, err := doHTTP(url, headers, timeout, reqBytes)
                if err != nil {
                    return nil, err
                }
                return decodeResult(out), nil

            case "sse":
                url := getString(nodeParams, "url", "")
                if url == "" {
                    return nil, fmt.Errorf("mcp: 'url' parameter is required for 'sse' connection type")
                }
                post := getString(nodeParams, "messagesPostEndpoint", "")
                headers := toStringMap(nodeParams["headers"]) // optional
                // Optional templated headers via Go templates
                if htRaw, ok := nodeParams["headersTemplate"]; ok && htRaw != nil {
                    if hm, ok := htRaw.(map[string]interface{}); ok {
                        vars := map[string]interface{}{"input": flowInput, "params": nodeParams}
                        for k, v := range hm {
                            if sv, ok := v.(string); ok {
                                if pt, err := utils.NewPromptTemplate(sv); err == nil {
                                    if rendered, err := pt.Render(vars); err == nil {
                                        if headers == nil { headers = map[string]string{} }
                                        headers[k] = rendered
                                    } else {
                                        return nil, fmt.Errorf("failed to render headersTemplate[%s]: %w", k, err)
                                    }
                                } else {
                                    return nil, fmt.Errorf("failed to parse headersTemplate[%s]: %w", k, err)
                                }
                            }
                        }
                    }
                }
                timeout := parseTimeoutMS(nodeParams["timeout"], 60000)

                // Optimization for servers that return synchronous HTTP results:
                // If a POST endpoint is provided, send the request via HTTP and return the response
                if post != "" {
                    out, err := doHTTP(post, headers, timeout, reqBytes)
                    if err != nil {
                        return nil, err
                    }
                    return decodeResult(out), nil
                }

                // Otherwise, use SSE subscription
                out, err := doSSE(url, post, headers, timeout, reqBytes)
                if err != nil {
                    return nil, err
                }
                return decodeResult(out), nil

            default:
                return nil, fmt.Errorf("unsupported connection type: '%s'", connectionType)
            }
        },
    }

    // Set initial params
    wrapper.SetParams(params)
    return wrapper, nil
}

// --- helpers ---

func getString(m map[string]interface{}, key, def string) string {
    if v, ok := m[key]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return def
}

func fieldsOrJSON(s string) []string {
    if s == "" {
        return nil
    }
    // try JSON array first
    var arr []string
    if json.Unmarshal([]byte(s), &arr) == nil {
        return arr
    }
    return strings.Fields(s)
}

func parseEnvList(s string) []string {
    if s == "" {
        return nil
    }
    // If JSON array of strings
    var arr []string
    if json.Unmarshal([]byte(s), &arr) == nil {
        return arr
    }
    // newline-separated KEY=VAL
    lines := strings.Split(s, "\n")
    out := make([]string, 0, len(lines))
    for _, line := range lines {
        if strings.TrimSpace(line) != "" {
            out = append(out, line)
        }
    }
    return out
}

func envSliceToMap(env []string) map[string]string {
    m := make(map[string]string, len(env))
    for _, kv := range env {
        if eq := strings.IndexByte(kv, '='); eq > 0 {
            k := kv[:eq]
            v := kv[eq+1:]
            m[k] = v
        }
    }
    return m
}

func toStringMap(v interface{}) map[string]string {
    if v == nil {
        return nil
    }
    out := map[string]string{}
    switch m := v.(type) {
    case map[string]string:
        for k, s := range m {
            out[k] = s
        }
    case map[string]interface{}:
        for k, val := range m {
            out[k] = fmt.Sprintf("%v", val)
        }
    case map[interface{}]interface{}:
        for k, val := range m {
            out[fmt.Sprintf("%v", k)] = fmt.Sprintf("%v", val)
        }
    }
    return out
}

func parseTimeoutMS(v interface{}, defMS int) time.Duration {
    switch t := v.(type) {
    case float64:
        return time.Duration(int(t)) * time.Millisecond
    case int:
        return time.Duration(t) * time.Millisecond
    case string:
        if strings.HasSuffix(t, "ms") || strings.HasSuffix(t, "s") || strings.HasSuffix(t, "m") {
            if d, err := time.ParseDuration(t); err == nil {
                return d
            }
        }
        if n, err := fmt.Sscanf(t, "%d", &defMS); err == nil && n == 1 {
            return time.Duration(defMS) * time.Millisecond
        }
    }
    return time.Duration(defMS) * time.Millisecond
}

func runCmd(command string, args []string, env []string, inputPayload []byte) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, command, args...)
    // inherit env and append provided overrides
    cmd.Env = append(os.Environ(), env...)

    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
    }
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("failed to start command: %w", err)
    }
    if _, err := stdin.Write(inputPayload); err != nil {
        cmd.Wait()
        return nil, fmt.Errorf("failed to write to stdin: %w, stderr: %s", err, stderr.String())
    }
    _ = stdin.Close()
    if err := cmd.Wait(); err != nil {
        return nil, fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
    }
    if stderr.Len() > 0 {
        fmt.Printf("MCP command stderr: %s", stderr.String())
    }
    return stdout.Bytes(), nil
}

func doHTTP(url string, headers map[string]string, timeout time.Duration, inputPayload []byte) ([]byte, error) {
    client := &http.Client{Timeout: timeout}
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(inputPayload))
    if err != nil {
        return nil, fmt.Errorf("failed to create http request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("http request failed: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("http request returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
    }
    return io.ReadAll(resp.Body)
}

func doSSE(url, postEndpoint string, headers map[string]string, timeout time.Duration, inputPayload []byte) ([]byte, error) {
    // Optional pre-post to separate endpoint prior to subscribing
    if postEndpoint != "" {
        postHeaders := make(map[string]string)
        for k, v := range headers {
            postHeaders[k] = v
        }
        postHeaders["Content-Type"] = "application/json"
        if _, err := doHTTP(postEndpoint, postHeaders, timeout, inputPayload); err != nil {
            return nil, fmt.Errorf("sse post to messages endpoint failed: %w", err)
        }
    }

    client := sse.NewClient(url)
    // Note: r3labs/sse client has limited header support for initial handshake

    var eventData []byte
    errCh := make(chan error, 1)
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    go func() {
        err := client.SubscribeWithContext(ctx, "message", func(msg *sse.Event) {
            if len(msg.Data) > 0 {
                eventData = msg.Data
                cancel()
            }
        })
        if err != nil && ctx.Err() == nil {
            errCh <- err
        }
    }()

    select {
    case <-ctx.Done():
        if ctx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("sse request timed out after %v", timeout)
        }
        if eventData == nil {
            return nil, fmt.Errorf("sse connection closed without receiving a message")
        }
        return eventData, nil
    case err := <-errCh:
        return nil, fmt.Errorf("sse subscription failed: %w", err)
    }
}

func decodeResult(out []byte) interface{} {
    var v interface{}
    if err := json.Unmarshal(out, &v); err == nil {
        return map[string]interface{}{"result": v}
    }
    return map[string]interface{}{"result": string(out)}
}
