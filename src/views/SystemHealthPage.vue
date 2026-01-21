<template>
	<div class="system-health-page">
		<div class="page-header">
			<h1>
				<font-awesome-icon :icon="['fas', 'heartbeat']" />
				System Health
			</h1>
			<div class="header-actions">
				<div class="last-updated">
					Last updated: {{ formatTime(lastUpdated) }}
				</div>
				<button class="btn btn-sm btn-primary" @click="refreshHealth" :disabled="loading">
					<font-awesome-icon :icon="['fas', loading ? 'spinner' : 'sync-alt']" :spin="loading" />
					Refresh
				</button>
			</div>
		</div>

		<div v-if="loading && !health" class="loading-container">
			<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" />
			<p>Loading system health...</p>
		</div>

		<div v-else-if="health" class="health-content">
			<!-- Overall Status Banner -->
			<div class="status-banner" :class="`status-${overallStatus}`">
				<div class="status-icon">
					<font-awesome-icon :icon="getStatusIcon(overallStatus)" size="2x" />
				</div>
				<div class="status-info">
					<h2>System Status: {{ overallStatus.toUpperCase() }}</h2>
					<p>All systems {{ overallStatus === 'healthy' ? 'operational' : 'experiencing issues' }}</p>
				</div>
			</div>

			<!-- Service Health Cards -->
			<div class="section">
				<h3 class="section-title">
					<font-awesome-icon :icon="['fas', 'server']" />
					Services
				</h3>
				<div class="service-grid">
					<div v-for="(service, name) in health.services" :key="name" class="service-card" :class="`status-${service.status}`">
						<div class="service-header">
							<font-awesome-icon :icon="getServiceIcon(name)" class="service-icon" />
							<h4>{{ formatServiceName(name) }}</h4>
						</div>
						<div class="service-status">
							<span class="status-badge" :class="`badge-${service.status}`">
								{{ service.status }}
							</span>
						</div>
						<div v-if="service.version" class="service-version">
							v{{ service.version }}
						</div>
					</div>
				</div>
			</div>

			<!-- Database Health -->
			<div class="section">
				<h3 class="section-title">
					<font-awesome-icon :icon="['fas', 'database']" />
					Database
				</h3>
				<div class="db-grid">
					<div class="metric-card">
						<div class="metric-icon">
							<font-awesome-icon :icon="['fas', 'check-circle']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">Status</div>
							<div class="metric-value">{{ health.database.status }}</div>
						</div>
					</div>
					<div class="metric-card">
						<div class="metric-icon">
							<font-awesome-icon :icon="['fas', 'tachometer-alt']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">Ping</div>
							<div class="metric-value">{{ health.database.ping_ms }}ms</div>
						</div>
					</div>
					<div class="metric-card">
						<div class="metric-icon">
							<font-awesome-icon :icon="['fas', 'hdd']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">Size</div>
							<div class="metric-value">{{ health.database.size_mb?.toFixed(2) }} MB</div>
						</div>
					</div>
					<div class="metric-card">
						<div class="metric-icon">
							<font-awesome-icon :icon="['fas', 'link']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">Connections</div>
							<div class="metric-value">{{ health.database.open_connections }}</div>
						</div>
					</div>
				</div>

				<!-- Table Counts -->
				<div class="table-counts">
					<h4>Database Tables</h4>
					<div class="table-grid">
						<div v-for="(count, table) in health.database.table_counts" :key="table" class="table-item">
							<span class="table-name">{{ formatTableName(table) }}</span>
							<span class="table-count">{{ count.toLocaleString() }}</span>
						</div>
					</div>
				</div>
			</div>

			<!-- System Resources -->
			<div class="section">
				<h3 class="section-title">
					<font-awesome-icon :icon="['fas', 'microchip']" />
					System Resources
				</h3>
				<div class="resources-grid">
					<div class="metric-card">
						<div class="metric-icon memory">
							<font-awesome-icon :icon="['fas', 'memory']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">Memory (Allocated)</div>
							<div class="metric-value">{{ health.system.memory.alloc_mb?.toFixed(2) }} MB</div>
						</div>
					</div>
					<div class="metric-card">
						<div class="metric-icon cpu">
							<font-awesome-icon :icon="['fas', 'microchip']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">Goroutines</div>
							<div class="metric-value">{{ health.system.num_goroutines }}</div>
						</div>
					</div>
					<div class="metric-card">
						<div class="metric-icon cpu">
							<font-awesome-icon :icon="['fas', 'server']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">CPU Cores</div>
							<div class="metric-value">{{ health.system.num_cpu }}</div>
						</div>
					</div>
					<div class="metric-card">
						<div class="metric-icon gc">
							<font-awesome-icon :icon="['fas', 'broom']" />
						</div>
						<div class="metric-content">
							<div class="metric-label">GC Cycles</div>
							<div class="metric-value">{{ health.system.memory.num_gc }}</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Application Statistics -->
			<div v-if="health.application" class="section">
				<h3 class="section-title">
					<font-awesome-icon :icon="['fas', 'chart-bar']" />
					Application Statistics
				</h3>
				<div class="app-stats-grid">
					<div class="stat-card">
						<div class="stat-icon videos">
							<font-awesome-icon :icon="['fas', 'video']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ health.application.total_videos?.toLocaleString() }}</div>
							<div class="stat-label">Videos</div>
							<div class="stat-sub">{{ health.application.total_video_size_gb?.toFixed(2) }} GB</div>
						</div>
					</div>
					<div class="stat-card">
						<div class="stat-icon performers">
							<font-awesome-icon :icon="['fas', 'users']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ health.application.total_performers?.toLocaleString() }}</div>
							<div class="stat-label">Performers</div>
						</div>
					</div>
					<div class="stat-card">
						<div class="stat-icon studios">
							<font-awesome-icon :icon="['fas', 'building']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ health.application.total_studios?.toLocaleString() }}</div>
							<div class="stat-label">Studios</div>
						</div>
					</div>
					<div class="stat-card">
						<div class="stat-icon tags">
							<font-awesome-icon :icon="['fas', 'tags']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ health.application.total_tags?.toLocaleString() }}</div>
							<div class="stat-label">Tags</div>
						</div>
					</div>
				</div>

				<!-- Scraper & Downloads -->
				<div class="secondary-stats">
					<div class="secondary-stat-card">
						<h4>
							<font-awesome-icon :icon="['fas', 'spider']" />
							Scraper
						</h4>
						<div class="secondary-stat-row">
							<span>Total Threads:</span>
							<strong>{{ health.application.scraper?.total_threads?.toLocaleString() }}</strong>
						</div>
						<div class="secondary-stat-row">
							<span>Active:</span>
							<strong class="text-success">{{ health.application.scraper?.active_threads?.toLocaleString() }}</strong>
						</div>
					</div>

					<div class="secondary-stat-card">
						<h4>
							<font-awesome-icon :icon="['fas', 'link']" />
							Download Links
						</h4>
						<div class="secondary-stat-row">
							<span>Total:</span>
							<strong>{{ health.application.download_links?.total?.toLocaleString() }}</strong>
						</div>
						<div class="secondary-stat-row">
							<span>Active:</span>
							<strong class="text-success">{{ health.application.download_links?.active?.toLocaleString() }}</strong>
						</div>
						<div class="secondary-stat-row">
							<span>Downloaded:</span>
							<strong class="text-info">{{ health.application.download_links?.downloaded?.toLocaleString() }}</strong>
						</div>
						<div class="secondary-stat-row">
							<span>Dead:</span>
							<strong class="text-danger">{{ health.application.download_links?.dead?.toLocaleString() }}</strong>
						</div>
					</div>

					<div class="secondary-stat-card">
						<h4>
							<font-awesome-icon :icon="['fas', 'brain']" />
							AI Usage (24h)
						</h4>
						<div class="secondary-stat-row">
							<span>Interactions:</span>
							<strong>{{ health.application.ai_usage_24h?.interactions?.toLocaleString() }}</strong>
						</div>
						<div class="secondary-stat-row">
							<span>Cost:</span>
							<strong class="text-warning">${{ health.application.ai_usage_24h?.cost_usd?.toFixed(4) }}</strong>
						</div>
					</div>

					<div class="secondary-stat-card">
						<h4>
							<font-awesome-icon :icon="['fas', 'tasks']" />
							Activities
						</h4>
						<div class="secondary-stat-row">
							<span>Running:</span>
							<strong class="text-success">{{ health.application.activities?.running }}</strong>
						</div>
						<div class="secondary-stat-row">
							<span>Paused:</span>
							<strong class="text-warning">{{ health.application.activities?.paused }}</strong>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, getCurrentInstance } from 'vue'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

const loading = ref(true)
const health = ref(null)
const lastUpdated = ref(null)
let refreshInterval = null

const overallStatus = computed(() => {
	if (!health.value) return 'unknown'

	const dbHealthy = health.value.database?.status === 'healthy'
	const servicesHealthy = Object.values(health.value.services || {}).every(s => s.status === 'healthy' || s.status === 'unavailable')

	if (dbHealthy && servicesHealthy) return 'healthy'
	return 'degraded'
})

const refreshHealth = async () => {
	loading.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/system/health')
		const data = await response.json()
		if (data.success) {
			health.value = data.data
			lastUpdated.value = new Date()
		} else {
			toast.error('Failed to load system health')
		}
	} catch (error) {
		console.error('Failed to load system health:', error)
		toast.error('Failed to connect to server')
	} finally {
		loading.value = false
	}
}

const getStatusIcon = (status) => {
	if (status === 'healthy') return ['fas', 'check-circle']
	if (status === 'degraded') return ['fas', 'exclamation-triangle']
	return ['fas', 'times-circle']
}

const getServiceIcon = (name) => {
	const icons = {
		scheduler: ['fas', 'clock'],
		scraper: ['fas', 'spider'],
		activity: ['fas', 'chart-line'],
		ai_audit: ['fas', 'brain'],
		jdownloader: ['fas', 'download']
	}
	return icons[name] || ['fas', 'cog']
}

const formatServiceName = (name) => {
	const names = {
		scheduler: 'Scheduler',
		scraper: 'Scraper',
		activity: 'Activity Tracker',
		ai_audit: 'AI Audit',
		jdownloader: 'JDownloader'
	}
	return names[name] || name
}

const formatTableName = (table) => {
	return table
		.split('_')
		.map(word => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ')
}

const formatTime = (date) => {
	if (!date) return 'Never'
	return date.toLocaleTimeString()
}

onMounted(() => {
	refreshHealth()
	// Auto-refresh every 30 seconds
	refreshInterval = setInterval(refreshHealth, 30000)
})

onBeforeUnmount(() => {
	if (refreshInterval) {
		clearInterval(refreshInterval)
	}
})
</script>

<style scoped>
.system-health-page {
	padding: 2rem;
	min-height: 100vh;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 2rem;
}

.page-header h1 {
	font-size: 2rem;
	font-weight: 700;
	display: flex;
	align-items: center;
	gap: 1rem;
	color: #e0e0e0;
}

.header-actions {
	display: flex;
	align-items: center;
	gap: 1rem;
}

.last-updated {
	color: #aaa;
	font-size: 0.9rem;
}

.loading-container {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 4rem;
	gap: 1rem;
	color: #aaa;
}

.status-banner {
	display: flex;
	align-items: center;
	gap: 2rem;
	padding: 2rem;
	border-radius: 12px;
	margin-bottom: 2rem;
	backdrop-filter: blur(20px);
	border: 2px solid;
}

.status-banner.status-healthy {
	background: rgba(76, 175, 80, 0.1);
	border-color: rgba(76, 175, 80, 0.4);
	color: #4caf50;
}

.status-banner.status-degraded {
	background: rgba(255, 152, 0, 0.1);
	border-color: rgba(255, 152, 0, 0.4);
	color: #ff9800;
}

.status-icon {
	font-size: 3rem;
}

.status-info h2 {
	margin: 0 0 0.5rem 0;
	font-size: 1.5rem;
}

.status-info p {
	margin: 0;
	opacity: 0.8;
}

.section {
	margin-bottom: 3rem;
}

.section-title {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	font-size: 1.3rem;
	color: #e0e0e0;
	margin-bottom: 1.5rem;
	padding-bottom: 0.75rem;
	border-bottom: 2px solid rgba(255, 255, 255, 0.1);
}

.service-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 1.5rem;
}

.service-card {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
	transition: all 0.3s ease;
}

.service-card:hover {
	background: rgba(255, 255, 255, 0.05);
	transform: translateY(-2px);
}

.service-card.status-healthy {
	border-left: 4px solid #4caf50;
}

.service-card.status-unavailable {
	border-left: 4px solid #f44336;
	opacity: 0.6;
}

.service-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 1rem;
}

.service-icon {
	font-size: 1.5rem;
	color: #667eea;
}

.service-header h4 {
	margin: 0;
	color: #e0e0e0;
	font-size: 1rem;
}

.service-status {
	margin-top: 0.75rem;
}

.status-badge {
	padding: 0.25rem 0.75rem;
	border-radius: 12px;
	font-size: 0.75rem;
	font-weight: 600;
	text-transform: uppercase;
}

.status-badge.badge-healthy {
	background: rgba(76, 175, 80, 0.2);
	color: #4caf50;
}

.status-badge.badge-unavailable {
	background: rgba(244, 67, 54, 0.2);
	color: #f44336;
}

.service-version {
	margin-top: 0.5rem;
	font-size: 0.8rem;
	color: #aaa;
}

.db-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 1.5rem;
	margin-bottom: 2rem;
}

.resources-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 1.5rem;
}

.metric-card {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
	display: flex;
	align-items: center;
	gap: 1rem;
	transition: all 0.3s ease;
}

.metric-card:hover {
	background: rgba(255, 255, 255, 0.05);
	transform: translateY(-2px);
}

.metric-icon {
	width: 50px;
	height: 50px;
	border-radius: 10px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.5rem;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
}

.metric-icon.memory {
	background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.metric-icon.cpu {
	background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.metric-icon.gc {
	background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.metric-content {
	flex: 1;
}

.metric-label {
	font-size: 0.85rem;
	color: #aaa;
	margin-bottom: 0.25rem;
}

.metric-value {
	font-size: 1.3rem;
	font-weight: 700;
	color: #e0e0e0;
}

.table-counts {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
}

.table-counts h4 {
	margin: 0 0 1rem 0;
	color: #e0e0e0;
}

.table-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 0.75rem;
}

.table-item {
	display: flex;
	justify-content: space-between;
	padding: 0.75rem;
	background: rgba(255, 255, 255, 0.02);
	border-radius: 6px;
}

.table-name {
	color: #aaa;
	font-size: 0.9rem;
}

.table-count {
	color: #e0e0e0;
	font-weight: 600;
}

.app-stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
	gap: 1.5rem;
	margin-bottom: 2rem;
}

.stat-card {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
	display: flex;
	align-items: center;
	gap: 1rem;
	transition: all 0.3s ease;
}

.stat-card:hover {
	background: rgba(255, 255, 255, 0.05);
	transform: translateY(-2px);
}

.stat-icon {
	width: 60px;
	height: 60px;
	border-radius: 12px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.8rem;
}

.stat-icon.videos {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.stat-icon.performers {
	background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-icon.studios {
	background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.stat-icon.tags {
	background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.stat-content {
	flex: 1;
}

.stat-value {
	font-size: 1.8rem;
	font-weight: 700;
	color: #e0e0e0;
	line-height: 1;
	margin-bottom: 0.25rem;
}

.stat-label {
	font-size: 0.9rem;
	color: #aaa;
	margin-bottom: 0.25rem;
}

.stat-sub {
	font-size: 0.8rem;
	color: #888;
}

.secondary-stats {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
	gap: 1.5rem;
}

.secondary-stat-card {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
}

.secondary-stat-card h4 {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	margin: 0 0 1rem 0;
	color: #e0e0e0;
	padding-bottom: 0.75rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.secondary-stat-row {
	display: flex;
	justify-content: space-between;
	padding: 0.5rem 0;
	color: #aaa;
	font-size: 0.9rem;
}

.secondary-stat-row strong {
	color: #e0e0e0;
}

.text-success {
	color: #4caf50 !important;
}

.text-warning {
	color: #ff9800 !important;
}

.text-danger {
	color: #f44336 !important;
}

.text-info {
	color: #2196f3 !important;
}

.btn {
	padding: 0.5rem 1rem;
	border-radius: 8px;
	border: none;
	font-size: 0.9rem;
	font-weight: 500;
	cursor: pointer;
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	transition: all 0.2s ease;
}

.btn-primary {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
}

.btn-primary:hover:not(:disabled) {
	transform: translateY(-2px);
	box-shadow: 0 5px 20px rgba(103, 126, 234, 0.4);
}

.btn-primary:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.btn-sm {
	padding: 0.4rem 0.8rem;
	font-size: 0.85rem;
}
</style>
