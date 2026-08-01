package models

import (
	"time"

	"gorm.io/gorm"
)

// Page represents a main wiki page
type Page struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Slug      string         `gorm:"uniqueIndex;not null" json:"slug"`
	Title     string         `gorm:"not null" json:"title"`
	Summary   string         `json:"summary"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Views     int64          `gorm:"default:0" json:"views"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Revisions   []Revision     `json:"revisions,omitempty" gorm:"foreignKey:PageID"`
	Attachments []Attachment   `json:"attachments,omitempty" gorm:"foreignKey:PageID"`
	Tags        []Tag          `json:"tags,omitempty" gorm:"many2many:page_tags;"`
}

// Attachment represents an uploaded file associated with a page
type Attachment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PageID    uint      `gorm:"index;not null" json:"page_id"`
	Filename  string    `gorm:"not null" json:"filename"`
	Original  string    `gorm:"not null" json:"original_name"`
	FilePath  string    `gorm:"not null" json:"file_path"`
	MimeType  string    `json:"mime_type"`
	FileSize  int64     `json:"file_size"`
	CreatedAt time.Time `json:"created_at"`
}

// Revision represents a historical edit version of a wiki page
type Revision struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PageID    uint      `gorm:"index;not null" json:"page_id"`
	Title     string    `gorm:"not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Comment   string    `json:"comment"`
	Author    string    `gorm:"default:'Anonym'" json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// Tag represents a tag attached to wiki pages
type Tag struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;not null" json:"name"`
	Slug string `gorm:"uniqueIndex;not null" json:"slug"`
}

// ApiKey represents an API key for REST API access
type ApiKey struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"uniqueIndex;not null" json:"key"`
	Description string    `json:"description"`
	Active      bool      `gorm:"default:true" json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreatePageRequest DTO for API page creation
type CreatePageRequest struct {
	Title   string   `json:"title" binding:"required"`
	Content string   `json:"content" binding:"required"`
	Summary string   `json:"summary"`
	Comment string   `json:"comment"`
	Tags    []string `json:"tags"`
}

// UpdatePageRequest DTO for API page update
type UpdatePageRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content" binding:"required"`
	Summary string   `json:"summary"`
	Comment string   `json:"comment"`
	Tags    []string `json:"tags"`
}
