<template>
	<div class="activity-status-widget" v-click-outside="closeDetails">
		<div class="status-indicator" @click="toggleDetails" :class="statusColor">
			<div class="status-dot"></div>
			<span class="status-label">{{ statusLabel }}</span>
			<span v-if="wsConnected" class="ws-indicator" title="Real-time updates active">
				<i class="bi bi-broadcast"></i>
			</span>
		</div>

		<div v-if="showDetails" class="status-details">
			<div class="details-body">
				<!-- Connection Status -->
				<div v-if="!wsConnected" class="connection-status mb-3">
					<small class="text-danger">Connecting...</small>
				</div>

				<!-- Current Activity -->
				<div v-if="currentActivity && wsConnected" class="current-activity mb-">
					<div class="d-flex justify-content-between align-items-center mb-2">
						<span class="badge" :class="getTaskTypeBadge(currentActivity.task_type)">
							{{ formatTaskType(currentActivity.task_type) }}
						</span>
						<span class="badge" :class="'bg-' + statusColor">{{ currentActivity.status }}</span>
					</div>
					<p class="mb-1">{{ currentActivity.message }}</p>
					<div v-if="currentActivity.progress !== undefined" class="progress" style="height: 4px">
						<div class="progress-bar" :class="'bg-' + statusColor" :style="{ width: currentActivity.progress + '%' }"></div>
					</div>
					<small>Started: {{ formatTime(currentActivity.started_at) }}</small>
				</div>

				<!-- Idle State -->
				<div v-else class="text-center">
					<i class="bi bi-check-circle fs-3"></i>
					<p class="text-primary">{{ idleMessage }}</p>
				</div>

				<!-- Status Summary -->
				<div class="status-summary">
					<div class="row g-2 text-light">
						<div class="col-6">
							<div class="stat-box">
								<div class="stat-value text-warning">{{ status.running_tasks || 0 }}</div>
								<div class="stat-label">Running</div>
							</div>
						</div>
						<div class="col-6">
							<div class="stat-box">
								<div class="stat-value text-info">{{ status.pending_tasks || 0 }}</div>
								<div class="stat-label">Pending</div>
							</div>
						</div>
						<div class="col-6">
							<div class="stat-box">
								<div class="stat-value text-success">{{ status.completed_tasks || 0 }}</div>
								<div class="stat-label">Completed</div>
							</div>
						</div>
						<div class="col-6">
							<div class="stat-box">
								<div class="stat-value text-danger">{{ status.failed_tasks || 0 }}</div>
								<div class="stat-label">Failed</div>
							</div>
						</div>
					</div>
				</div>

				<!-- View All Link -->
				<div class="text-center mt-3">
					<router-link to="/activity" class="btn btn-sm btn-outline-primary" @click="closeDetails"> View All Activities </router-link>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { activityAPI } from '@/services/api'
import websocketService from '@/services/websocket'

export default {
	name: 'ActivityStatusWidget',
	data() {
		return {
			status: {},
			currentActivity: null,
			showDetails: false,
			wsConnected: false,
			unsubscribeWs: null,
			unsubscribeStatus: null,
			unsubscribeConnection: null,
		}
	},
	computed: {
		statusColor() {
			if (this.status.failed_tasks > 0 || (this.currentActivity && this.currentActivity.status === 'failed')) {
				return 'red'
			}
			if (this.status.running_tasks > 0 || this.status.pending_tasks > 0) {
				return 'yellow'
			}
			return 'green'
		},
		statusLabel() {
			if (this.statusColor === 'red') return 'Error'
			if (this.statusColor === 'yellow') return 'Processing'
			return 'Idle'
		},
		idleMessage() {
			return 'No active tasks'
		},
	},
	directives: {
		clickOutside: {
			mounted(el, binding) {
				el.clickOutsideEvent = function (event) {
					if (!(el === event.target || el.contains(event.target))) {
						binding.value(event)
					}
				}
				document.addEventListener('click', el.clickOutsideEvent)
			},
			unmounted(el) {
				document.removeEventListener('click', el.clickOutsideEvent)
			},
		},
	},
	async mounted() {
		// Load initial status
		await this.loadStatus()

		// Connect to WebSocket
		websocketService.connect()

		// Subscribe to WebSocket events
		this.unsubscribeConnection = websocketService.on('connected', (data) => {
			this.wsConnected = data.connected
			if (data.connected) {
				// Reload status when connected
				this.loadStatus()
			}
		})

		this.unsubscribeStatus = websocketService.on('status_update', (data) => {
			this.status = data
			this.updateCurrentActivity()
		})

		this.unsubscribeWs = websocketService.on('activity_update', (data) => {
			console.log('Activity update received:', data)
			// Update current activity if it matches
			if (this.currentActivity && this.currentActivity.id === data.id) {
				this.currentActivity = data
			}
			// Reload status to get updated counts
			this.loadStatus()
		})

		// Check initial connection state
		this.wsConnected = websocketService.isConnected()
	},
	beforeUnmount() {
		// Unsubscribe from WebSocket events
		if (this.unsubscribeWs) this.unsubscribeWs()
		if (this.unsubscribeStatus) this.unsubscribeStatus()
		if (this.unsubscribeConnection) this.unsubscribeConnection()

		// Disconnect WebSocket
		websocketService.disconnect()
	},
	methods: {
		async loadStatus() {
			try {
				const response = await activityAPI.getStatus()
				this.status = response.data || {}

				// Get the current running/pending task
				if (this.status.current_tasks && this.status.current_tasks.length > 0) {
					this.currentActivity = this.status.current_tasks[0]
				} else {
					// Check for failed tasks
					try {
						const failedResponse = await activityAPI.getAll({ status: 'failed', limit: 1 })
						if (failedResponse.data && failedResponse.data.length > 0) {
							this.currentActivity = failedResponse.data[0]
						} else {
							this.currentActivity = null
						}
					} catch (error) {
						this.currentActivity = null
					}
				}
			} catch (error) {
				console.error('Failed to load status:', error)
			}
		},
		updateCurrentActivity() {
			// Update current activity based on status
			if (this.status.current_tasks && this.status.current_tasks.length > 0) {
				this.currentActivity = this.status.current_tasks[0]
			} else {
				this.currentActivity = null
			}
		},
		toggleDetails() {
			this.showDetails = !this.showDetails
		},
		closeDetails() {
			this.showDetails = false
		},
		formatTaskType(taskType) {
			return taskType
				.split('_')
				.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
				.join(' ')
		},
		formatTime(dateString) {
			const date = new Date(dateString)
			return date.toLocaleTimeString()
		},
		getTaskTypeBadge(taskType) {
			const badges = {
				scanning: 'bg-info',
				indexing: 'bg-primary',
				ai_tagging: 'bg-purple',
				metadata_fetch: 'bg-cyan',
				thumbnail_generation: 'bg-warning',
				video_analysis: 'bg-success',
				file_operation: 'bg-secondary',
			}
			return badges[taskType] || 'bg-dark'
		},
	},
}
</script>

<style scoped>
.activity-status-widget {
	position: relative;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 0.75rem;
	backdrop-filter: blur(10px);
	transition: all 0.3s ease;
	min-width: 250px;
}

.activity-status-widget:hover {
	background: rgba(255, 255, 255, 0.08);
}

.status-indicator {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	cursor: pointer;
	user-select: none;
	padding: 0.5rem;
	border-radius: 0.5rem;
	transition: all 0.3s ease;
}

.status-indicator:hover {
	background: rgba(255, 255, 255, 0.1);
}

.status-dot {
	width: 8px;
	height: 8px;
	border-radius: 50%;
	transition: all 0.3s ease;
}

.status-indicator.green .status-dot {
	background: #28a745;
	box-shadow: 0 0 8px rgba(40, 167, 69, 0.5);
}

.status-indicator.yellow .status-dot {
	background: #ffc107;
	box-shadow: 0 0 8px rgba(255, 193, 7, 0.5);
}

.status-indicator.red .status-dot {
	background: #dc3545;
	box-shadow: 0 0 8px rgba(220, 53, 69, 0.5);
}

.status-label {
	font-weight: 600;
	font-size: 0.875rem;
	color: #00d9ff;
}

.ws-indicator {
	font-size: 0.75rem;
	color: #28a745;
}

.status-details {
	position: absolute;
	top: 100%;
	left: 0;
	right: 0;
	z-index: 1000;
	background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%) !important;
	border-radius: 0.75rem;
	border: 1px solid rgba(255, 255, 255, 0.1);
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
	margin-top: 0.5rem;
	overflow: hidden;
}

.details-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.75rem 1rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	background: rgba(0, 0, 0, 0.2);
}

.details-body {
	padding: 1rem;
}

.connection-status {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 0.75rem;
}

.current-activity {
	background: rgba(0, 0, 0, 0.2);
	border-radius: 0.5rem;
	padding: 0.75rem;
}

.stat-box {
	text-align: center;
	padding: 0.5rem;
	border-radius: 0.5rem;
	background: rgba(0, 0, 0, 0.2);
}

.stat-value {
	font-size: 1.25rem;
	font-weight: bold;
	margin-bottom: 0.25rem;
}

.stat-label {
	font-size: 0.75rem;
	opacity: 0.7;
}

.btn-outline-primary {
	border-color: rgba(0, 217, 255, 0.5);
	color: #00d9ff;
	font-size: 0.75rem;
}

.btn-outline-primary:hover {
	background-color: #00d9ff;
	border-color: #00d9ff;
	color: #000;
}

/* Responsive adjustments */
@media (max-width: 768px) {
	.activity-status-widget {
		min-width: 200px;
	}

	.status-details {
		left: 50%;
		transform: translateX(-50%);
		max-width: 90vw;
	}
}
</style>
