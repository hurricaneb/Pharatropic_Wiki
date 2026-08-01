package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"wiki/internal/middleware"
	"wiki/internal/models"
	"wiki/internal/repository"

	"github.com/gin-gonic/gin"
)

type WikiHandler struct {
	repo *repository.WikiRepository
}

func NewWikiHandler(repo *repository.WikiRepository) *WikiHandler {
	return &WikiHandler{repo: repo}
}

// GetHealth returns system status
func (h *WikiHandler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "wiki-api",
		"version": "1.0.0",
	})
}

// ListPages GET /api/v1/pages
func (h *WikiHandler) ListPages(c *gin.Context) {
	search := c.Query("search")
	tag := c.Query("tag")

	pages, err := h.repo.ListPages(search, tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": pages, "total": len(pages)})
}

// GetPage GET /api/v1/pages/:slug
func (h *WikiHandler) GetPage(c *gin.Context) {
	slug := c.Param("slug")

	page, err := h.repo.GetPageBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sidan hittades inte"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": page})
}

// CreatePage POST /api/v1/pages
func (h *WikiHandler) CreatePage(c *gin.Context) {
	var req models.CreatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Fältet 'title' och 'content' är obligatoriska"})
		return
	}

	author := c.GetString("username")
	page, err := h.repo.CreatePage(&req, author)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sidan har skapats!",
		"data":    page,
	})
}

// UpdatePage PUT /api/v1/pages/:slug
func (h *WikiHandler) UpdatePage(c *gin.Context) {
	slug := c.Param("slug")

	var req models.UpdatePageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ogiltigt dataformat"})
		return
	}

	author := c.GetString("username")
	page, err := h.repo.UpdatePage(slug, &req, author)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sidan har uppdaterats!",
		"data":    page,
	})
}

// DeletePage DELETE /api/v1/pages/:slug
func (h *WikiHandler) DeletePage(c *gin.Context) {
	slug := c.Param("slug")

	if err := h.repo.DeletePage(slug); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sidan har raderats"})
}

// GetRevisions GET /api/v1/pages/:slug/revisions
func (h *WikiHandler) GetRevisions(c *gin.Context) {
	slug := c.Param("slug")

	revisions, err := h.repo.GetRevisions(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": revisions, "total": len(revisions)})
}

// SearchPages GET /api/v1/search?q=...
func (h *WikiHandler) SearchPages(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sökparameter 'q' saknas"})
		return
	}

	pages, err := h.repo.SearchPages(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"query": q, "results": pages, "total": len(pages)})
}

// ListTags GET /api/v1/tags
func (h *WikiHandler) ListTags(c *gin.Context) {
	tags, err := h.repo.ListTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tags})
}

// UploadAttachment POST /api/v1/pages/:slug/attachments
func (h *WikiHandler) UploadAttachment(c *gin.Context) {
	slug := c.Param("slug")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ingen fil skickades i formuläret ('file')"})
		return
	}

	// Ensure upload directory exists
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kunde inte skapa uppladdningsmapp"})
		return
	}

	// Generate safe unique filename (replace spaces with underscores for valid Markdown URLs)
	rawBase := filepath.Base(file.Filename)
	cleanBase := strings.ReplaceAll(rawBase, " ", "_")
	uniqueFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), cleanBase)
	dst := filepath.Join(uploadDir, uniqueFilename)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Misslyckades att spara filen"})
		return
	}

	filePath := fmt.Sprintf("/uploads/%s", uniqueFilename)
	mimeType := file.Header.Get("Content-Type")

	attachment, err := h.repo.SaveAttachment(slug, uniqueFilename, file.Filename, filePath, mimeType, file.Size)
	if err != nil {
		os.Remove(dst) // rollback file save on DB error
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate Markdown snippet with encoded spaces in URL if any
	encodedFilePath := strings.ReplaceAll(filePath, " ", "%20")
	var markdownSnippet string
	if strings.HasPrefix(mimeType, "image/") {
		markdownSnippet = fmt.Sprintf("![%s](%s)", file.Filename, encodedFilePath)
	} else {
		markdownSnippet = fmt.Sprintf("[%s](%s)", file.Filename, encodedFilePath)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Filen har laddats upp!",
		"data":     attachment,
		"markdown": markdownSnippet,
	})
}

// GetAttachments GET /api/v1/pages/:slug/attachments
func (h *WikiHandler) GetAttachments(c *gin.Context) {
	slug := c.Param("slug")

	attachments, err := h.repo.GetAttachments(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": attachments, "total": len(attachments)})
}

// DeleteAttachment DELETE /api/v1/attachments/:id
func (h *WikiHandler) DeleteAttachment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ogiltigt ID"})
		return
	}

	att, err := h.repo.GetAttachmentByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Remove file from disk
	filePath := filepath.Join("./uploads", att.Filename)
	os.Remove(filePath)

	if err := h.repo.DeleteAttachment(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bilagan har raderats"})
}

// GetBacklinks GET /api/v1/pages/:slug/backlinks
func (h *WikiHandler) GetBacklinks(c *gin.Context) {
	slug := c.Param("slug")

	backlinks, err := h.repo.GetBacklinks(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": backlinks, "total": len(backlinks)})
}

// RevertRevision POST /api/v1/pages/:slug/revert/:revision_id
func (h *WikiHandler) RevertRevision(c *gin.Context) {
	slug := c.Param("slug")
	revIDStr := c.Param("revision_id")

	revID, err := strconv.ParseUint(revIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ogiltigt revisions-ID"})
		return
	}

	author := c.GetString("username")
	page, err := h.repo.RevertPageRevision(slug, uint(revID), author)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Sidan har återställts till revision #%d!", revID),
		"data":    page,
	})
}

// Auth Handlers

// AuthLogin POST /api/v1/auth/login
func (h *WikiHandler) AuthLogin(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Användarnamn och lösenord krävs"})
		return
	}

	user, err := h.repo.GetUserByUsernameOrEmail(req.Username)
	if err != nil || !h.repo.ValidateUserPassword(user, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Felaktigt användarnamn eller lösenord"})
		return
	}

	token, err := middleware.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kunde inte skapa inloggningstoken"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Inloggningen lyckades!",
		"token":   token,
		"user":    user,
	})
}

// AuthMe GET /api/v1/auth/me
func (h *WikiHandler) AuthMe(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		username := c.GetString("username")
		if username != "" {
			c.JSON(http.StatusOK, gin.H{"user": gin.H{"username": username, "role": "admin"}})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Ej inloggad"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": userVal})
}

// Admin Handlers

// AdminCreateUser POST /api/v1/admin/users
func (h *WikiHandler) AdminCreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Användarnamn, e-post och lösenord krävs"})
		return
	}

	user, err := h.repo.CreateUser(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Användarkontot har skapats!",
		"data":    user,
	})
}

// AdminListUsers GET /api/v1/admin/users
func (h *WikiHandler) AdminListUsers(c *gin.Context) {
	users, err := h.repo.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

// User API Key Handlers

// UserListApiKeys GET /api/v1/user/keys
func (h *WikiHandler) UserListApiKeys(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Du måste vara inloggad som användare"})
		return
	}

	user := userVal.(*models.User)
	keys, err := h.repo.ListUserApiKeys(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": keys})
}

// UserCreateApiKey POST /api/v1/user/keys
func (h *WikiHandler) UserCreateApiKey(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Du måste vara inloggad som användare"})
		return
	}

	var req models.CreateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nyckelnamn krävs"})
		return
	}

	user := userVal.(*models.User)
	apiKey, rawSecret, err := h.repo.CreateUserApiKey(user.ID, req.Name, req.Expires)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "API-nyckeln har skapats!",
		"data":    apiKey,
		"key":     rawSecret,
	})
}

// UserRevokeApiKey DELETE /api/v1/user/keys/:id
func (h *WikiHandler) UserRevokeApiKey(c *gin.Context) {
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Du måste vara inloggad som användare"})
		return
	}

	keyIDStr := c.Param("id")
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ogiltigt nyckel-ID"})
		return
	}

	user := userVal.(*models.User)
	if err := h.repo.RevokeUserApiKey(user.ID, uint(keyID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API-nyckeln har återkallats"})
}
