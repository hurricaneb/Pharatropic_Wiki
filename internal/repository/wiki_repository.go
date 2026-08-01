package repository

import (
	"errors"
	"strings"
	"time"

	"wiki/internal/models"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

type WikiRepository struct {
	db *gorm.DB
}

func NewWikiRepository(db *gorm.DB) *WikiRepository {
	return &WikiRepository{db: db}
}

func (r *WikiRepository) CreatePage(req *models.CreatePageRequest) (*models.Page, error) {
	pageSlug := slug.Make(req.Title)
	if pageSlug == "" {
		return nil, errors.New("ogiltig titel")
	}

	var existing models.Page
	if err := r.db.Where("slug = ?", pageSlug).First(&existing).Error; err == nil {
		return nil, errors.New("en sida med den titeln finns redan")
	}

	var tags []models.Tag
	for _, tName := range req.Tags {
		tName = strings.TrimSpace(tName)
		if tName == "" {
			continue
		}
		tSlug := slug.Make(tName)
		var tag models.Tag
		r.db.Where(models.Tag{Slug: tSlug}).FirstOrCreate(&tag, models.Tag{Name: tName, Slug: tSlug})
		tags = append(tags, tag)
	}

	comment := req.Comment
	if comment == "" {
		comment = "Sida skapad"
	}

	page := models.Page{
		Title:   req.Title,
		Slug:    pageSlug,
		Summary: req.Summary,
		Content: req.Content,
		Tags:    tags,
	}

	if err := r.db.Create(&page).Error; err != nil {
		return nil, err
	}

	// Create initial revision
	revision := models.Revision{
		PageID:  page.ID,
		Title:   page.Title,
		Content: page.Content,
		Comment: comment,
		Author:  "Användare",
	}
	r.db.Create(&revision)

	return &page, nil
}

func (r *WikiRepository) GetPageBySlug(slug string) (*models.Page, error) {
	var page models.Page
	err := r.db.Preload("Tags").Preload("Attachments").Preload("Revisions", func(db *gorm.DB) *gorm.DB {
		return db.Order("revisions.created_at DESC")
	}).Where("slug = ?", slug).First(&page).Error
	if err != nil {
		return nil, err
	}

	// Increment view count asynchronously/in background
	r.db.Model(&page).UpdateColumn("views", gorm.Expr("views + ?", 1))

	return &page, nil
}

func (r *WikiRepository) ListPages(search string, tag string) ([]models.Page, error) {
	var pages []models.Page
	query := r.db.Preload("Tags").Order("updated_at DESC")

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?", searchPattern, searchPattern)
	}

	if tag != "" {
		tagSlug := slug.Make(tag)
		query = query.Joins("JOIN page_tags ON page_tags.page_id = pages.id").
			Joins("JOIN tags ON tags.id = page_tags.tag_id").
			Where("tags.slug = ?", tagSlug)
	}

	err := query.Find(&pages).Error
	return pages, err
}

func (r *WikiRepository) UpdatePage(pageSlug string, req *models.UpdatePageRequest) (*models.Page, error) {
	var page models.Page
	if err := r.db.Preload("Tags").Where("slug = ?", pageSlug).First(&page).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	if req.Title != "" && req.Title != page.Title {
		page.Title = req.Title
	}

	page.Content = req.Content
	page.Summary = req.Summary
	page.UpdatedAt = time.Now()

	// Update tags if provided
	if req.Tags != nil {
		var tags []models.Tag
		for _, tName := range req.Tags {
			tName = strings.TrimSpace(tName)
			if tName == "" {
				continue
			}
			tSlug := slug.Make(tName)
			var tag models.Tag
			r.db.Where(models.Tag{Slug: tSlug}).FirstOrCreate(&tag, models.Tag{Name: tName, Slug: tSlug})
			tags = append(tags, tag)
		}
		r.db.Model(&page).Association("Tags").Replace(tags)
	}

	if err := r.db.Save(&page).Error; err != nil {
		return nil, err
	}

	comment := req.Comment
	if comment == "" {
		comment = "Sida uppdaterad"
	}

	// Create new revision
	revision := models.Revision{
		PageID:  page.ID,
		Title:   page.Title,
		Content: page.Content,
		Comment: comment,
		Author:  "Användare",
	}
	r.db.Create(&revision)

	return &page, nil
}

func (r *WikiRepository) DeletePage(slug string) error {
	var page models.Page
	if err := r.db.Where("slug = ?", slug).First(&page).Error; err != nil {
		return errors.New("sidan hittades inte")
	}
	return r.db.Delete(&page).Error
}

func (r *WikiRepository) GetRevisions(slug string) ([]models.Revision, error) {
	var page models.Page
	if err := r.db.Where("slug = ?", slug).First(&page).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	var revisions []models.Revision
	err := r.db.Where("page_id = ?", page.ID).Order("created_at DESC").Find(&revisions).Error
	return revisions, err
}

func (r *WikiRepository) SearchPages(q string) ([]models.Page, error) {
	return r.ListPages(q, "")
}

func (r *WikiRepository) ListTags() ([]models.Tag, error) {
	var tags []models.Tag
	err := r.db.Find(&tags).Error
	return tags, err
}

func (r *WikiRepository) SaveAttachment(pageSlug string, filename string, originalName string, filePath string, mimeType string, fileSize int64) (*models.Attachment, error) {
	var page models.Page
	if err := r.db.Where("slug = ?", pageSlug).First(&page).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	attachment := models.Attachment{
		PageID:   page.ID,
		Filename: filename,
		Original: originalName,
		FilePath: filePath,
		MimeType: mimeType,
		FileSize: fileSize,
	}

	if err := r.db.Create(&attachment).Error; err != nil {
		return nil, err
	}

	return &attachment, nil
}

func (r *WikiRepository) GetAttachments(pageSlug string) ([]models.Attachment, error) {
	var page models.Page
	if err := r.db.Where("slug = ?", pageSlug).First(&page).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	var attachments []models.Attachment
	err := r.db.Where("page_id = ?", page.ID).Order("created_at DESC").Find(&attachments).Error
	return attachments, err
}

func (r *WikiRepository) GetAttachmentByID(id uint) (*models.Attachment, error) {
	var att models.Attachment
	if err := r.db.First(&att, id).Error; err != nil {
		return nil, errors.New("bilagan hittades inte")
	}
	return &att, nil
}

func (r *WikiRepository) DeleteAttachment(id uint) error {
	return r.db.Delete(&models.Attachment{}, id).Error
}

func (r *WikiRepository) GetBacklinks(targetSlug string) ([]models.Page, error) {
	var targetPage models.Page
	if err := r.db.Where("slug = ?", targetSlug).First(&targetPage).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	wikiLinkPattern1 := "%[[" + targetPage.Title + "]]%"
	wikiLinkPattern2 := "%[[" + targetPage.Title + "|%"
	slugPattern1 := "%/pages/" + targetSlug + "%"
	slugPattern2 := "%/wiki/" + targetSlug + "%"
	wikilinkSchemePattern := "%#wikilink:" + targetSlug + "%"

	var backlinks []models.Page
	err := r.db.Preload("Tags").
		Where("id != ?", targetPage.ID).
		Where(
			"content LIKE ? OR content LIKE ? OR content LIKE ? OR content LIKE ? OR content LIKE ?",
			wikiLinkPattern1, wikiLinkPattern2, slugPattern1, slugPattern2, wikilinkSchemePattern,
		).
		Order("updated_at DESC").
		Find(&backlinks).Error

	return backlinks, err
}
