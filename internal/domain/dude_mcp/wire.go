// Package dudemcp is the MCP platform domain: source registry, swagger discovery,
// context doc serving, and API proxy. It assembles its own dependency graph.
package dudemcp

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/server"
	"github.com/Dude-One/dude_mcp/internal/domain/id/middleware"
	"github.com/Dude-One/dude_mcp/internal/domain/id/service"
	"github.com/Dude-One/dude_mcp/internal/domain/dude_mcp/adapter"
	"github.com/Dude-One/dude_mcp/internal/domain/dude_mcp/delivery"
	dudemcpservice "github.com/Dude-One/dude_mcp/internal/domain/dude_mcp/service"
	"gorm.io/gorm"
)

// Module is the MCP domain's composition root.
type Module struct {
	Server      *server.MCPServer
	adminSource *delivery.AdminSourceHandler
	authSvc     service.AuthService
	registry    *adapter.DBRegistry
}

// Reload invalidates the DB registry cache so the next tool call re-fetches
// from the database. Call this after any admin panel source change.
func (m *Module) Reload() error {
	return m.registry.Reload()
}

// NewModule wires the MCP domain from DB + sources.yaml (seed only).
//
//   - sourcesPath: path to sources.yaml — used to seed the DB on first run only.
//   - db:          Postgres handle — source of truth after first run.
//   - authSvc:     id domain auth service — used by admin middleware.
func NewModule(sourcesPath string, db *gorm.DB, authSvc service.AuthService) (*Module, error) {
	ttlSec, _ := strconv.Atoi(os.Getenv("SWAGGER_CACHE_TTL"))
	if ttlSec <= 0 {
		ttlSec = 300
	}
	cacheTTL := time.Duration(ttlSec) * time.Second

	// Seed DB from YAML on first run (when mcp_sources is empty).
	yamlReg, err := adapter.NewYAMLRegistry(sourcesPath)
	if err != nil {
		return nil, fmt.Errorf("loading yaml registry for seed: %w", err)
	}
	// context.Background() is acceptable here — this runs once at startup.
	if err := adapter.SeedFromYAML(context.Background(), db, yamlReg); err != nil {
		return nil, fmt.Errorf("seeding sources from yaml: %w", err)
	}

	// DB registry is the live source of truth.
	dbReg := adapter.NewDBRegistry(db)

	swaggerReader := adapter.NewHTTPSwaggerReader(cacheTTL)
	contextReader := adapter.NewHTTPContextReader()
	apiCaller := adapter.NewHTTPAPICaller()

	svc := dudemcpservice.NewMCPService(dbReg, swaggerReader, contextReader, apiCaller)
	handlers := delivery.NewHandlers(svc)
	mcpServer := delivery.NewMCPServer(handlers)

	adminRepo := adapter.NewPostgresAdminSourceRepository(db)
	adminSource := delivery.NewAdminSourceHandler(adminRepo)

	return &Module{
		Server:      mcpServer,
		adminSource: adminSource,
		authSvc:     authSvc,
		registry:    dbReg,
	}, nil
}

// RegisterAdminRoutes mounts admin source management routes.
func (m *Module) RegisterAdminRoutes(rg *gin.RouterGroup) {
	adminOnly := middleware.AdminOnly(m.authSvc)
	sources := rg.Group("/sources", adminOnly)
	{
		sources.GET("", m.adminSource.ListSources)
		sources.POST("", m.adminSource.CreateSource)
		sources.GET("/:id", m.adminSource.GetSource)
		sources.PUT("/:id", m.adminSource.UpdateSource)
		sources.DELETE("/:id", m.adminSource.DeleteSource)
		sources.PUT("/:id/toggle", m.adminSource.ToggleSource)
	}
}
