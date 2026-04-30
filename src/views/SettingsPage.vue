<template>
	<div class="settings-page">
		<div class="container-fluid mt-3">
			<!-- Settings Navigation Tabs -->
			<ul class="nav nav-tabs mb-4" role="tablist">
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'general' }]" @click="activeTab = 'general'" type="button">
						<font-awesome-icon :icon="['fas', 'cog']" class="me-2" />
						General
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'library' }]" @click="activeTab = 'library'" type="button">
						<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
						Library
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'display' }]" @click="activeTab = 'display'" type="button">
						<font-awesome-icon :icon="['fas', 'eye']" class="me-2" />
						Display
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'privacy' }]" @click="activeTab = 'privacy'" type="button">
						<font-awesome-icon :icon="['fas', 'user']" class="me-2" />
						Privacy
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'backups' }]" @click="activeTab = 'backups'" type="button">
						<font-awesome-icon :icon="['fas', 'database']" class="me-2" />
						Backups
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'integrations' }]" @click="activeTab = 'integrations'" type="button">
						<font-awesome-icon :icon="['fas', 'plug']" class="me-2" />
						Integrations
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'server' }]" @click="activeTab = 'server'" type="button">
						<font-awesome-icon :icon="['fas', 'server']" class="me-2" />
						Server
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'about' }]" @click="activeTab = 'about'" type="button">
						<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
						About
					</button>
				</li>
			</ul>

			<!-- Tab Content -->
			<div class="tab-content">
				<!-- General Settings -->
				<div v-if="activeTab === 'general'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>Default View Mode</h4>
							<p>Choose the default view for browsing videos and performers</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.defaultViewMode" class="form-select">
								<option value="grid">Grid View</option>
								<option value="list">List View</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Theme</h4>
							<p>Select your preferred color theme</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.theme" class="form-select">
								<option value="dark">Dark Mode</option>
								<option value="light">Light Mode</option>
								<option value="auto">Auto (System Default)</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Items Per Page</h4>
							<p>Number of items to display per page</p>
						</div>
						<div class="setting-control">
							<input v-model.number="settings.itemsPerPage" type="number" class="form-control" min="10" max="100" step="10" />
						</div>
					</div>
				</div>

				<!-- Library Settings -->
				<div v-if="activeTab === 'library'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>Video Library Path</h4>
							<p>Default location for video files</p>
						</div>
						<div class="setting-control">
							<input v-model="settings.videoLibraryPath" type="text" class="form-control" placeholder="C:\Videos" />
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Auto-Scan on Startup</h4>
							<p>Automatically scan for new videos when the application starts</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.autoScanOnStartup" class="form-check-input" type="checkbox" id="autoScanSwitch" />
								<label class="form-check-label" for="autoScanSwitch">
									{{ settings.autoScanOnStartup ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Watch for Changes</h4>
							<p>Monitor library folders for new or deleted files</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.watchForChanges" class="form-check-input" type="checkbox" id="watchChangesSwitch" />
								<label class="form-check-label" for="watchChangesSwitch">
									{{ settings.watchForChanges ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Supported Video Formats</h4>
							<p>File extensions to include when scanning (comma-separated)</p>
						</div>
						<div class="setting-control">
							<input v-model="settings.videoFormats" type="text" class="form-control" placeholder="mp4,mkv,avi,mov,wmv" />
						</div>
					</div>
				</div>

				<!-- Display Settings -->
				<div v-if="activeTab === 'display'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>Thumbnail Quality</h4>
							<p>Quality of generated thumbnails (higher quality = larger file size)</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.thumbnailQuality" class="form-select">
								<option value="low">Low (Fast)</option>
								<option value="medium">Medium (Balanced)</option>
								<option value="high">High (Best Quality)</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Grid Card Size</h4>
							<p>Size of cards in grid view</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.gridCardSize" class="form-select">
								<option value="small">Small</option>
								<option value="medium">Medium</option>
								<option value="large">Large</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Show File Extensions</h4>
							<p>Display file extensions in video titles</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.showFileExtensions" class="form-check-input" type="checkbox" id="extensionsSwitch" />
								<label class="form-check-label" for="extensionsSwitch">
									{{ settings.showFileExtensions ? 'Show' : 'Hide' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Animated Previews</h4>
							<p>Show animated previews on hover (requires preview generation)</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.animatedPreviews" class="form-check-input" type="checkbox" id="previewsSwitch" />
								<label class="form-check-label" for="previewsSwitch">
									{{ settings.animatedPreviews ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>
				</div>

				<!-- Privacy Settings -->
				<div v-if="activeTab === 'privacy'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>
								<font-awesome-icon :icon="['fas', 'dog']" class="me-2 text-danger" />
								Default Zoo Filter
							</h4>
							<p>Default filter setting for zoo content when viewing performers</p>
						</div>
						<div class="setting-control">
							<select v-model="settings.defaultZooFilter" class="form-select">
								<option value="all">Show All</option>
								<option value="zoo-only">Zoo Only</option>
								<option value="non-zoo">Non-Zoo Only</option>
							</select>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Blur Thumbnails</h4>
							<p>Blur thumbnails until hovered over for privacy</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.blurThumbnails" class="form-check-input" type="checkbox" id="blurSwitch" />
								<label class="form-check-label" for="blurSwitch">
									{{ settings.blurThumbnails ? 'Enabled' : 'Disabled' }}
								</label>
							</div>
						</div>
					</div>

					<div class="setting-item">
						<div class="setting-info">
							<h4>Show Metadata</h4>
							<p>Display performer metadata (age, measurements, etc.)</p>
						</div>
						<div class="setting-control">
							<div class="form-check form-switch">
								<input v-model="settings.showMetadata" class="form-check-input" type="checkbox" id="metadataSwitch" />
								<label class="form-check-label" for="metadataSwitch">
									{{ settings.showMetadata ? 'Show' : 'Hide' }}
								</label>
							</div>
						</div>
					</div>
				</div>

				<!-- Backups -->
				<div v-if="activeTab === 'backups'" class="settings-section">
					<div class="setting-item">
						<div class="setting-info">
							<h4>Database Backups</h4>
							<p>Manage database backups, create restore points, and configure automatic backup schedules</p>
						</div>
						<div class="setting-control">
							<router-link to="/backups" class="btn btn-primary">
								<font-awesome-icon :icon="['fas', 'database']" />
								Manage Backups
							</router-link>
						</div>
					</div>
				</div>

				<!-- Integrations -->
				<div v-if="activeTab === 'integrations'" class="settings-section">
					<!-- JDownloader 2 Section -->
					<div class="card bg-dark mb-4">
						<div class="card-header d-flex justify-content-between align-items-center">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'download']" class="me-2 text-warning" />
								JDownloader 2
							</h5>
							<div class="d-flex align-items-center gap-2">
								<span :class="['badge', jdStatus.connected ? 'bg-success' : 'bg-secondary']">
									{{ jdStatus.connected ? 'Connected' : 'Disconnected' }}
								</span>
								<div class="form-check form-switch mb-0">
									<input v-model="jdConfig.enabled" class="form-check-input" type="checkbox" id="jdEnabledSwitch" @change="saveJDConfig" />
								</div>
							</div>
						</div>
						<div class="card-body">
							<div class="row g-3">
								<!-- Connection Type -->
								<div class="col-md-6">
									<label class="form-label">Connection Type</label>
									<select v-model="jdConfig.connectionType" class="form-select" @change="saveJDConfig">
										<option value="local">Local API (Docker/Same Network)</option>
										<option value="cloud" disabled>MyJDownloader Cloud (Coming Soon)</option>
									</select>
								</div>

								<!-- Host -->
								<div class="col-md-4">
									<label class="form-label">Host / IP Address</label>
									<input v-model="jdConfig.localHost" type="text" class="form-control" placeholder="192.168.1.10" />
								</div>

								<!-- Port -->
								<div class="col-md-2">
									<label class="form-label">API Port</label>
									<input v-model.number="jdConfig.localPort" type="number" class="form-control" placeholder="3129" />
								</div>

								<!-- Password -->
								<div class="col-md-6">
									<label class="form-label">Password (optional)</label>
									<div class="input-group">
										<input
											v-model="jdConfig.localPassword"
											:type="showJDPassword ? 'text' : 'password'"
											class="form-control"
											placeholder="Leave blank if not configured"
										/>
										<button class="btn btn-outline-secondary" type="button" @click="showJDPassword = !showJDPassword">
											<font-awesome-icon :icon="['fas', showJDPassword ? 'eye-slash' : 'eye']" />
										</button>
									</div>
								</div>

								<!-- Use Auth -->
								<div class="col-md-6 d-flex align-items-end">
									<div class="form-check">
										<input v-model="jdConfig.localUseAuth" class="form-check-input" type="checkbox" id="jdUseAuth" />
										<label class="form-check-label" for="jdUseAuth">Use Password Authentication</label>
									</div>
								</div>

								<!-- Default Package Name -->
								<div class="col-md-6">
									<label class="form-label">Default Package Name</label>
									<input v-model="jdConfig.defaultPackageName" type="text" class="form-control" placeholder="Leave blank for auto (uses URL host)" />
								</div>

								<!-- Auto-start Downloads -->
								<div class="col-md-6 d-flex align-items-end">
									<div class="form-check">
										<input v-model="jdConfig.autoStartDownloads" class="form-check-input" type="checkbox" id="jdAutoStart" />
										<label class="form-check-label" for="jdAutoStart">Auto-start downloads when adding links</label>
									</div>
								</div>
							</div>

							<!-- Actions -->
							<div class="d-flex gap-2 mt-4">
								<button class="btn btn-primary" @click="saveJDConfig" :disabled="savingJD">
									<font-awesome-icon :icon="['fas', savingJD ? 'spinner' : 'save']" :spin="savingJD" class="me-2" />
									Save Configuration
								</button>
								<button class="btn btn-outline-info" @click="testJDConnection" :disabled="testingJD">
									<font-awesome-icon :icon="['fas', testingJD ? 'spinner' : 'plug']" :spin="testingJD" class="me-2" />
									Test Connection
								</button>
							</div>

							<!-- Test Result -->
							<div v-if="jdTestResult" class="alert mt-3" :class="jdTestResult.success ? 'alert-success' : 'alert-danger'">
								<font-awesome-icon :icon="['fas', jdTestResult.success ? 'check-circle' : 'times-circle']" class="me-2" />
								<strong>{{ jdTestResult.message }}</strong>
								<p v-if="jdTestResult.errorDetails" class="mb-0 mt-2 small">{{ jdTestResult.errorDetails }}</p>
								<p v-if="jdTestResult.version" class="mb-0 mt-1 small text-muted">Version: {{ jdTestResult.version }}</p>
							</div>

							<!-- Info Box -->
							<div class="alert alert-info mt-3 mb-0">
								<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
								<strong>Setup Instructions:</strong>
								<ol class="mb-0 mt-2 small">
									<li>In JDownloader 2: Settings → Advanced Settings → search "RemoteAPI"</li>
									<li>Enable "Deprecated API" option</li>
									<li>If using Docker, ensure the API port (default 3129) is exposed</li>
									<li>Your Docker container is at: <code>192.168.1.10:5800</code> (Web UI) - API port is different</li>
								</ol>
							</div>
						</div>
					</div>

					<!-- LM Studio AI Settings Card -->
					<div class="card bg-dark mb-4">
						<div class="card-header d-flex justify-content-between align-items-center">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'robot']" class="me-2 text-info" />
								OpenRouter (AI Companion)
							</h5>
							<div class="d-flex align-items-center gap-2">
								<span :class="['badge', aiConfigStatus.connected ? 'bg-success' : 'bg-secondary']">
									{{ aiConfigStatus.connected ? 'Connected' : 'Disconnected' }}
								</span>
								<div class="form-check form-switch mb-0">
									<input v-model="aiConfig.enabled" class="form-check-input" type="checkbox" id="aiEnabledSwitch" @change="saveAIConfig" />
								</div>
							</div>
						</div>
						<div class="card-body">
							<div class="row g-3 mb-3">
								<div class="col-md-6">
									<label class="form-label text-muted small">API URL</label>
									<input v-model="aiConfig.apiUrl" type="text" class="form-control bg-dark text-white border-secondary" placeholder="https://openrouter.ai/api" />
								</div>
								<div class="col-md-6">
									<label class="form-label text-muted small">API Key</label>
									<input v-model="aiConfig.apiKey" type="password" class="form-control bg-dark text-white border-secondary" placeholder="sk-or-..." autocomplete="new-password" />
								</div>
							</div>
							<div class="row g-3 mb-3">
								<div class="col-md-4">
									<label class="form-label text-muted small">Model</label>
									<input v-model="aiConfig.modelName" type="text" class="form-control bg-dark text-white border-secondary" placeholder="mistralai/mistral-nemo" />
								</div>
								<div class="col-md-4">
									<label class="form-label text-muted small">Max Tokens</label>
									<input v-model.number="aiConfig.maxTokens" type="number" class="form-control bg-dark text-white border-secondary" min="100" max="8192" />
								</div>
								<div class="col-md-4">
									<label class="form-label text-muted small">Temperature ({{ aiConfig.temperature }})</label>
									<input v-model.number="aiConfig.temperature" type="range" class="form-range" min="0" max="1" step="0.1" />
								</div>
							</div>
							<div class="d-flex gap-2">
								<button class="btn btn-outline-info btn-sm" @click="testAIConnection" :disabled="aiTestingConnection">
									<font-awesome-icon :icon="['fas', aiTestingConnection ? 'spinner' : 'plug']" :spin="aiTestingConnection" class="me-1" />
									Test Connection
								</button>
								<button class="btn btn-primary btn-sm" @click="saveAIConfig" :disabled="aiSavingConfig">
									<font-awesome-icon :icon="['fas', 'save']" class="me-1" />
									Save
								</button>
							</div>
							<div v-if="aiConfigStatus.message" class="mt-2">
								<small :class="aiConfigStatus.connected ? 'text-success' : 'text-warning'">
									{{ aiConfigStatus.message }}
								</small>
							</div>
							<div v-if="aiConfigStatus.models && aiConfigStatus.models.length > 0" class="mt-2">
								<small class="text-muted">Available models: {{ aiConfigStatus.models.join(', ') }}</small>
							</div>
						</div>
					</div>
				</div>

				<!-- About -->
				<!-- Server Settings -->
				<div v-if="activeTab === 'server'" class="settings-section">

					<!-- Service Control Panel -->
					<div class="setting-group mb-4">
						<div class="d-flex align-items-center gap-3 mb-3">
							<h4 class="mb-0">
								<font-awesome-icon :icon="['fas', 'server']" class="me-2" />
								Service Control
							</h4>
							<button class="btn btn-sm btn-outline-secondary" :disabled="loadingServices" @click="loadServiceStatus">
								<font-awesome-icon :icon="['fas', 'sync-alt']" :spin="loadingServices" class="me-1" />
								Refresh
							</button>
						</div>
						<div class="service-cards d-flex gap-3 mb-3 flex-wrap">
							<div v-for="(svc, key) in services" :key="key" class="service-card card p-3" style="min-width:220px;flex:1">
								<div class="d-flex align-items-center justify-content-between mb-1">
									<strong>{{ key === 'api' ? 'API Server' : 'Web Server' }}</strong>
									<span class="badge" :class="serviceStatusClass(svc)">{{ serviceStatusLabel(svc) }}</span>
								</div>
								<small class="text-muted d-block mb-2">{{ svc ? svc.name : '...' }}</small>
								<div class="d-flex gap-2 align-items-center">
									<button
										class="btn btn-sm btn-success"
										title="Start"
										:disabled="serviceActionPending !== null || !svc || svc.status === 'running' || !svc.installed"
										@click="doServiceAction(key, 'start')"
									>
										<font-awesome-icon :icon="['fas', 'play']" />
									</button>
									<button
										class="btn btn-sm btn-warning"
										title="Restart"
										:disabled="serviceActionPending !== null || !svc || svc.status !== 'running'"
										@click="doServiceAction(key, 'restart')"
									>
										<font-awesome-icon :icon="['fas', 'redo']" />
									</button>
									<button
										class="btn btn-sm btn-danger"
										title="Stop"
										:disabled="serviceActionPending !== null || !svc || svc.status !== 'running'"
										@click="doServiceAction(key, 'stop')"
									>
										<font-awesome-icon :icon="['fas', 'stop']" />
									</button>
								</div>
							</div>
						</div>
						<div class="d-flex gap-2 mb-2">
							<button
								class="btn btn-sm btn-outline-warning"
								:disabled="serviceActionPending !== null"
								@click="doServiceAction('both', 'restart')"
							>
								<font-awesome-icon :icon="['fas', 'redo']" class="me-1" />
								Restart All
							</button>
							<button
								class="btn btn-sm btn-outline-danger"
								:disabled="serviceActionPending !== null"
								@click="doServiceAction('both', 'stop')"
							>
								<font-awesome-icon :icon="['fas', 'stop']" class="me-1" />
								Stop All
							</button>
						</div>
						<p class="text-muted small mb-0">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-1" />
							Services must be installed as Windows services for these controls to work. Run <code>install-services.bat</code> as Administrator.
						</p>
					</div>

					<!-- Live Log Viewer -->
					<div class="setting-group mb-4">
						<div class="d-flex align-items-center justify-content-between mb-2">
							<h4 class="mb-0">
								<font-awesome-icon :icon="['fas', 'terminal']" class="me-2" />
								Recent Logs
							</h4>
							<button class="btn btn-sm btn-outline-secondary" :disabled="loadingLogs" @click="loadConsoleLogs">
								<font-awesome-icon :icon="['fas', 'sync-alt']" :spin="loadingLogs" />
							</button>
						</div>
						<div class="log-viewer" ref="logViewer">
							<div v-if="consoleLogs.length === 0 && !loadingLogs" class="text-muted small p-2">No logs found.</div>
							<div v-for="log in consoleLogs" :key="log.id" class="log-entry" :class="'log-level-' + log.level">
								<span class="log-time">{{ formatLogTime(log.created_at) }}</span>
								<span class="log-badge" :class="'log-badge-' + log.level">{{ log.level }}</span>
								<span class="log-source">{{ log.source }}</span>
								<span class="log-msg">{{ log.message }}</span>
							</div>
						</div>
					</div>

					<hr />

					<div class="setting-item">
						<div class="setting-info">
							<h4>API Port</h4>
							<p>The port the backend API server listens on. Default: 8080.</p>
						</div>
						<div class="setting-control">
							<input
								v-model="appConfig.server_port"
								type="number"
								class="form-control"
								min="1024"
								max="65535"
								style="width: 120px"
							/>
						</div>
					</div>
					<div class="setting-item">
						<div class="setting-info">
							<h4>Web Port</h4>
							<p>The port the frontend web server listens on. Default: 3000.</p>
						</div>
						<div class="setting-control">
							<input
								v-model="appConfig.web_port"
								type="number"
								class="form-control"
								min="1024"
								max="65535"
								style="width: 120px"
							/>
						</div>
					</div>
					<p class="text-warning small mt-2">
						<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-1" />
						A restart of both services is required for port changes to take effect.
					</p>
					<div class="settings-footer">
						<button class="btn btn-primary" :disabled="savingAppConfig" @click="saveAppConfig">
							<font-awesome-icon v-if="savingAppConfig" :icon="['fas', 'spinner']" spin class="me-2" />
							Save &amp; Restart Required
						</button>
					</div>
				</div>

				<div v-if="activeTab === 'about'" class="settings-section">
					<div class="about-content">
						<div class="app-info">
							<h3>
								<font-awesome-icon :icon="['fas', 'video']" class="me-3 text-primary" />
								Video Storage AI
							</h3>
							<p class="version">Version {{ appVersion || '...' }}</p>
							<p class="description">
								A powerful video library management system with AI-powered organization, metadata management, and intelligent search capabilities.
							</p>
						</div>

						<div class="features-list">
							<h4>Features</h4>
							<ul>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Automatic video cataloging and organization
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Performer and studio database management
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Advanced filtering and search capabilities
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Metadata integration and enrichment
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Tag-based organization system
								</li>
								<li>
									<font-awesome-icon :icon="['fas', 'check-circle']" class="text-success me-2" />
									Thumbnail generation and preview support
								</li>
							</ul>
						</div>

						<div class="tech-stack">
							<h4>Technology</h4>
							<div class="tech-badges">
								<span class="badge bg-primary">Vue.js 3</span>
								<span class="badge bg-success">Go + Gin</span>
								<span class="badge bg-info">SQLite</span>
								<span class="badge bg-warning text-dark">FFmpeg</span>
							</div>
						</div>

						<!-- Update checker -->
						<div class="update-section mt-4">
							<h4>Updates</h4>
							<div v-if="!updateInfo" class="d-flex align-items-center gap-3">
								<button class="btn btn-outline-primary" @click="checkForUpdate" :disabled="checkingUpdate">
									<font-awesome-icon :icon="['fas', 'sync-alt']" :spin="checkingUpdate" class="me-2" />
									{{ checkingUpdate ? 'Checking...' : 'Check for Updates' }}
								</button>
							</div>
							<div v-else class="update-result">
								<div v-if="updateInfo.update_available" class="alert alert-success d-flex align-items-center gap-3 mb-2">
									<font-awesome-icon :icon="['fas', 'arrow-up']" />
									<span>
										Update available: <strong>v{{ updateInfo.latest_version }}</strong>
										<span v-if="updateInfo.published_at" class="text-muted ms-2 small">
											({{ new Date(updateInfo.published_at).toLocaleDateString() }})
										</span>
									</span>
									<button class="btn btn-success btn-sm ms-auto" @click="installUpdate" :disabled="installingUpdate">
										<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
										{{ installingUpdate ? 'Downloading...' : 'Install Update' }}
									</button>
								</div>
								<div v-else class="alert alert-info d-flex align-items-center gap-2 mb-2">
									<font-awesome-icon :icon="['fas', 'check-circle']" />
									You're on the latest version (v{{ updateInfo.current_version }}).
								</div>
								<div v-if="updateInfo.release_notes" class="release-notes small text-muted border rounded p-2 mb-2" style="max-height:120px;overflow-y:auto;white-space:pre-wrap;">{{ updateInfo.release_notes }}</div>
								<button class="btn btn-outline-secondary btn-sm" @click="updateInfo = null">
									<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
									Dismiss
								</button>
							</div>
						</div>

						<div class="links-section mt-4">
							<h4>Links</h4>
							<div class="link-buttons">
								<button class="btn btn-outline-primary">
									<font-awesome-icon :icon="['fas', 'book']" class="me-2" />
									Documentation
								</button>
								<button class="btn btn-outline-secondary">
									<font-awesome-icon :icon="['fas', 'code-branch']" class="me-2" />
									GitHub
								</button>
								<button class="btn btn-outline-info">
									<font-awesome-icon :icon="['fas', 'question-circle']" class="me-2" />
									Support
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Save Button -->
			<div v-if="activeTab !== 'about'" class="settings-footer">
				<button class="btn btn-primary btn-lg" @click="saveSettings">
					<font-awesome-icon :icon="['fas', 'save']" class="me-2" />
					Save Settings
				</button>
				<button class="btn btn-secondary btn-lg ms-3" @click="resetSettings">
					<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
					Reset to Defaults
				</button>
			</div>
		</div>
	</div>
</template>

<script>
import settingsService from '@/services/settingsService'
import { jdownloaderAPI, aiAPI, appConfigAPI, systemAPI, consoleLogAPI } from '@/services/api'
import websocketService from '@/services/websocket'

export default {
	name: 'SettingsPage',
	data() {
		return {
			activeTab: 'general',
			settings: settingsService.getSettings(),
			// JDownloader settings
			jdConfig: {
				connectionType: 'local',
				localHost: '192.168.1.10',
				localPort: 3129,
				localPassword: '',
				localUseAuth: false,
				defaultPackageName: '',
				autoStartDownloads: false,
				enabled: true,
			},
			jdStatus: {
				connected: false,
				type: 'local',
			},
			jdTestResult: null,
			showJDPassword: false,
			testingJD: false,
			savingJD: false,
			// AI / OpenRouter settings
			aiConfig: {
				apiUrl: 'https://openrouter.ai/api',
				apiKey: '',
				modelName: 'mistralai/mistral-nemo',
				temperature: 0.7,
				maxTokens: 2000,
				enabled: true,
			},
			aiConfigStatus: {
				connected: false,
				message: '',
				models: [],
			},
			aiTestingConnection: false,
			aiSavingConfig: false,
			// Server / app config
			appConfig: { server_port: '8080' },
			savingAppConfig: false,
			appVersion: null,
			// Update checker
			checkingUpdate: false,
			installingUpdate: false,
			updateInfo: null,
			// Service control
			services: { api: null, web: null },
			loadingServices: false,
			serviceActionPending: null,
			// Log viewer
			consoleLogs: [],
			loadingLogs: false,
			wsUnsubLog: null,
		}
	},
	watch: {
		// Watch for theme changes and apply immediately
		'settings.theme'(newTheme) {
			settingsService.updateSetting('theme', newTheme)
			settingsService.applyTheme()
		},
		// Load configs when integrations tab is active
		activeTab(newTab) {
			if (newTab === 'integrations') {
				this.loadJDConfig()
				this.loadAIConfig()
			}
			if (newTab === 'server') {
				this.loadAppConfig()
				this.loadServiceStatus()
				this.loadConsoleLogs()
			}
			if (newTab === 'about') {
				this.loadVersion()
			}
		},
	},
	mounted() {
		// Load JD config if starting on integrations tab
		if (this.activeTab === 'integrations') {
			this.loadJDConfig()
		}
		this.loadVersion()
		// Subscribe to live log events
		this.wsUnsubLog = websocketService.on('console_log', (data) => {
			if (data) {
				this.consoleLogs.unshift(data)
				if (this.consoleLogs.length > 100) {
					this.consoleLogs.splice(100)
				}
			}
		})
	},
	beforeUnmount() {
		if (this.wsUnsubLog) {
			this.wsUnsubLog()
			this.wsUnsubLog = null
		}
	},
	methods: {
		saveSettings() {
			const success = settingsService.saveSettings(this.settings)
			if (success) {
				settingsService.applyTheme()
				this.$toast.success('Settings Saved', 'Your preferences have been updated')
			} else {
				this.$toast.error('Save Failed', 'Could not save settings')
			}
		},
		resetSettings() {
			if (confirm('Are you sure you want to reset all settings to their default values?')) {
				settingsService.resetSettings()
				this.settings = settingsService.getSettings()
				settingsService.applyTheme()
				this.$toast.info('Settings Reset', 'All settings have been restored to defaults')
			}
		},
		// JDownloader methods
		async loadJDConfig() {
			try {
				const config = await jdownloaderAPI.getConfig()
				this.jdConfig = {
					connectionType: config.connectionType || 'local',
					localHost: config.localHost || '192.168.1.10',
					localPort: config.localPort || 3129,
					localPassword: config.localPassword || '',
					localUseAuth: config.localUseAuth || false,
					defaultPackageName: config.defaultPackageName || '',
					autoStartDownloads: config.autoStartDownloads || false,
					enabled: config.enabled !== false,
				}
				// Also check status
				this.checkJDStatus()
			} catch (error) {
				console.error('Failed to load JDownloader config:', error)
			}
		},
		async checkJDStatus() {
			try {
				const status = await jdownloaderAPI.getStatus()
				this.jdStatus = {
					connected: status.connected || false,
					type: status.type || 'local',
				}
			} catch (error) {
				this.jdStatus = { connected: false, type: 'local' }
			}
		},
		async saveJDConfig() {
			this.savingJD = true
			try {
				await jdownloaderAPI.updateConfig(this.jdConfig)
				this.$toast.success('JDownloader configuration saved')
				this.checkJDStatus()
			} catch (error) {
				console.error('Failed to save JDownloader config:', error)
				this.$toast.error('Failed to save configuration')
			} finally {
				this.savingJD = false
			}
		},
		async testJDConnection() {
			this.testingJD = true
			this.jdTestResult = null
			try {
				// Save config first
				await jdownloaderAPI.updateConfig(this.jdConfig)
				// Then test
				const result = await jdownloaderAPI.testConnection()
				this.jdTestResult = result
				if (result.success) {
					this.jdStatus.connected = true
					this.$toast.success('Connection successful!')
				}
			} catch (error) {
				console.error('Failed to test JDownloader connection:', error)
				this.jdTestResult = {
					success: false,
					message: 'Connection test failed',
					errorDetails: error.message || 'Unknown error',
				}
			} finally {
				this.testingJD = false
			}
		},
		// AI / OpenRouter methods
		async loadAIConfig() {
			try {
				const response = await aiAPI.getConfig()
				const data = response.data?.data || response.data || {}
				this.aiConfig = {
					apiUrl: data.apiUrl || 'https://openrouter.ai/api',
					apiKey: data.apiKey || '',
					modelName: data.modelName || 'mistralai/mistral-nemo',
					temperature: data.temperature ?? 0.7,
					maxTokens: data.maxTokens || 2000,
					enabled: data.enabled !== false,
				}
			} catch (error) {
				console.error('Failed to load AI config:', error)
			}
		},
		async saveAIConfig() {
			this.aiSavingConfig = true
			try {
				await aiAPI.updateConfig(this.aiConfig)
				this.$toast.success('AI config saved!')
			} catch (error) {
				console.error('Failed to save AI config:', error)
				this.$toast.error('Failed to save AI configuration')
			} finally {
				this.aiSavingConfig = false
			}
		},
		async loadVersion() {
			try {
				const res = await systemAPI.getHealth()
				this.appVersion = res?.data?.version || null
			} catch {
				// non-critical, leave as null
			}
		},
		async checkForUpdate() {
			this.checkingUpdate = true
			this.updateInfo = null
			try {
				const res = await systemAPI.checkUpdate()
				this.updateInfo = res.data?.data || null
			} catch (error) {
				this.$toast.error('Update check failed', error?.response?.data?.message || error.message)
			} finally {
				this.checkingUpdate = false
			}
		},
		async installUpdate() {
			if (!this.updateInfo?.download_url) {
				this.$toast.error('No download URL available')
				return
			}
			this.installingUpdate = true
			try {
				await systemAPI.installUpdate(this.updateInfo.download_url)
				this.$toast.success('Update installed', 'The service will restart momentarily.')
			} catch (error) {
				this.$toast.error('Update failed', error?.response?.data?.message || error.message)
			} finally {
				this.installingUpdate = false
			}
		},
		// Server / app config methods
		async loadAppConfig() {
			try {
				const response = await appConfigAPI.get()
				this.appConfig = response.data?.data || { server_port: '8080' }
			} catch (error) {
				console.error('Failed to load app config:', error)
			}
		},
		async saveAppConfig() {
			this.savingAppConfig = true
			try {
				await appConfigAPI.update(this.appConfig)
				this.$toast.success('Server config saved', 'Restart the app for the new port to take effect')
			} catch (error) {
				console.error('Failed to save app config:', error)
				this.$toast.error('Failed to save server config')
			} finally {
				this.savingAppConfig = false
			}
		},
		async testAIConnection() {
			this.aiTestingConnection = true
			try {
				// Save config first
				await aiAPI.updateConfig(this.aiConfig)
				const response = await aiAPI.testConnection()
				const data = response.data?.data || response.data || {}
				this.aiConfigStatus = {
					connected: data.connected || false,
					message: data.message || (data.connected ? 'Connected!' : 'Not connected'),
					models: data.models || [],
				}
				if (data.connected) {
					this.$toast.success('LM Studio connected!')
				} else {
					this.$toast.warning(data.error || 'Could not connect to LM Studio')
				}
			} catch (error) {
				console.error('Failed to test AI connection:', error)
				this.aiConfigStatus = {
					connected: false,
					message: 'Connection test failed: ' + (error.message || 'Unknown error'),
					models: [],
				}
			} finally {
				this.aiTestingConnection = false
			}
		},
		// Service control methods
		async loadServiceStatus() {
			this.loadingServices = true
			try {
				const res = await systemAPI.getServices()
				const data = res.data?.data || {}
				this.services = {
					api: data.api || null,
					web: data.web || null,
				}
			} catch (error) {
				console.error('Failed to load service status:', error)
			} finally {
				this.loadingServices = false
			}
		},
		async doServiceAction(service, action) {
			this.serviceActionPending = service
			try {
				await systemAPI.controlService(service, action)
				this.$toast.success(`${action} ${service} initiated`)
				// Refresh status after the service has had time to change state
				setTimeout(() => this.loadServiceStatus(), 3000)
			} catch (error) {
				this.$toast.error(`Failed to ${action} ${service}`, error?.response?.data?.message || error.message)
			} finally {
				this.serviceActionPending = null
			}
		},
		async loadConsoleLogs() {
			this.loadingLogs = true
			try {
				const res = await consoleLogAPI.getAll({ limit: 50, page: 1 })
				const data = res.data?.data || res.data || []
				this.consoleLogs = Array.isArray(data) ? data : (data.logs || data.items || [])
			} catch (error) {
				console.error('Failed to load console logs:', error)
			} finally {
				this.loadingLogs = false
			}
		},
		serviceStatusClass(svc) {
			if (!svc || !svc.installed) return 'bg-secondary'
			switch (svc.status) {
				case 'running': return 'bg-success'
				case 'starting': return 'bg-info'
				case 'stopping': return 'bg-warning text-dark'
				case 'stopped': return 'bg-danger'
				default: return 'bg-secondary'
			}
		},
		serviceStatusLabel(svc) {
			if (!svc) return '...'
			if (!svc.installed) return 'Not Installed'
			switch (svc.status) {
				case 'running': return 'Running'
				case 'starting': return 'Starting...'
				case 'stopping': return 'Stopping...'
				case 'stopped': return 'Stopped'
				default: return svc.status || 'Unknown'
			}
		},
		formatLogTime(ts) {
			if (!ts) return ''
			try {
				return new Date(ts).toLocaleTimeString()
			} catch {
				return ts
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/settings_page.css';
</style>
