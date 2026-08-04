package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wiki/internal/config"
	"wiki/internal/database"
	"wiki/internal/handlers"
	"wiki/internal/mcp"
	"wiki/internal/middleware"
	"wiki/internal/repository"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	mcpFlag := flag.Bool("mcp", false, "Start in MCP (Model Context Protocol) Stdio mode")
	flag.Parse()

	cfg := config.LoadConfig()

	// Initialize Database
	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Misslyckades att initiera databasen: %v", err)
	}

	wikiRepo := repository.NewWikiRepository(db)
	wikiHandler := handlers.NewWikiHandler(wikiRepo)
	mcpServer := mcp.NewServer(wikiRepo)

	// If launched in Stdio MCP mode
	if *mcpFlag {
		if err := mcpServer.ServeStdio(); err != nil {
			log.Fatalf("MCP Server Error: %v", err)
		}
		return
	}

	router := gin.Default()

	// Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-Key"}
	router.Use(cors.New(corsConfig))

	// Serve uploaded files statically
	os.MkdirAll("./uploads", 0755)
	router.Static("/uploads", "./uploads")

	// API v1 Routes
	v1 := router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(wikiRepo, cfg.MasterAPIKey))
	{
		v1.GET("/health", wikiHandler.GetHealth)

		// MCP Server Endpoint (Active by default over SSE / HTTP)
		v1.Any("/mcp", mcpServer.GinHandler())

		// Authentication
		v1.POST("/auth/login", wikiHandler.AuthLogin)
		v1.GET("/auth/me", wikiHandler.AuthMe)

		// Page read operations (Public)
		v1.GET("/pages", wikiHandler.ListPages)
		v1.GET("/pages/:slug", wikiHandler.GetPage)
		v1.GET("/pages/:slug/revisions", wikiHandler.GetRevisions)
		v1.GET("/pages/:slug/backlinks", wikiHandler.GetBacklinks)
		v1.GET("/pages/:slug/attachments", wikiHandler.GetAttachments)
		v1.GET("/search", wikiHandler.SearchPages)
		v1.GET("/tags", wikiHandler.ListTags)

		// Write operations (Require Auth)
		authed := v1.Group("")
		authed.Use(middleware.RequireAuth())
		{
			authed.POST("/pages", wikiHandler.CreatePage)
			authed.PUT("/pages/:slug", wikiHandler.UpdatePage)
			authed.DELETE("/pages/:slug", wikiHandler.DeletePage)
			authed.POST("/pages/:slug/attachments", wikiHandler.UploadAttachment)
			authed.DELETE("/attachments/:id", wikiHandler.DeleteAttachment)
			authed.POST("/pages/:slug/revert/:revision_id", wikiHandler.RevertRevision)

			// User API Keys
			authed.GET("/user/keys", wikiHandler.UserListApiKeys)
			authed.POST("/user/keys", wikiHandler.UserCreateApiKey)
			authed.DELETE("/user/keys/:id", wikiHandler.UserRevokeApiKey)
		}

		// Admin operations (Require Admin Role)
		admin := v1.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.POST("/users", wikiHandler.AdminCreateUser)
			admin.GET("/users", wikiHandler.AdminListUsers)
		}
	}

	// Serve Static SPA Frontend (web/dist) if built
	webDist := "./web/dist"
	if _, err := os.Stat(webDist); err == nil {
		router.Use(staticFileMiddleware(webDist))
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 Wiki Server startad på http://localhost%s", addr)
	log.Printf("📡 REST API tillgängligt på http://localhost%s/api/v1", addr)
	log.Printf("🤖 MCP Server tillgänglig på http://localhost%s/api/v1/mcp", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Fel vid start av server: %v", err)
	}
}

func staticFileMiddleware(distPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		// Don't intercept API calls
		if len(reqPath) >= 4 && reqPath[:4] == "/api" {
			c.Next()
			return
		}

		filePath := filepath.Join(distPath, filepath.Clean(reqPath))
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			c.File(filePath)
			c.Abort()
			return
		}

		// Single Page Application fallback to index.html
		indexPath := filepath.Join(distPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
			c.Abort()
			return
		}

		c.Next()
	}
}
