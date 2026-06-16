/**
 * useChatUI — panel visibility, context stats, message scrolling, memory panel, voice input.
 */
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { aiAPI, librariesAPI, activityAPI } from '@/services/api'

export function useChatUI({ chatHistory, toggleChat }) {
	// --- Core UI state ---
	const isOpen = ref(false)
	const unreadCount = ref(0)
	const chatMessages = ref(null)
	const showStats = ref(false)
	const proactiveEnabled = ref(localStorage.getItem('ai_companion_proactive') !== 'false')
	const companionConnected = ref(false)

	// --- Memory panel ---
	const showMemoryPanel = ref(false)
	const memories = ref([])
	const memoriesLoading = ref(false)
	const memoryCategory = ref('')
	const memorySearch = ref('')

	const memoryCounts = computed(() => {
		const counts = {}
		for (const m of memories.value) counts[m.category] = (counts[m.category] || 0) + 1
		return Object.entries(counts)
			.map(([name, count]) => ({ name, count }))
			.sort((a, b) => b.count - a.count)
	})

	const filteredMemories = computed(() => {
		let list = memories.value
		if (memoryCategory.value) list = list.filter((m) => m.category === memoryCategory.value)
		if (memorySearch.value) {
			const q = memorySearch.value.toLowerCase()
			list = list.filter((m) => m.key.toLowerCase().includes(q) || m.value.toLowerCase().includes(q))
		}
		return list
	})

	const loadMemories = async () => {
		memoriesLoading.value = true
		try {
			const res = await aiAPI.getMemories()
			memories.value = (res?.data?.data ?? res?.data ?? []).sort(
				(a, b) => new Date(b.updated_at) - new Date(a.updated_at),
			)
		} catch { /* silently ignore */ }
		finally { memoriesLoading.value = false }
	}

	const toggleMemoryPanel = () => {
		showMemoryPanel.value = !showMemoryPanel.value
		if (showMemoryPanel.value) {
			memoryCategory.value = ''
			memorySearch.value = ''
			loadMemories()
		}
	}

	const deleteMemory = async (id) => {
		try {
			await aiAPI.deleteMemory(id)
			memories.value = memories.value.filter((m) => m.id !== id)
		} catch { /* best-effort */ }
	}

	const formatMemoryDate = (dateStr) => {
		if (!dateStr) return ''
		return new Date(dateStr).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
	}

	// --- Context stats strip ---
	const contextStats = ref(null)
	const contextStatsLoading = ref(false)
	const contextStatsDismissed = ref(false)

	const loadContextStats = async () => {
		contextStatsLoading.value = true
		try {
			const [libRes, statusRes] = await Promise.all([
				librariesAPI.getAll(),
				activityAPI.getStatus(),
			])
			const libs = Array.isArray(libRes?.data) ? libRes.data : []
			const totalVideos = libs.reduce((sum, l) => sum + (l.video_count || 0), 0)
			const status = statusRes?.data || {}
			contextStats.value = {
				videos: totalVideos,
				running: status.running_tasks || 0,
				failed: status.failed_tasks || 0,
			}
		} catch { /* strip stays hidden */ }
		finally { contextStatsLoading.value = false }
	}

	// --- Scrolling ---
	const scrollToBottom = () => {
		if (chatMessages.value) {
			chatMessages.value.scrollTop = chatMessages.value.scrollHeight
		}
	}

	// --- Global keyboard shortcut (Ctrl+Shift+A) ---
	const handleGlobalShortcut = (e) => {
		if (e.ctrlKey && e.shiftKey && (e.key === 'A' || e.key === 'a')) {
			e.preventDefault()
			toggleChat?.()
		}
	}

	// --- Toggle actions ---
	const toggleSearch = () => {
		// Handled by showSearch in useChatHistory — this is for header button
	}

	const toggleProactive = () => {
		proactiveEnabled.value = !proactiveEnabled.value
		localStorage.setItem('ai_companion_proactive', proactiveEnabled.value ? 'true' : 'false')
	}

	const toggleChat = (opts) => {
		if (opts?.open) isOpen.value = true
		else if (opts?.close) isOpen.value = false
		else isOpen.value = !isOpen.value
		if (isOpen.value) {
			unreadCount.value = 0
			nextTick(() => scrollToBottom())
		}
	}

	// --- Lifecycle ---
	const setupShortcuts = () => {
		document.addEventListener('keydown', handleGlobalShortcut)
	}
	const cleanupShortcuts = () => {
		document.removeEventListener('keydown', handleGlobalShortcut)
	}

	return {
		// State
		isOpen,
		unreadCount,
		chatMessages,
		showStats,
		proactiveEnabled,
		companionConnected,
		// Memory
		showMemoryPanel,
		memories,
		memoriesLoading,
		memoryCategory,
		memorySearch,
		memoryCounts,
		filteredMemories,
		loadMemories,
		toggleMemoryPanel,
		deleteMemory,
		formatMemoryDate,
		// Context stats
		contextStats,
		contextStatsLoading,
		contextStatsDismissed,
		loadContextStats,
		// Scrolling
		scrollToBottom,
		// Toggles
		toggleSearch,
		toggleProactive,
		toggleChat,
		// Lifecycle
		setupShortcuts,
		cleanupShortcuts,
	}
}
