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
							<hr class="my-4" />

							<!-- Run Path Analysis -->
							<div class="task-item">
								<div class="task-info">
									<h6>Run Path Analysis</h6>
									<p class="text-light mb-0">Auto-link performers, tags, and studios based on video folder and file names</p>
								</div>
								<button class="btn btn-warning mt-3" @click="runPathAnalysis" :disabled="isTaskRunning('path_analysis')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('path_analysis') ? 'spinner' : 'link']"
										:spin="isTaskRunning('path_analysis')"
										class="me-2"
									/>
									{{ isTaskRunning('path_analysis') ? 'Analysing...' : 'Run Path Analysis' }}
								</button>
							</div>
							<hr class="my-4" />

							<!-- AI Health Check -->
							<div class="task-item">
								<div class="task-info">
									<h6>AI Health Check</h6>
									<p class="text-light mb-0">Auto-fix missing metadata, missing thumbnails, and unlinked performers across all videos. Also runs automatically every 6 hours.</p>
								</div>
								<button class="btn btn-purple mt-3" @click="runHealthCheck" :disabled="isTaskRunning('video_health_check')">
									<font-awesome-icon
										:icon="['fas', isTaskRunning('video_health_check') ? 'spinner' : 'heart-pulse']"
										:spin="isTaskRunning('video_health_check')"
										class="me-2"
									/>
									{{ isTaskRunning('video_health_check') ? 'Checking...' : 'Run Health Check' }}
								</button>
								<div v-if="getTaskProgress('video_health_check')" class="task-feedback mt-2">
									<div class="progress" style="height: 6px">
										<div class="progress-bar bg-purple" :style="{ width: getTaskProgress('video_health_check') + '%' }"></div>
									</div>
									<small class="text-purple">{{ getTaskProgress('video_health_check') }}%</small>
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
			<!-- App Control -->
			<div class="row g-3 mt-3">
				<div class="col-12">
					<div class="card task-card">
						<div class="card-header">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'power-off']" class="me-2" />
								App Control
							</h5>
						</div>
						<div class="card-body">
							<div class="row g-3">
								<!-- Restart Services -->
								<div class="col-md-6">
									<div class="task-item">
										<div class="task-info">
											<h6>Restart Services</h6>
											<p class="text-light mb-0">Restart both the API and Web services without rebuilding. The page will briefly become unavailable.</p>
										</div>
										<button class="btn btn-warning mt-3" @click="restartServices" :disabled="appControlRunning">
											<font-awesome-icon :icon="['fas', appControlRunning ? 'spinner' : 'redo']" :spin="appControlRunning" class="me-2" />
											{{ appControlRunning ? 'Restarting...' : 'Restart Services' }}
										</button>
									</div>
								</div>

								<!-- Build & Deploy -->
								<div class="col-md-6">
									<div class="task-item">
										<div class="task-info">
											<h6>Build &amp; Deploy</h6>
											<p class="text-light mb-0">Run build.bat — builds the frontend and API, then restarts services. Takes ~5 minutes. The app will be unavailable during the build.</p>
										</div>
										<button class="btn btn-danger mt-3" @click="buildAndDeploy" :disabled="appControlRunning">
											<font-awesome-icon :icon="['fas', appControlRunning ? 'spinner' : 'hammer']" :spin="appControlRunning" class="me-2" />
											{{ appControlRunning ? 'Building...' : 'Build & Deploy' }}
										</button>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
			</div>

			<!-- Duplicate Detection -->
			<div class="row g-3 mt-3">
				<div class="col-12">
					<div class="card task-card">
						<div class="card-header d-flex align-items-center justify-content-between">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', 'clone']" class="me-2" />
								Duplicate Detection
							</h5>
							<div v-if="duplicates.scanned" class="text-light small">
								{{ duplicates.totalGroups }} group{{ duplicates.totalGroups !== 1 ? 's' : '' }} &mdash;
								{{ duplicates.totalDuplicates }} videos &mdash;
								<span class="text-warning">{{ formatBytes(duplicates.wastedBytes) }} wasted</span>
							</div>
						</div>
						<div class="card-body">
							<div class="task-item mb-3">
								<div class="task-info">
									<h6>Find Duplicate Videos</h6>
									<p class="text-light mb-0">Detect videos with identical file size and duration — exact file copies stored in multiple locations.</p>
								</div>
								<button class="btn btn-warning mt-3" @click="scanDuplicates" :disabled="duplicates.loading">
									<font-awesome-icon :icon="['fas', duplicates.loading ? 'spinner' : 'clone']" :spin="duplicates.loading" class="me-2" />
									{{ duplicates.loading ? 'Scanning...' : 'Scan for Duplicates' }}
								</button>
							</div>

							<!-- Results -->
							<div v-if="duplicates.scanned && duplicates.groups.length === 0" class="alert alert-success mb-0">
								<font-awesome-icon :icon="['fas', 'check-circle']" class="me-2" />
								No duplicates found — your library is clean!
							</div>

							<div v-if="duplicates.groups.length > 0">
								<div v-for="(group, gi) in duplicates.groups" :key="gi" class="duplicate-group mb-3">
									<div class="duplicate-group-header d-flex align-items-center justify-content-between mb-2">
										<span class="text-light small">
											{{ formatBytes(group.file_size) }} &bull; {{ Math.round(group.duration) }}s
										</span>
										<div class="d-flex align-items-center gap-2">
											<button
												class="btn btn-sm btn-outline-info"
												:disabled="group.qualityLoading"
												@click="compareQuality(gi)"
											>
												<font-awesome-icon :icon="['fas', group.qualityLoading ? 'spinner' : 'chart-bar']" :spin="group.qualityLoading" class="me-1" />
												{{ group.qualityLoading ? 'Scoring...' : 'Compare Quality' }}
											</button>
											<span class="badge bg-warning text-dark">{{ group.videos.length }} copies</span>
										</div>
									</div>
									<div class="duplicate-videos">
										<div v-for="video in group.videos" :key="video.id" class="duplicate-video-row d-flex align-items-center gap-3 p-2">
											<!-- Thumbnail with fallback overlay -->
											<div class="dup-thumb dup-thumb-placeholder flex-shrink-0 d-flex align-items-center justify-content-center" style="position:relative;overflow:hidden">
												<font-awesome-icon :icon="['fas', 'film']" class="text-secondary" />
												<img
													v-if="video.thumbnail_path"
													:src="getThumbnailUrl(video.thumbnail_path)"
													style="position:absolute;inset:0;width:100%;height:100%;object-fit:cover"
													@error="(e) => e.target.remove()"
													alt=""
												/>
											</div>
											<div class="flex-grow-1 min-w-0">
												<div class="d-flex align-items-center gap-2">
													<span class="text-truncate fw-semibold small" :title="video.title">{{ video.title }}</span>
													<span
														v-if="video.is_topaz_enhanced"
														class="flex-shrink-0 badge text-white"
														style="font-size:0.6rem;background:#7c3aed;white-space:nowrap"
													>Topaz {{ video.topaz_type }}</span>
												</div>
												<div class="text-truncate text-secondary" style="font-size:0.75rem" :title="video.file_path">{{ video.file_path }}</div>
											</div>
											<button
												class="btn btn-sm btn-outline-danger flex-shrink-0"
												:disabled="duplicates.deletingId === video.id"
												@click="deleteDuplicate(video.id, gi)"
											>
												<font-awesome-icon :icon="['fas', duplicates.deletingId === video.id ? 'spinner' : 'trash']" :spin="duplicates.deletingId === video.id" />
											</button>
										</div>
									</div>

									<!-- Quality comparison results -->
									<div v-if="group.quality" class="quality-results mt-3 pt-3" style="border-top: 1px solid #444">
										<div class="d-flex align-items-center gap-2 mb-2">
											<font-awesome-icon :icon="['fas', 'trophy']" class="text-warning" />
											<span class="small fw-semibold text-light">Quality Comparison</span>
										</div>
										<div class="quality-recommendation text-info small mb-3 fst-italic">{{ group.quality.recommendation }}</div>
										<div v-for="score in group.quality.videos" :key="score.video_id" class="quality-score-row mb-3">
											<div class="d-flex gap-2">
												<!-- Thumbnail with fallback overlay -->
												<div
													style="width:72px;height:54px;flex-shrink:0;background:#2a2a2a;border-radius:4px;position:relative;overflow:hidden"
													class="d-flex align-items-center justify-content-center"
												>
													<font-awesome-icon :icon="['fas', 'film']" class="text-secondary" />
													<img
														v-if="score.thumbnail_path"
														:src="getThumbnailUrl(score.thumbnail_path)"
														style="position:absolute;inset:0;width:100%;height:100%;object-fit:cover"
														@error="(e) => e.target.remove()"
														alt=""
													/>
												</div>

												<div class="flex-grow-1 min-w-0">
													<!-- Rank + title + Topaz badge + score -->
													<div class="d-flex align-items-center gap-2 mb-1">
														<span
															class="rank-badge flex-shrink-0 d-inline-flex align-items-center justify-content-center"
															:class="score.is_recommended ? 'bg-warning text-dark' : 'bg-secondary text-white'"
															style="width:20px;height:20px;border-radius:50%;font-size:0.7rem;font-weight:700"
														>{{ score.rank }}</span>
														<span class="small text-truncate flex-grow-1 fw-semibold" :title="score.file_path">{{ score.title }}</span>
														<span
															v-if="score.is_topaz_enhanced"
															class="flex-shrink-0 badge text-white"
															style="font-size:0.6rem;background:#7c3aed;white-space:nowrap"
														>Topaz {{ score.topaz_type }}</span>
														<span class="small fw-bold flex-shrink-0" :class="getScoreClass(score.overall_score)">{{ score.overall_score }}%</span>
													</div>
													<!-- Score bar -->
													<div class="progress mb-1" style="height:3px">
														<div class="progress-bar" :class="getScoreBarClass(score.overall_score)" :style="{ width: score.overall_score + '%' }"></div>
													</div>
													<!-- Metadata row -->
													<div class="d-flex flex-wrap gap-2 align-items-center" style="font-size:0.68rem;color:#888">
														<span v-if="score.resolution" class="text-light fw-semibold">{{ score.resolution }}</span>
														<span v-if="score.codec" class="text-uppercase">{{ score.codec }}</span>
														<span v-if="score.fps > 0">{{ score.fps }} fps</span>
														<span v-if="score.bitrate > 0">{{ formatBitrate(score.bitrate) }}</span>
														<span v-if="score.file_size > 0">{{ formatBytes(score.file_size) }}</span>
														<span class="ms-auto">V <strong :class="getScoreClass(score.video_score)">{{ score.video_score }}%</strong></span>
														<span>T <strong :class="getScoreClass(score.technical_score)">{{ score.technical_score }}%</strong></span>
													</div>
												</div>
											</div>
										</div>
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
import { performersAPI, databaseAPI, videosAPI, systemAPI, aiAPI, getAssetURL } from '@/services/api'
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
			appControlRunning: false,
			duplicates: {
				groups: [],
				totalGroups: 0,
				totalDuplicates: 0,
				wastedBytes: 0,
				loading: false,
				scanned: false,
				deletingId: null,
			},
		}
	},
	mounted() {
		this.loadDatabaseInfo()
		this.connectWebSocket()
		this.loadActiveTasks()
	},
	beforeUnmount() {
		if (this.wsUnsubscribe) {
			this.wsUnsubscribe()
		}
	},
	methods: {
		connectWebSocket() {
			// Subscribe to activity updates
			this.wsUnsubscribe = websocketService.on('activity_update', (data) => {
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

				await videosAPI.scanAllParallel(config)
				this.$toast.success('Scan Started', 'Library scan has been initiated. Watch the progress above!')
			} catch (error) {
				console.error('Failed to scan libraries:', error)
				this.$toast.error('Scan Failed', error.response?.data?.error || 'Failed to start library scan')
			}
		},

		async generatePreviews() {
			try {
				await videosAPI.generatePreviews()
				this.$toast.success('Preview Generation Started', 'Preview generation has been initiated. Watch the progress above!')
			} catch (error) {
				console.error('Failed to generate previews:', error)
				this.$toast.error('Generation Failed', error.response?.data?.error || 'Failed to start preview generation')
			}
		},

		async generateVideoThumbnails() {
			try {
				await videosAPI.generateThumbnails()
				this.$toast.success('Thumbnail Generation Started', 'Video thumbnail generation has been initiated. Watch the progress above!')
			} catch (error) {
				console.error('Failed to generate video thumbnails:', error)
				this.$toast.error('Generation Failed', error.response?.data?.error || 'Failed to start video thumbnail generation')
			}
		},

		async runPathAnalysis() {
			try {
				await videosAPI.runPathAnalysis()
				this.$toast.success('Path Analysis Started', 'Auto-linking performers, tags, and studios from folder and file names. Watch the Activity page for progress!')
			} catch (error) {
				console.error('Failed to start path analysis:', error)
				this.$toast.error('Path Analysis Failed', error.response?.data?.error || 'Failed to start path analysis')
			}
		},

		async runHealthCheck() {
			try {
				await aiAPI.runHealthCheck()
				this.$toast.success('Health Check Started', 'Fixing missing metadata, thumbnails, and unlinked performers. Watch the Activity page for progress!')
			} catch (error) {
				console.error('Failed to start health check:', error)
				this.$toast.error('Health Check Failed', error.response?.data?.error || 'Failed to start health check')
			}
		},

		async scanDuplicates() {
			this.duplicates.loading = true
			this.duplicates.scanned = false
			try {
				const res = await videosAPI.getDuplicates()
				const data = res.data
				this.duplicates.groups = (data.groups || []).map((g) => ({ ...g, quality: null, qualityLoading: false }))
				this.duplicates.totalGroups = data.total_groups || 0
				this.duplicates.totalDuplicates = data.total_duplicates || 0
				this.duplicates.wastedBytes = data.wasted_bytes || 0
				this.duplicates.scanned = true
			} catch (error) {
				console.error('Failed to scan for duplicates:', error)
				this.$toast.error('Scan Failed', 'Failed to scan for duplicates')
			} finally {
				this.duplicates.loading = false
			}
		},

		async deleteDuplicate(videoId, groupIdx) {
			if (!confirm('Delete this video file and its database record? This cannot be undone.')) return
			this.duplicates.deletingId = videoId
			try {
				await videosAPI.delete(videoId)
				const group = this.duplicates.groups[groupIdx]
				group.videos = group.videos.filter((v) => v.id !== videoId)
				this.duplicates.totalDuplicates--
				this.duplicates.wastedBytes -= group.file_size
				if (group.videos.length <= 1) {
					this.duplicates.groups.splice(groupIdx, 1)
					this.duplicates.totalGroups--
				}
				this.$toast.success('Deleted', 'Video deleted successfully')
			} catch (error) {
				console.error('Failed to delete video:', error)
				this.$toast.error('Delete Failed', error.response?.data?.error || 'Failed to delete video')
			} finally {
				this.duplicates.deletingId = null
			}
		},

		async compareQuality(gi) {
			const group = this.duplicates.groups[gi]
			group.qualityLoading = true
			try {
				const res = await videosAPI.qualityCompare(group.videos.map((v) => v.id))
				group.quality = res.data
			} catch (error) {
				console.error('Quality compare failed:', error)
				this.$toast.error('Quality Compare Failed', error?.response?.data?.message || 'Failed to score videos')
			} finally {
				group.qualityLoading = false
			}
		},

		getScoreClass(score) {
			if (score >= 80) return 'text-success'
			if (score >= 60) return 'text-info'
			if (score >= 40) return 'text-warning'
			return 'text-danger'
		},

		getScoreBarClass(score) {
			if (score >= 80) return 'bg-success'
			if (score >= 60) return 'bg-info'
			if (score >= 40) return 'bg-warning'
			return 'bg-danger'
		},

		formatBytes(bytes) {
			if (!bytes || bytes < 1024) return (bytes || 0) + ' B'
			if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
			if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB'
			return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB'
		},

		formatBitrate(bps) {
			if (!bps) return ''
			if (bps >= 1_000_000) return (bps / 1_000_000).toFixed(1) + ' Mbps'
			return Math.round(bps / 1000) + ' Kbps'
		},

		getThumbnailUrl(path) {
			if (!path) return ''
			const encoded = path.split('/').map(s => encodeURIComponent(s)).join('/')
			return getAssetURL(encoded)
		},

		async scanPerformers() {
			try {
				await performersAPI.scan()
				this.$toast.success('Performer Scan Started', 'Performer scan has been initiated. Watch the progress above!')
			} catch (error) {
				console.error('Failed to scan performers:', error)
				this.$toast.error('Scan Failed', error.response?.data?.error || 'Failed to start performer scan')
			}
		},

		async generatePerformerThumbnails() {
			try {
				await performersAPI.generateThumbnails()
				this.$toast.success('Thumbnail Generation Started', 'Performer thumbnail generation has been initiated. Watch the progress above!')
			} catch (error) {
				console.error('Failed to generate performer thumbnails:', error)
				this.$toast.error('Generation Failed', error.response?.data?.error || 'Failed to start thumbnail generation')
			}
		},

		async fetchAllMetadata() {
			try {
				await performersAPI.fetchAllMetadata()
				this.$toast.success('Metadata Fetch Started', 'Metadata fetching has been initiated. Watch the progress above!')
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

		async restartServices() {
			if (!confirm('Restart both API and Web services? The app will be briefly unavailable.')) return
			this.appControlRunning = true
			try {
				await systemAPI.controlService('both', 'restart')
				this.$toast.success('Restart Initiated', 'Services are restarting. Reload the page in ~10 seconds.')
			} catch (error) {
				console.error('Failed to restart services:', error)
				this.$toast.error('Restart Failed', error.response?.data?.error || 'Failed to restart services')
			} finally {
				this.appControlRunning = false
			}
		},

		async buildAndDeploy() {
			if (!confirm('Run Build & Deploy?\n\nThis will:\n1. Stop services\n2. Build frontend (~2 min)\n3. Build API (~1 min)\n4. Restart services\n\nThe app will be unavailable for ~5 minutes.')) return
			this.appControlRunning = true
			try {
				await systemAPI.buildAndDeploy()
				this.$toast.success('Build Started', 'Build & Deploy launched. The app will restart automatically in ~5 minutes.')
			} catch (error) {
				console.error('Failed to start build:', error)
				this.$toast.error('Build Failed', error.response?.data?.error || 'Failed to start build')
			} finally {
				this.appControlRunning = false
			}
		},
	},
}
</script>

<style scoped>
.btn-purple {
	background-color: #8b5cf6;
	border-color: #8b5cf6;
	color: #fff;
}
.btn-purple:hover:not(:disabled) {
	background-color: #7c3aed;
	border-color: #7c3aed;
	color: #fff;
}
.btn-purple:disabled {
	opacity: 0.65;
}
.bg-purple {
	background-color: #8b5cf6 !important;
}
.text-purple {
	color: #8b5cf6 !important;
}

.tasks-page {
	min-height: 100vh;
	background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

/* Force all cards and elements to dark theme */
.tasks-page .card {
	background: rgba(0, 0, 0, 0.3) !important;
	border: 1px solid rgba(0, 217, 255, 0.2) !important;
	color: #fff !important;
}

.tasks-page .card-body {
	background: transparent !important;
	color: rgba(255, 255, 255, 0.9) !important;
}

.tasks-page .card-header {
	background: rgba(0, 217, 255, 0.1) !important;
	border-bottom: 1px solid rgba(0, 217, 255, 0.2) !important;
	color: #00d9ff !important;
}

.tasks-page label {
	color: rgba(255, 255, 255, 0.6) !important;
}

.tasks-page h1, .tasks-page h2, .tasks-page h3, .tasks-page h4, .tasks-page h5, .tasks-page h6 {
	color: #fff !important;
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
	background: rgba(0, 0, 0, 0.3) !important;
	border: 1px solid rgba(0, 217, 255, 0.2) !important;
	backdrop-filter: blur(10px);
	transition: all 0.3s ease;
}

.task-card .card-body {
	background: transparent !important;
	color: rgba(255, 255, 255, 0.9) !important;
}

.task-card:hover {
	border-color: rgba(0, 217, 255, 0.4);
	box-shadow: 0 8px 24px rgba(0, 217, 255, 0.15);
}

.task-card .card-header {
	background: rgba(0, 217, 255, 0.1) !important;
	border-bottom: 1px solid rgba(0, 217, 255, 0.2) !important;
	padding: 1rem 1.5rem;
}

.task-card .card-header h5 {
	color: #00d9ff !important;
	font-weight: 600;
	margin: 0;
}

.task-item {
	padding: 1rem 0;
}

.task-item h6 {
	color: #fff !important;
	font-weight: 600;
	margin-bottom: 0.5rem;
}

.task-item p {
	color: rgba(255, 255, 255, 0.8) !important;
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
	background: rgba(0, 0, 0, 0.2) !important;
	border-radius: 8px;
}

.stat-item label {
	color: rgba(255, 255, 255, 0.6) !important;
	font-size: 0.875rem;
	text-transform: uppercase;
	letter-spacing: 0.5px;
}

.stat-item h4 {
	color: #00d9ff !important;
	margin-top: 0.5rem;
	font-weight: 600;
}

.duplicate-group {
	background: rgba(255, 255, 255, 0.04);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 8px;
	overflow: hidden;
}

.duplicate-group-header {
	padding: 0.5rem 0.75rem;
	background: rgba(255, 255, 255, 0.06);
}

.duplicate-video-row {
	border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.duplicate-video-row:last-child {
	border-bottom: none;
}

.dup-thumb {
	width: 80px;
	height: 48px;
	object-fit: cover;
	border-radius: 4px;
}

.dup-thumb-placeholder {
	width: 80px;
	height: 48px;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 4px;
}
</style>
