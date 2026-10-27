# Browser Extension (Firefox Bridge)

Lets the AI Companion see and work with Emil's **real Firefox**: current page,
open tabs, downloads, history, bookmarks — plus actions (navigate, click, type,
scroll). The user keeps his normal Firefox: real logins, bookmarks, extensions.
The extension only ever talks to the local backend (`localhost:8080`).

## Architecture

```
Firefox (MV3 extension)                     Go backend                      Vue frontend
extension/background.js  --WebSocket-->  services/browser_bridge.go
extension/content.js     <---commands---     |  request/reply (correlation ids)
        |                                   |  caches: tab state, frame, bookmarks
        |                                   |  broadcasts browser_state -> websocket hub
        |                                   +--> ai_companion_tools_browser.go (10 tools)
        |                                   +--> api/browser_ws_handler.go (/api/v1/browser/*)
        |                                                        |
        |                                                        +--> useBrowserView.js -> AIPage "Browser" tab
        |                                                              (live screenshot every 3.5s while open)
```

Files:

| File | Role |
|---|---|
| `extension/manifest.json` | MV3, permissions, gecko id, data-collection declaration |
| `extension/background.js` | WebSocket bridge, tab tracking, command executor, alarms keep-alive |
| `extension/content.js` | DOM snapshots with CSS selectors + click/type/scroll |
| `api/internal/services/browser_bridge.go` | Conn manager, pending-request replay, state/frame/bookmark cache |
| `api/internal/services/ai_companion_tools_browser.go` | The 10 `browser_*` tools |
| `api/internal/api/browser_ws_handler.go` | `/browser/ws`, `/browser/status`, `/browser/screenshot` (GET cached, POST fresh) |
| `src/composables/ai/useBrowserView.js` | Frontend live view state |
| `src/views/AIPage.vue` | Browser tab (7th tab) |

## Message protocol

Extension → backend (JSON envelopes):

- `{"type":"hello","data":{version,ua}}` — sent on connect + every keepalive alarm
- `{"type":"tab_state","data":{tabs:[BrowserTabInfo],active}}` — debounced 300ms on every tab change
- `{"type":"bookmarks","data":[BrowserBookmark]}` — pushed ≤1s after any bookmark change
- `{"type":"result","id":"brN","ok":true,"data":...}` / `{"ok":false,"error":"..."}` — reply to a command

Backend → extension:

- `{"type":"command","id":"brN","action":"...","args":{...}}`

Pending commands younger than 20s are **replayed on reconnect** — an MV3
background-suspension blip cannot eat an in-flight AI action. Commands time out
at 8–12s with a retry-friendly error.

## Tools (all `browser_*`, registered via `registerBrowserTools()`)

| Tool | Kind | Notes |
|---|---|---|
| `browser_status` | read | connected? active tab, open tabs, last error |
| `browser_read` | read | DOM snapshot (≤600 elements, selectors) + body text; also captures screenshot for the Browser panel |
| `browser_downloads` | read | Firefox `downloads.search`, newest first |
| `browser_history` | read | history search, days/limit |
| `browser_bookmarks` | read | local cache (3000 newest), auto-fills via `bookmarks_get` |
| `browser_navigate` | action | URL, bare domain → https, phrase → DuckDuckGo search |
| `browser_history_nav` | action | back / forward / reload |
| `browser_click` | action | CSS selector from a fresh `browser_read` |
| `browser_type` | action | native value setter (React/Vue safe), `submit` uses `form.requestSubmit()` |
| `browser_scroll` | action | page-level: down/up/top/bottom |

**Safety**: `supervisorProposalTools` is an allowlist — browser tools are
therefore never auto-proposed by the Companion Supervisor. Tool descriptions
instruct: act only on what Brix asked for, never confirm purchases/deletions/
account changes unless explicitly told, `submit=true` only when asked.

## Browser tab (frontend)

- Status badge (extension connected/offline), manual Refresh, live capture every 3.5s while the tab is open
- Screenshot from `POST /api/v1/browser/screenshot` (fresh capture) — `GET` serves the cached frame
- Side panel: active page title/URL + open tab list
- `browser_state` arrives over the existing `/api/v1/ws` hub (case added in `websocket.js`)

## Gotchas (learned the hard way)

- **Firefox MV3 background is an event page**, not a service worker: use
  `"background": {"scripts": [...]}`. It suspends → `browser.alarms` (0.5min)
  reconnects; backend replays recent pending commands on reconnect.
- **`downloads.query` does not exist in Firefox** — use `downloads.search`.
  web-ext lint catches this.
- **MV3 requires** `browser_specific_settings.gecko.id` and
  `data_collection_permissions` (since Nov 2025). Data to localhost still
  leaves the browser → declare honestly: browsingActivity, websiteContent,
  websiteActivity, bookmarksInfo, searchTerms. `strict_min_version` 140 +
  `gecko_android` 142 (Mozilla's recommended combo, lint-clean).
- **`captureVisibleTab`** only captures the *active* tab of a window and fails
  on privileged pages (`about:*`, `moz-extension:`) — errors surface as friendly
  tool messages, last good frame is kept.
- **Selectors go stale after navigation** — tool descriptions force
  `browser_read` before acting.
- **Content script injection fallback**: pages loaded before install get
  `browser.scripting.executeScript` on first command.
- **web-ext lint = the verification gate** (0 errors / 0 warnings), same spirit
  as `go build` + `npm run build`.

## Testing (dev)

1. Backend + frontend running (or services up).
2. `about:debugging` → This Firefox → Load Temporary Add-on → pick `extension/manifest.json`.
3. Console of background shows nothing fatal; `GET :8080/api/v1/browser/status` shows `connected:true`.
4. AI → Browser tab shows live screenshots; ask the Companion "what page am I on?".

## Release / signing (AMO unlisted)

1. `npx web-ext build` → `extension/web-ext-artifacts/*.zip` (gitignored).
2. AMO developer hub (Mozilla account, unlisted addon) → upload zip → review (usually fast for unlisted) → download signed `.xpi`.
3. Install the `.xpi` in Firefox once (permanent, survives restarts; temporary loads do not).
4. Rebuild zip on every extension change and re-upload as a new version.

Override the backend URL if needed: `browser.storage.local.set({ bridgeUrl: "ws://host:port/api/v1/browser/ws" })`.
