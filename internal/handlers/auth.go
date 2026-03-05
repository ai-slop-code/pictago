package handlers

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"pictago/internal/audit"
	"pictago/internal/auth"
	"pictago/internal/requestctx"
)

func (s *Server) loginHandler() http.HandlerFunc {
	logLogin := func(username, outcome string, statusCode int, r *http.Request) {
		s.Audit.Log(audit.BuildEvent(audit.EventOpts{
			Action: "user-login", Category: []string{"authentication"}, Type: []string{"access", "user"},
			Outcome: outcome, Message: "login " + outcome + ": " + username,
			User: &audit.UserFields{Name: username}, URL: requestctx.RequestURL(r),
			ClientIP: requestctx.ClientIP(r.Context()), StatusCode: statusCode, Method: r.Method,
		}))
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "auth.login")
		defer span.End()
		r = r.WithContext(ctx)
		if r.Method != http.MethodPost {
			span.SetStatus(codes.Error, "method not allowed")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" || body.Password == "" {
			span.SetStatus(codes.Error, "bad request")
			logLogin(body.Username, "failure", http.StatusBadRequest, r)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
			return
		}
		valid, err := s.Auth.ValidateUser(r.Context(), body.Username, body.Password)
		if err != nil || !valid {
			span.SetStatus(codes.Error, "invalid credentials")
			logLogin(body.Username, "failure", http.StatusUnauthorized, r)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		userID, err := s.Auth.GetUserID(r.Context(), body.Username)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "get user id")
			logLogin(body.Username, "failure", http.StatusInternalServerError, r)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		token, err := s.Auth.CreateSession(r.Context(), userID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "create session")
			logLogin(body.Username, "failure", http.StatusInternalServerError, r)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		logLogin(body.Username, "success", http.StatusOK, r)
		http.SetCookie(w, &http.Cookie{
			Name:     auth.SessionCookieName(),
			Value:    token,
			Path:     "/",
			MaxAge:   int(7 * 24 * 3600),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		writeJSON(w, http.StatusOK, map[string]string{"username": body.Username})
	}
}

func (s *Server) logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "auth.logout")
		defer span.End()
		r = r.WithContext(ctx)
		username := auth.UsernameFromRequest(r)
		if username == "" {
			if cookie, _ := r.Cookie(auth.SessionCookieName()); cookie != nil && cookie.Value != "" {
				if userID, err := s.Auth.LookupSession(r.Context(), cookie.Value); err == nil {
					if name, _ := s.Auth.GetUsernameByID(r.Context(), userID); name != "" {
						username = name
					}
				}
			}
		}
		if cookie, _ := r.Cookie(auth.SessionCookieName()); cookie != nil && cookie.Value != "" {
			if err := s.Auth.DeleteSession(r.Context(), cookie.Value); err != nil {
				log.Printf("logout: delete session: %v", err)
			}
		}
		if username != "" {
			s.Audit.Log(audit.BuildEvent(audit.EventOpts{
				Action: "user-logout", Category: []string{"authentication", "session"}, Type: []string{"end", "user"},
				Outcome: "success", Message: "logout: " + username,
				User: &audit.UserFields{Name: username}, URL: requestctx.RequestURL(r),
				ClientIP: requestctx.ClientIP(r.Context()), StatusCode: 200, Method: r.Method,
			}))
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

func (s *Server) loginPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("pictago").Start(r.Context(), "auth.login.page")
		defer span.End()
		r = r.WithContext(ctx)
		if r.URL.Path != "/login" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(s.UI, "login.html")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "read login page")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(data); err != nil {
			log.Printf("login page write: %v", err)
		}
	}
}
