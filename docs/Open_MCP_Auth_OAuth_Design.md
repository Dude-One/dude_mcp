# Trio Auth — OAuth Design Document

> **Bounded context:** Auth  
> **Purpose:** Issue, validate, and revoke user-specific tokens so external AI clients (ChatGPT, Claude, Cursor) authenticate to Trio as a known user.  
> **Phase:** 1b — required before any ChatGPT integration is real.

---

## Why OAuth and not just an API key

| | Shared API key | OAuth per user |
|---|---|---|
| Who can you identify? | Which GPT called | Which user called |
| If key leaks | All users affected | Only that user's token |
| Per-user access control | No | Yes |
| Per-user activity in panel | No | Yes |
| User can revoke | No | Yes |
| Required for user panel | No | **Yes** |

The moment you provide a panel where users connect **their own** ChatGPT account to your system, OAuth is the only correct mechanism.

---

## How it works end to end

```
1. User logs into YOUR panel (username + password or SSO)

2. User clicks "Connect my ChatGPT"
   → panel opens your Custom GPT link

3. ChatGPT says "Sign in to Trio to continue"
   → redirects to GET /oauth/authorize?client_id=...&redirect_uri=...&state=...

4. Your server shows login page (or detects active session)
   → user approves access
   → your server issues a short-lived auth CODE
   → redirects back to ChatGPT: https://chat.openai.com/...?code=abc123&state=...

5. ChatGPT exchanges code for tokens:
   POST /oauth/token
   { grant_type: authorization_code, code: abc123, client_id: ..., client_secret: ... }
   → your server returns:
   { access_token: "eyJ...", refresh_token: "rfr...", expires_in: 3600, token_type: "Bearer" }

6. ChatGPT stores these tokens PER USER

7. Every API call from that user's ChatGPT:
   POST /context-ai/projects/open-mcp-servermanagement/ask
   Authorization: Bearer eyJ...
   → your middleware decodes token → userID + tenantID + allowedProjects

8. When access_token expires ChatGPT calls:
   POST /oauth/token { grant_type: refresh_token, refresh_token: "rfr..." }
   → your server returns a new access_token

9. User disconnects from panel:
   POST /oauth/revoke { token: "rfr..." }
   → token invalidated → next ChatGPT call gets 401
```

---

## DDD Mapping

| DDD Concept | Artifact |
|-------------|---------|
| Bounded context | `Auth` — separate from `ProjectContextAI` |
| Aggregate root | `ClientToken` |
| Entity | `OAuthClient` (registered GPT application) |
| Value objects | `TokenID`, `AuthCode`, `TokenClaims`, `Scope` |
| Repository port | `TokenRepository`, `OAuthClientRepository`, `AuthCodeRepository` |
| Application service | `OAuthService` |
| Adapter | `MemoryTokenRepository` (Phase 1), `PostgresTokenRepository` (Phase 2) |
| REST handler | `OAuthHandler` — `/oauth/*` endpoints |
| Middleware | `OAuthMiddleware` — validates Bearer on every protected request |

---

## Domain Entities — `core/auth/domain/entities/`

### `OAuthClient` — the registered GPT application

```go
// OAuthClient represents a registered external AI client (e.g. your Custom GPT).
// You create one per AI product that integrates with Trio.
type OAuthClient struct {
    ClientID     string   // "trio-chatgpt-gpt"
    ClientSecret string   // hashed — never stored plain
    Name         string   // "Open MCP ChatGPT Assistant"
    RedirectURIs []string // ["https://chat.openai.com/aip/..."]
    AllowedScopes []Scope // ["context:read"]
    CreatedAt    time.Time
}
```

---

### `ClientToken` — aggregate root

```go
// ClientToken is the aggregate root for an issued OAuth token pair.
// One instance per connected user+client combination.
type ClientToken struct {
    id           TokenID
    userID       string
    tenantID     string
    clientID     string          // which OAuthClient issued this
    accessToken  string          // JWT — short lived (1h)
    refreshToken string          // opaque — long lived (30d)
    scopes       []Scope
    issuedAt     time.Time
    expiresAt    time.Time
    revokedAt    *time.Time

    domainEvents []DomainEvent
}

func IssueToken(userID, tenantID, clientID string, scopes []Scope) *ClientToken {
    now := time.Now()
    t := &ClientToken{
        id:           NewTokenID(),
        userID:       userID,
        tenantID:     tenantID,
        clientID:     clientID,
        scopes:       scopes,
        issuedAt:     now,
        expiresAt:    now.Add(1 * time.Hour),
        accessToken:  generateJWT(userID, tenantID, scopes, now.Add(1*time.Hour)),
        refreshToken: generateOpaqueToken(),
    }
    t.record(TokenIssuedEvent{TokenID: t.id, UserID: userID, ClientID: clientID})
    return t
}

func (t *ClientToken) Refresh() error {
    if t.IsRevoked() {
        return errors.New("token is revoked")
    }
    t.accessToken = generateJWT(t.userID, t.tenantID, t.scopes, time.Now().Add(1*time.Hour))
    t.expiresAt = time.Now().Add(1 * time.Hour)
    t.record(TokenRefreshedEvent{TokenID: t.id})
    return nil
}

func (t *ClientToken) Revoke() {
    now := time.Now()
    t.revokedAt = &now
    t.record(TokenRevokedEvent{TokenID: t.id, UserID: t.userID})
}

func (t *ClientToken) IsRevoked() bool   { return t.revokedAt != nil }
func (t *ClientToken) IsExpired() bool   { return time.Now().After(t.expiresAt) }
func (t *ClientToken) UserID() string    { return t.userID }
func (t *ClientToken) TenantID() string  { return t.tenantID }
func (t *ClientToken) AccessToken() string  { return t.accessToken }
func (t *ClientToken) RefreshToken() string { return t.refreshToken }
```

---

### Value Objects — `core/auth/domain/entities/value_objects.go`

```go
type TokenID string
func NewTokenID() TokenID { return TokenID(uuid.NewString()) }

type Scope string
const (
    ScopeContextRead  Scope = "context:read"
    ScopeContextWrite Scope = "context:write"  // future
    ScopeAdminRead    Scope = "admin:read"      // future
)

// TokenClaims — decoded from JWT, passed through middleware to use cases
type TokenClaims struct {
    UserID          string
    TenantID        string
    ClientID        string
    Scopes          []Scope
    AllowedProjects []string  // loaded from user's profile
    ExpiresAt       time.Time
}

// AuthCode — short-lived code exchanged for tokens in step 4-5 of flow
type AuthCode struct {
    Code      string
    ClientID  string
    UserID    string
    TenantID  string
    ExpiresAt time.Time
    Used      bool
}
```

---

## Ports — `core/auth/ports/`

```go
// TokenRepository stores and retrieves issued token pairs
type TokenRepository interface {
    Save(ctx context.Context, token *entities.ClientToken) error
    FindByRefreshToken(ctx context.Context, refreshToken string) (*entities.ClientToken, error)
    FindByUserAndClient(ctx context.Context, userID, clientID string) (*entities.ClientToken, error)
    Revoke(ctx context.Context, id entities.TokenID) error
}

// OAuthClientRepository stores registered OAuth clients (your GPT apps)
type OAuthClientRepository interface {
    FindByClientID(ctx context.Context, clientID string) (*entities.OAuthClient, error)
}

// AuthCodeRepository stores short-lived auth codes
type AuthCodeRepository interface {
    Save(ctx context.Context, code entities.AuthCode) error
    Consume(ctx context.Context, code string) (*entities.AuthCode, error) // marks as used, returns it
}

// OAuthService — application port
type OAuthService interface {
    // Step 4 — user approved, issue auth code
    Authorize(ctx context.Context, clientID, userID, tenantID string, scopes []entities.Scope) (code string, err error)

    // Step 5 — exchange code for tokens
    ExchangeCode(ctx context.Context, clientID, clientSecret, code string) (*entities.ClientToken, error)

    // Step 8 — refresh expired access token
    RefreshToken(ctx context.Context, clientID, clientSecret, refreshToken string) (*entities.ClientToken, error)

    // Step 9 — user revokes from panel
    RevokeToken(ctx context.Context, token string) error

    // Called by middleware on every protected request
    ValidateAccessToken(ctx context.Context, accessToken string) (*entities.TokenClaims, error)
}
```

---

## Application Service — `core/auth/usecases/oauth_service_impl.go`

```go
type OAuthServiceImpl struct {
    tokens  ports.TokenRepository
    clients ports.OAuthClientRepository
    codes   ports.AuthCodeRepository
    jwtKey  []byte
}

func (s *OAuthServiceImpl) ExchangeCode(
    ctx context.Context,
    clientID, clientSecret, code string,
) (*entities.ClientToken, error) {
    // 1. Validate client
    client, err := s.clients.FindByClientID(ctx, clientID)
    if err != nil || !client.ValidateSecret(clientSecret) {
        return nil, errors.New("invalid client credentials")
    }

    // 2. Consume auth code (single use)
    authCode, err := s.codes.Consume(ctx, code)
    if err != nil || authCode.Used || time.Now().After(authCode.ExpiresAt) {
        return nil, errors.New("invalid or expired code")
    }

    // 3. Issue token pair
    token := entities.IssueToken(
        authCode.UserID,
        authCode.TenantID,
        clientID,
        []entities.Scope{entities.ScopeContextRead},
    )

    // 4. Save
    if err := s.tokens.Save(ctx, token); err != nil {
        return nil, err
    }

    return token, nil
}

func (s *OAuthServiceImpl) ValidateAccessToken(
    ctx context.Context,
    accessToken string,
) (*entities.TokenClaims, error) {
    claims, err := parseJWT(accessToken, s.jwtKey)
    if err != nil || time.Now().After(claims.ExpiresAt) {
        return nil, errors.New("invalid or expired token")
    }
    return claims, nil
}
```

---

## REST Endpoints — `internal/auth/oauth_handler.go`

### `GET /oauth/authorize`

ChatGPT redirects the user here for login.

```
Query params:
  client_id     string  required
  redirect_uri  string  required
  response_type string  "code"
  state         string  required (CSRF protection — must be echoed back)
  scope         string  "context:read"

Flow:
  → validate client_id exists + redirect_uri is whitelisted
  → if user has active panel session → show "Approve Open MCP ChatGPT?" screen
  → if no session → show login form
  → on approval → issue auth code → redirect to redirect_uri?code=xxx&state=xxx
```

### `POST /oauth/token`

ChatGPT calls this to get tokens.

```
Body (application/x-www-form-urlencoded):
  grant_type    "authorization_code" | "refresh_token"
  code          string  (for authorization_code)
  refresh_token string  (for refresh_token)
  client_id     string
  client_secret string
  redirect_uri  string

Response 200:
{
  "access_token":  "eyJhbGci...",
  "refresh_token": "rfr_abc123...",
  "token_type":    "Bearer",
  "expires_in":    3600
}

Error 400/401:
{
  "error": "invalid_grant",
  "error_description": "Authorization code expired"
}
```

### `POST /oauth/revoke`

User disconnects from your panel.

```
Body:
  token  string  (refresh_token to revoke)

Response 200: {} (always 200 per OAuth spec, even if token not found)
```

---

## Middleware — `internal/middleware/oauth_middleware.go`

Replaces API key check for all user-facing routes.

```go
func OAuthMiddleware(oauthService ports.OAuthService) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if !strings.HasPrefix(header, "Bearer ") {
            c.JSON(401, gin.H{"error": "missing bearer token"})
            c.Abort()
            return
        }

        token := strings.TrimPrefix(header, "Bearer ")
        claims, err := oauthService.ValidateAccessToken(c.Request.Context(), token)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid or expired token"})
            c.Abort()
            return
        }

        // Attach identity to context for use cases
        ctx := withUserID(c.Request.Context(), claims.UserID)
        ctx = withTenantID(ctx, claims.TenantID)
        ctx = withAllowedProjects(ctx, claims.AllowedProjects)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
```

---

## ChatGPT GPT Actions Config

Set this in the OpenAI Custom GPT dashboard under **Configure → Actions → Authentication**:

```
Authentication type: OAuth
Client ID:           trio-chatgpt-gpt
Client Secret:       (generated secret from your system)
Authorization URL:   https://your-server.com/oauth/authorize
Token URL:           https://your-server.com/oauth/token
Scope:               context:read
Token Exchange:      POST
```

---

## Package Layout

```
core/
  auth/
    domain/
      entities/
        client_token.go         ← aggregate root
        oauth_client.go         ← registered GPT app
        value_objects.go        ← TokenID, Scope, TokenClaims, AuthCode
      events/
        token_events.go         ← TokenIssued, TokenRefreshed, TokenRevoked
    ports/
      token_repository.go
      oauth_client_repository.go
      auth_code_repository.go
      oauth_service.go

    usecases/
      oauth_service_impl.go

adapters/
  auth/
    memory_token_repository.go       ← Phase 1 (in-process)
    memory_auth_code_repository.go
    memory_oauth_client_repository.go
    postgres_token_repository.go     ← Phase 2 (persistent)

internal/
  auth/
    oauth_handler.go                 ← /oauth/authorize, /oauth/token, /oauth/revoke
  middleware/
    oauth_middleware.go              ← Bearer token validation
```

---

## Security Rules

- **Never store access tokens** — only store refresh tokens (hashed)
- **Auth codes expire in 5 minutes** and are single-use
- **Refresh tokens expire in 30 days**
- **Access tokens expire in 1 hour** (JWT, validated without DB lookup)
- **client_secret** stored as bcrypt hash — never plain text
- **Redirect URIs** must be exact match (no wildcards) — prevent open redirect
- **state parameter** must be echoed back exactly — CSRF protection
- **HTTPS only** — never serve OAuth endpoints over HTTP
- **Revoke all tokens** when user deletes their Trio account

---

## Implementation Order

| Step | File | What |
|------|------|------|
| 1 | `core/auth/domain/entities/value_objects.go` | TokenID, Scope, TokenClaims, AuthCode |
| 2 | `core/auth/domain/entities/oauth_client.go` | OAuthClient entity |
| 3 | `core/auth/domain/entities/client_token.go` | Aggregate root + IssueToken, Refresh, Revoke |
| 4 | `core/auth/domain/events/token_events.go` | TokenIssued, Refreshed, Revoked |
| 5 | `core/auth/ports/` | 4 port interfaces |
| 6 | `core/auth/usecases/oauth_service_impl.go` | Authorize, ExchangeCode, RefreshToken, Revoke, Validate |
| 7 | `adapters/auth/memory_*.go` | 3 in-memory repositories |
| 8 | `internal/auth/oauth_handler.go` | 3 OAuth endpoints |
| 9 | `internal/middleware/oauth_middleware.go` | Bearer token middleware |
| 10 | `internal/app/container.go` | Wire auth into container |
| 11 | GPT Actions dashboard | Set OAuth config, test flow |

---

## What NOT to Do

- Do not skip OAuth and use a shared API key when users connect their own ChatGPT — you will have no user identity
- Do not store plain access tokens in the database — only store refresh tokens (hashed)
- Do not accept auth codes older than 5 minutes — replay attack prevention
- Do not implement token logic inside the REST handler — delegate to `OAuthService` port only
- Do not share `ClientToken` aggregate with `ProjectContextAI` bounded context — pass only `TokenClaims` value object through context
