package database

import (
	"log"

	"wiki/internal/models"

	"github.com/gosimple/slug"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto migrate tables
	err = db.AutoMigrate(
		&models.User{},
		&models.Page{},
		&models.Revision{},
		&models.Attachment{},
		&models.Tag{},
		&models.ApiKey{},
	)
	if err != nil {
		return nil, err
	}

	// Seed default admin account if no users exist
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		adminUser := models.User{
			Username:     "admin",
			Email:        "admin@pharatropic.local",
			PasswordHash: string(hash),
			Role:         "admin",
		}
		if err := db.Create(&adminUser).Error; err == nil {
			log.Println("Seeded default admin user (username: admin, password: admin)")
		}
	}

	// Seed initial welcome page if no pages exist
	var count int64
	db.Model(&models.Page{}).Count(&count)
	if count == 0 {
		seedDatabase(db)
	}

	return db, nil
}

func seedDatabase(db *gorm.DB) {
	log.Println("Seeding database with default start page...")

	welcomeTitle := "Välkommen till Wikin"
	welcomeSlug := slug.Make(welcomeTitle)
	welcomeContent := "# Välkommen till Pharatropic Wiki (PTC Wiki)! 🚀\n\nDetta är en modern, snabb och utökbar Wiki skriven i **Go** med ett komplett **REST API** och ett **React UI**.\n\n## 📖 Funktioner\n- **Markdown-stöd**: Skriv i ren Markdown med automatisk kodmarkering och preview.\n- **Versionshistorik**: Varje ändring sparas automatiskt som en ny revision så att du kan jämföra tidigare versioner.\n- **Fulltextsökning**: Hitta sidor blixtsnabbt.\n- **REST API (`/api/v1`)**: Integrera wikin med externa system via HTTP REST API.\n\n## 🛠️ REST API-exempel\nDu kan hämta denna sida via API:t:\n```bash\ncurl http://localhost:8080/api/v1/pages/valkommen-till-wikin\n```\n\nSkapa gärna nya sidor eller redigera denna!\n"

	startTag := models.Tag{Name: "Start", Slug: "start"}
	guideTag := models.Tag{Name: "Guide", Slug: "guide"}
	db.FirstOrCreate(&startTag, models.Tag{Slug: "start"})
	db.FirstOrCreate(&guideTag, models.Tag{Slug: "guide"})

	page := models.Page{
		Title:   welcomeTitle,
		Slug:    welcomeSlug,
		Summary: "Välkomstsida för Pharatropic Wiki (PTC Wiki)",
		Content: welcomeContent,
		Tags:    []models.Tag{startTag, guideTag},
	}

	if err := db.Create(&page).Error; err == nil {
		// Create initial revision
		revision := models.Revision{
			PageID:  page.ID,
			Title:   page.Title,
			Content: page.Content,
			Comment: "Inledande skapande av välkomstsida",
			Author:  "admin",
		}
		db.Create(&revision)
	}
}
