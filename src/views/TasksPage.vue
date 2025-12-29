<template>
	<div class="tasks-page">
		<div class="container-fluid py-4">
			<div class="page-header mb-4">
				<h1>
					<font-awesome-icon :icon="['fas', 'tasks']" class="me-3" />
					Task Center
				</h1>
				<p class="text-light">Real-time task monitoring with instant feedback</p>
			</div>

			<!-- Task Categories -->
			<div class="row g-3">
				<!-- Library Tasks -->
				<div class="col-md-6">
					<div class="card task-card h-100">
						<div class="card-header">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
								Library Tasks
							</h5>
						</div>
						<div class="card-body">
							<!-- Scan All Libraries -->
							<div class="task-item">
								<div class="task-info">
									<h6>Scan All Libraries</h6>
									<p class="text-light mb-0">Scan all libraries for new or changed videos</p>
								</div>
								<button class="btn btn-primary mt-3" @click="scanAllLibraries" :disabled="isTaskRunning('library_scan_all')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('library_scan_all') ? 'spinner' : 'sync']"
										:spin="isTaskRunning('library_scan_all')"
										class="me-2"
									/>
									{{ isTaskRunning('library_scan_all') ? 'Scanning...' : 'Scan Libraries' }}
								</button>
								<div v-if="getTaskProgress('library_scan_all')" class="task-feedback mt-2">
									<div class="progress" style="height: 6px">
										<div class="progress-bar bg-primary" :style="{ width: getTaskProgress('library_scan_all') + '%' }"></div>
									</div>
									<small class="text-primary">{{ getTaskProgress('library_scan_all') }}%</small>
								</div>
							</div>
							<hr class="my-4" />

							<!-- Generate Previews -->
							<div class="task-item">
								<div class="task-info">
									<h6>Generate Previews</h6>
									<p class="text-light mb-0">Generate hover preview storyboards for all videos</p>
								</div>
								<button class="btn btn-info mt-3" @click="generatePreviews" :disabled="isTaskRunning('preview_generation')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('preview_generation') ? 'spinner' : 'images']"
										:spin="isTaskRunning('preview_generation')"
										class="me-2"
									/>
									{{ isTaskRunning('preview_generation') ? 'Generating...' : 'Generate Previews' }}
								</button>
								<div v-if="getTaskProgress('preview_generation')" class="task-feedback mt-2">
									<div class="progress" style="height: 6px">
										<div class="progress-bar bg-info" :style="{ width: getTaskProgress('preview_generation') + '%' }"></div>
									</div>
									<small class="text-info">{{ getTaskProgress('preview_generation') }}%</small>
								</div>
							</div>
							<hr class="my-4" />

							<!-- Generate Video Thumbnails -->
							<div class="task-item">
								<div class="task-info">
									<h6>Generate Video Thumbnails</h6>
									<p class="text-light mb-0">Generate static thumbnails for all videos</p>
								</div>
								<button class="btn btn-success mt-3" @click="generateVideoThumbnails" :disabled="isTaskRunning('video_thumbnail_generation')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('video_thumbnail_generation') ? 'spinner' : 'file-image']"
										:spin="isTaskRunning('video_thumbnail_generation')"
										class="me-2"
									/>
									{{ isTaskRunning('video_thumbnail_generation') ? 'Generating...' : 'Generate Video Thumbnails' }}
								</button>
								<div v-if="getTaskProgress('video_thumbnail_generation')" class="task-feedback mt-2">
									<div class="progress" style="height: 6px">
										<div class="progress-bar bg-success" :style="{ width: getTaskProgress('video_thumbnail_generation') + '%' }"></div>
									</div>
									<small class="text-success">{{ getTaskProgress('video_thumbnail_generation') }}%</small>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Performer Tasks -->
				<div class="col-md-6">
					<div class="card task-card h-100">
						<div class="card-header">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'user']" class="me-2" />
								Performer Tasks
							</h5>
						</div>
						<div class="card-body">
							<!-- Scan Performers -->
							<div class="task-item">
								<div class="task-info">
									<h6>Scan Performers</h6>
									<p class="text-light mb-0">Scan performer folders and create previews</p>
								</div>
								<button class="btn btn-primary mt-3" @click="scanPerformers" :disabled="isTaskRunning('performer_scan')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('performer_scan') ? 'spinner' : 'user-plus']"
										:spin="isTaskRunning('performer_scan')"
										class="me-2"
									/>
									{{ isTaskRunning('performer_scan') ? 'Scanning...' : 'Scan Performers' }}
								</button>
								<div v-if="getTaskProgress('performer_scan')" class="task-feedback mt-2">
									<div class="progress" style="height: 6px">
										<div class="progress-bar bg-primary" :style="{ width: getTaskProgress('performer_scan') + '%' }"></div>
									</div>
									<small class="text-primary">{{ getTaskProgress('performer_scan') }}%</small>
								</div>
							</div>
							<hr class="my-4" />

							<!-- Generate Performer Thumbnails -->
							<div class="task-item">
								<div class="task-info">
									<h6>Generate Performer Thumbnails</h6>
									<p class="text-light mb-0">Generate thumbnails from performer preview videos</p>
								</div>
								<button class="btn btn-success mt-3" @click="generatePerformerThumbnails" :disabled="isTaskRunning('performer_thumbnail_generation')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('performer_thumbnail_generation') ? 'spinner' : 'image']"
										:spin="isTaskRunning('performer_thumbnail_generation')"
										class="me-2"
									/>
									{{ isTaskRunning('performer_thumbnail_generation') ? 'Generating...' : 'Generate Thumbnails' }}
								</button>
								<div v-if="getTaskProgress('performer_thumbnail_generation')" class="task-feedback mt-2">
									<div class="progress" style="height: 6px">
										<div class="progress-bar bg-success" :style="{ width: getTaskProgress('performer_thumbnail_generation') + '%' }"></div>
									</div>
									<small class="text-success">{{ getTaskProgress('performer_thumbnail_generation') }}%</small>
								</div>
							</div>
							<hr class="my-4" />

							<!-- Fetch Metadata -->
							<div class="task-item">
								<div class="task-info">
									<h6>Fetch Metadata</h6>
									<p class="text-light mb-0">Fetch metadata for all performers from AdultDataLink</p>
								</div>
								<button class="btn btn-warning mt-3" @click="fetchAllMetadata" :disabled="isTaskRunning('metadata_fetch')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('metadata_fetch') ? 'spinner' : 'download']"
										:spin="isTaskRunning('metadata_fetch')"
										class="me-2"
									/>
									{{ isTaskRunning('metadata_fetch') ? 'Fetching...' : 'Fetch Metadata' }}
								</button>
								<div v-if="getTaskProgress('metadata_fetch')" class="task-feedback mt-2">
									<div class="progress" style="height: 6px">
										<div class="progress-bar bg-warning" :style="{ width: getTaskProgress('metadata_fetch') + '%' }"></div>
									</div>
									<small class="text-warning">{{ getTaskProgress('metadata_fetch') }}%</small>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Database Info -->
			<div class="row g-3 mt-3">
				<div class="col-12">
					<div class="card task-card">
						<div class="card-header">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'database']" class="me-2" />
								Database Information
							</h5>
						</div>
						<div class="card-body">
							<div class="row g-3">
								<div class="col-md-3">
									<div class="stat-item">
										<label>Videos</label>
										<h4>{{ databaseInfo.videoCount.toLocaleString() }}</h4>
									</div>
								</div>
								<div class="col-md-3">
									<div class="stat-item">
										<label>Performers</label>
										<h4>{{ databaseInfo.performerCount.toLocaleString() }}</h4>
									</div>
								</div>
								<div class="col-md-3">
									<div class="stat-item">
										<label>Studios</label>
										<h4>{{ databaseInfo.studioCount.toLocaleString() }}</h4>
									</div>
								</div>
								<div class="col-md-3">
									<div class="stat-item">
										<label>Tags</label>
										<h4>{{ databaseInfo.tagCount.toLocaleString() }}</h4>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Database Operations -->
			<div class="row g-3 mt-3">
				<div class="col-12">
					<div class="card task-card">
						<div class="card-header">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'tools']" class="me-2" />
								Database Operations
							</h5>
						</div>
						<div class="card-body">
							<div class="row g-3">
								<!-- Optimize Database -->
								<div class="col-md-6">
									<div class="task-item">
										<div class="task-info">
											<h6>Optimize Database</h6>
											<p class="text-light mb-0">Run VACUUM to optimize database performance and reclaim space</p>
										</div>
										<button class="btn btn-primary mt-3" @click="optimizeDatabase" :disabled="isDbOperationRunning">
											<font-awesome-icon :icon="['fas', isDbOperationRunning ? 'spinner' : 'wrench']" :spin="isDbOperationRunning" class="me-2" />
											{{ isDbOperationRunning ? 'Optimizing...' : 'Optimize Database' }}
										</button>
									</div>
								</div>

								<!-- Backup Database -->
								<div class="col-md-6">
									<div class="task-item">
										<div class="task-info">
											<h6>Backup Database</h6>
											<p class="text-light mb-0">Create a backup of the entire database</p>
										</div>
										<button class="btn btn-success mt-3" @click="backupDatabase" :disabled="isDbOperationRunning">
											<font-awesome-icon :icon="['fas', isDbOperationRunning ? 'spinner' : 'save']" :spin="isDbOperationRunning" class="me-2" />
											{{ isDbOperationRunning ? 'Backing Up...' : 'Backup Database' }}
										</button>
									</div>
								</div>

								<!-- Restore Database -->
								<div class="col-md-6">
									<div class="task-item">
										<div class="task-info">
											<h6>Restore Database</h6>
											<p class="text-light mb-0">Restore database from a backup file</p>
										</div>
										<button class="btn btn-warning mt-3" @click="restoreDatabase" :disabled="isDbOperationRunning">
											<font-awesome-icon :icon="['fas', isDbOperationRunning ? 'spinner' : 'upload']" :spin="isDbOperationRunning" class="me-2" />
											{{ isDbOperationRunning ? 'Restoring...' : 'Restore Database' }}
										</button>
									</div>
								</div>

								<!-- Clear All Activities -->
								<div class="col-md-6">
									<div class="task-item">
										<div class="task-info">
											<h6>Clear All Activity Logs</h6>
											<p class="text-light mb-0">Delete all activity logs from the database</p>
										</div>
										<button class="btn btn-danger mt-3" @click="clearAllActivities" :disabled="isDbOperationRunning">
											<font-awesome-icon :icon="['fas', isDbOperationRunning ? 'spinner' : 'trash']" :spin="isDbOperationRunning" class="me-2" />
											{{ isDbOperationRunning ? 'Clearing...' : 'Clear All Activity Logs' }}
										</button>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { performersAPI, databaseAPI, videosAPI } from '@/services/api'
import websocketService from '@/services/websocket'

export default {
	name: 'TasksPage',
	data() {
		return {
			activeTasks: [],
			databaseInfo: {
				videoCount: 0,
				performerCount: 0,
				studioCount: 0,
				tagCount: 0,
			},
			wsUnsubscribe: null,
			isDbOperationRunning: false,
		}
	},
	mounted() {
		this.loadDatabaseInfo()
		this.connectWebSocket()
		this.loadActiveTasks()

		// Refresh active tasks every 2 seconds as backup
		this.refreshInterval = setInterval(() => {
			this.loadActiveTasks()
		}, 2000)
	},
	beforeUnmount() {
		if (this.wsUnsubscribe) {
			this.wsUnsubscribe()
		}
		if (this.refreshInterval) {
			clearInterval(this.refreshInterval)
		}
	},
	methods: {
		connectWebSocket() {
			console.log('TasksPage: Connecting to WebSocket...')

			// Subscribe to activity updates
			this.wsUnsubscribe = websocketService.on('activity_update', (data) => {
				console.log('TasksPage: Activity update received:', data)
				this.handleActivityUpdate(data)
			})

			// Connect if not already connected
			if (!websocketService.isConnected()) {
				websocketService.connect()
			}
		},

		handleActivityUpdate(activity) {
			// Find existing task in activeTasks
			const index = this.activeTasks.findIndex((t) => t.id === activity.id)

			if (activity.status === 'completed' || activity.status === 'failed') {
				// Remove completed/failed tasks
				if (index > -1) {
					this.activeTasks.splice(index, 1)
				}

				// Show toast notification
				if (activity.status === 'completed') {
					this.$toast.success('Task Completed', activity.message || `${this.formatTaskType(activity.task_type)} completed successfully`)
				} else {
					this.$toast.error('Task Failed', activity.message || `${this.formatTaskType(activity.task_type)} failed`)
				}

				// Reload database info after task completion
				this.loadDatabaseInfo()
			} else {
				// Update or add active task
				if (index > -1) {
					this.activeTasks[index] = { ...this.activeTasks[index], ...activity }
				} else {
					this.activeTasks.push(activity)
				}
			}
		},

		async loadActiveTasks() {
			try {
				const response = await fetch('http://localhost:8080/api/v1/activity?status=running')
				const data = await response.json()

				if (data.success && data.data) {
					// Only update if different to avoid flicker
					const newTasks = data.data
					if (JSON.stringify(newTasks) !== JSON.stringify(this.activeTasks)) {
						this.activeTasks = newTasks
					}
				}
			} catch (error) {
				console.error('Failed to load active tasks:', error)
			}
		},

		async loadDatabaseInfo() {
			try {
				const statsResponse = await databaseAPI.getStats()
				const stats = statsResponse.data

				this.databaseInfo.videoCount = stats.video_count || 0
				this.databaseInfo.performerCount = stats.performer_count || 0
				this.databaseInfo.studioCount = stats.studio_count || 0
				this.databaseInfo.tagCount = stats.tag_count || 0
			} catch (error) {
				console.error('Failed to load database info:', error)
			}
		},

		isTaskRunning(taskType) {
			return this.activeTasks.some((task) => task.task_type === taskType && (task.status === 'running' || task.status === 'pending'))
		},

		getTaskProgress(taskType) {
			const task = this.activeTasks.find((t) => t.task_type === taskType)
			return task ? task.progress : null
		},

		formatTaskType(taskType) {
			return taskType
				.split('_')
				.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
				.join(' ')
		},

		async scanAllLibraries() {
			try {
				const config = {
					server_drives: ['Z:', 'Y:'],
					local_drives: ['C:', 'D:'],
					server_max_concurrent: 2,
					local_max_concurrent: 8,
				}

				const response = await videosAPI.scanAllParallel(config)

				if (response.status === 202 || response.status === 200) {
					this.$toast.success('Scan Started', 'Library scan has been initiated. Watch the progress above!')
					this.loadActiveTasks()
				}
			} catch (error) {
				console.error('Failed to scan libraries:', error)
				this.$toast.error('Scan Failed', error.response?.data?.error || 'Failed to start library scan')
			}
		},

		async generatePreviews() {
			try {
				const response = await videosAPI.generatePreviews()

				if (response.status === 202 || response.status === 200) {
					this.$toast.success('Preview Generation Started', 'Preview generation has been initiated. Watch the progress above!')
					this.loadActiveTasks()
				}
			} catch (error) {
				console.error('Failed to generate previews:', error)
				this.$toast.error('Generation Failed', error.response?.data?.error || 'Failed to start preview generation')
			}
		},

		async generateVideoThumbnails() {
			try {
				const response = await videosAPI.generateThumbnails()

				if (response.status === 202 || response.status === 200) {
					this.$toast.success('Thumbnail Generation Started', 'Video thumbnail generation has been initiated. Watch the progress above!')
					this.loadActiveTasks()
				}
			} catch (error) {
				console.error('Failed to generate video thumbnails:', error)
				this.$toast.error('Generation Failed', error.response?.data?.error || 'Failed to start video thumbnail generation')
			}
		},

		async scanPerformers() {
			try {
				const response = await performersAPI.scan()

				if (response.status === 202 || response.status === 200) {
					this.$toast.success('Performer Scan Started', 'Performer scan has been initiated. Watch the progress above!')
					this.loadActiveTasks()
				}
			} catch (error) {
				console.error('Failed to scan performers:', error)
				this.$toast.error('Scan Failed', error.response?.data?.error || 'Failed to start performer scan')
			}
		},

		async generatePerformerThumbnails() {
			try {
				const response = await performersAPI.generateThumbnails()

				if (response.status === 202 || response.status === 200) {
					this.$toast.success('Thumbnail Generation Started', 'Performer thumbnail generation has been initiated. Watch the progress above!')
					this.loadActiveTasks()
				}
			} catch (error) {
				console.error('Failed to generate performer thumbnails:', error)
				this.$toast.error('Generation Failed', error.response?.data?.error || 'Failed to start thumbnail generation')
			}
		},

		async fetchAllMetadata() {
			try {
				const response = await performersAPI.fetchAllMetadata()

				if (response.status === 202 || response.status === 200) {
					this.$toast.success('Metadata Fetch Started', 'Metadata fetching has been initiated. Watch the progress above!')
					this.loadActiveTasks()
				}
			} catch (error) {
				console.error('Failed to fetch metadata:', error)
				this.$toast.error('Fetch Failed', error.response?.data?.error || 'Failed to start metadata fetch')
			}
		},

		async optimizeDatabase() {
			if (!confirm('Optimize database? This will run VACUUM to reclaim space and improve performance. The application may be briefly unavailable.')) {
				return
			}

			this.isDbOperationRunning = true
			try {
				const response = await databaseAPI.optimize()

				if (response.status === 200) {
					this.$toast.success('Database Optimized', 'Database has been successfully optimized!')
					await this.loadDatabaseInfo()
				}
			} catch (error) {
				console.error('Failed to optimize database:', error)
				this.$toast.error('Optimization Failed', error.response?.data?.error || 'Failed to optimize database')
			} finally {
				this.isDbOperationRunning = false
			}
		},

		async backupDatabase() {
			if (!confirm('Create a database backup? This will create a timestamped backup file.')) {
				return
			}

			this.isDbOperationRunning = true
			try {
				const response = await databaseAPI.backup()

				if (response.status === 200) {
					const backupPath = response.data.backup_path || 'backup created'
					this.$toast.success('Backup Created', `Database backup created: ${backupPath}`)
				}
			} catch (error) {
				console.error('Failed to backup database:', error)
				this.$toast.error('Backup Failed', error.response?.data?.error || 'Failed to create database backup')
			} finally {
				this.isDbOperationRunning = false
			}
		},

		async restoreDatabase() {
			// First, get list of available backups
			this.isDbOperationRunning = true
			try {
				const backupsResponse = await databaseAPI.listBackups()
				const backups = backupsResponse.data.backups || []

				if (backups.length === 0) {
					this.$toast.warning('No Backups Found', 'No backup files available to restore from')
					this.isDbOperationRunning = false
					return
				}

				// Show backup selection dialog
				const backupList = backups.map((b, i) => `${i + 1}. ${b.name} (${b.size}, ${b.created_at})`).join('\n')
				const selection = prompt(`Select a backup to restore:\n\n${backupList}\n\nEnter the number (1-${backups.length}):`)

				if (!selection) {
					this.isDbOperationRunning = false
					return
				}

				const index = parseInt(selection) - 1
				if (isNaN(index) || index < 0 || index >= backups.length) {
					this.$toast.error('Invalid Selection', 'Please enter a valid backup number')
					this.isDbOperationRunning = false
					return
				}

				const selectedBackup = backups[index]

				if (!confirm(`Restore from backup: ${selectedBackup.name}?\n\nWARNING: This will replace the current database. This action cannot be undone!`)) {
					this.isDbOperationRunning = false
					return
				}

				const response = await databaseAPI.restore({ backup_path: selectedBackup.path })

				if (response.status === 200) {
					this.$toast.success('Database Restored', `Database has been restored from ${selectedBackup.name}`)
					await this.loadDatabaseInfo()
					// Reload the page to refresh all data
					setTimeout(() => {
						window.location.reload()
					}, 2000)
				}
			} catch (error) {
				console.error('Failed to restore database:', error)
				this.$toast.error('Restore Failed', error.response?.data?.error || 'Failed to restore database')
			} finally {
				this.isDbOperationRunning = false
			}
		},

		async clearAllActivities() {
			if (!confirm('Delete ALL activity logs? This will permanently remove all activity history from the database. This action cannot be undone!')) {
				return
			}

			// Double confirmation for destructive action
			if (!confirm('Are you ABSOLUTELY sure? This will delete ALL activity logs permanently.')) {
				return
			}

			this.isDbOperationRunning = true
			try {
				const response = await fetch('http://localhost:8080/api/v1/activity/clear-all', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
				})

				const data = await response.json()

				if (response.ok) {
					this.$toast.success('Activities Cleared', `Successfully deleted ${data.deleted_count || 'all'} activity logs`)
					await this.loadDatabaseInfo()
				} else {
					throw new Error(data.error || 'Failed to clear activities')
				}
			} catch (error) {
				console.error('Failed to clear activities:', error)
				this.$toast.error('Clear Failed', error.message || 'Failed to clear activity logs')
			} finally {
				this.isDbOperationRunning = false
			}
		},
	},
}
</script>

<style scoped>
.tasks-page {
	min-height: 100vh;
	background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.page-header h1 {
	color: #00d9ff;
	font-weight: 700;
	text-shadow: 0 0 20px rgba(0, 217, 255, 0.3);
}

.section-title {
	color: #00d9ff;
	font-weight: 600;
	text-transform: uppercase;
	letter-spacing: 1px;
}

.active-tasks-monitor {
	background: rgba(0, 217, 255, 0.05);
	border: 1px solid rgba(0, 217, 255, 0.2);
	border-radius: 12px;
	padding: 1.5rem;
	animation: pulse-border 2s ease-in-out infinite;
}

@keyframes pulse-border {
	0%,
	100% {
		border-color: rgba(0, 217, 255, 0.2);
	}
	50% {
		border-color: rgba(0, 217, 255, 0.5);
	}
}

.task-progress-card {
	background: rgba(0, 0, 0, 0.4);
	border: 1px solid rgba(0, 217, 255, 0.3);
	transition: all 0.3s ease;
}

.task-progress-card:hover {
	transform: translateY(-2px);
	box-shadow: 0 4px 12px rgba(0, 217, 255, 0.2);
}

.task-card {
	background: rgba(0, 0, 0, 0.3);
	border: 1px solid rgba(0, 217, 255, 0.2);
	backdrop-filter: blur(10px);
	transition: all 0.3s ease;
}

.task-card:hover {
	border-color: rgba(0, 217, 255, 0.4);
	box-shadow: 0 8px 24px rgba(0, 217, 255, 0.15);
}

.task-card .card-header {
	background: rgba(0, 217, 255, 0.1);
	border-bottom: 1px solid rgba(0, 217, 255, 0.2);
	padding: 1rem 1.5rem;
}

.task-card .card-header h5 {
	color: #00d9ff;
	font-weight: 600;
	margin: 0;
}

.task-item {
	padding: 1rem 0;
}

.task-item h6 {
	color: #fff;
	font-weight: 600;
	margin-bottom: 0.5rem;
}

.task-item .btn {
	width: 100%;
	font-weight: 600;
	text-transform: uppercase;
	letter-spacing: 0.5px;
	transition: all 0.3s ease;
}

.task-item .btn:hover:not(:disabled) {
	transform: translateY(-2px);
	box-shadow: 0 4px 12px rgba(0, 217, 255, 0.3);
}

.task-feedback {
	background: rgba(0, 0, 0, 0.3);
	padding: 0.5rem;
	border-radius: 6px;
	border: 1px solid rgba(0, 217, 255, 0.2);
}

.progress {
	background: rgba(0, 0, 0, 0.3);
}

.stat-item {
	text-align: center;
	padding: 1rem;
	background: rgba(0, 0, 0, 0.2);
	border-radius: 8px;
}

.stat-item label {
	color: rgba(255, 255, 255, 0.6);
	font-size: 0.875rem;
	text-transform: uppercase;
	letter-spacing: 0.5px;
}

.stat-item h4 {
	color: #00d9ff;
	margin-top: 0.5rem;
	font-weight: 600;
}
</style>
