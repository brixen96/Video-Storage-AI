<template>
	<nav class="navbar navbar-expand-lg navbar-dark bg-dark sticky-top shadow-lg">
		<div class="container-fluid">
			<!-- Brand -->
			<router-link to="/" class="navbar-brand d-flex align-items-center">
				<font-awesome-icon :icon="['fas', 'video']" class="me-2 brand-icon" />
				<span class="brand-text">Video Storage AI</span>
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
				</ul>

				<!-- Right side items -->
				<div class="d-flex align-items-center gap-3">
					<!-- Activity Status Widget -->
					<div class="widget-container d-none d-lg-block">
						<ActivityStatusWidget />
					</div>

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
import websocketService from '@/services/websocket'

export default {
	name: 'NavBar',
	components: {
		ActivityStatusWidget,
	},
	data() {
		return {
			searchQuery: '',
			wsConnected: false,
			unsubscribeConnection: null,
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
	},
	beforeUnmount() {
		if (this.unsubscribeConnection) {
			this.unsubscribeConnection()
		}
	},
	methods: {
		handleSearch() {
			if (this.searchQuery.trim()) {
				this.$router.push({ path: '/videos', query: { search: this.searchQuery } })
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/navbar.css';
</style>
