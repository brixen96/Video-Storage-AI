<template>
	<div v-if="activities.length > 0" class="activity-tracker">
		<div class="activity-header" @click="toggleExpanded">
			<font-awesome-icon v-if="!hasPausedTasks" :icon="['fas', 'spinner']" spin class="me-2" />
			<font-awesome-icon v-else :icon="['fas', 'pause']" class="me-2 text-warning" />
			<span>{{ statusText }}</span>
			<font-awesome-icon :icon="['fas', expanded ? 'chevron-down' : 'chevron-up']" class="ms-auto" />
		</div>
		<div v-if="expanded" class="activity-list">
			<div v-for="activity in activities" :key="activity.id" class="activity-item">
				<div class="activity-icon">
					<font-awesome-icon v-if="activity.is_paused" :icon="['fas', 'pause']" class="text-warning" />
					<font-awesome-icon v-else-if="activity.task_type === 'video_conversion'" :icon="['fas', 'sync']" spin />
					<font-awesome-icon v-else :icon="['fas', 'spinner']" spin />
				</div>
				<div class="activity-details">
					<div class="activity-message">
						{{ activity.message }}
						<span v-if="activity.is_paused" class="badge bg-warning text-dark ms-2">Paused</span>
					</div>
					<div class="activity-time">{{ formatTime(activity.created_at) }}</div>
				</div>
				<div class="activity-actions">
					<button
						v-if="!activity.is_paused"
						class="btn btn-sm btn-outline-warning"
						@click="pauseActivity(activity.id)"
						:disabled="pausingId === activity.id"
						title="Pause task"
					>
						<font-awesome-icon :icon="['fas', pausingId === activity.id ? 'spinner' : 'pause']" :spin="pausingId === activity.id" />
					</button>
					<button
						v-else
						class="btn btn-sm btn-outline-success"
						@click="resumeActivity(activity.id)"
						:disabled="resumingId === activity.id"
						title="Resume task"
					>
						<font-awesome-icon :icon="['fas', resumingId === activity.id ? 'spinner' : 'play']" :spin="resumingId === activity.id" />
					</button>
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
			pollInterval: null,
			pausingId: null,
			resumingId: null,
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

		// Poll every 10 seconds to clean up stale tasks
		this.pollInterval = setInterval(() => {
			if (this.activities.length > 0) {
				this.fetchActivities()
			}
		}, 10000)
	},
	beforeUnmount() {
		// Unsubscribe from WebSocket
		if (this.unsubscribeActivity) {
			this.unsubscribeActivity()
		}
		// Clear polling interval
		if (this.pollInterval) {
			clearInterval(this.pollInterval)
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
					// Use Vue.set or spread to ensure reactivity
					this.activities.splice(index, 1, { ...data })
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
		async pauseActivity(id) {
			// Prevent duplicate requests
			if (this.pausingId === id) {
				return
			}

			this.pausingId = id
			try {
				await activityAPI.pause(id)
				this.$toast.success('Task paused successfully', 'You can resume it later')

				// Update the activity in the list immediately to show paused state
				const index = this.activities.findIndex(a => a.id === id)
				if (index >= 0) {
					this.activities.splice(index, 1, { ...this.activities[index], is_paused: true })
				}

				// Refetch to ensure UI updates with server state
				await this.fetchActivities()
			} catch (error) {
				console.error('Failed to pause activity:', error)
				this.$toast.error('Failed to pause task', error.response?.data?.error || 'Please try again')
			} finally {
				this.pausingId = null
			}
		},
		async resumeActivity(id) {
			// Prevent duplicate requests
			if (this.resumingId === id) {
				return
			}

			this.resumingId = id
			try {
				const response = await activityAPI.resume(id)
				this.$toast.success('Task resumed', response.data?.message || 'Task will continue from where it left off')

				// Update the activity in the list immediately to show resumed state
				const index = this.activities.findIndex(a => a.id === id)
				if (index >= 0) {
					this.activities.splice(index, 1, { ...this.activities[index], is_paused: false })
				}

				// Refetch to ensure UI updates with server state
				await this.fetchActivities()
			} catch (error) {
				console.error('Failed to resume activity:', error)
				this.$toast.error('Failed to resume task', error.response?.data?.error || 'Please try again')
			} finally {
				this.resumingId = null
			}
		},
	},
	computed: {
		hasPausedTasks() {
			return this.activities.some(a => a.is_paused)
		},
		runningCount() {
			return this.activities.filter(a => !a.is_paused).length
		},
		pausedCount() {
			return this.activities.filter(a => a.is_paused).length
		},
		statusText() {
			const running = this.runningCount
			const paused = this.pausedCount

			if (running > 0 && paused > 0) {
				return `${running} running, ${paused} paused`
			} else if (paused > 0) {
				return `${paused} ${paused === 1 ? 'task' : 'tasks'} paused`
			} else {
				return `${running} ${running === 1 ? 'task' : 'tasks'} running`
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/activity_tracker.css';
</style>
