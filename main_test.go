package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"wiki/internal/database"
	"wiki/internal/handlers"
	"wiki/internal/models"
	"wiki/internal/repository"

	"github.com/gin-gonic/gin"
)

func setupTestRouter(t *testing.T) (*gin.Engine, func()) {
	gin.SetMode(gin.TestMode)
	dbFile := "test_wiki.db"
	os.Remove(dbFile)

	db, err := database.InitDB(dbFile)
	if err != nil {
		t.Fatalf("Failed to init test db: %v", err)
	}

	repo := repository.NewWikiRepository(db)
	h := handlers.NewWikiHandler(repo)

	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", h.GetHealth)
		v1.GET("/pages", h.ListPages)
		v1.GET("/pages/:slug", h.GetPage)
		v1.POST("/pages", h.CreatePage)
		v1.PUT("/pages/:slug", h.UpdatePage)
		v1.DELETE("/pages/:slug", h.DeletePage)
		v1.POST("/pages/:slug/attachments", h.UploadAttachment)
		v1.GET("/pages/:slug/attachments", h.GetAttachments)
		v1.DELETE("/attachments/:id", h.DeleteAttachment)
		v1.GET("/pages/:slug/revisions", h.GetRevisions)
		v1.GET("/pages/:slug/backlinks", h.GetBacklinks)
		v1.GET("/search", h.SearchPages)
	}

	cleanup := func() {
		os.Remove(dbFile)
	}

	return r, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCreateAndFetchPage(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// 1. Create page
	createPayload := models.CreatePageRequest{
		Title:   "Test Sida",
		Content: "# Test Innehåll\nDetta är en testtext.",
		Summary: "Test sammanfattning",
		Tags:    []string{"test", "demo"},
	}
	bodyBytes, _ := json.Marshal(createPayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Fetch created page via slug
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/pages/test-sida", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK for created page, got %d", w2.Code)
	}

	// 3. Update page & check revisions
	updatePayload := models.UpdatePageRequest{
		Title:   "Test Sida Uppdaterad",
		Content: "# Uppdaterat Innehåll",
		Comment: "Ändrade titeln och innehållet",
	}
	updateBytes, _ := json.Marshal(updatePayload)

	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("PUT", "/api/v1/pages/test-sida", bytes.NewBuffer(updateBytes))
	req3.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK for update, got %d", w3.Code)
	}

	// 4. Fetch revisions
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/api/v1/pages/test-sida/revisions", nil)
	router.ServeHTTP(w4, req4)

	if w4.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK for revisions, got %d", w4.Code)
	}
}
