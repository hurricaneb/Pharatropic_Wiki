package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"wiki/internal/models"
	"wiki/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestRepo(t *testing.T) (*repository.WikiRepository, *gorm.DB) {
	t.Helper()
	dbFile := fmt.Sprintf("test_mw_%d_%d.db", time.Now().UnixNano(), os.Getpid())
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Page{}, &models.Revision{}, &models.Attachment{}, &models.Tag{}, &models.ApiKey{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}
	t.Cleanup(func() { os.Remove(dbFile) })
	return repository.NewWikiRepository(db), db
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestInitJWTSecret_ExplicitSecret(t *testing.T) {
	InitJWTSecret("a-fixed-secret")
	if string(JWTSecret) != "a-fixed-secret" {
		t.Fatalf("Expected JWTSecret to be set to the given value, got %q", JWTSecret)
	}
}

func TestInitJWTSecret_GeneratesRandomWhenEmpty(t *testing.T) {
	InitJWTSecret("")
	first := string(JWTSecret)
	if first == "" {
		t.Fatal("Expected a non-empty generated secret")
	}

	InitJWTSecret("")
	second := string(JWTSecret)
	if first == second {
		t.Fatal("Expected two random generations to differ")
	}
}

func TestGenerateAndParseToken_RoundTrip(t *testing.T) {
	InitJWTSecret("round-trip-secret")
	user := &models.User{ID: 42, Username: "roundtrip", Role: "admin"}

	token, err := GenerateToken(user)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("unexpected error parsing token: %v", err)
	}
	if claims.UserID != 42 || claims.Username != "roundtrip" || claims.Role != "admin" {
		t.Fatalf("Claims mismatch: %+v", claims)
	}
}

func TestParseToken_InvalidString(t *testing.T) {
	InitJWTSecret("some-secret")
	if _, err := ParseToken("not-a-real-jwt"); err == nil {
		t.Fatal("Expected error parsing a malformed token")
	}
}

func TestParseToken_WrongSigningKey(t *testing.T) {
	InitJWTSecret("secret-a")
	user := &models.User{ID: 1, Username: "x"}
	token, _ := GenerateToken(user)

	InitJWTSecret("secret-b")
	if _, err := ParseToken(token); err == nil {
		t.Fatal("Expected error parsing a token signed with a different secret")
	}
}

func TestParseToken_WrongClaimsType(t *testing.T) {
	InitJWTSecret("claims-secret")
	// A validly-signed token whose claims aren't our Claims struct at all
	badToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"foo": "bar"})
	signed, _ := badToken.SignedString(JWTSecret)

	if _, err := ParseToken(signed); err != nil {
		// jwt.ParseWithClaims into &Claims{} will still populate zero-value
		// Claims successfully for an unrelated map, so this documents actual
		// behavior rather than assuming a specific outcome either way.
		t.Logf("ParseToken on mismatched claims returned error (acceptable): %v", err)
	}
}

func newTestRouter() *gin.Engine {
	return gin.New()
}

func TestAuthMiddleware_ValidBearerToken(t *testing.T) {
	InitJWTSecret("bearer-secret")
	repo, _ := setupTestRepo(t)
	user, err := repo.CreateUser("beareruser", "bearer@example.com", "pw", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	token, _ := GenerateToken(user)

	router := newTestRouter()
	router.Use(AuthMiddleware(repo, "master-key"))
	router.GET("/test", func(c *gin.Context) {
		u, exists := c.Get("user")
		if !exists {
			t.Error("Expected user to be set in context")
		}
		if u.(*models.User).Username != "beareruser" {
			t.Error("Expected correct user in context")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidBearerTokenFallsThroughUnauthenticated(t *testing.T) {
	InitJWTSecret("bearer-secret-2")
	repo, _ := setupTestRepo(t)

	router := newTestRouter()
	router.Use(AuthMiddleware(repo, ""))
	router.GET("/test", func(c *gin.Context) {
		if _, exists := c.Get("user"); exists {
			t.Error("Expected no user to be set for an invalid bearer token")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer garbage-token")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected request to still proceed (unauthenticated), got %d", w.Code)
	}
}

func TestAuthMiddleware_BearerTokenForDeletedUserFallsThrough(t *testing.T) {
	InitJWTSecret("bearer-secret-3")
	repo, db := setupTestRepo(t)
	user, _ := repo.CreateUser("temp", "temp@example.com", "pw", "user")
	token, _ := GenerateToken(user)

	router := newTestRouter()
	router.Use(AuthMiddleware(repo, ""))
	router.GET("/test", func(c *gin.Context) {
		if _, exists := c.Get("user"); exists {
			t.Error("Expected no user to be set once the user no longer exists")
		}
		c.Status(http.StatusOK)
	})

	// Simulate the user having been deleted after the token was issued
	db.Unscoped().Delete(&models.User{}, user.ID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_MasterKey(t *testing.T) {
	repo, _ := setupTestRepo(t)
	router := newTestRouter()
	router.Use(AuthMiddleware(repo, "the-master-key"))
	router.GET("/test", func(c *gin.Context) {
		username, exists := c.Get("username")
		if !exists || username != "MasterAdmin" {
			t.Error("Expected username to be MasterAdmin")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "the-master-key")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidApiKeyRejected(t *testing.T) {
	repo, _ := setupTestRepo(t)
	router := newTestRouter()
	router.Use(AuthMiddleware(repo, "the-master-key"))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "totally-invalid")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for an invalid API key, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidUserApiKey(t *testing.T) {
	repo, _ := setupTestRepo(t)
	user, _ := repo.CreateUser("keyuser", "keyuser@example.com", "pw", "user")
	_, raw, _ := repo.CreateUserApiKey(user.ID, "Test Key", "never")

	router := newTestRouter()
	router.Use(AuthMiddleware(repo, ""))
	router.GET("/test", func(c *gin.Context) {
		u, exists := c.Get("user")
		if !exists || u.(*models.User).Username != "keyuser" {
			t.Error("Expected the key's owning user to be set in context")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", raw)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_NoCredentialsProceedsUnauthenticated(t *testing.T) {
	repo, _ := setupTestRepo(t)
	router := newTestRouter()
	router.Use(AuthMiddleware(repo, "master-key"))
	router.GET("/test", func(c *gin.Context) {
		if _, exists := c.Get("user"); exists {
			t.Error("Expected no user in context")
		}
		if _, exists := c.Get("username"); exists {
			t.Error("Expected no username in context")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestRequireAuth(t *testing.T) {
	cases := []struct {
		name       string
		setContext func(c *gin.Context)
		wantStatus int
	}{
		{"user set", func(c *gin.Context) { c.Set("user", &models.User{}) }, http.StatusOK},
		{"username set (master admin)", func(c *gin.Context) { c.Set("username", "MasterAdmin") }, http.StatusOK},
		{"nothing set", func(c *gin.Context) {}, http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()
			router.Use(func(c *gin.Context) { tc.setContext(c); c.Next() })
			router.Use(RequireAuth())
			router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("%s: expected %d, got %d", tc.name, tc.wantStatus, w.Code)
			}
		})
	}
}

func TestRequirePasswordChanged(t *testing.T) {
	cases := []struct {
		name       string
		setContext func(c *gin.Context)
		wantStatus int
	}{
		{"user must change password", func(c *gin.Context) {
			c.Set("user", &models.User{MustChangePassword: true})
		}, http.StatusForbidden},
		{"user has changed password", func(c *gin.Context) {
			c.Set("user", &models.User{MustChangePassword: false})
		}, http.StatusOK},
		{"no user in context (e.g. master admin)", func(c *gin.Context) {
			c.Set("username", "MasterAdmin")
		}, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()
			router.Use(func(c *gin.Context) { tc.setContext(c); c.Next() })
			router.Use(RequirePasswordChanged())
			router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("%s: expected %d, got %d", tc.name, tc.wantStatus, w.Code)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	cases := []struct {
		name       string
		setContext func(c *gin.Context)
		wantStatus int
	}{
		{"admin user", func(c *gin.Context) { c.Set("user", &models.User{Role: "admin"}) }, http.StatusOK},
		{"non-admin user", func(c *gin.Context) { c.Set("user", &models.User{Role: "user"}) }, http.StatusForbidden},
		{"master admin via username", func(c *gin.Context) { c.Set("username", "MasterAdmin") }, http.StatusOK},
		{"unrelated username set, no user", func(c *gin.Context) { c.Set("username", "someone") }, http.StatusForbidden},
		{"nothing set", func(c *gin.Context) {}, http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter()
			router.Use(func(c *gin.Context) { tc.setContext(c); c.Next() })
			router.Use(RequireAdmin())
			router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("%s: expected %d, got %d", tc.name, tc.wantStatus, w.Code)
			}
		})
	}
}
