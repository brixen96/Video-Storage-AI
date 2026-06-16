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
import { aiCompanionAPI } from '@/services/api'
import websocketService from '@/services/websocket'
import MarkdownMessage from './MarkdownMessage.vue'

import { PAGE_LABELS, PAGE_SUGGESTIONS, GENERIC_PROMPTS, WELCOME_MESSAGE } from '@/composables/ai/constants'
import { useChatHistory } from '@/composables/ai/useChatHistory'
import { useChatStreaming } from '@/composables/ai/useChatStreaming'
import { useChatInput } from '@/composables/ai/useChatInput'
import { useChatUI } from '@/composables/ai/useChatUI'
import { useEntityContext } from '@/composables/ai/useEntityContext'
import { useVideoCards } from '@/composables/ai/useVideoCards'
import { useVoiceInput } from '@/composables/ai/useVoiceInput'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// ── Composables ──────────────────────────────────────────────
const {
	chatHistory, showSearch, searchQuery, sentHistory, historyIndex, savedDraft,
	pinnedMessages, filteredMessageCount, displayedMessages, chatStats,
	resolveIndex, pushMessage, popMessages, removeMessage, replaceRange,
	clearSearch, loadHistory, clearHistory, pushPromptHistory,
} = useChatHistory()

const {
	isStreaming, activeToolCall, streamToolSteps, deriveFollowUpActions,
	streamMessage, stopStreaming, toolLabel,
} = useChatStreaming()

const {
	companionInput, textareaRef, slashActive, slashResults, slashIndex,
	mentionActive, mentionResults, mentionIndex,
	closeSlash, closeMention, handleInputKeydown: _rawHandleKeydown, autoResizeTextarea,
	setupWatchers: setupInputWatchers, cleanupWatchers: cleanupInputWatchers,
} = useChatInput({ sentHistory, historyIndex, savedDraft, pushPromptHistory })

// Wraps the composable's keydown handler with component-level callbacks
const handleInputKeydown = (e) => {
	_rawHandleKeydown(e, {
		onSend: sendCompanionMessage,
		onMentionClose: closeMention,
		onSlashClose: closeSlash,
	})
}

const {
	isOpen, unreadCount, chatMessages, showStats, proactiveEnabled, companionConnected,
	showMemoryPanel, memories, memoriesLoading, memoryCategory, memorySearch,
	memoryCounts, filteredMemories, loadMemories, toggleMemoryPanel, deleteMemory, formatMemoryDate,
	contextStats, contextStatsLoading, contextStatsDismissed, loadContextStats,
	scrollToBottom, toggleProactive, toggleChat,
	setupShortcuts, cleanupShortcuts,
} = useChatUI({ chatHistory, toggleChat: null }) // wired up below

const {
	entityContext, entityCard, buildPageContext, setupWatchers: setupEntityWatchers,
} = useEntityContext()

const {
	getVideoThumbURL, navigateToVideo: _navToVideo, compareCardsInAnalyzer: _compareInAnalyzer,
	rateCardVideo: _rateCard, toggleCardFavorite: _toggleFav,
} = useVideoCards()

const { isListening, toggleVoiceInput: _toggleVoice } = useVoiceInput({
	onResult: (transcript, isFinal) => {
		companionInput.value = transcript
		if (isFinal) sendCompanionMessage()
	},
})

// ── Video card wrappers (with stopPropagation + close + toast) ──
const navigateToVideo = (videoId) => { _navToVideo(videoId, () => { isOpen.value = false }) }
const compareCardsInAnalyzer = (videos) => { _compareInAnalyzer(videos, () => { isOpen.value = false }) }
const rateCardVideo = async (video, rating, event) => {
	event.stopPropagation()
	await _rateCard(video, rating)
}
const toggleCardFavorite = async (video, event) => {
	event.stopPropagation()
	await _toggleFav(video)
}
const askAboutCard = (video, event) => {
	event.stopPropagation()
	companionInput.value = `Suggest tags for video ID ${video.id}: "${video.title}"`
	nextTick(() => sendCompanionMessage())
}
const toggleVoiceInput = () => {
	if (!_toggleVoice()) {
		toast.error('Not Supported', 'Voice input is not supported in your browser')
	}
}

// ── State that depends on other composables ──
const isCompanionThinking = ref(false)
const editingMessage = ref(null)
const editingContent = ref('')

// ── Suggested prompts (context-aware per page) ──
const suggestedPrompts = computed(() => {
	const routeName = proxy.$router.currentRoute.value.name
	return PAGE_SUGGESTIONS[routeName] ?? GENERIC_PROMPTS
})
const suggestedPageLabel = computed(() => {
	const routeName = proxy.$router.currentRoute.value.name
	if (!PAGE_SUGGESTIONS[routeName]) return null
	return entityContext.value ?? (PAGE_LABELS[routeName] ?? String(routeName))
})

// ── Pinned messages panel state ──
const pinnedExpanded = ref(false)
const searchInput = ref(null)

// ── Watch context stats when chat opens ──
watch(isOpen, (val) => {
	if (val) {
		contextStatsDismissed.value = false
		loadContextStats()
	}
})

// ── Chat actions ──
const minimizeChat = () => { isOpen.value = false }
const closeChat = () => { isOpen.value = false }

const checkCompanionConnection = async () => {
	try {
		await aiCompanionAPI.chat([{ role: 'user', content: 'Hi' }], { max_tokens: 5 })
		companionConnected.value = true
	} catch {
		companionConnected.value = true // Still connected — fallback available
	}
}

const sendCompanionMessage = async () => {
	const userMessage = companionInput.value.trim()
	if (!userMessage) return

	const pageContext = buildPageContext()
	pushMessage({
		role: 'user', content: userMessage, pageContext, timestamp: new Date(),
	})
	pushPromptHistory(userMessage)
	companionInput.value = ''
	isCompanionThinking.value = true

	await nextTick()
	scrollToBottom()

	const historyPayload = chatHistory.value
		.slice(0, -1)
		.filter((m) => m.role === 'user' || m.role === 'assistant')
		.slice(-10)
		.map((m) => ({ role: m.role, content: m.content }))

	const fullMessage = `[User is on: ${pageContext}]\n${userMessage}`

	const msgIndex = chatHistory.value.length
	pushMessage({
		role: 'assistant', content: '', timestamp: new Date(),
		source: 'llm', streaming: true,
	})
	isCompanionThinking.value = false

	await nextTick()
	scrollToBottom()

	streamMessage(fullMessage, historyPayload, {
		onTextChunk: (chunk) => {
			chatHistory.value[msgIndex].content += chunk
		},
		onVideoCards: (videos) => {
			pushMessage({ role: 'cards', source: 'video_cards', videos, timestamp: new Date() })
		},
		scrollFn: async () => { await nextTick(); scrollToBottom() },
		onDone: (toolSteps) => {
			chatHistory.value[msgIndex].streaming = false
			if (toolSteps.length > 0) chatHistory.value[msgIndex].toolSteps = toolSteps
			const followUps = deriveFollowUpActions(chatHistory.value[msgIndex].content)
			if (followUps.length > 0) chatHistory.value[msgIndex].followUpActions = followUps
			if (!isOpen.value) unreadCount.value++
		},
		onError: (error) => {
			if (error.name === 'AbortError') {
				const lastMsg = chatHistory.value[chatHistory.value.length - 1]
				if (lastMsg?.streaming) {
					lastMsg.streaming = false
					if (lastMsg.content.length === 0) popMessages(1)
				}
				return
			}
			const lastMsg = chatHistory.value[chatHistory.value.length - 1]
			if (lastMsg?.streaming) lastMsg.streaming = false
			pushMessage({
				role: 'assistant',
				content: "I'm currently unavailable. Please check that the backend server is running and try again.",
				timestamp: new Date(),
			})
			toast.error('AI Unavailable', 'Could not connect to AI Companion service. Please check the backend server.')
		},
	})
}

const askQuickQuestion = (question) => {
	companionInput.value = question
	sendCompanionMessage()
}

// ── Message editing ──
const isEditing = (msg) => {
	const hist = msg.originalIndex !== undefined ? chatHistory.value[msg.originalIndex] : msg
	return editingMessage.value === hist
}
const startEditMessage = (message) => {
	const hist = message.originalIndex !== undefined ? chatHistory.value[message.originalIndex] : message
	editingMessage.value = hist
	editingContent.value = hist.content
}
const cancelEditMessage = () => { editingMessage.value = null; editingContent.value = '' }
const confirmEditMessage = async () => {
	const msg = editingMessage.value
	const newContent = editingContent.value.trim()
	if (!msg || !newContent) return
	const idx = chatHistory.value.indexOf(msg)
	if (idx < 0) return
	cancelEditMessage()
	replaceRange(idx, 1)
	companionInput.value = newContent
	await nextTick()
	await sendCompanionMessage()
}

// ── Clear / search / export / copy ──
const clearChatHistory = async () => {
	if (chatHistory.value.length === 0) return
	if (!confirm('Are you sure you want to clear the chat history?')) return
	await clearHistory()
	toast.info('Chat Cleared', 'Chat history has been cleared')
}

const toggleSearch = async () => {
	showSearch.value = !showSearch.value
	if (showSearch.value) {
		await nextTick()
		searchInput.value?.focus()
	} else {
		clearSearch()
	}
}

const exportChat = () => {
	const dataStr = JSON.stringify({
		exported_at: new Date().toISOString(),
		message_count: chatHistory.value.length,
		messages: chatHistory.value,
	}, null, 2)
	const url = URL.createObjectURL(new Blob([dataStr], { type: 'application/json' }))
	const link = document.createElement('a')
	link.href = url
	link.download = `ai-companion-chat-${new Date().toISOString().split('T')[0]}.json`
	link.click()
	URL.revokeObjectURL(url)
	toast.success('Chat Exported', 'Your conversation has been exported successfully')
}

const copyMessage = async (content) => {
	try {
		await navigator.clipboard.writeText(content)
		toast.success('Copied!', 'Message copied to clipboard')
	} catch {
		toast.error('Copy Failed', 'Could not copy message to clipboard')
	}
}

// ── Regenerate / react / pin ──
const regenerateResponse = async (message) => {
	const histIdx = resolveIndex(message)
	if (histIdx < 1) return
	const prevMsg = chatHistory.value[histIdx - 1]
	if (!prevMsg || prevMsg.role !== 'user') return
	replaceRange(histIdx - 1, 2)
	companionInput.value = prevMsg.content
	await nextTick()
	await sendCompanionMessage()
}

const reactToMessage = async (message, reaction) => {
	const histMsg = chatHistory.value.find((m) => m === message)
		?? (message.originalIndex !== undefined ? chatHistory.value[message.originalIndex] : null)
	if (!histMsg) return
	if (histMsg.reaction === reaction) { histMsg.reaction = null; return }
	histMsg.reaction = reaction
	if (reaction === 'up') {
		try {
			const { aiAPI } = await import('@/services/api')
			await aiAPI.saveMemory({
				key: `helpful_response_${Date.now()}`,
				value: histMsg.content.slice(0, 1000),
				category: 'insight',
			})
			toast.success('Saved to memory', '', 2500)
		} catch { /* best-effort */ }
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

// ── Formatting helpers (used in template) ──
const formatMessageTime = (timestamp) => {
	const date = new Date(timestamp)
	return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
}

// ── Entity context greeting ──
let unsubscribeCompanionEvent = null
let unsubscribeActivityUpdate = null
setupEntityWatchers(chatHistory, unreadCount, isOpen)

// ── External open-ai-chat event (from ScraperPage Ask AI button) ──
const handleOpenAIChat = (event) => {
	isOpen.value = true
	unreadCount.value = 0
	const msg = event.detail?.message
	if (msg) {
		companionInput.value = msg
		nextTick(() => sendCompanionMessage())
	}
	nextTick(() => scrollToBottom())
}

// ── Lifecycle ──
onMounted(async () => {
	await loadHistory()
	await checkCompanionConnection()

	setupInputWatchers()

	// Subscribe to proactive companion events via WebSocket
	unsubscribeCompanionEvent = websocketService.on('companion_event', (event) => {
		if (!proactiveEnabled.value) return
		pushMessage({
			role: 'proactive', content: event.message, timestamp: new Date(),
			severity: event.type || event.severity || 'info', source: event.source || 'system',
		})
		if (!isOpen.value) unreadCount.value++
		nextTick(() => scrollToBottom())
	})

	// Refresh context stats whenever an activity completes/fails
	unsubscribeActivityUpdate = websocketService.on('activity_update', () => {
		if (isOpen.value && !contextStatsDismissed.value) loadContextStats()
	})

	window.addEventListener('open-ai-chat', handleOpenAIChat)
	setupShortcuts()

	// Welcome message on first open
	if (companionConnected.value && chatHistory.value.length === 0) {
		pushMessage({ role: 'assistant', content: WELCOME_MESSAGE, timestamp: new Date() })
	}
})

onUnmounted(() => {
	cleanupInputWatchers()
	if (unsubscribeCompanionEvent) unsubscribeCompanionEvent()
	if (unsubscribeActivityUpdate) unsubscribeActivityUpdate()
	window.removeEventListener('open-ai-chat', handleOpenAIChat)
	cleanupShortcuts()
})
</script>

<style scoped>
@import '@/styles/components/ai_companion_chat.css';
</style>
