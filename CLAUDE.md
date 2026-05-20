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

## Asset URLs — Always Use getAssetURL
Never build asset URLs as root-relative paths (e.g. `/assets/thumbnails/...`) — the dev proxy only covers `/api/**`, so root-relative asset paths hit the Vue dev server and 404.
Import `getAssetURL` from `@/services/api`. Encode filenames with special chars per-segment first:
```js
import { getAssetURL } from '@/services/api'
const url = getAssetURL(path.split('/').map(s => encodeURIComponent(s)).join('/'))
```

## Response Shape
Frontend `api.js` interceptor returns `response.data` (the HTTP body).
After `await videosAPI.getAll()`, `res` is already `{data: [...], total: N, message: ""}`.
Use `res.data` for the payload, not `res.data.data`.

## Windows Service Context
The backend runs as a Windows service (Session 0). Any process launched via `exec.Command` (e.g. `explorer.exe`) is invisible to the user's desktop.
**Never rely on exec.Command to show UI to the user.** Always return data to the frontend and let the browser handle display/clipboard.
Keep exec calls as best-effort (they work in dev/regular-process mode).

## Async Context Menu Actions — Toast Pattern
When an action hides the context menu immediately, show a loading toast before the `await`:
```js
const id = this.$toast.loading('Working...', subtitle)
try {
    const res = await someAPI.call()
    this.$toast.removeToast(id)
    this.$toast.success('Done', detail, 6000)
} catch (err) {
    this.$toast.removeToast(id)
    this.$toast.error('Failed', '', 5000)
}
```
