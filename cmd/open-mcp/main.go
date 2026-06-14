// Open MCP — single binary entry point.
//
// Usage:
//
//	open-mcp server   — panel + admin HTTP server (:8080)
//	open-mcp mcp      — MCP JSON-RPC server (:8081)
//	open-mcp migrate  — apply schema and seed, then exit
//
// @title           Open MCP Panel API
// @version         1.0
// @description     Mock identity provider and panel for Open MCP (login, sessions, context notes, admin).
//
// @host      localhost:8080
// @BasePath  /
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/open-mcp/open-mcp/internal/domain/id"
	"github.com/open-mcp/open-mcp/internal/domain/id/adapter"
	idmw "github.com/open-mcp/open-mcp/internal/domain/id/middleware"
	openmcp "github.com/open-mcp/open-mcp/internal/domain/open_mcp"
	mcpadapter "github.com/open-mcp/open-mcp/internal/domain/open_mcp/adapter"

	_ "github.com/open-mcp/open-mcp/docs"
	_ "github.com/open-mcp/open-mcp/internal/docs"
)

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: open-mcp <server|mcp|migrate>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "server":
		runServer()
	case "mcp":
		runMCP()
	case "migrate":
		runMigrate()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q — use server, mcp, or migrate\n", os.Args[1])
		os.Exit(1)
	}
}

func runServer() {
	db, err := adapter.OpenFromEnv()
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	log.Println("applying schema (GORM AutoMigrate)...")
	if err := adapter.Migrate(db); err != nil {
		log.Fatalf("migrate id tables: %v", err)
	}
	if err := mcpadapter.MigrateAdminTables(db); err != nil {
		log.Fatalf("migrate mcp tables: %v", err)
	}

	log.Println("seeding dev data...")
	if err := adapter.Seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	idMod := id.NewModule(db)

	sourcesPath := os.Getenv("SOURCES_PATH")
	if sourcesPath == "" {
		sourcesPath = "./config/sources.yaml"
	}
	mcpMod, err := openmcp.NewModule(sourcesPath, db, idMod.AuthService())
	if err != nil {
		log.Fatalf("init mcp module: %v", err)
	}

	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health/ready", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	idMod.RegisterRoutes(r)

	r.Static("/static/mcp", "./internal/domain/open_mcp/view")
	r.GET("/admin", func(c *gin.Context) {
		c.File("./internal/domain/open_mcp/view/index.html")
	})
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin")
	})

	mcpAdminURL := os.Getenv("MCP_ADMIN_URL")
	if mcpAdminURL == "" {
		mcpAdminURL = "http://localhost:8082"
	}

	admin := r.Group("/admin")
	idMod.RegisterAdminRoutes(admin)
	mcpMod.RegisterAdminRoutes(admin)

	adminOnly := idmw.AdminOnly(idMod.AuthService())
	admin.GET("/mcp/status", adminOnly, func(c *gin.Context) {
		resp, err := http.Get(mcpAdminURL + "/health")
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "offline", "error": err.Error()})
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			c.Data(http.StatusServiceUnavailable, "application/json", body)
			return
		}
		c.Data(resp.StatusCode, "application/json", body)
	})
	admin.POST("/mcp/reload", adminOnly, func(c *gin.Context) {
		resp, err := http.Post(mcpAdminURL+"/reload", "application/json", nil)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MCP server unreachable: " + err.Error()})
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", body)
	})


	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Open MCP panel on :%s (swagger: http://localhost:%s/swagger/index.html)", port, port)
	log.Printf("Admin panel: http://localhost:%s/admin", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runMCP() {
	db, err := adapter.OpenFromEnv()
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	log.Println("applying schema (GORM AutoMigrate)...")
	if err := adapter.Migrate(db); err != nil {
		log.Fatalf("migrate id tables: %v", err)
	}
	if err := mcpadapter.MigrateAdminTables(db); err != nil {
		log.Fatalf("migrate mcp tables: %v", err)
	}

	log.Println("seeding dev data...")
	if err := adapter.Seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	idMod := id.NewModule(db)

	sourcesPath := os.Getenv("SOURCES_PATH")
	if sourcesPath == "" {
		sourcesPath = "./config/sources.yaml"
	}

	mod, err := openmcp.NewModule(sourcesPath, db, idMod.AuthService())
	if err != nil {
		log.Fatalf("initialising MCP domain: %v", err)
	}

	adminPort := os.Getenv("MCP_ADMIN_PORT")
	if adminPort == "" {
		adminPort = "8082"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Reload: respond immediately then exit so Docker restarts the container
	// with fresh config loaded from the database.
	mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "restarting"})
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
	})

	go func() {
		log.Printf("MCP admin server on :%s  (health + reload)", adminPort)
		if err := http.ListenAndServe(":"+adminPort, mux); err != nil {
			log.Fatalf("MCP admin server error: %v", err)
		}
	}()

	port := os.Getenv("MCP_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Open MCP server on :%s  (sources: %s)", port, sourcesPath)

	httpServer := server.NewStreamableHTTPServer(mod.Server)
	if err := httpServer.Start(":" + port); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}

func runMigrate() {
	db, err := adapter.OpenFromEnv()
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	log.Println("running GORM AutoMigrate...")
	if err := adapter.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	log.Println("seeding dev data...")
	if err := adapter.Seed(db); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Println("done — open_mcp_db is ready")
}
