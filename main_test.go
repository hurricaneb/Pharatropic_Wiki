package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"wiki/internal/database"
	"wiki/internal/handlers"
	"wiki/internal/middleware"
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
	v1.Use(middleware.AuthMiddleware(repo, "master-secret"))
	{
		v1.GET("/health", h.GetHealth)
		v1.POST("/auth/login", h.AuthLogin)
		v1.GET("/auth/me", h.AuthMe)

		v1.GET("/pages", h.ListPages)
		v1.GET("/pages/:slug", h.GetPage)
		v1.GET("/pages/:slug/revisions", h.GetRevisions)
		v1.GET("/pages/:slug/backlinks", h.GetBacklinks)
		v1.GET("/pages/:slug/attachments", h.GetAttachments)
		v1.GET("/search", h.SearchPages)

		authed := v1.Group("")
		authed.Use(middleware.RequireAuth())
		{
			authed.POST("/pages", h.CreatePage)
			authed.PUT("/pages/:slug", h.UpdatePage)
			authed.DELETE("/pages/:slug", h.DeletePage)
			authed.POST("/pages/:slug/attachments", h.UploadAttachment)
			authed.DELETE("/attachments/:id", h.DeleteAttachment)
			authed.POST("/pages/:slug/revert/:revision_id", h.RevertRevision)

			authed.GET("/user/keys", h.UserListApiKeys)
			authed.POST("/user/keys", h.UserCreateApiKey)
			authed.DELETE("/user/keys/:id", h.UserRevokeApiKey)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.POST("/users", h.AdminCreateUser)
			admin.GET("/users", h.AdminListUsers)
		}
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
	req.Header.Set("X-API-Key", "master-secret")
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
	req3.Header.Set("X-API-Key", "master-secret")
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

func TestBacklinksEndpoint(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// 1. Create target page
	p1 := models.CreatePageRequest{
		Title:   "Välkommen till Wikin",
		Content: "Välkomstsida content",
	}
	b1, _ := json.Marshal(p1)
	w1 := httptest.NewRecorder()
	r1, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(b1))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w1, r1)

	// 2. Create linking page "Test" that contains [[Välkommen till Wikin]]
	p2 := models.CreatePageRequest{
		Title:   "Test",
		Content: "Länkar till [[Välkommen till Wikin]] här!",
	}
	b2, _ := json.Marshal(p2)
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(b2))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w2, r2)

	// 3. Fetch backlinks for "valkommen-till-wikin"
	w3 := httptest.NewRecorder()
	r3, _ := http.NewRequest("GET", "/api/v1/pages/valkommen-till-wikin/backlinks", nil)
	router.ServeHTTP(w3, r3)

	if w3.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for backlinks, got %d", w3.Code)
	}

	var res struct {
		Data []models.Page `json:"data"`
	}
	json.Unmarshal(w3.Body.Bytes(), &res)

	if len(res.Data) != 1 {
		t.Fatalf("Expected 1 backlink, got %d: %s", len(res.Data), w3.Body.String())
	}
	if res.Data[0].Slug != "test" {
		t.Fatalf("Expected backlink from 'test', got %s", res.Data[0].Slug)
	}
}

func TestRevertRevisionEndpoint(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// 1. Create page
	p1 := models.CreatePageRequest{
		Title:   "Rollback Test",
		Content: "Original Version",
	}
	b1, _ := json.Marshal(p1)
	w1 := httptest.NewRecorder()
	r1, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(b1))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w1, r1)

	// 2. Update page to new version
	p2 := models.UpdatePageRequest{
		Content: "Updated Version",
		Comment: "Second Edit",
	}
	b2, _ := json.Marshal(p2)
	w2 := httptest.NewRecorder()
	r2, _ := http.NewRequest("PUT", "/api/v1/pages/rollback-test", bytes.NewBuffer(b2))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w2, r2)

	// 3. Fetch revisions to get initial revision ID
	wRev := httptest.NewRecorder()
	rRev, _ := http.NewRequest("GET", "/api/v1/pages/rollback-test/revisions", nil)
	router.ServeHTTP(wRev, rRev)

	var revsRes struct {
		Data []models.Revision `json:"data"`
	}
	json.Unmarshal(wRev.Body.Bytes(), &revsRes)
	initialRevID := revsRes.Data[len(revsRes.Data)-1].ID

	// 4. Revert back to initial revision
	w3 := httptest.NewRecorder()
	r3, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/pages/rollback-test/revert/%d", initialRevID), nil)
	r3.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w3, r3)

	if w3.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for revert, got %d: %s", w3.Code, w3.Body.String())
	}

	// 4. Verify page content is now "Original Version"
	w4 := httptest.NewRecorder()
	r4, _ := http.NewRequest("GET", "/api/v1/pages/rollback-test", nil)
	router.ServeHTTP(w4, r4)

	var res struct {
		Data models.Page `json:"data"`
	}
	json.Unmarshal(w4.Body.Bytes(), &res)

	if res.Data.Content != "Original Version" {
		t.Fatalf("Expected content 'Original Version' after revert, got '%s'", res.Data.Content)
	}
}

func TestUserAuthAndApiKeys(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	// 1. Login as default seeded admin
	loginPayload := models.LoginRequest{
		Username: "admin",
		Password: "admin",
	}
	bLogin, _ := json.Marshal(loginPayload)
	wLogin := httptest.NewRecorder()
	rLogin, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(bLogin))
	rLogin.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wLogin, rLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for admin login, got %d: %s", wLogin.Code, wLogin.Body.String())
	}

	var loginRes struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)
	token := loginRes.Token

	// 2. Admin creates a new user "henrik"
	newUserPayload := models.CreateUserRequest{
		Username: "henrik",
		Email:    "henrik@pharatropic.local",
		Password: "secretpassword",
		Role:     "user",
	}
	bUser, _ := json.Marshal(newUserPayload)
	wUser := httptest.NewRecorder()
	rUser, _ := http.NewRequest("POST", "/api/v1/admin/users", bytes.NewBuffer(bUser))
	rUser.Header.Set("Content-Type", "application/json")
	rUser.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(wUser, rUser)

	if wUser.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for user creation, got %d: %s", wUser.Code, wUser.Body.String())
	}

	// 3. Login as new user "henrik"
	henrikLogin := models.LoginRequest{
		Username: "henrik",
		Password: "secretpassword",
	}
	bHenrik, _ := json.Marshal(henrikLogin)
	wHenrik := httptest.NewRecorder()
	rHenrik, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(bHenrik))
	rHenrik.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wHenrik, rHenrik)

	if wHenrik.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for henrik login, got %d", wHenrik.Code)
	}

	var henrikRes struct {
		Token string `json:"token"`
	}
	json.Unmarshal(wHenrik.Body.Bytes(), &henrikRes)
	henrikToken := henrikRes.Token

	// 4. Henrik creates an API key with 30d expiry
	keyPayload := models.CreateApiKeyRequest{
		Name:    "CLI Deployment Key",
		Expires: "30d",
	}
	bKey, _ := json.Marshal(keyPayload)
	wKey := httptest.NewRecorder()
	rKey, _ := http.NewRequest("POST", "/api/v1/user/keys", bytes.NewBuffer(bKey))
	rKey.Header.Set("Content-Type", "application/json")
	rKey.Header.Set("Authorization", "Bearer "+henrikToken)
	router.ServeHTTP(wKey, rKey)

	if wKey.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for API key, got %d: %s", wKey.Code, wKey.Body.String())
	}

	var keyRes struct {
		Key  string        `json:"key"`
		Data models.ApiKey `json:"data"`
	}
	json.Unmarshal(wKey.Body.Bytes(), &keyRes)
	apiKeySecret := keyRes.Key

	// 5. Use API Key to create a new wiki page as author "henrik"
	pagePayload := models.CreatePageRequest{
		Title:   "Henrik API Page",
		Content: "# Created via User API Key",
	}
	bPage, _ := json.Marshal(pagePayload)
	wPage := httptest.NewRecorder()
	rPage, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(bPage))
	rPage.Header.Set("Content-Type", "application/json")
	rPage.Header.Set("X-API-Key", apiKeySecret)
	router.ServeHTTP(wPage, rPage)

	if wPage.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for page creation via API Key, got %d: %s", wPage.Code, wPage.Body.String())
	}

	// 6. Verify page revision author is "henrik"
	wGet := httptest.NewRecorder()
	rGet, _ := http.NewRequest("GET", "/api/v1/pages/henrik-api-page", nil)
	router.ServeHTTP(wGet, rGet)

	var getRes struct {
		Data models.Page `json:"data"`
	}
	json.Unmarshal(wGet.Body.Bytes(), &getRes)
	if len(getRes.Data.Revisions) > 0 && getRes.Data.Revisions[0].Author != "henrik" {
		t.Fatalf("Expected revision author to be 'henrik', got '%s'", getRes.Data.Revisions[0].Author)
	}

	// 7. Revoke API key
	wRevoke := httptest.NewRecorder()
	rRevoke, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/user/keys/%d", keyRes.Data.ID), nil)
	rRevoke.Header.Set("Authorization", "Bearer "+henrikToken)
	router.ServeHTTP(wRevoke, rRevoke)

	if wRevoke.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for key revocation, got %d", wRevoke.Code)
	}

	// 8. Subsequent call with revoked API key must fail
	wFail := httptest.NewRecorder()
	rFail, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(bPage))
	rFail.Header.Set("Content-Type", "application/json")
	rFail.Header.Set("X-API-Key", apiKeySecret)
	router.ServeHTTP(wFail, rFail)

	if wFail.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for revoked API key, got %d", wFail.Code)
	}
}
