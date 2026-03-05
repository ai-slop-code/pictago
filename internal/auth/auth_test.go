package auth

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStore_CreateUser_ValidateUser_GetUserID(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	if err := s.CreateUser(ctx, "alice", "secret123"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	valid, err := s.ValidateUser(ctx, "alice", "secret123")
	if err != nil {
		t.Fatalf("ValidateUser: %v", err)
	}
	if !valid {
		t.Error("ValidateUser: expected true for correct password")
	}
	valid, _ = s.ValidateUser(ctx, "alice", "wrong")
	if valid {
		t.Error("ValidateUser: expected false for wrong password")
	}
	id, err := s.GetUserID(ctx, "alice")
	if err != nil || id < 1 {
		t.Fatalf("GetUserID: %v, id=%d", err, id)
	}
	name, err := s.GetUsernameByID(ctx, id)
	if err != nil || name != "alice" {
		t.Fatalf("GetUsernameByID: %v, name=%q", err, name)
	}
}

func TestStore_CreateUser_duplicate(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CreateUser(ctx, "bob", "pass"); err != nil {
		t.Fatal(err)
	}
	err = s.CreateUser(ctx, "bob", "other")
	if err == nil {
		t.Error("CreateUser(duplicate) should fail")
	}
}
