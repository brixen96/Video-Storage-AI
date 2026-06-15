<template>
	<div class="ai-companion-float">
		<!-- Floating Chat Button -->
		<button v-if="!isOpen" class="chat-toggle-button" @click="toggleChat" :class="{ 'has-notification': unreadCount > 0 }" title="Open AI Companion (Ctrl+Shift+A)">
			<font-awesome-icon :icon="['fas', 'robot']" class="chat-icon" />
			<span v-if="unreadCount > 0" class="notification-badge">{{ unreadCount > 9 ? '9+' : unreadCount }}</span>
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
						<button class="btn-header-action" :class="{ active: showMemoryPanel }" @click="toggleMemoryPanel" title="AI Memory">
							<font-awesome-icon :icon="['fas', 'brain']" />
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

				<!-- Context Stats Strip -->
				<div v-if="!contextStatsDismissed" class="context-stats-strip">
					<template v-if="contextStatsLoading">
						<div class="stat-skeleton" v-for="i in 3" :key="i"></div>
					</template>
					<template v-else-if="contextStats">
						<button class="stat-pill" @click="askQuickQuestion('Library overview')">
							<font-awesome-icon :icon="['fas', 'film']" />
							{{ contextStats.videos.toLocaleString() }} videos
						</button>
						<button
							class="stat-pill"
							:class="{ 'stat-pill--active': contextStats.running > 0 }"
							@click="askQuickQuestion('What tasks are currently running?')"
						>
							<font-awesome-icon :icon="['fas', 'circle-notch']" :spin="contextStats.running > 0" />
							{{ contextStats.running > 0 ? `${contextStats.running} running` : 'Idle' }}
						</button>
						<button
							v-if="contextStats.failed > 0"
							class="stat-pill stat-pill--danger"
							@click="askQuickQuestion('Check for errors')"
						>
							<font-awesome-icon :icon="['fas', 'exclamation-circle']" />
							{{ contextStats.failed }} failed
						</button>
						<button class="stat-strip-icon" @click="loadContextStats" title="Refresh stats">
							<font-awesome-icon :icon="['fas', 'sync-alt']" />
						</button>
					</template>
					<button class="stat-strip-icon stat-strip-dismiss" @click="contextStatsDismissed = true" title="Dismiss">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>

				<!-- Pinned Messages Panel -->
				<div v-if="pinnedMessages.length > 0" class="pinned-panel">
					<button class="pinned-panel-header" @click="pinnedExpanded = !pinnedExpanded">
						<font-awesome-icon :icon="['fas', 'thumbtack']" />
						{{ pinnedMessages.length }} pinned
						<font-awesome-icon :icon="['fas', pinnedExpanded ? 'chevron-up' : 'chevron-down']" class="pinned-chevron" />
					</button>
					<transition name="slide-down">
						<div v-if="pinnedExpanded" class="pinned-list">
							<div
								v-for="msg in pinnedMessages"
								:key="msg.timestamp?.getTime()"
								class="pinned-item"
								@click="scrollToMessage(msg)"
								title="Click to jump to message"
							>
								<div class="pinned-text">{{ msg.content.slice(0, 100) }}{{ msg.content.length > 100 ? '…' : '' }}</div>
								<button class="btn-unpin" @click.stop="togglePinMessage(msg)" title="Unpin">
									<font-awesome-icon :icon="['fas', 'times']" />
								</button>
							</div>
						</div>
					</transition>
				</div>

				<!-- Memory Panel -->
				<div v-if="showMemoryPanel" class="memory-panel">
					<div class="memory-panel-top">
						<div class="memory-panel-title">
							<font-awesome-icon :icon="['fas', 'brain']" />
							<span>AI Memory</span>
							<span class="memory-total-count">{{ memories.length }}</span>
						</div>
						<div class="memory-search-wrap">
							<font-awesome-icon :icon="['fas', 'search']" class="memory-search-icon" />
							<input v-model="memorySearch" class="memory-search-input" placeholder="Search memories…" />
						</div>
					</div>
					<div class="memory-category-tabs">
						<button :class="['mem-tab', { active: memoryCategory === '' }]" @click="memoryCategory = ''">All</button>
						<button
							v-for="cat in memoryCounts"
							:key="cat.name"
							:class="['mem-tab', `mem-tab--${cat.name}`, { active: memoryCategory === cat.name }]"
							@click="memoryCategory = cat.name"
						>{{ cat.name }} <span class="mem-tab-count">{{ cat.count }}</span></button>
					</div>
					<div class="memory-list">
						<div v-if="memoriesLoading" class="memory-empty">
							<font-awesome-icon :icon="['fas', 'spinner']" spin /> Loading…
						</div>
						<div v-else-if="filteredMemories.length === 0" class="memory-empty">
							<font-awesome-icon :icon="['fas', 'brain']" class="memory-empty-icon" />
							<div v-if="memories.length === 0">No memories yet. I save useful insights when you 👍 a response.</div>
							<div v-else>No memories match your filter.</div>
						</div>
						<div v-for="m in filteredMemories" :key="m.id" class="memory-item">
							<div class="memory-item-header">
								<span class="memory-key">{{ m.key.replace(/_/g, ' ') }}</span>
								<span :class="['memory-cat-badge', `cat-${m.category}`]">{{ m.category }}</span>
								<button class="btn-forget-memory" @click="deleteMemory(m.id)" title="Forget this">
									<font-awesome-icon :icon="['fas', 'times']" />
								</button>
							</div>
							<div class="memory-value">{{ m.value }}</div>
							<div class="memory-meta">
								<span class="memory-date">{{ formatMemoryDate(m.created_at) }}</span>
								<span v-if="m.importance > 0" class="memory-importance">{{ '★'.repeat(Math.min(m.importance, 5)) }}</span>
								<button class="btn-recall-memory" @click="askQuickQuestion(`Tell me about: ${m.key.replace(/_/g, ' ')}`); showMemoryPanel = false" title="Ask about this">
									<font-awesome-icon :icon="['fas', 'comment']" />
								</button>
							</div>
						</div>
					</div>
				</div>

				<!-- Chat Messages -->
				<div v-if="!showMemoryPanel" class="chat-messages" ref="chatMessages">
					<!-- Suggested Prompts (shown when empty) -->
					<div v-if="chatHistory.length === 0 && !isCompanionThinking" class="suggested-prompts">
						<!-- Entity context card -->
						<div v-if="entityCard" class="entity-context-card">
							<div class="entity-card-header">
								<font-awesome-icon :icon="entityCard.icon" class="entity-card-type-icon" />
								<span class="entity-card-name">{{ entityCard.name }}</span>
							</div>
							<div v-if="entityCard.fields.length > 0" class="entity-card-stats">
								<span v-for="field in entityCard.fields" :key="field.value" class="entity-stat">
									<font-awesome-icon :icon="field.icon" class="entity-stat-icon" />
									{{ field.value }}
								</span>
							</div>
						</div>
						<div class="suggested-header">
							<span class="suggested-page-label" v-if="suggestedPageLabel">
								<font-awesome-icon :icon="['fas', 'map-marker-alt']" class="me-1" />{{ suggestedPageLabel }}
							</span>
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
							<!-- Compare quality button — only when 2+ library videos present -->
							<div v-if="message.videos.filter(v => v.id > 0).length >= 2" class="compare-quality-row">
								<button class="btn-compare-quality" @click="compareCardsInAnalyzer(message.videos)" title="Open in Video Quality Analyzer">
									<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-1" />
									Compare Quality
								</button>
							</div>
						</div>

						<!-- Regular text messages -->
						<div v-else class="chat-message" :class="[message.role, message.severity, { 'daily-briefing': message.source === 'daily_briefing', 'library-insights': message.source === 'library_insights', 'entity-greeting': message.source === 'entity_greeting' }]" :data-msg-ts="message.timestamp?.getTime()">
							<div class="message-avatar" :class="{ 'proactive-avatar': message.role === 'proactive' }">
								<font-awesome-icon :icon="['fas', message.role === 'user' ? 'user' : message.role === 'proactive' ? (message.source === 'daily_briefing' ? 'sun' : message.source === 'library_insights' ? 'heartbeat' : message.source === 'analysis_engine' ? 'lightbulb' : message.source === 'entity_greeting' ? 'map-marker-alt' : 'bell') : 'robot']" />
							</div>
							<div class="message-content">
								<div class="message-text" :class="{ 'streaming-text': message.streaming }">
									<!-- User message — plain text or inline edit -->
									<template v-if="message.role === 'user'">
										<template v-if="isEditing(message)">
											<textarea class="edit-message-input" v-model="editingContent" @keydown.ctrl.enter.prevent="confirmEditMessage" @keydown.esc.prevent="cancelEditMessage" rows="3" autofocus></textarea>
											<div class="edit-message-actions">
												<button class="btn-edit-confirm" @click="confirmEditMessage" title="Ctrl+Enter">
													<font-awesome-icon :icon="['fas', 'paper-plane']" /> Send
												</button>
												<button class="btn-edit-cancel" @click="cancelEditMessage" title="Esc">
													<font-awesome-icon :icon="['fas', 'times']" /> Cancel
												</button>
											</div>
										</template>
										<template v-else>{{ message.content }}</template>
									</template>
									<div v-if="message.role === 'user' && message.pageContext && !isEditing(message)" class="message-page-context">
										<font-awesome-icon :icon="['fas', 'map-marker-alt']" />
										{{ message.pageContext }}
									</div>
									<!-- Use markdown rendering for AI responses -->
									<MarkdownMessage v-else :content="message.content" />
									<!-- Live tool indicator during streaming (before first text chunk) -->
									<span v-if="message.streaming && activeToolCall" class="tool-progress-indicator mt-1 d-block">
										<font-awesome-icon :icon="['fas', 'cog']" spin class="me-1" />
										{{ toolLabel(activeToolCall) }}…
									</span>
								</div>
								<!-- Tool steps used — shown under finished LLM messages -->
								<div v-if="message.role === 'assistant' && message.toolSteps && message.toolSteps.length > 0 && !message.streaming" class="tool-steps-row">
									<font-awesome-icon :icon="['fas', 'wrench']" class="me-1 text-muted" />
									<span v-for="(step, i) in message.toolSteps" :key="i" class="tool-step-badge" :title="step">{{ toolLabel(step) }}</span>
								</div>
								<!-- Follow-up action chips — smart suggestions based on response content -->
								<div v-if="message.role === 'assistant' && !message.streaming && message.followUpActions?.length" class="follow-up-actions">
									<button
										v-for="(action, i) in message.followUpActions"
										:key="i"
										class="follow-up-chip"
										@click="askQuickQuestion(action.message)"
										:title="action.message"
									>
										<font-awesome-icon :icon="action.icon" />
										{{ action.label }}
									</button>
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
									<button v-if="message.role === 'user' && !isEditing(message)" class="btn-msg-action btn-edit-message" @click="startEditMessage(message)" title="Edit message">
										<font-awesome-icon :icon="['fas', 'pen']" />
									</button>
									<button v-if="message.role === 'assistant' && !message.streaming" class="btn-msg-action btn-pin-message" :class="{ active: message.pinned }" @click="togglePinMessage(message)" :title="message.pinned ? 'Unpin' : 'Pin this response'">
										<font-awesome-icon :icon="['fas', 'thumbtack']" />
									</button>
									<button v-if="message.role === 'assistant' && !message.streaming" class="btn-msg-action btn-regenerate" @click="regenerateResponse(message)" title="Regenerate response">
										<font-awesome-icon :icon="['fas', 'redo']" />
									</button>
									<button v-if="message.role === 'assistant' && !message.streaming" class="btn-msg-action btn-msg-reaction" :class="{ active: message.reaction === 'up' }" @click="reactToMessage(message, 'up')" title="Helpful — save to memory">
										<font-awesome-icon :icon="['fas', 'thumbs-up']" />
									</button>
									<button v-if="message.role === 'assistant' && !message.streaming" class="btn-msg-action btn-msg-reaction btn-msg-reaction--down" :class="{ active: message.reaction === 'down' }" @click="reactToMessage(message, 'down')" title="Not helpful">
										<font-awesome-icon :icon="['fas', 'thumbs-down']" />
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
								<span v-if="activeToolCall" class="tool-progress-indicator">
									<font-awesome-icon :icon="['fas', 'cog']" spin class="me-1" />
									{{ toolLabel(activeToolCall) }}…
								</span>
								<span v-else class="typing-indicator">
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
					<!-- /command palette -->
					<transition name="mention-pop">
						<div v-if="slashActive && slashResults.length > 0" class="mention-dropdown slash-palette">
							<div class="slash-palette-header">
								<font-awesome-icon :icon="['fas', 'terminal']" class="me-1" />
								Commands
							</div>
							<div
								v-for="(cmd, i) in slashResults"
								:key="cmd.cmd"
								class="mention-option slash-option"
								:class="{ 'mention-option--active': i === slashIndex }"
								@mousedown.prevent
								@click="selectSlash(cmd)"
								@mouseenter="slashIndex = i"
							>
								<div class="slash-cmd-icon">
									<font-awesome-icon :icon="cmd.icon" />
								</div>
								<div class="slash-cmd-info">
									<span class="slash-cmd-name">/{{ cmd.cmd }}</span>
									<span class="slash-cmd-label">{{ cmd.label }}</span>
								</div>
								<kbd v-if="cmd.param" class="slash-cmd-param">+ query</kbd>
							</div>
						</div>
					</transition>

					<!-- @mention dropdown -->
					<transition name="mention-pop">
						<div v-if="mentionActive && mentionResults.length > 0" class="mention-dropdown">
							<div
								v-for="(performer, i) in mentionResults"
								:key="performer.id"
								class="mention-option"
								:class="{ 'mention-option--active': i === mentionIndex }"
								@mousedown.prevent
								@click="selectMention(performer)"
								@mouseenter="mentionIndex = i"
							>
								<div class="mention-avatar">
									<img
										v-if="performer.thumbnail_path"
										:src="getAssetURL(performer.thumbnail_path)"
										@error="(e) => e.target.style.display = 'none'"
									/>
									<font-awesome-icon v-else :icon="['fas', 'user']" />
								</div>
								<span class="mention-name">{{ performer.name }}</span>
								<span class="mention-count">{{ performer.video_count }} videos</span>
							</div>
						</div>
					</transition>
					<div class="input-group">
						<textarea
							ref="textareaRef"
							v-model="companionInput"
							class="form-control"
							placeholder="Ask anything… / for commands, @name for performers"
							@keydown="handleInputKeydown"
							:disabled="isCompanionThinking"
							rows="1"
						></textarea>
						<button class="btn btn-voice-input" @click="toggleVoiceInput" :class="{ listening: isListening }" :disabled="isCompanionThinking" title="Voice Input">
							<font-awesome-icon :icon="['fas', isListening ? 'stop' : 'microphone']" />
							<span v-if="isListening" class="listening-pulse"></span>
						</button>
						<button v-if="isStreaming" class="btn btn-stop-stream" @click="stopStreaming" title="Stop generating">
							<font-awesome-icon :icon="['fas', 'stop']" />
						</button>
						<button v-else class="btn btn-primary" @click="sendCompanionMessage" :disabled="isCompanionThinking || !companionInput.trim()">
							<font-awesome-icon :icon="['fas', isCompanionThinking ? 'spinner' : 'paper-plane']" :spin="isCompanionThinking" />
						</button>
					</div>
				</div>
			</div>
		</transition>
	</div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted, getCurrentInstance } from 'vue'
import { useRouter } from 'vue-router'
import { aiCompanionAPI, aiAPI, getAssetURL, videosAPI, librariesAPI, activityAPI, performersAPI } from '@/services/api'
import websocketService from '@/services/websocket'
import MarkdownMessage from './MarkdownMessage.vue'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const router = useRouter()

const PAGE_LABELS = {
	Home: 'Dashboard', Videos: 'Videos Library', VideoPlayer: 'Video Player',
	ScrapedVideoPlayer: 'Video Player', Performers: 'Performers', PerformerDetails: 'Performer details',
	Libraries: 'Libraries', Browser: 'File Browser', Studios: 'Studios', Tags: 'Tags',
	Activity: 'Activity Feed', Settings: 'Settings', Tasks: 'Tasks', EditList: 'Edit List',
	Scraper: 'Scraper', ScraperThread: 'Scraper Thread', SystemHealth: 'System Health',
	Backups: 'Backups', JDownloader: 'JDownloader', AI: 'AI Config', Analyzer: 'Video Analyzer',
}
const buildPageContext = () => {
	const route = router.currentRoute.value
	const label = PAGE_LABELS[route.name] ?? String(route.name ?? 'App')
	const parts = [label]
	if (entityContext.value) {
		parts.push(entityContext.value)
	} else if (route.params.id) {
		parts.push(`#${route.params.id}`)
	}
	if (route.query.search) parts.push(`search: "${route.query.search}"`)
	return parts.join(' · ')
}

// Video card helpers
const getVideoThumbURL = (thumbnailPath) => {
	if (!thumbnailPath) return null
	return getAssetURL(thumbnailPath)
}
const navigateToVideo = (videoId) => {
	router.push({ path: '/videos', query: { highlight: videoId } })
	isOpen.value = false
}

const compareCardsInAnalyzer = (videos) => {
	const ids = videos.filter(v => v.id > 0).map(v => v.id).join(',')
	if (!ids) return
	router.push({ path: '/analyzer', query: { ids } })
	isOpen.value = false
}
const rateCardVideo = async (video, rating, event) => {
	event.stopPropagation()
	const newRating = video.rating === rating ? 0 : rating // click same star = clear rating
	try {
		await videosAPI.update(video.id, { rating: newRating })
		video.rating = newRating
	} catch {
		toast.error('Error', 'Failed to update rating')
	}
}
const toggleCardFavorite = async (video, event) => {
	event.stopPropagation()
	const newFav = !video.is_favorite
	try {
		await videosAPI.update(video.id, { is_favorite: newFav })
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
const unreadCount = ref(0)
const showSearch = ref(false)
const searchQuery = ref('')
const searchInput = ref(null)
const textareaRef = ref(null)
const pinnedExpanded = ref(false)
const pinnedMessages = computed(() => chatHistory.value.filter((m) => m.pinned))
const autoResizeTextarea = () => {
	const el = textareaRef.value
	if (!el) return
	el.style.height = 'auto'
	el.style.height = Math.min(el.scrollHeight, 120) + 'px'
}
const filteredMessageCount = computed(() => {
	if (!searchQuery.value) return 0
	const q = searchQuery.value.toLowerCase()
	return chatHistory.value.filter(m => m.content.toLowerCase().includes(q)).length
})

const memoryCounts = computed(() => {
	const counts = {}
	for (const m of memories.value) counts[m.category] = (counts[m.category] || 0) + 1
	return Object.entries(counts).map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count)
})

const filteredMemories = computed(() => {
	let list = memories.value
	if (memoryCategory.value) list = list.filter(m => m.category === memoryCategory.value)
	if (memorySearch.value) {
		const q = memorySearch.value.toLowerCase()
		list = list.filter(m => m.key.toLowerCase().includes(q) || m.value.toLowerCase().includes(q))
	}
	return list
})
const isListening = ref(false)
const recognition = ref(null)
const streamAbortController = ref(null)
const editingMessage = ref(null)
const editingContent = ref('')

// Entity context — name-resolved label + card for entity detail pages (performer, video)
const entityContext = ref(null) // string like "Jessica Jones (12 videos)"
const entityCard = ref(null)    // { type, icon, name, fields: [{icon, value}] }

const _fmtDuration = (secs) => {
	const s = Math.floor(secs)
	const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), sec = s % 60
	return h > 0 ? `${h}h ${m}m` : `${m}:${sec.toString().padStart(2, '0')}`
}
const _ageFromDate = (dateStr) => {
	if (!dateStr) return null
	const age = Math.floor((Date.now() - new Date(dateStr).getTime()) / (365.25 * 24 * 60 * 60 * 1000))
	return age > 0 && age < 120 ? age : null
}

const loadEntityContext = async (route) => {
	entityContext.value = null
	entityCard.value = null
	const { name, params } = route
	if (!params?.id) return
	try {
		if (name === 'PerformerDetails') {
			const res = await performersAPI.getById(params.id)
			const p = res?.data?.data ?? res?.data
			if (!p?.name) return
			const count = p.video_count ?? null
			entityContext.value = count != null ? `${p.name} (${count} videos)` : p.name
			const fields = []
			if (count != null) fields.push({ icon: ['fas', 'film'], value: `${count} videos` })
			const age = _ageFromDate(p.metadata?.birthdate)
			if (age) fields.push({ icon: ['fas', 'birthday-cake'], value: `${age} yrs` })
			if (p.metadata?.ethnicity) fields.push({ icon: ['fas', 'globe'], value: p.metadata.ethnicity })
			if (p.metadata?.hair_color) fields.push({ icon: ['fas', 'user'], value: p.metadata.hair_color })
			if (p.metadata?.height) fields.push({ icon: ['fas', 'arrows-alt'], value: p.metadata.height })
			entityCard.value = { type: 'performer', icon: ['fas', 'user-circle'], name: p.name, fields }
		} else if (name === 'VideoPlayer') {
			const res = await videosAPI.getById(params.id)
			const v = res?.data?.data ?? res?.data
			if (!v?.title) return
			entityContext.value = `"${v.title}"`
			const fields = []
			if (v.duration) fields.push({ icon: ['fas', 'clock'], value: _fmtDuration(v.duration) })
			if (v.resolution) fields.push({ icon: ['fas', 'tv'], value: v.resolution })
			if (v.fps) fields.push({ icon: ['fas', 'film'], value: `${Math.round(v.fps)} fps` })
			if (v.rating > 0) fields.push({ icon: ['fas', 'star'], value: '★'.repeat(v.rating) })
			if (v.performers?.length > 0) fields.push({ icon: ['fas', 'users'], value: v.performers.slice(0, 2).map(p => p.name).join(', ') })
			entityCard.value = { type: 'video', icon: ['fas', 'film'], name: v.title, fields }
		}
	} catch {
		// silently ignore — missing context is better than broken context
	}
}
const typingText = ref('')
const isTyping = ref(false)
const showStats = ref(false)
const showMemoryPanel = ref(false)
const memories = ref([])
const memoriesLoading = ref(false)
const memoryCategory = ref('')
const memorySearch = ref('')
const proactiveEnabled = ref(localStorage.getItem('ai_companion_proactive') !== 'false')
let unsubscribeCompanionEvent = null
let unsubscribeActivityUpdate = null

// Context stats strip
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
	} catch {
		// silently ignore — strip stays hidden
	} finally {
		contextStatsLoading.value = false
	}
}

watch(isOpen, (val) => {
	if (val) {
		contextStatsDismissed.value = false
		loadContextStats()
	}
})

// Proactive entity greeting — fires once per entity per browser session
const _sessionGreeted = new Set(JSON.parse(sessionStorage.getItem('ai_entity_greeted') || '[]'))

const maybeShowEntityGreeting = (card) => {
	const route = router.currentRoute.value
	const key = `${card.type}-${route.params?.id}`
	if (_sessionGreeted.has(key)) return
	_sessionGreeted.add(key)
	sessionStorage.setItem('ai_entity_greeted', JSON.stringify([..._sessionGreeted]))

	let greeting = ''
	if (card.type === 'performer') {
		const vidField = card.fields.find((f) => String(f.value).includes('video'))
		const extras = card.fields.filter((f) => f !== vidField).map((f) => f.value).join(' · ')
		greeting = `I see you're on **${card.name}**'s page${vidField ? ` — ${vidField.value}` : ''}${extras ? ` (${extras})` : ''}. I can summarise their catalog, compare video quality, or find content by tag or rating. What would you like to know?`
	} else if (card.type === 'video') {
		const dur = card.fields.find((f) => f.icon[1] === 'clock')?.value ?? ''
		const res = card.fields.find((f) => f.icon[1] === 'tv')?.value ?? ''
		const cast = card.fields.find((f) => f.icon[1] === 'users')?.value ?? ''
		greeting = `Now playing **"${card.name}"**${dur ? ` · ${dur}` : ''}${res ? ` · ${res}` : ''}${cast ? ` · ${cast}` : ''}. I can suggest similar videos, manage tags, or compare quality with another file. Ask me anything!`
	}

	if (!greeting) return
	chatHistory.value.push({
		role: 'proactive',
		content: greeting,
		timestamp: new Date(),
		severity: 'info',
		source: 'entity_greeting',
	})
	if (!isOpen.value) unreadCount.value++
}

// Keep entity name in sync whenever the route changes
watch(() => router.currentRoute.value, (route) => loadEntityContext(route), { immediate: true })

// Trigger greeting whenever a new entity card resolves
watch(entityCard, (card) => { if (card) maybeShowEntityGreeting(card) })

// Tool progress tracking for live streaming feedback
const activeToolCall = ref('')   // name of the tool currently executing
const streamToolSteps = ref([])  // list of tool names called in this response

const TOOL_LABELS = {
	search_videos: 'Searching videos',
	get_video_details: 'Fetching video details',
	update_video_metadata: 'Updating video metadata',
	get_favorites: 'Loading favorites',
	get_top_rated: 'Loading top-rated videos',
	set_video_rating: 'Setting rating',
	toggle_video_favorite: 'Toggling favorite',
	suggest_videos_to_watch: 'Finding watch suggestions',
	get_library_videos: 'Loading library videos',
	find_duplicate_videos: 'Scanning for duplicates',
	find_orphaned_videos: 'Scanning for orphaned files',
	delete_video_records: 'Deleting records',
	find_untagged_videos: 'Finding untagged videos',
	bulk_apply_tags: 'Bulk applying tags',
	suggest_tags_for_video: 'Suggesting tags',
	apply_tags_to_video: 'Applying tags',
	remove_tags_from_video: 'Removing tags',
	get_all_tags: 'Loading tags',
	search_performers: 'Searching performers',
	get_top_performers: 'Loading top performers',
	get_performer_stats: 'Loading performer stats',
	link_performer_to_video: 'Linking performer',
	unlink_performer_from_video: 'Unlinking performer',
	get_all_libraries: 'Loading libraries',
	scan_library_structure: 'Scanning library structure',
	analyze_library_structure: 'Analyzing library structure',
	suggest_video_placement: 'Finding best placement',
	move_video_to_library: 'Moving video',
	list_folder_contents: 'Listing folder contents',
	create_folder: 'Creating folder',
	rename_folder: 'Renaming folder',
	delete_folder: 'Deleting folder',
	rename_video_file: 'Renaming file',
	delete_video_file: 'Deleting file',
	get_scraper_status: 'Checking scrapers',
	get_jdownloader_status: 'Checking JDownloader',
	search_scraped_content: 'Searching scraped content',
	suggest_scraper_config: 'Analyzing scraper config',
	save_memory: 'Saving memory',
	recall_memories: 'Recalling memories',
	get_video_link_stats: 'Loading link stats',
	get_unlinked_videos: 'Finding unlinked videos',
	auto_link_by_path: 'Auto-linking by path',
	get_library_stats: 'Loading library stats',
	get_daily_briefing: 'Preparing briefing',
	get_library_health: 'Checking library health',
	get_running_tasks: 'Checking running tasks',
	get_recent_activity: 'Loading recent activity',
	get_recent_errors: 'Checking errors',
}

const toolLabel = (name) => TOOL_LABELS[name] || name.replace(/_/g, ' ')

// Heuristic follow-up actions based on response content
const FOLLOW_UP_RULES = [
	{ pattern: /untagged|no tags|missing tags/i, label: 'Auto-tag now', message: 'Auto-tag my untagged videos', icon: ['fas', 'tags'] },
	{ pattern: /duplicate/i, label: 'Find duplicates', message: 'Find duplicate videos in my library', icon: ['fas', 'copy'] },
	{ pattern: /orphaned/i, label: 'Scan orphaned files', message: 'Scan for orphaned files in my library', icon: ['fas', 'exclamation-triangle'] },
	{ pattern: /thumbnail.{0,40}(missing|broken|fail|unavailable)|missing.{0,20}thumbnail/i, label: 'Fix thumbnails', message: 'Run a health check to fix thumbnails', icon: ['fas', 'image'] },
	{ pattern: /error.{0,40}(scraper|download|jdownloader)|(scraper|download|jdownloader).{0,40}error/i, label: 'Check errors', message: 'Check for errors', icon: ['fas', 'exclamation-circle'] },
	{ pattern: /watch next|recommend.{0,20}watch|what.{0,20}watch/i, label: 'Suggest something to watch', message: 'What should I watch next?', icon: ['fas', 'play-circle'] },
	{ pattern: /daily briefing|today.{0,10}summary/i, label: "Today's briefing", message: "Give me today's briefing", icon: ['fas', 'sun'] },
	{ pattern: /top.{0,10}performer|best performer/i, label: 'Top performers', message: 'What are my top performers?', icon: ['fas', 'users'] },
	{ pattern: /top.{0,10}rated|highest rated|best rated/i, label: 'Top rated videos', message: 'Show my top rated videos', icon: ['fas', 'star'] },
]

const deriveFollowUpActions = (text) => {
	if (text.length < 80) return [] // skip trivial/one-line responses
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

// /command palette
const SLASH_COMMANDS = [
	{ cmd: 'briefing',   icon: ['fas', 'sun'],                label: "Today's Briefing",    fill: "Give me today's briefing" },
	{ cmd: 'status',     icon: ['fas', 'bolt'],               label: 'Current Status',       fill: "What's happening right now?" },
	{ cmd: 'health',     icon: ['fas', 'heartbeat'],          label: 'Library Health',       fill: "How's my library doing?" },
	{ cmd: 'search',     icon: ['fas', 'search'],             label: 'Search Videos',        fill: 'Search for videos about ',      param: true },
	{ cmd: 'tag',        icon: ['fas', 'tags'],               label: 'Auto-tag Untagged',    fill: 'Auto-tag my untagged videos' },
	{ cmd: 'duplicates', icon: ['fas', 'copy'],               label: 'Find Duplicates',      fill: 'Find duplicate videos in my library' },
	{ cmd: 'performers', icon: ['fas', 'users'],              label: 'Top Performers',       fill: 'What are my top performers?' },
	{ cmd: 'watch',      icon: ['fas', 'play-circle'],        label: 'Watch Next',           fill: 'What should I watch next?' },
	{ cmd: 'favorites',  icon: ['fas', 'heart'],              label: 'My Favorites',         fill: 'Show my favorite videos' },
	{ cmd: 'errors',     icon: ['fas', 'exclamation-circle'], label: 'Check Errors',         fill: 'Check for errors' },
	{ cmd: 'orphaned',   icon: ['fas', 'exclamation-triangle'],label: 'Orphaned Files',      fill: 'Scan for orphaned files in my library' },
	{ cmd: 'stats',      icon: ['fas', 'chart-bar'],          label: 'Library Stats',        fill: 'Library overview' },
	{ cmd: 'tasks',      icon: ['fas', 'tasks'],              label: 'Running Tasks',        fill: 'What tasks are currently running?' },
	{ cmd: 'jdownloader',icon: ['fas', 'download'],           label: 'JDownloader Status',   fill: 'JDownloader status' },
	{ cmd: 'scrapers',   icon: ['fas', 'spider'],             label: 'Scraper Summary',      fill: 'Scraper summary' },
]

const slashActive = ref(false)
const slashResults = ref([])
const slashIndex = ref(0)

const closeSlash = () => {
	slashActive.value = false
	slashResults.value = []
	slashIndex.value = 0
}

const selectSlash = (cmd) => {
	if (!cmd) return
	companionInput.value = cmd.fill
	closeSlash()
	if (!cmd.param) nextTick(() => sendCompanionMessage())
}

watch(companionInput, () => nextTick(autoResizeTextarea))

watch(companionInput, (val) => {
	if (!val.startsWith('/')) { closeSlash(); return }
	const query = val.slice(1).toLowerCase()
	slashResults.value = SLASH_COMMANDS.filter(
		(c) => c.cmd.startsWith(query) || c.label.toLowerCase().includes(query)
	)
	slashActive.value = slashResults.value.length > 0
	slashIndex.value = 0
})

// Consolidated keyboard handler for both dropdowns
const handleInputKeydown = (e) => {
	const inMention = mentionActive.value && mentionResults.value.length > 0
	const inSlash = slashActive.value && slashResults.value.length > 0

	if (!inMention && !inSlash) {
		if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendCompanionMessage() }
		else if (e.key === 'ArrowUp' && !e.shiftKey && sentHistory.value.length > 0 && !companionInput.value.includes('\n')) {
			e.preventDefault()
			if (historyIndex.value === -1) savedDraft.value = companionInput.value
			historyIndex.value = Math.min(historyIndex.value + 1, sentHistory.value.length - 1)
			companionInput.value = sentHistory.value[sentHistory.value.length - 1 - historyIndex.value]
		}
		else if (e.key === 'ArrowDown' && historyIndex.value !== -1 && !companionInput.value.includes('\n')) {
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
		: () => selectSlash(slashResults.value[slashIndex.value])

	if (e.key === 'ArrowDown') { e.preventDefault(); if (index.value < results.value.length - 1) index.value++ }
	else if (e.key === 'ArrowUp') { e.preventDefault(); if (index.value > 0) index.value-- }
	else if (e.key === 'Enter' || e.key === 'Tab') { e.preventDefault(); select() }
	else if (e.key === 'Escape') { e.preventDefault(); inMention ? closeMention() : closeSlash() }
}

// Prompt history — terminal-style ArrowUp/Down cycling
const sentHistory = ref(JSON.parse(localStorage.getItem('ai_companion_prompt_history') || '[]'))
const historyIndex = ref(-1)
const savedDraft = ref('')

// @mention autocomplete
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
watch(companionInput, (val) => {
	if (_mentionTimer) clearTimeout(_mentionTimer)
	const idx = val.lastIndexOf('@')
	// Only trigger when @ is at start or preceded by whitespace (not in email addresses)
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

// Contextual suggested prompts — change based on the active page
const PAGE_SUGGESTIONS = {
	Home: [
		{ text: "Give me today's briefing", icon: ['fas', 'sun'] },
		{ text: "What's happening right now?", icon: ['fas', 'bolt'] },
		{ text: 'Library overview', icon: ['fas', 'chart-bar'] },
		{ text: 'Check for errors', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Show library insights', icon: ['fas', 'lightbulb'] },
		{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
	],
	Videos: [
		{ text: 'Find duplicate videos', icon: ['fas', 'copy'] },
		{ text: 'Auto-tag my untagged videos', icon: ['fas', 'tags'] },
		{ text: 'Show my top rated videos', icon: ['fas', 'star'] },
		{ text: 'What videos are missing metadata?', icon: ['fas', 'exclamation-circle'] },
		{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
		{ text: 'Show my favorite videos', icon: ['fas', 'heart'] },
	],
	Browser: [
		{ text: 'Help me organize my downloads', icon: ['fas', 'folder-open'] },
		{ text: "What's in my Downloads library?", icon: ['fas', 'download'] },
		{ text: 'Find unmapped video files', icon: ['fas', 'search-location'] },
		{ text: 'Scan for orphaned files', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Move these files to Backup', icon: ['fas', 'hdd'] },
		{ text: 'Which files should I keep?', icon: ['fas', 'question-circle'] },
	],
	Performers: [
		{ text: 'Who are my top performers?', icon: ['fas', 'users'] },
		{ text: 'Tell me about my top performer', icon: ['fas', 'user-circle'] },
		{ text: 'Find performers without photos', icon: ['fas', 'camera'] },
		{ text: 'Show performer statistics', icon: ['fas', 'chart-bar'] },
		{ text: 'Merge duplicate performers', icon: ['fas', 'user-check'] },
	],
	PerformerDetails: [
		{ text: 'Summarise this performer', icon: ['fas', 'user-circle'] },
		{ text: 'Find all videos with this performer', icon: ['fas', 'film'] },
		{ text: 'Suggest tags for this performer', icon: ['fas', 'tags'] },
		{ text: 'Rate all unwatched videos for this performer', icon: ['fas', 'star'] },
	],
	Libraries: [
		{ text: 'Check library health', icon: ['fas', 'heartbeat'] },
		{ text: 'Find missing files in my libraries', icon: ['fas', 'exclamation-circle'] },
		{ text: 'How much storage am I using?', icon: ['fas', 'hdd'] },
		{ text: 'Scan my libraries for new videos', icon: ['fas', 'sync-alt'] },
	],
	Studios: [
		{ text: 'What are my top studios?', icon: ['fas', 'building'] },
		{ text: 'Show studio statistics', icon: ['fas', 'chart-bar'] },
		{ text: 'Find videos without a studio', icon: ['fas', 'exclamation-circle'] },
		{ text: 'Merge duplicate studios', icon: ['fas', 'clone'] },
	],
	Tags: [
		{ text: 'What are my most used tags?', icon: ['fas', 'tags'] },
		{ text: 'Find videos with no tags', icon: ['fas', 'tag'] },
		{ text: 'Auto-tag my untagged videos', icon: ['fas', 'magic'] },
		{ text: 'Suggest tag categories for my library', icon: ['fas', 'sitemap'] },
	],
	Activity: [
		{ text: "What's happened recently?", icon: ['fas', 'history'] },
		{ text: 'Show scraper activity', icon: ['fas', 'spider'] },
		{ text: 'Any errors in the last 24 hours?', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Summarise this week', icon: ['fas', 'chart-bar'] },
	],
	Tasks: [
		{ text: "What tasks are running right now?", icon: ['fas', 'circle-notch'] },
		{ text: 'Show completed tasks', icon: ['fas', 'check-circle'] },
		{ text: 'Are there any failed tasks?', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Cancel all running tasks', icon: ['fas', 'stop'] },
	],
	Scraper: [
		{ text: "What's scraping right now?", icon: ['fas', 'spider'] },
		{ text: 'Scraper summary', icon: ['fas', 'chart-bar'] },
		{ text: 'Check for scraper errors', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Show recent scraper activity', icon: ['fas', 'history'] },
	],
	ScraperThread: [
		{ text: 'Summarise this scraper', icon: ['fas', 'spider'] },
		{ text: 'Show errors for this scraper', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'What did this scraper find recently?', icon: ['fas', 'history'] },
	],
	JDownloader: [
		{ text: 'JDownloader status', icon: ['fas', 'download'] },
		{ text: "What's downloading right now?", icon: ['fas', 'circle-notch'] },
		{ text: 'Show download queue', icon: ['fas', 'list'] },
		{ text: 'Any failed downloads?', icon: ['fas', 'exclamation-triangle'] },
	],
	Analyzer: [
		{ text: 'Which file is best for Topaz upscaling?', icon: ['fas', 'star'] },
		{ text: 'Explain the sharpness scores', icon: ['fas', 'search-plus'] },
		{ text: 'Find my highest quality videos', icon: ['fas', 'trophy'] },
		{ text: 'What does perceptual efficiency mean?', icon: ['fas', 'question-circle'] },
	],
	AI: [
		{ text: "Give me today's briefing", icon: ['fas', 'sun'] },
		{ text: 'Show library insights', icon: ['fas', 'lightbulb'] },
		{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
		{ text: 'Find duplicate videos', icon: ['fas', 'copy'] },
	],
	EditList: [
		{ text: 'Help me build an edit list', icon: ['fas', 'film'] },
		{ text: 'Find videos for a compilation', icon: ['fas', 'list-check'] },
		{ text: 'Suggest scenes by performer', icon: ['fas', 'user'] },
	],
	SystemHealth: [
		{ text: 'How is my system doing?', icon: ['fas', 'heartbeat'] },
		{ text: 'Check for errors in the last scan', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Show database statistics', icon: ['fas', 'database'] },
	],
}

const _genericPrompts = [
	{ text: "What's happening right now?", icon: ['fas', 'bolt'] },
	{ text: 'Library overview', icon: ['fas', 'chart-bar'] },
	{ text: 'Check for errors', icon: ['fas', 'exclamation-triangle'] },
	{ text: 'Show library insights', icon: ['fas', 'lightbulb'] },
	{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
	{ text: "Give me today's briefing", icon: ['fas', 'sun'] },
]

const suggestedPrompts = computed(() => {
	const routeName = router.currentRoute.value.name
	return PAGE_SUGGESTIONS[routeName] ?? _genericPrompts
})

const suggestedPageLabel = computed(() => {
	const routeName = router.currentRoute.value.name
	if (!PAGE_SUGGESTIONS[routeName]) return null
	return entityContext.value ?? (PAGE_LABELS[routeName] ?? String(routeName))
})

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
	unreadCount.value = 0
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

	const pageContext = buildPageContext()

	// Add user message to chat
	chatHistory.value.push({
		role: 'user',
		content: userMessage,
		pageContext,
		timestamp: new Date(),
	})

	saveChatHistory()
	// Push to prompt history (capped at 50, persisted across reloads)
	sentHistory.value.push(userMessage)
	if (sentHistory.value.length > 50) sentHistory.value.shift()
	historyIndex.value = -1
	savedDraft.value = ''
	localStorage.setItem('ai_companion_prompt_history', JSON.stringify(sentHistory.value))
	companionInput.value = ''
	isCompanionThinking.value = true
	activeToolCall.value = ''
	streamToolSteps.value = []

	await nextTick()
	scrollToBottom()

	try {
		const historyPayload = chatHistory.value
			.slice(0, -1) // exclude the user message we just pushed
			.filter((msg) => msg.role === 'user' || msg.role === 'assistant')
			.slice(-10)
			.map((msg) => ({ role: msg.role, content: msg.content }))

		streamAbortController.value = new AbortController()
		const response = await fetch('http://localhost:8080/api/v1/ai/chat/stream', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ message: `[User is on: ${pageContext}]\n${userMessage}`, history: historyPayload }),
			signal: streamAbortController.value.signal,
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

				if (currentEventType === 'tool_call') {
					try {
						const { tool } = JSON.parse(data)
						activeToolCall.value = tool
						streamToolSteps.value.push(tool)
					} catch { /* ignore */ }
					currentEventType = ''
					continue
				}

				try {
					const chunk = JSON.parse(data)
					// First text chunk — hide tool activity indicator
					if (chunk && activeToolCall.value) activeToolCall.value = ''
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
		if (streamToolSteps.value.length > 0) {
			chatHistory.value[msgIndex].toolSteps = [...streamToolSteps.value]
		}
		const followUps = deriveFollowUpActions(chatHistory.value[msgIndex].content)
		if (followUps.length > 0) {
			chatHistory.value[msgIndex].followUpActions = followUps
		}
		activeToolCall.value = ''
		streamToolSteps.value = []
		isStreaming.value = false
		saveChatHistory()

		if (!isOpen.value) unreadCount.value++
		await nextTick()
		scrollToBottom()
	} catch (error) {
		isCompanionThinking.value = false
		isStreaming.value = false
		activeToolCall.value = ''
		streamToolSteps.value = []
		if (error.name === 'AbortError') {
			// User cancelled — finalize whatever was streamed so far
			const lastMsg = chatHistory.value[chatHistory.value.length - 1]
			if (lastMsg?.streaming) {
				lastMsg.streaming = false
				if (lastMsg.content.length === 0) {
					chatHistory.value.pop() // nothing arrived yet — remove placeholder
				}
			}
			return
		}
		console.error('Backend AI failed:', error)
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

const stopStreaming = () => {
	streamAbortController.value?.abort()
}

const isEditing = (msg) => {
	const hist = msg.originalIndex !== undefined ? chatHistory.value[msg.originalIndex] : msg
	return editingMessage.value === hist
}
const startEditMessage = (message) => {
	const hist = message.originalIndex !== undefined ? chatHistory.value[message.originalIndex] : message
	editingMessage.value = hist
	editingContent.value = hist.content
}
const cancelEditMessage = () => {
	editingMessage.value = null
	editingContent.value = ''
}
const confirmEditMessage = async () => {
	const msg = editingMessage.value
	const newContent = editingContent.value.trim()
	if (!msg || !newContent) return
	const idx = chatHistory.value.indexOf(msg)
	if (idx < 0) return
	cancelEditMessage()
	chatHistory.value.splice(idx)
	companionInput.value = newContent
	await nextTick()
	await sendCompanionMessage()
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

// Regenerate response — takes the message object so it works in both normal + search mode
const regenerateResponse = async (message) => {
	const histIdx = (() => {
		const i = chatHistory.value.indexOf(message)
		return i !== -1 ? i : (message.originalIndex ?? -1)
	})()
	if (histIdx < 1) return
	const prevMsg = chatHistory.value[histIdx - 1]
	if (!prevMsg || prevMsg.role !== 'user') return
	const userContent = prevMsg.content
	chatHistory.value.splice(histIdx - 1, 2)
	companionInput.value = userContent
	await nextTick()
	await sendCompanionMessage()
}

// React to an AI message — thumbs up saves it as a memory, both toggle
const reactToMessage = async (message, reaction) => {
	const histMsg = chatHistory.value.find((m) => m === message)
		?? (message.originalIndex !== undefined ? chatHistory.value[message.originalIndex] : null)
	if (!histMsg) return
	if (histMsg.reaction === reaction) {
		histMsg.reaction = null
		return
	}
	histMsg.reaction = reaction
	if (reaction === 'up') {
		try {
			await aiAPI.saveMemory({
				key: `helpful_response_${Date.now()}`,
				value: histMsg.content.slice(0, 1000),
				category: 'insight',
			})
			toast.success('Saved to memory', '', 2500)
		} catch {
			// best-effort
		}
	}
}

const togglePinMessage = (message) => {
	const histMsg = chatHistory.value.find((m) => m === message)
		?? (message.originalIndex !== undefined ? chatHistory.value[message.originalIndex] : null)
	if (!histMsg) return
	histMsg.pinned = !histMsg.pinned
	if (histMsg.pinned && !pinnedExpanded.value) pinnedExpanded.value = true
}

const scrollToMessage = (msg) => {
	const ts = msg.timestamp?.getTime?.()
	if (!ts) return
	const el = chatMessages.value?.querySelector(`[data-msg-ts="${ts}"]`)
	if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
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

const loadMemories = async () => {
	memoriesLoading.value = true
	try {
		const res = await aiAPI.getMemories()
		memories.value = (res?.data?.data ?? res?.data ?? []).sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at))
	} catch {
		// silently ignore
	} finally {
		memoriesLoading.value = false
	}
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
		memories.value = memories.value.filter(m => m.id !== id)
		toast.success('Forgotten', '', 2000)
	} catch {
		toast.error('Error', 'Failed to delete memory')
	}
}

const formatMemoryDate = (dateStr) => {
	if (!dateStr) return ''
	return new Date(dateStr).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
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
		unreadCount.value++
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

// Global keyboard shortcut: Ctrl+Shift+A toggles the chat from anywhere
const handleGlobalShortcut = (e) => {
	if (e.ctrlKey && e.shiftKey && (e.key === 'A' || e.key === 'a')) {
		e.preventDefault()
		toggleChat()
	}
}

// Handler for the global open-ai-chat event (from ScraperPage Ask AI button)
const handleOpenAIChat = (event) => {
	isOpen.value = true
	unreadCount.value = 0
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

	// Refresh context stats whenever an activity completes/fails
	unsubscribeActivityUpdate = websocketService.on('activity_update', () => {
		if (isOpen.value && !contextStatsDismissed.value) loadContextStats()
	})

	// Listen for external requests to open with a pre-filled message
	window.addEventListener('open-ai-chat', handleOpenAIChat)
	document.addEventListener('keydown', handleGlobalShortcut)

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
	if (unsubscribeCompanionEvent) unsubscribeCompanionEvent()
	if (unsubscribeActivityUpdate) unsubscribeActivityUpdate()
	window.removeEventListener('open-ai-chat', handleOpenAIChat)
	document.removeEventListener('keydown', handleGlobalShortcut)
})
</script>

<style scoped>
@import '@/styles/components/ai_companion_chat.css';
</style>
