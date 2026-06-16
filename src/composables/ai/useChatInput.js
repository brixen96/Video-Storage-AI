/**
 * useChatInput — textarea state, slash commands, @mentions, prompt history navigation.
 */
import { ref, watch, nextTick } from 'vue'
import { performersAPI } from '@/services/api'
import { SLASH_COMMANDS } from './constants'

export function useChatInput({ sentHistory, historyIndex, savedDraft, pushPromptHistory }) {
	const companionInput = ref('')
	const textareaRef = ref(null)

	// --- Auto-resize textarea ---
	const autoResizeTextarea = () => {
		const el = textareaRef.value
		if (!el) return
		el.style.height = 'auto'
		el.style.height = Math.min(el.scrollHeight, 120) + 'px'
	}

	// --- Slash command palette ---
	const slashActive = ref(false)
	const slashResults = ref([])
	const slashIndex = ref(0)

	const closeSlash = () => {
		slashActive.value = false
		slashResults.value = []
		slashIndex.value = 0
	}

	const selectSlash = (cmd, onSend) => {
		if (!cmd) return
		companionInput.value = cmd.fill
		closeSlash()
		if (!cmd.param) nextTick(() => onSend?.())
	}

	// --- @mention autocomplete ---
	const mentionActive = ref(false)
	const mentionResults = ref([])
	const mentionIndex = ref(0)
	const mentionStart = ref(-1)

	const closeMention = () => {
		mentionActive.value = false
		mentionResults.value = []
		mentionIndex.value = 0
	}

	const selectMention = (performer) => {
		if (!performer) return
		const val = companionInput.value
		companionInput.value = val.slice(0, mentionStart.value) + performer.name + ' '
		closeMention()
	}

	let _mentionTimer = null

	// --- Keyboard handler (unified for slash, mention, and normal input) ---
	const handleInputKeydown = (e, { onSend, onMentionClose, onSlashClose }) => {
		const inMention = mentionActive.value && mentionResults.value.length > 0
		const inSlash = slashActive.value && slashResults.value.length > 0

		if (!inMention && !inSlash) {
			if (e.key === 'Enter' && !e.shiftKey) {
				e.preventDefault()
				onSend?.()
			} else if (e.key === 'ArrowUp' && !e.shiftKey && sentHistory.value.length > 0 && !companionInput.value.includes('\n')) {
				e.preventDefault()
				if (historyIndex.value === -1) savedDraft.value = companionInput.value
				historyIndex.value = Math.min(historyIndex.value + 1, sentHistory.value.length - 1)
				companionInput.value = sentHistory.value[sentHistory.value.length - 1 - historyIndex.value]
			} else if (e.key === 'ArrowDown' && historyIndex.value !== -1 && !companionInput.value.includes('\n')) {
				e.preventDefault()
				historyIndex.value--
				companionInput.value = historyIndex.value === -1 ? savedDraft.value : sentHistory.value[sentHistory.value.length - 1 - historyIndex.value]
			}
			return
		}

		const index = inMention ? mentionIndex : slashIndex
		const results = inMention ? mentionResults : slashResults
		const select = inMention
			? () => selectMention(mentionResults.value[mentionIndex.value])
			: () => selectSlash(slashResults.value[slashIndex.value], onSend)

		if (e.key === 'ArrowDown') { e.preventDefault(); if (index.value < results.value.length - 1) index.value++ }
		else if (e.key === 'ArrowUp') { e.preventDefault(); if (index.value > 0) index.value-- }
		else if (e.key === 'Enter' || e.key === 'Tab') { e.preventDefault(); select() }
		else if (e.key === 'Escape') {
			e.preventDefault()
			if (inMention) { closeMention(); onMentionClose?.() } else { closeSlash(); onSlashClose?.() }
		}
	}

	// --- Watchers (setup on mount, cleanup on unmount) ---
	const setupWatchers = () => {
		// Auto-resize on every input change
		watch(companionInput, () => nextTick(autoResizeTextarea))

		// Slash command filtering
		watch(companionInput, (val) => {
			if (!val.startsWith('/')) { closeSlash(); return }
			const query = val.slice(1).toLowerCase()
			slashResults.value = SLASH_COMMANDS.filter(
				(c) => c.cmd.startsWith(query) || c.label.toLowerCase().includes(query),
			)
			slashActive.value = slashResults.value.length > 0
			slashIndex.value = 0
		})

		// @mention filtering
		watch(companionInput, (val) => {
			if (_mentionTimer) clearTimeout(_mentionTimer)
			const idx = val.lastIndexOf('@')
			if (idx === -1 || (idx > 0 && !/\s/.test(val[idx - 1]))) {
				closeMention()
				return
			}
			const afterAt = val.slice(idx + 1)
			if (afterAt.length === 0 || (afterAt.match(/\s/g) || []).length > 2) {
				closeMention()
				return
			}
			mentionStart.value = idx
			_mentionTimer = setTimeout(async () => {
				try {
					const res = await performersAPI.getAll(afterAt.trim())
					mentionResults.value = (res?.data || []).slice(0, 6)
					mentionActive.value = mentionResults.value.length > 0
					mentionIndex.value = 0
				} catch {
					closeMention()
				}
			}, 200)
		})
	}

	const cleanupWatchers = () => {
		if (_mentionTimer) clearTimeout(_mentionTimer)
	}

	return {
		companionInput,
		textareaRef,
		slashActive,
		slashResults,
		slashIndex,
		mentionActive,
		mentionResults,
		mentionIndex,
		closeSlash,
		closeMention,
		handleInputKeydown,
		autoResizeTextarea,
		setupWatchers,
		cleanupWatchers,
	}
}
