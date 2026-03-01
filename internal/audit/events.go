package audit

import (
	"fmt"
	"net/http"
	"strconv"
)

func reqInfo(r *http.Request, statusCode int) (url string, client *ClientFields, httpF *HTTPFields) {
	if r != nil {
		if r.URL != nil {
			url = r.URL.String()
			if r.URL.Scheme == "" {
				url = "http://" + r.Host + url
			}
		}
		client = &ClientFields{IP: clientIP(r)}
		httpF = &HTTPFields{
			RequestMethod:      r.Method,
			ResponseStatusCode: int64(statusCode),
		}
	}
	return url, client, httpF
}

// LogUserCreated logs admin or bootstrap user creation.
func (l *Logger) LogUserCreated(actor, targetUsername, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"iam", "configuration"},
			Action:   "user-created",
			Type:     []string{"creation", "user"},
			Outcome:  outcome,
		},
		User: &UserFields{
			Name:   actor,
			Target: &UserFields{Name: targetUsername},
		},
		Message: fmt.Sprintf("user created: %s (by %s)", targetUsername, actor),
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogLogin logs successful or failed login.
func (l *Logger) LogLogin(username, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"authentication"},
			Action:   "user-login",
			Type:     []string{"access", "user"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: username},
		Message: fmt.Sprintf("login %s: %s", outcome, username),
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogLogout logs user logout.
func (l *Logger) LogLogout(username string, r *http.Request) {
	url, client, httpF := reqInfo(r, 200)
	ev := Event{
		Event: EventFields{
			Category: []string{"authentication", "session"},
			Action:   "user-logout",
			Type:     []string{"end", "user"},
			Outcome:  "success",
		},
		User:    &UserFields{Name: username},
		Message: "logout: " + username,
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogFileUpload logs file upload.
func (l *Logger) LogFileUpload(username, filePath string, sizeBytes int64, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"file"},
			Action:   "file-upload",
			Type:     []string{"creation", "access"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: username},
		File:    &FileFields{Path: filePath, Size: sizeBytes},
		Message: fmt.Sprintf("file upload %s: %s (%d bytes)", outcome, filePath, sizeBytes),
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogFileDeleted logs file deletion.
func (l *Logger) LogFileDeleted(username, filePath, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"file"},
			Action:   "file-deleted",
			Type:     []string{"deletion", "access"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: username},
		File:    &FileFields{Path: filePath},
		Message: fmt.Sprintf("file deleted %s: %s", outcome, filePath),
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogFileAccess logs public file access (GET /files/...).
func (l *Logger) LogFileAccess(filePath, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"file", "web"},
			Action:   "file-access",
			Type:     []string{"access"},
			Outcome:  outcome,
		},
		File:    &FileFields{Path: filePath},
		Message: fmt.Sprintf("public file access %s: %s", outcome, filePath),
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogAPIKeyCreated logs API key creation.
func (l *Logger) LogAPIKeyCreated(username, keyName, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"iam", "authentication"},
			Action:   "api-key-created",
			Type:     []string{"creation", "access"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: username},
		Message: fmt.Sprintf("API key created %s: %s (name=%s)", outcome, username, keyName),
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogAPIKeyDeleted logs API key deletion.
func (l *Logger) LogAPIKeyDeleted(username string, keyID int64, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"iam", "authentication"},
			Action:   "api-key-deleted",
			Type:     []string{"deletion", "access"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: username},
		Message: fmt.Sprintf("API key deleted %s: user=%s key_id=%d", outcome, username, keyID),
		Extra:   map[string]interface{}{"pictago.api_key.id": keyID},
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogUserDeleted logs admin deletion of a user.
func (l *Logger) LogUserDeleted(actor string, targetUserID int64, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"iam"},
			Action:   "user-deleted",
			Type:     []string{"deletion", "user"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: actor, Target: &UserFields{ID: strconv.FormatInt(targetUserID, 10)}},
		Message: fmt.Sprintf("user deleted %s: id=%d by %s", outcome, targetUserID, actor),
		Extra:   map[string]interface{}{"pictago.target_user_id": targetUserID},
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogCollectionDeleted logs admin deletion of a user's collection.
func (l *Logger) LogCollectionDeleted(actor string, targetUserID int64, collectionName, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"file", "iam"},
			Action:   "collection-deleted",
			Type:     []string{"deletion", "access"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: actor, Target: &UserFields{ID: strconv.FormatInt(targetUserID, 10)}},
		Message: fmt.Sprintf("collection deleted %s: user_id=%d collection=%s by %s", outcome, targetUserID, collectionName, actor),
		Extra:   map[string]interface{}{"pictago.target_user_id": targetUserID, "pictago.collection": collectionName},
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// LogPasswordChange logs user password change.
func (l *Logger) LogPasswordChange(username, outcome string, statusCode int, r *http.Request) {
	url, client, httpF := reqInfo(r, statusCode)
	ev := Event{
		Event: EventFields{
			Category: []string{"authentication", "iam"},
			Action:   "user-password-change",
			Type:     []string{"change", "user"},
			Outcome:  outcome,
		},
		User:    &UserFields{Name: username},
		Message: fmt.Sprintf("password change %s: %s", outcome, username),
	}
	if url != "" {
		ev.URL = &URLFields{Full: url}
	}
	ev.Client = client
	ev.HTTP = httpF
	l.Log(ev)
}

// responseWriterWithStatus captures status code for audit.
type responseWriterWithStatus struct {
	http.ResponseWriter
	status int
}

func (w *responseWriterWithStatus) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// WrapFilesHandler wraps the /files/ handler to log each request (public file access).
func (l *Logger) WrapFilesHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != "HEAD" {
			next.ServeHTTP(w, r)
			return
		}
		rw := &responseWriterWithStatus{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		outcome := "success"
		if rw.status >= 400 {
			outcome = "failure"
		}
		path := r.URL.Path
		if path == "" {
			path = "/"
		}
		l.LogFileAccess(path, outcome, rw.status, r)
	})
}

// NoopLogger returns a logger that does nothing (when audit path is disabled).
func NoopLogger() *Logger { return &Logger{} }
