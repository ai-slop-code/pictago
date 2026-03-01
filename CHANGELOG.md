# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

- **Audit log**: all actions (user create/login/logout, file upload/delete, public file access, API key create/delete, password change, admin user/collection delete) are logged in ECS (Elastic Common Schema) format to a configurable file (`AUDIT_LOG_PATH`, default `./data/audit.json`). One JSON object per line; suitable for ingestion into Elasticsearch or other ECS-compatible tools.

## [0.1.0] - 2025-03-01

Initial release.

- Image upload with optional resize/optimize (JPEG, PNG, GIF, WebP)
- Per-user collections and direct links; search and delete files
- Admin UI at `/admin/` (session or API key auth)
- User management (admin): create/delete users; view and delete collections per user
- API keys: create and revoke in UI; use `X-API-Key` or `Bearer` header
- Change password; logout
- SQLite for users, sessions, API keys, upload metadata; random user IDs
- Root path `/` returns 404; login at `/login`, files at `/files/*`
