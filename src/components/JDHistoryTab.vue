<template>
	<div>
		<div class="d-flex justify-content-between align-items-center mb-3">
			<span class="text-muted">Links sent to JDownloader from this app</span>
			<button class="btn btn-outline-danger btn-sm" @click="clearHistory" :disabled="history.length === 0">
				<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
				Clear History
			</button>
		</div>

		<div v-if="history.length === 0" class="empty-state">
			<font-awesome-icon :icon="['fas', 'history']" class="empty-icon" />
			<p>No history yet</p>
			<p class="text-muted small">Links sent to JDownloader will appear here</p>
		</div>

		<div v-else class="history-list">
			<div v-for="item in history" :key="item.id" class="history-item">
				<div class="history-info">
					<div class="history-title">{{ item.videoTitle || item.linkUrl }}</div>
					<div class="history-meta">
						<span class="package">{{ item.packageName || 'No package' }}</span>
						<span class="time">{{ formatDate(item.sentAt) }}</span>
					</div>
				</div>
				<span :class="['status-badge', getHistoryStatusClass(item.status)]">{{ item.status }}</span>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { jdownloaderAPI } from '@/services/api'

const history = ref([])

const loadHistory = async () => {
	try {
		const result = await jdownloaderAPI.getHistory(100, 0)
		history.value = result.items || result || []
	} catch (error) {
		console.error('Failed to load history:', error)
		history.value = []
	}
}

const clearHistory = async () => {
	if (!confirm('Are you sure you want to clear the history?')) return
	try {
		await jdownloaderAPI.clearHistory()
		history.value = []
	} catch (error) {
		console.error('Failed to clear history:', error)
	}
}

const formatDate = (dateString) => {
	if (!dateString) return ''
	return new Date(dateString).toLocaleString()
}

const getHistoryStatusClass = (status) => {
	if (!status) return 'secondary'
	const lower = status.toLowerCase()
	if (lower === 'sent') return 'success'
	if (lower === 'failed') return 'danger'
	if (lower === 'queued') return 'warning'
	return 'secondary'
}

onMounted(loadHistory)
</script>

<style scoped>
.empty-state {
	text-align: center;
	padding: 3rem 1rem;
	color: rgba(255, 255, 255, 0.6);
}

.empty-state .empty-icon {
	font-size: 3rem;
	margin-bottom: 1rem;
	opacity: 0.5;
}

.history-list {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.history-item {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.875rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 8px;
	border: 1px solid rgba(255, 255, 255, 0.08);
}

.history-info {
	flex: 1;
	min-width: 0;
}

.history-title {
	font-size: 0.9rem;
	color: rgba(255, 255, 255, 0.9);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.history-meta {
	display: flex;
	gap: 1rem;
	margin-top: 0.25rem;
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
}

.status-badge {
	padding: 0.2rem 0.5rem;
	border-radius: 4px;
	font-size: 0.7rem;
	font-weight: 500;
	text-transform: uppercase;
}

.status-badge.success {
	background: rgba(76, 175, 80, 0.2);
	color: #4caf50;
}

.status-badge.danger {
	background: rgba(244, 67, 54, 0.2);
	color: #f44336;
}

.status-badge.warning {
	background: rgba(255, 193, 7, 0.2);
	color: #ffc107;
}

.status-badge.secondary {
	background: rgba(158, 158, 158, 0.2);
	color: #9e9e9e;
}
</style>
