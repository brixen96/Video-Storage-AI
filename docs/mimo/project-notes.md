# Video Storage AI — Project Notes

> Last updated: 2026-06-08

## Quick Reference

### User Info

-   **User OS**: Windows
-   **Workspace**: `c:\Repos\Video Storage AI`

### Tech Stack

-   **Frontend**: Vue 3 (Options API views, Composition API components), Bootstrap 5, FontAwesome 7, Axios
-   **Backend**: Go + Gin, SQLite (WAL mode), WebSocket (gorilla/websocket)
-   **Build**: `npm run build` (frontend), `go build ./...` (backend)
-   **Dev Server**: Frontend :8081, Backend :8080

### Critical Rules (from CLAUDE.md)

1. **Lazy service init** — Never `var svc = services.NewX()` at package level
2. **Always use `respondSuccess`** / `respondBadRequest` / `respondInternalError` from `api/internal/api/respond.go`
3. **FontAwesome** — Two-step: import + `library.add()`
4. **Asset URLs** — Always `getAssetURL()` from `@/services/api`
5. **Response shape** — Axios interceptor unwraps `response.data`, so `res.data` is the payload
6. **No `exec.Command` for UI** — Windows service context (Session 0)
7. **Toast pattern** — Loading → success/error for async context menu actions

### Scale

-   21 views, 35+ components, 47 handler files, 67 service files, 20 model files
-   200+ API endpoints, 30+ DB tables, 8+ scrapers

### Key File Locations

-   Routes: `api/internal/api/router.go`
-   Response helpers: `api/internal/api/respond.go`
-   API client: `src/services/api.js`
-   FontAwesome: `src/plugins/fontawesome.js`
-   DB migrations: `api/internal/database/database.go`
-   Thumbnails: `api/assets/thumbnails/{libraryID}/`
