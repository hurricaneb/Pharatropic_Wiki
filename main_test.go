package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"wiki/internal/config"
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

	cfg := &config.Config{DBDriver: "sqlite", DBPath: dbFile}
	middleware.InitJWTSecret("test-jwt-secret")
	db, err := database.InitDB(cfg)
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
		v1.PUT("/auth/password", middleware.RequireAuth(), h.ChangePassword)

		v1.GET("/pages", h.ListPages)
		v1.GET("/pages/:slug", h.GetPage)
		v1.GET("/pages/:slug/revisions", h.GetRevisions)
		v1.GET("/pages/:slug/backlinks", h.GetBacklinks)
		v1.GET("/pages/:slug/attachments", h.GetAttachments)
		v1.GET("/search", h.SearchPages)

		authed := v1.Group("")
		authed.Use(middleware.RequireAuth(), middleware.RequirePasswordChanged())
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
		admin.Use(middleware.RequireAdmin(), middleware.RequirePasswordChanged())
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

	isPublic := true
	// 1. Create page
	createPayload := models.CreatePageRequest{
		Title:    "Test Sida",
		Content:  "# Test Innehåll\nDetta är en testtext.",
		Summary:  "Test sammanfattning",
		IsPublic: &isPublic,
		Tags:     []string{"test", "demo"},
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

	isPublic := true
	// 1. Create target page
	p1 := models.CreatePageRequest{
		Title:    "Välkommen till Wikin",
		Content:  "Välkomstsida content",
		IsPublic: &isPublic,
	}
	b1, _ := json.Marshal(p1)
	w1 := httptest.NewRecorder()
	r1, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(b1))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w1, r1)

	// 2. Create linking page "Test" that contains [[Välkommen till Wikin]]
	p2 := models.CreatePageRequest{
		Title:    "Test",
		Content:  "Länkar till [[Välkommen till Wikin]] här!",
		IsPublic: &isPublic,
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

	isPublic := true
	// 1. Create page
	p1 := models.CreatePageRequest{
		Title:    "Rollback Test",
		Content:  "Original Version",
		IsPublic: &isPublic,
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
	rRev.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(wRev, rRev)

	var revsRes struct {
		Data []models.Revision `json:"data"`
	}
	json.Unmarshal(wRev.Body.Bytes(), &revsRes)
	if len(revsRes.Data) == 0 {
		t.Fatalf("Expected revisions for rollback-test, got none")
	}
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

func TestMustChangePasswordBlocksWriteActions(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	loginPayload := models.LoginRequest{Username: "admin", Password: "admin"}
	bLogin, _ := json.Marshal(loginPayload)
	wLogin := httptest.NewRecorder()
	rLogin, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(bLogin))
	rLogin.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wLogin, rLogin)

	var loginRes struct {
		Token string `json:"token"`
	}
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)
	token := loginRes.Token

	// Attempting to create a page before changing the password must be blocked
	pagePayload := models.CreatePageRequest{Title: "Should Be Blocked", Content: "x"}
	bPage, _ := json.Marshal(pagePayload)
	wPage := httptest.NewRecorder()
	rPage, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(bPage))
	rPage.Header.Set("Content-Type", "application/json")
	rPage.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(wPage, rPage)

	if wPage.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden before password change, got %d: %s", wPage.Code, wPage.Body.String())
	}

	// Wrong current password must be rejected
	wBadChange := httptest.NewRecorder()
	bBadChange, _ := json.Marshal(models.ChangePasswordRequest{CurrentPassword: "wrong", NewPassword: "a-much-stronger-password"})
	rBadChange, _ := http.NewRequest("PUT", "/api/v1/auth/password", bytes.NewBuffer(bBadChange))
	rBadChange.Header.Set("Content-Type", "application/json")
	rBadChange.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(wBadChange, rBadChange)

	if wBadChange.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for wrong current password, got %d: %s", wBadChange.Code, wBadChange.Body.String())
	}

	// Correct password change clears the flag and unblocks writes
	wChange := httptest.NewRecorder()
	bChange, _ := json.Marshal(models.ChangePasswordRequest{CurrentPassword: "admin", NewPassword: "a-much-stronger-password"})
	rChange, _ := http.NewRequest("PUT", "/api/v1/auth/password", bytes.NewBuffer(bChange))
	rChange.Header.Set("Content-Type", "application/json")
	rChange.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(wChange, rChange)

	if wChange.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for password change, got %d: %s", wChange.Code, wChange.Body.String())
	}

	wPage2 := httptest.NewRecorder()
	rPage2, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(bPage))
	rPage2.Header.Set("Content-Type", "application/json")
	rPage2.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(wPage2, rPage2)

	if wPage2.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created after password change, got %d: %s", wPage2.Code, wPage2.Body.String())
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

	if !loginRes.User.MustChangePassword {
		t.Fatalf("Expected seeded admin to have must_change_password=true")
	}

	// 1b. Seeded admin must change its password before it can act as admin
	changePayload := models.ChangePasswordRequest{
		CurrentPassword: "admin",
		NewPassword:     "a-much-stronger-password",
	}
	bChange, _ := json.Marshal(changePayload)
	wChange := httptest.NewRecorder()
	rChange, _ := http.NewRequest("PUT", "/api/v1/auth/password", bytes.NewBuffer(bChange))
	rChange.Header.Set("Content-Type", "application/json")
	rChange.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(wChange, rChange)

	if wChange.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for password change, got %d: %s", wChange.Code, wChange.Body.String())
	}

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

func TestPrivateAndPublicPageVisibility(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	isPublicFalse := false
	isPublicTrue := true

	// 1. Create a private page (default when is_public is omitted or false)
	privReq := models.CreatePageRequest{
		Title:    "Hemlig Sida",
		Content:  "Detta innehåll är privat",
		IsPublic: &isPublicFalse,
	}
	bPriv, _ := json.Marshal(privReq)
	wPriv := httptest.NewRecorder()
	rPriv, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(bPriv))
	rPriv.Header.Set("Content-Type", "application/json")
	rPriv.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(wPriv, rPriv)

	if wPriv.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for private page, got %d: %s", wPriv.Code, wPriv.Body.String())
	}

	// 2. Create a public page (is_public = true)
	pubReq := models.CreatePageRequest{
		Title:    "Offentlig Guide",
		Content:  "Detta innehåll är publikt för alla",
		IsPublic: &isPublicTrue,
	}
	bPub, _ := json.Marshal(pubReq)
	wPub := httptest.NewRecorder()
	rPub, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(bPub))
	rPub.Header.Set("Content-Type", "application/json")
	rPub.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(wPub, rPub)

	if wPub.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for public page, got %d: %s", wPub.Code, wPub.Body.String())
	}

	// 3. Unauthenticated GET /api/v1/pages -> Should only return public pages!
	wUnauthList := httptest.NewRecorder()
	rUnauthList, _ := http.NewRequest("GET", "/api/v1/pages", nil)
	router.ServeHTTP(wUnauthList, rUnauthList)

	var unauthListRes struct {
		Data []models.Page `json:"data"`
	}
	json.Unmarshal(wUnauthList.Body.Bytes(), &unauthListRes)

	// Default welcome page (is_public: true) + Offentlig Guide (is_public: true) = 2 pages
	for _, p := range unauthListRes.Data {
		if !p.IsPublic {
			t.Fatalf("Unauthenticated list returned private page '%s'", p.Title)
		}
	}

	// 4. Unauthenticated GET /api/v1/pages/hemlig-sida -> Should return 401 Unauthorized
	wUnauthGet := httptest.NewRecorder()
	rUnauthGet, _ := http.NewRequest("GET", "/api/v1/pages/hemlig-sida", nil)
	router.ServeHTTP(wUnauthGet, rUnauthGet)

	if wUnauthGet.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for private page access without auth, got %d: %s", wUnauthGet.Code, wUnauthGet.Body.String())
	}

	// 5. Authenticated GET /api/v1/pages/hemlig-sida -> Should return 200 OK
	wAuthGet := httptest.NewRecorder()
	rAuthGet, _ := http.NewRequest("GET", "/api/v1/pages/hemlig-sida", nil)
	rAuthGet.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(wAuthGet, rAuthGet)

	if wAuthGet.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for authenticated private page access, got %d: %s", wAuthGet.Code, wAuthGet.Body.String())
	}

	// 6. Update public page without specifying is_public (omitted) -> Must maintain is_public: true
	upReq := models.UpdatePageRequest{
		Content: "Uppdaterad guide text utan att ändra synlighet",
	}
	bUp, _ := json.Marshal(upReq)
	wUp := httptest.NewRecorder()
	rUp, _ := http.NewRequest("PUT", "/api/v1/pages/offentlig-guide", bytes.NewBuffer(bUp))
	rUp.Header.Set("Content-Type", "application/json")
	rUp.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(wUp, rUp)

	if wUp.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for update, got %d", wUp.Code)
	}

	var upRes struct {
		Data models.Page `json:"data"`
	}
	json.Unmarshal(wUp.Body.Bytes(), &upRes)
	if !upRes.Data.IsPublic {
		t.Fatalf("Expected page to remain public after update with omitted is_public")
	}
}

// --- Subpages (one level of hierarchy) ---

func createPageViaAPI(t *testing.T, router *gin.Engine, req models.CreatePageRequest) (models.Page, int) {
	t.Helper()
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/api/v1/pages", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w, r)

	var res struct {
		Data  models.Page `json:"data"`
		Error string      `json:"error"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res.Data, w.Code
}

func updatePageViaAPI(t *testing.T, router *gin.Engine, slug string, req models.UpdatePageRequest) (models.Page, int, string) {
	t.Helper()
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "/api/v1/pages/"+slug, bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w, r)

	var res struct {
		Data  models.Page `json:"data"`
		Error string      `json:"error"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res.Data, w.Code, res.Error
}

func getPageViaAPI(t *testing.T, router *gin.Engine, slug string) (models.Page, int) {
	t.Helper()
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/api/v1/pages/"+slug, nil)
	r.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w, r)

	var res struct {
		Data models.Page `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	return res.Data, w.Code
}

func TestSubpages_CreateWithParentAndFetchHierarchy(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	parent, code := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Föräldrasida", Content: "root"})
	if code != http.StatusCreated {
		t.Fatalf("Expected 201 for parent page, got %d", code)
	}

	child, code := createPageViaAPI(t, router, models.CreatePageRequest{
		Title: "Barnsida", Content: "child", ParentSlug: parent.Slug,
	})
	if code != http.StatusCreated {
		t.Fatalf("Expected 201 for child page, got %d", code)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("Expected child.ParentID to be %d, got %v", parent.ID, child.ParentID)
	}

	fetchedChild, _ := getPageViaAPI(t, router, child.Slug)
	if fetchedChild.Parent == nil || fetchedChild.Parent.Slug != parent.Slug {
		t.Fatalf("Expected fetched child to have Parent populated with slug %q, got %+v", parent.Slug, fetchedChild.Parent)
	}

	fetchedParent, _ := getPageViaAPI(t, router, parent.Slug)
	if len(fetchedParent.Children) != 1 || fetchedParent.Children[0].Slug != child.Slug {
		t.Fatalf("Expected fetched parent to have 1 child with slug %q, got %+v", child.Slug, fetchedParent.Children)
	}
}

func TestSubpages_RejectMoreThanOneLevelDeep(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	grandparent, _ := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Farfar", Content: "x"})
	parent, code := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Förälder Två", Content: "x", ParentSlug: grandparent.Slug})
	if code != http.StatusCreated {
		t.Fatalf("Expected 201 for parent page, got %d", code)
	}

	_, code = createPageViaAPI(t, router, models.CreatePageRequest{Title: "Barnbarn", Content: "x", ParentSlug: parent.Slug})
	if code != http.StatusBadRequest {
		t.Fatalf("Expected 400 when nesting a second level deep, got %d", code)
	}
}

func TestSubpages_PageWithChildrenCannotBecomeChild(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	pageA, _ := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Sida A", Content: "x"})
	_, code := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Sida B", Content: "x", ParentSlug: pageA.Slug})
	if code != http.StatusCreated {
		t.Fatalf("Expected 201 creating child of A, got %d", code)
	}

	otherParent, _ := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Annan Förälder", Content: "x"})
	newParentSlug := otherParent.Slug
	_, code, errMsg := updatePageViaAPI(t, router, pageA.Slug, models.UpdatePageRequest{Content: "x", ParentSlug: &newParentSlug})
	if code != http.StatusBadRequest {
		t.Fatalf("Expected 400 when giving a parent-with-children a parent of its own, got %d (%s)", code, errMsg)
	}
}

func TestSubpages_CannotBeOwnParent(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	page, _ := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Självrefererande Sida", Content: "x"})
	selfSlug := page.Slug
	_, code, _ := updatePageViaAPI(t, router, page.Slug, models.UpdatePageRequest{Content: "x", ParentSlug: &selfSlug})
	if code != http.StatusBadRequest {
		t.Fatalf("Expected 400 when a page is set as its own parent, got %d", code)
	}
}

func TestSubpages_DeletingParentPromotesChildren(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	parent, _ := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Tillfällig Förälder", Content: "x"})
	child, code := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Kvarlevande Barn", Content: "x", ParentSlug: parent.Slug})
	if code != http.StatusCreated {
		t.Fatalf("Expected 201 for child page, got %d", code)
	}

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("DELETE", "/api/v1/pages/"+parent.Slug, nil)
	r.Header.Set("X-API-Key", "master-secret")
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 deleting parent page, got %d", w.Code)
	}

	fetchedChild, code := getPageViaAPI(t, router, child.Slug)
	if code != http.StatusOK {
		t.Fatalf("Expected child page to survive parent deletion, got %d", code)
	}
	if fetchedChild.ParentID != nil {
		t.Fatalf("Expected orphaned child to have nil ParentID, got %v", fetchedChild.ParentID)
	}
}

func TestSubpages_DetachFromParent(t *testing.T) {
	router, cleanup := setupTestRouter(t)
	defer cleanup()

	parent, _ := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Bas Förälder", Content: "x"})
	child, _ := createPageViaAPI(t, router, models.CreatePageRequest{Title: "Löst Barn", Content: "x", ParentSlug: parent.Slug})

	emptySlug := ""
	updated, code, _ := updatePageViaAPI(t, router, child.Slug, models.UpdatePageRequest{Content: "x", ParentSlug: &emptySlug})
	if code != http.StatusOK {
		t.Fatalf("Expected 200 detaching child from parent, got %d", code)
	}
	if updated.ParentID != nil {
		t.Fatalf("Expected ParentID to be nil after detaching, got %v", updated.ParentID)
	}
}
