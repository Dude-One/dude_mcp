package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/Dude-One/dude_mcp/internal/domain/id/repository"
)

// AdminUserHandler handles admin operations on users.
type AdminUserHandler struct {
	users repository.UserRepository
}

func NewAdminUserHandler(users repository.UserRepository) *AdminUserHandler {
	return &AdminUserHandler{users: users}
}

// ListUsers godoc
// @Summary      List all users
// @Tags         admin
// @Produce      json
// @Router       /admin/users [get]
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	users, err := h.users.FindAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type userRow struct {
		ID              string   `json:"id"`
		Username        string   `json:"username"`
		DisplayName     string   `json:"display_name"`
		Role            string   `json:"role"`
		AllowedProjects []string `json:"allowed_projects"`
		IsActive        bool     `json:"is_active"`
	}

	out := make([]userRow, 0, len(users))
	for _, u := range users {
		out = append(out, userRow{
			ID:              u.ID.String(),
			Username:        u.Username,
			DisplayName:     u.DisplayName,
			Role:            string(u.Role),
			AllowedProjects: u.AllowedProjects,
			IsActive:        u.IsActive,
		})
	}
	c.JSON(http.StatusOK, out)
}

// UpdateUserRole godoc
// @Summary      Set user role (admin/user)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Router       /admin/users/:id/role [put]
func (h *AdminUserHandler) UpdateUserRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var body struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Role != "admin" && body.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or user"})
		return
	}
	if err := h.users.UpdateRole(c.Request.Context(), id, body.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": body.Role})
}

// UpdateUserProjects godoc
// @Summary      Update user's allowed_projects
// @Tags         admin
// @Accept       json
// @Produce      json
// @Router       /admin/users/:id/projects [put]
func (h *AdminUserHandler) UpdateUserProjects(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var body struct {
		Projects []string `json:"projects" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.users.UpdateAllowedProjects(c.Request.Context(), id, body.Projects); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": body.Projects})
}
