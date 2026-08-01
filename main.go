package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wiki/internal/config"
	"wiki/internal/database"
	"wiki/internal/handlers"
	"wiki/internal/middleware"
	"wiki/internal/repository"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	// Initialize Database
	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Misslyckades att initiera databasen: %v", err)
	}

	wikiRepo := repository.NewWikiRepository(db)
	wikiHandler := handlers.NewWikiHandler(wikiRepo)

	router := gin.Default()

	// Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-Key"}
	router.Use(cors.New(corsConfig))

	// API v1 Routes
	v1 := router.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(cfg.MasterAPIKey))
	{
		v1.GET("/health", wikiHandler.GetHealth)

		// Page operations
		v1.GET("/pages", wikiHandler.ListPages)
		v1.GET("/pages/:slug", wikiHandler.GetPage)
		v1.POST("/pages", wikiHandler.CreatePage)
		v1.PUT("/pages/:slug", wikiHandler.UpdatePage)
		v1.DELETE("/pages/:slug", wikiHandler.DeletePage)

		// Revision history
		v1.GET("/pages/:slug/revisions", wikiHandler.GetRevisions)

		// Search & Tags
		v1.GET("/search", wikiHandler.SearchPages)
		v1.GET("/tags", wikiHandler.ListTags)
	}

	// Serve Static SPA Frontend (web/dist) if built
	webDist := "./web/dist"
	if _, err := os.Stat(webDist); err == nil {
		router.Use(staticFileMiddleware(webDist))
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 Wiki Server startad på http://localhost%s", addr)
	log.Printf("📡 REST API tillgängligt på http://localhost%s/api/v1", addr)

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
