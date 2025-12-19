<template>
	<div class="tasks-page">
		<div class="container-fluid py-4">
			<div class="page-header mb-4">
				<h1>
					<font-awesome-icon :icon="['fas', 'tasks']" class="me-3" />
					Task Center
				</h1>
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
							<div class="task-item">
								<div class="task-info">
									<h6>Scan All Libraries</h6>
									<p class="text-light mb-0">Scan all libraries for new or changed videos</p>
								</div>
								<button class="btn btn-primary mt-3" @click="scanAllLibraries" :disabled="scanning">
									<font-awesome-icon :icon="['fas', scanning ? 'spinner' : 'sync']" :spin="scanning" class="me-2" />
									{{ scanning ? 'Scanning...' : 'Scan Libraries' }}
								</button>
							</div>
							<hr class="my-4" />
							<div class="task-item">
								<div class="task-info">
									<h6>Generate Previews</h6>
									<p class="text-light mb-0">Generate hover preview storyboards for all videos (10 frames per video)</p>
								</div>
								<button class="btn btn-info mt-3" @click="generatePreviews" :disabled="generatingPreviews">
									<font-awesome-icon :icon="['fas', generatingPreviews ? 'spinner' : 'images']" :spin="generatingPreviews" class="me-2" />
									{{ generatingPreviews ? 'Generating...' : 'Generate Previews' }}
								</button>
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
							<div class="task-item">
								<div class="task-info">
									<h6>Fetch Missing Metadata</h6>
									<p class="text-light mb-0">Fetch metadata from AdultDataLink for performers without metadata (excludes Zoo performers)</p>
								</div>
								<button class="btn btn-success mt-3" @click="fetchMissingMetadata" :disabled="fetchingMetadata">
									<font-awesome-icon :icon="['fas', fetchingMetadata ? 'spinner' : 'download']" :spin="fetchingMetadata" class="me-2" />
									{{ fetchingMetadata ? 'Fetching...' : 'Fetch Metadata' }}
								</button>
							</div>
						</div>
					</div>
				</div>

				<!-- Database Tasks -->
				<div class="col-md-12">
					<div class="card task-card">
						<div class="card-header">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'database']" class="me-2" />
								Database Management
							</h5>
						</div>
						<div class="card-body">
							<!-- Database Stats -->
							<div class="row g-3 mb-4">
								<div class="col-md-3">
									<div class="card stat-card h-100">
										<div class="card-body">
											<div class="d-flex justify-content-between align-items-center">
												<div>
													<h6 class="mb-1 text-primary">Videos</h6>
													<h2 class="mb-0">{{ databaseInfo.videoCount }}</h2>
												</div>
												<font-awesome-icon :icon="['fas', 'video']" size="2x" class="text-primary" />
											</div>
										</div>
									</div>
								</div>
								<div class="col-md-3">
									<div class="card stat-card h-100">
										<div class="card-body">
											<div class="d-flex justify-content-between align-items-center">
												<div>
													<h6 class="mb-1 text-primary">Performers</h6>
													<h2 class="mb-0">{{ databaseInfo.performerCount }}</h2>
												</div>
												<font-awesome-icon :icon="['fas', 'user']" size="2x" class="text-success" />
											</div>
										</div>
									</div>
								</div>
								<div class="col-md-3">
									<div class="card stat-card h-100">
										<div class="card-body">
											<div class="d-flex justify-content-between align-items-center">
												<div>
													<h6 class="mb-1 text-primary">Studios</h6>
													<h2 class="mb-0">{{ databaseInfo.studioCount }}</h2>
												</div>
												<font-awesome-icon :icon="['fas', 'building']" size="2x" class="text-warning" />
											</div>
										</div>
									</div>
								</div>
								<div class="col-md-3">
									<div class="card stat-card h-100">
										<div class="card-body">
											<div class="d-flex justify-content-between align-items-center">
												<div>
													<h6 class="mb-1 text-primary">Tags</h6>
													<h2 class="mb-0">{{ databaseInfo.tagCount }}</h2>
												</div>
												<font-awesome-icon :icon="['fas', 'tags']" size="2x" class="text-info" />
											</div>
										</div>
									</div>
								</div>
							</div>

							<!-- Database Info -->
							<div class="row g-3 mb-4">
								<div class="col-md-6">
									<div class="info-item">
										<strong>Database Size:</strong>
										<span class="text-light ms-2">{{ databaseInfo.size }}</span>
									</div>
								</div>
								<div class="col-md-6">
									<div class="info-item">
										<strong>Location:</strong>
										<span class="text-light ms-2" style="font-family: monospace; font-size: 0.9em">{{ databaseInfo.location }}</span>
									</div>
								</div>
							</div>

							<!-- Database Operations -->
							<div class="task-item">
								<div class="task-info mb-3">
									<h6>Database Operations</h6>
									<p class="text-light mb-0">Maintain, backup, and restore your database</p>
								</div>
								<div class="d-flex gap-2 flex-wrap">
									<button class="btn btn-warning" @click="optimizeDatabase">
										<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
										Optimize
									</button>
									<button class="btn btn-success" @click="backupDatabase">
										<font-awesome-icon :icon="['fas', 'save']" class="me-2" />
										Backup
									</button>
									<button class="btn btn-info" @click="restoreDatabase">
										<font-awesome-icon :icon="['fas', 'upload']" class="me-2" />
										Restore
									</button>
									<button class="btn btn-danger" @click="confirmClearLogs" :disabled="clearingLogs">
										<font-awesome-icon :icon="['fas', clearingLogs ? 'spinner' : 'trash']" :spin="clearingLogs" class="me-2" />
										{{ clearingLogs ? 'Clearing...' : 'Clear Activity Logs' }}
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Task Progress Modal -->
			<div v-if="taskProgress.show" class="task-progress-modal">
				<div class="task-progress-content">
					<h4>{{ taskProgress.title }}</h4>
					<p>{{ taskProgress.message }}</p>
					<div class="progress">
						<div
							class="progress-bar progress-bar-striped progress-bar-animated"
							:style="{ width: taskProgress.percent + '%' }"
							role="progressbar"
						>
							{{ taskProgress.percent }}%
						</div>
					</div>
					<div class="task-progress-details mt-2">
						<small>{{ taskProgress.details }}</small>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
import { librariesAPI, performersAPI, databaseAPI, videosAPI, activityAPI } from '@/services/api'

export default {
	name: 'TasksPage',
	data() {
		return {
			scanning: false,
			fetchingMetadata: false,
			generatingPreviews: false,
			clearingLogs: false,
			databaseInfo: {
				location: 'C:\\Repos\\Video Storage AI\\api\\video-storage.db',
				size: 'Loading...',
				videoCount: 0,
				performerCount: 0,
				studioCount: 0,
				tagCount: 0,
			},
			taskProgress: {
				show: false,
				title: '',
				message: '',
				percent: 0,
				details: '',
			},
		}
	},
	mounted() {
		this.loadDatabaseInfo()
	},
	methods: {
		async loadDatabaseInfo() {
			try {
				const statsResponse = await databaseAPI.getStats()
				const stats = statsResponse.data

				this.databaseInfo.videoCount = stats.video_count || 0
				this.databaseInfo.performerCount = stats.performer_count || 0
				this.databaseInfo.studioCount = stats.studio_count || 0
				this.databaseInfo.tagCount = stats.tag_count || 0

				const sizeInMB = stats.size / 1024 / 1024
				this.databaseInfo.size = sizeInMB > 1 ? `${sizeInMB.toFixed(2)} MB` : `${(stats.size / 1024).toFixed(2)} KB`
			} catch (error) {
				console.error('Failed to load database info:', error)
				this.databaseInfo.size = 'Unknown'
			}
		},

		async scanAllLibraries() {
			if (this.scanning) return

			this.scanning = true
			this.showProgress('Scanning Libraries', 'Starting parallel scan with drive-aware optimization...', 0)

			try {
				// Get all libraries to show count
				const libResponse = await librariesAPI.getAll()
				const libraries = (libResponse && libResponse.data) || []

				if (libraries.length === 0) {
					this.$toast.warning('No Libraries', 'No libraries found to scan')
					this.hideProgress()
					this.scanning = false
					return
				}

				// Group libraries for display
				const serverLibs = libraries.filter((lib) => lib.path.startsWith('Z:') || lib.path.startsWith('Y:'))
				const localLibs = libraries.filter((lib) => lib.path.startsWith('C:') || lib.path.startsWith('D:'))

				this.updateProgress(
					'Scanning Libraries',
					`Starting parallel scan: ${serverLibs.length} server libraries, ${localLibs.length} local libraries`,
					10,
					'Using multi-threaded scanning optimized for your hardware'
				)

				// Configure parallel scanning with smart defaults
				const config = {
					server_drives: ['Z:', 'Y:'],
					local_drives: ['C:', 'D:'],
					server_max_concurrent: 2, // Conservative for server (avoid overload)
					local_max_concurrent: 8, // Aggressive for local PC (you said it's turbo fast!)
				}

				// Start parallel scan
				await videosAPI.scanAllParallel(config)

				this.updateProgress('Scanning Libraries', 'Parallel scan in progress...', 50, 'Libraries are being scanned simultaneously')

				// Simulate progress updates (actual progress happens in background)
				const progressInterval = setInterval(() => {
					if (this.taskProgress.percent < 90) {
						this.taskProgress.percent += 5
						this.updateProgress(
							'Scanning Libraries',
							'Processing videos in parallel...',
							this.taskProgress.percent,
							`Server: ${serverLibs.length} libs | Local: ${localLibs.length} libs | Running concurrently`
						)
					}
				}, 3000)

				// Wait a bit for scanning to complete (you can adjust this or implement websocket updates)
				setTimeout(() => {
					clearInterval(progressInterval)
					this.updateProgress('Scanning Libraries', 'Scan initiated!', 100, 'Check Activity page for detailed progress')
					this.$toast.success(
						'Parallel Scan Started',
						`${libraries.length} libraries are being scanned in parallel. Server: ${serverLibs.length} @ 50% CPU | Local: ${localLibs.length} @ max speed`
					)

					setTimeout(() => {
						this.hideProgress()
						this.loadDatabaseInfo()
					}, 2000)
				}, 5000)
			} catch (error) {
				console.error('Failed to scan libraries:', error)
				this.$toast.error('Scan Failed', 'Could not start parallel scan')
				this.hideProgress()
			} finally {
				setTimeout(() => {
					this.scanning = false
				}, 5000)
			}
		},

		async generatePreviews() {
			console.log('Generate Previews button clicked!')
			if (this.generatingPreviews) {
				console.log('Already generating, skipping...')
				return
			}

			this.generatingPreviews = true
			console.log('Starting preview generation...')
			this.showProgress('Generating Previews', 'Starting preview generation with drive-aware optimization...', 0)

			try {
				// Get video count for display
				console.log('Fetching database stats...')
				const statsResponse = await databaseAPI.getStats()
				console.log('Stats response:', statsResponse)
				const videoCount = (statsResponse.data && statsResponse.data.video_count) || 0
				console.log('Video count:', videoCount)

				if (videoCount === 0) {
					this.$toast.warning('No Videos', 'No videos found to generate previews for')
					this.hideProgress()
					this.generatingPreviews = false
					return
				}

				this.updateProgress(
					'Generating Previews',
					`Preparing to generate previews for ${videoCount} videos`,
					10,
					'Using multi-threaded processing optimized for your hardware'
				)

				// Configure preview generation with VERY conservative defaults to prevent system freeze
				// Each concurrent library spawns multiple ffmpeg processes with hardware acceleration
				const config = {
					server_drives: ['Z:', 'Y:'],
					local_drives: ['C:', 'D:'],
					server_max_concurrent: 1, // Very conservative for server (1 library at a time)
					local_max_concurrent: 2, // Conservative for local PC (2 libraries at a time max)
				}

				// Start preview generation
				console.log('Calling API with config:', config)
				const response = await videosAPI.generatePreviews(config)
				console.log('API response:', response)

				this.updateProgress('Generating Previews', 'Preview generation in progress...', 50, 'Generating 10 frames per video')

				// Simulate progress updates
				const progressInterval = setInterval(() => {
					if (this.taskProgress.percent < 90) {
						this.taskProgress.percent += 3
						this.updateProgress(
							'Generating Previews',
							'Processing videos in parallel...',
							this.taskProgress.percent,
							`Generating storyboard thumbnails for hover effects`
						)
					}
				}, 4000)

				// Wait and complete
				setTimeout(() => {
					clearInterval(progressInterval)
					this.updateProgress('Generating Previews', 'Preview generation started!', 100, 'Check Activity page for detailed progress')
					this.$toast.success(
						'Preview Generation Started',
						`Generating ${videoCount} video previews in parallel. Each video gets 10 thumbnail frames for hover effects!`
					)

					setTimeout(() => {
						this.hideProgress()
					}, 2000)
				}, 5000)
			} catch (error) {
				console.error('Failed to generate previews:', error)
				this.$toast.error('Generation Failed', 'Could not start preview generation')
				this.hideProgress()
			} finally {
				setTimeout(() => {
					this.generatingPreviews = false
				}, 5000)
			}
		},

		async fetchMissingMetadata() {
			if (this.fetchingMetadata) return

			this.fetchingMetadata = true
			this.showProgress('Fetching Metadata', 'Loading performers...', 0)

			try {
				// Get all performers
				const perfResponse = await performersAPI.getAll()
				const allPerformers = (perfResponse && perfResponse.data) || []

				// Filter performers without metadata and exclude zoo performers
				const performersNeedingMetadata = allPerformers.filter((p) => {
					return !p.zoo && (!p.metadata || Object.keys(p.metadata).length === 0)
				})

				if (performersNeedingMetadata.length === 0) {
					this.$toast.info('No Performers', 'All non-zoo performers already have metadata')
					this.hideProgress()
					this.fetchingMetadata = false
					return
				}

				this.updateProgress('Fetching Metadata', `Found ${performersNeedingMetadata.length} performers needing metadata`, 10)

				let successCount = 0
				let failCount = 0

				// Fetch metadata for each performer
				for (let i = 0; i < performersNeedingMetadata.length; i++) {
					const performer = performersNeedingMetadata[i]
					const progress = 10 + Math.floor((i / performersNeedingMetadata.length) * 80)

					this.updateProgress(
						'Fetching Metadata',
						`Fetching: ${performer.name}`,
						progress,
						`Performer ${i + 1} of ${performersNeedingMetadata.length} | Success: ${successCount} | Failed: ${failCount}`
					)

					try {
						await performersAPI.fetchMetadata(performer.id)
						successCount++
						// Small delay to avoid rate limiting
						await new Promise((resolve) => setTimeout(resolve, 500))
					} catch (error) {
						console.error(`Failed to fetch metadata for ${performer.name}:`, error)
						failCount++
					}
				}

				this.updateProgress('Fetching Metadata', 'Complete!', 100, `Success: ${successCount} | Failed: ${failCount}`)
				this.$toast.success('Metadata Fetched', `Successfully fetched ${successCount} performers. Failed: ${failCount}`)

				setTimeout(() => {
					this.hideProgress()
				}, 2000)
			} catch (error) {
				console.error('Failed to fetch metadata:', error)
				this.$toast.error('Fetch Failed', 'Could not fetch performer metadata')
				this.hideProgress()
			} finally {
				this.fetchingMetadata = false
			}
		},

		async optimizeDatabase() {
			if (confirm('This will optimize the database. This may take a few moments. Continue?')) {
				this.$toast.info('Optimizing', 'Database optimization in progress...')
				try {
					await databaseAPI.optimize()
					this.$toast.success('Complete', 'Database has been optimized')
					this.loadDatabaseInfo()
				} catch (error) {
					console.error('Database optimization failed:', error)
					this.$toast.error('Optimization Failed', 'Could not optimize database')
				}
			}
		},

		async backupDatabase() {
			this.$toast.info('Backing Up', 'Creating database backup...')
			try {
				const result = await databaseAPI.backup()
				this.$toast.success('Backup Complete', `Database backed up to: ${result.data.backup_path}`)
			} catch (error) {
				console.error('Database backup failed:', error)
				this.$toast.error('Backup Failed', 'Could not backup database')
			}
		},

		async restoreDatabase() {
			try {
				const backupsResponse = await databaseAPI.listBackups()
				const backups = backupsResponse.data

				if (!backups || backups.length === 0) {
					this.$toast.warning('No Backups', 'No backup files found')
					return
				}

				let backupList = 'Available backups:\n\n'
				backups.forEach((backup, index) => {
					const date = new Date(backup.timestamp).toLocaleString()
					const size = (backup.size / 1024 / 1024).toFixed(2) + ' MB'
					backupList += `${index + 1}. ${date} (${size})\n`
				})
				backupList += '\nEnter the number of the backup to restore (or cancel):'

				const selection = prompt(backupList)
				if (!selection) return

				const index = parseInt(selection) - 1
				if (index < 0 || index >= backups.length) {
					this.$toast.error('Invalid Selection', 'Please select a valid backup number')
					return
				}

				const selectedBackup = backups[index]

				if (
					confirm(
						`WARNING: This will restore the database from backup and overwrite all current data.\n\nBackup: ${new Date(selectedBackup.timestamp).toLocaleString()}\n\nAre you sure you want to continue?`
					)
				) {
					this.$toast.info('Restoring', 'Restoring database from backup...')
					await databaseAPI.restore(selectedBackup.backup_path)
					this.$toast.success('Restore Complete', 'Database has been restored successfully')
					this.loadDatabaseInfo()
				}
			} catch (error) {
				console.error('Database restore failed:', error)
				this.$toast.error('Restore Failed', 'Could not restore database')
			}
		},

		confirmClearLogs() {
			if (
				confirm(
					'WARNING: This will permanently delete all activity logs. This cannot be undone.\n\nAre you sure you want to continue?'
				)
			) {
				this.clearLogs()
			}
		},

		async clearLogs() {
			this.clearingLogs = true
			try {
				const response = await activityAPI.clearAll()
				const deletedCount = response.data?.deleted_count || 0
				this.$toast.success('Logs Cleared', `All ${deletedCount} activity logs have been deleted`)
				this.loadDatabaseInfo()
			} catch (error) {
				console.error('Failed to clear logs:', error)
				this.$toast.error('Clear Failed', 'Could not clear activity logs')
			} finally {
				this.clearingLogs = false
			}
		},

		showProgress(title, message, percent) {
			this.taskProgress = {
				show: true,
				title,
				message,
				percent,
				details: '',
			}
		},

		updateProgress(title, message, percent, details = '') {
			this.taskProgress.title = title
			this.taskProgress.message = message
			this.taskProgress.percent = percent
			this.taskProgress.details = details
		},

		hideProgress() {
			this.taskProgress.show = false
		},
	},
}
</script>

<style scoped>
.tasks-page {
	min-height: 100vh;
	background: #0f0c29;
}

.page-header h1 {
	color: white;
	font-weight: 600;
	font-size: 2rem;
}

.task-card {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	transition: all 0.3s;
}

.task-card:hover {
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(255, 255, 255, 0.2);
}

.task-card .card-header {
	background: rgba(255, 255, 255, 0.03);
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	padding: 1rem 1.5rem;
}

.task-card .card-header h5 {
	color: white;
	font-weight: 600;
}

.task-card .card-body {
	padding: 1.5rem;
}

.task-item {
	color: white;
}

.task-info h6 {
	color: white;
	font-weight: 600;
	margin-bottom: 0.5rem;
}

.stat-card {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	transition: all 0.3s;
}

.stat-card:hover {
	background: rgba(255, 255, 255, 0.06);
	transform: translateY(-2px);
}

.stat-card h6 {
	font-size: 0.85rem;
	text-transform: uppercase;
	letter-spacing: 0.5px;
}

.stat-card h2 {
	color: white;
	font-weight: 700;
}

.info-item {
	color: white;
	padding: 0.5rem 0;
}

.task-progress-modal {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.8);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 9999;
}

.task-progress-content {
	background: #1a1a2e;
	padding: 2rem;
	border-radius: 15px;
	min-width: 500px;
	max-width: 600px;
	box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
	border: 1px solid rgba(255, 255, 255, 0.1);
}

.task-progress-content h4 {
	margin-bottom: 1rem;
	color: white;
	font-weight: 600;
}

.task-progress-content p {
	margin-bottom: 1rem;
	color: rgba(255, 255, 255, 0.8);
}

.progress {
	height: 30px;
	border-radius: 15px;
	overflow: hidden;
	background: rgba(255, 255, 255, 0.1);
}

.progress-bar {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	font-weight: 600;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
}

.task-progress-details {
	text-align: center;
	color: rgba(255, 255, 255, 0.6);
	margin-top: 0.75rem;
}

.btn {
	transition: all 0.3s;
}

.btn:disabled {
	opacity: 0.6;
	cursor: not-allowed;
}
</style>
