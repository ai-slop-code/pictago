package handlers

import (
	"context"
	"io/fs"
	"net/http"

	"pictago/internal/audit"
	"pictago/internal/auth"
	"pictago/internal/middleware"
	"pictago/internal/requestctx"
	"pictago/internal/storage"
)

// Server holds dependencies and exposes Routes() and EnsureUser().
type Server struct {
	Auth           auth.Store
	Files          storage.Store
	Audit          audit.Logger
	MaxUploadBytes int64
	ThumbDir       string // directory for thumbnails (e.g. data/thumbnails); empty disables thumbnails
	UI             fs.FS
	Version        string // build version (e.g. from -ldflags), empty means "dev"
}

// NewServer returns a new Server with the given dependencies.
func NewServer(authStore auth.Store, fileStore storage.Store, auditLog audit.Logger, maxUploadBytes int64, thumbDir string, ui fs.FS) *Server {
	return &Server{
		Auth:           authStore,
		Files:          fileStore,
		Audit:          auditLog,
		MaxUploadBytes: maxUploadBytes,
		ThumbDir:       thumbDir,
		UI:             ui,
		Version:        "dev",
	}
}

// EnsureUser creates the default admin user if the DB has no users (first run only).
func (s *Server) EnsureUser() {
	ctx := context.Background()
	err := s.Auth.CreateUser(ctx, "admin", "admin")
	if err == nil {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action:   "user-created",
			Category: []string{"iam", "configuration"},
			Type:     []string{"creation", "user"},
			Outcome:  "success",
			Message:  "user created: admin (by system)",
			User:     &audit.UserFields{Name: "system", Target: &audit.UserFields{Name: "admin"}},
		}))
	}
}

// Routes returns the root http.Handler with all routes registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Operational endpoints (no auth)
	mux.HandleFunc("/health", s.healthHandler())
	mux.HandleFunc("/ready", s.readyHandler())
	mux.HandleFunc("/version", s.versionHandler())

	filesPrefix := "/files/"
	fileServer := http.StripPrefix(filesPrefix, http.FileServer(http.Dir(s.Files.Root())))
	filesHandler := middleware.SecurityHeaders(
		audit.WrapFilesHandler(s.Audit, fileServer, requestctx.ClientIPFromRequest),
		false,
	)
	mux.Handle(filesPrefix, filesHandler)

	mux.HandleFunc("/api/login", s.loginHandler())
	mux.HandleFunc("/logout", s.logoutHandler())
	mux.HandleFunc("/login", s.loginPageHandler())

	api := http.NewServeMux()
	api.HandleFunc("/api/upload", s.uploadHandler())
	api.HandleFunc("/api/files", s.listHandler())
	api.HandleFunc("/api/files/", s.deleteFileHandler())
	api.HandleFunc("/api/stats", s.statsHandler())
	api.HandleFunc("/api/users", s.usersHandler())
	api.HandleFunc("/api/users/", s.usersHandler())
	api.HandleFunc("/api/me", s.meHandler())
	api.HandleFunc("/api/me/password", s.changePasswordHandler())
	api.HandleFunc("/api/keys", s.apiKeysHandler())
	api.HandleFunc("/api/keys/", s.apiKeyByIDHandler())
	api.HandleFunc("/api/config", s.configHandler())
	api.HandleFunc("/api/thumbnails/", s.thumbnailHandler())

	mux.Handle("/api/", auth.Middleware(s.Auth)(middleware.SecurityHeaders(api, true)))
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})
	mux.Handle("/admin/", auth.Middleware(s.Auth)(middleware.SecurityHeaders(http.HandlerFunc(s.serveUI), true)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	return mux
}
