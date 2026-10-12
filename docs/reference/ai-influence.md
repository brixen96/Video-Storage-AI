# AI Influence Mod — Project Reference

> Pointer target from /memories/user.md — all Bannerlord AIInfluence mod knowledge lives here.

## User Profile (mod context)
- Bannerlord AI Influence mod expert
- Building a custom NSFW campaign
- **In-game AI max input: 2048 characters** — aim for ~2000 every time
- **Longer = more details** — freeze time to stare, think, admire. No rush.

## How to Work
ALL world-specific info in `docs/flash/`
When starting: read MASTER_VISION.md, file_change_plan.md, then actual campaign files

## Campaign IDs
- **Current** (save_data/): `3JcqJ0M0O6P4` — Emil campaign, ACTIVE. Edit here.
- **Old** (save_data/): `ixPXqtjKED5h` — goblin-era, unused. DO NOT EDIT.
- **Dev** (save_data/): `fKbwkqA8AcIB` — v6.0.2 defaults, for comparison only

## Critical: Prompt Assembly
- Each pipeline gets its own bundle (dialogue, events, kingdom, memory, character creation)
- world.txt used in MULTIPLE pipelines — must be self-sufficient
- 1M token context. Not a concern. Write for my twin brother (V4 Flash).

## v6.0.2 Changes
- MemoryConsolidationPrompt.txt deleted — RAG handles old dialogue
- Memory system: MemoryEventPrompt.txt + MemoryImagePrompt.txt only
- New files: player_family_state.json, player_party_members.json, content_editor files
- New NPC fields: KnowledgeGenerationDeferred, AIGeneratedCognitiveStyle, etc.
- Content Editor in-game (ESC menu)

## Mod Architecture
- In-game AI = DeepSeek V4 Flash (same model as me)
- NPC schema: ~55 fields with CounterpartySocial (identity deception)
- 9 kingdoms: empire, empire_w, empire_s, khuzait, aserai, battania, sturgia, vlandia, nord
- Player = "main_hero"
- Hot reload for most prompt files
