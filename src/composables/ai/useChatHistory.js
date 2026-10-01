/**
 * useChatHistory — message state, persistence, and computed derivations.
 * Uses module-level singletons so all callers share the same state.
 */
import { ref, computed } from 'vue'
import { aiAPI } from '@/services/api'

// ── Singleton state (shared across floating widget + AI page) ──────
const chatHistory = ref([])
const showSearch = ref(false)
const searchQuery = ref('')
const sentHistory = ref(JSON.parse(localStorage.getItem('ai_companion_prompt_history') || '[]'))
const historyIndex = ref(-1)
const savedDraft = ref('')

export function useChatHistory() {

	// --- Computed derivations ---

	const pinnedMessages = computed(() =>
		chatHistory.value.filter((m) => m.pinned),
	)

	const filteredMessageCount = computed(() => {
		if (!searchQuery.value) return 0
		const q = searchQuery.value.toLowerCase()
		return chatHistory.value.filter((m) => m.content.toLowerCase().includes(q)).length
	})

	const displayedMessages = computed(() => {
		if (!searchQuery.value) return chatHistory.value
		const query = searchQuery.value.toLowerCase()
		const filtered = chatHistory.value.map((msg, index) => ({
			...msg,
			highlighted: msg.content.toLowerCase().includes(query),
			originalIndex: index,
		})).filter((msg) => msg.highlighted)
		return filtered.length > 0 ? filtered : chatHistory.value
	})

	const chatStats = computed(() => {
		const total = chatHistory.value.length
		const userMessages = chatHistory.value.filter((m) => m.role === 'user').length
		const aiMessages = chatHistory.value.filter((m) => m.role === 'assistant').length
		const avgLen = total > 0
			? Math.round(chatHistory.value.reduce((s, m) => s + m.content.length, 0) / total)
			: 0
		let sessionDuration = 'N/A'
		if (total > 0) {
			const diffMs = new Date(chatHistory.value[total - 1].timestamp) - new Date(chatHistory.value[0].timestamp)
			const mins = Math.floor(diffMs / 60000)
			sessionDuration = mins < 60 ? `${mins}m` : `${Math.floor(mins / 60)}h ${mins % 60}m`
		}
		return { totalMessages: total, userMessages, aiMessages, avgMessageLength: avgLen, sessionDuration }
	})

	// --- Actions ---

	/** Find the real chatHistory index for a message (handles search-mode messages with originalIndex) */
	const resolveIndex = (msg) => {
		const idx = chatHistory.value.indexOf(msg)
		return idx !== -1 ? idx : (msg.originalIndex ?? -1)
	}

	/** Push a message and return it */
	const pushMessage = (msg) => {
		chatHistory.value.push(msg)
		return msg
	}

	/** Remove the last N messages */
	const popMessages = (count = 1) => {
		chatHistory.value.splice(chatHistory.value.length - count, count)
	}

	/** Remove a specific message by reference */
	const removeMessage = (msg) => {
		const idx = chatHistory.value.indexOf(msg)
		if (idx !== -1) chatHistory.value.splice(idx, 1)
	}

	/** Replace a contiguous range of messages (used by regenerate) */
	const replaceRange = (startIdx, count, ...newMsgs) => {
		chatHistory.value.splice(startIdx, count, ...newMsgs)
	}

	const clearSearch = () => { searchQuery.value = '' }

	/** Persist current chat history to backend DB */
	const persistHistory = async () => {
		// no-op — server already persists via the streaming endpoint
	}

	/** Load chat history from backend DB (fallback to localStorage for legacy data) */
	const loadHistory = async () => {
		try {
			const { data: res } = await aiAPI.getChatHistory(100)
			if (res.data && res.data.length > 0) {
				chatHistory.value = res.data.map((m) => ({
					role: m.role,
					content: m.content,
					timestamp: new Date(m.created_at),
					source: m.role !== 'user' ? m.source : undefined,				toolSteps: m.tool_steps ? JSON.parse(m.tool_steps) : undefined,				}))
				localStorage.removeItem('ai_chat_history')
				return
			}
		} catch {
			// Backend unavailable — fall back to localStorage
		}
		// Legacy localStorage fallback
		try {
			const saved = localStorage.getItem('ai_chat_history')
			if (saved) {
				chatHistory.value = JSON.parse(saved).slice(-50).map((msg) => ({
					...msg,
					timestamp: new Date(msg.timestamp),
				}))
			}
		} catch { /* ignore */ }
	}

	/** Clear all chat history */
	const clearHistory = async () => {
		try { await aiAPI.clearChatHistory() } catch { /* best-effort */ }
		chatHistory.value = []
		localStorage.removeItem('ai_chat_history')
	}

	// Prompt history — terminal-style ArrowUp/Down cycling
	const pushPromptHistory = (text) => {
		sentHistory.value.push(text)
		if (sentHistory.value.length > 50) sentHistory.value.shift()
		historyIndex.value = -1
		savedDraft.value = ''
		localStorage.setItem('ai_companion_prompt_history', JSON.stringify(sentHistory.value))
	}

	return {
		// State
		chatHistory,
		showSearch,
		searchQuery,
		sentHistory,
		historyIndex,
		savedDraft,
		// Computed
		pinnedMessages,
		filteredMessageCount,
		displayedMessages,
		chatStats,
		// Actions
		resolveIndex,
		pushMessage,
		popMessages,
		removeMessage,
		replaceRange,
		clearSearch,
		persistHistory,
		loadHistory,
		clearHistory,
		pushPromptHistory,
	}
}
