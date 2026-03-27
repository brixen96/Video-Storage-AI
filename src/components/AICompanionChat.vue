<template>
	<div class="ai-companion-float">
		<!-- Floating Chat Button -->
		<button v-if="!isOpen" class="chat-toggle-button" @click="toggleChat" :class="{ 'has-notification': hasNewMessage }">
			<font-awesome-icon :icon="['fas', 'robot']" class="chat-icon" />
			<span v-if="hasNewMessage" class="notification-badge">1</span>
		</button>

		<!-- Chat Window -->
		<transition name="slide-up">
			<div v-if="isOpen" class="chat-window">
				<!-- Chat Header -->
				<div class="chat-header">
					<div class="chat-header-content">
						<font-awesome-icon :icon="['fas', 'robot']" class="header-icon" />
						<div class="header-text">
							<h4>AI Companion</h4>
							<small :class="companionConnected ? 'text-success' : 'text-warning'">
								<font-awesome-icon :icon="['fas', companionConnected ? 'circle' : 'exclamation-circle']" class="status-dot" />
								{{ companionConnected ? 'Online' : 'Connecting...' }}
							</small>
						</div>
					</div>
					<div class="chat-header-actions">
						<button class="btn-header-action" :class="{ active: proactiveEnabled }" @click="toggleProactive" :title="proactiveEnabled ? 'Disable Proactive Alerts' : 'Enable Proactive Alerts'">
							<font-awesome-icon :icon="['fas', proactiveEnabled ? 'bell' : 'bell-slash']" />
						</button>
						<button class="btn-header-action" @click="toggleSearch" title="Search Messages">
							<font-awesome-icon :icon="['fas', 'search']" />
						</button>
						<button class="btn-header-action" @click="exportChat" title="Export Chat">
							<font-awesome-icon :icon="['fas', 'download']" />
						</button>
						<button class="btn-header-action" @click="minimizeChat" title="Minimize">
							<font-awesome-icon :icon="['fas', 'minus']" />
						</button>
						<button class="btn-header-action" @click="closeChat" title="Close">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
				</div>

				<!-- Search Bar -->
				<transition name="slide-down">
					<div v-if="showSearch" class="search-bar">
						<div class="search-input-group">
							<font-awesome-icon :icon="['fas', 'search']" class="search-icon" />
							<input ref="searchInput" v-model="searchQuery" type="text" class="search-input" placeholder="Search messages..." @keyup.esc="clearSearch" />
							<button v-if="searchQuery" class="btn-clear-search" @click="clearSearch">
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
						</div>
						<div v-if="searchQuery" class="search-results-info">Found {{ filteredMessageCount }} message(s)</div>
					</div>
				</transition>

				<!-- Chat Messages -->
				<div class="chat-messages" ref="chatMessages">
					<!-- Suggested Prompts (shown when empty) -->
					<div v-if="chatHistory.length === 0 && !isCompanionThinking" class="suggested-prompts">
						<div class="suggested-header">
							<h5>💡 Try asking me:</h5>
						</div>
						<div class="prompt-grid">
							<button v-for="prompt in suggestedPrompts" :key="prompt.text" class="prompt-card" @click="askQuickQuestion(prompt.text)">
								<div class="prompt-icon">
									<font-awesome-icon :icon="prompt.icon" />
								</div>
								<div class="prompt-text">{{ prompt.text }}</div>
							</button>
						</div>
					</div>

					<!-- Chat Messages -->
					<template v-for="(message, index) in displayedMessages" :key="index">
						<!-- Inline video cards -->
						<div v-if="message.role === 'cards'" class="video-cards-row">
							<div
								v-for="video in message.videos"
								:key="video.id"
								class="video-mini-card"
								@click="navigateToVideo(video.id)"
								:title="video.title"
							>
								<div class="mini-thumb-wrap">
									<img
										v-if="video.thumbnail_path"
										:src="getVideoThumbURL(video.thumbnail_path)"
										class="mini-thumb"
										@error="(e) => e.target.style.display = 'none'"
									/>
									<div class="mini-thumb-fallback">
										<font-awesome-icon :icon="['fas', 'film']" />
									</div>
									<!-- Favorite overlay -->
									<button
										class="mini-fav-btn"
										:class="{ active: video.is_favorite }"
										@click="toggleCardFavorite(video, $event)"
										title="Toggle favorite"
									>
										<font-awesome-icon :icon="['fas', 'heart']" />
									</button>
								</div>
								<div class="mini-info">
									<div class="mini-title">{{ video.title.length > 42 ? video.title.slice(0, 42) + '…' : video.title }}</div>
									<div class="mini-performers" v-if="video.performers">{{ video.performers.split(', ').slice(0, 2).join(', ') }}</div>
									<div class="mini-meta">
										<span class="mini-stars">
											<span
												v-for="n in 5"
												:key="n"
												:class="n <= (video.rating || 0) ? 'star-filled' : 'star-empty'"
												@click.stop="rateCardVideo(video, n, $event)"
												class="star-btn"
												title="Rate"
											>★</span>
										</span>
									</div>
									<div class="mini-actions">
										<button class="mini-action-btn" @click="askAboutCard(video, $event)" title="Suggest tags">
											<font-awesome-icon :icon="['fas', 'tags']" />
										</button>
									</div>
								</div>
							</div>
						</div>

						<!-- Regular text messages -->
						<div v-else class="chat-message" :class="[message.role, message.severity, { 'daily-briefing': message.source === 'daily_briefing', 'library-insights': message.source === 'library_insights' }]">
							<div class="message-avatar" :class="{ 'proactive-avatar': message.role === 'proactive' }">
								<font-awesome-icon :icon="['fas', message.role === 'user' ? 'user' : message.role === 'proactive' ? (message.source === 'daily_briefing' ? 'sun' : message.source === 'library_insights' ? 'heartbeat' : message.source === 'analysis_engine' ? 'lightbulb' : 'bell') : 'robot']" />
							</div>
							<div class="message-content">
								<div class="message-text" :class="{ 'streaming-text': message.streaming }">
									<!-- Use plain text for user messages -->
									<template v-if="message.role === 'user'">{{ message.content }}</template>
									<!-- Use markdown rendering for AI responses -->
									<MarkdownMessage v-else :content="message.content" />
								</div>
								<div class="message-footer">
									<div class="message-time">{{ formatMessageTime(message.timestamp) }}</div>
									<span v-if="message.source && message.role === 'assistant'" class="message-source" :class="message.source">
										<font-awesome-icon :icon="['fas', message.source === 'llm' ? 'brain' : 'cog']" />
										{{ message.source === 'llm' ? 'LLM' : 'Rule' }}
									</span>
									<button class="btn-copy-message" @click="copyMessage(message.content)" title="Copy message">
										<font-awesome-icon :icon="['fas', 'copy']" />
									</button>
								</div>
							</div>
						</div>
					</template>
					<div v-if="isCompanionThinking && !isStreaming" class="chat-message assistant thinking">
						<div class="message-avatar">
							<font-awesome-icon :icon="['fas', 'robot']" spin />
						</div>
						<div class="message-content">
							<div class="message-text">
								<span class="typing-indicator">
									<span></span>
									<span></span>
									<span></span>
								</span>
							</div>
						</div>
					</div>
				</div>

				<!-- Statistics Panel -->
				<transition name="slide-down">
					<div v-if="showStats && chatHistory.length > 0" class="stats-panel">
						<div class="stats-header">
							<h6>📊 Conversation Stats</h6>
							<button class="btn-close-stats" @click="showStats = false">
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
						</div>
						<div class="stats-grid">
							<div class="stat-item">
								<div class="stat-icon">
									<font-awesome-icon :icon="['fas', 'comments']" />
								</div>
								<div class="stat-details">
									<div class="stat-value">{{ chatStats.totalMessages }}</div>
									<div class="stat-label">Total Messages</div>
								</div>
							</div>
							<div class="stat-item">
								<div class="stat-icon user">
									<font-awesome-icon :icon="['fas', 'user']" />
								</div>
								<div class="stat-details">
									<div class="stat-value">{{ chatStats.userMessages }}</div>
									<div class="stat-label">Your Questions</div>
								</div>
							</div>
							<div class="stat-item">
								<div class="stat-icon ai">
									<font-awesome-icon :icon="['fas', 'robot']" />
								</div>
								<div class="stat-details">
									<div class="stat-value">{{ chatStats.aiMessages }}</div>
									<div class="stat-label">AI Responses</div>
								</div>
							</div>
							<div class="stat-item">
								<div class="stat-icon">
									<font-awesome-icon :icon="['fas', 'clock']" />
								</div>
								<div class="stat-details">
									<div class="stat-value">{{ chatStats.sessionDuration }}</div>
									<div class="stat-label">Session Time</div>
								</div>
							</div>
							<div class="stat-item">
								<div class="stat-icon">
									<font-awesome-icon :icon="['fas', 'text-width']" />
								</div>
								<div class="stat-details">
									<div class="stat-value">{{ chatStats.avgMessageLength }}</div>
									<div class="stat-label">Avg Length</div>
								</div>
							</div>
						</div>
					</div>
				</transition>

				<!-- Quick Actions -->
				<div class="quick-actions-bar">
					<button class="btn-quick-action" @click="askQuickQuestion('What\'s happening right now?')" title="Current Status">
						<font-awesome-icon :icon="['fas', 'bolt']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('JDownloader status')" title="JDownloader">
						<font-awesome-icon :icon="['fas', 'download']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Scraper summary')" title="Scrapers">
						<font-awesome-icon :icon="['fas', 'spider']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Library overview')" title="Library">
						<font-awesome-icon :icon="['fas', 'chart-bar']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Check for errors')" title="Errors">
						<font-awesome-icon :icon="['fas', 'exclamation-triangle']" />
					</button>
					<button class="btn-quick-action" @click="showStats = !showStats" :class="{ active: showStats }" title="Chat Stats">
						<font-awesome-icon :icon="['fas', 'chart-line']" />
					</button>
					<button class="btn-quick-action btn-danger" @click="clearChatHistory" title="Clear Chat">
						<font-awesome-icon :icon="['fas', 'trash']" />
					</button>
				</div>

				<!-- Chat Input -->
				<div class="chat-input-container">
					<div class="input-group">
						<input
							v-model="companionInput"
							type="text"
							class="form-control"
							placeholder="Ask me anything..."
							@keyup.enter="sendCompanionMessage"
							:disabled="isCompanionThinking"
						/>
						<button class="btn btn-voice-input" @click="toggleVoiceInput" :class="{ listening: isListening }" :disabled="isCompanionThinking" title="Voice Input">
							<font-awesome-icon :icon="['fas', isListening ? 'stop' : 'microphone']" />
							<span v-if="isListening" class="listening-pulse"></span>
						</button>
						<button class="btn btn-primary" @click="sendCompanionMessage" :disabled="isCompanionThinking || !companionInput.trim()">
							<font-awesome-icon :icon="['fas', isCompanionThinking ? 'spinner' : 'paper-plane']" :spin="isCompanionThinking" />
						</button>
					</div>
				</div>
			</div>
		</transition>
	</div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted, getCurrentInstance } from 'vue'
import { useRouter } from 'vue-router'
import { aiCompanionAPI, aiAPI, getAssetURL, videoAPI } from '@/services/api'
import websocketService from '@/services/websocket'
import MarkdownMessage from './MarkdownMessage.vue'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const router = useRouter()

// Video card helpers
const getVideoThumbURL = (thumbnailPath) => {
	if (!thumbnailPath) return null
	return getAssetURL(thumbnailPath)
}
const navigateToVideo = (videoId) => {
	router.push({ path: '/videos', query: { highlight: videoId } })
	isOpen.value = false
}
const rateCardVideo = async (video, rating, event) => {
	event.stopPropagation()
	const newRating = video.rating === rating ? 0 : rating // click same star = clear rating
	try {
		await videoAPI.update(video.id, { rating: newRating })
		video.rating = newRating
	} catch {
		toast.error('Error', 'Failed to update rating')
	}
}
const toggleCardFavorite = async (video, event) => {
	event.stopPropagation()
	const newFav = !video.is_favorite
	try {
		await videoAPI.update(video.id, { is_favorite: newFav })
		video.is_favorite = newFav
	} catch {
		toast.error('Error', 'Failed to update favorite')
	}
}
const askAboutCard = (video, event) => {
	event.stopPropagation()
	companionInput.value = `Suggest tags for video ID ${video.id}: "${video.title}"`
	nextTick(() => sendCompanionMessage())
}

// State
const isOpen = ref(false)
const companionInput = ref('')
const chatHistory = ref([])
const isCompanionThinking = ref(false)
const isStreaming = ref(false)
const companionConnected = ref(false)
const chatMessages = ref(null)
const hasNewMessage = ref(false)
const showSearch = ref(false)
const searchQuery = ref('')
const searchInput = ref(null)
const filteredMessageCount = computed(() => {
	if (!searchQuery.value) return 0
	const q = searchQuery.value.toLowerCase()
	return chatHistory.value.filter(m => m.content.toLowerCase().includes(q)).length
})
const isListening = ref(false)
const recognition = ref(null)
const typingText = ref('')
const isTyping = ref(false)
const showStats = ref(false)
const proactiveEnabled = ref(localStorage.getItem('ai_companion_proactive') !== 'false')
let unsubscribeCompanionEvent = null

// Conversation statistics
const chatStats = computed(() => {
	const totalMessages = chatHistory.value.length
	const userMessages = chatHistory.value.filter((m) => m.role === 'user').length
	const aiMessages = chatHistory.value.filter((m) => m.role === 'assistant').length
	const avgMessageLength = totalMessages > 0 ? Math.round(chatHistory.value.reduce((sum, m) => sum + m.content.length, 0) / totalMessages) : 0

	// Calculate session duration if there are messages
	let sessionDuration = 'N/A'
	if (totalMessages > 0) {
		const firstMessage = chatHistory.value[0].timestamp
		const lastMessage = chatHistory.value[totalMessages - 1].timestamp
		const diffMs = new Date(lastMessage) - new Date(firstMessage)
		const diffMins = Math.floor(diffMs / 60000)
		if (diffMins < 60) {
			sessionDuration = `${diffMins}m`
		} else {
			const hours = Math.floor(diffMins / 60)
			const mins = diffMins % 60
			sessionDuration = `${hours}h ${mins}m`
		}
	}

	return {
		totalMessages,
		userMessages,
		aiMessages,
		avgMessageLength,
		sessionDuration,
	}
})

// eslint-disable-next-line no-unused-vars
const displayedMessages = computed(() => {
	if (!searchQuery.value) return chatHistory.value

	const query = searchQuery.value.toLowerCase()
	const filtered = chatHistory.value
		.map((msg, index) => ({
			...msg,
			highlighted: msg.content.toLowerCase().includes(query),
			originalIndex: index,
		}))
		.filter((msg) => msg.highlighted)

	return filtered.length > 0 ? filtered : chatHistory.value
})

// Suggested prompts - context-aware
const suggestedPrompts = ref([
	{
		text: "What's happening right now?",
		icon: ['fas', 'bolt'],
	},
	{
		text: 'JDownloader status',
		icon: ['fas', 'download'],
	},
	{
		text: 'Scraper summary',
		icon: ['fas', 'spider'],
	},
	{
		text: 'Library overview',
		icon: ['fas', 'chart-bar'],
	},
	{
		text: 'Check for errors',
		icon: ['fas', 'exclamation-triangle'],
	},
	{
		text: 'Show insights',
		icon: ['fas', 'lightbulb'],
	},
	{
		text: 'What are my top performers?',
		icon: ['fas', 'users'],
	},
	{
		text: 'Show my favorite videos',
		icon: ['fas', 'heart'],
	},
	{
		text: 'Show my top rated videos',
		icon: ['fas', 'star'],
	},
	{
		text: 'Find duplicate videos in my library',
		icon: ['fas', 'copy'],
	},
	{
		text: "What's in my Downloads library?",
		icon: ['fas', 'download'],
	},
	{
		text: 'Help me organize my downloads into Backup',
		icon: ['fas', 'folder-open'],
	},
	{
		text: "Give me today's briefing",
		icon: ['fas', 'sun'],
	},
	{
		text: 'What should I watch next?',
		icon: ['fas', 'play-circle'],
	},
	{
		text: 'Tell me about my top performer',
		icon: ['fas', 'user-circle'],
	},
	{
		text: 'Scan for orphaned files in my library',
		icon: ['fas', 'exclamation-triangle'],
	},
	{
		text: "How's my library doing?",
		icon: ['fas', 'heartbeat'],
	},
	{
		text: 'Auto-tag my untagged videos',
		icon: ['fas', 'tags'],
	},
	{
		text: 'Rate a video for me',
		icon: ['fas', 'star'],
	},
])

// Load chat history from backend DB (fallback to localStorage for legacy data)
const loadChatHistory = async () => {
	try {
		const { data: res } = await aiAPI.getChatHistory(100)
		if (res.data && res.data.length > 0) {
			chatHistory.value = res.data.map((m) => ({
				role: m.role,
				content: m.content,
				timestamp: new Date(m.created_at),
				source: m.source !== 'user' ? m.source : undefined,
			}))
			// Clear any stale localStorage copy
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
			const parsed = JSON.parse(saved)
			chatHistory.value = parsed.slice(-50).map((msg) => ({
				...msg,
				timestamp: new Date(msg.timestamp),
			}))
		}
	} catch {
		// ignore
	}
}

// No-op kept for call sites that predate DB persistence
const saveChatHistory = () => {}

// Functions
const toggleChat = () => {
	isOpen.value = !isOpen.value
	hasNewMessage.value = false
	if (isOpen.value) {
		nextTick(() => scrollToBottom())
	}
}

const minimizeChat = () => {
	isOpen.value = false
}

const closeChat = () => {
	isOpen.value = false
}

const checkCompanionConnection = async () => {
	try {
		await aiCompanionAPI.chat(
			[
				{
					role: 'user',
					content: 'Hi',
				},
			],
			{ max_tokens: 5 }
		)
		companionConnected.value = true
	} catch (error) {
		// LM Studio not available, will use backend AI instead
		companionConnected.value = true // Still mark as connected since we have fallback
	}
}

const sendCompanionMessage = async () => {
	const userMessage = companionInput.value.trim()
	if (!userMessage) return

	// Add user message to chat
	chatHistory.value.push({
		role: 'user',
		content: userMessage,
		timestamp: new Date(),
	})

	saveChatHistory()
	companionInput.value = ''
	isCompanionThinking.value = true

	await nextTick()
	scrollToBottom()

	try {
		const historyPayload = chatHistory.value
			.slice(0, -1) // exclude the user message we just pushed
			.filter((msg) => msg.role === 'user' || msg.role === 'assistant')
			.slice(-10)
			.map((msg) => ({ role: msg.role, content: msg.content }))

		const response = await fetch('http://localhost:8080/api/v1/ai/chat/stream', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ message: userMessage, history: historyPayload }),
		})

		if (!response.ok) throw new Error(`HTTP ${response.status}`)

		// Push a placeholder message that will grow as chunks arrive
		const msgIndex = chatHistory.value.length
		chatHistory.value.push({
			role: 'assistant',
			content: '',
			timestamp: new Date(),
			source: 'llm',
			streaming: true,
		})
		isCompanionThinking.value = false
		isStreaming.value = true

		await nextTick()
		scrollToBottom()

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
			buffer = lines.pop() // keep any incomplete line

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
						if (videos && videos.length > 0) {
							chatHistory.value.push({
								role: 'cards',
								source: 'video_cards',
								videos,
								timestamp: new Date(),
							})
							await nextTick()
							scrollToBottom()
						}
					} catch { /* ignore */ }
					currentEventType = ''
					continue
				}

				try {
					const chunk = JSON.parse(data)
					chatHistory.value[msgIndex].content += chunk
					// Throttle scroll to every ~100ms to avoid layout thrash
					const now = Date.now()
					if (now - scrollThrottle > 100) {
						scrollThrottle = now
						await nextTick()
						scrollToBottom()
					}
				} catch {
					// ignore malformed chunk
				}
			}
		}

		chatHistory.value[msgIndex].streaming = false
		isStreaming.value = false
		saveChatHistory()

		if (!isOpen.value) hasNewMessage.value = true
		await nextTick()
		scrollToBottom()
	} catch (error) {
		console.error('Backend AI failed:', error)
		isCompanionThinking.value = false
		isStreaming.value = false
		chatHistory.value.push({
			role: 'assistant',
			content: "I'm currently unavailable. Please check that the backend server is running and try again.",
			timestamp: new Date(),
		})
		toast.error('AI Unavailable', 'Could not connect to AI Companion service. Please check the backend server.')
	} finally {
		isCompanionThinking.value = false
		isStreaming.value = false
	}
}

const askQuickQuestion = (question) => {
	companionInput.value = question
	sendCompanionMessage()
}

const clearChatHistory = async () => {
	if (chatHistory.value.length === 0) return
	if (!confirm('Are you sure you want to clear the chat history?')) return
	try {
		await aiAPI.clearChatHistory()
	} catch {
		// best-effort
	}
	chatHistory.value = []
	localStorage.removeItem('ai_chat_history')
	toast.info('Chat Cleared', 'Chat history has been cleared')
}

// Search functions
// eslint-disable-next-line no-unused-vars
const toggleSearch = async () => {
	showSearch.value = !showSearch.value
	if (showSearch.value) {
		await nextTick()
		searchInput.value?.focus()
	} else {
		clearSearch()
	}
}

// eslint-disable-next-line no-unused-vars
const filterMessages = () => {
	// The computed property handles this automatically
}

const clearSearch = () => {
	searchQuery.value = ''
}

// Export chat
// eslint-disable-next-line no-unused-vars
const exportChat = () => {
	const chatData = {
		exported_at: new Date().toISOString(),
		message_count: chatHistory.value.length,
		messages: chatHistory.value,
	}

	const dataStr = JSON.stringify(chatData, null, 2)
	const dataBlob = new Blob([dataStr], { type: 'application/json' })
	const url = URL.createObjectURL(dataBlob)

	const link = document.createElement('a')
	link.href = url
	link.download = `ai-companion-chat-${new Date().toISOString().split('T')[0]}.json`
	link.click()

	URL.revokeObjectURL(url)
	toast.success('Chat Exported', 'Your conversation has been exported successfully')
}

// Copy message
// eslint-disable-next-line no-unused-vars
const copyMessage = async (content) => {
	try {
		await navigator.clipboard.writeText(content)
		toast.success('Copied!', 'Message copied to clipboard')
	} catch (err) {
		console.error('Failed to copy:', err)
		toast.error('Copy Failed', 'Could not copy message to clipboard')
	}
}

// Regenerate response
// eslint-disable-next-line no-unused-vars
const regenerateResponse = async (index) => {
	if (index < 1) return // Need at least one user message before this

	// Find the user message that triggered this response
	const userMessageIndex = index - 1
	if (userMessageIndex < 0 || chatHistory.value[userMessageIndex].role !== 'user') return

	// Remove the old response
	chatHistory.value.splice(index, 1)

	// Resend the user message
	companionInput.value = chatHistory.value[userMessageIndex].content
	// Remove the user message too so it doesn't duplicate
	chatHistory.value.splice(userMessageIndex, 1)

	// Send again
	await sendCompanionMessage()
}

// Voice input functions
const initSpeechRecognition = () => {
	if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
		const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition
		recognition.value = new SpeechRecognition()
		recognition.value.continuous = false
		recognition.value.interimResults = true
		recognition.value.lang = 'en-US'

		recognition.value.onstart = () => {
			isListening.value = true
			companionInput.value = ''
		}

		recognition.value.onresult = (event) => {
			const transcript = Array.from(event.results)
				.map((result) => result[0])
				.map((result) => result.transcript)
				.join('')

			companionInput.value = transcript

			// If result is final, send the message
			if (event.results[0].isFinal) {
				sendCompanionMessage()
			}
		}

		recognition.value.onerror = (event) => {
			console.error('Speech recognition error:', event.error)
			isListening.value = false
			toast.error('Voice Input Error', `Could not recognize speech: ${event.error}`)
		}

		recognition.value.onend = () => {
			isListening.value = false
		}
	}
}

const toggleVoiceInput = () => {
	if (!recognition.value) {
		initSpeechRecognition()
	}

	if (!recognition.value) {
		toast.error('Not Supported', 'Voice input is not supported in your browser')
		return
	}

	if (isListening.value) {
		recognition.value.stop()
	} else {
		recognition.value.start()
	}
}

// Typing animation for AI responses
// eslint-disable-next-line no-unused-vars
const typeMessage = async (message, speed = 30) => {
	isTyping.value = true
	typingText.value = ''

	for (let i = 0; i < message.length; i++) {
		typingText.value += message[i]
		await new Promise((resolve) => setTimeout(resolve, speed))
	}

	isTyping.value = false
	return message
}

// eslint-disable-next-line no-unused-vars
const toggleProactive = () => {
	proactiveEnabled.value = !proactiveEnabled.value
	localStorage.setItem('ai_companion_proactive', proactiveEnabled.value ? 'true' : 'false')
}

const handleCompanionEvent = (event) => {
	if (!proactiveEnabled.value) return

	// Add proactive message to chat
	chatHistory.value.push({
		role: 'proactive',
		content: event.message,
		timestamp: new Date(),
		severity: event.type || event.severity || 'info',
		source: event.source || 'system',
	})
	saveChatHistory()

	// Show notification badge if chat is closed
	if (!isOpen.value) {
		hasNewMessage.value = true
	}

	nextTick(() => scrollToBottom())
}

const formatMessageTime = (timestamp) => {
	const date = new Date(timestamp)
	return date.toLocaleTimeString('en-US', {
		hour: '2-digit',
		minute: '2-digit',
	})
}

const scrollToBottom = () => {
	if (chatMessages.value) {
		chatMessages.value.scrollTop = chatMessages.value.scrollHeight
	}
}

// Handler for the global open-ai-chat event (from ScraperPage Ask AI button)
const handleOpenAIChat = (event) => {
	isOpen.value = true
	hasNewMessage.value = false
	const msg = event.detail?.message
	if (msg) {
		companionInput.value = msg
		nextTick(() => {
			sendCompanionMessage()
		})
	}
	nextTick(() => scrollToBottom())
}

// Initialize on mount
onMounted(async () => {
	// Load persistent chat history from backend
	await loadChatHistory()

	await checkCompanionConnection()

	// Subscribe to proactive companion events via WebSocket
	unsubscribeCompanionEvent = websocketService.on('companion_event', handleCompanionEvent)

	// Listen for external requests to open with a pre-filled message
	window.addEventListener('open-ai-chat', handleOpenAIChat)

	// Add welcome message only if no chat history exists
	if (companionConnected.value && chatHistory.value.length === 0) {
		chatHistory.value.push({
			role: 'assistant',
			content: `# 👋 Hello! I'm your AI Companion

I'm the **intelligent core** of Video Storage AI with advanced capabilities:

## 🎯 What I Can Do:
- 📊 **Analyze** your video library health and statistics
- 🔍 **Find** duplicates, performers, and specific content
- ❤️ **List** your favorites and top-rated videos
- 📈 **Predict** library growth and provide insights
- ⭐ **Evaluate** performer quality and metadata completeness
- 🚨 **Monitor** for errors and issues
- 💡 **Suggest** optimizations and improvements

## 🧠 Smart Features:
- **Markdown Support** - I can format responses beautifully
- **Code Highlighting** - Share code snippets with syntax colors
- **Memory System** - I remember our conversations
- **Real-time Data** - All answers are from your actual database

Try one of the suggested prompts above, or ask me anything about your library!`,
			timestamp: new Date(),
		})
		saveChatHistory()
	}
})

onUnmounted(() => {
	if (unsubscribeCompanionEvent) {
		unsubscribeCompanionEvent()
	}
	window.removeEventListener('open-ai-chat', handleOpenAIChat)
})
</script>

<style scoped>
@import '@/styles/components/ai_companion_chat.css';
</style>
