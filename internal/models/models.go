package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Username           string         `gorm:"uniqueIndex;not null" json:"username"`
	Email              string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash       string         `gorm:"not null" json:"-"`
	Role               string         `gorm:"default:'user'" json:"role"` // 'admin' or 'user'
	MustChangePassword bool           `gorm:"default:false" json:"must_change_password"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
	ApiKeys            []ApiKey       `json:"api_keys,omitempty" gorm:"foreignKey:UserID"`
}

// Page represents a main wiki page. Pages may have one level of subpages:
// a page with a ParentID cannot itself be a parent (enforced in the
// repository), keeping the hierarchy exactly one level deep.
type Page struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
	Title       string         `gorm:"not null" json:"title"`
	Summary     string         `json:"summary"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	IsPublic    bool           `gorm:"default:false;index" json:"is_public"`
	Views       int64          `gorm:"default:0" json:"views"`
	ParentID    *uint          `gorm:"index" json:"parent_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Parent      *Page          `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID"`
	Children    []Page         `json:"children,omitempty" gorm:"foreignKey:ParentID;references:ID"`
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

// ApiKey represents an API key owned by a user
type ApiKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	Name       string     `gorm:"not null" json:"name"`
	Key        string     `gorm:"uniqueIndex;not null" json:"-"` // SHA-256 hash of the raw secret, never serialized
	Prefix     string     `json:"prefix"`
	Active     bool       `gorm:"default:true" json:"active"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreatePageRequest DTO for API page creation
type CreatePageRequest struct {
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	Summary    string   `json:"summary"`
	Comment    string   `json:"comment"`
	IsPublic   *bool    `json:"is_public"`
	Tags       []string `json:"tags"`
	ParentSlug string   `json:"parent_slug"` // slug of the top-level page this becomes a subpage of, empty for a top-level page
}

// UpdatePageRequest DTO for API page update
type UpdatePageRequest struct {
	Title      string   `json:"title"`
	Content    string   `json:"content" binding:"required"`
	Summary    string   `json:"summary"`
	Comment    string   `json:"comment"`
	IsPublic   *bool    `json:"is_public"`
	Tags       []string `json:"tags"`
	ParentSlug *string  `json:"parent_slug"` // nil = unchanged, "" = detach to top-level, else = set parent
}

// Auth DTOs
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

type CreateApiKeyRequest struct {
	Name      string `json:"name" binding:"required"`
	Expires   string `json:"expires"`    // 'never', '7d', '30d', '90d', '1y'
	ExpiresAt string `json:"expires_at"` // fallback alias
}
