package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes self-registers the ID domain's HTTP routes on the shared router.
func RegisterRoutes(r gin.IRouter, panel *PanelHandler, ctx *ContextHandler) {
	// Root and /panel redirect to the admin panel — the only UI entry point.
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin") })
	r.GET("/panel", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin") })

	// Panel API — used by the admin UI and the MCP server.
	panelAPI := r.Group("/panel")
	{
		panelAPI.POST("/login", panel.Login)
		panelAPI.POST("/signup", panel.SignUp)
		panelAPI.POST("/logout", panel.Logout)
		panelAPI.GET("/me", panel.Me)
		panelAPI.GET("/context-notes", panel.GetContextNotes)
		panelAPI.PUT("/context-notes", panel.SaveContextNotes)
	}

	// Standard Open MCP context convention — MCP fetches docs from these routes.
	r.GET("/mcp/context", ctx.ListContext)
	r.GET("/mcp/context/*path", ctx.ReadContext)
}
