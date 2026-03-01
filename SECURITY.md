# Security

Summary of common attack vectors considered and how this codebase handles them. For reporting vulnerabilities, open an issue or contact the maintainers.

---

## Addressed in code

### 1. **Path traversal (directory escape)**

- **Risk:** Upload filename like `../../../etc/passwd` to write outside the store.
- **Mitigation:** `storage` uses `filepath.Base()` so only the last path segment is used. Filenames are further restricted to safe ASCII (see below).

### 2. **Filename injection (null bytes, control chars, path separators)**

- **Risk:** Null byte (`file.jpg\x00.php`) or slashes/backslashes to confuse parsers or path handling.
- **Mitigation:** `safeFilename()` in `internal/storage/storage.go` strips null bytes, control characters, and disallows `/ \ : * ? " < > |`. Only printable ASCII (0x20–0x7E) is allowed in stored filenames.

### 3. **Stored XSS via filenames**

- **Risk:** Filename like `"><script>alert(1)</script>.jpg` is returned in the API and rendered in the UI; if inserted into `innerHTML` unescaped, it runs in the browser.
- **Mitigation:**  
  - Server: filenames sanitized so angle quotes and quotes are not stored.  
  - UI: paths from the API are escaped with `escapeHtml()` before being used in `innerHTML` (link text and `href` attribute), so legacy or edge-case values cannot break out of the HTML.

### 4. **SQL injection**

- **Risk:** User input in auth used in raw SQL.
- **Mitigation:** All DB access uses parameterized queries (`?` placeholders). No string concatenation into SQL.

### 5. **Authentication**

- **Risk:** Weak or default credentials, credential stuffing.
- **Mitigation:** Passwords hashed with bcrypt (default cost). Default `admin`/`admin` is only created if no user exists; **change this in production**. No built-in rate limiting (see below).

### 6. **Upload size and resource exhaustion**

- **Risk:** Very large uploads exhaust memory or disk.
- **Mitigation:** `ParseMultipartForm(maxUploadBytes)` limits request body to 10 MiB. Whole file is still read into memory for optional optimisation; for very high concurrency, consider streaming to disk first.

### 7. **Content-type and script execution**

- **Risk:** Uploaded file served with wrong type or executed as code.
- **Mitigation:** Files under `/files/` are served with Go’s `FileServer` (static files only, no execution). `X-Content-Type-Options: nosniff` is set so browsers don’t reinterpret content type.

### 8. **Security headers**

- **Mitigation:**  
  - `X-Content-Type-Options: nosniff` on all responses.  
  - For UI (served at `/admin/` only): `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'self'; script-src 'self' https://unpkg.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'` to reduce clickjacking and XSS impact.

### 9. **Root path**

- **Mitigation:** `GET /` returns 404. The admin UI is only served at `/admin/`, reducing exposure of the app to casual discovery.

---

## Limitations / operational recommendations

### No rate limiting

- **Risk:** Brute force on Basic Auth; upload or API abuse.
- **Recommendation:** Put the app behind a reverse proxy (e.g. nginx, Caddy) or middleware that rate-limits by IP for `/api/upload` and the login (e.g. 401 responses).

### Default credentials

- **Risk:** First run creates `admin`/`admin`; if not changed, anyone who can reach the UI can log in.
- **Recommendation:** Change the default password immediately after first deploy, or seed the DB with a strong password and remove/disable the default-creation logic in production.

### No audit logging

- **Risk:** Hard to detect abuse or investigate incidents.
- **Recommendation:** Log failed logins and uploads (at least method, path, and client IP) to a log aggregator.

### File type not enforced

- **Risk:** Non-image files can be uploaded and served from `/files/` (no execution; still uses disk and can be used for phishing or abuse).
- **Recommendation:** Optional: validate image magic bytes (e.g. JPEG/PNG/GIF headers) and reject non-image uploads, or enforce a strict allowlist of extensions/content types.

### Basic Auth over HTTP

- **Risk:** Credentials sent in clear text if TLS is not used.
- **Recommendation:** Use HTTPS in production (reverse proxy or Go with TLS). Do not rely on Basic Auth over plain HTTP for sensitive environments.

---

## Quick checklist

- [ ] Change default `admin` password (or remove default user and seed your own).
- [ ] Run behind HTTPS (e.g. reverse proxy with TLS).
- [ ] Add rate limiting (proxy or middleware) for login and upload.
- [ ] Ensure `DATA_DIR` / `FILES_DIR` are not under web root of other apps and have correct permissions.
- [ ] Monitor disk usage and consider caps or cleanup for `/files/`.
