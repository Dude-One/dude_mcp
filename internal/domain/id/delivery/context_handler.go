package delivery

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContextHandler serves the domain context docs at GET /mcp/context and
// GET /mcp/context/{path}. It implements the standard Open MCP context convention
// so MCP can discover and fetch docs the same way it fetches Swagger.
type ContextHandler struct {
	root string // absolute or relative path to the context directory
}

// NewContextHandler creates a handler serving files from root.
// For the panel (ID domain) root should be "./context/project_id".
func NewContextHandler(root string) *ContextHandler {
	return &ContextHandler{root: root}
}

// ListContext godoc
// @Summary      List context docs
// @Description  Returns a JSON array of relative file paths available under the domain context root.
// @Tags         mcp-context
// @Produce      json
// @Success      200  {array}   string
// @Router       /mcp/context [get]
func (h *ContextHandler) ListContext(c *gin.Context) {
	var paths []string

	err := filepath.WalkDir(h.root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(h.root, path)
		if relErr != nil {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})

	if err != nil || paths == nil {
		paths = []string{}
	}

	c.Header("Content-Type", "application/json")
	enc := json.NewEncoder(c.Writer)
	enc.SetIndent("", "  ")
	_ = enc.Encode(paths)
}

// ReadContext godoc
// @Summary      Read one context doc
// @Description  Returns the raw content of a single context file. Rejects path traversal.
// @Tags         mcp-context
// @Produce      plain
// @Param        path  path  string  true  "Relative file path, e.g. README.md or guides/devices.md"
// @Success      200   {string}  string
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /mcp/context/{path} [get]
func (h *ContextHandler) ReadContext(c *gin.Context) {
	rel := c.Param("path")
	rel = strings.TrimPrefix(rel, "/")

	if strings.Contains(rel, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	full := filepath.Join(h.root, filepath.FromSlash(rel))

	// Guard: resolved path must still be inside root.
	absRoot, _ := filepath.Abs(h.root)
	absFull, _ := filepath.Abs(full)
	if !strings.HasPrefix(absFull, absRoot+string(os.PathSeparator)) && absFull != absRoot {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read error"})
		return
	}

	c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
}
