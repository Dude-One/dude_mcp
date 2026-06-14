package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trio/triov2-mcp/internal/domain/id/entity"
	"github.com/trio/triov2-mcp/internal/domain/id/service"
)

// AdminOnly rejects requests from non-admin users with 403.
// Requires the session to be resolved — place after session middleware or inline.
func AdminOnly(svc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("trio_session")
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
			c.Abort()
			return
		}

		claims, err := svc.ValidateSession(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			c.Abort()
			return
		}

		if claims.Role != entity.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		// Stash claims for downstream handlers.
		c.Set("claims", claims)
		c.Next()
	}
}
