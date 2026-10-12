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

## Abandoned / Superseded Approaches (don't revisit)
- Background task queue — superseded by SSE streaming + progress callbacks
- `run_analysis_in_background` — superseded by proactive insights system
- WebSocket streaming for chat — SSE streaming used instead (WS hub exists for other features)
