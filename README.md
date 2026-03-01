# Pictago

**v0.1** — Minimal image host in Go with a Vue.js admin UI and REST API. Images are served at direct URLs; the UI supports uploads, per-user statistics, collections, user management (admin), and API keys. Auth is session-based (cookie) or API key; credentials, sessions, and upload records are stored in SQLite.

## Features

- **Upload** images only (JPEG, PNG, GIF, WebP). Max size configurable (default 10 MB).
- **Optional optimisation** at upload: resize (max 1920px), re-encode as JPEG.
- **Collections**: set a collection name at upload; files are stored under `/files/<user_id>/<collection>/<filename>` and listed with direct links. Search and delete per file; admin can delete entire collections.
- **Per-user stats**: file count, total size, and per-collection breakdown in the UI.
- **User management** (admin only): list users with stats, create/delete users, expand per-user collections and delete a collection (and all its files). Non-admin users cannot access user management. Change own password.
- **Sessions**: login at `/login`; session stored in DB and cookie. **Log out** at `/logout` clears the session and cookie.
- **API keys**: create keys in the UI (API keys tab); use header `X-API-Key: <key>` or `Authorization: Bearer <key>` to call the API without the UI.
- **Direct links** listed per user; images at `/files/*` are public.
- **Audit log**: all security-relevant actions (login, logout, user create/delete, file upload/delete, public file access, API key create/delete, password change) are written in [Elastic Common Schema (ECS)](https://www.elastic.co/guide/en/ecs/current/index.html) format to a configurable file (default `./data/audit.json`).
- **SQLite** for users, sessions, API keys, and upload metadata (bcrypt for passwords). User IDs are random 63-bit integers (no predictable id=1 for the first user).

## Run

```bash
go build -o pictago .
./pictago
```

Default: listen on `:8080`, data under `./data` (files in `./data/files`, DB at `./data/users.db`).

**First run:** default user `admin` / `admin` is created. **Change this password immediately** via the UI (Password tab). Only images are accepted; max upload size is configurable via env.

### Env

| Variable         | Default                  | Description                          |
|-----------------|--------------------------|--------------------------------------|
| `ADDR`          | `:8080`                  | Listen address                       |
| `DATA_DIR`      | `./data`                 | Data directory                       |
| `FILES_DIR`     | `$DATA_DIR/files`        | Upload directory                     |
| `DB_PATH`       | `$DATA_DIR/users.db`     | SQLite path                          |
| `AUDIT_LOG_PATH`| `$DATA_DIR/audit.json`   | Audit log file (ECS JSON lines)      |
| `MAX_UPLOAD_MB` | `10`                     | Max image size in MiB                |

## Routes

| Path        | Description                    |
|-------------|--------------------------------|
| `/`         | 404 (not found)                |
| `/login`    | Login page                     |
| `/logout`   | Log out, redirect to `/login`  |
| `/admin/`   | Admin UI (requires auth)       |
| `/api/*`    | REST API (see below)           |
| `/files/*`  | Public image files             |

## Auth

- **Login**: `POST /api/login` with JSON `{"username","password"}`. Sets session cookie and returns `{"username":"..."}`.
- **Logout**: `GET /logout` clears session and cookie, redirects to `/login`.
- **API keys**: send `X-API-Key: <key>` or `Authorization: Bearer <key>` on any API request instead of a session cookie. Create/revoke keys in the UI (API keys tab).

## API (session cookie or API key required except where noted)

- `GET /api/me` — current user `{"username":"..."}`.
- `POST /api/upload` — multipart: `file`, optional `collection` (name, default `default`), optional `optimize` (`true`/`false`). Images only. Returns `{"path":"/files/<user_id>/<collection>/<filename>"}`.
- `GET /api/files` — list current user’s uploads: `[{"id", "path", "size", "collection", "filename"}, ...]`.
- `GET /api/stats` — current user stats: `{"file_count", "total_size", "collection_count", "collections":[{ "name", "file_count", "total_size" }]}`.
- `DELETE /api/files/:id` — delete one upload (current user only).
- `GET /api/config` — `{"max_upload_bytes", "max_upload_mb"}`.
- `GET /api/keys` — list your API keys (prefix only).
- `POST /api/keys` — body `{"name"}` (optional). Creates key; returns `{"key":"ih_...","message":"..."}` — copy the key once.
- `DELETE /api/keys/:id` — revoke an API key.
- `POST /api/me/password` — body `{"current_password","new_password"}` change own password.

**Admin only:**

- `GET /api/users` — list users with `id`, `username`, `file_count`, `total_size` (id as string for large IDs).
- `POST /api/users` — body `{"username","password"}` create user.
- `DELETE /api/users/:id` — delete user and their upload records.
- `GET /api/users/:id/stats` — stats for that user (same shape as `/api/stats`).
- `DELETE /api/users/:id/collections/:name` — delete a collection and all its files for that user.

## UI

- Open `http://localhost:8080/admin/`. If not logged in, you are redirected to `/login`. Sign in with username/password.
- **Upload**: collection name (optional), drag or choose images, optional “Optimise”, see stats and direct links. Copy URL, open in new tab, or delete per file.
- **Users** (admin only): add/delete users; expand a user to see their collections and delete a collection (and all its files).
- **API keys**: create keys (optional name); copy the key once. Revoke keys from the list.
- **Password**: change your password.
- **Log out**: link in the header; clears session and redirects to login.

## Security

See [SECURITY.md](SECURITY.md) for threat model, mitigations, and operational recommendations (default credentials, HTTPS, rate limiting).

## License

MIT — see [LICENSE](LICENSE).
