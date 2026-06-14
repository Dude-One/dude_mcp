package delivery

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trio/triov2-mcp/internal/domain/id/service"
)

// PanelHandler manages panel browser sessions.
type PanelHandler struct {
	svc service.AuthService
}

func NewPanelHandler(svc service.AuthService) *PanelHandler {
	return &PanelHandler{svc: svc}
}

// Login godoc
// @Summary      Panel login
// @Description  Authenticates against the mock IdP. Sets trio_session cookie.
// @Tags         panel
// @Accept       json
// @Produce      json
// @Param        body  body      docs.PanelLoginRequest  true  "Login credentials"
// @Success      200   {object}  docs.StatusResponse
// @Failure      400   {object}  docs.ErrorResponse
// @Failure      401   {object}  docs.ErrorResponse
// @Router       /panel/login [post]
func (h *PanelHandler) Login(c *gin.Context) {
	var body struct {
		Username   string `json:"username" form:"username" binding:"required"`
		Credential string `json:"credential" form:"credential" binding:"required"`
	}
	if err := c.ShouldBind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and credential are required"})
		return
	}

	sessionToken, err := h.svc.Login(c.Request.Context(), body.Username, body.Credential)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.SetCookie("trio_session", sessionToken, int(24*time.Hour/time.Second), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// SignUp godoc
// @Summary      Panel sign up
// @Description  Creates a new password user on the default tenant and signs them in.
// @Tags         panel
// @Accept       json
// @Produce      json
// @Param        body  body      docs.PanelSignUpRequest  true  "Sign up details"
// @Success      201   {object}  docs.StatusResponse
// @Failure      400   {object}  docs.ErrorResponse
// @Failure      409   {object}  docs.ErrorResponse
// @Router       /panel/signup [post]
func (h *PanelHandler) SignUp(c *gin.Context) {
	var body struct {
		Username    string `json:"username" binding:"required"`
		Credential  string `json:"credential" binding:"required"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and credential are required"})
		return
	}

	sessionToken, err := h.svc.SignUp(c.Request.Context(), body.Username, body.Credential, body.DisplayName)
	if err != nil {
		if err.Error() == "username already taken" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("trio_session", sessionToken, int(24*time.Hour/time.Second), "/", "", false, true)
	c.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

// Logout godoc
// @Summary      Panel logout
// @Description  Revokes the trio_session cookie.
// @Tags         panel
// @Produce      json
// @Success      200  {object}  docs.StatusResponse
// @Router       /panel/logout [post]
func (h *PanelHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie("trio_session"); err == nil && token != "" {
		_ = h.svc.Logout(c.Request.Context(), token)
	}
	c.SetCookie("trio_session", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

// Me godoc
// @Summary      Current panel user
// @Description  Returns identity from the active trio_session cookie.
// @Tags         panel
// @Produce      json
// @Success      200  {object}  docs.PanelMeResponse
// @Failure      401  {object}  docs.ErrorResponse
// @Router       /panel/me [get]
func (h *PanelHandler) Me(c *gin.Context) {
	token, err := c.Cookie("trio_session")
	if err != nil || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}
	claims, err := h.svc.ValidateSession(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":          claims.UserID,
		"tenant_id":        claims.TenantID,
		"username":         claims.Username,
		"display_name":     claims.DisplayName,
		"allowed_projects": claims.AllowedProjects,
		"role":             claims.Role,
	})
}

// GetContextNotes godoc
// @Summary      Get LLM context notes
// @Description  Returns the user's saved context notes.
// @Tags         panel
// @Produce      json
// @Success      200  {object}  docs.ContextNotesResponse
// @Failure      401  {object}  docs.ErrorResponse
// @Router       /panel/context-notes [get]
func (h *PanelHandler) GetContextNotes(c *gin.Context) {
	token, err := c.Cookie("trio_session")
	if err != nil || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}
	notes, err := h.svc.GetContextNotes(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

// SaveContextNotes godoc
// @Summary      Save LLM context notes
// @Description  Persists the user's context notes.
// @Tags         panel
// @Accept       json
// @Produce      json
// @Param        body  body      docs.ContextNotesSaveRequest  true  "Notes body"
// @Success      200   {object}  docs.StatusResponse
// @Failure      400   {object}  docs.ErrorResponse
// @Failure      401   {object}  docs.ErrorResponse
// @Router       /panel/context-notes [put]
func (h *PanelHandler) SaveContextNotes(c *gin.Context) {
	token, err := c.Cookie("trio_session")
	if err != nil || token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}
	var body struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.svc.SaveContextNotes(c.Request.Context(), token, body.Notes); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
