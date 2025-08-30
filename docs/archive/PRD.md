## Product Requirements Document – **Flowrunner**

**Revision:** 1.0  **Date:** July 16 2025  **Owner:** Trevor Martin (tcmartin)

---

### 1 — Purpose

Flowrunner is a **lightweight, YAML‑driven orchestration service** built on top of **Flowlib**, enabling users to define, manage, and trigger workflows without writing Go code. It provides:

* **YAML schema** for node graphs, actions, and parameters
* **HTTP server** with RESTful endpoints for CRUD on flows
* **Execution API** to trigger flows via webhooks or CLI
* **Credentials store** for per‑account secrets (e.g. API keys)
* **Pluggable node registry** for custom or community nodes

Flowrunner empowers non‑developers and integrators to automate processes quickly with versioned, shareable definitions.

---

### 2 — In‑Scope Functionality

| Area                     | Description                                                                             |
| ------------------------ | --------------------------------------------------------------------------------------- |
| **YAML Loader**          | Parse a YAML file into a `Flowlib` graph; validate schema.                              |
| **Flow Registry CRUD**   | Create, Read, Update, Delete flows via HTTP API & CLI.                                  |
| **Trigger Endpoint**     | `POST /flows/{id}/run` to start execution; supports JSON payload for shared context.    |
| **Accounts & Secrets**   | Multi‑tenant support: each account has isolated flows and a credentials vault.          |
| **Node Plugins**         | Register custom Go plugins (implement `flowlib.Node`) at runtime.                       |
| **Inline Scripting**     | Allow small JS snippets (e.g. `math.random`) as “prep” or “exec” hooks via embedded V8. |
| **Webhooks & Callbacks** | Configure HTTP callbacks for flow completion or per‑node events.                        |
| **Logging & Auditing**   | Structured logs per execution; persistence for audit trail.                             |

---

### 3 — Out‑of‑Scope (MVP)

* Visual GUI / drag‑drop editor
* Long‑term persistence beyond in‑memory or simple file-based store
* Scheduling / cron triggers (deferred to future scheduler service)
* Metrics dashboard / monitoring UI

---

### 4 — API Specifications

#### 4.1 — Flow CRUD

```
POST   /api/v1/flows        → create new flow (YAML body)
GET    /api/v1/flows        → list flows
GET    /api/v1/flows/{id}   → retrieve YAML definition
PUT    /api/v1/flows/{id}   → update flow YAML
DELETE /api/v1/flows/{id}   → delete flow
```

#### 4.2 — Execution

```
POST /api/v1/flows/{id}/run
Request JSON: { "shared": { /* any JSON */ } }
Response JSON: { "execution_id": "uuid", "status": "running" }
```

#### 4.3 — Accounts & Auth

* HTTP Basic or Bearer Token authentication
* `/api/v1/accounts/{acct}/secrets` CRUD endpoints for vault entries

---

### 5 — Quality Goals & Acceptance Criteria

| Feature                | Criteria                                                   |
| ---------------------- | ---------------------------------------------------------- |
| YAML Loader            | Invalid schema → 400 error; valid YAML → AST built.        |
| Flow Execution         | End‑to‑end flows complete within 5s for 10‑node graph.     |
| Multi‑tenant Isolation | Flow definitions and secrets segregated per account.       |
| Node Plugin API        | Dynamic plugin loading → no server restart needed.         |
| Inline Scripting       | JS runs in sandbox, <50ms per snippet.                     |
| Webhooks               | Callbacks deliver payload within 200ms of node completion. |

---

### 6 — Milestone Checklist

* [ ] Design YAML schema and JSON API contracts
* [ ] Implement HTTP server with authentication middleware
* [ ] Build YAML→Flowlib graph loader
* [ ] CRUD endpoints and CLI integration
* [ ] Flow runtime harness (`/run` endpoint)
* [ ] Accounts & secrets vault (in‑memory store)
* [ ] Node plugin registry & loader
* [ ] Embed JS engine for inline scripting
* [ ] Webhook dispatcher and retry logic
* [ ] Basic logging and execution audit trail
* [ ] Draft documentation and examples

---

### 7 — Next Steps & Roadmap

1. **v0.1.0**: Core API, YAML loader, execution endpoint, simple CLI.
2. **v0.2.0**: Accounts, secrets vault, plugin loader.
3. **v0.3.0**: Inline scripting, webhooks, logging.
4. **v1.0.0**: Persistent store, scheduling, metrics integration.

---

> *Flowrunner builds on Flowlib’s lightweight engine to deliver a developer‑friendly automation service—no Docker, no heavy infra.*
