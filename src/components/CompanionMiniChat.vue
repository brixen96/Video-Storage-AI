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
				class="mini-auto-toggle"
				:class="{ on: autoReact }"
				title="Auto-react: when an NPC responds in-game, the Companion instantly analyzes it and drafts your reply"
				@click="autoReact = !autoReact"
			>
				<font-awesome-icon :icon="['fas', autoReact ? 'bolt' : 'bolt']" class="me-1" />
				Auto
			</button>
			<button
				v-if="contextNpc"
				class="btn btn-sm btn-info mini-generate"
				:disabled="isStreaming"
				:title="generateTitle"
				@click="generateResponse"
			>
				<font-awesome-icon :icon="['fas', 'bolt']" class="me-1" />
				{{ generateLabel }}
			</button>
			<span v-if="isStreaming" class="mini-thinking">
				<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-1" />
				thinking...
			</span>
		</div>

		<div ref="msgsRef" class="mini-chat-messages">
			<div v-if="visibleMessages.length === 0 && !isStreaming" class="mini-empty">
				<p>Ask the Companion for help with your roleplay — or use the quick chips below.</p>
			</div>
			<div v-for="(msg, i) in visibleMessages" :key="i" class="mini-msg" :class="msg.role">
				<div class="mini-role">{{ msg.role === 'user' ? 'Emil' : 'Companion' }}</div>
				<div class="mini-content">{{ msg.content }}</div>
			</div>
			<div v-if="isStreaming" class="mini-msg assistant">
				<div class="mini-role">Companion</div>
				<div class="mini-content">
					<div class="typing-indicator"><span></span><span></span><span></span></div>
				</div>
			</div>
		</div>

		<!-- Persistent quick actions (often-used prompts) -->
		<div v-if="contextNpc" class="mini-quick-row">
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Analyze the mood of everyone here and what it means for me')">🧠 Mood</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Give me 3 distinct response options with a one-line summary each')">💡 3 options</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Summarize the scene so far in 3-4 sentences')">📜 Scene summary</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('Suggest the next beat of this scene — where could it go from here?')">🗺️ Next beat</button>
			<button class="mini-quick-chip" :disabled="isStreaming" @click="quickAction('What is this NPC likely to do or say next, and why?')">🔮 Predict</button>
		</div>

		<div class="mini-modes">
			<button
				v-for="m in modes"
				:key="m.id"
				class="mini-mode-chip"
				:class="{ active: mode === m.id }"
				@click="mode = m.id"
				:title="m.hint"
			>
				<font-awesome-icon :icon="['fas', m.icon]" class="me-1" />
				{{ m.label }}
			</button>
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
			<div class="mini-len-wrap" title="Response length — Dynamic lets the Companion pick per moment; hard cap 2048 (in-game text input limit)">
				<label class="mini-dyn-label">
					<input v-model="dynamicLength" type="checkbox" class="me-1" />
					Dynamic
				</label>
				<input
					v-model.number="minResponseLength"
					type="number"
					class="form-control form-control-sm mini-len-input"
					min="0"
					max="2048"
					step="50"
					:disabled="isStreaming || dynamicLength"
				/>
				<span class="mini-len-label">min&nbsp;len</span>
			</div>
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
import { ref, computed, watch, nextTick } from 'vue'
import { useChatHistory } from '@/composables/ai/useChatHistory'
import { useChatStreaming } from '@/composables/ai/useChatStreaming'
import { bannerlordAPI } from '@/services/api'

const props = defineProps({
	// When set, the Companion is told which NPC conversation is open
	contextNpc: { type: Object, default: null },
	// Full roster — enables linking additional NPCs into group scenes
	availableNpcs: { type: Array, default: () => [] },
	// Parent bumps this when the NPC posts a new in-game response
	autoReactKey: { type: Number, default: 0 },
})

const { chatHistory, pushMessage } = useChatHistory()
const { isStreaming, streamMessage, stopStreaming } = useChatStreaming()

const input = ref('')
const msgsRef = ref(null)

// ── Companion modes ──
const modes = [
	{ id: 'write', label: 'Write', icon: 'pen', hint: 'Generate the next in-game response' },
	{ id: 'improve', label: 'Improve', icon: 'magic', hint: 'Improve a draft Brix wrote himself + feedback' },
	{ id: 'ooc', label: 'OOC', icon: 'comment-dots', help: true, hint: 'Help write an OOC (out-of-character) message' },
	{ id: 'plan', label: 'Plan', icon: 'route', hint: 'Plan a scene or scenario with this NPC' },
]
const mode = ref(localStorage.getItem('rp_companion_mode') || 'write')
watch(mode, (v) => localStorage.setItem('rp_companion_mode', v))

const modePlaceholder = computed(() => {
	const npcName = props.contextNpc?.name || 'the NPC'
	switch (mode.value) {
		case 'improve': return 'Paste your draft response here...'
		case 'ooc': return 'What do you want to say OOC?'
		case 'plan': return 'Describe the scene direction you want...'
		default: return props.contextNpc ? `Ask about ${npcName} or request a response...` : 'Ask the Companion...'
	}
})

const generateLabel = computed(() => {
	switch (mode.value) {
		case 'improve': return 'Improve draft'
		case 'ooc': return 'Draft OOC'
		case 'plan': return 'Plan scene'
		default: return 'Generate response'
	}
})

const generateTitle = computed(() => {
	switch (mode.value) {
		case 'improve': return 'Improve the draft in the input and give feedback'
		case 'ooc': return 'Draft an OOC message based on the input'
		case 'plan': return 'Plan the next scene/scenario with this NPC'
		default: return 'Generate your next in-game response from the current dialogue'
	}
})

// Minimum response length (chars) - persisted across sessions
const minResponseLength = ref(parseInt(localStorage.getItem('rp_min_response_length') || '400', 10))
watch(minResponseLength, (v) => {
	localStorage.setItem('rp_min_response_length', String(Math.min(v || 0, 2048)))
})

// Dynamic length: the Companion picks a natural length per moment.
// Hard cap of 2048 chars — the in-game text input limit.
const MAX_RESPONSE_LENGTH = 2048
const dynamicLength = ref(localStorage.getItem('rp_length_dynamic') !== '0')
watch(dynamicLength, (v) => localStorage.setItem('rp_length_dynamic', v ? '1' : '0'))

// Auto-react: instantly analyze + draft a reply when an NPC responds in-game
const autoReact = ref(localStorage.getItem('rp_auto_react') !== '0')
watch(autoReact, (v) => localStorage.setItem('rp_auto_react', v ? '1' : '0'))

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
	const frameMode = opts.mode || mode.value
	const primaryCtx = npcContexts.value[allNpcs.value[0].file_path] || null
	const lines = []
	const campaign = primaryCtx?.campaign_id || 'current'
	lines.push(`[ROLEPLAY CONTEXT — Bannerlord AIInfluence campaign ${campaign}]`)

	// Mode-specific framing
	switch (frameMode) {
		case 'autoreact':
			lines.push(`Brix plays Emil. ${opts.autoReactNpc || 'An NPC'} just responded in-game (the newest dialogue entry). Do TWO things: (1) give quick FEEDBACK on their response — 2-3 sentences on tone, subtext, and what they really want (2-4 bullets); (2) then draft Brix's NEXT REPLY in the RP style — *asterisks for actions/thoughts*, "quotes" for speech, no preamble.`)
			break
		case 'improve':
			lines.push(`Brix plays Emil. He wrote a response HIMSELF (his text follows the request marker). IMPROVE it: keep his intent, voice, and plot beats — fix flow, grammar, and adherence to the campaign RP rules below. Output in TWO parts: (1) the improved version ONLY — no preamble — using *asterisks for actions/thoughts* and "quotes" for speech; (2) a line '---' then 2-4 short feedback bullets on what you changed and why.`)
			break
		case 'ooc':
			lines.push(`Brix plays Emil. He wants to send an OOC (out-of-character) message to the game's AI per the campaign's OOC protocol (see rules below). Help him write it clearly and completely. Output ONLY the final OOC message text, formatted as [OOC: ...] — no preamble, no explanation.`)
			break
		case 'plan':
			lines.push(`Brix plays Emil. He wants to PLAN the next scene/scenario with the NPC(s) below. Based ONLY on the conversation(s) so far and the campaign rules, propose 2-3 distinct scene directions with brief beats for each, then recommend one. Be concrete about what could happen moment to moment. Output the plan only — no preamble.`)
			break
		default:
			lines.push(`Brix plays Emil. Help him write his NEXT IN-GAME RESPONSE to the NPC(s) below, as a direct continuation of the conversation(s). Follow the campaign RP rules (below). Output ONLY the response text — no preamble, no explanation, no markdown, no quotes around it. Use *asterisks for actions and inner thoughts* and "quotes" for speech, exactly like the dialogue examples.`)
	}

	lines.push(`FACTS: The conversation(s) below are the ONLY source of what has been said and what the NPC(s) know or have heard. Do NOT invent or reference backstory, rumors, personality traits, or world facts. Nothing exists except what is in the dialogue and the mechanical states below.`)

	// Length: Dynamic = Companion picks per moment; Fixed = at least N chars.
	// Hard cap 2048 — the in-game text input limit.
	if (dynamicLength.value) {
		lines.push(`LENGTH: choose a natural length for the moment — reactions-heavy scenes longer, quick exchanges shorter. NEVER exceed ${MAX_RESPONSE_LENGTH} characters (in-game text input limit). No padding, no room descriptions.`)
	} else if (minResponseLength.value > 0) {
		const n = Math.min(minResponseLength.value, MAX_RESPONSE_LENGTH)
		lines.push(`LENGTH: at least ${n} characters, NEVER more than ${MAX_RESPONSE_LENGTH} (in-game text input limit). Fill naturally with reactions, inner thoughts, small physical details, and dialogue. No padding, no room descriptions.`)
	}

	if (primaryCtx?.rp_rules) {
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
	chatHistory.value
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

// Shared send path — also used by auto-react with a prebuilt context block.
// The auto-react user message is a synthetic instruction, not typed by Brix.
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
		...chatHistory.value
			.filter((m) => m.role === 'user' || m.role === 'assistant')
			.slice(-14)
			.map((m) => ({ role: m.role, content: m.content })),
	]

	const msgIndex = chatHistory.value.length
	pushMessage({ role: 'assistant', content: '', timestamp: new Date(), source: 'llm', streaming: true })
	await scrollDown()

	streamMessage(text, historyPayload, {
		onTextChunk: (chunk) => {
			chatHistory.value[msgIndex].content += chunk
			scrollDown()
		},
		onDone: () => {
			chatHistory.value[msgIndex].streaming = false
		},
		onError: (error) => {
			if (error.name === 'AbortError') {
				const lastMsg = chatHistory.value[chatHistory.value.length - 1]
				if (lastMsg?.streaming) {
					lastMsg.streaming = false
					if (lastMsg.content.length === 0) chatHistory.value.splice(-1, 1)
				}
				return
			}
			const lastMsg = chatHistory.value[chatHistory.value.length - 1]
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
	})
}

const quickAction = (text) => {
	input.value = text
	send()
}

const generateResponse = () => {
	switch (mode.value) {
		case 'improve':
			if (input.value.trim()) quickAction('Improve my draft: ' + input.value.trim())
			else input.value = '' // focus input — needs a draft
			break
		case 'ooc':
			if (input.value.trim()) quickAction('Draft this OOC message: ' + input.value.trim())
			else input.value = ''
			break
		case 'plan':
			quickAction('Help me plan the next scene')
			break
		default:
			quickAction('Write my next in-game response')
	}
}

// ── Auto-react: NPC responded in-game → instant feedback + reply draft ──
watch(
	() => props.autoReactKey,
	(newKey, oldKey) => {
		if (!newKey || newKey <= (oldKey || 0)) return
		if (!autoReact.value || isStreaming.value || !props.contextNpc) return
		// Let the freshly reloaded dialogue + context cache settle first
		setTimeout(() => {
			if (isStreaming.value) return
			const block = buildContextBlock({ mode: 'autoreact', autoReactNpc: props.contextNpc.name })
			sendWithSystemContext(block, 'Analyze their response and draft my reply')
		}, 600)
	}
)
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

.mini-auto-toggle {
	background: rgba(0, 0, 0, 0.3);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: rgba(255, 255, 255, 0.5);
	border-radius: 10px;
	font-size: 0.7rem;
	font-weight: 700;
	padding: 0.15rem 0.5rem;
	cursor: pointer;
	transition: all 0.15s ease;
}

.mini-auto-toggle.on {
	background: rgba(40, 167, 69, 0.15);
	border-color: rgba(40, 167, 69, 0.5);
	color: #7ee2a8;
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

.mini-len-wrap {
	display: flex;
	flex-direction: column;
	align-items: center;
	flex-shrink: 0;
}

.mini-len-input {
	width: 74px;
	text-align: center;
	font-size: 0.75rem;
}

.mini-len-label {
	font-size: 0.55rem;
	text-transform: uppercase;
	letter-spacing: 0.04em;
	color: rgba(255, 255, 255, 0.4);
	margin-top: 1px;
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

.mini-dyn-label {
	font-size: 0.6rem;
	color: rgba(255, 255, 255, 0.55);
	display: flex;
	align-items: center;
	white-space: nowrap;
	margin: 0;
	cursor: pointer;
}

.mini-dyn-label input {
	cursor: pointer;
}

.mini-modes {
	display: flex;
	gap: 0.3rem;
	padding: 0.45rem 0.6rem 0;
}

.mini-mode-chip {
	flex: 1;
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
</style>
