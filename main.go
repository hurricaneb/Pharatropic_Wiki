package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	middleware.InitJWTSecret(cfg.JWTSecret)

	// Initialize Database
	db, err := database.InitDB(cfg)
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

	// Serve uploaded files as forced downloads (never inline), since
	// uploads may be any file type and must not execute in the app's origin
	os.MkdirAll("./uploads", 0755)
	router.GET("/uploads/:filename", serveUploadFile)

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
		v1.PUT("/auth/password", middleware.RequireAuth(), wikiHandler.ChangePassword)

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
		authed.Use(middleware.RequireAuth(), middleware.RequirePasswordChanged())
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
		admin.Use(middleware.RequireAdmin(), middleware.RequirePasswordChanged())
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

// inlineSafeMimeTypes are content types that browsers cannot turn into
// script execution, so they may be displayed inline for a good preview
// experience (images, PDFs). Everything else is forced to download instead,
// since uploads accept arbitrary file types and something like an uploaded
// HTML or SVG file rendered inline could run script in the app's own origin
// and steal the auth token from localStorage.
var inlineSafeMimeTypes = []string{
	"image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp", "image/x-icon",
	"application/pdf",
}

func serveUploadFile(c *gin.Context) {
	filename := filepath.Base(c.Param("filename"))
	filePath := filepath.Join("./uploads", filename)

	f, err := os.Open(filePath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer f.Close()

	if info, err := f.Stat(); err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	// Sniff the actual file content rather than trusting the extension or
	// the Content-Type recorded at upload time, both of which can be spoofed.
	head := make([]byte, 512)
	n, _ := f.Read(head)
	sniffed := http.DetectContentType(head[:n])

	disposition := "attachment"
	for _, safe := range inlineSafeMimeTypes {
		if strings.HasPrefix(sniffed, safe) {
			disposition = "inline"
			break
		}
	}

	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filename))
	c.File(filePath)
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
