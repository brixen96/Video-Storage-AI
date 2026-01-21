<template>
	<div class="link-health-page">
		<div class="page-header">
			<div class="header-content">
				<h1><font-awesome-icon :icon="['fas', 'heartbeat']" /> Link Health Monitor</h1>
				<p>Monitor download link availability and health status</p>
			</div>
			<div class="header-actions">
				<button class="btn btn-primary" @click="verifyAllLinks" :disabled="verifying">
					<font-awesome-icon :icon="['fas', verifying ? 'spinner' : 'check-circle']" :spin="verifying" />
					{{ verifying ? 'Verifying...' : 'Verify All Links' }}
				</button>
			</div>
		</div>

		<!-- Stats Dashboard -->
		<div class="stats-grid" v-if="stats">
			<div class="stat-card health-card">
				<div class="stat-icon" :class="`health-${getHealthClass(stats.health_percent)}`">
					<font-awesome-icon :icon="['fas', 'heartbeat']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Link Health</div>
					<div class="stat-value">{{ stats.health_percent ? stats.health_percent.toFixed(1) : 0 }}%</div>
					<div class="health-bar">
						<div
							class="health-bar-fill"
							:class="`health-${getHealthClass(stats.health_percent)}`"
							:style="{ width: (stats.health_percent || 0) + '%' }"
						></div>
					</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon valid">
					<font-awesome-icon :icon="['fas', 'check-circle']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Valid Links</div>
					<div class="stat-value">{{ stats.valid_links || 0 }}</div>
					<div class="stat-sub">{{ ((stats.valid_links / stats.total_links) * 100 || 0).toFixed(1) }}% of total</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon dead">
					<font-awesome-icon :icon="['fas', 'times-circle']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Dead Links</div>
					<div class="stat-value">{{ stats.dead_links || 0 }}</div>
					<div class="stat-sub">{{ ((stats.dead_links / stats.total_links) * 100 || 0).toFixed(1) }}% of total</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon unverified">
					<font-awesome-icon :icon="['fas', 'question-circle']" />
				</div>
				<div class="stat-content">
					<div class="stat-label">Unverified</div>
					<div class="stat-value">{{ stats.unverified_links || 0 }}</div>
					<div class="stat-sub">Need verification</div>
				</div>
			</div>
		</div>

		<!-- Last Verification Time -->
		<div v-if="stats && stats.last_verified" class="last-verification">
			<font-awesome-icon :icon="['fas', 'clock']" />
			Last verification: {{ formatTime(stats.last_verified) }}
		</div>

		<!-- Dead Links Section -->
		<div class="dead-links-section">
			<div class="section-header">
				<h2>
					<font-awesome-icon :icon="['fas', 'times-circle']" />
					Dead Links
				</h2>
				<button class="btn btn-secondary btn-sm" @click="loadDeadLinks">
					<font-awesome-icon :icon="['fas', 'sync']" :spin="loading" />
					Refresh
				</button>
			</div>

			<div v-if="loading" class="loading-state">
				<font-awesome-icon :icon="['fas', 'spinner']" spin size="2x" />
				<p>Loading dead links...</p>
			</div>

			<div v-else-if="deadLinks.length === 0" class="empty-state">
				<font-awesome-icon :icon="['fas', 'check-circle']" size="3x" />
				<p>No dead links found!</p>
				<p class="empty-sub">All your download links are healthy</p>
			</div>

			<div v-else class="dead-links-list">
				<div v-for="link in deadLinks" :key="link.id" class="dead-link-card">
					<div class="link-status">
						<font-awesome-icon :icon="['fas', 'times-circle']" class="status-icon dead" />
						<span class="status-code" v-if="link.verification_status_code">
							{{ link.verification_status_code }}
						</span>
					</div>
					<div class="link-info">
						<div class="link-url">
							<a :href="link.url" target="_blank" rel="noopener noreferrer">
								{{ truncateURL(link.url) }}
							</a>
						</div>
						<div class="link-meta">
							<span class="link-provider" v-if="link.provider">
								<font-awesome-icon :icon="['fas', 'server']" />
								{{ link.provider }}
							</span>
							<span class="link-verified" v-if="link.last_verified_at">
								<font-awesome-icon :icon="['fas', 'clock']" />
								{{ formatTime(link.last_verified_at) }}
							</span>
							<span class="link-error" v-if="link.verification_error">
								<font-awesome-icon :icon="['fas', 'exclamation-triangle']" />
								{{ link.verification_error }}
							</span>
						</div>
					</div>
					<div class="link-actions">
						<button
							class="btn btn-sm btn-warning"
							@click="reverifyLink(link)"
							:title="'Re-verify link'"
						>
							<font-awesome-icon :icon="['fas', 'redo']" />
							Re-verify
						</button>
						<button
							class="btn btn-sm btn-danger"
							@click="confirmDeleteLink(link)"
							:title="'Delete dead link'"
						>
							<font-awesome-icon :icon="['fas', 'trash']" />
							Delete
						</button>
					</div>
				</div>
			</div>

			<!-- Pagination -->
			<div v-if="deadLinks.length > 0" class="pagination">
				<button
					class="btn btn-secondary btn-sm"
					@click="prevPage"
					:disabled="currentPage === 0"
				>
					<font-awesome-icon :icon="['fas', 'chevron-left']" />
					Previous
				</button>
				<span class="page-info">Page {{ currentPage + 1 }}</span>
				<button
					class="btn btn-secondary btn-sm"
					@click="nextPage"
					:disabled="deadLinks.length < pageSize"
				>
					Next
					<font-awesome-icon :icon="['fas', 'chevron-right']" />
				</button>
			</div>
		</div>

		<!-- Delete Confirmation Modal -->
		<div v-if="showDeleteModal" class="modal-overlay" @click="showDeleteModal = false">
			<div class="modal-content" @click.stop>
				<div class="modal-header">
					<h3><font-awesome-icon :icon="['fas', 'trash']" /> Delete Dead Link</h3>
					<button class="close-btn" @click="showDeleteModal = false">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p>Are you sure you want to delete this dead link?</p>
					<p class="link-preview">{{ selectedLink?.url }}</p>
					<p>This action cannot be undone.</p>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showDeleteModal = false">Cancel</button>
					<button class="btn btn-danger" @click="deleteLink" :disabled="deleting">
						<font-awesome-icon :icon="['fas', deleting ? 'spinner' : 'trash']" :spin="deleting" />
						{{ deleting ? 'Deleting...' : 'Delete Link' }}
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted, getCurrentInstance } from 'vue'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

const loading = ref(true)
const verifying = ref(false)
const deleting = ref(false)
const stats = ref(null)
const deadLinks = ref([])
const selectedLink = ref(null)
const showDeleteModal = ref(false)
const currentPage = ref(0)
const pageSize = 50

const loadStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/links/verification-stats')
		const data = await response.json()
		if (data.success) {
			stats.value = data.data
		}
	} catch (error) {
		console.error('Failed to load stats:', error)
		toast.error('Failed to load verification statistics')
	}
}

const loadDeadLinks = async () => {
	loading.value = true
	try {
		const offset = currentPage.value * pageSize
		const response = await fetch(
			`http://localhost:8080/api/v1/scraper/links/dead?limit=${pageSize}&offset=${offset}`
		)
		const data = await response.json()
		if (data.success) {
			deadLinks.value = data.data.links || []
		}
	} catch (error) {
		console.error('Failed to load dead links:', error)
		toast.error('Failed to load dead links')
	} finally {
		loading.value = false
	}
}

const verifyAllLinks = async () => {
	verifying.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/links/verify-all', {
			method: 'POST',
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Link verification started')
			// Reload stats after a delay
			setTimeout(() => {
				loadStats()
				loadDeadLinks()
			}, 3000)
		} else {
			toast.error(data.error?.message || 'Failed to start verification')
		}
	} catch (error) {
		console.error('Failed to verify links:', error)
		toast.error('Failed to start verification')
	} finally {
		verifying.value = false
	}
}

const reverifyLink = async (link) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/links/${link.id}/verify`, {
			method: 'POST',
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Link re-verified')
			loadDeadLinks()
			loadStats()
		} else {
			toast.error(data.error?.message || 'Failed to re-verify link')
		}
	} catch (error) {
		console.error('Failed to re-verify link:', error)
		toast.error('Failed to re-verify link')
	}
}

const confirmDeleteLink = (link) => {
	selectedLink.value = link
	showDeleteModal.value = true
}

const deleteLink = async () => {
	deleting.value = true
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/links/${selectedLink.value.id}`, {
			method: 'DELETE',
		})
		const data = await response.json()
		if (data.success) {
			toast.success('Dead link deleted')
			showDeleteModal.value = false
			loadDeadLinks()
			loadStats()
		} else {
			toast.error(data.error?.message || 'Failed to delete link')
		}
	} catch (error) {
		console.error('Failed to delete link:', error)
		toast.error('Failed to delete link')
	} finally {
		deleting.value = false
	}
}

const getHealthClass = (percent) => {
	if (percent >= 80) return 'good'
	if (percent >= 50) return 'warning'
	return 'bad'
}

const truncateURL = (url) => {
	if (url.length > 80) {
		return url.substring(0, 77) + '...'
	}
	return url
}

const formatTime = (dateStr) => {
	const date = new Date(dateStr)
	const now = new Date()
	const diffMs = now - date
	const diffMins = Math.floor(diffMs / 60000)
	const diffHours = Math.floor(diffMs / 3600000)
	const diffDays = Math.floor(diffMs / 86400000)

	if (diffMins < 1) return 'Just now'
	if (diffMins < 60) return `${diffMins}m ago`
	if (diffHours < 24) return `${diffHours}h ago`
	if (diffDays < 7) return `${diffDays}d ago`
	return date.toLocaleDateString()
}

const nextPage = () => {
	currentPage.value++
	loadDeadLinks()
}

const prevPage = () => {
	if (currentPage.value > 0) {
		currentPage.value--
		loadDeadLinks()
	}
}

onMounted(() => {
	loadStats()
	loadDeadLinks()
})
</script>

<style scoped>
.link-health-page {
	padding: 2rem;
	max-width: 1600px;
	margin: 0 auto;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	margin-bottom: 2rem;
	padding-bottom: 1.5rem;
	border-bottom: 2px solid rgba(255, 255, 255, 0.1);
}

.header-content h1 {
	margin: 0;
	font-size: 2rem;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
}

.header-content p {
	margin: 0.5rem 0 0 0;
	color: #aaa;
}

.stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
	gap: 1.5rem;
	margin-bottom: 2rem;
}

.stat-card {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 1.5rem;
	display: flex;
	gap: 1rem;
	align-items: flex-start;
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
	font-size: 1.5rem;
	color: white;
	flex-shrink: 0;
}

.stat-icon.valid {
	background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
}

.stat-icon.dead {
	background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%);
}

.stat-icon.unverified {
	background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.stat-icon.health-good {
	background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
}

.stat-icon.health-warning {
	background: linear-gradient(135deg, #f2994a 0%, #f2c94c 100%);
}

.stat-icon.health-bad {
	background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%);
}

.stat-content {
	flex: 1;
}

.stat-label {
	font-size: 0.9rem;
	color: #aaa;
	margin-bottom: 0.25rem;
}

.stat-value {
	font-size: 1.8rem;
	font-weight: 700;
	color: #e0e0e0;
}

.stat-sub {
	font-size: 0.85rem;
	color: #888;
	margin-top: 0.25rem;
}

.health-bar {
	margin-top: 0.75rem;
	height: 8px;
	background: rgba(255, 255, 255, 0.1);
	border-radius: 4px;
	overflow: hidden;
}

.health-bar-fill {
	height: 100%;
	transition: width 0.5s ease;
	border-radius: 4px;
}

.health-bar-fill.health-good {
	background: linear-gradient(90deg, #11998e 0%, #38ef7d 100%);
}

.health-bar-fill.health-warning {
	background: linear-gradient(90deg, #f2994a 0%, #f2c94c 100%);
}

.health-bar-fill.health-bad {
	background: linear-gradient(90deg, #eb3349 0%, #f45c43 100%);
}

.last-verification {
	text-align: center;
	padding: 1rem;
	background: rgba(255, 255, 255, 0.03);
	border-radius: 8px;
	color: #aaa;
	font-size: 0.9rem;
	margin-bottom: 2rem;
}

.dead-links-section {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	padding: 2rem;
}

.section-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 1.5rem;
}

.section-header h2 {
	margin: 0;
	color: #e0e0e0;
	font-size: 1.3rem;
}

.loading-state,
.empty-state {
	text-align: center;
	padding: 3rem;
	color: #aaa;
}

.empty-state svg {
	color: #38ef7d;
	margin-bottom: 1rem;
}

.empty-sub {
	color: #666;
	font-size: 0.9rem;
}

.dead-links-list {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.dead-link-card {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1rem;
	background: rgba(255, 255, 255, 0.02);
	border: 1px solid rgba(235, 51, 73, 0.3);
	border-left: 4px solid #eb3349;
	border-radius: 8px;
	transition: all 0.2s ease;
}

.dead-link-card:hover {
	background: rgba(255, 255, 255, 0.05);
	border-color: rgba(235, 51, 73, 0.5);
}

.link-status {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	flex-shrink: 0;
}

.status-icon {
	font-size: 1.5rem;
	color: #eb3349;
}

.status-code {
	background: rgba(235, 51, 73, 0.2);
	color: #eb3349;
	padding: 0.25rem 0.5rem;
	border-radius: 4px;
	font-size: 0.85rem;
	font-weight: 600;
}

.link-info {
	flex: 1;
	min-width: 0;
}

.link-url a {
	color: #e0e0e0;
	text-decoration: none;
	font-size: 0.95rem;
	word-break: break-all;
}

.link-url a:hover {
	color: #667eea;
	text-decoration: underline;
}

.link-meta {
	display: flex;
	gap: 1rem;
	flex-wrap: wrap;
	margin-top: 0.5rem;
	font-size: 0.85rem;
	color: #aaa;
}

.link-error {
	color: #f2994a;
}

.link-actions {
	display: flex;
	gap: 0.5rem;
	flex-shrink: 0;
}

.pagination {
	display: flex;
	justify-content: center;
	align-items: center;
	gap: 1rem;
	margin-top: 2rem;
	padding-top: 1rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.page-info {
	color: #aaa;
	font-size: 0.9rem;
}

.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 2000;
	padding: 1rem;
}

.modal-content {
	background: rgba(30, 30, 40, 0.98);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	max-width: 600px;
	width: 100%;
	max-height: 90vh;
	overflow-y: auto;
	box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
}

.modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.modal-header h3 {
	margin: 0;
	color: #e0e0e0;
	font-size: 1.2rem;
}

.close-btn {
	background: none;
	border: none;
	color: #aaa;
	font-size: 1.2rem;
	cursor: pointer;
	padding: 0.5rem;
	transition: color 0.2s;
}

.close-btn:hover {
	color: #fff;
}

.modal-body {
	padding: 1.5rem;
	color: #ccc;
}

.link-preview {
	background: rgba(255, 255, 255, 0.05);
	padding: 1rem;
	border-radius: 4px;
	word-break: break-all;
	font-family: monospace;
	font-size: 0.9rem;
	margin: 1rem 0;
}

.modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 1rem;
	padding: 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}
</style>
