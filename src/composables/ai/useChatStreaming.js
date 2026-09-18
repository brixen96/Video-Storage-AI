/**
 * useChatStreaming — SSE streaming, tool progress, abort control.
 */
import { ref } from 'vue'
import { API_ORIGIN } from '@/services/api'
import { TOOL_LABELS, FOLLOW_UP_RULES } from './constants'

export function useChatStreaming() {
	const isStreaming = ref(false)
	const activeToolCall = ref('')
	const streamToolSteps = ref([])
	const streamAbortController = ref(null)

	/** Derive follow-up action chips from response text */
	const deriveFollowUpActions = (text) => {
		if (text.length < 80) return []
		const seen = new Set()
		const actions = []
		for (const rule of FOLLOW_UP_RULES) {
			if (rule.pattern.test(text) && !seen.has(rule.label)) {
				seen.add(rule.label)
				actions.push({ label: rule.label, message: rule.message, icon: rule.icon })
				if (actions.length === 3) break
			}
		}
		return actions
	}

	/**
	 * Stream a chat message from the backend SSE endpoint.
	 * Returns an object with { msgs, abort, onComplete } for the caller
	 * to wire into chatHistory and UI state.
	 *
	 * @param {string} fullMessage - The message text (may include page context prefix)
	 * @param {Array} historyPayload - Recent history as {role, content}[]
	 * @param {object} hooks - { onTextChunk, onToolCall, onVideoCards, onDone, onError, scrollFn }
	 */
	const streamMessage = (fullMessage, historyPayload, hooks = {}) => {
		const { onTextChunk, onToolCall, onVideoCards, onDone, onError, scrollFn, onTokenUsage, onToolProgress } = hooks

		isStreaming.value = true
		activeToolCall.value = ''
		streamToolSteps.value = []

		streamAbortController.value = new AbortController()

		const run = async () => {
			try {
				const response = await fetch(`${API_ORIGIN}/api/v1/ai/chat/stream`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ message: fullMessage, history: historyPayload }),
					signal: streamAbortController.value.signal,
				})

				if (!response.ok) throw new Error(`HTTP ${response.status}`)

				const reader = response.body.getReader()
				const decoder = new TextDecoder()
				let buffer = ''
				let scrollThrottle = 0
				let streamDone = false
				let currentEventType = ''

				while (!streamDone) {
					const { done, value } = await reader.read()
					if (done) break
					buffer += decoder.decode(value, { stream: true })
					const lines = buffer.split('\n')
					buffer = lines.pop()

					for (const line of lines) {
						if (line.startsWith('event: ')) {
							currentEventType = line.slice(7).trim()
							continue
						}
						if (!line.startsWith('data: ')) {
							if (line === '') currentEventType = ''
							continue
						}
						const data = line.slice(6)
						if (data === '[DONE]') { streamDone = true; break }

						if (currentEventType === 'video_cards') {
							try {
								const { videos } = JSON.parse(data)
								if (videos?.length > 0) onVideoCards?.(videos)
							} catch { /* ignore */ }
							currentEventType = ''
							continue
						}

						if (currentEventType === 'tool_call') {
							try {
								const { tool } = JSON.parse(data)
								activeToolCall.value = tool
								streamToolSteps.value.push(tool)
								onToolCall?.(tool)
							} catch { /* ignore */ }
							currentEventType = ''
							continue
						}

						if (currentEventType === 'token_usage') {
							try {
								const usage = JSON.parse(data)
								onTokenUsage?.(usage)
							} catch { /* ignore */ }
							currentEventType = ''
							continue
						}

						if (currentEventType === 'tool_progress') {
							try {
								const progress = JSON.parse(data)
								onToolProgress?.(progress)
							} catch { /* ignore */ }
							currentEventType = ''
							continue
						}

						try {
							const chunk = JSON.parse(data)
							if (chunk && activeToolCall.value) activeToolCall.value = ''
							onTextChunk?.(chunk)
							const now = Date.now()
							if (now - scrollThrottle > 100) {
								scrollThrottle = now
								scrollFn?.()
							}
						} catch {
							// ignore malformed chunk
						}
					}
				}

				const finalToolSteps = [...streamToolSteps.value]
				activeToolCall.value = ''
				streamToolSteps.value = []
				isStreaming.value = false
				onDone?.(finalToolSteps)
			} catch (error) {
				activeToolCall.value = ''
				streamToolSteps.value = []
				isStreaming.value = false
				onError?.(error)
			}
		}

		run()

		return {
			abort: () => streamAbortController.value?.abort(),
			controller: streamAbortController,
		}
	}

	const stopStreaming = () => {
		streamAbortController.value?.abort()
	}

	const toolLabel = (name) => TOOL_LABELS[name] || name.replace(/_/g, ' ')

	return {
		isStreaming,
		activeToolCall,
		streamToolSteps,
		streamAbortController,
		deriveFollowUpActions,
		streamMessage,
		stopStreaming,
		toolLabel,
	}
}
