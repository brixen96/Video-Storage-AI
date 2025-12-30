<template>
	<div v-if="activities.length > 0" class="activity-tracker">
		<div class="activity-header" @click="toggleExpanded">
			<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2" />
			<span>{{ activities.length }} {{ activities.length === 1 ? 'task' : 'tasks' }} running</span>
			<font-awesome-icon :icon="['fas', expanded ? 'chevron-down' : 'chevron-up']" class="ms-auto" />
		</div>
		<div v-if="expanded" class="activity-list">
			<div v-for="activity in activities" :key="activity.id" class="activity-item">
				<div class="activity-icon">
					<font-awesome-icon v-if="activity.task_type === 'video_conversion'" :icon="['fas', 'sync']" spin />
					<font-awesome-icon v-else :icon="['fas', 'spinner']" spin />
				</div>
				<div class="activity-details">
					<div class="activity-message">{{ activity.message }}</div>
					<div class="activity-time">{{ formatTime(activity.created_at) }}</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { activityAPI } from '@/services/api'
import websocketService from '@/services/websocket'

export default {
	name: 'ActivityTracker',
	data() {
		return {
			activities: [],
			expanded: true,
			unsubscribeActivity: null,
		}
	},
	mounted() {
		// Initial fetch
		this.fetchActivities()

		// Subscribe to WebSocket updates instead of polling
		this.unsubscribeActivity = websocketService.on('activity_update', (data) => {
			this.handleActivityUpdate(data)
		})

		// Connect WebSocket if not already connected
		if (!websocketService.isConnected()) {
			websocketService.connect()
		}
	},
	beforeUnmount() {
		// Unsubscribe from WebSocket
		if (this.unsubscribeActivity) {
			this.unsubscribeActivity()
		}
	},
	methods: {
		async fetchActivities() {
			try {
				const response = await activityAPI.getAll({ status: 'running' })
				this.activities = response.data || []
			} catch (error) {
				// Silently fail - WebSocket will provide updates
			}
		},
		handleActivityUpdate(data) {
			// Handle real-time activity updates from WebSocket
			if (data.status === 'running') {
				// Add or update activity
				const index = this.activities.findIndex(a => a.id === data.id)
				if (index >= 0) {
					this.activities[index] = data
				} else {
					this.activities.push(data)
				}
			} else {
				// Remove completed/failed activities
				this.activities = this.activities.filter(a => a.id !== data.id)
			}
		},
		toggleExpanded() {
			this.expanded = !this.expanded
		},
		formatTime(timestamp) {
			if (!timestamp) return ''
			const date = new Date(timestamp)
			const now = new Date()
			const diff = Math.floor((now - date) / 1000) // seconds

			if (diff < 60) return 'Just now'
			if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
			if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
			return date.toLocaleDateString()
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/activity_tracker.css';
</style>
