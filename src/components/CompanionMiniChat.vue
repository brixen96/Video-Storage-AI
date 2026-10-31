<template>
	<div class="mini-chat">
		<div class="mini-chat-header">
			<font-awesome-icon :icon="['fas', 'robot']" class="me-1" />
			AI Companion
			<span v-if="contextNpc" class="mini-context-chip">
				<font-awesome-icon :icon="['fas', 'mask']" class="me-1" />
				{{ contextNpc.name }}
			</span>
			<span v-for="lk in linkedNpcs" :key="lk.file_path" class="mini-context-chip linked">
				<font-awesome-icon :icon="['fas', 'users']" class="me-1" />
				{{ lk.name }}
				<button class="mini-chip-x" title="Remove from scene" @click="removeLinked(lk)">×</button>
			</span>
			<select
				v-if="availableNpcs.length > 1"
				v-model="addNpcSel"
				class="mini-npc-select"
				title="Link another NPC into this conversation (group scenes)"
				:disabled="isStreaming"
			>
				<option value="" disabled>+ NPC</option>
				<option v-for="n in linkableNpcs" :key="n.file_path" :value="n.file_path">{{ n.name }}</option>
			</select>
			<button
				v-if="contextNpc"
				class="btn btn-sm btn-info mini-generate"
				:class="{ 'has-update': hasUpdate }"
				:disabled="isStreaming"
				:title="npcRespondedTitle"
				@click="onNpcResponded"
			>
				<font-awesome-icon :icon="['fas', 'bolt']" class="me-1" />
				NPC Responded!
			</button>
			<span v-if="isStreaming" class="mini-thinking">
				<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-1" />
				thinking...
			</span>
		</div>

		<div ref="msgsRef" class="mini-chat-messages">
			<div v-if="visibleMessages.length === 0 && !isStreaming" class="mini-empty">
				<p>Chat with the Companion normally — or switch on an add-on below for roleplay help.</p>
			</div>
			<div v-for="(msg, i) in visibleMessages" :key="i" class="mini-msg" :class="msg.role">
				<div class="mini-role">{{ msg.role === 'user' ? 'Emil' : 'Companion' }}</div>
				<div class="mini-content">
					<!-- Single bubble: typing dots live INSIDE the streaming message (a standalone second bubble used to appear alongside it) -->
					<span v-if="msg.role === 'assistant' && msg.streaming && !msg.content" class="typing-indicator"><span></span><span></span><span></span></span>
					<template v-else>{{ msg.content }}</template>
				</div>
			</div>
		</div>

		<!-- Persistent quick actions (often-used prompts) -->
		<div v-if="contextNpc" class="mini-quick-row">
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickWrite">✍️ Write reply</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Analyze the mood of everyone here and what it means for me')">🧠 Mood</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Give me 3 distinct response options with a one-line summary each')">💡 3 options</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Summarize the scene so far in 3-4 sentences')">📜 Scene summary</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Suggest the next beat of this scene — where could it go from here?')">🗺️ Next beat</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('What is this NPC likely to do or say next, and why?')">🔮 Predict</button>
		</div>

		<!-- Add-ons: optional roleplay behavior on top of normal chat. Nothing highlighted = plain conversation. -->
		<div class="mini-modes" title="Add-ons — none active = normal chat. Click to toggle on, click again to turn off">
			<span class="mini-modes-label">Add-ons</span>
			<button
				v-for="m in addOns"
				:key="m.id"
				class="mini-mode-chip"
				:class="{ active: addOn === m.id }"
				@click="toggleAddOn(m.id)"
				:title="m.hint"
			>
				<font-awesome-icon :icon="['fas', m.icon]" class="me-1" />
				{{ m.label }}
			</button>
		</div>

		<!-- Focus: elements to weave into generated drafts (scene-fitting) -->
		<div v-if="contextNpc" class="mini-modes mini-focus" title="Focus — active chips are woven into generated drafts where the scene fits. Never forced.">
			<span class="mini-modes-label">Focus</span>
			<button
				v-for="c in focusChips"
				:key="c"
				class="mini-mode-chip focus-chip"
				:class="{ active: focusActive.includes(c) }"
				@click="toggleFocus(c)"
				:title="focusActive.includes(c) ? 'Active — click to deactivate' : 'Click to weave this into drafts'"
			>
				{{ c }}
				<span v-if="isCustomFocus(c)" class="focus-x" title="Remove chip" @click.stop="removeFocusChip(c)">×</span>
			</button>
			<input
				v-if="showFocusAdd"
				ref="focusAddRef"
				v-model="focusAddText"
				class="mini-focus-add"
				placeholder="add chip…"
				@keyup.enter="addFocusChip"
				@keyup.esc="showFocusAdd = false"
			/>
			<button v-else class="mini-focus-plus" title="Add your own focus chip" @click="openFocusAdd">＋</button>
		</div>

		<div class="mini-chat-input">
			<input
				v-model="input"
				type="text"
				class="form-control form-control-sm"
				:placeholder="modePlaceholder"
				@keyup.enter="send"
				:disabled="isStreaming"
			/>
			<button v-if="isStreaming" class="btn btn-sm btn-danger" title="Stop" @click="stopStreaming">
				<font-awesome-icon :icon="['fas', 'stop']" />
			</button>
			<button v-else class="btn btn-sm btn-primary" :disabled="!input.trim()" title="Send" @click="send">
				<font-awesome-icon :icon="['fas', 'paper-plane']" />
			</button>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useChatStreaming } from '@/composables/ai/useChatStreaming'
import { aiAPI, bannerlordAPI } from '@/services/api'

const props = defineProps({
	// When set, the Companion is told which NPC conversation is open
	contextNpc: { type: Object, default: null },
	// Full roster — enables linking additional NPCs into group scenes
	availableNpcs: { type: Array, default: () => [] },
	// True when the newest in-game dialogue entry is the NPC's — lights up the
	// "NPC Responded!" button. Nothing fires automatically; Brix clicks it.
	npcResponded: { type: Boolean, default: false },
})

// The mini-chat keeps its OWN history ref (rpHistory) rather than the shared
// singleton from useChatHistory, so RP chats and the main companion chat never
// share a timeline. Persisted to the backend 'rp' thread — see loadRpHistory.
const rpHistory = ref([])
const pushMessage = (msg) => {
	rpHistory.value.push(msg)
	return msg
}
const { isStreaming, streamMessage, stopStreaming } = useChatStreaming()

// Load the persisted RP thread from the backend on mount.
const loadRpHistory = async () => {
	try {
		const res = await aiAPI.getRpHistory(100)
		if (res?.data?.length) {
			rpHistory.value = res.data.map((m) => ({
				role: m.role,
				content: m.content,
				timestamp: new Date(m.created_at),
				source: m.role !== 'user' ? m.source : undefined,
				toolSteps: m.tool_steps ? JSON.parse(m.tool_steps) : undefined,
			}))
		}
	} catch { /* backend down / no history yet — start empty */ }
}

onMounted(() => {
	loadRpHistory()
})

const input = ref('')
const msgsRef = ref(null)

// ── Add-ons: optional roleplay behavior layered on NORMAL CHAT ──
// '' (no add-on) = plain conversation — no RP framing, no character limits.
// NOTE: there is deliberately NO "Write" add-on. Writing the next in-game
// response is the DEFAULT the Companion does when you just ask for it in plain
// chat — it shouldn't need a toggle. These add-ons are extra behaviors only.
const addOns = [
	{ id: 'improve', label: 'Improve', icon: 'magic', hint: 'Add-on: improve a draft Brix wrote himself + feedback' },
	{ id: 'ooc', label: 'OOC', icon: 'comment-dots', hint: 'Add-on: help write an OOC (out-of-character) message' },
	{ id: 'plan', label: 'Plan', icon: 'route', hint: 'Add-on: plan a scene or scenario with this NPC' },
]
// Fresh key (old 'rp_companion_mode' is intentionally not migrated) so the
// first launch after this change lands in normal chat. Unknown stored values
// (e.g. a stale 'write' from before that add-on was removed) are dropped.
const VALID_ADDONS = addOns.map((m) => m.id)
const savedAddOn = localStorage.getItem('rp_companion_addon') || ''
const addOn = ref(VALID_ADDONS.includes(savedAddOn) ? savedAddOn : '')
watch(addOn, (v) => localStorage.setItem('rp_companion_addon', v))

const toggleAddOn = (id) => {
	addOn.value = addOn.value === id ? '' : id
}

// ── Focus chips: elements the Companion weaves into generated drafts ──
// Multi-select, persisted. Interpretation is scene-driven (the FOCUS line in
// buildContextBlock explains how to use each naturally). Custom chips editable.
const DEFAULT_FOCUS = ['Tits', 'High Heels', 'Eyes', 'Legs', 'Lips']
const loadJSON = (key, fallback) => {
	try {
		const v = JSON.parse(localStorage.getItem(key))
		return Array.isArray(v) ? v : fallback
	} catch { return fallback }
}
const focusChips = ref(loadJSON('rp_focus_chips', DEFAULT_FOCUS))
const focusActive = ref(loadJSON('rp_focus_active', []))
watch(focusChips, (v) => localStorage.setItem('rp_focus_chips', JSON.stringify(v)), { deep: true })
watch(focusActive, (v) => localStorage.setItem('rp_focus_active', JSON.stringify(v)), { deep: true })
const toggleFocus = (chip) => {
	focusActive.value = focusActive.value.includes(chip)
		? focusActive.value.filter((x) => x !== chip)
		: [...focusActive.value, chip]
}
const isCustomFocus = (chip) => !DEFAULT_FOCUS.includes(chip)
const removeFocusChip = (chip) => {
	focusChips.value = focusChips.value.filter((x) => x !== chip)
	focusActive.value = focusActive.value.filter((x) => x !== chip)
}
const showFocusAdd = ref(false)
const focusAddText = ref('')
const focusAddRef = ref(null)
const openFocusAdd = () => {
	showFocusAdd.value = true
	nextTick(() => focusAddRef.value?.focus())
}
const addFocusChip = () => {
	const t = focusAddText.value.trim()
	if (t && !focusChips.value.includes(t)) focusChips.value.push(t)
	focusAddText.value = ''
	showFocusAdd.value = false
}

const modePlaceholder = computed(() => {
	const npcName = props.contextNpc?.name || 'the NPC'
	switch (addOn.value) {
		case 'improve': return 'Paste your draft response here...'
		case 'ooc': return 'What do you want to say OOC?'
		case 'plan': return 'Describe the scene direction you want...'
		default: return props.contextNpc ? `Ask ${npcName} anything, or just request your next response...` : 'Chat with the Companion...'
	}
})

// Response length: DYNAMIC only (min-len input + toggle were removed), now with
// an explicit TARGET plus a FLOOR — drafts were stalling at a uniform short
// length, so generated in-game responses must land in [MIN, MAX] aiming at
// TARGET. Hard cap = 2048 chars (the in-game text input limit) — scoped to
// in-game Bannerlord responses only (see buildContextBlock).
const MAX_RESPONSE_LENGTH = 2048
const TARGET_RESPONSE_LENGTH = 1800
const MIN_RESPONSE_LENGTH = 1600

// ── Manual NPC-responded trigger (replaced auto-react) ──
// Auto-react used to fire on EVERY file change; now nothing runs on its own.
const reacted = ref(false)
watch(() => props.npcResponded, (v) => { if (!v) reacted.value = false })
watch(() => props.contextNpc?.file_path, () => { reacted.value = false })

const hasUpdate = computed(() => props.npcResponded && !reacted.value)

const npcRespondedTitle = computed(() => {
	if (hasUpdate.value) return 'The NPC just answered in-game — analyze their reply and draft your reply (manual: nothing runs automatically)'
	return 'Analyze the current conversation and draft your reply (manual trigger)'
})

// ── Linked NPCs (group scenes) ──
const linkedNpcs = ref([])
const addNpcSel = ref('')

const linkableNpcs = computed(() => {
	const taken = new Set(allNpcs.value.map((n) => n.file_path))
	return props.availableNpcs.filter((n) => !taken.has(n.file_path))
})

const allNpcs = computed(() => {
	const list = []
	const seen = new Set()
	for (const n of [props.contextNpc, ...linkedNpcs.value]) {
		if (n && !seen.has(n.file_path)) {
			seen.add(n.file_path)
			list.push(n)
		}
	}
	return list
})

watch(addNpcSel, (path) => {
	if (!path) return
	const npc = props.availableNpcs.find((n) => n.file_path === path)
	if (npc) linkedNpcs.value.push(npc)
	addNpcSel.value = ''
})

const removeLinked = (npc) => {
	linkedNpcs.value = linkedNpcs.value.filter((n) => n.file_path !== npc.file_path)
}

// ── Full RP context cache (per NPC file path) ──
const npcContexts = ref({})

watch(
	() => allNpcs.value.map((n) => n.file_path).join('|'),
	async () => {
		for (const npc of allNpcs.value) {
			if (npcContexts.value[npc.file_path]) continue
			try {
				const res = await bannerlordAPI.getRpContext(npc.file_path)
				if (res?.data) {
					npcContexts.value = { ...npcContexts.value, [npc.file_path]: res.data }
				}
			} catch { /* NPC stays context-less; block degrades gracefully */ }
		}
	},
	{ immediate: true }
)

// Build the RP context block. Sent as a SYSTEM-role history entry (not part of
// the user message) so persisted chat history stays clean and readable.
// Knowledge scope: ONLY the conversation dialogue + mechanical state (mood/relation).
// Backstory/personality/world lore are deliberately EXCLUDED — no cheating.
// Supports multiple linked NPCs (group scenes): one section per NPC.
const buildContextBlock = (opts = {}) => {
	if (allNpcs.value.length === 0) return ''
	const frameMode = opts.mode || addOn.value
	// No add-on (or explicit 'chat') = normal conversation: context is provided
	// but no RP output rules, no RP rules block, no character cap.
	const isChat = !frameMode || frameMode === 'chat'
	const primaryCtx = npcContexts.value[allNpcs.value[0].file_path] || null
	const lines = []
	const campaign = primaryCtx?.campaign_id || 'current'
	lines.push(`[ROLEPLAY CONTEXT — Bannerlord AIInfluence campaign ${campaign}]`)

	// Framing per add-on — '' (no add-on) = normal chat
	switch (frameMode) {
		case 'autoreact':
			lines.push(`Brix plays Emil. ${opts.autoReactNpc || 'An NPC'} just responded in-game (the newest dialogue entry). Do TWO things: (1) give quick FEEDBACK on their response — 2-3 sentences on tone, subtext, and what they really want (2-4 bullets); (2) then draft Brix's NEXT REPLY in the RP style — *asterisks for actions/thoughts*, "quotes" for speech, no preamble.`)
			break
		case 'improve':
			lines.push(`Brix plays Emil. He wrote a response HIMSELF (his latest message below is the draft). IMPROVE it: keep his intent, voice, and plot beats — fix flow, grammar, and adherence to the campaign RP rules below. Output in TWO parts: (1) the improved version ONLY — no preamble — using *asterisks for actions/thoughts* and "quotes" for speech; (2) a line '---' then 2-4 short feedback bullets on what you changed and why.`)
			break
		case 'ooc':
			lines.push(`Brix plays Emil. He wants to send an OOC (out-of-character) message to the game's AI per the campaign's OOC protocol (see rules below). Help him write it clearly and completely. Output ONLY the final OOC message text, formatted as [OOC: ...] — no preamble, no explanation.`)
			break
		case 'plan':
			lines.push(`Brix plays Emil. He wants to PLAN the next scene/scenario with the NPC(s) below. Based ONLY on the conversation(s) so far and the campaign rules, propose 2-3 distinct scene directions with brief beats for each, then recommend one. Be concrete about what could happen moment to moment. Output the plan only — no preamble.`)
			break
		default:
			// Normal chat — no add-on active. This is the DEFAULT, and it does
			// double duty: casual conversation AND (when Brix asks) drafting the
			// next in-game response. No toggle needed for writing.
			lines.push(`Brix plays Emil. You are chatting with him about the NPC(s) below. Answer conversationally. If he ASKS you to write/draft his next in-game response (or anything to paste into the game), switch into the RP style for that piece: use *asterisks for actions and inner thoughts* and "quotes" for speech, exactly like the dialogue examples, no preamble — but keep chatting normally around it.`)
			break
	}

	// POV — the standing rule: the mini companion writes ONLY for the player.
	lines.push(`POV: You write ONLY for Emil (the player character). You never write AS an NPC, never draft NPC dialogue for the game — NPCs speak through the game itself — and every piece of in-game text you draft is from Emil's first-person perspective. Telling Brix what an NPC is likely to do or say next (prediction) is fine; writing their reply FOR the game is not.`)

	lines.push(`FACTS: The conversation(s) below are the ONLY source of what has been said and what the NPC(s) know or have heard. Do NOT invent or reference backstory, rumors, personality traits, or world facts. Nothing exists except what is in the dialogue and the mechanical states below.`)

	// Length — DYNAMIC only (min-len input + toggle were removed), but with an
	// explicit TARGET so the Companion doesn't default to ~400 chars. Game input
	// limit is 2048: the companion's MESSAGE to Brix is always uncapped; only
	// each individual piece of in-game text is capped.
	if (!isChat && ['improve', 'autoreact', 'ooc'].includes(frameMode)) {
		if (frameMode === 'ooc') {
			lines.push(`LENGTH: your reply to Brix is NOT limited as a whole — explain, give feedback, and draft freely. But EVERY OOC message he will paste into the game must aim for ${TARGET_RESPONSE_LENGTH} characters and NEVER exceed ${MAX_RESPONSE_LENGTH} — the game's text input limit. Rich and complete — no padding, but never skimp either.`)
		} else {
			lines.push(`LENGTH: your reply to Brix is NOT limited as a whole — explain, give feedback, and draft freely. But EVERY in-game response you draft for him must aim for ${TARGET_RESPONSE_LENGTH} characters, NEVER fall below ${MIN_RESPONSE_LENGTH}, and NEVER exceed ${MAX_RESPONSE_LENGTH} — the game's text input limit. If you draft several options or responses for multiple NPCs, apply that to EACH. Rich, detailed, immersive responses — expand beats and detail to reach the floor naturally, never pad.`)
		}
	} else if (isChat) {
		// In plain chat the Companion may still be asked to draft a response —
		// give it the same target + floor so a "write my next response" ask
		// doesn't come back as a stunted stub.
		lines.push(`LENGTH: any in-game response you draft for Brix (even in plain chat) must aim for ${TARGET_RESPONSE_LENGTH} characters, NEVER fall below ${MIN_RESPONSE_LENGTH}, and NEVER exceed ${MAX_RESPONSE_LENGTH} — the game's text input limit. OOC messages and your casual chat replies have no minimum. Rich, detailed, immersive responses — expand to reach the floor naturally, never pad.`)
	}

	// Focus chips — elements to weave into drafts where the scene fits
	if (focusActive.value.length) {
		lines.push(`FOCUS: when drafting in-game text, weave in 1-3 natural touches of ${focusActive.value.join(', ')} — only where the scene supports it, never forced. Interpret each to fit the moment: e.g. 'Tits' could be his gaze landing there, a touch, a massage; 'Eyes' could be eye contact or a look exchanged. Fitting the scene beats forcing the list.`)
	}

	// Review workflow — the master (V4-Flash) orchestrates the V3.2 Exp review
	// itself via the review_rp_draft tool: generate → review → revise, per draft.
	if (frameMode === 'autoreact' || frameMode === 'improve' || isChat) {
		lines.push(`REVIEW WORKFLOW: for EVERY in-game response draft you write, after drafting it call the review_rp_draft tool with the draft (plus focus chips and a one-line scene summary), then revise the draft using the returned feedback before presenting it. Do this per draft option (e.g. for 3 options: draft 1 → review → revise → draft 2 → ...). Present only the reviewed, revised versions — never the raw first drafts, and never mention the review step to Brix.`)
	}

	// RP rules govern IN-GAME responses — excluded from normal chat so casual
	// conversation is never constrained by response formatting rules.
	if (primaryCtx?.rp_rules && !isChat) {
		lines.push(`\n## RP Rules (highest priority)\n${primaryCtx.rp_rules}`)
	}
	if (primaryCtx?.player_description) {
		lines.push(`\n## Emil — the player character (playerdescription.txt)\n${primaryCtx.player_description}`)
	}

	// One section per NPC (primary first, then linked group members)
	for (const npc of allNpcs.value) {
		const c = npcContexts.value[npc.file_path]
		if (!c) {
			lines.push(`\n## NPC: ${npc.name} (no context loaded yet — do not use this NPC in the response)`)
			continue
		}
		const n = c.npc || {}
		lines.push(`\n## NPC: ${n.name || npc.name} (${n.string_id || npc.string_id})`)
		if (n.mood || n.relation_desc) {
			lines.push(`State: Mood ${n.mood || 'unknown'}${n.mood_reason ? ' (' + n.mood_reason + ')' : ''} · Relation to Emil ${n.relation_desc || '?'} (${n.relation_value ?? '?'})${n.current_task ? ' · Task: ' + n.current_task : ''}`)
		}
		if (c.recent_dialogue?.length) {
			lines.push(`Conversation with ${n.name || npc.name} (oldest first — the last entry is the current moment):`)
			c.recent_dialogue.forEach((e, i) => lines.push(`[${i + 1}] ${e}`))
		}
	}
	lines.push(`\n---\nBrix's request: `)
	return lines.join('\n')
}

// Last 20 messages, filtering out the app's default welcome message
// (not relevant in the RP panel - *asterisks preserved as plain text*)
const visibleMessages = computed(() =>
	rpHistory.value
		.filter((m) => !(m.role === 'assistant' && m.content.startsWith("# \ud83d\udc4b Hello! I'm your AI Companion")))
		.slice(-20)
)

const scrollDown = async () => {
	await nextTick()
	if (msgsRef.value) msgsRef.value.scrollTop = msgsRef.value.scrollHeight
}

watch(visibleMessages, scrollDown, { deep: false })

const send = async () => {
	const text = input.value.trim()
	if (!text || isStreaming.value) return
	input.value = ''

	// The user message stays CLEAN (persisted to DB, shown in history).
	// Full RP context rides as a SYSTEM-role history entry - the LLM sees it,
	// but it never pollutes the saved chat.
	let contextEntry = null
	if (props.contextNpc) {
		const block = buildContextBlock()
		if (block) contextEntry = { role: 'system', content: block }
	}
	await sendMessageWithHistory(text, contextEntry)
}

// Shared send path — also used by the manual "NPC Responded!" trigger with a
// prebuilt context block. The user message is a synthetic instruction, not
// typed by Brix.
const sendWithSystemContext = async (contextBlock, userText) => {
	if (isStreaming.value) return
	const contextEntry = contextBlock ? { role: 'system', content: contextBlock } : null
	await sendMessageWithHistory(userText, contextEntry)
}

const sendMessageWithHistory = async (text, contextEntry) => {
	pushMessage({ role: 'user', content: text, timestamp: new Date() })
	await scrollDown()

	const historyPayload = [
		...(contextEntry ? [contextEntry] : []),
		...rpHistory.value
			.filter((m) => m.role === 'user' || m.role === 'assistant')
			.slice(-14)
			.map((m) => ({ role: m.role, content: m.content })),
	]

	const msgIndex = rpHistory.value.length
	pushMessage({ role: 'assistant', content: '', timestamp: new Date(), source: 'llm', streaming: true })
	await scrollDown()

	streamMessage(text, historyPayload, {
		onTextChunk: (chunk) => {
			rpHistory.value[msgIndex].content += chunk
			scrollDown()
		},
		onDone: () => {
			rpHistory.value[msgIndex].streaming = false
		},
		onError: (error) => {
			if (error.name === 'AbortError') {
				const lastMsg = rpHistory.value[rpHistory.value.length - 1]
				if (lastMsg?.streaming) {
					lastMsg.streaming = false
					if (lastMsg.content.length === 0) rpHistory.value.splice(-1, 1)
				}
				return
			}
			const lastMsg = rpHistory.value[rpHistory.value.length - 1]
			if (lastMsg?.streaming) lastMsg.streaming = false
			if (lastMsg?.content && lastMsg.content.length > 0) {
				lastMsg.content += '\n\n⚠️ *Stream interrupted*'
			} else {
				pushMessage({
					role: 'assistant',
					content: 'Sorry, I encountered an error. Check your OpenRouter connection.',
					timestamp: new Date(),
					source: 'error',
				})
			}
		},
	}, { scope: 'rp' }) // persist to the RP thread, not the main companion chat
}

const quickAction = (text) => {
	input.value = text
	send()
}

// Writing is the DEFAULT behavior now — no add-on toggle needed. The default
// chat framing already tells the Companion to switch into RP style when Brix
// asks for something to paste into the game.
const quickWrite = () => {
	quickAction('Write my next in-game response')
}

// ── Manual trigger: Brix clicks "NPC Responded!" ──
// Replaces auto-react: file changes no longer make the Companion act on their
// own — the dialogue view refreshes live, and Brix decides when to analyze.
const onNpcResponded = () => {
	if (!props.contextNpc || isStreaming.value) return
	reacted.value = true
	const block = buildContextBlock({ mode: 'autoreact', autoReactNpc: props.contextNpc.name })
	sendWithSystemContext(block, 'Analyze their response and draft my reply')
}
</script>

<style scoped>
.mini-chat {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	overflow: hidden;
}

.mini-chat-header {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.65rem 1rem;
	background: rgba(0, 0, 0, 0.3);
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	font-weight: 700;
	color: #00d9ff;
	font-size: 0.9rem;
}

.mini-context-chip {
	font-size: 0.7rem;
	font-weight: 600;
	background: rgba(0, 217, 255, 0.1);
	border: 1px solid rgba(0, 217, 255, 0.3);
	color: #00d9ff;
	border-radius: 10px;
	padding: 0.15rem 0.55rem;
	display: inline-flex;
	align-items: center;
}

.mini-context-chip.linked {
	background: rgba(255, 193, 7, 0.08);
	border-color: rgba(255, 193, 7, 0.35);
	color: #ffd54f;
}

.mini-chip-x {
	background: none;
	border: none;
	color: inherit;
	cursor: pointer;
	font-size: 0.85rem;
	line-height: 1;
	padding: 0 0 0 0.3rem;
	opacity: 0.7;
}

.mini-chip-x:hover {
	opacity: 1;
}

.mini-npc-select {
	background: rgba(0, 0, 0, 0.3);
	border: 1px dashed rgba(255, 255, 255, 0.25);
	color: rgba(255, 255, 255, 0.6);
	border-radius: 10px;
	font-size: 0.7rem;
	padding: 0.15rem 0.4rem;
	max-width: 120px;
}

.mini-npc-select option {
	background: #1a1a2e;
	color: #fff;
}

.mini-thinking {
	font-size: 0.72rem;
	font-weight: 400;
	color: rgba(255, 255, 255, 0.5);
}

.mini-generate {
	margin-left: auto;
	font-weight: 700;
	font-size: 0.75rem;
	background: rgba(0, 217, 255, 0.15);
	border: 1px solid rgba(0, 217, 255, 0.5);
	color: #00d9ff;
}

.mini-generate:hover:not(:disabled) {
	background: rgba(0, 217, 255, 0.3);
	color: #fff;
}

.mini-generate:disabled {
	opacity: 0.5;
}

/* New NPC dialogue entry waiting — pulse until clicked or superseded */
.mini-generate.has-update {
	background: rgba(40, 167, 69, 0.22);
	border-color: #28a745;
	color: #7ee2a8;
	animation: mini-npc-pulse 1.6s ease-in-out infinite;
}

@keyframes mini-npc-pulse {
	0%, 100% { box-shadow: 0 0 0 0 rgba(40, 167, 69, 0.45); }
	50% { box-shadow: 0 0 0 5px rgba(40, 167, 69, 0); }
}

.mini-chat-messages {
	flex: 1;
	overflow-y: auto;
	padding: 0.9rem;
	display: flex;
	flex-direction: column;
	gap: 0.7rem;
	background: rgba(0, 0, 0, 0.2);
	scroll-behavior: smooth;
}

.mini-empty {
	flex: 1;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: 0.6rem;
	color: rgba(255, 255, 255, 0.4);
	font-size: 0.85rem;
	text-align: center;
}

.mini-empty p {
	margin: 0;
}

.mini-quick-actions {
	display: flex;
	flex-direction: column;
	gap: 0.4rem;
	align-items: center;
}

.mini-hint-btn {
	background: rgba(0, 217, 255, 0.08);
	border: 1px solid rgba(0, 217, 255, 0.3);
	color: #00d9ff;
	border-radius: 8px;
	padding: 0.45rem 0.9rem;
	font-size: 0.8rem;
	cursor: pointer;
	transition: background 0.2s;
}

.mini-hint-btn:hover {
	background: rgba(0, 217, 255, 0.16);
}

.mini-msg {
	display: flex;
	flex-direction: column;
}

.mini-role {
	font-size: 0.6rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	margin-bottom: 0.2rem;
}

.mini-msg.user .mini-role {
	color: rgba(0, 217, 255, 0.7);
}

.mini-msg.assistant .mini-role {
	color: rgba(255, 255, 255, 0.45);
}

.mini-content {
	font-size: 0.8rem;
	line-height: 1.5;
	color: rgba(255, 255, 255, 0.85);
	white-space: pre-wrap;
	word-break: break-word;
}

.mini-msg.user .mini-content {
	background: rgba(0, 217, 255, 0.08);
	border: 1px solid rgba(0, 217, 255, 0.2);
	border-radius: 10px;
	border-top-right-radius: 3px;
	padding: 0.45rem 0.7rem;
	align-self: flex-end;
	max-width: 92%;
}

.mini-msg.assistant .mini-content {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 10px;
	border-top-left-radius: 3px;
	padding: 0.45rem 0.7rem;
}

.mini-chat-input {
	display: flex;
	gap: 0.4rem;
	padding: 0.6rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	background: rgba(0, 0, 0, 0.25);
}

.mini-chat-input .form-control {
	background: rgba(0, 0, 0, 0.3);
	border: 1px solid rgba(255, 255, 255, 0.15);
	color: #fff;
	font-size: 0.82rem;
}

.mini-chat-input .form-control::placeholder {
	color: rgba(255, 255, 255, 0.35);
}

/* Mode chips */
.mini-quick-row {
	display: flex;
	gap: 0.3rem;
	padding: 0.45rem 0.6rem 0;
	flex-wrap: wrap;
}

.mini-quick-chip {
	background: rgba(255, 255, 255, 0.04);
	border: 1px solid rgba(255, 255, 255, 0.12);
	color: rgba(255, 255, 255, 0.65);
	border-radius: 7px;
	padding: 0.22rem 0.5rem;
	font-size: 0.68rem;
	font-weight: 600;
	cursor: pointer;
	transition: all 0.15s ease;
}

.mini-quick-chip:hover:not(:disabled) {
	color: #00d9ff;
	border-color: rgba(0, 217, 255, 0.4);
}

.mini-quick-chip:disabled {
	opacity: 0.4;
	cursor: default;
}

.mini-modes {
	display: flex;
	gap: 0.3rem;
	padding: 0.45rem 0.6rem 0;
	flex-wrap: wrap;
	align-items: center;
}

.mini-modes-label {
	font-size: 0.58rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: rgba(255, 255, 255, 0.35);
	flex-shrink: 0;
}

.mini-mode-chip {
	flex: 1 1 auto;
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: rgba(255, 255, 255, 0.6);
	border-radius: 7px;
	padding: 0.3rem 0.4rem;
	font-size: 0.72rem;
	font-weight: 600;
	cursor: pointer;
	transition: all 0.15s ease;
}

.mini-mode-chip:hover {
	color: #fff;
	border-color: rgba(0, 217, 255, 0.4);
}

.mini-mode-chip.active {
	background: rgba(0, 217, 255, 0.12);
	border-color: #00d9ff;
	color: #00d9ff;
}

/* ── Focus chips (woven into generated drafts) ── */
.mini-focus {
	margin-top: 0.3rem;
}

.focus-chip {
	font-size: 0.72rem;
}

.focus-x {
	margin-left: 0.35rem;
	opacity: 0.6;
	font-weight: 700;
}

.focus-x:hover {
	opacity: 1;
	color: #ff6b6b;
}

.mini-focus-add {
	width: 90px;
	font-size: 0.72rem;
	padding: 0.15rem 0.45rem;
	border-radius: 10px;
	border: 1px dashed rgba(0, 217, 255, 0.4);
	background: rgba(0, 0, 0, 0.25);
	color: #fff;
}

.mini-focus-plus {
	border: 1px dashed rgba(255, 255, 255, 0.25);
	background: transparent;
	color: rgba(255, 255, 255, 0.5);
	border-radius: 10px;
	font-size: 0.72rem;
	padding: 0.15rem 0.5rem;
	cursor: pointer;
}

.mini-focus-plus:hover {
	color: #00d9ff;
	border-color: rgba(0, 217, 255, 0.4);
}
</style>
