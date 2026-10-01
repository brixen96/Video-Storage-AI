<template>
	<div class="ai-page">
		<div class="container-fluid mt-3">
			<!-- Page Header -->
			<div class="page-header text-center mb-4">
				<h1>
					<font-awesome-icon :icon="['fas', 'robot']" class="me-3" />
					AI Assistant
				</h1>
				<p class="lead">Intelligent tools to organize, analyze, and manage your video library</p>
			</div>

			<!-- Tab Navigation -->
			<ul class="nav nav-tabs ai-tabs mb-4">
				<li class="nav-item">
					<button class="nav-link" :class="{ active: activeTab === 'companion' }" @click="activeTab = 'companion'">
						<font-awesome-icon :icon="['fas', 'robot']" class="me-2" />
						Companion
						<span class="status-dot ms-2" :class="companionOnline ? 'bg-success' : 'bg-danger'"></span>
					</button>
				</li>
				<li class="nav-item">
					<button class="nav-link" :class="{ active: activeTab === 'chat' }" @click="activeTab = 'chat'">
						<font-awesome-icon :icon="['fas', 'comments']" class="me-2" />
						Chat
					</button>
				</li>
				<li class="nav-item">
					<button class="nav-link" :class="{ active: activeTab === 'tasks' }" @click="activeTab = 'tasks'">
						<font-awesome-icon :icon="['fas', 'clipboard-list']" class="me-2" />
						Tasks
					</button>
				</li>
				<li class="nav-item">
					<button class="nav-link" :class="{ active: activeTab === 'tools' }" @click="activeTab = 'tools'">
						<font-awesome-icon :icon="['fas', 'tools']" class="me-2" />
						Analysis Tools
					</button>
				</li>			<li class="nav-item">
				<button class="nav-link" :class="{ active: activeTab === 'monitor' }" @click="activeTab = 'monitor'">
					<font-awesome-icon :icon="['fas', 'chart-line']" class="me-2" />
					Monitor
				</button>
			</li>			</ul>

			<!-- Companion Tab -->
			<div v-if="activeTab === 'companion'" class="companion-tab">
				<!-- Status & Config Row -->
				<div class="row g-4 mb-4">
					<!-- AI Connection Status -->
					<div class="col-md-4">
						<div class="status-card">
							<div class="status-card-header">
								<font-awesome-icon :icon="['fas', 'server']" class="me-2" />
								OpenRouter Connection
							</div>
							<div class="status-card-body">
								<div class="status-indicator" :class="companionOnline ? 'online' : 'offline'">
									<font-awesome-icon :icon="['fas', companionOnline ? 'check-circle' : 'times-circle']" />
									{{ companionOnline ? 'Connected' : 'Disconnected' }}
								</div>
								<div v-if="aiConfig" class="config-details mt-3">
									<div class="config-item">
										<span class="config-label">URL</span>
										<span class="config-value">{{ aiConfig.api_url || 'Not configured' }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Model</span>
										<span class="config-value">{{ aiConfig.model_name || 'Auto-detect' }}</span>
									</div>
								</div>
								<button class="btn btn-sm btn-outline-info mt-3 w-100" @click="testConnection" :disabled="testingConnection">
									<font-awesome-icon :icon="['fas', testingConnection ? 'spinner' : 'plug']" :spin="testingConnection" class="me-1" />
									{{ testingConnection ? 'Testing...' : 'Test Connection' }}
								</button>
								<div v-if="connectionTestResult" class="mt-2 small" :class="connectionTestResult.success ? 'text-success' : 'text-danger'">
									{{ connectionTestResult.message }}
								</div>
							</div>
						</div>
					</div>

					<!-- AI Config -->
					<div class="col-md-4">
						<div class="status-card">
							<div class="status-card-header">
								<font-awesome-icon :icon="['fas', 'sliders-h']" class="me-2" />
								Configuration
							</div>
							<div class="status-card-body">
								<div v-if="aiConfig" class="config-details">
									<div class="config-item">
										<span class="config-label">Temperature</span>
										<span class="config-value">{{ aiConfig.temperature }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Max Tokens</span>
										<span class="config-value">{{ aiConfig.max_tokens }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Enabled</span>
										<span class="config-value">
											<font-awesome-icon :icon="['fas', aiConfig.enabled ? 'check' : 'times']" :class="aiConfig.enabled ? 'text-success' : 'text-danger'" />
										</span>
									</div>
								</div>
								<div v-else class="text-muted">Loading config...</div>
								<router-link to="/settings" class="btn btn-sm btn-outline-secondary mt-3 w-100">
									<font-awesome-icon :icon="['fas', 'cog']" class="me-1" />
									Edit in Settings
								</router-link>
							</div>
						</div>
					</div>

					<!-- Companion Stats -->
					<div class="col-md-4">
						<div class="status-card">
							<div class="status-card-header">
								<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />
								Companion Stats
							</div>
							<div class="status-card-body">
								<div v-if="companionStatus" class="config-details">
									<div class="config-item">
										<span class="config-label">Uptime</span>
										<span class="config-value">{{ companionStatus.uptime || 'N/A' }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Messages</span>
										<span class="config-value">{{ companionStatus.message_count || 0 }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Tools Available</span>
										<span class="config-value">{{ companionStatus.tools_count || 15 }}</span>
									</div>
								</div>
								<div v-else class="config-details">
									<div class="config-item">
										<span class="config-label">Status</span>
										<span class="config-value text-muted">Loading...</span>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Chat Tab -->
			<div v-if="activeTab === 'chat'" class="chat-tab">
				<!-- Chat Section -->
				<div class="chat-section">
					<div class="chat-section-header">
						<h3>
							<font-awesome-icon :icon="['fas', 'comments']" class="me-2" />
							Chat with AI Companion
						</h3>
						<div class="d-flex gap-2">
							<button class="btn btn-sm btn-outline-danger" @click="clearChat" :disabled="chatHistory.length === 0">
								<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
								Clear
							</button>
						</div>
					</div>
					<div class="chat-container">
						<div ref="chatMessagesRef" class="chat-messages">
							<div v-if="chatHistory.length === 0" class="empty-chat">
								<font-awesome-icon :icon="['fas', 'robot']" size="3x" class="mb-3" />
								<p>Start a conversation with the AI Companion</p>
								<p class="text-muted small">Ask about your library, get suggestions, or manage tasks</p>
							</div>
							<div v-for="(msg, idx) in chatHistory" :key="idx" class="chat-message" :class="msg.role">
								<div class="message-avatar">
									<font-awesome-icon :icon="['fas', msg.role === 'user' ? 'user' : 'robot']" />
								</div>
								<div class="message-content">
									<template v-if="msg.role === 'user'">{{ msg.content }}</template>
									<MarkdownMessage v-else :content="msg.content" />
									<div class="message-time">{{ formatTime() }}</div>
								</div>
							</div>
							<div v-if="isStreaming" class="chat-message assistant">
								<div class="message-avatar">
									<font-awesome-icon :icon="['fas', 'robot']" />
								</div>
								<div class="message-content">
									<div class="message-text">
										<div class="typing-indicator">
											<span></span><span></span><span></span>
										</div>
									</div>
								</div>
							</div>
						</div>

						<!-- Quick Actions -->
						<div class="quick-actions mb-2">
							<button class="btn btn-sm btn-outline-info" @click="sendQuickMessage('What is the current status of my library?')">Library Status</button>
							<button class="btn btn-sm btn-outline-info" @click="sendQuickMessage('Show me recent activity')">Recent Activity</button>
							<button class="btn btn-sm btn-outline-info" @click="sendQuickMessage('Any issues I should know about?')">Check Issues</button>
							<button class="btn btn-sm btn-outline-info" @click="sendQuickMessage('Suggest some improvements')">Suggestions</button>
						</div>

						<!-- Chat Input -->
						<div class="chat-input-container">
							<div class="input-group">
								<input
									v-model="chatInput"
									type="text"
									class="form-control"
									placeholder="Type a message..."
									@keyup.enter="sendMessage"
									:disabled="isStreaming"
								/>
								<button v-if="isStreaming" class="btn btn-danger" @click="stopStreaming" title="Stop generating">
									<font-awesome-icon :icon="['fas', 'stop']" />
								</button>
								<button v-else class="btn btn-primary" @click="sendMessage" :disabled="!chatInput.trim() || isStreaming">
									<font-awesome-icon :icon="['fas', 'paper-plane']" />
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Tasks Tab -->
			<div v-if="activeTab === 'tasks'" class="tasks-tab">
				<TaskBoard />
			</div>

			<!-- Analysis Tools Tab -->
			<div v-if="activeTab === 'tools'">
				<!-- Dashboard Summary Card -->
				<div class="dashboard-summary mb-5">
					<div class="row g-4">
						<div class="col-md-3">
							<div class="summary-card">
								<div class="summary-icon">
									<font-awesome-icon :icon="['fas', 'heart']" :class="getHealthIconClass()" />
								</div>
								<div class="summary-content">
									<div class="summary-value" :class="getHealthScoreClass()">{{ libraryHealthScore }}%</div>
									<div class="summary-label">Library Health</div>
								</div>
							</div>
						</div>
						<div class="col-md-3">
							<div class="summary-card">
								<div class="summary-icon bg-primary">
									<font-awesome-icon :icon="['fas', 'video']" />
								</div>
								<div class="summary-content">
									<div class="summary-value">{{ totalVideosProcessed }}</div>
									<div class="summary-label">Videos Processed</div>
								</div>
							</div>
						</div>
						<div class="col-md-3">
							<div class="summary-card">
								<div class="summary-icon bg-success">
									<font-awesome-icon :icon="['fas', 'check-circle']" />
								</div>
								<div class="summary-content">
									<div class="summary-value">{{ totalMatches }}</div>
									<div class="summary-label">AI Matches Found</div>
								</div>
							</div>
						</div>
						<div class="col-md-3">
							<div class="summary-card">
								<div class="summary-icon bg-warning">
									<font-awesome-icon :icon="['fas', 'exclamation-triangle']" />
								</div>
								<div class="summary-content">
									<div class="summary-value">{{ totalIssues }}</div>
									<div class="summary-label">Issues Detected</div>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- AI Features Grid -->
				<div class="row g-4">
					<div class="col-md-6">
						<AILinkPerformers ref="linkPerformersRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AISuggestTags ref="suggestTagsRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AIDetectScenes ref="detectScenesRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AIClassifyContent ref="classifyContentRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AIAnalyzeQuality ref="analyzeQualityRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AIDetectMissingMetadata ref="detectMetadataRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AIDetectDuplicates ref="detectDuplicatesRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AISuggestNaming ref="suggestNamingRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AILibraryAnalytics ref="libraryAnalyticsRef" @statsUpdated="updateDashboard" />
					</div>
					<div class="col-md-6">
						<AIAnalyzeThumbnails ref="analyzeThumbnailsRef" @statsUpdated="updateDashboard" />
					</div>
				</div>
			</div>

			<!-- Monitor Tab -->
			<div v-if="activeTab === 'monitor'">
				<AIMonitorDashboard />
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { aiAPI } from '@/services/api'

// Import all AI feature components
import AILinkPerformers from '@/components/ai/AILinkPerformers.vue'
import AISuggestTags from '@/components/ai/AISuggestTags.vue'
import AIDetectScenes from '@/components/ai/AIDetectScenes.vue'
import AIClassifyContent from '@/components/ai/AIClassifyContent.vue'
import AIAnalyzeQuality from '@/components/ai/AIAnalyzeQuality.vue'
import AIDetectMissingMetadata from '@/components/ai/AIDetectMissingMetadata.vue'
import AIDetectDuplicates from '@/components/ai/AIDetectDuplicates.vue'
import AISuggestNaming from '@/components/ai/AISuggestNaming.vue'
import AILibraryAnalytics from '@/components/ai/AILibraryAnalytics.vue'
import AIAnalyzeThumbnails from '@/components/ai/AIAnalyzeThumbnails.vue'
import AIMonitorDashboard from '@/components/ai/AIMonitorDashboard.vue'
import MarkdownMessage from '@/components/MarkdownMessage.vue'
import TaskBoard from '@/components/TaskBoard.vue'

// Shared composables — same state as the floating widget
import { useChatHistory } from '@/composables/ai/useChatHistory'
import { useChatStreaming } from '@/composables/ai/useChatStreaming'

const { chatHistory, pushMessage, clearHistory, loadHistory } = useChatHistory()
const { isStreaming, streamMessage, stopStreaming } = useChatStreaming()

// Tab state
const activeTab = ref('companion')

// Companion state
const companionOnline = ref(false)
const companionStatus = ref(null)
const aiConfig = ref(null)
const testingConnection = ref(false)
const connectionTestResult = ref(null)

// Chat state (local UI helpers only — messages live in chatHistory)
const chatInput = ref('')
const chatMessagesRef = ref(null)

// Analysis tools refs
const linkPerformersRef = ref(null)
const suggestTagsRef = ref(null)
const detectScenesRef = ref(null)
const classifyContentRef = ref(null)
const analyzeQualityRef = ref(null)
const detectMetadataRef = ref(null)
const detectDuplicatesRef = ref(null)
const suggestNamingRef = ref(null)
const libraryAnalyticsRef = ref(null)
const analyzeThumbnailsRef = ref(null)

// Dashboard computed properties
const totalVideosProcessed = computed(() => {
	let total = 0
	if (linkPerformersRef.value?.linkStats) total += linkPerformersRef.value.linkStats.videosAnalyzed || 0
	if (suggestTagsRef.value?.tagStats) total += suggestTagsRef.value.tagStats.videosAnalyzed || 0
	if (detectScenesRef.value?.sceneStats) total += detectScenesRef.value.sceneStats.videosAnalyzed || 0
	if (analyzeQualityRef.value?.qualityStats) total += analyzeQualityRef.value.qualityStats.videosAnalyzed || 0
	return total
})

const totalMatches = computed(() => {
	let total = 0
	if (linkPerformersRef.value?.linkStats) total += linkPerformersRef.value.linkStats.matchesFound || 0
	if (suggestTagsRef.value?.tagStats) total += suggestTagsRef.value.tagStats.tagsSuggested || 0
	if (detectScenesRef.value?.sceneStats) total += detectScenesRef.value.sceneStats.scenesFound || 0
	return total
})

const totalIssues = computed(() => {
	let total = 0
	if (analyzeQualityRef.value?.qualityStats) total += analyzeQualityRef.value.qualityStats.issuesFound || 0
	if (detectMetadataRef.value?.metadataStats) total += detectMetadataRef.value.metadataStats.issuesFound || 0
	if (detectDuplicatesRef.value?.duplicateStats) total += detectDuplicatesRef.value.duplicateStats.duplicateGroups || 0
	return total
})

const libraryHealthScore = computed(() => {
	const videosWithIssues = totalIssues.value
	const videosProcessed = totalVideosProcessed.value || 1
	const healthPercentage = Math.max(0, Math.min(100, 100 - (videosWithIssues / videosProcessed) * 100))
	return Math.round(healthPercentage)
})

const getHealthScoreClass = () => {
	const score = libraryHealthScore.value
	if (score >= 80) return 'text-success'
	if (score >= 60) return 'text-warning'
	return 'text-danger'
}

const getHealthIconClass = () => {
	const score = libraryHealthScore.value
	if (score >= 80) return 'text-success pulse-animation'
	if (score >= 60) return 'text-warning'
	return 'text-danger'
}

const updateDashboard = () => {
	// Computed properties auto-recalculate
}

// Companion methods
const loadCompanionData = async () => {
	try {
		const [statusRes, configRes] = await Promise.allSettled([
			aiAPI.getStatus(),
			aiAPI.getConfig(),
		])

		if (statusRes.status === 'fulfilled' && statusRes.value?.data) {
			companionStatus.value = statusRes.value.data
			companionOnline.value = true
		}

		if (configRes.status === 'fulfilled') {
			aiConfig.value = configRes.value?.data || configRes.value
		}
	} catch {
		companionOnline.value = false
	}
}

const testConnection = async () => {
	testingConnection.value = true
	connectionTestResult.value = null
	try {
		const res = await aiAPI.testConnection()
		connectionTestResult.value = { success: true, message: res?.message || 'Connection successful!' }
		companionOnline.value = true
	} catch {
						connectionTestResult.value = { success: false, message: 'Connection failed. Check your OpenRouter API key and network connection.' }
		companionOnline.value = false
	} finally {
		testingConnection.value = false
	}
}

// Chat methods
const formatTime = () => {
	return new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const scrollToBottom = async () => {
	await nextTick()
	if (chatMessagesRef.value) {
		chatMessagesRef.value.scrollTop = chatMessagesRef.value.scrollHeight
	}
}

const sendMessage = async () => {
	const text = chatInput.value.trim()
	if (!text || isStreaming.value) return

	chatInput.value = ''

	// Push user message
	pushMessage({ role: 'user', content: text, timestamp: new Date() })
	await nextTick()
	scrollToBottom()

	// Prepare history for context
	const historyPayload = chatHistory.value
		.filter((m) => m.role === 'user' || m.role === 'assistant')
		.slice(-10)
		.map((m) => ({ role: m.role, content: m.content }))

	// Create placeholder for streaming response
	const msgIndex = chatHistory.value.length
	pushMessage({
		role: 'assistant', content: '', timestamp: new Date(),
		source: 'llm', streaming: true,
	})
	await nextTick()
	scrollToBottom()

	// Stream the response
	streamMessage(text, historyPayload, {
		onTextChunk: (chunk) => {
			chatHistory.value[msgIndex].content += chunk
		},
		scrollFn: async () => { await nextTick(); scrollToBottom() },
		onDone: (toolSteps) => {
			chatHistory.value[msgIndex].streaming = false
			if (toolSteps.length > 0) chatHistory.value[msgIndex].toolSteps = toolSteps
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
					role: 'assistant', content: 'Sorry, I encountered an error. Please check your OpenRouter connection.',
					timestamp: new Date(), source: 'error',
				})
			}
		},
	})
}

const sendQuickMessage = (text) => {
	chatInput.value = text
	sendMessage()
}

const clearChat = async () => {
	await clearHistory()
}

// Reload chat history when switching to chat tab
import { watch } from 'vue'
watch(activeTab, (newTab) => {
	if (newTab === 'chat') {
		loadHistory()
	}
})

onMounted(async () => {
	loadCompanionData()
	await loadHistory()
})
</script>

<style scoped>
@import '@/styles/pages/ai_page.css';

/* Tab Navigation */
.ai-tabs {
	border-bottom: 2px solid rgba(255, 255, 255, 0.1);
}

.ai-tabs .nav-link {
	color: rgba(255, 255, 255, 0.6);
	border: none;
	padding: 0.75rem 1.5rem;
	font-weight: 600;
	font-size: 1.05rem;
	transition: all 0.3s ease;
	border-bottom: 3px solid transparent;
	background: transparent;
}

.ai-tabs .nav-link:hover {
	color: rgba(255, 255, 255, 0.9);
	border-bottom-color: rgba(0, 217, 255, 0.3);
}

.ai-tabs .nav-link.active {
	color: #00d9ff;
	border-bottom-color: #00d9ff;
	background: transparent;
}

.status-dot {
	display: inline-block;
	width: 8px;
	height: 8px;
	border-radius: 50%;
}

/* Status Cards */
.status-card {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	overflow: hidden;
	height: 100%;
}

.status-card-header {
	background: rgba(0, 0, 0, 0.3);
	padding: 1rem 1.25rem;
	font-weight: 600;
	color: #00d9ff;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.status-card-body {
	padding: 1.25rem;
}

.status-indicator {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	font-size: 1.1rem;
	font-weight: 600;
	padding: 0.75rem 1rem;
	border-radius: 8px;
}

.status-indicator.online {
	background: rgba(40, 167, 69, 0.15);
	color: #28a745;
	border: 1px solid rgba(40, 167, 69, 0.3);
}

.status-indicator.offline {
	background: rgba(220, 53, 69, 0.15);
	color: #dc3545;
	border: 1px solid rgba(220, 53, 69, 0.3);
}

.config-details {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.config-item {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.4rem 0;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.config-label {
	color: rgba(255, 255, 255, 0.6);
	font-size: 0.9rem;
}

.config-value {
	color: #fff;
	font-weight: 500;
	font-size: 0.9rem;
	text-align: right;
	max-width: 60%;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

/* Chat Section */
.chat-section {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	overflow: hidden;
}

.chat-section-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1rem 1.5rem;
	background: rgba(0, 0, 0, 0.3);
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.chat-section-header h3 {
	margin: 0;
	font-size: 1.2rem;
	color: #00d9ff;
}

.chat-section .chat-container {
	padding: 1.25rem;
}

.chat-section .chat-messages {
	height: 450px;
	overflow-y: auto;
	padding: 1rem;
	display: flex;
	flex-direction: column;
	gap: 1rem;
	background: rgba(0, 0, 0, 0.2);
	border-radius: 8px;
	margin-bottom: 1rem;
	scroll-behavior: smooth;
}

.empty-chat {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	height: 100%;
	color: rgba(255, 255, 255, 0.4);
	text-align: center;
}

.empty-chat p {
	margin: 0.25rem 0;
}
</style>
