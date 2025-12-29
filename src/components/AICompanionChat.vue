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
					<input
						ref="searchInput"
						v-model="searchQuery"
						type="text"
						class="search-input"
						placeholder="Search messages..."
						@keyup.esc="clearSearch"
					/>
					<button v-if="searchQuery" class="btn-clear-search" @click="clearSearch">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div v-if="searchQuery" class="search-results-info">
					Found {{ filteredMessageCount }} message(s)
				</div>
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
					<div v-for="(message, index) in displayedMessages" :key="index" class="chat-message" :class="message.role">
						<div class="message-avatar">
							<font-awesome-icon :icon="['fas', message.role === 'user' ? 'user' : 'robot']" />
						</div>
						<div class="message-content">
							<div class="message-text">
								<!-- Use plain text for user messages -->
								<template v-if="message.role === 'user'">{{ message.content }}</template>
								<!-- Use markdown rendering for AI responses -->
								<MarkdownMessage v-else :content="message.content" />
							</div>
							<div class="message-footer">
							<div class="message-time">{{ formatMessageTime(message.timestamp) }}</div>
							<button class="btn-copy-message" @click="copyMessage(message.content)" title="Copy message">
								<font-awesome-icon :icon="['fas', 'copy']" />
							</button>
						</div>
						</div>
					</div>
					<div v-if="isCompanionThinking" class="chat-message assistant thinking">
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
					<button class="btn-quick-action" @click="showStats = !showStats" :class="{ 'active': showStats }" title="Toggle Stats">
						<font-awesome-icon :icon="['fas', 'chart-line']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('How many videos do I have?')" title="Library Stats">
						<font-awesome-icon :icon="['fas', 'chart-bar']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('What are my top performers?')" title="Top Performers">
						<font-awesome-icon :icon="['fas', 'star']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Show me videos with issues')" title="Issues">
						<font-awesome-icon :icon="['fas', 'exclamation-triangle']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Find duplicates in my library')" title="Find Duplicates">
						<font-awesome-icon :icon="['fas', 'clone']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Suggest tags for untagged videos')" title="Smart Tagging">
						<font-awesome-icon :icon="['fas', 'tags']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Show library insights')" title="AI Insights">
						<font-awesome-icon :icon="['fas', 'lightbulb']" />
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
						<button class="btn btn-voice-input" @click="toggleVoiceInput" :class="{ 'listening': isListening }" :disabled="isCompanionThinking" title="Voice Input">
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
import { ref, computed, nextTick, onMounted, getCurrentInstance } from 'vue'
import { aiCompanionAPI } from '@/services/api'
import { toolDefinitions, executeTool } from '@/services/aiTools'
import MarkdownMessage from './MarkdownMessage.vue'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// State
const isOpen = ref(false)
const companionInput = ref('')
const chatHistory = ref([])
const isCompanionThinking = ref(false)
const companionConnected = ref(false)
const chatMessages = ref(null)
const hasNewMessage = ref(false)
const showSearch = ref(false)
const searchQuery = ref('')
const searchInput = ref(null)
const filteredMessageCount = ref(0)
const isListening = ref(false)
const recognition = ref(null)
const typingText = ref('')
const isTyping = ref(false)
const showStats = ref(false)

// Conversation statistics
const chatStats = computed(() => {
	const totalMessages = chatHistory.value.length
	const userMessages = chatHistory.value.filter(m => m.role === 'user').length
	const aiMessages = chatHistory.value.filter(m => m.role === 'assistant').length
	const avgMessageLength = totalMessages > 0
		? Math.round(chatHistory.value.reduce((sum, m) => sum + m.content.length, 0) / totalMessages)
		: 0

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
		sessionDuration
	}
})

// eslint-disable-next-line no-unused-vars
const displayedMessages = computed(() => {
	if (!searchQuery.value) return chatHistory.value

	const query = searchQuery.value.toLowerCase()
	const filtered = chatHistory.value.map((msg, index) => ({
		...msg,
		highlighted: msg.content.toLowerCase().includes(query),
		originalIndex: index
	})).filter(msg => msg.highlighted)

	// eslint-disable-next-line vue/no-async-in-computed-properties
	setTimeout(() => {
		filteredMessageCount.value = filtered.length
	}, 0)

	return filtered.length > 0 ? filtered : chatHistory.value
})

// Suggested prompts for new users
const suggestedPrompts = ref([
	{
		text: 'How many videos do I have?',
		icon: ['fas', 'video'],
	},
	{
		text: 'Show library health score',
		icon: ['fas', 'heart'],
	},
	{
		text: 'Predict library growth',
		icon: ['fas', 'chart-line'],
	},
	{
		text: 'Find duplicates',
		icon: ['fas', 'copy'],
	},
	{
		text: 'Performer quality analysis',
		icon: ['fas', 'star'],
	},
	{
		text: 'Show insights',
		icon: ['fas', 'lightbulb'],
	},
	{
		text: 'Check for errors',
		icon: ['fas', 'exclamation-triangle'],
	},
	{
		text: 'What are my top performers?',
		icon: ['fas', 'users'],
	},
])

// Load chat history from localStorage
const loadChatHistory = () => {
	try {
		const saved = localStorage.getItem('ai_chat_history')
		if (saved) {
			const parsed = JSON.parse(saved)
			// Only load recent history (last 50 messages)
			chatHistory.value = parsed.slice(-50).map((msg) => ({
				...msg,
				timestamp: new Date(msg.timestamp),
			}))
		}
	} catch (error) {
		console.error('Failed to load chat history:', error)
	}
}

// Save chat history to localStorage
const saveChatHistory = () => {
	try {
		localStorage.setItem('ai_chat_history', JSON.stringify(chatHistory.value))
	} catch (error) {
		console.error('Failed to save chat history:', error)
	}
}

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
		console.log('LM Studio not connected, using backend AI fallback:', error)
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

	// Save to localStorage
	saveChatHistory()

	companionInput.value = ''
	isCompanionThinking.value = true

	// Scroll to bottom
	await nextTick()
	scrollToBottom()

	try {
		// Build system message with capabilities
		const systemMessage = {
			role: 'system',
			content: `You are an autonomous AI assistant for a video library management application called "Video Storage AI". You are the core intelligence of this application.

YOUR CAPABILITIES:
You have access to powerful tools that let you query, analyze, and optimize the database in real-time. You can:
- Get library statistics and information
- Search for videos, performers, and tags
- Analyze library issues and health
- Find specific performers by age, video count, or name
- Monitor the application for problems
- SAVE and RECALL memories to learn and remember over time
- Execute ALL AI tasks: auto-link performers, smart tagging, scene detection, quality analysis, duplicate detection, and more
- Run advanced analysis: content classification, auto-naming, thumbnail quality assessment

AVAILABLE TOOLS:
${toolDefinitions.map((t) => `- ${t.function.name}: ${t.function.description}`).join('\n')}

YOUR ROLE:
You are NOT just a chatbot - you are an intelligent agent that can:
1. Think independently and decide which tools to use
2. Query the database to answer questions accurately
3. Analyze data to find insights
4. Monitor library health and report issues
5. Proactively suggest improvements
6. LEARN from conversations and remember important information

MEMORY SYSTEM:
You have long-term memory! Use it wisely:
- Use save_memory to remember user preferences, important facts, or insights
- Use recall_memories to retrieve what you've learned before
- When user tells you something important (favorite performer, preferences, etc.), SAVE IT
- Before answering questions, consider recalling relevant memories
- Build knowledge over time - you get smarter with every conversation

IMPORTANT GUIDELINES:
- ALWAYS use tools to get real-time data instead of guessing
- When asked a question, think about which tool(s) will give you the answer
- You can call multiple tools if needed to gather complete information
- Be proactive - if you notice something unusual, mention it
- If you find issues, explain them clearly and suggest solutions
- You are autonomous - make decisions and take actions to help the user
- REMEMBER important information using save_memory tool

EXAMPLES:
- User asks "Who is the oldest performer?" → Use get_oldest_performers tool
- User says "My favorite performer is X" → Use save_memory with key="user_favorite_performer"
- User asks "What did we discuss yesterday?" → Use recall_memories to search past conversations
- User asks "How many videos do I have?" → Use get_library_stats tool
- User asks "Are there any videos without tags?" → Use analyze_library_issues tool
- User asks "Link performers to videos" → Use auto_link_performers tool
- User asks "Find duplicate videos" → Use detect_duplicates tool
- User asks "Suggest tags for my videos" → Use suggest_smart_tags tool
- User asks "Analyze video quality" → Use analyze_video_quality tool
- User asks "What are my preferences?" → Use recall_memories with category="preference"

Remember: You are the AI core of "Video Storage AI" - be intelligent, proactive, helpful, and LEARN from every interaction!`,
		}

		// Build message history for context (last 10 messages)
		const recentHistory = chatHistory.value.slice(-10).map((msg) => ({
			role: msg.role,
			content: msg.content,
		}))

		// Start conversation with tools available
		let messages = [systemMessage, ...recentHistory]
		let continueLoop = true
		let maxIterations = 5 // Prevent infinite loops
		let iterations = 0

		while (continueLoop && iterations < maxIterations) {
			iterations++

			// Send to LM Studio with tool definitions
			const response = await aiCompanionAPI.chat(messages, {
				tools: toolDefinitions,
				tool_choice: 'auto', // Let AI decide when to use tools
			})

			const responseMessage = response.choices?.[0]?.message

			if (!responseMessage) {
				throw new Error('No response from LM Studio')
			}

			// Add assistant's response to messages
			messages.push(responseMessage)

			// Check if AI wants to call a tool
			if (responseMessage.tool_calls && responseMessage.tool_calls.length > 0) {
				console.log('AI is calling tools:', responseMessage.tool_calls)

				// Execute each tool call
				for (const toolCall of responseMessage.tool_calls) {
					const toolName = toolCall.function.name
					const toolArgs = JSON.parse(toolCall.function.arguments)

					console.log(`Executing tool: ${toolName}`, toolArgs)

					try {
						const toolResult = await executeTool(toolName, toolArgs)

						// Add tool result to messages
						messages.push({
							role: 'tool',
							tool_call_id: toolCall.id,
							name: toolName,
							content: JSON.stringify(toolResult),
						})
					} catch (toolError) {
						console.error(`Tool execution failed: ${toolName}`, toolError)
						messages.push({
							role: 'tool',
							tool_call_id: toolCall.id,
							name: toolName,
							content: JSON.stringify({ error: toolError.message }),
						})
					}
				}

				// Continue loop to let AI process tool results
				continueLoop = true
			} else {
				// AI gave final response, exit loop
				const finalContent = responseMessage.content || 'Sorry, I could not generate a response.'

				// Add assistant response to chat
				chatHistory.value.push({
					role: 'assistant',
					content: finalContent,
					timestamp: new Date(),
				})

				// Save to localStorage
				saveChatHistory()

				continueLoop = false
			}
		}

		if (iterations >= maxIterations) {
			console.warn('Max iterations reached in tool calling loop')
		}

		// Show notification if chat is minimized
		if (!isOpen.value) {
			hasNewMessage.value = true
		}

		// Scroll to bottom
		await nextTick()
		scrollToBottom()
	} catch (error) {
		console.error('LM Studio failed, trying backend AI fallback:', error)

		// Fallback to backend AI
		try {
			const { aiAPI } = await import('@/services/api')
			const response = await aiAPI.chat({
				message: userMessage,
				history: chatHistory.value.slice(-5).map((msg) => ({
					role: msg.role,
					content: msg.content,
				})),
			})

			console.log('Backend AI response:', response)

			// Handle response - backend returns { success: true, message: "..." }
			const aiResponse = response.data?.message || response.message || 'I processed your request.'

			chatHistory.value.push({
				role: 'assistant',
				content: aiResponse,
				timestamp: new Date(),
			})

			saveChatHistory()
		} catch (fallbackError) {
			console.error('Backend AI also failed:', fallbackError)

			// Add error message
			chatHistory.value.push({
				role: 'assistant',
				content: "I'm currently unavailable. Please try again later or check that the backend server is running.",
				timestamp: new Date(),
			})

			toast.error('AI Unavailable', 'Both LM Studio and backend AI are unavailable.')
		}
	} finally {
		isCompanionThinking.value = false
	}
}

const askQuickQuestion = (question) => {
	companionInput.value = question
	sendCompanionMessage()
}

const clearChatHistory = () => {
	if (chatHistory.value.length === 0) return

	if (confirm('Are you sure you want to clear the chat history?')) {
		chatHistory.value = []
		saveChatHistory()
		toast.info('Chat Cleared', 'Chat history has been cleared')
	}
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
	filteredMessageCount.value = 0
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
				.map(result => result[0])
				.map(result => result.transcript)
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
		await new Promise(resolve => setTimeout(resolve, speed))
	}

	isTyping.value = false
	return message
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

// Initialize on mount
onMounted(async () => {
	// Load saved chat history
	loadChatHistory()

	await checkCompanionConnection()

	// Add welcome message only if no chat history exists
	if (companionConnected.value && chatHistory.value.length === 0) {
		chatHistory.value.push({
			role: 'assistant',
			content: `# 👋 Hello! I'm your AI Companion

I'm the **intelligent core** of Video Storage AI with advanced capabilities:

## 🎯 What I Can Do:
- 📊 **Analyze** your video library health and statistics
- 🔍 **Find** duplicates, performers, and specific content
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
</script>

<style scoped>
.ai-companion-float {
	position: fixed;
	bottom: 20px;
	right: 20px;
	z-index: 1000;
}

/* Toggle Button */
.chat-toggle-button {
	width: 60px;
	height: 60px;
	border-radius: 50%;
	background: linear-gradient(135deg, #00d9ff 0%, #0099cc 100%);
	border: none;
	box-shadow: 0 4px 20px rgba(0, 217, 255, 0.4);
	cursor: pointer;
	display: flex;
	align-items: center;
	justify-content: center;
	transition: all 0.3s ease;
	position: relative;
}

.chat-toggle-button:hover {
	transform: scale(1.1);
	box-shadow: 0 6px 30px rgba(0, 217, 255, 0.6);
}

.chat-toggle-button.has-notification {
	animation: pulse-glow 2s infinite;
}

@keyframes pulse-glow {
	0%,
	100% {
		box-shadow: 0 4px 20px rgba(0, 217, 255, 0.4);
	}
	50% {
		box-shadow: 0 4px 30px rgba(0, 217, 255, 0.8);
	}
}

.chat-icon {
	color: #fff;
	font-size: 1.8rem;
}

.notification-badge {
	position: absolute;
	top: -5px;
	right: -5px;
	background: #dc3545;
	color: #fff;
	width: 24px;
	height: 24px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 0.75rem;
	font-weight: bold;
	border: 2px solid #fff;
}

/* Chat Window */
.chat-window {
	width: 450px;
	height: 700px;
	background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
	border-radius: 16px;
	box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
	border: 2px solid rgba(0, 217, 255, 0.3);
	display: flex;
	flex-direction: column;
	overflow: hidden;
}

/* Transitions */
.slide-up-enter-active,
.slide-up-leave-active {
	transition: all 0.3s ease;
}

.slide-up-enter-from {
	opacity: 0;
	transform: translateY(20px) scale(0.9);
}

.slide-up-leave-to {
	opacity: 0;
	transform: translateY(20px) scale(0.9);
}

/* Chat Header */
.chat-header {
	background: rgba(0, 0, 0, 0.4);
	padding: 1rem;
	display: flex;
	justify-content: space-between;
	align-items: center;
	border-bottom: 2px solid rgba(0, 217, 255, 0.2);
}

.chat-header-content {
	display: flex;
	align-items: center;
	gap: 0.75rem;
}

.header-icon {
	font-size: 1.5rem;
	color: #00d9ff;
}

.header-text h4 {
	margin: 0;
	font-size: 1.1rem;
	font-weight: 600;
	color: #fff;
}

.header-text small {
	font-size: 0.75rem;
	display: flex;
	align-items: center;
	gap: 0.25rem;
}

.status-dot {
	font-size: 0.5rem;
}

.chat-header-actions {
	display: flex;
	gap: 0.5rem;
}

.btn-header-action {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	padding: 0.5rem;
	border-radius: 6px;
	transition: all 0.2s ease;
	width: 32px;
	height: 32px;
	display: flex;
	align-items: center;
	justify-content: center;
}

.btn-header-action:hover {
	background: rgba(255, 255, 255, 0.1);
	color: #fff;
}

/* Chat Messages */
.chat-messages {
	flex: 1;
	overflow-y: auto;
	padding: 1rem;
	display: flex;
	flex-direction: column;
	gap: 1rem;
	background: rgba(0, 0, 0, 0.2);
	scroll-behavior: smooth;
}

.chat-message {
	display: flex;
	gap: 0.75rem;
	animation: slideIn 0.3s ease-out;
}

@keyframes slideIn {
	from {
		opacity: 0;
		transform: translateY(10px);
	}
	to {
		opacity: 1;
		transform: translateY(0);
	}
}

.chat-message.user {
	flex-direction: row-reverse;
}

.message-avatar {
	width: 36px;
	height: 36px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1rem;
	flex-shrink: 0;
}

.chat-message.user .message-avatar {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: #fff;
}

.chat-message.assistant .message-avatar {
	background: linear-gradient(135deg, #00d9ff 0%, #0099cc 100%);
	color: #fff;
}

.chat-message.thinking .message-avatar {
	animation: pulse 1.5s infinite;
}

@keyframes pulse {
	0%,
	100% {
		opacity: 1;
	}
	50% {
		opacity: 0.5;
	}
}

.message-content {
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
	max-width: 75%;
}

.chat-message.user .message-content {
	align-items: flex-end;
}

.message-text {
	background: rgba(255, 255, 255, 0.1);
	padding: 0.65rem 0.85rem;
	border-radius: 12px;
	color: #fff;
	line-height: 1.4;
	word-wrap: break-word;
	border: 1px solid rgba(255, 255, 255, 0.1);
	font-size: 0.9rem;
}

.chat-message.user .message-text {
	background: linear-gradient(135deg, rgba(102, 126, 234, 0.3) 0%, rgba(118, 75, 162, 0.3) 100%);
	border-color: rgba(102, 126, 234, 0.3);
}

.chat-message.assistant .message-text {
	background: linear-gradient(135deg, rgba(0, 217, 255, 0.2) 0%, rgba(0, 153, 204, 0.2) 100%);
	border-color: rgba(0, 217, 255, 0.3);
}

.message-time {
	font-size: 0.7rem;
	color: rgba(255, 255, 255, 0.4);
	padding: 0 0.5rem;
}

.typing-indicator {
	display: flex;
	gap: 0.25rem;
	padding: 0.5rem;
}

.typing-indicator span {
	width: 6px;
	height: 6px;
	border-radius: 50%;
	background: rgba(0, 217, 255, 0.6);
	animation: typing 1.4s infinite;
}

.typing-indicator span:nth-child(1) {
	animation-delay: 0s;
}

.typing-indicator span:nth-child(2) {
	animation-delay: 0.2s;
}

.typing-indicator span:nth-child(3) {
	animation-delay: 0.4s;
}

@keyframes typing {
	0%,
	60%,
	100% {
		transform: translateY(0);
		opacity: 0.6;
	}
	30% {
		transform: translateY(-8px);
		opacity: 1;
	}
}

/* Quick Actions Bar */
.quick-actions-bar {
	display: flex;
	gap: 0.5rem;
	padding: 0.75rem;
	background: rgba(0, 0, 0, 0.3);
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	overflow-x: auto;
	overflow-y: hidden;
	scrollbar-width: thin;
	scrollbar-color: rgba(0, 217, 255, 0.5) rgba(0, 0, 0, 0.2);
}

.quick-actions-bar::-webkit-scrollbar {
	height: 6px;
}

.quick-actions-bar::-webkit-scrollbar-track {
	background: rgba(0, 0, 0, 0.2);
	border-radius: 3px;
}

.quick-actions-bar::-webkit-scrollbar-thumb {
	background: rgba(0, 217, 255, 0.5);
	border-radius: 3px;
}

.quick-actions-bar::-webkit-scrollbar-thumb:hover {
	background: rgba(0, 217, 255, 0.7);
}

.btn-quick-action {
	flex: 0 0 auto;
	min-width: 2.5rem;
	background: rgba(0, 217, 255, 0.1);
	border: 1px solid rgba(0, 217, 255, 0.3);
	color: #00d9ff;
	padding: 0.5rem;
	border-radius: 6px;
	cursor: pointer;
	transition: all 0.2s ease;
}

.btn-quick-action:hover {
	background: rgba(0, 217, 255, 0.2);
	border-color: #00d9ff;
	transform: translateY(-2px);
}

.btn-quick-action.btn-danger {
	background: rgba(220, 53, 69, 0.1);
	border-color: rgba(220, 53, 69, 0.3);
	color: #dc3545;
}

.btn-quick-action.btn-danger:hover {
	background: rgba(220, 53, 69, 0.2);
	border-color: #dc3545;
}

/* Chat Input */
.chat-input-container {
	padding: 1rem;
	background: rgba(0, 0, 0, 0.3);
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.chat-input-container .input-group {
	background: rgba(0, 0, 0, 0.3);
	border-radius: 8px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	overflow: hidden;
	transition: all 0.3s ease;
}

.chat-input-container .input-group:focus-within {
	border-color: rgba(0, 217, 255, 0.5);
	box-shadow: 0 0 0 3px rgba(0, 217, 255, 0.1);
}

.chat-input-container .form-control {
	background: transparent;
	border: none;
	color: #fff;
	padding: 0.65rem 0.85rem;
	font-size: 0.9rem;
}

.chat-input-container .form-control:focus {
	background: transparent;
	color: #fff;
	box-shadow: none;
}

.chat-input-container .form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.chat-input-container .btn-primary {
	background: linear-gradient(135deg, #00d9ff 0%, #0099cc 100%);
	border: none;
	padding: 0.65rem 1rem;
	font-weight: 600;
	transition: all 0.3s ease;
}

.chat-input-container .btn-primary:hover:not(:disabled) {
	background: linear-gradient(135deg, #00ffff 0%, #00d9ff 100%);
	transform: scale(1.05);
}

.chat-input-container .btn-primary:disabled {
	background: rgba(108, 117, 125, 0.5);
	opacity: 0.6;
}

/* Voice Input Button */
.btn-voice-input {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(0, 217, 255, 0.3);
	padding: 0.65rem 1rem;
	margin-right: 0.5rem;
	font-weight: 600;
	color: #00d9ff;
	transition: all 0.3s ease;
	position: relative;
	overflow: visible;
}

.btn-voice-input:hover:not(:disabled) {
	background: rgba(0, 217, 255, 0.2);
	border-color: #00d9ff;
	transform: scale(1.05);
}

.btn-voice-input:disabled {
	opacity: 0.4;
	cursor: not-allowed;
}

.btn-voice-input.listening {
	background: linear-gradient(135deg, #ff4444 0%, #cc0000 100%);
	border-color: #ff4444;
	color: #fff;
	animation: pulse-glow 2s infinite;
}

.listening-pulse {
	position: absolute;
	top: 50%;
	left: 50%;
	transform: translate(-50%, -50%);
	width: 100%;
	height: 100%;
	border-radius: 0.375rem;
	background: rgba(255, 68, 68, 0.4);
	animation: pulse-ring 1.5s cubic-bezier(0.215, 0.61, 0.355, 1) infinite;
	pointer-events: none;
}

@keyframes pulse-glow {
	0%, 100% {
		box-shadow: 0 0 10px rgba(255, 68, 68, 0.5);
	}
	50% {
		box-shadow: 0 0 20px rgba(255, 68, 68, 0.8);
	}
}

@keyframes pulse-ring {
	0% {
		transform: translate(-50%, -50%) scale(0.8);
		opacity: 1;
	}
	100% {
		transform: translate(-50%, -50%) scale(1.5);
		opacity: 0;
	}
}

/* Search Bar */
.search-bar {
	background: rgba(0, 0, 0, 0.2);
	padding: 1rem;
	border-bottom: 1px solid rgba(0, 217, 255, 0.2);
}

.search-input-group {
	display: flex;
	align-items: center;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(0, 217, 255, 0.3);
	border-radius: 0.5rem;
	padding: 0.5rem 1rem;
	gap: 0.75rem;
}

.search-icon {
	color: #00d9ff;
	font-size: 1rem;
}

.search-input {
	flex: 1;
	background: transparent;
	border: none;
	color: #fff;
	font-size: 0.95rem;
	outline: none;
}

.search-input::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.btn-clear-search {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	padding: 0.25rem;
	transition: color 0.2s ease;
}

.btn-clear-search:hover {
	color: #ff4444;
}

.search-results-info {
	margin-top: 0.5rem;
	font-size: 0.85rem;
	color: #00d9ff;
	text-align: center;
}

.slide-down-enter-active, .slide-down-leave-active {
	transition: all 0.3s ease;
}

.slide-down-enter-from, .slide-down-leave-to {
	max-height: 0;
	opacity: 0;
	overflow: hidden;
}

.slide-down-enter-to, .slide-down-leave-from {
	max-height: 200px;
	opacity: 1;
}

/* Statistics Panel */
.stats-panel {
	background: linear-gradient(135deg, rgba(0, 217, 255, 0.1) 0%, rgba(0, 153, 204, 0.1) 100%);
	border-bottom: 1px solid rgba(0, 217, 255, 0.2);
	padding: 1rem;
}

.stats-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1rem;
}

.stats-header h6 {
	margin: 0;
	color: #00d9ff;
	font-size: 0.95rem;
	font-weight: 600;
}

.btn-close-stats {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	padding: 0.25rem;
	transition: color 0.2s ease;
}

.btn-close-stats:hover {
	color: #ff4444;
}

.stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
	gap: 0.75rem;
}

.stat-item {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	background: rgba(0, 0, 0, 0.3);
	padding: 0.75rem;
	border-radius: 0.5rem;
	border: 1px solid rgba(0, 217, 255, 0.2);
	transition: all 0.3s ease;
}

.stat-item:hover {
	background: rgba(0, 0, 0, 0.4);
	border-color: #00d9ff;
	transform: translateY(-2px);
}

.stat-icon {
	width: 2.5rem;
	height: 2.5rem;
	display: flex;
	align-items: center;
	justify-content: center;
	background: linear-gradient(135deg, #00d9ff 0%, #0099cc 100%);
	border-radius: 0.5rem;
	color: #fff;
	font-size: 1.1rem;
	flex-shrink: 0;
}

.stat-icon.user {
	background: linear-gradient(135deg, #00ff88 0%, #00cc66 100%);
}

.stat-icon.ai {
	background: linear-gradient(135deg, #ff00ff 0%, #cc00cc 100%);
}

.stat-details {
	flex: 1;
	min-width: 0;
}

.stat-value {
	font-size: 1.25rem;
	font-weight: 700;
	color: #fff;
	line-height: 1.2;
}

.stat-label {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.6);
	text-transform: uppercase;
	letter-spacing: 0.5px;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.btn-quick-action.active {
	background: linear-gradient(135deg, #00d9ff 0%, #0099cc 100%);
	border-color: #00d9ff;
	color: #fff;
}

/* Message Footer with Copy Button */
.message-footer {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-top: 0.25rem;
}

.btn-copy-message {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.4);
	cursor: pointer;
	padding: 0.25rem 0.5rem;
	font-size: 0.85rem;
	transition: all 0.2s ease;
	opacity: 0;
}

.chat-message:hover .btn-copy-message {
	opacity: 1;
}

.btn-copy-message:hover {
	color: #00d9ff;
	transform: scale(1.1);
}

/* Suggested Prompts */
.suggested-prompts {
	padding: 1.5rem 1rem;
	animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
	from {
		opacity: 0;
		transform: translateY(10px);
	}
	to {
		opacity: 1;
		transform: translateY(0);
	}
}

.suggested-header h5 {
	margin: 0 0 1rem 0;
	color: #00d9ff;
	font-size: 1rem;
	font-weight: 600;
	text-align: center;
}

.prompt-grid {
	display: grid;
	grid-template-columns: repeat(2, 1fr);
	gap: 0.75rem;
}

.prompt-card {
	background: rgba(0, 217, 255, 0.1);
	border: 1px solid rgba(0, 217, 255, 0.3);
	border-radius: 10px;
	padding: 0.85rem;
	cursor: pointer;
	transition: all 0.3s ease;
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 0.5rem;
	text-align: center;
}

.prompt-card:hover {
	background: rgba(0, 217, 255, 0.2);
	border-color: #00d9ff;
	transform: translateY(-3px);
	box-shadow: 0 4px 12px rgba(0, 217, 255, 0.3);
}

.prompt-icon {
	width: 36px;
	height: 36px;
	border-radius: 50%;
	background: linear-gradient(135deg, #00d9ff 0%, #0099cc 100%);
	display: flex;
	align-items: center;
	justify-content: center;
	color: #fff;
	font-size: 1rem;
}

.prompt-text {
	font-size: 0.8rem;
	color: rgba(255, 255, 255, 0.9);
	font-weight: 500;
	line-height: 1.3;
}

/* Scrollbar */
.chat-messages::-webkit-scrollbar {
	width: 6px;
}

.chat-messages::-webkit-scrollbar-track {
	background: rgba(0, 0, 0, 0.2);
	border-radius: 10px;
}

.chat-messages::-webkit-scrollbar-thumb {
	background: rgba(0, 217, 255, 0.3);
	border-radius: 10px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
	background: rgba(0, 217, 255, 0.5);
}

/* Responsive */
@media (max-width: 768px) {
	.chat-window {
		width: calc(100vw - 40px);
		height: calc(100vh - 100px);
		max-height: 600px;
	}
}
</style>
