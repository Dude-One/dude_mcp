# Dude MCP — Observability Panel

> **Pattern:** Tee on LLM stream → Event Bus → SSE → Admin Panel  
> **Visibility:** REST gate + MCP gate (repo-native gate is invisible — direct disk read)  
> **Phase:** 1.5 — add after REST handler is working

---

## What You Can See

Since your Go server sits between every AI client and the LLM, you have full visibility into:

| Signal | Available | From |
|--------|-----------|------|
| Raw question | Yes | REST / MCP handler |
| Project queried | Yes | Path param / tool arg |
| Caller gate | Yes | REST vs MCP context |
| Caller identity | Yes | API key header |
| Documents selected | Yes | `DocumentSelector` output |
| LLM tokens (live stream) | Yes | `LLMGateway` tee |
| Final answer | Yes | `KnowledgeAnswer` |
| Sources cited | Yes | `KnowledgeAnswer.Sources` |
| Latency per step | Yes | Instrumented middleware |
| Errors | Yes | Error return values |
| Repo-native reads | **No** | Direct disk — no server |

---

## Architecture

```
AI Client (ChatGPT / Cursor / Claude)
        │
        │  HTTP POST or MCP tool call
        ▼
┌─────────────────────────────────┐
│   ProjectContextAIHandler       │
│   or MCPServer                  │
│         │                       │
│         ▼                       │
│   ProjectContextAIServiceImpl   │
│         │                       │
│         ├─► DocumentSelector    │
│         ├─► PromptComposer      │
│         │                       │
│         ▼                       │
│   LLMGateway.Complete()         │
│         │                       │
│    ┌────┴──────┐                │
│    │  TEE      │                │
│    ├──────────►│── stream ──►   │ back to AI client
│    │           │                │
│    └──────────►│── events ──►   │ EventBus
└───────────────────────────────-─┘
                                  │
                              EventBus
                                  │
                    ┌─────────────┴──────────────┐
                    │    SSE endpoint             │
                    │  GET /admin/events/stream   │
                    └─────────────────────────────┘
                                  │
                          Admin Panel (browser)
```

---

## Domain Event — `core/domain/events/ask_event.go`

```go
package events

import "time"

type AskStatus string

const (
    AskStarted   AskStatus = "started"
    AskStreaming  AskStatus = "streaming"
    AskDone      AskStatus = "done"
    AskError     AskStatus = "error"
)

// AskEvent is emitted at each stage of the Ask() pipeline.
// It is NOT a domain entity — it is an observation side-effect.
type AskEvent struct {
    TraceID      string    `json:"trace_id"`
    Project      string    `json:"project"`
    Gate         string    `json:"gate"`         // "rest" | "mcp"
    CallerID     string    `json:"caller_id"`    // API key prefix or MCP client
    Question     string    `json:"question"`
    SelectedDocs []string  `json:"selected_docs,omitempty"`
    TokenChunk   string    `json:"token_chunk,omitempty"` // non-empty only on AskStreaming
    Answer       string    `json:"answer,omitempty"`
    Sources      []string  `json:"sources,omitempty"`
    Model        string    `json:"model,omitempty"`
    LatencyMs    int64     `json:"latency_ms,omitempty"`
    Status       AskStatus `json:"status"`
    Error        string    `json:"error,omitempty"`
    Timestamp    time.Time `json:"timestamp"`
}
```

---

## Event Bus Port — `core/ports/event_bus.go`

Keep the domain clean — publish through a port, never import a concrete bus in `core/`.

```go
package ports

import "github.com/yourorg/dude-mcp/core/domain/events"

// EventBus is the port for publishing domain observation events.
// The core never depends on the concrete implementation.
type EventBus interface {
    PublishAsk(event events.AskEvent)
}

// NoopEventBus discards all events. Use in tests or when observability is disabled.
type NoopEventBus struct{}

func (n *NoopEventBus) PublishAsk(_ events.AskEvent) {}
```

---

## In-Memory Event Bus Adapter — `adapters/eventbus/memory_event_bus.go`

Fan-out to all connected SSE subscribers. Zero dependencies.

```go
package eventbus

import (
    "sync"

    "github.com/yourorg/dude-mcp/core/domain/events"
)

type MemoryEventBus struct {
    mu          sync.RWMutex
    subscribers map[string]chan events.AskEvent
}

func NewMemoryEventBus() *MemoryEventBus {
    return &MemoryEventBus{
        subscribers: make(map[string]chan events.AskEvent),
    }
}

// PublishAsk fans out to all active SSE subscribers.
// Non-blocking: slow subscribers are skipped (dropped event).
func (b *MemoryEventBus) PublishAsk(event events.AskEvent) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers {
        select {
        case ch <- event:
        default:
            // subscriber too slow — drop rather than block the pipeline
        }
    }
}

// Subscribe returns a channel that receives events and a cancel func.
func (b *MemoryEventBus) Subscribe(id string) (<-chan events.AskEvent, func()) {
    ch := make(chan events.AskEvent, 64)
    b.mu.Lock()
    b.subscribers[id] = ch
    b.mu.Unlock()
    cancel := func() {
        b.mu.Lock()
        delete(b.subscribers, id)
        close(ch)
        b.mu.Unlock()
    }
    return ch, cancel
}
```

---

## Instrumented Use Case — `core/usecases/project_context_ai_service.go`

Add event publishing inside `Ask()` — the only place that changes.

```go
func (s *ProjectContextAIServiceImpl) Ask(
    ctx context.Context,
    query entities.KnowledgeQuery,
) (*entities.KnowledgeAnswer, error) {

    traceID := newTraceID()
    gate := gateFromContext(ctx) // "rest" or "mcp"
    callerID := callerFromContext(ctx)
    start := time.Now()

    // 1. Emit: started
    s.bus.PublishAsk(events.AskEvent{
        TraceID:   traceID,
        Project:   query.ProjectName,
        Gate:      gate,
        CallerID:  callerID,
        Question:  query.Question,
        Status:    events.AskStarted,
        Timestamp: time.Now(),
    })

    // 2. Load project docs
    project, err := s.repo.LoadProject(ctx, query.ProjectName)
    if err != nil {
        s.bus.PublishAsk(events.AskEvent{
            TraceID: traceID, Status: events.AskError,
            Error: err.Error(), Timestamp: time.Now(),
        })
        return nil, err
    }

    // 3. Select relevant docs
    selected := s.selector.SelectRelevant(project.Documents, query.Question, query.MaxSources)
    selectedPaths := make([]string, len(selected))
    for i, d := range selected {
        selectedPaths[i] = d.Path
    }

    // Emit: docs selected
    s.bus.PublishAsk(events.AskEvent{
        TraceID:      traceID,
        SelectedDocs: selectedPaths,
        Status:       events.AskStreaming,
        Timestamp:    time.Now(),
    })

    // 4. Build prompt
    prompt := s.composer.BuildPrompt(selected, query.Question)

    // 5. Call LLM
    resp, err := s.llm.Complete(ctx, ports.LLMCompletionRequest{
        SystemPrompt: prompt.System,
        UserPrompt:   prompt.User,
        Model:        s.cfg.Model,
        MaxTokens:    s.cfg.MaxTokens,
    })
    if err != nil {
        s.bus.PublishAsk(events.AskEvent{
            TraceID: traceID, Status: events.AskError,
            Error: err.Error(), Timestamp: time.Now(),
        })
        return nil, err
    }

    // 6. Build answer
    sources := make([]entities.DocumentRef, len(selected))
    for i, d := range selected {
        sources[i] = entities.DocumentRef{Path: d.Path, Title: d.Title}
    }
    answer := &entities.KnowledgeAnswer{
        Answer:  resp.Text,
        Sources: sources,
        Model:   resp.Model,
    }

    // 7. Emit: done
    sourcePaths := make([]string, len(sources))
    for i, s := range sources { sourcePaths[i] = s.Path }

    s.bus.PublishAsk(events.AskEvent{
        TraceID:   traceID,
        Project:   query.ProjectName,
        Gate:      gate,
        CallerID:  callerID,
        Question:  query.Question,
        Answer:    resp.Text,
        Sources:   sourcePaths,
        Model:     resp.Model,
        LatencyMs: time.Since(start).Milliseconds(),
        Status:    events.AskDone,
        Timestamp: time.Now(),
    })

    return answer, nil
}
```

---

## Context Helpers — `core/usecases/context_keys.go`

Attach gate + caller ID in handlers so the use case can read them.

```go
package usecases

type contextKey string

const (
    ContextKeyGate     contextKey = "gate"
    ContextKeyCallerID contextKey = "caller_id"
)

func WithGate(ctx context.Context, gate string) context.Context {
    return context.WithValue(ctx, ContextKeyGate, gate)
}

func WithCallerID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, ContextKeyCallerID, id)
}

func gateFromContext(ctx context.Context) string {
    if v, ok := ctx.Value(ContextKeyGate).(string); ok {
        return v
    }
    return "unknown"
}

func callerFromContext(ctx context.Context) string {
    if v, ok := ctx.Value(ContextKeyCallerID).(string); ok {
        return v
    }
    return "unknown"
}
```

In the REST handler, set these before calling the service:

```go
func (h *ProjectContextAIHandler) Ask(c *gin.Context) {
    apiKey := c.GetHeader("X-API-Key")
    callerID := maskKey(apiKey) // show only first 8 chars in panel

    ctx := usecases.WithGate(c.Request.Context(), "rest")
    ctx = usecases.WithCallerID(ctx, callerID)

    answer, err := h.service.Ask(ctx, query)
    ...
}
```

---

## SSE Admin Handler — `internal/rest/admin_events_handler.go`

```go
package rest

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/yourorg/dude-mcp/adapters/eventbus"
)

type AdminEventsHandler struct {
    bus *eventbus.MemoryEventBus
}

// GET /admin/events/stream
// Streams AskEvents as Server-Sent Events to the browser panel.
// Protected by admin API key middleware.
func (h *AdminEventsHandler) Stream(c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no") // disable nginx buffering

    subID := uuid.NewString()
    events, cancel := h.bus.Subscribe(subID)
    defer cancel()

    // send a heartbeat every 15s to keep the connection alive
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()

    c.Stream(func(w io.Writer) bool {
        select {
        case event, ok := <-events:
            if !ok {
                return false
            }
            data, err := json.Marshal(event)
            if err != nil {
                return true
            }
            fmt.Fprintf(w, "data: %s\n\n", data)
            return true

        case <-ticker.C:
            fmt.Fprintf(w, ": heartbeat\n\n")
            return true

        case <-c.Request.Context().Done():
            return false
        }
    })
}
```

---

## Router Wiring — `router.go`

```go
// Admin routes — protected by separate admin key middleware
admin := r.Group("/admin", AdminKeyMiddleware(cfg.AdminKey))
{
    admin.GET("/events/stream", adminEventsHandler.Stream)
}

// AI client routes
api := r.Group("/context-ai", APIKeyMiddleware(cfg.APIKey))
{
    api.GET("/projects",                      contextAIHandler.ListProjects)
    api.GET("/projects/:name/documents",      contextAIHandler.ListDocuments)
    api.POST("/projects/:name/ask",           contextAIHandler.Ask)
    api.POST("/projects/:name/refresh",       contextAIHandler.Refresh)
}
```

---

## Container Wiring — `container.go`

```go
// Event bus
bus := eventbus.NewMemoryEventBus()

// Services
contextAIService := usecases.NewProjectContextAIServiceImpl(
    fsRepo,
    llmGateway,
    documentSelector,
    promptComposer,
    bus,   // ← inject event bus
    cfg,
)

// Handlers
contextAIHandler   := rest.NewProjectContextAIHandler(contextAIService)
adminEventsHandler := rest.NewAdminEventsHandler(bus)
```

---

## Browser Panel — Minimal JavaScript

Your admin panel subscribes to the SSE stream:

```javascript
const es = new EventSource('/admin/events/stream', {
  headers: { 'X-Admin-Key': ADMIN_KEY }
});

es.onmessage = (e) => {
  const event = JSON.parse(e.data);

  switch (event.status) {
    case 'started':
      addRow(event); // new row in the live table
      break;
    case 'streaming':
      updateRow(event.trace_id, { docs: event.selected_docs });
      break;
    case 'done':
      updateRow(event.trace_id, {
        answer:    event.answer,
        sources:   event.sources,
        latency:   event.latency_ms + 'ms',
        model:     event.model,
        status:    'done',
      });
      break;
    case 'error':
      updateRow(event.trace_id, { status: 'error', error: event.error });
      break;
  }
};
```

---

## What the Panel Can Show

| Widget | Data source |
|--------|-------------|
| Live feed table | `AskEvent` stream |
| Questions per project (bar chart) | Aggregate `AskDone` events |
| Avg latency per gate | `latency_ms` on `AskDone` events |
| Top cited sources | Frequency count on `sources` |
| Error rate | `AskError` / `AskDone` ratio |
| Active AI clients | Distinct `caller_id` in last N minutes |
| Gate usage (REST vs MCP) | `gate` field distribution |

---

## Environment Variables

| Var | Default | Purpose |
|-----|---------|---------|
| `ADMIN_KEY` | — | Separate key for `/admin/*` routes |
| `OBSERVABILITY_ENABLED` | `true` | Set to `false` to use `NoopEventBus` |
| `EVENT_BUS_BUFFER` | `64` | Per-subscriber channel buffer size |

---

## Security Notes

- `/admin/events/stream` must be on a **separate auth key** from the AI client key.
- Never expose `question` content in error responses to the AI client — only in the panel.
- `callerID` should be the **first 8 chars** of the API key only — never the full key.
- Do not include the full LLM prompt in events by default — it may contain sensitive context doc content.

---

## Implementation Order

1. Add `AskEvent` struct — `core/domain/events/`
2. Add `EventBus` port — `core/ports/`
3. Add `NoopEventBus` — use in all tests
4. Add `MemoryEventBus` adapter — `adapters/eventbus/`
5. Inject bus into `ProjectContextAIServiceImpl`, emit events in `Ask()`
6. Add context helpers (`WithGate`, `WithCallerID`)
7. Update REST handler to set gate + callerID on context
8. Add `AdminEventsHandler` with SSE stream endpoint
9. Wire in `container.go` and `router.go`
10. Build minimal browser panel consuming the SSE feed
