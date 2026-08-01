package handlers

import (
	"net/http"

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

	page, err := h.repo.CreatePage(&req)
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

	page, err := h.repo.UpdatePage(slug, &req)
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
