<template>
	<nav class="navbar navbar-expand-lg navbar-dark bg-dark sticky-top shadow-lg">
		<div class="container-fluid">
			<!-- Brand -->
			<router-link to="/" class="navbar-brand d-flex align-items-center">
				<font-awesome-icon :icon="['fas', 'video']" class="me-2 brand-icon" />
				<span class="brand-text">Video Storage AI</span>
				<span v-if="appVersion" class="badge bg-secondary ms-2" style="font-size:0.6rem;vertical-align:middle;">v{{ appVersion }}</span>
				<!-- WebSocket Status Indicator -->
				<span class="ws-status-indicator ms-3" :class="{ connected: wsConnected, disconnected: !wsConnected }" :title="wsConnected ? 'WebSocket Connected' : 'WebSocket Disconnected'">
					<font-awesome-icon :icon="['fas', 'circle']" class="status-dot" />
				</span>
			</router-link>

			<!-- Mobile toggle button -->
			<button
				class="navbar-toggler"
				type="button"
				data-bs-toggle="collapse"
				data-bs-target="#navbarNav"
				aria-controls="navbarNav"
				aria-expanded="false"
				aria-label="Toggle navigation"
			>
				<span class="navbar-toggler-icon"></span>
			</button>

			<!-- Navigation links -->
			<div class="collapse navbar-collapse" id="navbarNav">
				<ul class="navbar-nav me-auto mb-2 mb-lg-0">
					<li class="nav-item">
						<router-link to="/" class="nav-link" exact-active-class="active">
							<font-awesome-icon :icon="['fas', 'home']" class="me-1" />
							Home
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/browser" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'folder-open']" class="me-1" />
							Browser
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/performers" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'users']" class="me-1" />
							Performers
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/videos" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
							Videos
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/libraries" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
							Libraries
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/studios" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'building']" class="me-1" />
							Studios
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/tags" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'tags']" class="me-1" />
							Tags
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/activity" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'chart-line']" class="me-1" />
							Activity
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/tasks" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'tasks']" class="me-1" />
							Tasks
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/edit-list" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'list-check']" class="me-1" />
							Edit List
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/scraper" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'spider']" class="me-1" />
							Scraper
						</router-link>
					</li>
						<li class="nav-item">
						<router-link to="/jdownloader" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
							JDownloader
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/ai" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'robot']" class="me-1" />
							AI
						</router-link>
					</li>
					<li class="nav-item">
						<router-link to="/analyzer" class="nav-link" active-class="active">
							<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-1" />
							Analyzer
						</router-link>
					</li>
				</ul>

				<!-- Right side items -->
				<div class="d-flex align-items-center gap-3">
					<!-- Global Search Trigger -->
					<button class="btn btn-sm btn-outline-secondary nav-search-btn d-none d-lg-flex align-items-center gap-2" @click="openGlobalSearch">
						<font-awesome-icon :icon="['fas', 'search']" />
						<kbd class="nav-search-kbd">Ctrl+K</kbd>
					</button>

					<!-- Activity Status Widget -->
					<div class="widget-container d-none d-lg-block">
						<ActivityStatusWidget />
					</div>

					<!-- Notifications -->
					<NotificationBell />

					<!-- System Health -->
					<router-link to="/system-health" class="btn btn-outline-light btn-sm">
						<font-awesome-icon :icon="['fas', 'heartbeat']" />
					</router-link>

					<!-- Settings -->
					<router-link to="/settings" class="btn btn-outline-light btn-sm">
						<font-awesome-icon :icon="['fas', 'cog']" />
					</router-link>
				</div>
			</div>
		</div>
	</nav>
</template>

<script>
import ActivityStatusWidget from './ActivityStatusWidget.vue'
import NotificationBell from './NotificationBell.vue'
import websocketService from '@/services/websocket'

export default {
	name: 'NavBar',
	components: {
		ActivityStatusWidget,
		NotificationBell,
	},
	data() {
		return {
			wsConnected: false,
			unsubscribeConnection: null,
			appVersion: '',
		}
	},
	mounted() {
		// Subscribe to WebSocket connection status
		this.unsubscribeConnection = websocketService.on('connected', (data) => {
			this.wsConnected = data.connected
		})

		// Check initial connection status
		this.wsConnected = websocketService.isConnected()

		// Connect WebSocket if not already connected
		if (!websocketService.isConnected()) {
			websocketService.connect()
		}

		// Use build-time version injected by web server (window.VSAI_VERSION)
		this.appVersion = window.VSAI_VERSION || 'dev'
	},
	beforeUnmount() {
		if (this.unsubscribeConnection) {
			this.unsubscribeConnection()
		}
	},
	methods: {
		openGlobalSearch() {
			document.dispatchEvent(new CustomEvent('open-global-search'))
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/navbar.css';
</style>
