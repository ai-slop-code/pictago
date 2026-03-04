// Package storage provides filesystem-backed file storage with safe filenames and collection paths.
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Store is the interface for file storage. *store implements it.
type Store interface {
	SaveInCollection(userID int64, collection, filename string, data []byte) (relativePath string, err error)
	Delete(relativePath string) error
	DeleteDirectory(relativeDir string) error
	Root() string
}

type store struct {
	root string
}

func New(root string) (Store, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	return &store{root: root}, nil
}

// safeFilename strips path components, null bytes, and restricts to printable ASCII suitable for URLs.
func safeFilename(filename string) string {
	name := filepath.Base(filename)
	name = strings.TrimSpace(name)
	// Remove null bytes and other control characters
	var b strings.Builder
	for _, r := range name {
		if r == 0 || r == '\uFFFD' || unicode.IsControl(r) {
			continue
		}
		if r < 0x20 || r > 0x7E {
			continue
		}
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			continue
		default:
			b.WriteRune(r)
		}
	}
	name = b.String()
	if name == "" || name == "." {
		return "file"
	}
	return name
}

// Save writes the reader to root/filename and returns the public path (e.g. /files/abc.jpg).
func (s *store) Save(filename string, r io.Reader) (path string, err error) {
	name := safeFilename(filename)
	if name == "" {
		name = "file"
	}
	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".bin"
	}
	base := name[:len(name)-len(ext)]
	// avoid overwrites: if file exists, add a number
	for i := 0; ; i++ {
		tryName := name
		if i > 0 {
			tryName = fmt.Sprintf("%s_%d%s", base, i, ext)
		}
		full := filepath.Join(s.root, tryName)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			f, err := os.Create(full)
			if err != nil {
				return "", err
			}
			_, err = io.Copy(f, r)
			f.Close()
			if err != nil {
				os.Remove(full)
				return "", err
			}
			return "/files/" + tryName, nil
		}
	}
}

// SaveBytes is a convenience for saving a byte slice (e.g. after optimization).
func (s *store) SaveBytes(filename string, data []byte) (path string, err error) {
	return s.Save(filename, &byteReader{data: data})
}

// safeCollectionName restricts to alphanumeric, dash, underscore; default "default".
func safeCollectionName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "default"
	}
	return out
}

// SaveInCollection saves data under root/userID/collection/filename and returns the relative path (e.g. "1/my-collection/foo.jpg").
func (s *store) SaveInCollection(userID int64, collection, filename string, data []byte) (relativePath string, err error) {
	col := safeCollectionName(collection)
	name := safeFilename(filename)
	if name == "" {
		name = "file"
	}
	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".bin"
	}
	base := name[:len(name)-len(ext)]
	dir := filepath.Join(s.root, fmt.Sprintf("%d", userID), col)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	for i := 0; ; i++ {
		tryName := name
		if i > 0 {
			tryName = fmt.Sprintf("%s_%d%s", base, i, ext)
		}
		full := filepath.Join(dir, tryName)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			if err := os.WriteFile(full, data, 0644); err != nil {
				return "", err
			}
			relativePath = fmt.Sprintf("%d/%s/%s", userID, col, tryName)
			return relativePath, nil
		}
	}
}

type byteReader struct {
	data []byte
	pos  int
}

func (b *byteReader) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

// List returns all filenames in the store (for building direct links).
func (s *store) List() ([]string, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

// Root returns the filesystem root path (for serving static files).
func (s *store) Root() string {
	return s.root
}

// Delete removes the file at relativePath (e.g. "userID/collection/filename") from the store.
func (s *store) Delete(relativePath string) error {
	if relativePath == "" || strings.Contains(relativePath, "..") {
		return os.ErrInvalid
	}
	full := filepath.Join(s.root, filepath.Clean(relativePath))
	return os.Remove(full)
}

// DeleteDirectory removes the directory at relativeDir (e.g. "userID/collection") and all its contents.
func (s *store) DeleteDirectory(relativeDir string) error {
	if relativeDir == "" || strings.Contains(relativeDir, "..") {
		return os.ErrInvalid
	}
	full := filepath.Join(s.root, filepath.Clean(relativeDir))
	return os.RemoveAll(full)
}
