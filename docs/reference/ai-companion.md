# AI Companion — Architecture Reference

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
