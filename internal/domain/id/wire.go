// Package id is the ID (identity) domain: IdP, panel auth, sessions, and the
// per-user context notes served to MCP. It assembles its own dependency graph
// and self-registers its HTTP routes.
package id

import (
	"github.com/gin-gonic/gin"
	"github.com/open-mcp/open-mcp/internal/domain/id/adapter"
	"github.com/open-mcp/open-mcp/internal/domain/id/delivery"
	idmiddleware "github.com/open-mcp/open-mcp/internal/domain/id/middleware"
	"github.com/open-mcp/open-mcp/internal/domain/id/service"
	"gorm.io/gorm"
)

// Module is the ID domain's composition root.
type Module struct {
	panel     *delivery.PanelHandler
	context   *delivery.ContextHandler
	adminUser *delivery.AdminUserHandler
	authSvc   service.AuthService
}

// NewModule wires the ID domain against the given database handle.
func NewModule(db *gorm.DB) *Module {
	userRepo := adapter.NewPostgresUserRepository(db)
	tenantRepo := adapter.NewPostgresTenantRepository(db)
	sessionRepo := adapter.NewPostgresSessionRepository(db)

	authSvc := service.NewAuthService(userRepo, tenantRepo, sessionRepo)

	return &Module{
		panel:     delivery.NewPanelHandler(authSvc),
		context:   delivery.NewContextHandler("./context/project_id"),
		adminUser: delivery.NewAdminUserHandler(userRepo),
		authSvc:   authSvc,
	}
}

// AuthService exposes the auth service for cross-domain wiring (e.g. MCP admin middleware).
func (m *Module) AuthService() service.AuthService { return m.authSvc }

// RegisterRoutes mounts the ID domain's routes on the shared router.
func (m *Module) RegisterRoutes(r gin.IRouter) {
	delivery.RegisterRoutes(r, m.panel, m.context)
}

// RegisterAdminRoutes mounts admin-only routes.
func (m *Module) RegisterAdminRoutes(rg *gin.RouterGroup) {
	adminOnly := idmiddleware.AdminOnly(m.authSvc)
	users := rg.Group("/users", adminOnly)
	{
		users.GET("", m.adminUser.ListUsers)
		users.PUT("/:id/role", m.adminUser.UpdateUserRole)
		users.PUT("/:id/projects", m.adminUser.UpdateUserProjects)
	}
}
