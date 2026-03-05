package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"pictago/internal/audit"
	"pictago/internal/auth"
	"pictago/internal/requestctx"
)

func (s *Server) apiKeysHandler() http.HandlerFunc {
	logKeyCreated := func(username, keyName, outcome string, statusCode int, r *http.Request) {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action: "api-key-created", Category: []string{"iam", "authentication"}, Type: []string{"creation", "access"},
			Outcome: outcome, Message: "API key created " + outcome + ": " + username + " (name=" + keyName + ")",
			User: &audit.UserFields{Name: username}, URL: requestctx.RequestURL(r),
			ClientIP: requestctx.ClientIP(r.Context()), StatusCode: statusCode, Method: r.Method,
		}))
	}
	return func(w http.ResponseWriter, r *http.Request) {
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
		switch r.Method {
		case http.MethodGet:
			ctx, span := otel.Tracer("pictago").Start(r.Context(), "apikeys.list")
			defer span.End()
			r = r.WithContext(ctx)
			keys, err := s.Auth.ListAPIKeys(r.Context(), userID)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "list keys")
				http.Error(w, "failed to list keys", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, keys)
			return
		case http.MethodPost:
			ctx, span := otel.Tracer("pictago").Start(r.Context(), "apikeys.create")
			defer span.End()
			r = r.WithContext(ctx)
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			keyName := strings.TrimSpace(body.Name)
			key, err := s.Auth.CreateAPIKey(r.Context(), userID, keyName)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "create key")
				logKeyCreated(username, keyName, "failure", http.StatusInternalServerError, r)
				http.Error(w, "failed to create key", http.StatusInternalServerError)
				return
			}
			logKeyCreated(username, keyName, "success", http.StatusCreated, r)
			writeJSON(w, http.StatusCreated, map[string]string{"key": key, "message": "Copy the key now; it will not be shown again."})
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func (s *Server) apiKeyByIDHandler() http.HandlerFunc {
	logKeyDeleted := func(username string, keyID int64, outcome string, statusCode int, r *http.Request) {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action: "api-key-deleted", Category: []string{"iam", "authentication"}, Type: []string{"deletion", "access"},
			Outcome: outcome, Message: "API key deleted " + outcome + ": user=" + username,
			User: &audit.UserFields{Name: username}, URL: requestctx.RequestURL(r),
			ClientIP: requestctx.ClientIP(r.Context()), StatusCode: statusCode, Method: r.Method,
			Extra: map[string]interface{}{"pictago.api_key.id": keyID},
		}))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "apikeys.revoke")
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
		idStr := strings.TrimPrefix(r.URL.Path, "/api/keys/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id < 1 {
			span.SetStatus(codes.Error, "invalid key id")
			logKeyDeleted(username, id, "failure", http.StatusBadRequest, r)
			http.Error(w, "invalid key id", http.StatusBadRequest)
			return
		}
		if err := s.Auth.DeleteAPIKey(r.Context(), id, userID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "not found")
			logKeyDeleted(username, id, "failure", http.StatusNotFound, r)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		logKeyDeleted(username, id, "success", http.StatusNoContent, r)
		w.WriteHeader(http.StatusNoContent)
	}
}
