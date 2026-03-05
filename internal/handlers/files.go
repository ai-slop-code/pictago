package handlers

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"pictago/internal/audit"
	"pictago/internal/auth"
	"pictago/internal/optimize"
	"pictago/internal/requestctx"
	"pictago/internal/validate"
)

func (s *Server) uploadHandler() http.HandlerFunc {
	logUpload := func(username, filePath string, sizeBytes int64, outcome string, statusCode int, r *http.Request) {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action:     "file-upload",
			Category:   []string{"file"},
			Type:       []string{"creation", "access"},
			Outcome:    outcome,
			Message:    "file upload " + outcome + ": " + filePath,
			User:       &audit.UserFields{Name: username},
			File:       &audit.FileFields{Path: filePath, Size: sizeBytes},
			URL:        requestctx.RequestURL(r),
			ClientIP:   requestctx.ClientIP(r.Context()),
			StatusCode: statusCode,
			Method:     r.Method,
		}))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "files.upload")
		defer span.End()
		r = r.WithContext(ctx)
		if r.Method != http.MethodPost {
			span.SetStatus(codes.Error, "method not allowed")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			span.SetStatus(codes.Error, "unauthorized")
			logUpload("", "", 0, "failure", http.StatusUnauthorized, r)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := s.Auth.GetUserID(r.Context(), username)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "user not found")
			logUpload(username, "", 0, "failure", http.StatusInternalServerError, r)
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		err = r.ParseMultipartForm(s.MaxUploadBytes)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "parse multipart")
			logUpload(username, "", 0, "failure", http.StatusBadRequest, r)
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
			span.RecordError(err)
			span.SetStatus(codes.Error, "missing or invalid file")
			logUpload(username, "", 0, "failure", http.StatusBadRequest, r)
			http.Error(w, "missing or invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "read file")
			logUpload(username, "", 0, "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to read file", http.StatusInternalServerError)
			return
		}
		if !validate.IsImage(data) {
			span.SetStatus(codes.Error, "not an image")
			logUpload(username, "", 0, "failure", http.StatusBadRequest, r)
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
			optimized, newCT, err := optimize.Image(data)
			if err == nil {
				outData = optimized
				if strings.HasPrefix(newCT, "image/jpeg") {
					ext := filepath.Ext(filename)
					filename = strings.TrimSuffix(filename, ext) + ".jpg"
				}
			}
		}
		relativePath, err := s.Files.SaveInCollection(userID, collection, filename, outData)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "save file")
			logUpload(username, "", 0, "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to save file", http.StatusInternalServerError)
			return
		}
		if err := s.Auth.RecordUpload(r.Context(), userID, relativePath, int64(len(outData))); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "record upload")
			logUpload(username, "/files/"+relativePath, int64(len(outData)), "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to record upload", http.StatusInternalServerError)
			return
		}
		if s.ThumbDir != "" {
			if thumb, err := optimize.Thumbnail(outData, 200); err == nil {
				thumbPath := filepath.Join(s.ThumbDir, relativePath)
				if err := os.MkdirAll(filepath.Dir(thumbPath), 0755); err != nil {
					log.Printf("thumbnail: mkdir %s: %v", filepath.Dir(thumbPath), err)
				} else if err := os.WriteFile(thumbPath, thumb, 0644); err != nil {
					log.Printf("thumbnail: write %s: %v", thumbPath, err)
				}
			}
		}
		fullPath := "/files/" + relativePath
		logUpload(username, fullPath, int64(len(outData)), "success", http.StatusCreated, r)
		writeJSON(w, http.StatusCreated, map[string]string{"path": fullPath})
	}
}

func (s *Server) listHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "files.list")
		defer span.End()
		r = r.WithContext(ctx)
		if r.Method != http.MethodGet {
			span.SetStatus(codes.Error, "method not allowed")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			span.SetStatus(codes.Error, "unauthorized")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := s.Auth.GetUserID(r.Context(), username)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "user not found")
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		records, err := s.Auth.ListUploads(r.Context(), userID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "list uploads")
			http.Error(w, "failed to list files", http.StatusInternalServerError)
			return
		}
		out := make([]fileListEntry, 0, len(records))
		for _, rec := range records {
			relativePath := strings.TrimPrefix(rec.Path, "/files/")
			thumbURL := ""
			if s.ThumbDir != "" && relativePath != "" {
				thumbURL = "/api/thumbnails/" + relativePath
			}
			out = append(out, fileListEntry{
				ID:           rec.ID,
				Path:         rec.Path,
				Size:         rec.Size,
				Collection:   rec.Collection,
				Filename:     rec.Filename,
				ThumbnailURL: thumbURL,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// fileListEntry is the JSON shape for GET /api/files (includes thumbnail_url for the UI).
type fileListEntry struct {
	ID           int64  `json:"id"`
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	Collection   string `json:"collection"`
	Filename     string `json:"filename"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// thumbnailHandler serves thumbnail images; auth-only (under /api/). Path: /api/thumbnails/<userID>/<collection>/<filename>.
func (s *Server) thumbnailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.ThumbDir == "" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := s.Auth.GetUserID(r.Context(), username)
		if err != nil {
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		relativePath := strings.TrimPrefix(r.URL.Path, "/api/thumbnails/")
		relativePath = strings.Trim(relativePath, "/")
		if relativePath == "" || strings.Contains(relativePath, "..") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		firstSegment := relativePath
		if idx := strings.Index(relativePath, "/"); idx >= 0 {
			firstSegment = relativePath[:idx]
		}
		if firstSegment != strconv.FormatInt(userID, 10) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		localPath := filepath.Join(s.ThumbDir, filepath.FromSlash(relativePath))
		data, err := os.ReadFile(localPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

func (s *Server) deleteFileHandler() http.HandlerFunc {
	logDelete := func(username, filePath, outcome string, statusCode int, r *http.Request) {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action:     "file-deleted",
			Category:   []string{"file"},
			Type:       []string{"deletion", "access"},
			Outcome:    outcome,
			Message:    "file deleted " + outcome + ": " + filePath,
			User:       &audit.UserFields{Name: username},
			File:       &audit.FileFields{Path: filePath},
			URL:        requestctx.RequestURL(r),
			ClientIP:   requestctx.ClientIP(r.Context()),
			StatusCode: statusCode,
			Method:     r.Method,
		}))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "files.delete")
		defer span.End()
		r = r.WithContext(ctx)
		if r.Method != http.MethodDelete {
			span.SetStatus(codes.Error, "method not allowed")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username := auth.UsernameFromRequest(r)
		if username == "" {
			span.SetStatus(codes.Error, "unauthorized")
			logDelete("", "", "failure", http.StatusUnauthorized, r)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID, err := s.Auth.GetUserID(r.Context(), username)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "user not found")
			logDelete(username, "", "failure", http.StatusInternalServerError, r)
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/files/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			span.SetStatus(codes.Error, "invalid file id")
			logDelete(username, "", "failure", http.StatusBadRequest, r)
			http.Error(w, "invalid file id", http.StatusBadRequest)
			return
		}
		relativePath, err := s.Auth.DeleteUpload(r.Context(), id, userID)
		if err != nil {
			span.RecordError(err)
			if errors.Is(err, sql.ErrNoRows) {
				span.SetStatus(codes.Error, "not found")
				logDelete(username, "", "failure", http.StatusNotFound, r)
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			span.SetStatus(codes.Error, "delete upload")
			logDelete(username, "", "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to delete", http.StatusInternalServerError)
			return
		}
		if err := s.Files.Delete(relativePath); err != nil {
			log.Printf("files.delete: remove file %s: %v", relativePath, err)
		}
		if s.ThumbDir != "" {
			if err := os.Remove(filepath.Join(s.ThumbDir, relativePath)); err != nil && !os.IsNotExist(err) {
				log.Printf("files.delete: remove thumbnail %s: %v", relativePath, err)
			}
		}
		logDelete(username, "/files/"+relativePath, "success", http.StatusNoContent, r)
		w.WriteHeader(http.StatusNoContent)
	}
}
