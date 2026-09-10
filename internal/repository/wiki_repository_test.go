package repository

import (
	"fmt"
	"os"
	"testing"
	"time"

	"wiki/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *WikiRepository {
	t.Helper()
	dbFile := fmt.Sprintf("test_repo_%d_%d.db", time.Now().UnixNano(), os.Getpid())
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Page{}, &models.Revision{}, &models.Attachment{}, &models.Tag{}, &models.ApiKey{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}
	t.Cleanup(func() { os.Remove(dbFile) })
	return NewWikiRepository(db)
}

func mustCreatePage(t *testing.T, repo *WikiRepository, req models.CreatePageRequest) *models.Page {
	t.Helper()
	page, err := repo.CreatePage(&req, "tester")
	if err != nil {
		t.Fatalf("failed to create page %q: %v", req.Title, err)
	}
	return page
}

// --- Pages ---

func TestCreatePage_DuplicateTitleRejected(t *testing.T) {
	repo := setupTestDB(t)
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Unik Titel", Content: "x"})

	_, err := repo.CreatePage(&models.CreatePageRequest{Title: "Unik Titel", Content: "y"}, "tester")
	if err == nil {
		t.Fatal("Expected error creating a page with a duplicate title")
	}
}

func TestCreatePage_InvalidTitleRejected(t *testing.T) {
	repo := setupTestDB(t)
	_, err := repo.CreatePage(&models.CreatePageRequest{Title: "!!!", Content: "x"}, "tester")
	if err == nil {
		t.Fatal("Expected error creating a page whose title slugifies to empty")
	}
}

func TestCreatePage_UnknownParentSlugRejected(t *testing.T) {
	repo := setupTestDB(t)
	_, err := repo.CreatePage(&models.CreatePageRequest{Title: "Barn", Content: "x", ParentSlug: "finns-inte"}, "tester")
	if err == nil {
		t.Fatal("Expected error creating a page with a nonexistent parent_slug")
	}
}

func TestCreatePage_EmptyAuthorDefaultsToAnvandare(t *testing.T) {
	repo := setupTestDB(t)
	page, err := repo.CreatePage(&models.CreatePageRequest{Title: "Anonym Skapare", Content: "x"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	revs, _ := repo.GetRevisions(page.Slug)
	if len(revs) != 1 || revs[0].Author != "Användare" {
		t.Fatalf("Expected default author 'Användare', got %+v", revs)
	}
}

func TestGetPageBySlug_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.GetPageBySlug("finns-inte", true); err == nil {
		t.Fatal("Expected error for nonexistent slug")
	}
}

func TestGetPageBySlug_PrivateRequiresAuth(t *testing.T) {
	repo := setupTestDB(t)
	isPrivate := false
	page := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Hemlig", Content: "x", IsPublic: &isPrivate})

	if _, err := repo.GetPageBySlug(page.Slug, false); err == nil {
		t.Fatal("Expected error fetching a private page while unauthenticated")
	}
	if _, err := repo.GetPageBySlug(page.Slug, true); err != nil {
		t.Fatalf("unexpected error fetching a private page while authenticated: %v", err)
	}
}

func TestGetPageBySlug_IncrementsViews(t *testing.T) {
	repo := setupTestDB(t)
	page := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Räknas", Content: "x"})

	// The view count persisted for a fetch only becomes visible on the *next*
	// read (the increment happens after the returned struct is populated),
	// so each successive call should see a strictly higher count than the last.
	first, _ := repo.GetPageBySlug(page.Slug, true)
	second, _ := repo.GetPageBySlug(page.Slug, true)
	third, _ := repo.GetPageBySlug(page.Slug, true)

	if !(first.Views < second.Views && second.Views < third.Views) {
		t.Fatalf("Expected strictly increasing views across fetches, got %d, %d, %d", first.Views, second.Views, third.Views)
	}
}

func TestGetPageBySlug_HidesPrivateChildrenWhenUnauthed(t *testing.T) {
	repo := setupTestDB(t)
	isPublic := true
	isPrivate := false
	parent := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Förälder", Content: "x", IsPublic: &isPublic})
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Privat Barn", Content: "x", IsPublic: &isPrivate, ParentSlug: parent.Slug})
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Publikt Barn", Content: "x", IsPublic: &isPublic, ParentSlug: parent.Slug})

	authed, _ := repo.GetPageBySlug(parent.Slug, true)
	if len(authed.Children) != 2 {
		t.Fatalf("Expected 2 children when authenticated, got %d", len(authed.Children))
	}

	unauthed, _ := repo.GetPageBySlug(parent.Slug, false)
	if len(unauthed.Children) != 1 || unauthed.Children[0].Title != "Publikt Barn" {
		t.Fatalf("Expected only the public child when unauthenticated, got %+v", unauthed.Children)
	}
}

func TestListPages_TagFilterAndVisibility(t *testing.T) {
	repo := setupTestDB(t)
	isPublic := true
	isPrivate := false
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Go Sida", Content: "x", IsPublic: &isPublic, Tags: []string{"go"}})
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Rust Sida", Content: "x", IsPublic: &isPublic, Tags: []string{"rust"}})
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Privat Go Sida", Content: "x", IsPublic: &isPrivate, Tags: []string{"go"}})

	goPages, err := repo.ListPages("", "go", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(goPages) != 2 {
		t.Fatalf("Expected 2 pages tagged 'go' when authenticated, got %d", len(goPages))
	}

	unauthedGoPages, _ := repo.ListPages("", "go", false)
	if len(unauthedGoPages) != 1 {
		t.Fatalf("Expected 1 public page tagged 'go' when unauthenticated, got %d", len(unauthedGoPages))
	}

	all, _ := repo.ListPages("", "", true)
	if len(all) != 3 {
		t.Fatalf("Expected 3 pages with no filters, got %d", len(all))
	}
}

func TestUpdatePage_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.UpdatePage("finns-inte", &models.UpdatePageRequest{Content: "x"}, "tester"); err == nil {
		t.Fatal("Expected error updating a nonexistent page")
	}
}

func TestUpdatePage_UnknownParentSlugRejected(t *testing.T) {
	repo := setupTestDB(t)
	page := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Sida", Content: "x"})
	unknown := "finns-inte"
	_, err := repo.UpdatePage(page.Slug, &models.UpdatePageRequest{Content: "y", ParentSlug: &unknown}, "tester")
	if err == nil {
		t.Fatal("Expected error updating a page with a nonexistent parent_slug")
	}
}

func TestUpdatePage_TagsAndVisibilityAndDetach(t *testing.T) {
	repo := setupTestDB(t)
	parent := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Bas", Content: "x"})
	child := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Kopplat Barn", Content: "x", ParentSlug: parent.Slug, Tags: []string{"gammal"}})

	isPublic := true
	updated, err := repo.UpdatePage(child.Slug, &models.UpdatePageRequest{
		Content:  "nytt innehåll",
		IsPublic: &isPublic,
		Tags:     []string{"ny", "  ", "tagg"},
	}, "tester")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.IsPublic {
		t.Fatal("Expected page to become public")
	}
	if updated.ParentID == nil {
		t.Fatal("Expected parent to remain set when ParentSlug is not provided (nil)")
	}

	withTags, _ := repo.GetPageBySlug(updated.Slug, true)
	if len(withTags.Tags) != 2 {
		t.Fatalf("Expected 2 tags after replacing (blank entries skipped), got %+v", withTags.Tags)
	}

	empty := ""
	detached, err := repo.UpdatePage(child.Slug, &models.UpdatePageRequest{Content: "x", ParentSlug: &empty}, "tester")
	if err != nil {
		t.Fatalf("unexpected error detaching parent: %v", err)
	}
	if detached.ParentID != nil {
		t.Fatalf("Expected ParentID to be nil after detaching, got %v", detached.ParentID)
	}
}

func TestUpdatePage_TitleChangeKeepsExistingSlug(t *testing.T) {
	repo := setupTestDB(t)
	page := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Original", Content: "x"})
	updated, err := repo.UpdatePage(page.Slug, &models.UpdatePageRequest{Title: "Nytt Namn", Content: "y"}, "tester")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Nytt Namn" || updated.Slug != page.Slug {
		t.Fatalf("Expected title updated but slug unchanged, got title=%q slug=%q", updated.Title, updated.Slug)
	}
}

func TestDeletePage_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	if err := repo.DeletePage("finns-inte"); err == nil {
		t.Fatal("Expected error deleting a nonexistent page")
	}
}

func TestDeletePage_PromotesChildrenToTopLevel(t *testing.T) {
	repo := setupTestDB(t)
	parent := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Tillfällig", Content: "x"})
	child := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Kvarlever", Content: "x", ParentSlug: parent.Slug})

	if err := repo.DeletePage(parent.Slug); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	survivor, err := repo.GetPageBySlug(child.Slug, true)
	if err != nil {
		t.Fatalf("Expected child to survive parent deletion: %v", err)
	}
	if survivor.ParentID != nil {
		t.Fatalf("Expected orphaned child to have nil ParentID, got %v", survivor.ParentID)
	}
}

func TestGetRevisions_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.GetRevisions("finns-inte"); err == nil {
		t.Fatal("Expected error fetching revisions for a nonexistent page")
	}
}

func TestSearchPages(t *testing.T) {
	repo := setupTestDB(t)
	isPublic := true
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Katt Guide", Content: "om katter", IsPublic: &isPublic})
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Hund Guide", Content: "om hundar", IsPublic: &isPublic})

	results, err := repo.SearchPages("katt", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Katt Guide" {
		t.Fatalf("Expected 1 result for 'katt', got %+v", results)
	}

	noResults, _ := repo.SearchPages("finnsinte", true)
	if len(noResults) != 0 {
		t.Fatalf("Expected 0 results for a non-matching query, got %d", len(noResults))
	}
}

func TestListTags(t *testing.T) {
	repo := setupTestDB(t)

	empty, err := repo.ListTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("Expected no tags initially, got %d", len(empty))
	}

	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Taggad Sida", Content: "x", Tags: []string{"go", "api"}})

	tags, err := repo.ListTags()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("Expected 2 tags, got %d: %+v", len(tags), tags)
	}
}

// --- Attachments ---

func TestSaveAttachment_PageNotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.SaveAttachment("finns-inte", "f.png", "f.png", "/uploads/f.png", "image/png", 100); err == nil {
		t.Fatal("Expected error saving an attachment to a nonexistent page")
	}
}

func TestAttachmentLifecycle(t *testing.T) {
	repo := setupTestDB(t)
	page := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Med Bilagor", Content: "x"})

	empty, err := repo.GetAttachments(page.Slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("Expected no attachments initially, got %d", len(empty))
	}

	att, err := repo.SaveAttachment(page.Slug, "unique.png", "original.png", "/uploads/unique.png", "image/png", 1234)
	if err != nil {
		t.Fatalf("unexpected error saving attachment: %v", err)
	}

	fetched, err := repo.GetAttachmentByID(att.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching attachment: %v", err)
	}
	if fetched.Original != "original.png" || fetched.PageID != page.ID {
		t.Fatalf("Fetched attachment mismatch: %+v", fetched)
	}

	list, err := repo.GetAttachments(page.Slug)
	if err != nil || len(list) != 1 {
		t.Fatalf("Expected 1 attachment listed, got %d, err=%v", len(list), err)
	}

	if err := repo.DeleteAttachment(att.ID); err != nil {
		t.Fatalf("unexpected error deleting attachment: %v", err)
	}
	if _, err := repo.GetAttachmentByID(att.ID); err == nil {
		t.Fatal("Expected error fetching a deleted attachment")
	}
}

func TestGetAttachments_PageNotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.GetAttachments("finns-inte"); err == nil {
		t.Fatal("Expected error listing attachments for a nonexistent page")
	}
}

func TestGetAttachmentByID_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.GetAttachmentByID(999); err == nil {
		t.Fatal("Expected error fetching a nonexistent attachment")
	}
}

// --- Backlinks ---

func TestGetBacklinks_TargetNotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.GetBacklinks("finns-inte", true); err == nil {
		t.Fatal("Expected error fetching backlinks for a nonexistent target page")
	}
}

func TestGetBacklinks_Empty(t *testing.T) {
	repo := setupTestDB(t)
	page := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Ensam Sida", Content: "Inga länkar hit"})
	backlinks, err := repo.GetBacklinks(page.Slug, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backlinks) != 0 {
		t.Fatalf("Expected no backlinks, got %d", len(backlinks))
	}
}

func TestGetBacklinks_HidesPrivateLinkersWhenUnauthed(t *testing.T) {
	repo := setupTestDB(t)
	isPublic := true
	isPrivate := false
	target := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Mål", Content: "x", IsPublic: &isPublic})
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Privat Länkare", Content: "Se [[Mål]]", IsPublic: &isPrivate})

	authedLinks, _ := repo.GetBacklinks(target.Slug, true)
	if len(authedLinks) != 1 {
		t.Fatalf("Expected 1 backlink when authenticated, got %d", len(authedLinks))
	}

	unauthedLinks, _ := repo.GetBacklinks(target.Slug, false)
	if len(unauthedLinks) != 0 {
		t.Fatalf("Expected 0 backlinks when unauthenticated (linker is private), got %d", len(unauthedLinks))
	}
}

// --- Revert ---

func TestRevertPageRevision_PageNotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.RevertPageRevision("finns-inte", 1, "tester"); err == nil {
		t.Fatal("Expected error reverting a nonexistent page")
	}
}

func TestRevertPageRevision_RevisionNotFoundForPage(t *testing.T) {
	repo := setupTestDB(t)
	pageA := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Sida A", Content: "x"})
	pageB := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Sida B", Content: "y"})

	revsB, _ := repo.GetRevisions(pageB.Slug)
	if len(revsB) == 0 {
		t.Fatal("Expected page B to have at least one revision")
	}

	// A revision ID that belongs to page B must be rejected for page A
	if _, err := repo.RevertPageRevision(pageA.Slug, revsB[0].ID, "tester"); err == nil {
		t.Fatal("Expected error reverting to a revision that belongs to a different page")
	}
}

func TestRevertPageRevision_Success(t *testing.T) {
	repo := setupTestDB(t)
	page := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Rollback", Content: "Original Version"})

	if _, err := repo.UpdatePage(page.Slug, &models.UpdatePageRequest{Content: "Updated Version"}, "tester"); err != nil {
		t.Fatalf("unexpected error updating page: %v", err)
	}

	revs, _ := repo.GetRevisions(page.Slug)
	if len(revs) != 2 {
		t.Fatalf("Expected 2 revisions, got %d", len(revs))
	}
	// revs is ordered created_at DESC, so the oldest (original) revision is last
	originalRev := revs[len(revs)-1]

	reverted, err := repo.RevertPageRevision(page.Slug, originalRev.ID, "tester")
	if err != nil {
		t.Fatalf("unexpected error reverting: %v", err)
	}
	if reverted.Content != "Original Version" {
		t.Fatalf("Expected content reverted to 'Original Version', got %q", reverted.Content)
	}

	revsAfter, _ := repo.GetRevisions(page.Slug)
	if len(revsAfter) != 3 {
		t.Fatalf("Expected reverting to create a 3rd revision, got %d", len(revsAfter))
	}
}

func TestResolveParentID_ViaCreateAndUpdate(t *testing.T) {
	repo := setupTestDB(t)
	grandparent := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Farfar", Content: "x"})
	parent := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Förälder Igen", Content: "x", ParentSlug: grandparent.Slug})

	// Nesting a second level deep must be rejected
	if _, err := repo.CreatePage(&models.CreatePageRequest{Title: "Barnbarn", Content: "x", ParentSlug: parent.Slug}, "tester"); err == nil {
		t.Fatal("Expected error nesting a second level deep")
	}

	pageWithChild := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Har Barn", Content: "x"})
	mustCreatePage(t, repo, models.CreatePageRequest{Title: "Dess Barn", Content: "x", ParentSlug: pageWithChild.Slug})

	otherParent := mustCreatePage(t, repo, models.CreatePageRequest{Title: "Annan Bas", Content: "x"})
	otherSlug := otherParent.Slug
	if _, err := repo.UpdatePage(pageWithChild.Slug, &models.UpdatePageRequest{Content: "x", ParentSlug: &otherSlug}, "tester"); err == nil {
		t.Fatal("Expected error giving a parent-with-children a parent of its own")
	}

	selfSlug := pageWithChild.Slug
	if _, err := repo.UpdatePage(pageWithChild.Slug, &models.UpdatePageRequest{Content: "x", ParentSlug: &selfSlug}, "tester"); err == nil {
		t.Fatal("Expected error setting a page as its own parent")
	}
}

// --- Users ---

func TestCreateUser_DefaultRole(t *testing.T) {
	repo := setupTestDB(t)
	user, err := repo.CreateUser("nyanvandare", "ny@example.com", "hemligt", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Role != "user" {
		t.Fatalf("Expected default role 'user', got %q", user.Role)
	}
}

func TestCreateUser_DuplicateUsernameRejected(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.CreateUser("dubbel", "a@example.com", "pw", "user"); err != nil {
		t.Fatalf("unexpected error creating first user: %v", err)
	}
	if _, err := repo.CreateUser("dubbel", "b@example.com", "pw", "user"); err == nil {
		t.Fatal("Expected error creating a user with a duplicate username")
	}
}

func TestGetUserByUsernameOrEmail(t *testing.T) {
	repo := setupTestDB(t)
	created, _ := repo.CreateUser("sokbar", "sokbar@example.com", "pw", "user")

	byUsername, err := repo.GetUserByUsernameOrEmail("sokbar")
	if err != nil || byUsername.ID != created.ID {
		t.Fatalf("Expected to find user by username, got %+v, err=%v", byUsername, err)
	}

	byEmail, err := repo.GetUserByUsernameOrEmail("sokbar@example.com")
	if err != nil || byEmail.ID != created.ID {
		t.Fatalf("Expected to find user by email, got %+v, err=%v", byEmail, err)
	}

	if _, err := repo.GetUserByUsernameOrEmail("finns-inte"); err == nil {
		t.Fatal("Expected error for a nonexistent identifier")
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, err := repo.GetUserByID(999); err == nil {
		t.Fatal("Expected error fetching a nonexistent user")
	}
}

func TestListUsers(t *testing.T) {
	repo := setupTestDB(t)
	empty, err := repo.ListUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("Expected no users initially, got %d", len(empty))
	}

	repo.CreateUser("anvandare1", "a1@example.com", "pw", "user")
	repo.CreateUser("anvandare2", "a2@example.com", "pw", "admin")

	users, err := repo.ListUsers()
	if err != nil || len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d, err=%v", len(users), err)
	}
}

func TestValidateUserPassword(t *testing.T) {
	repo := setupTestDB(t)
	user, _ := repo.CreateUser("pwtest", "pwtest@example.com", "korrektlosen", "user")

	if !repo.ValidateUserPassword(user, "korrektlosen") {
		t.Fatal("Expected correct password to validate")
	}
	if repo.ValidateUserPassword(user, "feltlosen") {
		t.Fatal("Expected incorrect password to fail validation")
	}
}

func TestChangePassword(t *testing.T) {
	repo := setupTestDB(t)
	user, _ := repo.CreateUser("byter", "byter@example.com", "gammaltlosen", "user")

	if err := repo.ChangePassword(999, "x", "ynewpassword"); err == nil {
		t.Fatal("Expected error changing password for a nonexistent user")
	}

	if err := repo.ChangePassword(user.ID, "feltlosen", "nyttlosenord123"); err == nil {
		t.Fatal("Expected error with wrong current password")
	}

	if err := repo.ChangePassword(user.ID, "gammaltlosen", "kort"); err == nil {
		t.Fatal("Expected error with a too-short new password")
	}

	if err := repo.ChangePassword(user.ID, "gammaltlosen", "nyttlosenord123"); err != nil {
		t.Fatalf("unexpected error on valid password change: %v", err)
	}

	updated, _ := repo.GetUserByID(user.ID)
	if updated.MustChangePassword {
		t.Fatal("Expected MustChangePassword to be cleared after a successful change")
	}
	if !repo.ValidateUserPassword(updated, "nyttlosenord123") {
		t.Fatal("Expected new password to validate after change")
	}
}

// --- API Keys ---

func TestCreateUserApiKey_UserNotFound(t *testing.T) {
	repo := setupTestDB(t)
	if _, _, err := repo.CreateUserApiKey(999, "Test Key", "never"); err == nil {
		t.Fatal("Expected error creating an API key for a nonexistent user")
	}
}

func TestCreateUserApiKey_ExpiryOptions(t *testing.T) {
	repo := setupTestDB(t)
	user, _ := repo.CreateUser("nyckelagare", "nyckel@example.com", "pw", "user")

	cases := []string{"7d", "30d", "90d", "1y", "never", ""}
	for _, opt := range cases {
		key, raw, err := repo.CreateUserApiKey(user.ID, "Key "+opt, opt)
		if err != nil {
			t.Fatalf("unexpected error for expiry option %q: %v", opt, err)
		}
		if raw == "" || key.Prefix == "" {
			t.Fatalf("Expected a raw secret and prefix for expiry option %q", opt)
		}
		if (opt == "never" || opt == "") && key.ExpiresAt != nil {
			t.Fatalf("Expected no expiry for option %q, got %v", opt, key.ExpiresAt)
		}
		if opt != "never" && opt != "" && key.ExpiresAt == nil {
			t.Fatalf("Expected an expiry date for option %q", opt)
		}
	}
}

func TestListUserApiKeys_IsolatedPerUser(t *testing.T) {
	repo := setupTestDB(t)
	userA, _ := repo.CreateUser("ownerA", "ownerA@example.com", "pw", "user")
	userB, _ := repo.CreateUser("ownerB", "ownerB@example.com", "pw", "user")

	repo.CreateUserApiKey(userA.ID, "A1", "never")
	repo.CreateUserApiKey(userA.ID, "A2", "never")
	repo.CreateUserApiKey(userB.ID, "B1", "never")

	keysA, err := repo.ListUserApiKeys(userA.ID)
	if err != nil || len(keysA) != 2 {
		t.Fatalf("Expected 2 keys for user A, got %d, err=%v", len(keysA), err)
	}

	keysB, err := repo.ListUserApiKeys(userB.ID)
	if err != nil || len(keysB) != 1 {
		t.Fatalf("Expected 1 key for user B, got %d, err=%v", len(keysB), err)
	}
}

func TestRevokeUserApiKey_OnlyOwnerCanRevoke(t *testing.T) {
	repo := setupTestDB(t)
	userA, _ := repo.CreateUser("revokeA", "revokeA@example.com", "pw", "user")
	userB, _ := repo.CreateUser("revokeB", "revokeB@example.com", "pw", "user")

	key, raw, _ := repo.CreateUserApiKey(userA.ID, "A Key", "never")

	// userB attempting to revoke userA's key must not affect it
	repo.RevokeUserApiKey(userB.ID, key.ID)
	if _, _, err := repo.ValidateApiKey(raw); err != nil {
		t.Fatalf("Expected key to remain valid after a non-owner revoke attempt, got err=%v", err)
	}

	if err := repo.RevokeUserApiKey(userA.ID, key.ID); err != nil {
		t.Fatalf("unexpected error revoking own key: %v", err)
	}
	if _, _, err := repo.ValidateApiKey(raw); err == nil {
		t.Fatal("Expected revoked key to no longer validate")
	}
}

func TestValidateApiKey_UnknownKeyRejected(t *testing.T) {
	repo := setupTestDB(t)
	if _, _, err := repo.ValidateApiKey("ptc_key_does_not_exist"); err == nil {
		t.Fatal("Expected error validating an unknown API key")
	}
}

func TestValidateApiKey_ExpiredKeyRejected(t *testing.T) {
	repo := setupTestDB(t)
	user, _ := repo.CreateUser("expiretest", "expiretest@example.com", "pw", "user")
	key, raw, _ := repo.CreateUserApiKey(user.ID, "Expiring Key", "never")

	// Force the key into the past directly, since CreateUserApiKey only offers future expiry presets
	past := time.Now().Add(-1 * time.Hour)
	repo.db.Model(&models.ApiKey{}).Where("id = ?", key.ID).Update("expires_at", past)

	if _, _, err := repo.ValidateApiKey(raw); err == nil {
		t.Fatal("Expected an expired API key to be rejected")
	}
}

func TestValidateApiKey_UserGoneAfterKeyIssued(t *testing.T) {
	repo := setupTestDB(t)
	user, _ := repo.CreateUser("forsvinner", "forsvinner@example.com", "pw", "user")
	_, raw, _ := repo.CreateUserApiKey(user.ID, "Orphaned Key", "never")

	repo.db.Unscoped().Delete(&models.User{}, user.ID)

	if _, _, err := repo.ValidateApiKey(raw); err == nil {
		t.Fatal("Expected error validating a key whose owning user no longer exists")
	}
}

func TestValidateApiKey_UpdatesLastUsedAt(t *testing.T) {
	repo := setupTestDB(t)
	user, _ := repo.CreateUser("lastused", "lastused@example.com", "pw", "user")
	_, raw, _ := repo.CreateUserApiKey(user.ID, "Key", "never")

	_, apiKey, err := repo.ValidateApiKey(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiKey.LastUsedAt == nil {
		t.Fatal("Expected LastUsedAt to be set after a successful validation")
	}
}
