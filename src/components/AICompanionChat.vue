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
						<button class="btn-header-action" @click="minimizeChat" title="Minimize">
							<font-awesome-icon :icon="['fas', 'minus']" />
						</button>
						<button class="btn-header-action" @click="closeChat" title="Close">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
				</div>

				<!-- Chat Messages -->
				<div class="chat-messages" ref="chatMessages">
					<div v-for="(message, index) in chatHistory" :key="index" class="chat-message" :class="message.role">
						<div class="message-avatar">
							<font-awesome-icon :icon="['fas', message.role === 'user' ? 'user' : 'robot']" />
						</div>
						<div class="message-content">
							<div class="message-text">{{ message.content }}</div>
							<div class="message-time">{{ formatMessageTime(message.timestamp) }}</div>
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

				<!-- Quick Actions -->
				<div class="quick-actions-bar">
					<button class="btn-quick-action" @click="askQuickQuestion('How many videos do I have?')" title="Library Stats">
						<font-awesome-icon :icon="['fas', 'chart-bar']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('What are my top performers?')" title="Top Performers">
						<font-awesome-icon :icon="['fas', 'star']" />
					</button>
					<button class="btn-quick-action" @click="askQuickQuestion('Show me videos with issues')" title="Issues">
						<font-awesome-icon :icon="['fas', 'exclamation-triangle']" />
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
import { ref, nextTick, onMounted, getCurrentInstance } from 'vue'
import { aiCompanionAPI } from '@/services/api'
import { toolDefinitions, executeTool } from '@/services/aiTools'

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
			content:
				"Hello! I'm your AI Companion - the intelligent core of Video Storage AI. I can query your database, analyze your library, find performers, and monitor for issues. I'm here to help manage and optimize your video collection. What would you like to know?",
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
}

.btn-quick-action {
	flex: 1;
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
