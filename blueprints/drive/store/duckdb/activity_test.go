package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// ============================================================
// Activity CRUD Tests
// ============================================================

func TestCreateActivity_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	activity := newTestActivity("act1", "user1", "upload", "file", "file1")

	if err := store.CreateActivity(ctx, activity); err != nil {
		t.Fatalf("create activity failed: %v", err)
	}

	// Verify by listing
	activities, err := store.ListActivitiesByUser(ctx, "user1", 10)
	if err != nil {
		t.Fatalf("list activities failed: %v", err)
	}
	if len(activities) != 1 {
		t.Errorf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].Action != "upload" {
		t.Errorf("expected action upload, got %s", activities[0].Action)
	}
}

func TestCreateActivity_AllFields(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	activity := &Activity{
		ID:           "act1",
		UserID:       "user1",
		Action:       "download",
		ResourceType: "file",
		ResourceID:   "file1",
		ResourceName: sql.NullString{String: "important.pdf", Valid: true},
		Details:      sql.NullString{String: `{"version": 2}`, Valid: true},
		IPAddress:    sql.NullString{String: "192.168.1.100", Valid: true},
		UserAgent:    sql.NullString{String: "Mozilla/5.0", Valid: true},
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}

	if err := store.CreateActivity(ctx, activity); err != nil {
		t.Fatalf("create activity failed: %v", err)
	}

	activities, _ := store.ListActivitiesByUser(ctx, "user1", 10)
	if len(activities) != 1 {
		t.Fatal("expected 1 activity")
	}

	got := activities[0]
	if got.ResourceName.String != "important.pdf" {
		t.Errorf("expected resource_name important.pdf, got %s", got.ResourceName.String)
	}
	if got.Details.String != `{"version": 2}` {
		t.Errorf("expected details, got %s", got.Details.String)
	}
	if got.IPAddress.String != "192.168.1.100" {
		t.Errorf("expected ip_address 192.168.1.100, got %s", got.IPAddress.String)
	}
}

// ============================================================
// Activity Listing Tests
// ============================================================

func TestListActivitiesByUser(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := newTestUser("user1", "alice@example.com")
	user2 := newTestUser("user2", "bob@example.com")
	store.CreateUser(ctx, user1)
	store.CreateUser(ctx, user2)

	// Create activities for both users
	for i := 1; i <= 3; i++ {
		act := newTestActivity("act1_"+string(rune('0'+i)), "user1", "action", "file", "file1")
		store.CreateActivity(ctx, act)
	}
	for i := 1; i <= 2; i++ {
		act := newTestActivity("act2_"+string(rune('0'+i)), "user2", "action", "file", "file2")
		store.CreateActivity(ctx, act)
	}

	user1Acts, _ := store.ListActivitiesByUser(ctx, "user1", 10)
	if len(user1Acts) != 3 {
		t.Errorf("expected 3 activities for user1, got %d", len(user1Acts))
	}

	user2Acts, _ := store.ListActivitiesByUser(ctx, "user2", 10)
	if len(user2Acts) != 2 {
		t.Errorf("expected 2 activities for user2, got %d", len(user2Acts))
	}
}

func TestListActivitiesByUser_Limit(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	for i := 1; i <= 20; i++ {
		act := newTestActivity("act"+string(rune('0'+i/10))+string(rune('0'+i%10)), "user1", "action", "file", "file1")
		store.CreateActivity(ctx, act)
	}

	activities, _ := store.ListActivitiesByUser(ctx, "user1", 5)
	if len(activities) != 5 {
		t.Errorf("expected 5 activities with limit, got %d", len(activities))
	}
}

func TestListActivitiesForResource(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create activities for different resources
	act1 := newTestActivity("act1", "user1", "view", "file", "file1")
	act2 := newTestActivity("act2", "user1", "edit", "file", "file1")
	act3 := newTestActivity("act3", "user1", "view", "file", "file2")

	store.CreateActivity(ctx, act1)
	store.CreateActivity(ctx, act2)
	store.CreateActivity(ctx, act3)

	activities, err := store.ListActivitiesForResource(ctx, "file", "file1", 10)
	if err != nil {
		t.Fatalf("list activities for resource failed: %v", err)
	}
	if len(activities) != 2 {
		t.Errorf("expected 2 activities for file1, got %d", len(activities))
	}
}

func TestListRecentActivities(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := newTestUser("user1", "alice@example.com")
	user2 := newTestUser("user2", "bob@example.com")
	store.CreateUser(ctx, user1)
	store.CreateUser(ctx, user2)

	// Create activities with different timestamps
	act1 := newTestActivity("act1", "user1", "upload", "file", "file1")
	act1.CreatedAt = time.Now().Add(-2 * time.Hour).Truncate(time.Microsecond)

	act2 := newTestActivity("act2", "user2", "download", "file", "file2")
	act2.CreatedAt = time.Now().Add(-1 * time.Hour).Truncate(time.Microsecond)

	act3 := newTestActivity("act3", "user1", "share", "file", "file1")
	act3.CreatedAt = time.Now().Truncate(time.Microsecond)

	store.CreateActivity(ctx, act1)
	store.CreateActivity(ctx, act2)
	store.CreateActivity(ctx, act3)

	activities, err := store.ListRecentActivities(ctx, 10)
	if err != nil {
		t.Fatalf("list recent activities failed: %v", err)
	}
	if len(activities) != 3 {
		t.Fatalf("expected 3 activities, got %d", len(activities))
	}

	// Should be ordered by created_at DESC (most recent first)
	if activities[0].Action != "share" {
		t.Errorf("expected most recent action to be share, got %s", activities[0].Action)
	}
}

// ============================================================
// Activity Cleanup Tests
// ============================================================

func TestDeleteActivitiesForResource(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create activities for a resource
	for i := 1; i <= 5; i++ {
		act := newTestActivity("act"+string(rune('0'+i)), "user1", "action", "file", "file1")
		store.CreateActivity(ctx, act)
	}

	// Create activities for another resource
	act := newTestActivity("act_other", "user1", "action", "file", "file2")
	store.CreateActivity(ctx, act)

	if err := store.DeleteActivitiesForResource(ctx, "file", "file1"); err != nil {
		t.Fatalf("delete activities for resource failed: %v", err)
	}

	// file1 activities should be deleted
	file1Acts, _ := store.ListActivitiesForResource(ctx, "file", "file1", 10)
	if len(file1Acts) != 0 {
		t.Errorf("expected 0 activities for file1, got %d", len(file1Acts))
	}

	// file2 activities should remain
	file2Acts, _ := store.ListActivitiesForResource(ctx, "file", "file2", 10)
	if len(file2Acts) != 1 {
		t.Errorf("expected 1 activity for file2, got %d", len(file2Acts))
	}
}

func TestDeleteOldActivities(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create old activities
	oldAct := newTestActivity("old1", "user1", "action", "file", "file1")
	oldAct.CreatedAt = time.Now().Add(-90 * 24 * time.Hour).Truncate(time.Microsecond) // 90 days ago
	store.CreateActivity(ctx, oldAct)

	oldAct2 := newTestActivity("old2", "user1", "action", "file", "file2")
	oldAct2.CreatedAt = time.Now().Add(-60 * 24 * time.Hour).Truncate(time.Microsecond) // 60 days ago
	store.CreateActivity(ctx, oldAct2)

	// Create recent activity
	recentAct := newTestActivity("recent", "user1", "action", "file", "file3")
	recentAct.CreatedAt = time.Now().Truncate(time.Microsecond)
	store.CreateActivity(ctx, recentAct)

	// Delete activities older than 30 days
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	if err := store.DeleteOldActivities(ctx, cutoff); err != nil {
		t.Fatalf("delete old activities failed: %v", err)
	}

	activities, _ := store.ListActivitiesByUser(ctx, "user1", 10)
	if len(activities) != 1 {
		t.Errorf("expected 1 activity after cleanup, got %d", len(activities))
	}
	if activities[0].ID != "recent" {
		t.Errorf("expected recent activity to remain, got %s", activities[0].ID)
	}
}

// ============================================================
// Activity Analytics Tests
// ============================================================

func TestCountActivitiesByAction(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create activities with different actions
	actions := []string{"upload", "upload", "upload", "download", "download", "share"}
	for i, action := range actions {
		act := newTestActivity("act"+string(rune('0'+i)), "user1", action, "file", "file"+string(rune('0'+i)))
		store.CreateActivity(ctx, act)
	}

	counts, err := store.CountActivitiesByAction(ctx, "user1")
	if err != nil {
		t.Fatalf("count activities by action failed: %v", err)
	}

	if counts["upload"] != 3 {
		t.Errorf("expected 3 uploads, got %d", counts["upload"])
	}
	if counts["download"] != 2 {
		t.Errorf("expected 2 downloads, got %d", counts["download"])
	}
	if counts["share"] != 1 {
		t.Errorf("expected 1 share, got %d", counts["share"])
	}
}

// ============================================================
// Business Use Cases - Audit Trail
// ============================================================

func TestAuditTrail_FileUpload(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "report.pdf")
	store.CreateFile(ctx, file)

	// Log the upload action
	activity := &Activity{
		ID:           "act1",
		UserID:       "user1",
		Action:       "file.upload",
		ResourceType: "file",
		ResourceID:   "file1",
		ResourceName: sql.NullString{String: "report.pdf", Valid: true},
		Details:      sql.NullString{String: `{"size": 1024, "mime_type": "application/pdf"}`, Valid: true},
		IPAddress:    sql.NullString{String: "10.0.0.1", Valid: true},
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	store.CreateActivity(ctx, activity)

	// Verify audit trail
	activities, _ := store.ListActivitiesForResource(ctx, "file", "file1", 10)
	if len(activities) != 1 {
		t.Fatal("expected upload activity in audit trail")
	}
	if activities[0].Action != "file.upload" {
		t.Errorf("expected file.upload action, got %s", activities[0].Action)
	}
}

func TestAuditTrail_FileDownload(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "report.pdf")
	store.CreateFile(ctx, file)

	// Log multiple downloads
	for i := 1; i <= 3; i++ {
		activity := &Activity{
			ID:           "download_" + string(rune('0'+i)),
			UserID:       "user1",
			Action:       "file.download",
			ResourceType: "file",
			ResourceID:   "file1",
			IPAddress:    sql.NullString{String: "10.0.0." + string(rune('0'+i)), Valid: true},
			CreatedAt:    time.Now().Add(time.Duration(i) * time.Hour).Truncate(time.Microsecond),
		}
		store.CreateActivity(ctx, activity)
	}

	activities, _ := store.ListActivitiesForResource(ctx, "file", "file1", 10)
	if len(activities) != 3 {
		t.Errorf("expected 3 download activities, got %d", len(activities))
	}
}

func TestAuditTrail_FileShare(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	alice := newTestUser("alice", "alice@example.com")
	bob := newTestUser("bob", "bob@example.com")
	store.CreateUser(ctx, alice)
	store.CreateUser(ctx, bob)

	file := newTestFile("file1", "alice", "document.docx")
	store.CreateFile(ctx, file)

	// Log the share action
	activity := &Activity{
		ID:           "act1",
		UserID:       "alice",
		Action:       "file.share",
		ResourceType: "file",
		ResourceID:   "file1",
		Details:      sql.NullString{String: `{"shared_with": "bob", "permission": "editor"}`, Valid: true},
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	store.CreateActivity(ctx, activity)

	activities, _ := store.ListActivitiesForResource(ctx, "file", "file1", 10)
	if len(activities) != 1 {
		t.Fatal("expected share activity")
	}
	if activities[0].Details.String != `{"shared_with": "bob", "permission": "editor"}` {
		t.Error("expected share details in activity")
	}
}

func TestAuditTrail_FileDelete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "old_file.txt")
	store.CreateFile(ctx, file)

	// Log the delete action (before actual deletion)
	activity := &Activity{
		ID:           "act1",
		UserID:       "user1",
		Action:       "file.delete",
		ResourceType: "file",
		ResourceID:   "file1",
		ResourceName: sql.NullString{String: "old_file.txt", Valid: true},
		Details:      sql.NullString{String: `{"permanent": true}`, Valid: true},
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	store.CreateActivity(ctx, activity)

	// Delete the file
	store.DeleteFile(ctx, "file1")

	// Activity should still exist for audit purposes
	userActs, _ := store.ListActivitiesByUser(ctx, "user1", 10)
	if len(userActs) != 1 {
		t.Fatal("delete activity should persist after file deletion")
	}
}

func TestAuditTrail_UserLogin(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	// Log login activity
	activity := &Activity{
		ID:           "login1",
		UserID:       "user1",
		Action:       "user.login",
		ResourceType: "session",
		ResourceID:   "sess1",
		IPAddress:    sql.NullString{String: "192.168.1.100", Valid: true},
		UserAgent:    sql.NullString{String: "Chrome/120.0", Valid: true},
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	store.CreateActivity(ctx, activity)

	activities, _ := store.ListActivitiesByUser(ctx, "user1", 10)
	if len(activities) != 1 {
		t.Fatal("expected login activity")
	}
	if activities[0].Action != "user.login" {
		t.Errorf("expected user.login action, got %s", activities[0].Action)
	}
}

func TestAuditTrail_SettingsChange(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	activity := &Activity{
		ID:           "act1",
		UserID:       "user1",
		Action:       "settings.update",
		ResourceType: "settings",
		ResourceID:   "user1",
		Details:      sql.NullString{String: `{"changed": ["theme", "notifications"]}`, Valid: true},
		CreatedAt:    time.Now().Truncate(time.Microsecond),
	}
	store.CreateActivity(ctx, activity)

	activities, _ := store.ListActivitiesByUser(ctx, "user1", 10)
	if len(activities) != 1 {
		t.Fatal("expected settings change activity")
	}
}

func TestAuditTrail_RetentionPolicy(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	// Create activities at different ages
	ages := []int{7, 30, 60, 90, 180} // days old
	for i, daysOld := range ages {
		act := &Activity{
			ID:           "act" + string(rune('0'+i)),
			UserID:       "user1",
			Action:       "action",
			ResourceType: "file",
			ResourceID:   "file" + string(rune('0'+i)),
			CreatedAt:    time.Now().Add(-time.Duration(daysOld) * 24 * time.Hour).Truncate(time.Microsecond),
		}
		store.CreateActivity(ctx, act)
	}

	// Initial count
	before, _ := store.ListActivitiesByUser(ctx, "user1", 100)
	if len(before) != 5 {
		t.Fatalf("expected 5 activities initially, got %d", len(before))
	}

	// Apply 90-day retention policy
	cutoff := time.Now().Add(-90 * 24 * time.Hour)
	store.DeleteOldActivities(ctx, cutoff)

	after, _ := store.ListActivitiesByUser(ctx, "user1", 100)
	// Should keep: 7, 30, 60 days (3 activities)
	// Should delete: 90, 180 days (2 activities)
	if len(after) != 3 {
		t.Errorf("expected 3 activities after retention policy, got %d", len(after))
	}
}
