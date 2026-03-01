package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pictago/internal/audit"
	"pictago/internal/auth"
	"pictago/internal/optimize"
	"pictago/internal/storage"
	"pictago/internal/validate"
)

//go:embed ui
var uiFS embed.FS

func main() {
	dataDir := getEnv("DATA_DIR", "./data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		log.Fatalf("resolve data dir: %v", err)
	}

	filesDir := getEnv("FILES_DIR", filepath.Join(dataDir, "files"))
	dbPath := getEnv("DB_PATH", filepath.Join(dataDir, "users.db"))
	auditLogPath := getEnv("AUDIT_LOG_PATH", filepath.Join(dataDir, "audit.json"))
	maxUploadMB := getEnvInt("MAX_UPLOAD_MB", 10)
	maxUploadBytes := int64(maxUploadMB) << 20

	auditLog, err := audit.New(auditLogPath)
	if err != nil {
		log.Fatalf("audit log: %v", err)
	}
	defer auditLog.Close()

	authStore, err := auth.NewStore(dbPath)
	if err != nil {
		log.Fatalf("auth store: %v", err)
	}
	defer authStore.Close()

	ensureUser(authStore, auditLog)

	fileStore, err := storage.New(filesDir)
	if err != nil {
		log.Fatalf("file store: %v", err)
	}

	mux := http.NewServeMux()

	filesPrefix := "/files/"
	filesHandler := securityHeaders(auditLog.WrapFilesHandler(http.StripPrefix(filesPrefix, http.FileServer(http.Dir(fileStore.Root())))), false)
	mux.Handle(filesPrefix, filesHandler)

	mux.HandleFunc("/api/login", loginHandler(authStore, auditLog))
	mux.HandleFunc("/logout", logoutHandler(authStore, auditLog))
	mux.HandleFunc("/login", loginPageHandler())

	api := http.NewServeMux()
	api.HandleFunc("/api/upload", uploadHandler(authStore, fileStore, auditLog, maxUploadBytes))
	api.HandleFunc("/api/files", listHandler(authStore))
	api.HandleFunc("/api/files/", deleteFileHandler(authStore, fileStore, auditLog))
	api.HandleFunc("/api/stats", statsHandler(authStore))
	api.HandleFunc("/api/users", usersHandler(authStore, fileStore, auditLog))
	api.HandleFunc("/api/users/", usersHandler(authStore, fileStore, auditLog))
	api.HandleFunc("/api/me", meHandler())
	api.HandleFunc("/api/me/password", changePasswordHandler(authStore, auditLog))
	api.HandleFunc("/api/keys", apiKeysHandler(authStore, auditLog))
	api.HandleFunc("/api/keys/", apiKeyByIDHandler(authStore, auditLog))
	api.HandleFunc("/api/config", configHandler(maxUploadBytes))

	mux.Handle("/api/", auth.Middleware(authStore)(securityHeaders(api, true)))
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})
	mux.Handle("/admin/", auth.Middleware(authStore)(securityHeaders(http.HandlerFunc(serveUI), true)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	addr := getEnv("ADDR", ":8080")
	if auditLogPath != "" {
		log.Printf("audit log: %s", auditLogPath)
	} else {
		log.Printf("audit log: disabled")
	}
	log.Printf("listening on %s (max upload %d MiB)", addr, maxUploadMB)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func securityHeaders(next http.Handler, forUI bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if forUI {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://unpkg.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'")
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return defaultVal
		}
		return n
	}
	return defaultVal
}

// ensureUser creates a default admin user if the DB has no users (first run only).
// Change the default password immediately via the UI; see SECURITY.md.
func ensureUser(store *auth.Store, auditLog *audit.Logger) {
	ctx := context.Background()
	err := store.CreateUser(ctx, "admin", "admin")
	if err == nil {
		auditLog.LogUserCreated("system", "admin", "success", 0, nil)
	}
}

func uploadHandler(authStore *auth.Store, fileStore *storage.Store, auditLog *audit.Logger, maxUploadBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			auditLog.LogFileUpload("", "", 0, "failure", http.StatusUnauthorized, r)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := authStore.GetUserID(r.Context(), username)
		if err != nil {
			auditLog.LogFileUpload(username, "", 0, "failure", http.StatusInternalServerError, r)
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		err = r.ParseMultipartForm(maxUploadBytes)
		if err != nil {
			auditLog.LogFileUpload(username, "", 0, "failure", http.StatusBadRequest, r)
			http.Error(w, "file too large or bad multipart form", http.StatusBadRequest)
			return
		}
		collection := strings.TrimSpace(r.FormValue("collection"))
		if collection == "" {
			collection = "default"
		}
		optimizeFlag := r.FormValue("optimize") == "true" || r.FormValue("optimize") == "1"
		file, header, err := r.FormFile("file")
		if err != nil {
			auditLog.LogFileUpload(username, "", 0, "failure", http.StatusBadRequest, r)
			http.Error(w, "missing or invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			auditLog.LogFileUpload(username, "", 0, "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to read file", http.StatusInternalServerError)
			return
		}
		if !validate.IsImage(data) {
			auditLog.LogFileUpload(username, "", 0, "failure", http.StatusBadRequest, r)
			http.Error(w, "only images are allowed (JPEG, PNG, GIF, WebP)", http.StatusBadRequest)
			return
		}
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = optimize.SniffContentType(data)
		}
		filename := header.Filename
		if filename == "" {
			filename = "upload"
		}
		outData := data
		if optimizeFlag {
			optimized, newCT, ok := optimize.Image(data, contentType)
			if ok {
				outData = optimized
				if strings.HasPrefix(newCT, "image/jpeg") {
					ext := filepath.Ext(filename)
					filename = strings.TrimSuffix(filename, ext) + ".jpg"
				}
			}
		}
		relativePath, err := fileStore.SaveInCollection(userID, collection, filename, outData)
		if err != nil {
			auditLog.LogFileUpload(username, "", 0, "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to save file", http.StatusInternalServerError)
			return
		}
		if err := authStore.RecordUpload(r.Context(), userID, relativePath, int64(len(outData))); err != nil {
			auditLog.LogFileUpload(username, "/files/"+relativePath, int64(len(outData)), "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to record upload", http.StatusInternalServerError)
			return
		}
		fullPath := "/files/" + relativePath
		auditLog.LogFileUpload(username, fullPath, int64(len(outData)), "success", http.StatusCreated, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"path": fullPath})
	}
}

func listHandler(authStore *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := authStore.GetUserID(r.Context(), username)
		if err != nil {
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		records, err := authStore.ListUploads(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to list files", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	}
}

func deleteFileHandler(authStore *auth.Store, fileStore *storage.Store, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			auditLog.LogFileDeleted("", "", "failure", http.StatusUnauthorized, r)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := authStore.GetUserID(r.Context(), username)
		if err != nil {
			auditLog.LogFileDeleted(username, "", "failure", http.StatusInternalServerError, r)
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/files/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			auditLog.LogFileDeleted(username, "", "failure", http.StatusBadRequest, r)
			http.Error(w, "invalid file id", http.StatusBadRequest)
			return
		}
		relativePath, err := authStore.DeleteUpload(r.Context(), id, userID)
		if err != nil {
			if err == sql.ErrNoRows {
				auditLog.LogFileDeleted(username, "", "failure", http.StatusNotFound, r)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			auditLog.LogFileDeleted(username, "", "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to delete", http.StatusInternalServerError)
			return
		}
		_ = fileStore.Delete(relativePath)
		auditLog.LogFileDeleted(username, "/files/"+relativePath, "success", http.StatusNoContent, r)
		w.WriteHeader(http.StatusNoContent)
	}
}

func statsHandler(authStore *auth.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := auth.UsernameFromRequest(r)
		if username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := authStore.GetUserID(r.Context(), username)
		if err != nil {
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		st, err := authStore.GetStats(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to get stats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(st)
	}
}

func configHandler(maxUploadBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"max_upload_bytes": maxUploadBytes,
			"max_upload_mb":    maxUploadBytes >> 20,
		})
	}
}

func usersHandler(authStore *auth.Store, fileStore *storage.Store, auditLog *audit.Logger) http.HandlerFunc {
	adminByID := adminUserByIDHandler(authStore, fileStore, auditLog)
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Only list/create when path is exactly /api/users or /api/users/; otherwise it's /api/users/:id/...
		if path != "/api/users" && path != "/api/users/" {
			adminByID(w, r)
			return
		}
		if !auth.IsAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet:
			list, err := authStore.ListUsersWithStats(r.Context())
			if err != nil {
				http.Error(w, "failed to list users", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
			return
		case http.MethodPost:
			var body struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Password == "" {
				auditLog.LogUserCreated("admin", body.Username, "failure", http.StatusBadRequest, r)
				http.Error(w, "invalid body: username and password required", http.StatusBadRequest)
				return
			}
			if err := authStore.CreateUser(r.Context(), body.Username, body.Password); err != nil {
				auditLog.LogUserCreated("admin", body.Username, "failure", http.StatusConflict, r)
				http.Error(w, "failed to create user", http.StatusConflict)
				return
			}
			auditLog.LogUserCreated("admin", body.Username, "success", http.StatusCreated, r)
			w.WriteHeader(http.StatusCreated)
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func adminUserByIDHandler(authStore *auth.Store, fileStore *storage.Store, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		actor := auth.UsernameFromRequest(r)
		suffix := strings.TrimPrefix(r.URL.Path, "/api/users/")
		suffix = strings.Trim(suffix, "/")
		parts := strings.SplitN(suffix, "/", 3) // id, optional "stats" or "collections", optional name
		if len(parts) < 1 || parts[0] == "" {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		userID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || userID < 1 {
			http.Error(w, "invalid user id", http.StatusBadRequest)
			return
		}
		switch {
		case len(parts) == 1:
			// DELETE /api/users/:id
			if r.Method != http.MethodDelete {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := authStore.DeleteUser(r.Context(), userID); err != nil {
				auditLog.LogUserDeleted(actor, userID, "failure", http.StatusInternalServerError, r)
				http.Error(w, "failed to delete user", http.StatusInternalServerError)
				return
			}
			auditLog.LogUserDeleted(actor, userID, "success", http.StatusNoContent, r)
			w.WriteHeader(http.StatusNoContent)
			return
		case len(parts) >= 2 && parts[1] == "stats":
			// GET /api/users/:id/stats
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			st, err := authStore.GetStats(r.Context(), userID)
			if err != nil {
				http.Error(w, "failed to get stats", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(st)
			return
		case len(parts) >= 3 && parts[1] == "collections":
			// DELETE /api/users/:id/collections/:name
			if r.Method != http.MethodDelete {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			collectionName := parts[2]
			paths, err := authStore.DeleteCollection(r.Context(), userID, collectionName)
			if err != nil {
				auditLog.LogCollectionDeleted(actor, userID, collectionName, "failure", http.StatusInternalServerError, r)
				http.Error(w, "failed to delete collection", http.StatusInternalServerError)
				return
			}
			for _, p := range paths {
				_ = fileStore.Delete(p)
			}
			if len(paths) > 0 {
				dir := filepath.Dir(paths[0])
				_ = fileStore.DeleteDirectory(dir)
			}
			auditLog.LogCollectionDeleted(actor, userID, collectionName, "success", http.StatusNoContent, r)
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
}

func meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": username})
	}
}

func apiKeysHandler(authStore *auth.Store, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := auth.UsernameFromRequest(r)
		if username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := authStore.GetUserID(r.Context(), username)
		if err != nil {
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		switch r.Method {
		case http.MethodGet:
			keys, err := authStore.ListAPIKeys(r.Context(), userID)
			if err != nil {
				http.Error(w, "failed to list keys", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(keys)
			return
		case http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			keyName := strings.TrimSpace(body.Name)
			key, err := authStore.CreateAPIKey(r.Context(), userID, keyName)
			if err != nil {
				auditLog.LogAPIKeyCreated(username, keyName, "failure", http.StatusInternalServerError, r)
				http.Error(w, "failed to create key", http.StatusInternalServerError)
				return
			}
			auditLog.LogAPIKeyCreated(username, keyName, "success", http.StatusCreated, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"key": key, "message": "Copy the key now; it will not be shown again."})
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func apiKeyByIDHandler(authStore *auth.Store, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := authStore.GetUserID(r.Context(), username)
		if err != nil {
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/keys/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			auditLog.LogAPIKeyDeleted(username, id, "failure", http.StatusBadRequest, r)
			http.Error(w, "invalid key id", http.StatusBadRequest)
			return
		}
		if err := authStore.DeleteAPIKey(r.Context(), id, userID); err != nil {
			auditLog.LogAPIKeyDeleted(username, id, "failure", http.StatusNotFound, r)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		auditLog.LogAPIKeyDeleted(username, id, "success", http.StatusNoContent, r)
		w.WriteHeader(http.StatusNoContent)
	}
}

func changePasswordHandler(authStore *auth.Store, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			auditLog.LogPasswordChange("", "failure", http.StatusUnauthorized, r)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NewPassword == "" {
			auditLog.LogPasswordChange(username, "failure", http.StatusBadRequest, r)
			http.Error(w, "invalid body: current_password and new_password required", http.StatusBadRequest)
			return
		}
		valid, err := authStore.ValidateUser(r.Context(), username, body.CurrentPassword)
		if err != nil || !valid {
			auditLog.LogPasswordChange(username, "failure", http.StatusUnauthorized, r)
			http.Error(w, "invalid current password", http.StatusUnauthorized)
			return
		}
		if err := authStore.UpdatePassword(r.Context(), username, body.NewPassword); err != nil {
			auditLog.LogPasswordChange(username, "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to update password", http.StatusInternalServerError)
			return
		}
		auditLog.LogPasswordChange(username, "success", http.StatusNoContent, r)
		w.WriteHeader(http.StatusNoContent)
	}
}

func loginHandler(authStore *auth.Store, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Password == "" {
			auditLog.LogLogin(body.Username, "failure", http.StatusBadRequest, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "username and password required"})
			return
		}
		valid, err := authStore.ValidateUser(r.Context(), body.Username, body.Password)
		if err != nil || !valid {
			auditLog.LogLogin(body.Username, "failure", http.StatusUnauthorized, r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
			return
		}
		userID, err := authStore.GetUserID(r.Context(), body.Username)
		if err != nil {
			auditLog.LogLogin(body.Username, "failure", http.StatusInternalServerError, r)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		token, err := authStore.CreateSession(r.Context(), userID)
		if err != nil {
			auditLog.LogLogin(body.Username, "failure", http.StatusInternalServerError, r)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		auditLog.LogLogin(body.Username, "success", http.StatusOK, r)
		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookieName(),
			Value:    token,
			Path:     "/",
			MaxAge:   int(7 * 24 * 3600),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": body.Username})
	}
}

func logoutHandler(authStore *auth.Store, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := auth.UsernameFromRequest(r)
		if username == "" {
			// Still might have had a session cookie; try to resolve for audit
			if cookie, _ := r.Cookie(auth.SessionCookieName()); cookie != nil && cookie.Value != "" {
				if userID, err := authStore.LookupSession(r.Context(), cookie.Value); err == nil {
					if name, _ := authStore.GetUsernameByID(r.Context(), userID); name != "" {
						username = name
					}
				}
			}
		}
		if cookie, _ := r.Cookie(auth.SessionCookieName()); cookie != nil && cookie.Value != "" {
			_ = authStore.DeleteSession(r.Context(), cookie.Value)
		}
		if username != "" {
			auditLog.LogLogout(username, r)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookieName(),
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func loginPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Login — Pictago</title><style>
*{box-sizing:border-box}body{font-family:system-ui,sans-serif;margin:0;min-height:100vh;background:#0f0f12;color:#e4e4e7;display:flex;align-items:center;justify-content:center;padding:1rem}
.card{background:#18181c;border:1px solid #2a2a32;border-radius:10px;padding:2rem;width:100%;max-width:320px}
h1{font-size:1.25rem;margin:0 0 1rem 0}
.form-group{margin-bottom:1rem}.form-group label{display:block;font-size:0.875rem;color:#71717a;margin-bottom:0.25rem}
.form-group input{width:100%;padding:0.5rem 0.75rem;border:1px solid #2a2a32;border-radius:6px;background:#0f0f12;color:#e4e4e7;font-size:1rem}
.form-group input:focus{outline:none;border-color:#a78bfa}
button{width:100%;padding:0.6rem;background:#a78bfa;color:#0f0f12;border:none;border-radius:6px;font-size:1rem;font-weight:500;cursor:pointer}
button:hover{background:#c4b5fd}.error{color:#f87171;font-size:0.875rem;margin-top:0.5rem}
</style></head><body><div class="card"><h1>Pictago</h1><form id="f" method="post" action="/api/login"><div class="form-group"><label for="u">Username</label><input type="text" id="u" name="username" required autocomplete="username"></div><div class="form-group"><label for="p">Password</label><input type="password" id="p" name="password" required autocomplete="current-password"></div><div id="err" class="error"></div><button type="submit">Sign in</button></form></div>
<script>document.getElementById("f").onsubmit=async function(e){e.preventDefault();var err=document.getElementById("err");err.textContent="";var r=await fetch("/api/login",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({username:document.getElementById("u").value,password:document.getElementById("p").value})});if(r.ok){window.location.href="/admin/";return}var j=await r.json();err.textContent=j.error||"Login failed";}</script></body></html>`)
	}
}

func serveUI(w http.ResponseWriter, r *http.Request) {
	data, err := uiFS.ReadFile("ui/index.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nonceStr := base64.StdEncoding.EncodeToString(nonce)
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://cdn.jsdelivr.net 'nonce-"+nonceStr+"' 'unsafe-eval'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := string(data)
	html = strings.Replace(html, "__CSP_NONCE__", nonceStr, 1)
	w.Write([]byte(html))
}
