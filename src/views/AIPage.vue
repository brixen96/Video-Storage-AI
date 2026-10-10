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
			</li>
			<li class="nav-item">
				<button class="nav-link" :class="{ active: activeTab === 'bannerlord' }" @click="activeTab = 'bannerlord'">
					<font-awesome-icon :icon="['fas', 'horse']" class="me-2" />
					Bannerlord
				</button>
			</li>
			</ul>

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
										<span class="config-value">{{ aiConfig.apiUrl || 'Not configured' }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Model</span>
										<span class="config-value">{{ aiConfig.modelName || 'Auto-detect' }}</span>
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
										<span class="config-value">{{ aiConfig.maxTokens }}</span>
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
				<div class="chat-layout">
					<!-- Chat Section -->
					<div class="chat-section">
						<div class="chat-section-header">
							<h3>
								<font-awesome-icon :icon="['fas', 'comments']" class="me-2" />
								Chat with AI Companion
							</h3>
							<div class="d-flex gap-2 align-items-center">
								<span v-if="aiConfig && aiConfig.reasoningEffort && aiConfig.reasoningEffort !== 'none'" class="badge bg-info-subtle text-info border border-info">
									<font-awesome-icon :icon="['fas', 'brain']" class="me-1" />
									{{ aiConfig.reasoningEffort }}
								</span>
								<button class="btn btn-sm btn-outline-danger" @click="clearChat" :disabled="chatHistory.length === 0 || isStreaming">
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

					<!-- Inspector Sidebar -->
					<div class="inspector-panel">
						<div class="inspector-tabs">
							<button class="inspector-tab" :class="{ active: inspectorTab === 'session' }" @click="inspectorTab = 'session'">
								<font-awesome-icon :icon="['fas', 'history']" class="me-1" />
								Session
							</button>
							<button class="inspector-tab" :class="{ active: inspectorTab === 'controls' }" @click="inspectorTab = 'controls'">
								<font-awesome-icon :icon="['fas', 'sliders-h']" class="me-1" />
								Controls
							</button>
							<button class="inspector-tab" :class="{ active: inspectorTab === 'memory' }" @click="inspectorTab = 'memory'">
								<font-awesome-icon :icon="['fas', 'brain']" class="me-1" />
								Memory
							</button>
							<button class="inspector-tab" :class="{ active: inspectorTab === 'roleplay' }" @click="inspectorTab = 'roleplay'">
								<font-awesome-icon :icon="['fas', 'mask']" class="me-1" />
								RP
							</button>
						</div>
						<div class="inspector-body">
							<!-- Session: live request/response insight -->
							<div v-if="inspectorTab === 'session'">
								<div class="inspector-section-title">Token Usage</div>
								<div class="token-grid">
									<div class="token-card">
										<span class="token-value">{{ lastRun.tokens.prompt }}</span>
										<span class="token-label">Prompt</span>
									</div>
									<div class="token-card">
										<span class="token-value">{{ lastRun.tokens.completion }}</span>
										<span class="token-label">Completion</span>
									</div>
									<div class="token-card reasoning">
										<span class="token-value">{{ lastRun.tokens.reasoning }}</span>
										<span class="token-label">Reasoning</span>
									</div>
									<div class="token-card total">
										<span class="token-value">{{ lastRun.tokens.total }}</span>
										<span class="token-label">Total</span>
									</div>
								</div>

								<div class="inspector-section-title mt-3">LLM Requests</div>
								<div v-if="lastRun.requests.length === 0" class="inspector-empty">No requests yet — send a message to see request info.</div>
								<div v-for="(req, i) in lastRun.requests" :key="'req-' + i" class="request-card">
									<div class="request-header">
										<span class="request-iter">#{{ req.iteration }}</span>
										<span :class="['badge', req.status === 'ok' ? 'bg-success' : 'bg-danger']">{{ req.status }}</span>
									</div>
									<div class="request-meta">
										<span><font-awesome-icon :icon="['fas', 'microchip']" class="me-1" />{{ req.model }}</span>
										<span><font-awesome-icon :icon="['fas', 'tools']" class="me-1" />{{ req.tools_sent }} tools</span>
										<span><font-awesome-icon :icon="['fas', 'database']" class="me-1" />{{ (req.body_size / 1024).toFixed(1) }} KB</span>
										<span><font-awesome-icon :icon="['fas', 'clock']" class="me-1" />{{ (req.latency_ms / 1000).toFixed(1) }}s</span>
										<span v-if="req.effort && req.effort !== 'none'"><font-awesome-icon :icon="['fas', 'brain']" class="me-1" />{{ req.effort }}</span>
									</div>
								</div>

								<div class="inspector-section-title mt-3">Tool Calls</div>
								<div v-if="lastRun.toolCalls.length === 0" class="inspector-empty">No tools called in this exchange.</div>
								<div v-for="(tc, i) in lastRun.toolCalls" :key="'tc-' + i" class="tool-row">
									<font-awesome-icon :icon="['fas', isStreaming && i === lastRun.toolCalls.length - 1 ? 'spinner' : 'check']" :class="['tool-icon', isStreaming && i === lastRun.toolCalls.length - 1 ? 'fa-spin text-info' : 'text-success']" />
									<span class="tool-name">{{ toolLabel(tc.name) }}</span>
									<span class="tool-time">{{ tc.at }}</span>
								</div>
							</div>

							<!-- Controls -->
							<div v-if="inspectorTab === 'controls'">
								<div class="inspector-section-title">Thinking Effort</div>
								<p class="inspector-hint">Controls OpenRouter reasoning tokens. Applied to every request. "None" disables reasoning entirely.</p>
								<div class="effort-grid">
									<button v-for="opt in effortOptions" :key="opt.value" class="effort-btn" :class="{ active: (aiConfig?.reasoningEffort || 'medium') === opt.value }" @click="setEffort(opt.value)" :disabled="savingEffort">
										<font-awesome-icon :icon="['fas', opt.icon]" class="me-1" />
										{{ opt.label }}
									</button>
								</div>

								<div class="inspector-section-title mt-3">Autonomy Mode</div>
								<p class="inspector-hint">How much the Companion Supervisor acts on its own. Assist = comments only. Co-pilot = proposes actions for approval. Autopilot = runs safe actions automatically — destructive ones always ask.</p>
								<div class="effort-grid">
									<button v-for="opt in autonomyOptions" :key="opt.value" class="effort-btn" :class="{ active: (aiConfig?.autonomyMode || 'assist') === opt.value }" @click="setAutonomy(opt.value)" :disabled="savingEffort">
										<font-awesome-icon :icon="['fas', opt.icon]" class="me-1" />
										{{ opt.label }}
									</button>
								</div>

								<div class="inspector-section-title mt-3">Model</div>
								<div class="config-details">
									<div class="config-item">
										<span class="config-label">Name</span>
										<span class="config-value">{{ aiConfig?.modelName || 'Not loaded' }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Temperature</span>
										<span class="config-value">{{ aiConfig?.temperature ?? '—' }}</span>
									</div>
									<div class="config-item">
										<span class="config-label">Max Tokens</span>
										<span class="config-value">{{ aiConfig?.maxTokens || '—' }}</span>
									</div>
								</div>
								<router-link to="/settings" class="btn btn-sm btn-outline-secondary mt-2 w-100">
									<font-awesome-icon :icon="['fas', 'cog']" class="me-1" />
									Edit in Settings
								</router-link>

								<div class="inspector-section-title mt-3">Development</div>
								<button class="btn btn-sm btn-outline-warning w-100" @click="resetCompanion" :disabled="resetting">
									<font-awesome-icon :icon="['fas', resetting ? 'spinner' : 'undo']" :spin="resetting" class="me-1" />
									Reset to Defaults
								</button>
								<p class="inspector-hint mt-2">Clears chat + all memories, restores the default app memory. AI config (model, key, effort) is kept.</p>
							</div>

							<!-- Memory -->
							<div v-if="inspectorTab === 'memory'">
								<div class="d-flex justify-content-between align-items-center mb-2">
									<div class="inspector-section-title mb-0">Memories ({{ memories.length }})</div>
									<button class="btn btn-sm btn-outline-secondary" @click="loadMemories" :disabled="loadingMemories">
										<font-awesome-icon :icon="['fas', 'sync-alt']" :spin="loadingMemories" />
									</button>
								</div>
								<div v-if="loadingMemories" class="inspector-empty">Loading...</div>
								<div v-else-if="memories.length === 0" class="inspector-empty">No memories stored.</div>
								<div v-for="mem in sortedMemories" :key="mem.id" class="memory-card" :class="{ 'app-memory': mem.key === 'app_profile' }">
									<div class="memory-header">
										<span class="memory-key">
											<font-awesome-icon v-if="mem.key === 'app_profile'" :icon="['fas', 'robot']" class="me-1 text-info" />
											{{ mem.key }}
										</span>
										<div class="memory-meta">
											<span class="badge bg-secondary">{{ mem.category }}</span>
											<span class="badge" :class="mem.importance >= 8 ? 'bg-warning text-dark' : 'bg-dark'">★{{ mem.importance }}</span>
											<button class="btn btn-sm btn-link text-danger p-0 ms-1" title="Delete memory" @click="deleteMemory(mem.id)">
												<font-awesome-icon :icon="['fas', 'trash']" />
											</button>
										</div>
									</div>
									<p class="memory-value" :class="{ expanded: expandedMemoryId === mem.id }">{{ mem.value }}</p>
									<button v-if="mem.value.length > 160" class="btn btn-sm btn-link btn-link p-0" @click="expandedMemoryId = expandedMemoryId === mem.id ? null : mem.id">
										{{ expandedMemoryId === mem.id ? 'Show less' : 'Show more' }}
									</button>
								</div>
							</div>

							<!-- Roleplay (Bannerlord AIInfluence) -->
							<div v-if="inspectorTab === 'roleplay'">
								<div class="inspector-section-title">Watched NPCs</div>
								<p class="inspector-hint">The Companion reads NPC JSON files on change — it automatically knows each NPC's latest response, mood, and relation. No more copying.</p>

								<div class="input-group input-group-sm mb-2">
									<input v-model="newWatchPath" type="text" class="form-control bg-dark text-white border-secondary" placeholder="Paste full path to NPC .json file..." @keyup.enter="addWatch" />
									<button class="btn btn-outline-info" @click="addWatch" :disabled="addingWatch || !newWatchPath.trim()">
										<font-awesome-icon :icon="['fas', addingWatch ? 'spinner' : 'eye']" :spin="addingWatch" />
									</button>
								</div>
								<div v-if="watchError" class="inspector-hint text-danger">{{ watchError }}</div>

								<div v-if="roleplayWatches.length === 0 && !loadingRoleplay" class="inspector-empty">No NPC files watched yet. Paste a path like:</div>
								<div v-if="roleplayWatches.length === 0" class="inspector-empty" style="font-family: monospace; font-size: 0.68rem; word-break: break-all;">C:\SteamLibrary\...\Empress_Rhagaea (lord_1_14).json</div>

								<div v-for="w in roleplayWatches" :key="w.id" class="memory-card">
									<div class="memory-header">
										<span class="memory-key">
											<font-awesome-icon :icon="['fas', 'mask']" class="me-1 text-info" />
											{{ w.npc_name || 'Unknown NPC' }}
										</span>
										<button class="btn btn-sm btn-link text-danger p-0" title="Unwatch" @click="removeWatch(w.id)">
											<font-awesome-icon :icon="['fas', 'times']" />
										</button>
									</div>
									<div class="inspector-hint" style="margin-bottom: 0;">
										<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
										{{ w.last_snapshot_at ? 'Updated ' + formatRPTime(w.last_snapshot_at) : 'No updates yet' }}
									</div>
								</div>

								<!-- Latest NPC states -->
								<div class="inspector-section-title mt-3">Live NPC States</div>
								<div v-if="roleplaySnapshots.length === 0" class="inspector-empty">No snapshots yet — states appear after the game writes an NPC file.</div>
								<div v-for="snap in roleplaySnapshots" :key="snap.file_path" class="memory-card">
									<div class="memory-header">
										<span class="memory-key">{{ snap.npc_name }}</span>
										<div class="memory-meta">
											<span v-if="snap.state?.relation_desc" class="badge" :class="(snap.state.relation_value || 0) >= 20 ? 'bg-success' : ((snap.state.relation_value || 0) <= -20 ? 'bg-danger' : 'bg-secondary')">{{ snap.state.relation_desc }} {{ snap.state.relation_value }}</span>
											<span v-if="snap.state?.mood" class="badge bg-dark">{{ snap.state.mood }}</span>
										</div>
									</div>
									<div v-if="snap.state?.current_task" class="inspector-hint">
										<font-awesome-icon :icon="['fas', 'route']" class="me-1" />{{ snap.state.current_task }}
									</div>
									<p v-if="snap.state?.last_response" class="memory-value" :class="{ expanded: expandedRPId === snap.file_path }">{{ snap.state.last_response }}</p>
									<button v-if="snap.state?.last_response?.length > 160" class="btn btn-sm btn-link p-0" @click="expandedRPId = expandedRPId === snap.file_path ? null : snap.file_path">
										{{ expandedRPId === snap.file_path ? 'Show less' : 'Show more' }}
									</button>
									<div class="inspector-hint mt-1">
										<font-awesome-icon :icon="['fas', 'history']" class="me-1" />
										{{ snap.state?.recent_history?.length || 0 }} history entries · {{ formatRPTime(snap.created_at) }}
									</div>
								</div>
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

			<!-- Bannerlord Tab -->
			<div v-if="activeTab === 'bannerlord'">
				<BannerlordPanel />
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick, watch } from 'vue'
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
import BannerlordPanel from '@/components/BannerlordPanel.vue'

// Shared composables — same state as the floating widget
import { useChatHistory } from '@/composables/ai/useChatHistory'
import { useChatStreaming } from '@/composables/ai/useChatStreaming'

const { chatHistory, pushMessage, clearHistory, loadHistory } = useChatHistory()
const { isStreaming, streamMessage, stopStreaming, toolLabel } = useChatStreaming()

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

// ── Inspector state (right panel) ──
const inspectorTab = ref('session')
const lastRun = ref({
	tokens: { prompt: 0, completion: 0, reasoning: 0, total: 0 },
	requests: [],
	toolCalls: [],
})
const effortOptions = [
	{ value: 'none', label: 'None', icon: 'ban' },
	{ value: 'low', label: 'Low', icon: 'bolt' },
	{ value: 'medium', label: 'Medium', icon: 'brain' },
	{ value: 'high', label: 'High', icon: 'fire' },
]
const savingEffort = ref(false)
const resetting = ref(false)
const memories = ref([])
const loadingMemories = ref(false)
const expandedMemoryId = ref(null)

// ── Roleplay watcher state ──
const roleplayWatches = ref([])
const roleplaySnapshots = ref([])
const loadingRoleplay = ref(false)
const newWatchPath = ref('')
const addingWatch = ref(false)
const watchError = ref('')
const expandedRPId = ref(null)

const sortedMemories = computed(() => {
	return [...memories.value].sort((a, b) => {
		if (a.key === 'app_profile') return -1
		if (b.key === 'app_profile') return 1
		return (b.importance || 0) - (a.importance || 0)
	})
})

const emptyRun = () => ({
	tokens: { prompt: 0, completion: 0, reasoning: 0, total: 0 },
	requests: [],
	toolCalls: [],
})

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
			const body = configRes.value?.data || configRes.value || {}
			aiConfig.value = body.data && typeof body.data === 'object' ? body.data : body
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

	// Reset per-exchange inspector state
	lastRun.value = emptyRun()

	// Stream the response
	streamMessage(text, historyPayload, {
		onTextChunk: (chunk) => {
			chatHistory.value[msgIndex].content += chunk
		},
		onTokenUsage: (usage) => {
			lastRun.value.tokens = {
				prompt: usage.prompt_tokens || 0,
				completion: usage.completion_tokens || 0,
				reasoning: usage.reasoning_tokens || 0,
				total: usage.total_tokens || 0,
			}
		},
		onToolCall: (tool) => {
			lastRun.value.toolCalls.push({ name: tool, at: new Date().toLocaleTimeString() })
		},
		onRequestInfo: (info) => {
			lastRun.value.requests.push(info)
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

// ── Inspector methods ──

const loadMemories = async () => {
	loadingMemories.value = true
	try {
		const res = await aiAPI.getMemories({ limit: 50 })
		const body = res?.data || res || {}
		memories.value = body.data || []
	} catch {
		memories.value = []
	} finally {
		loadingMemories.value = false
	}
}

const deleteMemory = async (id) => {
	try {
		await aiAPI.deleteMemory(id)
		memories.value = memories.value.filter((m) => m.id !== id)
	} catch { /* ignore */ }
}

// ── Roleplay watcher methods ──

const loadRoleplay = async () => {
	loadingRoleplay.value = true
	try {
		const [wRes, sRes] = await Promise.allSettled([
			aiAPI.getRoleplayWatches(),
			aiAPI.getRoleplaySnapshots(),
		])
		if (wRes.status === 'fulfilled') {
			const body = wRes.value?.data || wRes.value || {}
			roleplayWatches.value = body.data || []
		}
		if (sRes.status === 'fulfilled') {
			const body = sRes.value?.data || sRes.value || {}
			roleplaySnapshots.value = body.data || []
		}
	} catch { /* ignore */ } finally {
		loadingRoleplay.value = false
	}
}

const addWatch = async () => {
	const path = newWatchPath.value.trim()
	if (!path || addingWatch.value) return
	addingWatch.value = true
	watchError.value = ''
	try {
		await aiAPI.addRoleplayWatch(path)
		newWatchPath.value = ''
		await loadRoleplay()
	} catch (err) {
		watchError.value = err?.response?.data?.message || err?.message || 'Failed to add watch'
	} finally {
		addingWatch.value = false
	}
}

const removeWatch = async (id) => {
	try {
		await aiAPI.removeRoleplayWatch(id)
		roleplayWatches.value = roleplayWatches.value.filter((w) => w.id !== id)
		await loadRoleplay()
	} catch { /* ignore */ }
}

const formatRPTime = (ts) => {
	if (!ts) return ''
	const d = new Date(ts)
	return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) + ' ' + d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

const setEffort = async (effort) => {
	if (!aiConfig.value || savingEffort.value) return
	const prev = aiConfig.value.reasoningEffort
	aiConfig.value.reasoningEffort = effort
	savingEffort.value = true
	try {
		await aiAPI.updateConfig({ ...aiConfig.value })
	} catch {
		aiConfig.value.reasoningEffort = prev
	} finally {
		savingEffort.value = false
	}
}

const autonomyOptions = [
	{ value: 'assist', label: 'Assist', icon: 'eye' },
	{ value: 'copilot', label: 'Co-pilot', icon: 'lightbulb' },
	{ value: 'autopilot', label: 'Autopilot', icon: 'robot' },
]

const setAutonomy = async (mode) => {
	if (!aiConfig.value || savingEffort.value) return
	const prev = aiConfig.value.autonomyMode
	aiConfig.value.autonomyMode = mode
	savingEffort.value = true
	try {
		await aiAPI.updateConfig({ ...aiConfig.value })
	} catch {
		aiConfig.value.autonomyMode = prev
	} finally {
		savingEffort.value = false
	}
}

const resetCompanion = async () => {
	const ok = confirm(
		'Reset companion to defaults?\n\n' +
		'• Chat messages will be deleted\n' +
		'• All memories except the default app memory will be deleted\n' +
		'• The default app memory will be restored\n\n' +
		'AI config (model, API key, effort) is NOT affected.'
	)
	if (!ok) return
	resetting.value = true
	try {
		await aiAPI.resetChat()
		chatHistory.value = []
		lastRun.value = emptyRun()
		await loadMemories()
	} catch { /* ignore */ } finally {
		resetting.value = false
	}
}

// Reload chat history when switching to chat tab
watch(activeTab, (newTab) => {
	if (newTab === 'chat') {
		loadHistory()
	}
})

// Load memories when the Memory sub-tab opens
watch(inspectorTab, (tab) => {
	if (tab === 'memory') {
		loadMemories()
	}
	if (tab === 'roleplay') {
		loadRoleplay()
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
	flex: 1;
	display: flex;
	flex-direction: column;
	min-height: 0;
	padding: 1.25rem;
}

.chat-section .chat-messages {
	flex: 1;
	height: auto;
	min-height: 0;
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

/* ── Two-panel chat layout ── */
.chat-layout {
	display: flex;
	gap: 1rem;
	align-items: stretch;
	height: calc(100vh - 270px);
	min-height: 520px;
}

.chat-layout .chat-section {
	flex: 1;
	min-width: 0;
	display: flex;
	flex-direction: column;
}

/* ── Inspector sidebar ── */
.inspector-panel {
	width: 380px;
	flex-shrink: 0;
	display: flex;
	flex-direction: column;
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	overflow: hidden;
}

.inspector-tabs {
	display: flex;
	background: rgba(0, 0, 0, 0.3);
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.inspector-tab {
	flex: 1;
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	padding: 0.75rem 0.5rem;
	font-weight: 600;
	font-size: 0.85rem;
	cursor: pointer;
	border-bottom: 2px solid transparent;
	transition: all 0.2s ease;
}

.inspector-tab:hover {
	color: rgba(255, 255, 255, 0.9);
}

.inspector-tab.active {
	color: #00d9ff;
	border-bottom-color: #00d9ff;
	background: rgba(0, 217, 255, 0.05);
}

.inspector-body {
	flex: 1;
	overflow-y: auto;
	padding: 1rem;
}

.inspector-section-title {
	font-size: 0.75rem;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: rgba(255, 255, 255, 0.5);
	margin-bottom: 0.5rem;
}

.inspector-hint {
	font-size: 0.78rem;
	color: rgba(255, 255, 255, 0.45);
	margin-bottom: 0.5rem;
}

.inspector-empty {
	font-size: 0.8rem;
	color: rgba(255, 255, 255, 0.35);
	padding: 0.5rem 0;
}

/* Token grid */
.token-grid {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 0.5rem;
}

.token-card {
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 8px;
	padding: 0.5rem 0.75rem;
	display: flex;
	flex-direction: column;
}

.token-card.reasoning {
	border-color: rgba(0, 217, 255, 0.3);
}

.token-card.total {
	border-color: rgba(255, 255, 255, 0.2);
}

.token-value {
	font-size: 1.15rem;
	font-weight: 700;
	color: #fff;
	font-variant-numeric: tabular-nums;
}

.token-card.reasoning .token-value {
	color: #00d9ff;
}

.token-label {
	font-size: 0.7rem;
	color: rgba(255, 255, 255, 0.5);
	text-transform: uppercase;
	letter-spacing: 0.04em;
}

/* Request cards */
.request-card {
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 8px;
	padding: 0.5rem 0.75rem;
	margin-bottom: 0.5rem;
}

.request-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 0.35rem;
}

.request-iter {
	font-weight: 700;
	color: #00d9ff;
	font-size: 0.85rem;
}

.request-meta {
	display: flex;
	flex-wrap: wrap;
	gap: 0.6rem;
	font-size: 0.72rem;
	color: rgba(255, 255, 255, 0.6);
}

/* Tool timeline */
.tool-row {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.3rem 0;
	border-bottom: 1px solid rgba(255, 255, 255, 0.05);
	font-size: 0.82rem;
}

.tool-name {
	flex: 1;
	color: rgba(255, 255, 255, 0.85);
}

.tool-time {
	color: rgba(255, 255, 255, 0.35);
	font-size: 0.7rem;
}

/* Effort selector */
.effort-grid {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 0.5rem;
}

.effort-btn {
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.12);
	color: rgba(255, 255, 255, 0.7);
	border-radius: 8px;
	padding: 0.55rem 0.75rem;
	font-size: 0.85rem;
	font-weight: 600;
	cursor: pointer;
	transition: all 0.2s ease;
}

.effort-btn:hover:not(:disabled) {
	border-color: rgba(0, 217, 255, 0.4);
	color: #fff;
}

.effort-btn.active {
	background: rgba(0, 217, 255, 0.12);
	border-color: #00d9ff;
	color: #00d9ff;
}

/* Memory cards */
.memory-card {
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 8px;
	padding: 0.6rem 0.75rem;
	margin-bottom: 0.5rem;
}

.memory-card.app-memory {
	border-color: rgba(0, 217, 255, 0.35);
	background: rgba(0, 217, 255, 0.05);
}

.memory-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 0.5rem;
	margin-bottom: 0.3rem;
}

.memory-key {
	font-weight: 700;
	font-size: 0.82rem;
	color: #fff;
	word-break: break-all;
}

.memory-meta {
	display: flex;
	align-items: center;
	gap: 0.3rem;
	flex-shrink: 0;
}

.memory-value {
	font-size: 0.78rem;
	color: rgba(255, 255, 255, 0.65);
	margin: 0;
	white-space: pre-wrap;
	word-break: break-word;
	display: -webkit-box;
	-webkit-line-clamp: 4;
	-webkit-box-orient: vertical;
	overflow: hidden;
}

.memory-value.expanded {
	-webkit-line-clamp: unset;
	overflow: visible;
}

/* Responsive: stack panels on narrow screens */
@media (max-width: 991.98px) {
	.chat-layout {
		flex-direction: column;
		height: auto;
	}
	.inspector-panel {
		width: 100%;
		max-height: 420px;
	}
	.chat-layout .chat-messages {
		height: 380px;
	}
}
</style>
