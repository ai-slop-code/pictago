package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"pictago/internal/audit"
	"pictago/internal/auth"
	"pictago/internal/requestctx"
)

func (s *Server) usersHandler() http.HandlerFunc {
	adminByID := s.adminUserByIDHandler()
	logUserCreated := func(actor, target, outcome string, statusCode int, r *http.Request) {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action:     "user-created",
			Category:   []string{"iam", "configuration"},
			Type:       []string{"creation", "user"},
			Outcome:    outcome,
			Message:    "user created: " + target + " (by " + actor + ")",
			User:       &audit.UserFields{Name: actor, Target: &audit.UserFields{Name: target}},
			URL:        requestctx.RequestURL(r),
			ClientIP:   requestctx.ClientIP(r.Context()),
			StatusCode: statusCode,
			Method:     r.Method,
		}))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
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
			ctx, span := otel.Tracer("pictago").Start(r.Context(), "users.list")
			defer span.End()
			r = r.WithContext(ctx)
			list, err := s.Auth.ListUsersWithStats(r.Context())
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "list users")
				http.Error(w, "failed to list users", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
			return
		case http.MethodPost:
			ctx, span := otel.Tracer("pictago").Start(r.Context(), "users.create")
			defer span.End()
			r = r.WithContext(ctx)
			var body struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Password == "" {
				span.SetStatus(codes.Error, "invalid body")
				logUserCreated("admin", body.Username, "failure", http.StatusBadRequest, r)
				http.Error(w, "invalid body: username and password required", http.StatusBadRequest)
				return
			}
			if err := s.Auth.CreateUser(r.Context(), body.Username, body.Password); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "create user")
				logUserCreated("admin", body.Username, "failure", http.StatusConflict, r)
				http.Error(w, "failed to create user", http.StatusConflict)
				return
			}
			logUserCreated("admin", body.Username, "success", http.StatusCreated, r)
			w.WriteHeader(http.StatusCreated)
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (s *Server) adminUserByIDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAdmin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		actor := auth.UsernameFromRequest(r)
		suffix := strings.TrimPrefix(r.URL.Path, "/api/users/")
		suffix = strings.Trim(suffix, "/")
		parts := strings.SplitN(suffix, "/", 3)
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
			ctx, span := otel.Tracer("pictago").Start(r.Context(), "users.delete")
			defer span.End()
			r = r.WithContext(ctx)
			if r.Method != http.MethodDelete {
				span.SetStatus(codes.Error, "method not allowed")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := s.Auth.DeleteUser(r.Context(), userID); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "delete user")
				s.Audit.Log(audit.BuildEvent(audit.EventOpts{
					Action: "user-deleted", Category: []string{"iam"}, Type: []string{"deletion", "user"},
					Outcome: "failure", Message: "user deleted failure: id=" + parts[0],
					User: &audit.UserFields{Name: actor, Target: &audit.UserFields{ID: parts[0]}},
					URL: requestctx.RequestURL(r), ClientIP: requestctx.ClientIP(r.Context()),
					StatusCode: http.StatusInternalServerError, Method: r.Method,
					Extra: map[string]interface{}{"pictago.target_user_id": userID},
				}))
				http.Error(w, "failed to delete user", http.StatusInternalServerError)
				return
			}
			s.Audit.Log(audit.BuildEvent(audit.EventOpts{
				Action: "user-deleted", Category: []string{"iam"}, Type: []string{"deletion", "user"},
				Outcome: "success", Message: "user deleted success: id=" + parts[0],
				User: &audit.UserFields{Name: actor, Target: &audit.UserFields{ID: parts[0]}},
				URL: requestctx.RequestURL(r), ClientIP: requestctx.ClientIP(r.Context()),
				StatusCode: http.StatusNoContent, Method: r.Method,
				Extra: map[string]interface{}{"pictago.target_user_id": userID},
			}))
			w.WriteHeader(http.StatusNoContent)
			return
		case len(parts) >= 2 && parts[1] == "stats":
			ctx, span := otel.Tracer("pictago").Start(r.Context(), "users.stats")
			defer span.End()
			r = r.WithContext(ctx)
			if r.Method != http.MethodGet {
				span.SetStatus(codes.Error, "method not allowed")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			st, err := s.Auth.GetStats(r.Context(), userID)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "get stats")
				http.Error(w, "failed to get stats", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(st)
			return
		case len(parts) >= 3 && parts[1] == "collections":
			ctx, span := otel.Tracer("pictago").Start(r.Context(), "users.collection.delete")
			defer span.End()
			r = r.WithContext(ctx)
			if r.Method != http.MethodDelete {
				span.SetStatus(codes.Error, "method not allowed")
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			collectionName := parts[2]
			paths, err := s.Auth.DeleteCollection(r.Context(), userID, collectionName)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "delete collection")
				s.Audit.Log(audit.BuildEvent(audit.EventOpts{
					Action: "collection-deleted", Category: []string{"file", "iam"}, Type: []string{"deletion", "access"},
					Outcome: "failure", Message: "collection deleted failure: " + collectionName,
					User: &audit.UserFields{Name: actor, Target: &audit.UserFields{ID: parts[0]}},
					URL: requestctx.RequestURL(r), ClientIP: requestctx.ClientIP(r.Context()),
					StatusCode: http.StatusInternalServerError, Method: r.Method,
					Extra: map[string]interface{}{"pictago.target_user_id": userID, "pictago.collection": collectionName},
				}))
				http.Error(w, "failed to delete collection", http.StatusInternalServerError)
				return
			}
			for _, p := range paths {
				_ = s.Files.Delete(p)
				if s.ThumbDir != "" {
					_ = os.Remove(filepath.Join(s.ThumbDir, p))
				}
			}
			if len(paths) > 0 {
				dir := filepath.Dir(paths[0])
				_ = s.Files.DeleteDirectory(dir)
			}
			s.Audit.Log(audit.BuildEvent(audit.EventOpts{
				Action: "collection-deleted", Category: []string{"file", "iam"}, Type: []string{"deletion", "access"},
				Outcome: "success", Message: "collection deleted success: " + collectionName,
				User: &audit.UserFields{Name: actor, Target: &audit.UserFields{ID: parts[0]}},
				URL: requestctx.RequestURL(r), ClientIP: requestctx.ClientIP(r.Context()),
				StatusCode: http.StatusNoContent, Method: r.Method,
				Extra: map[string]interface{}{"pictago.target_user_id": userID, "pictago.collection": collectionName},
			}))
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
}

func (s *Server) meHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "me")
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"username": username})
	}
}

func (s *Server) changePasswordHandler() http.HandlerFunc {
	logPw := func(username, outcome string, statusCode int, r *http.Request) {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action: "user-password-change", Category: []string{"authentication", "iam"}, Type: []string{"change", "user"},
			Outcome: outcome, Message: "password change " + outcome + ": " + username,
			User: &audit.UserFields{Name: username}, URL: requestctx.RequestURL(r),
			ClientIP: requestctx.ClientIP(r.Context()), StatusCode: statusCode, Method: r.Method,
		}))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "password.change")
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
			logPw("", "failure", http.StatusUnauthorized, r)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NewPassword == "" {
			span.SetStatus(codes.Error, "invalid body")
			logPw(username, "failure", http.StatusBadRequest, r)
			http.Error(w, "invalid body: current_password and new_password required", http.StatusBadRequest)
			return
		}
		valid, err := s.Auth.ValidateUser(r.Context(), username, body.CurrentPassword)
		if err != nil || !valid {
			span.SetStatus(codes.Error, "invalid current password")
			logPw(username, "failure", http.StatusUnauthorized, r)
			http.Error(w, "invalid current password", http.StatusUnauthorized)
			return
		}
		if err := s.Auth.UpdatePassword(r.Context(), username, body.NewPassword); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "update password")
			logPw(username, "failure", http.StatusInternalServerError, r)
			http.Error(w, "failed to update password", http.StatusInternalServerError)
			return
		}
		logPw(username, "success", http.StatusNoContent, r)
		w.WriteHeader(http.StatusNoContent)
	}
}
