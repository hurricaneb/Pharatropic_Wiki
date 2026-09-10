package mcp

import (
	"fmt"
	"os"
	"testing"
	"time"

	"wiki/internal/models"
	"wiki/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newIsolatedRepo returns a repository backed by a fresh, unseeded database
// (unlike setupTestRepo in mcp_test.go, which seeds a default admin and
// welcome page via database.InitDB) so tests can assert exact counts.
func newIsolatedRepo(t *testing.T) *repository.WikiRepository {
	t.Helper()
	dbFile := fmt.Sprintf("test_mcp_tools_%d_%d.db", time.Now().UnixNano(), os.Getpid())
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Page{}, &models.Revision{}, &models.Attachment{}, &models.Tag{}, &models.ApiKey{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}
	t.Cleanup(func() { os.Remove(dbFile) })
	return repository.NewWikiRepository(db)
}

func TestExecuteToolCall_ListPages(t *testing.T) {
	repo := newIsolatedRepo(t)
	isPublic := true
	repo.CreatePage(&models.CreatePageRequest{Title: "Sida Ett", Content: "x", IsPublic: &isPublic}, "tester")

	res, err := ExecuteToolCall(repo, "list_pages", map[string]interface{}{}, "tester", true)
	if err != nil || res.IsError {
		t.Fatalf("unexpected error/tool-error: err=%v res=%+v", err, res)
	}

	filtered, err := ExecuteToolCall(repo, "list_pages", map[string]interface{}{"search": "finns-inte-alls"}, "tester", true)
	if err != nil || filtered.IsError {
		t.Fatalf("unexpected error/tool-error filtering: err=%v res=%+v", err, filtered)
	}
}

func TestExecuteToolCall_ReadPage(t *testing.T) {
	repo := newIsolatedRepo(t)
	repo.CreatePage(&models.CreatePageRequest{Title: "Läsbar Sida", Content: "innehåll"}, "tester")

	missingSlug, _ := ExecuteToolCall(repo, "read_page", map[string]interface{}{}, "tester", true)
	if !missingSlug.IsError {
		t.Fatal("Expected tool error when 'slug' is missing")
	}

	notFound, _ := ExecuteToolCall(repo, "read_page", map[string]interface{}{"slug": "finns-inte"}, "tester", true)
	if !notFound.IsError {
		t.Fatal("Expected tool error reading a nonexistent page")
	}

	found, _ := ExecuteToolCall(repo, "read_page", map[string]interface{}{"slug": "lasbar-sida"}, "tester", true)
	if found.IsError {
		t.Fatalf("Expected successful read, got error: %+v", found)
	}
}

func TestExecuteToolCall_SearchPages(t *testing.T) {
	repo := newIsolatedRepo(t)
	repo.CreatePage(&models.CreatePageRequest{Title: "Sökbar Katt", Content: "x"}, "tester")

	missingQuery, _ := ExecuteToolCall(repo, "search_pages", map[string]interface{}{}, "tester", true)
	if !missingQuery.IsError {
		t.Fatal("Expected tool error when 'query' is missing")
	}

	found, _ := ExecuteToolCall(repo, "search_pages", map[string]interface{}{"query": "katt"}, "tester", true)
	if found.IsError {
		t.Fatalf("Expected successful search, got error: %+v", found)
	}
}

func TestExecuteToolCall_CreatePage(t *testing.T) {
	repo := newIsolatedRepo(t)

	missingFields, _ := ExecuteToolCall(repo, "create_page", map[string]interface{}{"title": "Bara Titel"}, "", true)
	if !missingFields.IsError {
		t.Fatal("Expected tool error when 'content' is missing")
	}

	full, _ := ExecuteToolCall(repo, "create_page", map[string]interface{}{
		"title":     "MCP Full Page",
		"content":   "# Hej",
		"summary":   "sammanfattning",
		"is_public": true,
		"tags":      []interface{}{"a", "", "b"},
	}, "", true)
	if full.IsError {
		t.Fatalf("Expected successful creation, got error: %+v", full)
	}

	duplicate, _ := ExecuteToolCall(repo, "create_page", map[string]interface{}{
		"title": "MCP Full Page", "content": "x",
	}, "tester", true)
	if !duplicate.IsError {
		t.Fatal("Expected tool error creating a page with a duplicate title")
	}
}

func TestExecuteToolCall_UpdatePage(t *testing.T) {
	repo := newIsolatedRepo(t)
	repo.CreatePage(&models.CreatePageRequest{Title: "Uppdateras", Content: "original"}, "tester")

	missingFields, _ := ExecuteToolCall(repo, "update_page", map[string]interface{}{"slug": "uppdateras"}, "tester", true)
	if !missingFields.IsError {
		t.Fatal("Expected tool error when 'content' is missing")
	}

	notFound, _ := ExecuteToolCall(repo, "update_page", map[string]interface{}{"slug": "finns-inte", "content": "x"}, "tester", true)
	if !notFound.IsError {
		t.Fatal("Expected tool error updating a nonexistent page")
	}

	ok, _ := ExecuteToolCall(repo, "update_page", map[string]interface{}{
		"slug": "uppdateras", "content": "nytt", "title": "Nytt Namn", "summary": "s",
		"is_public": true, "comment": "ändrat", "tags": []interface{}{"x"},
	}, "tester", true)
	if ok.IsError {
		t.Fatalf("Expected successful update, got error: %+v", ok)
	}
}

func TestExecuteToolCall_GetBacklinks(t *testing.T) {
	repo := newIsolatedRepo(t)
	repo.CreatePage(&models.CreatePageRequest{Title: "Mål", Content: "x"}, "tester")
	repo.CreatePage(&models.CreatePageRequest{Title: "Länkare", Content: "se [[Mål]]"}, "tester")

	missingSlug, _ := ExecuteToolCall(repo, "get_backlinks", map[string]interface{}{}, "tester", true)
	if !missingSlug.IsError {
		t.Fatal("Expected tool error when 'slug' is missing")
	}

	notFound, _ := ExecuteToolCall(repo, "get_backlinks", map[string]interface{}{"slug": "finns-inte"}, "tester", true)
	if !notFound.IsError {
		t.Fatal("Expected tool error for a nonexistent target page")
	}

	ok, _ := ExecuteToolCall(repo, "get_backlinks", map[string]interface{}{"slug": "mal"}, "tester", true)
	if ok.IsError {
		t.Fatalf("Expected successful backlinks lookup, got error: %+v", ok)
	}
}

func TestExecuteToolCall_ListTags(t *testing.T) {
	repo := newIsolatedRepo(t)
	repo.CreatePage(&models.CreatePageRequest{Title: "Taggad", Content: "x", Tags: []string{"go"}}, "tester")

	res, err := ExecuteToolCall(repo, "list_tags", map[string]interface{}{}, "tester", true)
	if err != nil || res.IsError {
		t.Fatalf("unexpected error/tool-error: err=%v res=%+v", err, res)
	}
}

func TestExecuteToolCall_UnknownTool(t *testing.T) {
	repo := newIsolatedRepo(t)
	res, err := ExecuteToolCall(repo, "does_not_exist", map[string]interface{}{}, "tester", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("Expected tool error for an unknown tool name")
	}
}

func TestGetResourceDefinitions(t *testing.T) {
	repo := newIsolatedRepo(t)
	isPublic := true
	isPrivate := false
	repo.CreatePage(&models.CreatePageRequest{Title: "Publik Resurs", Content: "x", Summary: "s", IsPublic: &isPublic}, "tester")
	repo.CreatePage(&models.CreatePageRequest{Title: "Privat Resurs", Content: "x", IsPublic: &isPrivate}, "tester")

	authed, err := GetResourceDefinitions(repo, true)
	if err != nil || len(authed) != 2 {
		t.Fatalf("Expected 2 resources when authenticated, got %d, err=%v", len(authed), err)
	}
	if authed[0].URI == "" || authed[0].MIMEType != "text/markdown" {
		t.Fatalf("Unexpected resource shape: %+v", authed[0])
	}

	unauthed, err := GetResourceDefinitions(repo, false)
	if err != nil || len(unauthed) != 1 {
		t.Fatalf("Expected 1 resource when unauthenticated, got %d, err=%v", len(unauthed), err)
	}
}

func TestReadResource(t *testing.T) {
	repo := newIsolatedRepo(t)
	isPrivate := false
	repo.CreatePage(&models.CreatePageRequest{Title: "Läsbar Resurs", Content: "resursinnehåll"}, "tester")
	repo.CreatePage(&models.CreatePageRequest{Title: "Privat Resurs", Content: "hemligt", IsPublic: &isPrivate}, "tester")

	if _, err := ReadResource(repo, "not-a-wiki-uri", true); err == nil {
		t.Fatal("Expected error for a URI without the wiki://pages/ prefix")
	}

	if _, err := ReadResource(repo, "wiki://pages/finns-inte", true); err == nil {
		t.Fatal("Expected error reading a nonexistent page resource")
	}

	result, err := ReadResource(repo, "wiki://pages/lasbar-resurs", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 1 || result.Contents[0].Text != "resursinnehåll" {
		t.Fatalf("Unexpected resource content: %+v", result.Contents)
	}

	if _, err := ReadResource(repo, "wiki://pages/privat-resurs", false); err == nil {
		t.Fatal("Expected error reading a private page resource while unauthenticated")
	}
}
