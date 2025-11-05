<template>
	<div class="activity-page">
		<div class="container-fluid">
			<!-- Header with Stats -->
			<div class="row">
				<div class="col-12">
					<div class="d-flex justify-content-between align-items-center mb-4 pt-3">
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
										<div class="d-flex justify-content-between align-items-start mb-2">
											<div>
												<h5 class="mb-1">
													<span class="badge me-2" :class="getTaskTypeBadge(activity.task_type)">
														{{ formatTaskType(activity.task_type) }}
													</span>
													{{ activity.message }}
												</h5>
												<small>
													<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
													Started: {{ formatDateTime(activity.started_at) }}
													<span v-if="activity.completed_at" class="ms-3">
														<font-awesome-icon :icon="['fas', 'flag-checkered']" class="me-1" />
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
										<small v-if="activity.status === 'running' || activity.status === 'pending'"> Progress: {{ activity.progress }}% </small>

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
	</div>
</template>

<script>
import { activityAPI } from '@/services/api'

export default {
	name: 'ActivityPage',
	data() {
		return {
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
		}
	},
	async mounted() {
		await this.loadStatus()
		await this.loadActivities()
		this.startAutoRefresh()
	},
	beforeUnmount() {
		this.stopAutoRefresh()
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
		startAutoRefresh() {
			// Refresh every 5 seconds
			this.autoRefresh = setInterval(() => {
				this.loadStatus()
				if (this.filters.status === 'running' || this.filters.status === '') {
					this.loadActivities()
				}
			}, 5000)
		},
		stopAutoRefresh() {
			if (this.autoRefresh) {
				clearInterval(this.autoRefresh)
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
	},
}
</script>

<style scoped>
.activity-page {
	min-height: calc(100vh - 60px);
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	color: #fff;
	padding-bottom: 2rem;
}

.stat-card {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 1rem;
	transition: all 0.3s ease;
	backdrop-filter: blur(10px);
}

.stat-card:hover {
	background: rgba(255, 255, 255, 0.1);
	border-color: #00d9ff;
	transform: translateY(-2px);
	box-shadow: 0 5px 15px rgba(0, 217, 255, 0.3);
}

.stat-card .card-body {
	padding: 1.5rem;
}

.stat-card h2 {
	color: #00d9ff;
	font-weight: bold;
}

.filter-bar {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 1rem;
	backdrop-filter: blur(10px);
}

.form-select,
.form-control {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: #fff;
}

.form-select:focus,
.form-control:focus {
	background: rgba(255, 255, 255, 0.15);
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.2rem rgba(0, 217, 255, 0.25);
	color: #fff;
}

.form-select option {
	background: #1a1a2e;
	color: #fff;
}

.form-label {
	color: #00d9ff;
	font-weight: 500;
	margin-bottom: 0.5rem;
}

.activity-timeline {
	position: relative;
}

.activity-item {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 1rem;
	transition: all 0.3s ease;
	backdrop-filter: blur(10px);
}

.activity-item:hover {
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(0, 217, 255, 0.5);
	transform: translateX(5px);
}

.activity-icon {
	width: 50px;
	height: 50px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.2rem;
}

.activity-icon.status-running {
	background: rgba(0, 123, 255, 0.2);
	color: #007bff;
	animation: pulse 2s infinite;
}

.activity-icon.status-pending {
	background: rgba(255, 193, 7, 0.2);
	color: #ffc107;
}

.activity-icon.status-completed {
	background: rgba(40, 167, 69, 0.2);
	color: #28a745;
}

.activity-icon.status-failed {
	background: rgba(220, 53, 69, 0.2);
	color: #dc3545;
}

@keyframes pulse {
	0%,
	100% {
		transform: scale(1);
		opacity: 1;
	}
	50% {
		transform: scale(1.05);
		opacity: 0.8;
	}
}

.progress {
	background: rgba(0, 0, 0, 0.3);
	border-radius: 0.5rem;
}

.details-section {
	background: rgba(0, 0, 0, 0.2);
	border-radius: 0.5rem;
	padding: 0.5rem;
}

.details-content {
	background: rgba(0, 0, 0, 0.4);
	border-radius: 0.5rem;
	border: 1px solid rgba(0, 217, 255, 0.3);
}

.details-content pre {
	color: #00d9ff;
	font-size: 0.85rem;
	max-height: 300px;
	overflow-y: auto;
}

.badge.bg-purple {
	background-color: #6f42c1 !important;
}

.badge.bg-cyan {
	background-color: #00d9ff !important;
	color: #000 !important;
}

.empty-state {
	background: rgba(255, 255, 255, 0.05);
	border: 2px dashed rgba(255, 255, 255, 0.2);
	border-radius: 1rem;
	padding: 3rem;
}

.btn-outline-primary {
	border-color: #00d9ff;
	color: #00d9ff;
}

.btn-outline-primary:hover {
	background-color: #00d9ff;
	color: #000;
}

.btn-outline-danger {
	border-color: #dc3545;
	color: #dc3545;
}

.btn-outline-danger:hover {
	background-color: #dc3545;
	color: #fff;
}

.btn-outline-secondary {
	border-color: rgba(255, 255, 255, 0.3);
	color: #fff;
}

.btn-outline-secondary:hover {
	background-color: rgba(255, 255, 255, 0.1);
	border-color: #00d9ff;
	color: #00d9ff;
}
</style>
