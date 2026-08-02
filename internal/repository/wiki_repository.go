package repository

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"wiki/internal/models"

	"github.com/gosimple/slug"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type WikiRepository struct {
	db *gorm.DB
}

func NewWikiRepository(db *gorm.DB) *WikiRepository {
	return &WikiRepository{db: db}
}

func (r *WikiRepository) CreatePage(req *models.CreatePageRequest, author string) (*models.Page, error) {
	pageSlug := slug.Make(req.Title)
	if pageSlug == "" {
		return nil, errors.New("ogiltig titel")
	}

	if author == "" {
		author = "Användare"
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

	isPublic := false
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	page := models.Page{
		Title:    req.Title,
		Slug:     pageSlug,
		Summary:  req.Summary,
		Content:  req.Content,
		IsPublic: isPublic,
		Tags:     tags,
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
		Author:  author,
	}
	r.db.Create(&revision)

	return &page, nil
}

func (r *WikiRepository) GetPageBySlug(slug string, isAuthed bool) (*models.Page, error) {
	var page models.Page
	err := r.db.Preload("Tags").Preload("Attachments").Preload("Revisions", func(db *gorm.DB) *gorm.DB {
		return db.Order("revisions.created_at DESC")
	}).Where("slug = ?", slug).First(&page).Error
	if err != nil {
		return nil, err
	}

	if !isAuthed && !page.IsPublic {
		return nil, errors.New("denna sida är privat och kräver inloggning")
	}

	// Increment view count asynchronously/in background
	r.db.Model(&page).UpdateColumn("views", gorm.Expr("views + ?", 1))

	return &page, nil
}

func (r *WikiRepository) ListPages(search string, tag string, isAuthed bool) ([]models.Page, error) {
	var pages []models.Page
	query := r.db.Preload("Tags").Order("updated_at DESC")

	if !isAuthed {
		query = query.Where("is_public = ?", true)
	}

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

func (r *WikiRepository) UpdatePage(pageSlug string, req *models.UpdatePageRequest, author string) (*models.Page, error) {
	var page models.Page
	if err := r.db.Preload("Tags").Where("slug = ?", pageSlug).First(&page).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	if author == "" {
		author = "Användare"
	}

	if req.Title != "" && req.Title != page.Title {
		page.Title = req.Title
	}

	page.Content = req.Content
	page.Summary = req.Summary
	if req.IsPublic != nil {
		page.IsPublic = *req.IsPublic
	}
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
		Author:  author,
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

func (r *WikiRepository) SearchPages(q string, isAuthed bool) ([]models.Page, error) {
	return r.ListPages(q, "", isAuthed)
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

func (r *WikiRepository) GetBacklinks(targetSlug string, isAuthed bool) ([]models.Page, error) {
	var targetPage models.Page
	if err := r.db.Where("slug = ?", targetSlug).First(&targetPage).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	lowerTitle := strings.ToLower(targetPage.Title)
	lowerSlug := strings.ToLower(targetSlug)

	p1 := "%[[" + lowerTitle + "]]%"
	p2 := "%[[" + lowerTitle + "|%"
	p3 := "%[[" + lowerSlug + "]]%"
	p4 := "%[[" + lowerSlug + "|%"
	p5 := "%/pages/" + lowerSlug + "%"
	p6 := "%/wiki/" + lowerSlug + "%"
	p7 := "%#wikilink:" + lowerSlug + "%"

	query := r.db.Preload("Tags").
		Where("id != ?", targetPage.ID).
		Where(
			"LOWER(content) LIKE ? OR LOWER(content) LIKE ? OR LOWER(content) LIKE ? OR LOWER(content) LIKE ? OR LOWER(content) LIKE ? OR LOWER(content) LIKE ? OR LOWER(content) LIKE ?",
			p1, p2, p3, p4, p5, p6, p7,
		)

	if !isAuthed {
		query = query.Where("is_public = ?", true)
	}

	var backlinks []models.Page
	err := query.Order("updated_at DESC").Find(&backlinks).Error

	return backlinks, err
}

func (r *WikiRepository) RevertPageRevision(pageSlug string, revisionID uint, author string) (*models.Page, error) {
	var page models.Page
	if err := r.db.Where("slug = ?", pageSlug).First(&page).Error; err != nil {
		return nil, errors.New("sidan hittades inte")
	}

	if author == "" {
		author = "Användare"
	}

	var targetRev models.Revision
	if err := r.db.Where("id = ? AND page_id = ?", revisionID, page.ID).First(&targetRev).Error; err != nil {
		return nil, errors.New("revisionen hittades inte för denna sida")
	}

	// Compute relative 1-based revision sequence number for this specific page
	var allRevs []models.Revision
	r.db.Where("page_id = ?", page.ID).Order("created_at ASC").Find(&allRevs)

	seqNo := 0
	for idx, rev := range allRevs {
		if rev.ID == targetRev.ID {
			seqNo = idx + 1
			break
		}
	}
	if seqNo == 0 {
		seqNo = int(targetRev.ID)
	}

	page.Content = targetRev.Content
	page.Title = targetRev.Title
	page.UpdatedAt = time.Now()

	if err := r.db.Save(&page).Error; err != nil {
		return nil, err
	}

	comment := fmt.Sprintf("Återställd till revision #%d", seqNo)
	newRev := models.Revision{
		PageID:  page.ID,
		Title:   page.Title,
		Content: page.Content,
		Comment: comment,
		Author:  author,
	}
	r.db.Create(&newRev)

	return &page, nil
}

// User repository methods

func (r *WikiRepository) CreateUser(username, email, password, role string) (*models.User, error) {
	if role == "" {
		role = "user"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *WikiRepository) GetUserByUsernameOrEmail(identifier string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error
	if err != nil {
		return nil, errors.New("användaren hittades inte")
	}
	return &user, nil
}

func (r *WikiRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, errors.New("användaren hittades inte")
	}
	return &user, nil
}

func (r *WikiRepository) ListUsers() ([]models.User, error) {
	var users []models.User
	err := r.db.Order("created_at ASC").Find(&users).Error
	return users, err
}

func (r *WikiRepository) ValidateUserPassword(user *models.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// ApiKey repository methods

func (r *WikiRepository) CreateUserApiKey(userID uint, name string, expiresOption string) (*models.ApiKey, string, error) {
	var user models.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return nil, "", errors.New("användaren hittades inte")
	}

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, "", err
	}
	rawSecret := "ptc_key_" + hex.EncodeToString(bytes)
	prefix := rawSecret[:12] + "..."

	var expiresAt *time.Time
	now := time.Now()

	switch expiresOption {
	case "7d":
		t := now.Add(7 * 24 * time.Hour)
		expiresAt = &t
	case "30d":
		t := now.Add(30 * 24 * time.Hour)
		expiresAt = &t
	case "90d":
		t := now.Add(90 * 24 * time.Hour)
		expiresAt = &t
	case "1y":
		t := now.Add(365 * 24 * time.Hour)
		expiresAt = &t
	case "never", "":
		expiresAt = nil
	}

	apiKey := models.ApiKey{
		UserID:    userID,
		Name:      name,
		Key:       rawSecret,
		Prefix:    prefix,
		Active:    true,
		ExpiresAt: expiresAt,
	}

	if err := r.db.Create(&apiKey).Error; err != nil {
		return nil, "", err
	}

	return &apiKey, rawSecret, nil
}

func (r *WikiRepository) ListUserApiKeys(userID uint) ([]models.ApiKey, error) {
	var keys []models.ApiKey
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (r *WikiRepository) RevokeUserApiKey(userID uint, keyID uint) error {
	return r.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&models.ApiKey{}).Error
}

func (r *WikiRepository) ValidateApiKey(rawKey string) (*models.User, *models.ApiKey, error) {
	var apiKey models.ApiKey
	if err := r.db.Where("key = ? AND active = ?", rawKey, true).First(&apiKey).Error; err != nil {
		return nil, nil, errors.New("ogiltig eller inaktiv API-nyckel")
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, nil, errors.New("API-nyckeln har gått ut")
	}

	now := time.Now()
	apiKey.LastUsedAt = &now
	r.db.Model(&apiKey).Update("last_used_at", now)

	var user models.User
	if err := r.db.First(&user, apiKey.UserID).Error; err != nil {
		return nil, nil, errors.New("användaren för denna API-nyckel finns inte")
	}

	return &user, &apiKey, nil
}
