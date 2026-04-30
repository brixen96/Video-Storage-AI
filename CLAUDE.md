# Video Storage AI

Self-hosted video library manager. Vue 3 frontend + Go (Gin) backend + SQLite.

## Build Commands
```bash
# Frontend
cd "c:\Repos\Video Storage AI" && npm run build

# Backend
cd "c:\Repos\Video Storage AI\api" && go build ./...
```
Always build both after changes to verify no errors before reporting done.

## Architecture
- **Frontend**: Vue 3, Options API for pages (`src/views/`), Composition API for components (`src/components/`)
- **Backend**: Go + Gin, routes in `api/internal/api/router.go`, handlers split by domain
- **Database**: SQLite via `database/sql`, migrations in `api/internal/database/database.go`
- **Deployment**: Docker, `/output` volume for video storage

## Critical Patterns — Read Before Writing Any Backend Code

### 1. Lazy Service Initialization
Never `var svc = services.NewX()` at package level — DB isn't ready yet.
Always use the lazy init pattern:
```go
var mySvc *services.MyService
func ensureMySvc() *services.MyService {
    if mySvc == nil { mySvc = services.NewMyService() }
    return mySvc
}
```

### 2. All Handlers Must Use respondSuccess
The axios interceptor unwraps `response.data` automatically. `c.JSON(http.StatusOK, data)` breaks the frontend.
Always use helpers from `api/internal/api/respond.go`:
```go
respondSuccess(c, data, "optional message")
respondBadRequest(c, "message", err)
respondInternalError(c, "message", err)
```

### 3. FontAwesome Icons
Two steps required in `src/plugins/fontawesome.js`:
1. Add to the named import block
2. Add to `library.add(...)` call

Missing either step = silent render failure.

## Key File Locations
- Routes: `api/internal/api/router.go`
- Shared response helpers: `api/internal/api/respond.go`
- API service (frontend): `src/services/api.js`
- FontAwesome registry: `src/plugins/fontawesome.js`
- DB migrations: `api/internal/database/database.go`
- Thumbnail storage: `api/assets/thumbnails/{libraryID}/...` served at `/assets/thumbnails/...`
  - Configured via `THUMBNAIL_DIR=./assets/thumbnails` in `api/.env`
  - **Never pre-set `thumbnail_path` in DB before FFmpeg writes the file** — set empty on create, update only after `GenerateThumbnailHierarchical` succeeds

## Response Shape
Frontend `api.js` interceptor returns `response.data` (the HTTP body).
After `await videosAPI.getAll()`, `res` is already `{data: [...], total: N, message: ""}`.
Use `res.data` for the payload, not `res.data.data`.
