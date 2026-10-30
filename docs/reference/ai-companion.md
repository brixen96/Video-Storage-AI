# AI Companion — Architecture Reference

## 3-Layer Memory System ("Know Brix", 2026-10-30)

All local SQLite — nothing leaves the PC. NSFW-aware by design: this is an adult
content library app, so tastes/boundaries are first-class profile sections and
both extraction prompts are told to capture them honestly.

1. **Core profile** — `user_profile` memory (category `profile`, importance 10,
   source `profile`), seeded with a section skeleton (Tastes / Favorite
   performers / Go-to sites / Boundaries / Communication style / Projects).
   Rendered by `GetProfileBlock()` as `## About Brix` in EVERY system prompt
   (both `ChatLLM` and `ChatLLMStream`), capped at 6000 chars. Editable in
   AIPage → Memory tab (dedicated card, PUT `/ai/memories/:id`).
2. **Relevant recall** — `GetRelevantMemoriesBlock(message)` runs on every
   turn: sanitizes the message into a safe FTS5 OR-query (raw text would be
   implicit AND → recall would silently die), takes top 5 by rank+importance,
   injects `## Relevant memories`, bumps `hit_count`/`last_used_at`.
   Excludes `app_profile`/`user_profile` (already injected separately).
3. **Auto-capture** — `ExtractMemoriesAfterTurn(user, assistant)` fires async
   after EVERY main-scope turn (handler, post-persist; RP scope never
   extracted — roleplay dialogue ≠ real taste). Lightweight LLM pass returns
   compact JSON facts → rows with `source:'auto'`, keys slugged `auto_*` so
   repeats refresh instead of duplicating. Skips exchanges < 40 chars.

**Consolidation** (`runMemoryConsolidation`, every 6h, independent of
supervisor autonomy mode): decays importance −1 for memories unused 30d
(floor 1, never `app_profile`/`user_profile`/importance ≥ 8); promotes
memories with `hit_count ≥ 3` into the profile via LLM merge; distills recent
browser downloads/history (via the bridge) into a `browser_patterns` memory
(`source:'browser'`, distilled sentences — never raw URL dumps).

**Migration 70**: `memories` += `source` (manual/auto/browser/profile),
`hit_count`, `last_used_at`. New file: `ai_companion_service_memory.go`.

**Superseded approaches:**
- Passive-only memory (model must call `recall_memories` to remember anything)
  — replaced by automatic per-turn injection; the tool still exists for deep search.
- Raw FTS match on user text — implicit-AND semantics broke recall; replaced by
  sanitized OR-query (`ftsOrQuery`).

## Composable Map (`src/composables/ai/`)
| File | Responsibility |
|---|---|
| constants.js | Static data: prompts, commands, labels, rules, welcome message |
| useChatHistory.js | Message state, persistence, search, stats, prompt history |
| useChatStreaming.js | SSE streaming, tool progress, abort, follow-up derivation |
| useChatInput.js | Textarea, slash commands, @mentions, keyboard |
| useChatUI.js | Panels, memory, context stats, scrolling, shortcuts |
| useEntityContext.js | Page detection, entity cards, greetings |
| useVideoCards.js | Rating, favorite, navigate, compare |
| useVoiceInput.js | Web Speech API |

Component: `AICompanionChat.vue` — 919 lines (refactored down from 1,830; template 531, script 387).

## Key Design Decisions (refactor 2026-06-16)
- `useChatInput.handleInputKeydown` takes callbacks via options object, component wraps with `sendCompanionMessage`
- `useEntityContext.setupWatchers` takes refs as args (chatHistory, unreadCount, isOpen) to avoid circular deps
- `useChatUI.toggleChat` is a setter from outside, wired after construction
- `reactToMessage` uses dynamic import for `aiAPI.saveMemory` to avoid circular dependency

## Bannerlord RP Mini-Chat (`CompanionMiniChat.vue`, 2026-10-13/14 redesign)
- **Separate RP thread (memory)**: mini-chat has its OWN `rpHistory` ref (NOT the shared `useChatHistory` singleton). Persists via `scope='rp'` — stream request carries `{scope}`, migration 69 added `ai_chat_messages.scope` (default 'main'), `GetChatHistoryScoped`/`SaveChatMessageScoped`/`ClearChatHistoryScoped` filter it, prune is per-500-per-scope. Loads on mount via `aiAPI.getRpHistory()`. Main chat Clear never touches RP.
- **Auto → Manual**: file changes NO LONGER make the Companion act. `BannerlordPanel` only reloads the dialogue view and passes `npcResponded` (newest entry is the NPC's) as a prop; the header **"NPC Responded!"** button is the manual trigger (old "Generate response" button + Auto toggle removed). Button pulses green (`has-update`) until clicked or the NPC turn passes.
- **Modes → Add-ons**: `addOn` ref (`''` = normal chat, the default; fresh localStorage key `rp_companion_addon`). Chips: Improve / OOC / Plan — **NO "Write" chip and NO "Chat" chip** (both Emil decisions): writing the next in-game response is the DEFAULT plain-chat behavior — the default framing tells the model to switch into RP style when asked; nothing highlighted = normal chat.
- **Length**: dynamic only — min-len input + Dynamic toggle removed. Cap is **per paste-able in-game text unit**: `MAX_RESPONSE_LENGTH=2048` hard cap + `TARGET_RESPONSE_LENGTH=1800` target (fixes drafts stalling at ~400 chars — verified in DB data, it was model default not a stale setting). Target line emitted for improve/autoreact/ooc AND for plain chat (chat replies uncapped, drafted in-game text targeted).
- **Single streaming bubble**: typing dots render INSIDE the empty streaming placeholder (`msg.streaming && !msg.content`) — the old standalone `v-if="isStreaming"` bubble caused a duplicate header-only bubble (same bug fixed in `AIPage.vue` chat tab).
- One-click generation survives as the ✍️ **Write reply** quick chip (sends through plain chat).

## Abandoned / Superseded Approaches (don't revisit)
- Background task queue — superseded by SSE streaming + progress callbacks
- `run_analysis_in_background` — superseded by proactive insights system
- WebSocket streaming for chat — SSE streaming used instead (WS hub exists for other features)
- Auto-react on file change (`autoReactKey` watcher) — acted on EVERY file change; superseded by the manual "NPC Responded!" button
- Companion modes as mutually-exclusive forced framing — superseded by add-ons with normal chat as default
- Min-length input + Dynamic toggle in the mini-chat — dynamic is now the only mode
- "Write" as an add-on — writing in-game responses is the default plain-chat behavior (Emil: shouldn't need a toggle)
- Mini-chat sharing the `useChatHistory` singleton with the main chat — superseded by its own `rpHistory` + `scope='rp'` persistence
