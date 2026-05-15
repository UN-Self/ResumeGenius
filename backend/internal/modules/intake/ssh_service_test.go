package intake

import (
	"testing"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/models"
)

func TestSSHKeyService_Create(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	key, err := svc.Create("user1", "my-deploy-key", "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if key.Alias != "my-deploy-key" {
		t.Fatalf("alias = %q, want %q", key.Alias, "my-deploy-key")
	}
}

func TestSSHKeyService_List(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	svc.Create("user1", "key1", "key-content-1")
	svc.Create("user1", "key2", "key-content-2")
	svc.Create("user2", "key3", "key-content-3")

	keys, err := svc.List("user1")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len = %d, want 2", len(keys))
	}
}

func TestSSHKeyService_GetDecryptedKey(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	original := "-----BEGIN OPENSSH PRIVATE KEY-----\ntest-key\n-----END OPENSSH PRIVATE KEY-----"
	created, _ := svc.Create("user1", "key1", original)

	decrypted, err := svc.GetDecryptedKey("user1", created.ID)
	if err != nil {
		t.Fatalf("GetDecryptedKey failed: %v", err)
	}
	if decrypted != original {
		t.Fatalf("decrypted = %q, want %q", decrypted, original)
	}
}

func TestSSHKeyService_Delete(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	created, _ := svc.Create("user1", "key1", "key-content")

	if err := svc.Delete("user1", created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	keys, _ := svc.List("user1")
	if len(keys) != 0 {
		t.Fatalf("len = %d, want 0", len(keys))
	}
}

func TestSSHKeyService_DeleteReferencedByAsset(t *testing.T) {
	db := SetupTestDB(t)
	svc := NewSSHKeyService(db, "0123456789abcdef0123456789abcdef")

	projSvc := NewProjectService(db)
	proj, err := projSvc.Create("user1", "test-project")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	created, _ := svc.Create("user1", "deploy-key", "key-content")

	// create an asset referencing this key
	uri := "git@github.com:test/repo.git"
	db.Model(&models.Asset{}).Create(&models.Asset{
		ProjectID: proj.ID,
		Type:      "git_repo",
		URI:       &uri,
		KeyID:     &created.ID,
	})

	err = svc.Delete("user1", created.ID)
	if err == nil {
		t.Fatal("expected error when assets reference the key, got nil")
	}
}
