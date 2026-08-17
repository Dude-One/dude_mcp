package docs

type ErrorResponse struct {
	Error string `json:"error" example:"invalid credentials"`
}

type StatusResponse struct {
	Status string `json:"status" example:"ok"`
}

type PanelLoginRequest struct {
	Username   string `json:"username" example:"admin"`
	Credential string `json:"credential" example:"admin"`
}

type PanelSignUpRequest struct {
	Username    string `json:"username" example:"jane"`
	Credential  string `json:"credential" example:"secret123"`
	DisplayName string `json:"display_name" example:"Jane Doe"`
}

type PanelMeResponse struct {
	UserID          string   `json:"user_id" example:"00000000-0000-4000-8000-000000000010"`
	TenantID        string   `json:"tenant_id" example:"00000000-0000-4000-8000-000000000001"`
	Username        string   `json:"username" example:"admin"`
	DisplayName     string   `json:"display_name" example:"Admin User"`
	AllowedProjects []string `json:"allowed_projects" example:"dude-mcp,dude-mcp-servermanagement"`
}

type ContextNotesResponse struct {
	Notes string `json:"notes" example:"# My context"`
}

type ContextNotesSaveRequest struct {
	Notes string `json:"notes" example:"# My context"`
}
