<template>
	<div class="mini-chat">
		<div class="mini-chat-header">
			<font-awesome-icon :icon="['fas', 'robot']" class="me-1" />
			AI Companion
			<span v-if="contextNpc" class="mini-context-chip">
				<font-awesome-icon :icon="['fas', 'mask']" class="me-1" />
				{{ contextNpc.name }}
			</span>
			<button
				v-if="contextNpc"
				class="btn btn-sm btn-info mini-generate"
				:disabled="isStreaming"
				title="Generate your next in-game response from the current dialogue"
				@click="generateResponse"
			>
				<font-awesome-icon :icon="['fas', 'bolt']" class="me-1" />
				Generate response
			</button>
			<span v-if="isStreaming" class="mini-thinking">
				<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-1" />
				thinking...
			</span>
		</div>

		<div ref="msgsRef" class="mini-chat-messages">
			<div v-if="visibleMessages.length === 0 && !isStreaming" class="mini-empty">
				<p>Ask the Companion for help with your roleplay.</p>
				<div v-if="contextNpc" class="mini-quick-actions">
					<button class="mini-hint-btn" @click="quickAction('Help me write my next response')">✍ Write my next response</button>
					<button class="mini-hint-btn" @click="quickAction('Analyze her current mood and what it means for me')">🧠 Analyze the mood</button>
					<button class="mini-hint-btn" @click="quickAction('What does she want to hear right now? Give me 2-3 response options.')">💡 Response options</button>
				</div>
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

		<div class="mini-chat-input">
			<input
				v-model="input"
				type="text"
				class="form-control form-control-sm"
				:placeholder="contextNpc ? 'Ask about ' + contextNpc.name + ' or request a response...' : 'Ask the Companion...'"
				@keyup.enter="send"
				:disabled="isStreaming"
			/>
			<div class="mini-len-wrap" title="Minimum response length in characters">
				<input
					v-model.number="minResponseLength"
					type="number"
					class="form-control form-control-sm mini-len-input"
					min="0"
					max="5000"
					step="50"
					:disabled="isStreaming"
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
})

const { chatHistory, pushMessage } = useChatHistory()
const { isStreaming, streamMessage, stopStreaming } = useChatStreaming()

const input = ref('')
const msgsRef = ref(null)

// Minimum response length (chars) - persisted across sessions
const minResponseLength = ref(parseInt(localStorage.getItem('rp_min_response_length') || '400', 10))
watch(minResponseLength, (v) => {
	localStorage.setItem('rp_min_response_length', String(v || 0))
})

// ── Full RP context (world lore, rules, player description, NPC identity) ──
const rpContext = ref(null)

watch(
	() => props.contextNpc?.file_path,
	async (path) => {
		rpContext.value = null
		if (!path) return
		try {
			const res = await bannerlordAPI.getRpContext(path)
			rpContext.value = res?.data || null
		} catch { /* fall back to short prefix */ }
	},
	{ immediate: true }
)

// Build the RP context block. Sent as a SYSTEM-role history entry (not part of
// the user message) so persisted chat history stays clean and readable.
// Knowledge scope: ONLY the conversation dialogue + mechanical state (mood/relation).
// Backstory/personality/world lore are deliberately EXCLUDED — no cheating.
const buildContextBlock = () => {
	const c = rpContext.value
	const npc = props.contextNpc
	if (!c || !npc) return ''
	const n = c.npc || {}
	const lines = []
	lines.push(`[ROLEPLAY CONTEXT — Bannerlord AIInfluence campaign ${c.campaign_id}]`)
	lines.push(`Brix plays Emil. Help him write his NEXT IN-GAME RESPONSE to the NPC below, as a direct continuation of the conversation. Follow the campaign RP rules (below). Output ONLY the response text — no preamble, no explanation, no markdown, no quotes around it. Use *asterisks for actions and inner thoughts* and "quotes" for speech, exactly like the dialogue examples.`)
	lines.push(`FACTS: The conversation below is the ONLY source of what has been said and what the NPC knows or has heard. Do NOT invent or reference backstory, rumors, personality traits, or world facts. Nothing exists except what is in the dialogue and the mechanical state below.`)
	if (minResponseLength.value > 0) {
		lines.push(`LENGTH: at least ${minResponseLength.value} characters — fill naturally with reactions, inner thoughts, small physical details, and dialogue. No padding, no room descriptions.`)
	}
	if (c.rp_rules) {
		lines.push(`\n## RP Rules (highest priority)\n${c.rp_rules}`)
	}
	if (c.player_description) {
		lines.push(`\n## Emil — the player character (playerdescription.txt)\n${c.player_description}`)
	}
	if (n.mood || n.relation_desc) {
		lines.push(`\n## Current mechanical state (from the game)\nMood: ${n.mood || 'unknown'}${n.mood_reason ? ' (' + n.mood_reason + ')' : ''}\nRelation to Emil: ${n.relation_desc || '?'} (${n.relation_value ?? '?'})`)
		if (n.current_task) {
			lines.push(`Task: ${n.current_task}`)
		}
	}
	if (c.recent_dialogue?.length) {
		lines.push(`\n## Conversation so far (oldest first — the last entry is the current moment)`)
		c.recent_dialogue.forEach((e, i) => lines.push(`[${i + 1}] ${e}`))
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
	quickAction('Write my next in-game response')
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
</style>
