<template>
	<div class="activity-dashboard">
		<div class="dashboard-header">
			<h1>Activity Dashboard</h1>
			<div class="header-actions">
				<button class="btn btn-refresh" @click="refreshAll" :disabled="loading">
					<font-awesome-icon :icon="['fas', loading ? 'spinner' : 'sync']" :spin="loading" />
					Refresh
				</button>
			</div>
		</div>

		<!-- System Health Status -->
		<div class="health-cards">
			<div class="health-card">
				<div class="health-icon">
					<font-awesome-icon :icon="['fas', 'database']" />
				</div>
				<div class="health-info">
					<div class="health-label">Database</div>
					<div class="health-value">{{ dbStats.size || 'N/A' }}</div>
				</div>
				<div class="health-status status-ok">
					<font-awesome-icon :icon="['fas', 'check-circle']" />
				</div>
			</div>

			<div class="health-card">
				<div class="health-icon jdownloader">
					<font-awesome-icon :icon="['fas', 'download']" />
				</div>
				<div class="health-info">
					<div class="health-label">JDownloader</div>
					<div class="health-value">{{ jdStatus.available ? 'Online' : 'Offline' }}</div>
				</div>
				<div :class="['health-status', jdStatus.available ? 'status-ok' : 'status-offline']">
					<font-awesome-icon :icon="['fas', jdStatus.available ? 'check-circle' : 'times-circle']" />
				</div>
			</div>

			<div class="health-card">
				<div class="health-icon ai">
					<font-awesome-icon :icon="['fas', 'brain']" />
				</div>
				<div class="health-info">
					<div class="health-label">AI Usage (24h)</div>
					<div class="health-value">${{ aiStats.last_24_hour_cost?.toFixed(2) || '0.00' }}</div>
				</div>
				<div class="health-status status-ok">
					<font-awesome-icon :icon="['fas', 'chart-line']" />
				</div>
			</div>

			<div class="health-card">
				<div class="health-icon downloads">
					<font-awesome-icon :icon="['fas', 'cloud-download-alt']" />
				</div>
				<div class="health-info">
					<div class="health-label">Downloaded</div>
					<div class="health-value">{{ downloadStats.downloaded || 0 }} / {{ downloadStats.total_links || 0 }}</div>
				</div>
				<div class="health-status status-ok">
					<font-awesome-icon :icon="['fas', 'check']" />
				</div>
			</div>
		</div>

		<!-- Active Operations -->
		<div class="section">
			<div class="section-header">
				<h2>
					<font-awesome-icon :icon="['fas', 'tasks']" />
					Active Now
					<span v-if="activeActivities.length > 0" class="badge">{{ activeActivities.length }}</span>
				</h2>
			</div>
			<div v-if="activeActivities.length === 0" class="empty-state">
				<font-awesome-icon :icon="['fas', 'check-circle']" size="3x" />
				<p>No active operations</p>
			</div>
			<div v-else class="active-operations">
				<div v-for="activity in activeActivities" :key="activity.id" class="operation-card">
					<div class="operation-icon">
						<font-awesome-icon :icon="getActivityIcon(activity.type)" :spin="activity.status === 'running'" />
					</div>
					<div class="operation-info">
						<div class="operation-title">{{ activity.type }}</div>
						<div class="operation-description">{{ activity.description }}</div>
						<div class="operation-progress">
							<div class="progress-bar">
								<div class="progress-fill" :style="{ width: activity.progress + '%' }"></div>
							</div>
							<span class="progress-text">{{ activity.progress }}%</span>
						</div>
					</div>
					<div class="operation-actions">
						<button
							v-if="activity.status === 'running' && !activity.is_paused"
							@click="pauseActivity(activity.id)"
							class="btn-icon"
							title="Pause"
						>
							<font-awesome-icon :icon="['fas', 'pause']" />
						</button>
						<button
							v-if="activity.is_paused"
							@click="resumeActivity(activity.id)"
							class="btn-icon btn-success"
							title="Resume"
						>
							<font-awesome-icon :icon="['fas', 'play']" />
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Recent Activity Feed -->
		<div class="section">
			<div class="section-header">
				<h2>
					<font-awesome-icon :icon="['fas', 'history']" />
					Recent Activity
				</h2>
				<div class="filter-tabs">
					<button
						v-for="filter in activityFilters"
						:key="filter.value"
						:class="['filter-tab', { active: selectedFilter === filter.value }]"
						@click="selectedFilter = filter.value"
					>
						{{ filter.label }}
					</button>
				</div>
			</div>
			<div class="activity-feed">
				<div v-for="activity in filteredActivities" :key="activity.id" class="activity-item">
					<div :class="['activity-status-dot', `status-${activity.status}`]"></div>
					<div class="activity-content">
						<div class="activity-header">
							<span class="activity-type">{{ activity.type }}</span>
							<span class="activity-time">{{ formatTimeAgo(activity.created_at) }}</span>
						</div>
						<div class="activity-description">{{ activity.description }}</div>
						<div v-if="activity.result" class="activity-result">
							{{ activity.result }}
						</div>
					</div>
					<div :class="['activity-icon', `status-${activity.status}`]">
						<font-awesome-icon :icon="getStatusIcon(activity.status)" />
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

// State
const loading = ref(false)
const activeActivities = ref([])
const recentActivities = ref([])
const jdStatus = ref({ available: false })
const aiStats = ref({})
const downloadStats = ref({})
const dbStats = ref({})
const selectedFilter = ref('all')

const activityFilters = [
	{ label: 'All', value: 'all' },
	{ label: 'Completed', value: 'completed' },
	{ label: 'Failed', value: 'failed' },
	{ label: 'Running', value: 'running' }
]

// Computed
const filteredActivities = computed(() => {
	if (selectedFilter.value === 'all') {
		return recentActivities.value
	}
	return recentActivities.value.filter(a => a.status === selectedFilter.value)
})

// Methods
const loadActiveActivities = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/activities?status=running,paused&limit=50')
		const data = await response.json()
		if (data.success) {
			activeActivities.value = data.data.activities || []
		}
	} catch (error) {
		console.error('Error loading active activities:', error)
	}
}

const loadRecentActivities = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/activities?limit=50')
		const data = await response.json()
		if (data.success) {
			recentActivities.value = data.data.activities || []
		}
	} catch (error) {
		console.error('Error loading recent activities:', error)
	}
}

const loadJDownloaderStatus = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/jdownloader/status')
		const data = await response.json()
		if (data.success) {
			jdStatus.value = data.data
		}
	} catch (error) {
		console.error('Error loading JDownloader status:', error)
	}
}

const loadAIStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/ai-audit/stats')
		const data = await response.json()
		if (data.success) {
			aiStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading AI stats:', error)
	}
}

const loadDownloadStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/downloads/stats')
		const data = await response.json()
		if (data.success) {
			downloadStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading download stats:', error)
	}
}

const loadDBStats = async () => {
	// Placeholder - would need backend endpoint
	dbStats.value = { size: '2.4 GB' }
}

const refreshAll = async () => {
	loading.value = true
	try {
		await Promise.all([
			loadActiveActivities(),
			loadRecentActivities(),
			loadJDownloaderStatus(),
			loadAIStats(),
			loadDownloadStats(),
			loadDBStats()
		])
	} finally {
		loading.value = false
	}
}

const pauseActivity = async (activityId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/activities/${activityId}/pause`, {
			method: 'POST'
		})
		const data = await response.json()
		if (data.success) {
			await loadActiveActivities()
		}
	} catch (error) {
		console.error('Error pausing activity:', error)
	}
}

const resumeActivity = async (activityId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/activities/${activityId}/resume`, {
			method: 'POST'
		})
		const data = await response.json()
		if (data.success) {
			await loadActiveActivities()
		}
	} catch (error) {
		console.error('Error resuming activity:', error)
	}
}

const getActivityIcon = (type) => {
	const icons = {
		'Scraper': ['fas', 'spider'],
		'Link Verification': ['fas', 'check-double'],
		'AI Companion': ['fas', 'brain'],
		'Download': ['fas', 'download'],
		'Export': ['fas', 'file-export']
	}
	return icons[type] || ['fas', 'cog']
}

const getStatusIcon = (status) => {
	const icons = {
		'completed': ['fas', 'check-circle'],
		'failed': ['fas', 'times-circle'],
		'running': ['fas', 'spinner'],
		'paused': ['fas', 'pause-circle']
	}
	return icons[status] || ['fas', 'question-circle']
}

const formatTimeAgo = (timestamp) => {
	const now = new Date()
	const time = new Date(timestamp)
	const diff = Math.floor((now - time) / 1000) // seconds

	if (diff < 60) return `${diff}s ago`
	if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
	if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
	return `${Math.floor(diff / 86400)}d ago`
}

// Auto-refresh
let refreshInterval
onMounted(() => {
	refreshAll()
	// Refresh every 5 seconds
	refreshInterval = setInterval(() => {
		loadActiveActivities()
		loadRecentActivities()
	}, 5000)
})

onBeforeUnmount(() => {
	if (refreshInterval) {
		clearInterval(refreshInterval)
	}
})
</script>

<style scoped>
.activity-dashboard {
	padding: 2rem;
	max-width: 1400px;
	margin: 0 auto;
}

.dashboard-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 2rem;
}

.dashboard-header h1 {
	font-size: 2rem;
	font-weight: 700;
	color: #fff;
	margin: 0;
}

.header-actions {
	display: flex;
	gap: 1rem;
}

.btn-refresh {
	padding: 0.75rem 1.5rem;
	border-radius: 10px;
	border: none;
	background: linear-gradient(135deg, #4dabf7 0%, #339af0 100%);
	color: #fff;
	font-weight: 500;
	cursor: pointer;
	transition: all 0.2s;
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.btn-refresh:hover:not(:disabled) {
	transform: translateY(-2px);
	box-shadow: 0 4px 12px rgba(74, 171, 247, 0.4);
}

.btn-refresh:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

/* Health Cards */
.health-cards {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
	gap: 1.5rem;
	margin-bottom: 2rem;
}

.health-card {
	background: rgba(255, 255, 255, 0.05);
	backdrop-filter: blur(20px);
	border-radius: 16px;
	padding: 1.5rem;
	border: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	align-items: center;
	gap: 1rem;
	transition: all 0.2s;
}

.health-card:hover {
	background: rgba(255, 255, 255, 0.08);
	transform: translateY(-2px);
}

.health-icon {
	width: 56px;
	height: 56px;
	border-radius: 12px;
	background: rgba(74, 171, 247, 0.15);
	color: #4dabf7;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.5rem;
}

.health-icon.jdownloader {
	background: rgba(255, 107, 107, 0.15);
	color: #ff6b6b;
}

.health-icon.ai {
	background: rgba(139, 92, 246, 0.15);
	color: #8b5cf6;
}

.health-icon.downloads {
	background: rgba(81, 207, 102, 0.15);
	color: #51cf66;
}

.health-info {
	flex: 1;
}

.health-label {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.6);
	margin-bottom: 0.25rem;
}

.health-value {
	font-size: 1.5rem;
	font-weight: 700;
	color: #fff;
}

.health-status {
	font-size: 1.5rem;
}

.status-ok {
	color: #51cf66;
}

.status-offline {
	color: #ff6b6b;
}

/* Sections */
.section {
	margin-bottom: 2rem;
}

.section-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1.5rem;
}

.section-header h2 {
	font-size: 1.5rem;
	font-weight: 600;
	color: #fff;
	display: flex;
	align-items: center;
	gap: 0.75rem;
	margin: 0;
}

.badge {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	min-width: 24px;
	height: 24px;
	padding: 0 8px;
	background: linear-gradient(135deg, #4dabf7 0%, #339af0 100%);
	border-radius: 12px;
	font-size: 0.85rem;
	font-weight: 600;
}

/* Filter Tabs */
.filter-tabs {
	display: flex;
	gap: 0.5rem;
}

.filter-tab {
	padding: 0.5rem 1rem;
	border-radius: 8px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	background: transparent;
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	transition: all 0.2s;
	font-size: 0.9rem;
}

.filter-tab:hover {
	background: rgba(255, 255, 255, 0.05);
	color: #fff;
}

.filter-tab.active {
	background: linear-gradient(135deg, #4dabf7 0%, #339af0 100%);
	color: #fff;
	border-color: transparent;
}

/* Active Operations */
.active-operations {
	display: grid;
	gap: 1rem;
}

.operation-card {
	background: rgba(255, 255, 255, 0.05);
	backdrop-filter: blur(20px);
	border-radius: 12px;
	padding: 1.5rem;
	border: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	align-items: center;
	gap: 1.5rem;
}

.operation-icon {
	width: 48px;
	height: 48px;
	border-radius: 12px;
	background: rgba(74, 171, 247, 0.15);
	color: #4dabf7;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.25rem;
}

.operation-info {
	flex: 1;
}

.operation-title {
	font-size: 1.1rem;
	font-weight: 600;
	color: #fff;
	margin-bottom: 0.25rem;
}

.operation-description {
	font-size: 0.9rem;
	color: rgba(255, 255, 255, 0.6);
	margin-bottom: 0.75rem;
}

.operation-progress {
	display: flex;
	align-items: center;
	gap: 1rem;
}

.progress-bar {
	flex: 1;
	height: 8px;
	background: rgba(255, 255, 255, 0.1);
	border-radius: 4px;
	overflow: hidden;
}

.progress-fill {
	height: 100%;
	background: linear-gradient(90deg, #4dabf7 0%, #339af0 100%);
	transition: width 0.3s;
}

.progress-text {
	font-size: 0.9rem;
	font-weight: 600;
	color: #4dabf7;
	min-width: 45px;
	text-align: right;
}

.operation-actions {
	display: flex;
	gap: 0.5rem;
}

.btn-icon {
	width: 40px;
	height: 40px;
	border-radius: 10px;
	border: none;
	background: rgba(255, 255, 255, 0.1);
	color: #fff;
	cursor: pointer;
	transition: all 0.2s;
	display: flex;
	align-items: center;
	justify-content: center;
}

.btn-icon:hover {
	background: rgba(255, 255, 255, 0.2);
	transform: scale(1.1);
}

.btn-icon.btn-success {
	background: rgba(81, 207, 102, 0.2);
	color: #51cf66;
}

/* Activity Feed */
.activity-feed {
	display: grid;
	gap: 1rem;
}

.activity-item {
	background: rgba(255, 255, 255, 0.03);
	backdrop-filter: blur(10px);
	border-radius: 10px;
	padding: 1rem 1.25rem;
	border: 1px solid rgba(255, 255, 255, 0.08);
	display: flex;
	align-items: flex-start;
	gap: 1rem;
	transition: all 0.2s;
}

.activity-item:hover {
	background: rgba(255, 255, 255, 0.05);
	border-color: rgba(255, 255, 255, 0.15);
}

.activity-status-dot {
	width: 8px;
	height: 8px;
	border-radius: 50%;
	margin-top: 0.5rem;
	flex-shrink: 0;
}

.activity-status-dot.status-completed {
	background: #51cf66;
}

.activity-status-dot.status-failed {
	background: #ff6b6b;
}

.activity-status-dot.status-running {
	background: #4dabf7;
	animation: pulse 2s infinite;
}

@keyframes pulse {
	0%, 100% { opacity: 1; }
	50% { opacity: 0.5; }
}

.activity-content {
	flex: 1;
	min-width: 0;
}

.activity-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 0.5rem;
}

.activity-type {
	font-weight: 600;
	color: #4dabf7;
	font-size: 0.9rem;
}

.activity-time {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.4);
}

.activity-description {
	font-size: 0.9rem;
	color: rgba(255, 255, 255, 0.8);
	margin-bottom: 0.25rem;
}

.activity-result {
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.5);
}

.activity-icon {
	width: 32px;
	height: 32px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	flex-shrink: 0;
	margin-top: 0.25rem;
}

.activity-icon.status-completed {
	background: rgba(81, 207, 102, 0.15);
	color: #51cf66;
}

.activity-icon.status-failed {
	background: rgba(255, 107, 107, 0.15);
	color: #ff6b6b;
}

.activity-icon.status-running {
	background: rgba(74, 171, 247, 0.15);
	color: #4dabf7;
}

/* Empty State */
.empty-state {
	text-align: center;
	padding: 4rem 2rem;
	color: rgba(255, 255, 255, 0.5);
}

.empty-state svg {
	color: rgba(255, 255, 255, 0.3);
	margin-bottom: 1rem;
}

.empty-state p {
	font-size: 1.1rem;
	margin: 0;
}
</style>
