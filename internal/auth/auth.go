// Package auth provides SQLite-backed user storage, sessions, API keys, and upload metadata.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite"
)

const cost = bcrypt.DefaultCost

const (
	sessionCookieName = "ih_session"
	sessionTTL        = 7 * 24 * time.Hour
	apiKeyPrefix      = "ih_"
)

type contextKey string

const ContextKeyUsername contextKey = "username"

type Store struct {
	db *sql.DB
}

type User struct {
	ID        int64  `json:"id,string"` // string in JSON so JS doesn't lose precision (IDs can exceed Number.MAX_SAFE_INTEGER)
	Username  string `json:"username"`
	FileCount int64  `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}

type UploadRecord struct {
	ID         int64  `json:"id"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Collection string `json:"collection"`
	Filename   string `json:"filename"`
}

type CollectionStats struct {
	Name      string `json:"name"`
	FileCount int64  `json:"file_count"`
	TotalSize int64  `json:"total_size"`
}

type UserStats struct {
	FileCount       int64             `json:"file_count"`
	TotalSize       int64             `json:"total_size"`
	CollectionCount int64             `json:"collection_count"`
	Collections     []CollectionStats `json:"collections"`
}

type APIKey struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	CreatedAt string `json:"created_at"`
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	ctx := context.Background()
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			filename TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE uploads ADD COLUMN path TEXT`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE uploads ADD COLUMN collection TEXT`)

	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id),
			expires_at DATETIME NOT NULL
		);
	`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			name TEXT NOT NULL DEFAULT '',
			key_hash TEXT NOT NULL,
			key_prefix TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}
	if err := s.migrateRandomizeUserID1(ctx); err != nil {
		return err
	}
	return nil
}

// migrateRandomizeUserID1 reassigns user id 1 to a random ID so the first user is not predictable.
func (s *Store) migrateRandomizeUserID1(ctx context.Context) error {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = 1 LIMIT 1`).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	newID, err := randomUserID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE uploads SET user_id = ? WHERE user_id = 1`, newID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET user_id = ? WHERE user_id = 1`, newID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE api_keys SET user_id = ? WHERE user_id = 1`, newID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET id = ? WHERE id = 1`, newID); err != nil {
		return err
	}
	return tx.Commit()
}

// randomUserID returns a positive 63-bit random int64 (so it fits in a signed DB integer and avoids zero).
func randomUserID() (int64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	u := binary.BigEndian.Uint64(b[:])
	u &= (1 << 63) - 1
	if u == 0 {
		u = 1
	}
	return int64(u), nil
}

func (s *Store) CreateUser(ctx context.Context, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		id, err := randomUserID()
		if err != nil {
			return err
		}
		_, err = s.db.ExecContext(ctx, `INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)`, id, username, string(hash))
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "unique constraint") {
			continue
		}
		return err
	}
	return sql.ErrNoRows
}

func (s *Store) ValidateUser(ctx context.Context, username, password string) (bool, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE username = ?`, username).Scan(&hash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) GetUserID(ctx context.Context, username string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE username = ?`, username).Scan(&id)
	return id, err
}

func collectionFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "default"
}

func (s *Store) RecordUpload(ctx context.Context, userID int64, path string, sizeBytes int64) error {
	filename := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		filename = path[idx+1:]
	}
	col := collectionFromPath(path)
	_, err := s.db.ExecContext(ctx, `INSERT INTO uploads (user_id, filename, path, collection, size_bytes) VALUES (?, ?, ?, ?, ?)`, userID, filename, path, col, sizeBytes)
	return err
}

func (s *Store) ListUploads(ctx context.Context, userID int64) ([]UploadRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(path, filename) AS path, size_bytes, COALESCE(collection, '') FROM uploads WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]UploadRecord, 0)
	for rows.Next() {
		var id int64
		var p, col string
		var sz int64
		if err := rows.Scan(&id, &p, &sz, &col); err != nil {
			return nil, err
		}
		if col == "" {
			col = collectionFromPath(p)
		}
		filename := p
		if idx := strings.LastIndex(p, "/"); idx >= 0 {
			filename = p[idx+1:]
		}
		out = append(out, UploadRecord{ID: id, Path: "/files/" + p, Size: sz, Collection: col, Filename: filename})
	}
	return out, rows.Err()
}

func (s *Store) GetStats(ctx context.Context, userID int64) (UserStats, error) {
	var st UserStats
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(size_bytes), 0) FROM uploads WHERE user_id = ?`, userID).Scan(&st.FileCount, &st.TotalSize)
	if err != nil {
		return st, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT COALESCE(NULLIF(TRIM(collection), ''), 'default') AS col, COUNT(*), COALESCE(SUM(size_bytes), 0) FROM uploads WHERE user_id = ? GROUP BY 1 ORDER BY 1`, userID)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	for rows.Next() {
		var c CollectionStats
		if err := rows.Scan(&c.Name, &c.FileCount, &c.TotalSize); err != nil {
			return st, err
		}
		st.Collections = append(st.Collections, c)
	}
	if err := rows.Err(); err != nil {
		return st, err
	}
	st.CollectionCount = int64(len(st.Collections))
	return st, nil
}

// DeleteCollection deletes all uploads for the given user and collection (normalized:
// trim, empty becomes "default"). It returns the relative paths of deleted files so the
// caller can remove them from the file store.
func (s *Store) DeleteCollection(ctx context.Context, userID int64, collectionName string) (relativePaths []string, err error) {
	col := strings.TrimSpace(collectionName)
	if col == "" {
		col = "default"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT path FROM uploads WHERE user_id = ? AND (COALESCE(NULLIF(TRIM(collection), ''), 'default') = ?)`, userID, col)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		if p != "" {
			paths = append(paths, p)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM uploads WHERE user_id = ? AND (COALESCE(NULLIF(TRIM(collection), ''), 'default') = ?)`, userID, col)
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func (s *Store) ListUsersWithStats(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, COALESCE(COUNT(up.id), 0), COALESCE(SUM(up.size_bytes), 0)
		FROM users u LEFT JOIN uploads up ON u.id = up.user_id
		GROUP BY u.id ORDER BY u.username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.FileCount, &u.TotalSize); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM uploads WHERE user_id = ?`, id); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, id); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE user_id = ?`, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) UpdatePassword(ctx context.Context, username, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), cost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE username = ?`, string(hash), username)
	return err
}

func (s *Store) DeleteUpload(ctx context.Context, uploadID, userID int64) (relativePath string, err error) {
	var path string
	err = s.db.QueryRowContext(ctx, `SELECT path FROM uploads WHERE id = ? AND user_id = ?`, uploadID, userID).Scan(&path)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", sql.ErrNoRows
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM uploads WHERE id = ? AND user_id = ?`, uploadID, userID)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (s *Store) GetUsernameByID(ctx context.Context, userID int64) (string, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT username FROM users WHERE id = ?`, userID).Scan(&name)
	return name, err
}

func (s *Store) CreateSession(ctx context.Context, userID int64) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = base64.URLEncoding.EncodeToString(b)
	token = strings.TrimRight(token, "=")
	expires := time.Now().Add(sessionTTL)
	_, err = s.db.ExecContext(ctx, `INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`, token, userID, expires.UTC().Format(time.RFC3339))
	return token, err
}

func (s *Store) LookupSession(ctx context.Context, token string) (userID int64, err error) {
	var exp string
	err = s.db.QueryRowContext(ctx, `SELECT user_id, expires_at FROM sessions WHERE token = ?`, token).Scan(&userID, &exp)
	if err != nil {
		return 0, err
	}
	t, _ := time.Parse(time.RFC3339, exp)
	if time.Now().After(t) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
		return 0, sql.ErrNoRows
	}
	return userID, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (s *Store) CreateAPIKey(ctx context.Context, userID int64, name string) (key string, err error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw := apiKeyPrefix + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	keyHash := hex.EncodeToString(sum[:])
	prefix := raw
	if len(prefix) > 12 {
		prefix = prefix[:12] + "…"
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO api_keys (user_id, name, key_hash, key_prefix) VALUES (?, ?, ?, ?)`, userID, name, keyHash, prefix)
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (s *Store) ValidateAPIKey(ctx context.Context, rawKey string) (userID int64, err error) {
	if !strings.HasPrefix(rawKey, apiKeyPrefix) {
		return 0, sql.ErrNoRows
	}
	sum := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(sum[:])
	err = s.db.QueryRowContext(ctx, `SELECT user_id FROM api_keys WHERE key_hash = ?`, keyHash).Scan(&userID)
	return userID, err
}

func (s *Store) ListAPIKeys(ctx context.Context, userID int64) ([]APIKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, key_prefix, created_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]APIKey, 0)
	for rows.Next() {
		var k APIKey
		var createdAt string
		if err := rows.Scan(&k.ID, &k.Name, &k.Prefix, &createdAt); err != nil {
			return nil, err
		}
		k.CreatedAt = createdAt
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID, userID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ? AND user_id = ?`, keyID, userID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Middleware checks session cookie or API key; sets username in context or returns 401.
func Middleware(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var userID int64
			var err error

			if cookie, _ := r.Cookie(sessionCookieName); cookie != nil && cookie.Value != "" {
				userID, err = store.LookupSession(ctx, cookie.Value)
				if err == nil {
					username, _ := store.GetUsernameByID(ctx, userID)
					if username != "" {
						ctx = context.WithValue(ctx, ContextKeyUsername, username)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
					apiKey = strings.TrimPrefix(auth, "Bearer ")
				}
			}
			if apiKey != "" {
				userID, err = store.ValidateAPIKey(ctx, apiKey)
				if err == nil {
					username, _ := store.GetUsernameByID(ctx, userID)
					if username != "" {
						ctx = context.WithValue(ctx, ContextKeyUsername, username)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			accept := r.Header.Get("Accept")
			if strings.Contains(accept, "application/json") || strings.HasPrefix(r.URL.Path, "/api/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"authorization required"}`))
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
		})
	}
}

// UsernameFromRequest returns the username set by Middleware, or "" if not set.
func UsernameFromRequest(r *http.Request) string {
	v := r.Context().Value(ContextKeyUsername)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// IsAdmin returns true if the request is authenticated as the admin user (user management).
func IsAdmin(r *http.Request) bool {
	return UsernameFromRequest(r) == "admin"
}

// SessionCookieName returns the cookie name used for sessions (for logout clearing).
func SessionCookieName() string {
	return sessionCookieName
}
