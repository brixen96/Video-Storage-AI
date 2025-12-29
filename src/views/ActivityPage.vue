<template>
	<div class="activity-page">
		<div class="container-fluid">
			<!-- Tab Navigation -->
			<div class="row">
				<div class="col-12">
					<ul class="nav nav-tabs mb-4 pt-3">
						<li class="nav-item">
							<a class="nav-link" :class="{ active: activeTab === 'activities' }" @click="activeTab = 'activities'" href="#">
								<font-awesome-icon :icon="['fas', 'chart-line']" class="me-2" />
								Activities
							</a>
						</li>
						<li class="nav-item">
							<a class="nav-link" :class="{ active: activeTab === 'console' }" @click="activeTab = 'console'" href="#">
								<font-awesome-icon :icon="['fas', 'terminal']" class="me-2" />
								Console Log
							</a>
						</li>
					</ul>
				</div>
			</div>

			<!-- Activities Tab -->
			<div v-if="activeTab === 'activities'">
				<!-- Header with Stats -->
				<div class="row">
					<div class="col-12">
						<div class="d-flex justify-content-between align-items-center mb-4">
							<div>
								<p class="lead mb-0">Real-time system activity and task monitoring</p>
							</div>
							<div class="btn-group">
								<button class="btn btn-outline-primary me-1" @click="loadActivities" :disabled="loading">
									<font-awesome-icon :icon="['fas', 'sync']" :spin="loading" class="me-2" />
									Refresh
								</button>
								<button class="btn btn-outline-danger" @click="confirmCleanOld" :disabled="loading">
									<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
									Clean Old
								</button>
							</div>
						</div>
					</div>
				</div>

			<!-- Status Cards -->
			<div class="row g-3 mb-4">
				<div class="col-md-3">
					<div class="stat-card card h-100">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h6 class="mb-1 text-primary">Running Tasks</h6>
									<h2 class="mb-0">{{ status.running_tasks || 0 }}</h2>
								</div>
								<font-awesome-icon :icon="['fas', 'spinner']" size="2x" class="text-primary" />
							</div>
						</div>
					</div>
				</div>
				<div class="col-md-3">
					<div class="stat-card card h-100">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h6 class="mb-1 text-primary">Pending Tasks</h6>
									<h2 class="mb-0">{{ status.pending_tasks || 0 }}</h2>
								</div>
								<font-awesome-icon :icon="['fas', 'clock']" size="2x" class="text-warning" />
							</div>
						</div>
					</div>
				</div>
				<div class="col-md-3">
					<div class="stat-card card h-100">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h6 class="mb-1 text-primary">Completed</h6>
									<h2 class="mb-0">{{ status.completed_tasks || 0 }}</h2>
								</div>
								<font-awesome-icon :icon="['fas', 'check-circle']" size="2x" class="text-success" />
							</div>
						</div>
					</div>
				</div>
				<div class="col-md-3">
					<div class="stat-card card h-100">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-center">
								<div>
									<h6 class="mb-1 text-primary">Failed</h6>
									<h2 class="mb-0">{{ status.failed_tasks || 0 }}</h2>
								</div>
								<font-awesome-icon :icon="['fas', 'exclamation-circle']" size="2x" class="text-danger" />
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Active Tasks Monitor -->
			<div v-if="runningTasks.length > 0" class="row mb-4">
				<div class="col-12">
					<div class="card">
						<div class="card-header">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-primary" />
								Active Tasks ({{ runningTasks.length }})
							</h5>
						</div>
						<div class="card-body">
							<div class="row g-3">
								<div v-for="task in runningTasks" :key="task.id" class="col-md-6">
									<div class="card task-progress-card">
										<div class="card-body">
											<div class="d-flex justify-content-between align-items-start mb-2">
												<h6 class="mb-0">{{ task.message }}</h6>
												<div class="d-flex gap-2">
													<span class="badge bg-primary">{{ task.status }}</span>
													<button class="btn btn-sm btn-outline-danger" @click="cancelTask(task.id)" title="Cancel Task">
														<font-awesome-icon :icon="['fas', 'times']" />
													</button>
												</div>
											</div>
											<div class="progress mb-2" style="height: 8px">
												<div class="progress-bar progress-bar-striped progress-bar-animated" :style="{ width: task.progress + '%' }" role="progressbar"></div>
											</div>
											<div class="d-flex justify-content-between">
												<small class="text-light">{{ task.progress }}%</small>
												<small class="text-light">{{ formatTaskType(task.task_type) }}</small>
											</div>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Filters -->
			<div class="row mb-3">
				<div class="col-md-12">
					<div class="filter-bar card">
						<div class="card-body">
							<div class="row g-3">
								<div class="col-md-4">
									<label class="form-label">Filter by Status</label>
									<select v-model="filters.status" @change="loadActivities" class="form-select">
										<option value="">All Statuses</option>
										<option value="running">Running</option>
										<option value="pending">Pending</option>
										<option value="completed">Completed</option>
										<option value="failed">Failed</option>
									</select>
								</div>
								<div class="col-md-4">
									<label class="form-label">Filter by Type</label>
									<select v-model="filters.task_type" @change="loadActivities" class="form-select">
										<option value="">All Types</option>
										<option value="scanning">Scanning</option>
										<option value="indexing">Indexing</option>
										<option value="ai_tagging">AI Tagging</option>
										<option value="metadata_fetch">Metadata Fetch</option>
										<option value="thumbnail_generation">Thumbnail Generation</option>
										<option value="video_analysis">Video Analysis</option>
										<option value="file_operation">File Operation</option>
									</select>
								</div>
								<div class="col-md-4">
									<label class="form-label">Items per Page</label>
									<select v-model="filters.limit" @change="loadActivities" class="form-select">
										<option :value="20">20</option>
										<option :value="50">50</option>
										<option :value="100">100</option>
										<option :value="200">200</option>
									</select>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Loading State -->
			<div v-if="loading && activities.length === 0" class="text-center py-5">
				<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" class="text-primary" />
				<p class="mt-3">Loading activities...</p>
			</div>

			<!-- Activity Timeline -->
			<div v-else-if="activities.length > 0" class="row">
				<div class="col-12">
					<div class="activity-timeline">
						<div v-for="activity in activities" :key="activity.id" class="activity-item card mb-3">
							<div class="card-body">
								<div class="row align-items-center">
									<div class="col-auto">
										<div class="activity-icon" :class="getStatusClass(activity.status)">
											<font-awesome-icon :icon="getStatusIcon(activity.status)" size="lg" />
										</div>
									</div>
									<div class="col">
										<div class="d-flex justify-content-between align-items-start mb-2 text-light">
											<div>
												<h5 class="mb-1">
													<span class="badge me-2" :class="getTaskTypeBadge(activity.task_type)">
														{{ formatTaskType(activity.task_type) }}
													</span>
													{{ activity.message }}
												</h5>
												<small>
													<font-awesome-icon :icon="['fas', 'clock']" class="me-1 text-primary" />
													Started: {{ formatDateTime(activity.started_at) }}
													<span v-if="activity.completed_at" class="ms-3">
														<font-awesome-icon :icon="['fas', 'flag-checkered']" class="me-1 text-success" />
														Completed: {{ formatDateTime(activity.completed_at) }}
													</span>
												</small>
											</div>
											<div class="text-end">
												<span class="badge" :class="getStatusBadge(activity.status)">
													{{ activity.status.toUpperCase() }}
												</span>
											</div>
										</div>

										<!-- Progress Bar -->
										<div v-if="activity.status === 'running' || activity.status === 'pending'" class="progress mb-2" style="height: 8px">
											<div
												class="progress-bar progress-bar-striped progress-bar-animated"
												:class="activity.status === 'running' ? 'bg-primary' : 'bg-warning'"
												:style="{ width: activity.progress + '%' }"
											></div>
										</div>
										<small class="text-warning" v-if="activity.status === 'running' || activity.status === 'pending'">
											Progress: {{ activity.progress }}%
										</small>

										<!-- Details -->
										<div v-if="activity.details && Object.keys(activity.details).length > 0" class="details-section mt-2">
											<button class="btn btn-sm btn-outline-secondary" @click="toggleDetails(activity.id)">
												<font-awesome-icon :icon="['fas', 'info-circle']" class="me-1" />
												{{ expandedDetails[activity.id] ? 'Hide' : 'Show' }} Details
											</button>
											<div v-if="expandedDetails[activity.id]" class="details-content mt-2 p-2">
												<pre class="mb-0">{{ JSON.stringify(activity.details, null, 2) }}</pre>
											</div>
										</div>
									</div>
									<div class="col-auto">
										<button class="btn btn-sm btn-outline-danger" @click="confirmDelete(activity)" title="Delete Activity">
											<font-awesome-icon :icon="['fas', 'trash']" />
										</button>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Empty State -->
			<div v-else class="empty-state text-center py-5">
				<font-awesome-icon :icon="['fas', 'chart-line']" size="5x" class="mb-3" />
				<h3>No Activities Found</h3>
				<p class="">No activities match your current filters</p>
			</div>
			</div>
			<!-- End Activities Tab -->

			<!-- Console Log Tab -->
			<div v-if="activeTab === 'console'">
				<div class="row">
					<div class="col-12">
						<div class="d-flex justify-content-between align-items-center mb-4">
							<div>
								<p class="lead mb-0">System console logs from API, AI Companion, and Frontend</p>
							</div>
							<div class="btn-group">
								<button class="btn btn-outline-primary me-1" @click="loadConsoleLogs" :disabled="consoleLoading">
									<font-awesome-icon :icon="['fas', 'sync']" :spin="consoleLoading" class="me-2" />
									Refresh
								</button>
								<button class="btn btn-outline-danger" @click="confirmClearConsoleLogs" :disabled="consoleLoading">
									<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
									Clear Logs
								</button>
							</div>
						</div>
					</div>
				</div>

				<!-- Console Log Filters -->
				<div class="row mb-3">
					<div class="col-md-12">
						<div class="filter-bar card">
							<div class="card-body">
								<div class="row g-3">
									<div class="col-md-3">
										<label class="form-label">Log Source</label>
										<select v-model="consoleFilters.source" @change="loadConsoleLogs" class="form-select">
											<option value="">All Sources</option>
											<option value="api">API</option>
											<option value="ai_companion">AI Companion</option>
											<option value="frontend">Frontend</option>
										</select>
									</div>
									<div class="col-md-3">
										<label class="form-label">Log Level</label>
										<select v-model="consoleFilters.level" @change="loadConsoleLogs" class="form-select">
											<option value="">All Levels</option>
											<option value="debug">Debug</option>
											<option value="info">Info</option>
											<option value="warning">Warning</option>
											<option value="error">Error</option>
										</select>
									</div>
									<div class="col-md-3">
										<label class="form-label">Search</label>
										<input
											v-model="consoleFilters.search"
											@input="loadConsoleLogs"
											type="text"
											class="form-control"
											placeholder="Search logs..."
										/>
									</div>
									<div class="col-md-3">
										<label class="form-label">Items per Page</label>
										<select v-model="consoleFilters.limit" @change="loadConsoleLogs" class="form-select">
											<option :value="50">50</option>
											<option :value="100">100</option>
											<option :value="200">200</option>
											<option :value="500">500</option>
										</select>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Console Log Display -->
				<div class="row">
					<div class="col-12">
						<div class="card console-log-card">
							<div class="card-body p-0">
								<div class="console-log-container">
									<div v-if="consoleLoading && consoleLogs.length === 0" class="text-center py-5">
										<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" class="text-primary" />
										<p class="mt-3">Loading console logs...</p>
									</div>
									<div v-else-if="consoleLogs.length > 0" class="console-log-list">
										<div
											v-for="log in consoleLogs"
											:key="log.id"
											class="console-log-entry"
											:class="'log-level-' + log.level"
										>
											<div class="log-timestamp">{{ formatDateTime(log.created_at) }}</div>
											<div class="log-source">
												<span class="badge" :class="getSourceBadge(log.source)">{{ log.source }}</span>
											</div>
											<div class="log-level">
												<span class="badge" :class="getLevelBadge(log.level)">{{ log.level }}</span>
											</div>
											<div class="log-message">{{ log.message }}</div>
											<div v-if="log.details" class="log-details">
												<button class="btn btn-sm btn-link" @click="toggleLogDetails(log.id)">
													<font-awesome-icon :icon="['fas', expandedLogDetails[log.id] ? 'chevron-up' : 'chevron-down']" />
												</button>
												<div v-if="expandedLogDetails[log.id]" class="log-details-content">
													<pre>{{ JSON.stringify(log.details, null, 2) }}</pre>
												</div>
											</div>
										</div>
									</div>
									<div v-else class="empty-state text-center py-5">
										<font-awesome-icon :icon="['fas', 'terminal']" size="5x" class="mb-3" />
										<h3>No Console Logs</h3>
										<p>No logs match your current filters</p>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
			<!-- End Console Log Tab -->
		</div>
	</div>
</template>

<script>
import { activityAPI, consoleLogAPI } from '@/services/api'
import websocketService from '@/services/websocket'

export default {
	name: 'ActivityPage',
	data() {
		return {
			activeTab: 'activities',
			activities: [],
			status: {},
			loading: false,
			autoRefresh: null,
			filters: {
				status: '',
				task_type: '',
				limit: 50,
			},
			expandedDetails: {},
			failedTaskTimers: {}, // Track timers for auto-dismissing failed tasks
			// Console Log tab data
			consoleLogs: [],
			consoleLoading: false,
			consoleFilters: {
				source: '',
				level: '',
				search: '',
				limit: 100,
			},
			expandedLogDetails: {},
		}
	},
	computed: {
		runningTasks() {
			return this.activities.filter((activity) => activity.status === 'running')
		},
	},
	async mounted() {
		await this.loadStatus()
		await this.loadActivities()

		websocketService.connect()

		this.unsubscribeStatus = websocketService.on('status_update', (status) => {
			this.status = status
		})

		this.unsubscribeActivity = websocketService.on('activity_update', (activity) => {
			const index = this.activities.findIndex((a) => a.id === activity.id)
			if (index !== -1) {
				this.activities.splice(index, 1, activity)
			} else {
				this.activities.unshift(activity)
			}

			// Auto-dismiss failed tasks after 10 seconds
			if (activity.status === 'failed') {
				// Clear any existing timer for this activity
				if (this.failedTaskTimers[activity.id]) {
					clearTimeout(this.failedTaskTimers[activity.id])
				}

				// Set new timer to auto-dismiss after 10 seconds
				this.failedTaskTimers[activity.id] = setTimeout(() => {
					const idx = this.activities.findIndex((a) => a.id === activity.id)
					if (idx !== -1) {
						this.activities.splice(idx, 1)
						this.$toast.info('Failed task auto-dismissed', `"${activity.message}" has been automatically removed`)
					}
					delete this.failedTaskTimers[activity.id]
				}, 10000)
			}
		})
	},
	beforeUnmount() {
		if (this.unsubscribeStatus) {
			this.unsubscribeStatus()
		}
		if (this.unsubscribeActivity) {
			this.unsubscribeActivity()
		}
		// Clean up all failed task timers
		Object.values(this.failedTaskTimers).forEach((timer) => clearTimeout(timer))
	},
	methods: {
		async loadActivities() {
			this.loading = true
			try {
				const response = await activityAPI.getAll(this.filters)
				this.activities = response.data || []
			} catch (error) {
				console.error('Failed to load activities:', error)
			} finally {
				this.loading = false
			}
		},
		async loadStatus() {
			try {
				const response = await activityAPI.getStatus()
				this.status = response.data || {}
			} catch (error) {
				console.error('Failed to load status:', error)
			}
		},
		async deleteActivity(id) {
			try {
				await activityAPI.delete(id)
				await this.loadActivities()
				await this.loadStatus()
			} catch (error) {
				console.error('Failed to delete activity:', error)
				alert('Failed to delete activity. Please try again.')
			}
		},
		confirmDelete(activity) {
			if (confirm(`Are you sure you want to delete this activity?\n\n"${activity.message}"`)) {
				this.deleteActivity(activity.id)
			}
		},
		async confirmCleanOld() {
			const days = prompt('Delete activities older than how many days?', '30')
			if (days !== null && !isNaN(days) && days > 0) {
				try {
					const response = await activityAPI.cleanOld(parseInt(days))
					alert(`Successfully deleted ${response.data.deleted_count} old activities`)
					await this.loadActivities()
					await this.loadStatus()
				} catch (error) {
					console.error('Failed to clean old activities:', error)
					alert('Failed to clean old activities. Please try again.')
				}
			}
		},
		toggleDetails(id) {
			this.expandedDetails[id] = !this.expandedDetails[id]
			this.$forceUpdate()
		},
		async cancelTask(id) {
			if (confirm('Are you sure you want to cancel this task?')) {
				try {
					// For now, mark as failed with cancellation message
					// TODO: Implement proper task cancellation in backend
					const activity = this.activities.find((a) => a.id === id)
					if (activity) {
						await activityAPI.update(id, {
							status: 'failed',
							message: `${activity.message} (Cancelled by user)`,
							progress: activity.progress,
						})
						await this.loadActivities()
						await this.loadStatus()
						this.$toast.success('Task cancelled successfully')
					}
				} catch (error) {
					console.error('Failed to cancel task:', error)
					this.$toast.error('Failed to cancel task. Please try again.')
				}
			}
		},

		getStatusIcon(status) {
			const icons = {
				running: ['fas', 'spinner'],
				pending: ['fas', 'clock'],
				completed: ['fas', 'check-circle'],
				failed: ['fas', 'exclamation-circle'],
			}
			return icons[status] || ['fas', 'question-circle']
		},
		getStatusClass(status) {
			return `status-${status}`
		},
		getStatusBadge(status) {
			const badges = {
				running: 'bg-primary',
				pending: 'bg-warning',
				completed: 'bg-success',
				failed: 'bg-danger',
			}
			return badges[status] || 'bg-secondary'
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
		formatTaskType(taskType) {
			return taskType
				.split('_')
				.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
				.join(' ')
		},
		formatDateTime(dateString) {
			const date = new Date(dateString)
			return date.toLocaleString()
		},

		// Console Log methods
		async loadConsoleLogs() {
			this.consoleLoading = true
			try {
				const params = {
					page: 1,
					limit: this.consoleFilters.limit,
				}

				if (this.consoleFilters.source) {
					params.source = this.consoleFilters.source
				}
				if (this.consoleFilters.level) {
					params.level = this.consoleFilters.level
				}
				if (this.consoleFilters.search) {
					params.search = this.consoleFilters.search
				}

				const response = await consoleLogAPI.getAll(params)
				this.consoleLogs = response.data || []
			} catch (error) {
				console.error('Failed to load console logs:', error)
				this.$toast.error('Failed to load console logs')
			} finally {
				this.consoleLoading = false
			}
		},
		async confirmClearConsoleLogs() {
			if (confirm('Are you sure you want to clear all console logs?')) {
				try {
					await consoleLogAPI.clearAll()
					this.consoleLogs = []
					this.$toast.success('Console logs cleared successfully')
					await this.loadConsoleLogs()
				} catch (error) {
					console.error('Failed to clear console logs:', error)
					this.$toast.error('Failed to clear console logs')
				}
			}
		},
		toggleLogDetails(id) {
			this.expandedLogDetails[id] = !this.expandedLogDetails[id]
			this.$forceUpdate()
		},
		getSourceBadge(source) {
			const badges = {
				api: 'bg-primary',
				ai_companion: 'bg-purple',
				frontend: 'bg-info',
			}
			return badges[source] || 'bg-secondary'
		},
		getLevelBadge(level) {
			const badges = {
				debug: 'bg-secondary',
				info: 'bg-info',
				warning: 'bg-warning',
				error: 'bg-danger',
			}
			return badges[level] || 'bg-secondary'
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/activity_page.css';
</style>
