# AI Observability — DDD Design Document

> **Open MCP** — Separate bounded context for capturing, storing, and streaming AI interaction traces.

---

## Overview

Every `Ask()` call that flows through the server is an **interaction** worth observing.  
This bounded context captures each interaction as an `InteractionTrace` aggregate, computes stats, and streams live events to an admin panel via SSE.

**What it is NOT:**
- It is not a logging library (`log.Info(...)` is infrastructure, not domain).
- It is not part of `ProjectContextAI` — it is a **separate bounded context** that listens to events from it.
- It does not care about LLM internals — it only cares about observable inputs and outputs.

**What it IS:**
- A first-class domain that owns the lifecycle of an `InteractionTrace`.
- The single source of truth for "what questions were asked, by whom, answered how fast, citing which docs."
- The publisher of a live event stream that any panel can subscribe to.

---

## Bounded Context Map

Two contexts. One publishes, one listens. They never share entities.

```
┌─────────────────────────────────┐        ┌──────────────────────────────────┐
│   ProjectContextAI (upstream)   │        │   AIObservability (downstream)   │
│                                 │        │                                  │
│   ProjectContextAIServiceImpl   │──────► │   TraceEventTranslator  (ACL)    │
│   publishes: AskEvent (DTO)     │ events │   translates → domain commands   │
│                                 │        │   → InteractionTrace aggregate   │
└─────────────────────────────────┘        └──────────────────────────────────┘
```

The **Anti-Corruption Layer (ACL)** is `TraceEventTranslator`.  
It translates the upstream `AskEvent` DTO into domain commands that the `InteractionTrace` aggregate understands.  
The observability domain never imports anything from `ProjectContextAI` except the raw event DTO.

---

## DDD Mapping

| DDD Concept | In this repo | Artifact |
|-------------|-------------|---------|
| **Bounded context** | separate from `ProjectContextAI` | `AIObservability` |
| **Aggregate root** | `core/observability/domain/entities/` | `InteractionTrace` |
| **Value objects** | typed, immutable, no identity | `TraceID`, `CallerIdentity`, `InteractionStatus`, `LatencyProfile`, `DocumentSelection`, `TokenUsage` |
| **Domain events** | raised by `InteractionTrace` | `InteractionStarted`, `DocumentsSelected`, `LLMCallMade`, `InteractionCompleted`, `InteractionFailed` |
| **Domain service** | `core/observability/domain/services/` | `TraceStatsCalculator` |
| **Repository port** | `core/observability/ports/` | `InteractionTraceRepository` |
| **Event stream port** | `core/observability/ports/` | `ObservabilityEventStream` |
| **Application service** | `core/observability/usecases/` | `ObservabilityServiceImpl` |
| **ACL adapter** | `adapters/observability/` | `TraceEventTranslator` |
| **Repository adapter** | `adapters/observability/` | `MemoryInteractionTraceRepository` |
| **Stream adapter** | `adapters/observability/` | `SSEEventStreamAdapter` |
| **REST handler** | `internal/rest/` | `AdminObservabilityHandler` |

---

## Domain Model

### Aggregate Root — `InteractionTrace`

The central entity. One instance per `Ask()` call. Mutable — its state evolves from `Started` → `Done` or `Error`.

```go
// core/observability/domain/entities/interaction_trace.go

type InteractionTrace struct {
    id             TraceID
    project        ProjectName
    caller         CallerIdentity
    question       string
    status         InteractionStatus
    selection      *DocumentSelection   // set after selector runs
    llmCall        *LLMCall             // set after LLM responds
    answer         string               // set on completion
    sources        []DocumentRef        // set on completion
    latency        LatencyProfile
    startedAt      time.Time
    completedAt    *time.Time
    errorMsg       string

    domainEvents   []DomainEvent        // uncommitted events
}

// Constructors (factory methods — no naked struct literals outside domain)

func StartInteraction(id TraceID, project ProjectName, caller CallerIdentity, question string) *InteractionTrace {
    t := &InteractionTrace{
        id:        id,
        project:   project,
        caller:    caller,
        question:  question,
        status:    StatusStarted,
        startedAt: time.Now(),
    }
    t.record(InteractionStartedEvent{
        TraceID:   id,
        Project:   project,
        Caller:    caller,
        Question:  question,
        Timestamp: t.startedAt,
    })
    return t
}

// State transitions — only the aggregate changes its own state

func (t *InteractionTrace) RecordDocumentsSelected(sel DocumentSelection) {
    t.selection = &sel
    t.latency.SelectionMs = time.Since(t.startedAt).Milliseconds()
    t.record(DocumentsSelectedEvent{TraceID: t.id, Selection: sel})
}

func (t *InteractionTrace) RecordLLMCall(call LLMCall) {
    t.llmCall = &call
    t.record(LLMCallMadeEvent{TraceID: t.id, Call: call})
}

func (t *InteractionTrace) Complete(answer string, sources []DocumentRef) {
    now := time.Now()
    t.answer = answer
    t.sources = sources
    t.status = StatusDone
    t.completedAt = &now
    t.latency.TotalMs = time.Since(t.startedAt).Milliseconds()
    t.latency.LLMMs = t.latency.TotalMs - t.latency.SelectionMs
    t.record(InteractionCompletedEvent{
        TraceID:   t.id,
        Answer:    answer,
        Sources:   sources,
        LatencyMs: t.latency.TotalMs,
        Timestamp: now,
    })
}

func (t *InteractionTrace) Fail(step string, err error) {
    now := time.Now()
    t.status = StatusError
    t.errorMsg = err.Error()
    t.completedAt = &now
    t.latency.TotalMs = time.Since(t.startedAt).Milliseconds()
    t.record(InteractionFailedEvent{
        TraceID:   t.id,
        Step:      step,
        Error:     err.Error(),
        LatencyMs: t.latency.TotalMs,
        Timestamp: now,
    })
}

// Pull uncommitted domain events (called by repository before saving)
func (t *InteractionTrace) DomainEvents() []DomainEvent { return t.domainEvents }
func (t *InteractionTrace) ClearEvents()                { t.domainEvents = nil }
func (t *InteractionTrace) record(e DomainEvent)        { t.domainEvents = append(t.domainEvents, e) }
```

---

### Value Objects — `core/observability/domain/entities/value_objects.go`

All value objects are **immutable** and compared by value, not identity.

```go
// TraceID — typed UUID wrapper
type TraceID string
func NewTraceID() TraceID { return TraceID(uuid.NewString()) }

// CallerIdentity — who made the call
type CallerIdentity struct {
    Gate     string // "rest" | "mcp"
    CallerID string // masked API key prefix or MCP client name
}

// InteractionStatus — lifecycle state
type InteractionStatus string
const (
    StatusStarted   InteractionStatus = "started"
    StatusDone      InteractionStatus = "done"
    StatusError     InteractionStatus = "error"
)

// DocumentSelection — output of DocumentSelector
type DocumentSelection struct {
    Paths  []string  // relative doc paths
    Scores []float64 // relevance scores (0–1), parallel to Paths
}

// LLMCall — what was sent to and received from the LLM
type LLMCall struct {
    Model            string
    PromptTokens     int
    CompletionTokens int
}

// LatencyProfile — timing breakdown per pipeline step (ms)
type LatencyProfile struct {
    SelectionMs int64
    LLMMs       int64
    TotalMs     int64
}

// DocumentRef — a cited source
type DocumentRef struct {
    Path  string
    Title string
}

// ProjectName — shared with ProjectContextAI via shared kernel (copy, not import)
type ProjectName string

// TraceStats — computed aggregate view (output of TraceStatsCalculator)
type TraceStats struct {
    TotalCount      int
    ErrorCount      int
    AvgLatencyMs    int64
    GateDistribution map[string]int
    TopSources      []SourceFrequency
    ActiveCallers   []string
}

type SourceFrequency struct {
    Path  string
    Count int
}
```

---

### Domain Events — `core/observability/domain/events/`

Raised by `InteractionTrace` aggregate transitions. Consumed by `SSEEventStreamAdapter` to push to the panel.

```go
type DomainEvent interface {
    EventType() string
    OccurredAt() time.Time
}

type InteractionStartedEvent struct {
    TraceID   TraceID
    Project   ProjectName
    Caller    CallerIdentity
    Question  string
    Timestamp time.Time
}
func (e InteractionStartedEvent) EventType() string    { return "interaction.started" }
func (e InteractionStartedEvent) OccurredAt() time.Time { return e.Timestamp }

type DocumentsSelectedEvent struct {
    TraceID   TraceID
    Selection DocumentSelection
    Timestamp time.Time
}
func (e DocumentsSelectedEvent) EventType() string    { return "interaction.docs_selected" }
func (e DocumentsSelectedEvent) OccurredAt() time.Time { return e.Timestamp }

type LLMCallMadeEvent struct {
    TraceID   TraceID
    Call      LLMCall
    Timestamp time.Time
}
func (e LLMCallMadeEvent) EventType() string    { return "interaction.llm_called" }
func (e LLMCallMadeEvent) OccurredAt() time.Time { return e.Timestamp }

type InteractionCompletedEvent struct {
    TraceID   TraceID
    Answer    string
    Sources   []DocumentRef
    LatencyMs int64
    Timestamp time.Time
}
func (e InteractionCompletedEvent) EventType() string    { return "interaction.completed" }
func (e InteractionCompletedEvent) OccurredAt() time.Time { return e.Timestamp }

type InteractionFailedEvent struct {
    TraceID   TraceID
    Step      string
    Error     string
    LatencyMs int64
    Timestamp time.Time
}
func (e InteractionFailedEvent) EventType() string    { return "interaction.failed" }
func (e InteractionFailedEvent) OccurredAt() time.Time { return e.Timestamp }
```

---

### Domain Service — `core/observability/domain/services/trace_stats_calculator.go`

Pure function — no side effects, no I/O. Takes traces, returns stats.

```go
type TraceStatsCalculator struct{}

func (c *TraceStatsCalculator) Calculate(traces []*entities.InteractionTrace) entities.TraceStats {
    stats := entities.TraceStats{
        GateDistribution: make(map[string]int),
    }
    sourceCount := make(map[string]int)
    callerSet := make(map[string]struct{})
    var totalLatency int64
    doneCount := 0

    for _, t := range traces {
        stats.TotalCount++
        if t.Status() == entities.StatusError {
            stats.ErrorCount++
        }
        if t.Status() == entities.StatusDone {
            totalLatency += t.Latency().TotalMs
            doneCount++
        }
        stats.GateDistribution[t.Caller().Gate]++
        callerSet[t.Caller().CallerID] = struct{}{}
        for _, s := range t.Sources() {
            sourceCount[s.Path]++
        }
    }

    if doneCount > 0 {
        stats.AvgLatencyMs = totalLatency / int64(doneCount)
    }
    for caller := range callerSet {
        stats.ActiveCallers = append(stats.ActiveCallers, caller)
    }
    // top 5 sources by frequency
    stats.TopSources = topN(sourceCount, 5)

    return stats
}
```

---

## Ports

### Repository — `core/observability/ports/interaction_trace_repository.go`

```go
type InteractionTraceRepository interface {
    Save(ctx context.Context, trace *entities.InteractionTrace) error
    FindByID(ctx context.Context, id entities.TraceID) (*entities.InteractionTrace, error)
    FindRecent(ctx context.Context, limit int) ([]*entities.InteractionTrace, error)
    FindByProject(ctx context.Context, project entities.ProjectName, limit int) ([]*entities.InteractionTrace, error)
}
```

### Event Stream — `core/observability/ports/observability_event_stream.go`

```go
// ObservabilityEventStream is the outbound port for streaming domain events
// to subscribers (SSE panel, WebSocket, etc.).
type ObservabilityEventStream interface {
    // Publish sends a domain event to all active subscribers.
    Publish(event events.DomainEvent)
    // Subscribe returns a channel of events and a cancel func.
    // The caller MUST call cancel() when done to avoid goroutine leaks.
    Subscribe(subscriberID string) (<-chan events.DomainEvent, func())
}
```

### Application Service — `core/observability/ports/observability_service.go`

```go
type TraceFilter struct {
    Project *entities.ProjectName
    Gate    *string
    Status  *entities.InteractionStatus
    Limit   int
}

type ObservabilityService interface {
    // Called by the ACL adapter — these are the only write paths
    HandleInteractionStarted(ctx context.Context, e events.InteractionStartedEvent) error
    HandleDocumentsSelected(ctx context.Context, e events.DocumentsSelectedEvent) error
    HandleLLMCallMade(ctx context.Context, e events.LLMCallMadeEvent) error
    HandleInteractionCompleted(ctx context.Context, e events.InteractionCompletedEvent) error
    HandleInteractionFailed(ctx context.Context, e events.InteractionFailedEvent) error

    // Query — panel reads
    GetRecentTraces(ctx context.Context, filter TraceFilter) ([]*entities.InteractionTrace, error)
    GetStats(ctx context.Context, filter TraceFilter) (*entities.TraceStats, error)

    // Stream — SSE subscription
    Subscribe(subscriberID string) (<-chan events.DomainEvent, func())
}
```

---

## Application Service — `core/observability/usecases/observability_service_impl.go`

Orchestrates: receive event → find/create trace → call aggregate method → save → stream events.

```go
type ObservabilityServiceImpl struct {
    repo   ports.InteractionTraceRepository
    stream ports.ObservabilityEventStream
    stats  *services.TraceStatsCalculator
}

func (s *ObservabilityServiceImpl) HandleInteractionStarted(
    ctx context.Context,
    e events.InteractionStartedEvent,
) error {
    trace := entities.StartInteraction(e.TraceID, e.Project, e.Caller, e.Question)
    if err := s.repo.Save(ctx, trace); err != nil {
        return err
    }
    s.publishDomainEvents(trace)
    return nil
}

func (s *ObservabilityServiceImpl) HandleInteractionCompleted(
    ctx context.Context,
    e events.InteractionCompletedEvent,
) error {
    trace, err := s.repo.FindByID(ctx, e.TraceID)
    if err != nil {
        return err
    }
    trace.Complete(e.Answer, e.Sources)
    if err := s.repo.Save(ctx, trace); err != nil {
        return err
    }
    s.publishDomainEvents(trace)
    return nil
}

func (s *ObservabilityServiceImpl) GetStats(
    ctx context.Context,
    filter ports.TraceFilter,
) (*entities.TraceStats, error) {
    traces, err := s.repo.FindRecent(ctx, 500) // last 500 for stats
    if err != nil {
        return nil, err
    }
    result := s.stats.Calculate(traces)
    return &result, nil
}

func (s *ObservabilityServiceImpl) Subscribe(id string) (<-chan events.DomainEvent, func()) {
    return s.stream.Subscribe(id)
}

func (s *ObservabilityServiceImpl) publishDomainEvents(trace *entities.InteractionTrace) {
    for _, e := range trace.DomainEvents() {
        s.stream.Publish(e)
    }
    trace.ClearEvents()
}
```

---

## Anti-Corruption Layer — `adapters/observability/trace_event_translator.go`

Translates the upstream `AskEvent` DTO (from `ProjectContextAI`) into domain commands for `ObservabilityService`.  
This is the **only** file that imports from both bounded contexts.

```go
// TraceEventTranslator is the ACL between ProjectContextAI and AIObservability.
// It subscribes to AskEvent (upstream DTO) and translates to ObservabilityService calls.
type TraceEventTranslator struct {
    service obsPorts.ObservabilityService
}

func (t *TraceEventTranslator) Handle(ctx context.Context, raw aiEvents.AskEvent) {
    switch raw.Status {
    case aiEvents.AskStarted:
        t.service.HandleInteractionStarted(ctx, obsEvents.InteractionStartedEvent{
            TraceID: obsEntities.TraceID(raw.TraceID),
            Project: obsEntities.ProjectName(raw.Project),
            Caller:  obsEntities.CallerIdentity{Gate: raw.Gate, CallerID: raw.CallerID},
            Question: raw.Question,
            Timestamp: raw.Timestamp,
        })

    case aiEvents.AskStreaming:
        if len(raw.SelectedDocs) > 0 {
            t.service.HandleDocumentsSelected(ctx, obsEvents.DocumentsSelectedEvent{
                TraceID:   obsEntities.TraceID(raw.TraceID),
                Selection: obsEntities.DocumentSelection{Paths: raw.SelectedDocs},
                Timestamp: raw.Timestamp,
            })
        }

    case aiEvents.AskDone:
        sources := make([]obsEntities.DocumentRef, len(raw.Sources))
        for i, s := range raw.Sources {
            sources[i] = obsEntities.DocumentRef{Path: s}
        }
        t.service.HandleInteractionCompleted(ctx, obsEvents.InteractionCompletedEvent{
            TraceID:   obsEntities.TraceID(raw.TraceID),
            Answer:    raw.Answer,
            Sources:   sources,
            LatencyMs: raw.LatencyMs,
            Timestamp: raw.Timestamp,
        })

    case aiEvents.AskError:
        t.service.HandleInteractionFailed(ctx, obsEvents.InteractionFailedEvent{
            TraceID:   obsEntities.TraceID(raw.TraceID),
            Error:     raw.Error,
            LatencyMs: raw.LatencyMs,
            Timestamp: raw.Timestamp,
        })
    }
}
```

---

## Adapters

### Repository — `adapters/observability/memory_trace_repository.go`

Ring buffer of the last N traces. Thread-safe. Zero dependencies.

```go
type MemoryInteractionTraceRepository struct {
    mu     sync.RWMutex
    traces map[entities.TraceID]*entities.InteractionTrace
    order  []entities.TraceID // insertion order for FindRecent
    limit  int                // max traces to keep in memory
}

func NewMemoryInteractionTraceRepository(limit int) *MemoryInteractionTraceRepository {
    return &MemoryInteractionTraceRepository{
        traces: make(map[entities.TraceID]*entities.InteractionTrace),
        limit:  limit,
    }
}

func (r *MemoryInteractionTraceRepository) Save(_ context.Context, trace *entities.InteractionTrace) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    if _, exists := r.traces[trace.ID()]; !exists {
        r.order = append(r.order, trace.ID())
        if len(r.order) > r.limit {
            oldest := r.order[0]
            r.order = r.order[1:]
            delete(r.traces, oldest)
        }
    }
    r.traces[trace.ID()] = trace
    return nil
}

func (r *MemoryInteractionTraceRepository) FindRecent(_ context.Context, limit int) ([]*entities.InteractionTrace, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    start := len(r.order) - limit
    if start < 0 { start = 0 }
    result := make([]*entities.InteractionTrace, 0, limit)
    for _, id := range r.order[start:] {
        result = append(result, r.traces[id])
    }
    return result, nil
}
```

### Event Stream — `adapters/observability/memory_event_stream.go`

Fan-out to all SSE subscribers.

```go
type MemoryObservabilityEventStream struct {
    mu          sync.RWMutex
    subscribers map[string]chan events.DomainEvent
}

func (s *MemoryObservabilityEventStream) Publish(event events.DomainEvent) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, ch := range s.subscribers {
        select {
        case ch <- event:
        default: // slow subscriber — drop, never block the pipeline
        }
    }
}

func (s *MemoryObservabilityEventStream) Subscribe(id string) (<-chan events.DomainEvent, func()) {
    ch := make(chan events.DomainEvent, 64)
    s.mu.Lock()
    s.subscribers[id] = ch
    s.mu.Unlock()
    return ch, func() {
        s.mu.Lock()
        delete(s.subscribers, id)
        close(ch)
        s.mu.Unlock()
    }
}
```

---

## Package Layout

```
core/
  observability/
    domain/
      entities/
        interaction_trace.go     ← aggregate root
        value_objects.go         ← TraceID, CallerIdentity, LatencyProfile, etc.
      events/
        domain_events.go         ← InteractionStarted, Completed, Failed, etc.
      services/
        trace_stats_calculator.go ← pure domain service
    ports/
      interaction_trace_repository.go
      observability_event_stream.go
      observability_service.go
    usecases/
      observability_service_impl.go

adapters/
  observability/
    memory_trace_repository.go   ← in-memory ring buffer
    memory_event_stream.go       ← fan-out SSE publisher
    trace_event_translator.go    ← ACL: AskEvent → domain commands

internal/
  rest/
    admin_observability_handler.go ← GET /admin/events/stream, GET /admin/traces
```

---

## Dependency Rule (Hexagonal)

```
[REST Handler]──────────────────────────────────────────► port only
      │                                                ObservabilityService
      ▼
[ObservabilityServiceImpl] ─────────────────────────► port only
      │                     InteractionTraceRepository
      │                     ObservabilityEventStream
      ▼
[InteractionTrace aggregate]
      │  raises domain events
      ▼
[MemoryObservabilityEventStream] ─────────────────► SSE adapter
      │
      ▼
[Admin Panel browser]

ACL (TraceEventTranslator):
[ProjectContextAI AskEvent] → translated → [ObservabilityServiceImpl.Handle*()]
```

**Rules enforced:**
- `core/observability/` never imports `adapters/` or `internal/`
- `core/observability/` never imports `core/projectcontextai/`
- Only `adapters/observability/trace_event_translator.go` imports from both contexts
- `InteractionTrace` aggregate is the **only** thing that changes trace state

---

## Implementation Order

| Step | File | What |
|------|------|------|
| 1 | `core/observability/domain/entities/value_objects.go` | All value objects |
| 2 | `core/observability/domain/events/domain_events.go` | All 5 domain events |
| 3 | `core/observability/domain/entities/interaction_trace.go` | Aggregate root + state machine |
| 4 | `core/observability/domain/services/trace_stats_calculator.go` | Pure stats service |
| 5 | `core/observability/ports/` | 3 port interfaces |
| 6 | `core/observability/usecases/observability_service_impl.go` | App service |
| 7 | `adapters/observability/memory_trace_repository.go` | Ring buffer repo |
| 8 | `adapters/observability/memory_event_stream.go` | Fan-out stream |
| 9 | `adapters/observability/trace_event_translator.go` | ACL |
| 10 | `internal/rest/admin_observability_handler.go` | SSE + query endpoints |
| 11 | `container.go` + `router.go` | Wire it all together |

---

## What NOT to Do

- Do not call `trace.Complete()` from outside the aggregate — state transitions are the aggregate's responsibility only.
- Do not let `ObservabilityServiceImpl` import anything from `adapters/` — only port interfaces.
- Do not share `InteractionTrace` entity with `ProjectContextAI` — they are in different bounded contexts.
- Do not put stats calculation inside the repository — it belongs in `TraceStatsCalculator` domain service.
- Do not use a global event bus — inject `ObservabilityEventStream` port through the constructor.
- Do not expose raw `InteractionTrace` structs over SSE — serialize to a public DTO in the REST handler.
