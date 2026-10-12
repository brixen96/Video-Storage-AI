# Deployment — Reference

## Services & Ports
- Port **3000** = Web server (serves SPA frontend from `dist/`)
- Port **8080** = API server (Go backend, `/api/v1/*` + `/health`)
- Windows services: `VideoStorageAI-API`, `VideoStorageAI-Web` (Session 0 — need admin to stop/start)
- Binaries: `release/VideoStorageAI-API.exe`, `release/VideoStorageAI-Web.exe`
- **DO NOT change ports or dev server behavior**
- User builds with `build.bat` each time — NOT `npm run serve` / yarn serve

## build.bat
- Auto-increments patch version, rebuilds, restarts services
- Hardened (e33c776): verified service stop/start polling (30s/20s caps), TCP health check on :8080 + :3000, all error paths pause, stale-binary trap fixed
- After ANY Go change: rebuild + restart required, and verify the service actually restarted

## Stop/Start Gotchas
- `Stop-Process` doesn't work on services — use `sc stop` / `Stop-Service` with admin elevation

## Version System
- `VERSION` file at project root (auto-incremented by build.bat)
- `config.Version` set via `-ldflags` at build time, BUT that injection is unreliable → fallback: web server reads VERSION file from `release/`
- Health endpoint returns `config.Version` as `version` field (verify via API health endpoint, not just the file)

## Frontend Serving
- Web server injects `window.VSAI_API_URL = "http://localhost:8080"` into index.html
- Dev proxy only covers `/api` — **`/health` is NOT proxied**, must use full API URL
- If browser shows "CORS error" on connection failures, check both services are actually running — the real cause is usually connection refused
