package delivery

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/open-mcp/open-mcp/internal/domain/open_mcp/entity"
	"github.com/open-mcp/open-mcp/internal/domain/open_mcp/repository"
)

// AdminSourceHandler handles admin CRUD for sources.
type AdminSourceHandler struct {
	repo repository.AdminSourceRepository
}

func NewAdminSourceHandler(repo repository.AdminSourceRepository) *AdminSourceHandler {
	return &AdminSourceHandler{repo: repo}
}

// ListSources godoc
// @Summary      List all sources
// @Tags         admin
// @Produce      json
// @Success      200  {array}   entity.Source
// @Router       /admin/sources [get]
func (h *AdminSourceHandler) ListSources(c *gin.Context) {
	sources, err := h.repo.ListSources(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sources)
}

// GetSource godoc
// @Summary      Get one source
// @Tags         admin
// @Produce      json
// @Param        id  path  string  true  "Source UUID"
// @Router       /admin/sources/:id [get]
func (h *AdminSourceHandler) GetSource(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	src, err := h.repo.GetSourceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, src)
}

// sourceRequest is the create/update request body — URL config only.
type sourceRequest struct {
	Name           string `json:"name"            binding:"required"`
	PlatformDomain string `json:"platform_domain" binding:"required"`
	Description    string `json:"description"`
	Type           string `json:"type"`
	BaseURL        string `json:"base_url"        binding:"required"`
	SwaggerPath    string `json:"swagger_path"`
	ContextPath    string `json:"context_path"`
}

func (r *sourceRequest) toEntity() entity.Source {
	swaggerPath := r.SwaggerPath
	if swaggerPath == "" {
		swaggerPath = "/swagger/doc.json"
	}
	contextPath := r.ContextPath
	if contextPath == "" {
		contextPath = "/mcp/context"
	}
	srcType := entity.SourceType(r.Type)
	if srcType == "" {
		srcType = entity.SourceTypeAPI
	}
	return entity.Source{
		Name:           r.Name,
		PlatformDomain: r.PlatformDomain,
		Description:    r.Description,
		Type:           srcType,
		BaseURL:        r.BaseURL,
		SwaggerPath:    swaggerPath,
		ContextPath:    contextPath,
	}
}

// CreateSource godoc
// @Summary      Create a source
// @Tags         admin
// @Accept       json
// @Produce      json
// @Router       /admin/sources [post]
func (h *AdminSourceHandler) CreateSource(c *gin.Context) {
	var req sourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.CreateSource(c.Request.Context(), req.toEntity()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

// UpdateSource godoc
// @Summary      Update a source
// @Tags         admin
// @Accept       json
// @Produce      json
// @Router       /admin/sources/:id [put]
func (h *AdminSourceHandler) UpdateSource(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	var req sourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.UpdateSource(c.Request.Context(), id, req.toEntity()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// DeleteSource godoc
// @Summary      Delete a source
// @Tags         admin
// @Router       /admin/sources/:id [delete]
func (h *AdminSourceHandler) DeleteSource(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	if err := h.repo.DeleteSource(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ToggleSource godoc
// @Summary      Enable or disable a source
// @Tags         admin
// @Router       /admin/sources/:id/toggle [put]
func (h *AdminSourceHandler) ToggleSource(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.ToggleSource(c.Request.Context(), id, body.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": body.Active})
}
