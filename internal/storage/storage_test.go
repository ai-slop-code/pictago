package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_SaveInCollection_Delete_Root(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Root() != dir {
		t.Errorf("Root() = %q, want %q", s.Root(), dir)
	}

	rel, err := s.SaveInCollection(1, "default", "test.jpg", []byte("image data"))
	if err != nil {
		t.Fatalf("SaveInCollection: %v", err)
	}
	if rel != "1/default/test.jpg" {
		t.Errorf("SaveInCollection returned %q, want 1/default/test.jpg", rel)
	}
	full := filepath.Join(dir, rel)
	if _, err := os.Stat(full); os.IsNotExist(err) {
		t.Errorf("file was not created at %s", full)
	}

	if err := s.Delete(rel); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(full); err == nil {
		t.Error("file still exists after Delete")
	}
}

func TestStore_SaveInCollection_safePaths(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Path traversal in filename should be sanitized (filepath.Base + safe chars)
	rel, err := s.SaveInCollection(1, "col", "../../../evil.jpg", []byte("x"))
	if err != nil {
		t.Fatalf("SaveInCollection: %v", err)
	}
	// Should be under 1/col/, not outside dir
	if rel != "1/col/evil.jpg" {
		t.Errorf("got %q", rel)
	}
	full := filepath.Join(dir, rel)
	if _, err := os.Stat(full); os.IsNotExist(err) {
		t.Errorf("file not at %s", full)
	}
	// Ensure nothing was written outside dir
	evilPath := filepath.Join(dir, "..", "evil.jpg")
	if _, err := os.Stat(evilPath); err == nil {
		t.Error("file escaped to parent dir")
	}
}

func TestStore_Delete_rejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	err := s.Delete("1/../../../etc/passwd")
	if err != os.ErrInvalid {
		t.Errorf("Delete(traversal) = %v, want ErrInvalid", err)
	}
}

func TestStore_DeleteDirectory(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.SaveInCollection(1, "col", "a.jpg", []byte("a"))
	s.SaveInCollection(1, "col", "b.jpg", []byte("b"))
	relDir := "1/col"
	if err := s.DeleteDirectory(relDir); err != nil {
		t.Fatalf("DeleteDirectory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, relDir)); err == nil {
		t.Error("directory still exists after DeleteDirectory")
	}
}
