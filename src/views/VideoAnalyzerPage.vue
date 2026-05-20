<template>
	<div class="container-fluid py-4">
		<!-- Header -->
		<div class="d-flex align-items-center mb-4">
			<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-3 text-info fs-3" />
			<div>
				<h2 class="mb-0">Video Quality Analyzer</h2>
				<p class="text-muted mb-0 small">Compare quality across library videos and uploaded files</p>
			</div>
		</div>

		<div class="row g-4">
			<!-- Left column: selection -->
			<div class="col-lg-5">
				<div class="card bg-dark border-secondary h-100">
					<div class="card-header border-secondary d-flex align-items-center justify-content-between">
						<span class="fw-semibold">Selected Videos</span>
						<span v-if="selectedVideos.length" class="badge bg-secondary">{{ selectedVideos.length }}</span>
					</div>
					<div class="card-body">
						<!-- Add buttons -->
						<div class="d-flex gap-2 mb-3">
							<button class="btn btn-outline-primary btn-sm flex-fill" @click="showPicker = true">
								<font-awesome-icon :icon="['fas', 'search']" class="me-1" />
								Add from Library
							</button>
							<button class="btn btn-outline-secondary btn-sm flex-fill" @click="$refs.fileInput.click()">
								<font-awesome-icon :icon="['fas', 'upload']" class="me-1" />
								Upload Files
							</button>
							<input ref="fileInput" type="file" multiple accept="video/*" class="d-none" @change="onFilesSelected" />
						</div>

						<!-- Drop zone (shown when list empty) -->
						<div
							v-if="selectedVideos.length === 0"
							class="drop-zone rounded border border-secondary border-dashed d-flex flex-column align-items-center justify-content-center text-muted py-5"
							:class="{ 'drag-over': isDragOver }"
							@dragover.prevent="isDragOver = true"
							@dragleave="isDragOver = false"
							@drop.prevent="onDrop"
						>
							<font-awesome-icon :icon="['fas', 'film']" class="fs-1 mb-3 opacity-50" />
							<p class="mb-1">Drag &amp; drop video files here</p>
							<p class="small">or use the buttons above</p>
						</div>

						<!-- Selected video list -->
						<div v-else>
							<!-- Drag zone as small strip when there are already items -->
							<div
								class="drop-zone-mini rounded border border-secondary border-dashed d-flex align-items-center justify-content-center text-muted mb-3 py-2"
								:class="{ 'drag-over': isDragOver }"
								@dragover.prevent="isDragOver = true"
								@dragleave="isDragOver = false"
								@drop.prevent="onDrop"
							>
								<font-awesome-icon :icon="['fas', 'upload']" class="me-2 small" />
								<span class="small">Drop more files here</span>
							</div>

							<div v-for="(video, idx) in selectedVideos" :key="video.key" class="selected-card d-flex align-items-start gap-2 mb-2 p-2 rounded border border-secondary bg-dark-subtle">
								<!-- Thumbnail -->
								<div class="selected-thumb flex-shrink-0 d-flex align-items-center justify-content-center" style="position:relative;overflow:hidden;width:72px;height:54px;background:#1a1a1a;border-radius:4px">
									<font-awesome-icon :icon="['fas', 'film']" class="text-secondary small" />
									<img
										v-if="video.thumbnail_path"
										:src="getThumbnailUrl(video.thumbnail_path)"
										style="position:absolute;inset:0;width:100%;height:100%;object-fit:cover"
										@error="e => e.target.remove()"
										alt=""
									/>
								</div>

								<!-- Info -->
								<div class="flex-fill min-width-0">
									<div class="text-truncate fw-semibold small" :title="video.title">{{ video.title }}</div>
									<div class="text-muted" style="font-size:0.72rem">
										<span v-if="video.resolution" class="me-2">{{ video.resolution }}</span>
										<span v-if="video.codec" class="me-2 text-uppercase">{{ video.codec }}</span>
										<span v-if="video.fps" class="me-2">{{ formatFPS(video.fps) }}</span>
										<span v-if="video.file_size" class="me-2">{{ formatBytes(video.file_size) }}</span>
										<span v-if="video.duration" class="me-2">{{ formatDuration(video.duration) }}</span>
									</div>
									<div style="font-size:0.68rem" :class="video.source === 'upload' ? 'text-warning' : 'text-info'">
										{{ video.source === 'upload' ? 'Uploaded file' : 'Library' }}
									</div>
								</div>

								<!-- Remove -->
								<button class="btn btn-sm btn-link text-danger p-0 flex-shrink-0" @click="removeVideo(idx)" title="Remove">
									<font-awesome-icon :icon="['fas', 'times']" />
								</button>
							</div>

							<div class="d-flex gap-2 mt-2">
								<button class="btn btn-sm btn-link text-muted p-0" @click="selectedVideos = []">Clear all</button>
							</div>
						</div>
					</div>

					<!-- Compare button -->
					<div class="card-footer border-secondary">
						<button
							class="btn btn-info w-100"
							:disabled="selectedVideos.length < 2 || comparing"
							@click="compareAll"
						>
							<font-awesome-icon v-if="comparing" :icon="['fas', 'spinner']" spin class="me-2" />
							<font-awesome-icon v-else :icon="['fas', 'chart-bar']" class="me-2" />
							{{ comparing ? 'Analyzing…' : 'Analyze & Compare Quality' }}
							<span v-if="selectedVideos.length >= 2" class="badge bg-dark ms-2">{{ selectedVideos.length }}</span>
						</button>
						<div v-if="selectedVideos.length < 2" class="text-muted text-center mt-1" style="font-size:0.75rem">
							Add at least 2 videos to compare
						</div>
					</div>
				</div>
			</div>

			<!-- Right column: results -->
			<div class="col-lg-7">
				<!-- Placeholder when no results -->
				<div v-if="!result" class="card bg-dark border-secondary h-100">
					<div class="card-body d-flex flex-column align-items-center justify-content-center text-muted py-5">
						<font-awesome-icon :icon="['fas', 'chart-bar']" class="fs-1 mb-3 opacity-25" />
						<p class="mb-0">Results will appear here after analysis</p>
					</div>
				</div>

				<!-- Results -->
				<div v-else class="card bg-dark border-secondary">
					<div class="card-header border-secondary d-flex align-items-center justify-content-between">
						<span class="fw-semibold">Quality Results</span>
						<button class="btn btn-sm btn-link text-muted p-0" @click="result = null">Clear results</button>
					</div>
					<div class="card-body">
						<!-- Recommendation banner -->
						<div class="alert alert-info border-info mb-3 py-2 px-3">
							<font-awesome-icon :icon="['fas', 'trophy']" class="me-2 text-warning" />
							<strong>{{ result.recommendation }}</strong>
						</div>

						<!-- Score rows -->
						<div v-for="score in result.videos" :key="score.video_id + '_' + score.title" class="result-row mb-3 p-3 rounded border" :class="score.is_recommended ? 'border-warning bg-dark' : 'border-secondary bg-dark'">
							<div class="d-flex align-items-start gap-3">
								<!-- Thumbnail -->
								<div class="flex-shrink-0 d-flex align-items-center justify-content-center" style="position:relative;overflow:hidden;width:88px;height:66px;background:#1a1a1a;border-radius:4px">
									<font-awesome-icon :icon="['fas', 'film']" class="text-secondary" />
									<img
										v-if="score.thumbnail_path"
										:src="getThumbnailUrl(score.thumbnail_path)"
										style="position:absolute;inset:0;width:100%;height:100%;object-fit:cover"
										@error="e => e.target.remove()"
										alt=""
									/>
								</div>

								<!-- Details -->
								<div class="flex-fill min-width-0">
									<div class="d-flex align-items-center gap-2 mb-1 flex-wrap">
										<!-- Rank badge -->
										<span class="badge" :class="score.is_recommended ? 'bg-warning text-dark' : 'bg-secondary'">
											<font-awesome-icon v-if="score.is_recommended" :icon="['fas', 'trophy']" class="me-1" />
											#{{ score.rank }}
										</span>
										<!-- Topaz badge -->
										<span v-if="score.is_topaz_enhanced" class="badge" style="background:#7c3aed">
											Topaz {{ score.topaz_type }}
										</span>
										<!-- Title -->
										<span class="fw-semibold text-truncate" :title="score.title" style="max-width:300px">{{ score.title }}</span>
									</div>

									<!-- Overall score bar -->
									<div class="d-flex align-items-center gap-2 mb-2">
										<span class="small text-muted" style="width:56px;flex-shrink:0">Overall</span>
										<div class="progress flex-fill" style="height:10px;background:#2a2a2a">
											<div
												class="progress-bar"
												:class="getScoreBarClass(score.overall_score)"
												:style="{ width: score.overall_score + '%' }"
											></div>
										</div>
										<span class="fw-bold small" :class="getScoreClass(score.overall_score)" style="width:36px;text-align:right">{{ score.overall_score.toFixed(0) }}%</span>
									</div>

									<!-- Sub-score bars -->
									<div class="d-flex gap-3 mb-2">
										<div class="flex-fill">
											<div class="d-flex align-items-center gap-2">
												<span class="text-muted" style="font-size:0.7rem;width:44px;flex-shrink:0">Video</span>
												<div class="progress flex-fill" style="height:5px;background:#2a2a2a">
													<div class="progress-bar bg-info" :style="{ width: score.video_score + '%' }"></div>
												</div>
												<span class="text-info" style="font-size:0.7rem;width:30px;text-align:right">{{ score.video_score.toFixed(0) }}%</span>
											</div>
										</div>
										<div class="flex-fill">
											<div class="d-flex align-items-center gap-2">
												<span class="text-muted" style="font-size:0.7rem;width:52px;flex-shrink:0">Technical</span>
												<div class="progress flex-fill" style="height:5px;background:#2a2a2a">
													<div class="progress-bar bg-secondary" :style="{ width: score.technical_score + '%' }"></div>
												</div>
												<span class="text-secondary" style="font-size:0.7rem;width:30px;text-align:right">{{ score.technical_score.toFixed(0) }}%</span>
											</div>
										</div>
									</div>

									<!-- Metadata chips -->
									<div class="d-flex flex-wrap gap-1">
										<span v-if="score.resolution" class="badge bg-dark border border-secondary" style="font-size:0.7rem">{{ score.resolution }}</span>
										<span v-if="score.codec" class="badge bg-dark border border-secondary text-uppercase" style="font-size:0.7rem">{{ score.codec }}</span>
										<span v-if="score.fps" class="badge bg-dark border border-secondary" style="font-size:0.7rem">{{ formatFPS(score.fps) }}</span>
										<span v-if="score.bitrate" class="badge bg-dark border border-secondary" style="font-size:0.7rem">{{ formatBitrate(score.bitrate) }}</span>
										<span v-if="score.file_size" class="badge bg-dark border border-secondary" style="font-size:0.7rem">{{ formatBytes(score.file_size) }}</span>
										<span v-if="score.duration" class="badge bg-dark border border-secondary" style="font-size:0.7rem">{{ formatDuration(score.duration) }}</span>
									</div>

									<!-- Action buttons (library videos only) -->
									<div v-if="score.video_id > 0" class="d-flex gap-1 mt-2">
										<button class="btn btn-outline-info result-action-btn" @click="$router.push('/watch/' + score.video_id)">
											<font-awesome-icon :icon="['fas', 'play']" class="me-1" />Watch
										</button>
										<template v-if="confirmDelete !== score.video_id">
											<button class="btn btn-outline-danger result-action-btn" @click="confirmDelete = score.video_id">
												<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />Delete
											</button>
										</template>
										<template v-else>
											<button class="btn btn-danger result-action-btn" @click="deleteVideo(score.video_id)">
												Confirm Delete
											</button>
											<button class="btn btn-secondary result-action-btn" @click="confirmDelete = null">
												Cancel
											</button>
										</template>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Library Picker Modal -->
		<div class="modal fade" :class="{ show: showPicker }" :style="showPicker ? 'display:block' : ''" tabindex="-1" @click.self="showPicker = false">
			<div class="modal-dialog modal-lg modal-dialog-scrollable">
				<div class="modal-content bg-dark border-secondary">
					<div class="modal-header border-secondary">
						<h5 class="modal-title">Add from Library</h5>
						<button type="button" class="btn-close btn-close-white" @click="showPicker = false"></button>
					</div>
					<div class="modal-body">
						<!-- Search + filters -->
						<div class="row g-2 mb-3">
							<div class="col-md-8">
								<input
									v-model="pickerSearch"
									type="text"
									class="form-control form-control-sm bg-dark text-white border-secondary"
									placeholder="Search by title, performer, studio…"
									@input="debouncedSearch"
								/>
							</div>
							<div class="col-md-4">
								<select v-model="pickerLibraryId" class="form-select form-select-sm bg-dark text-white border-secondary" @change="loadPickerVideos(1)">
									<option value="">All Libraries</option>
									<option v-for="lib in libraries" :key="lib.id" :value="lib.id">{{ lib.name }}</option>
								</select>
							</div>
						</div>

						<!-- Loading -->
						<div v-if="pickerLoading" class="text-center py-4">
							<font-awesome-icon :icon="['fas', 'spinner']" spin class="text-muted" />
						</div>

						<!-- Empty state -->
						<div v-else-if="pickerVideos.length === 0" class="text-muted text-center py-4">
							No videos found
						</div>

						<!-- Video list -->
						<div v-else>
							<div
								v-for="video in pickerVideos"
								:key="video.id"
								class="picker-row d-flex align-items-center gap-2 p-2 rounded mb-1 border"
								:class="pickerSelected.has(video.id) ? 'border-primary bg-dark' : 'border-secondary bg-dark'"
								style="cursor:pointer"
								@click="togglePickerSelect(video.id)"
							>
								<input
									type="checkbox"
									:checked="pickerSelected.has(video.id)"
									class="form-check-input flex-shrink-0"
									@click.stop="togglePickerSelect(video.id)"
								/>
								<!-- Thumb -->
								<div class="flex-shrink-0" style="position:relative;overflow:hidden;width:56px;height:42px;background:#1a1a1a;border-radius:3px">
									<div class="w-100 h-100 d-flex align-items-center justify-content-center">
										<font-awesome-icon :icon="['fas', 'film']" class="text-secondary" style="font-size:0.8rem" />
									</div>
									<img
										v-if="video.thumbnail_path"
										:src="getThumbnailUrl(video.thumbnail_path)"
										style="position:absolute;inset:0;width:100%;height:100%;object-fit:cover"
										@error="e => e.target.remove()"
										alt=""
									/>
								</div>
								<!-- Info -->
								<div class="flex-fill min-width-0">
									<div class="text-truncate small fw-semibold" :title="video.title">{{ video.title }}</div>
									<div class="text-muted" style="font-size:0.7rem">
										<span v-if="video.resolution" class="me-2">{{ video.resolution }}</span>
										<span v-if="video.codec" class="me-2 text-uppercase">{{ video.codec }}</span>
										<span v-if="video.file_size" class="me-2">{{ formatBytes(video.file_size) }}</span>
										<span v-if="video.duration" class="me-2">{{ formatDuration(video.duration) }}</span>
									</div>
								</div>
								<!-- Already-added indicator -->
								<span v-if="isAlreadyAdded(video.id)" class="badge bg-success flex-shrink-0" style="font-size:0.65rem">Added</span>
							</div>

							<!-- Pagination -->
							<div v-if="pickerTotalPages > 1" class="d-flex justify-content-center gap-2 mt-3">
								<button class="btn btn-sm btn-outline-secondary" :disabled="pickerPage <= 1" @click="loadPickerVideos(pickerPage - 1)">‹</button>
								<span class="small text-muted align-self-center">{{ pickerPage }} / {{ pickerTotalPages }}</span>
								<button class="btn btn-sm btn-outline-secondary" :disabled="pickerPage >= pickerTotalPages" @click="loadPickerVideos(pickerPage + 1)">›</button>
							</div>
						</div>
					</div>
					<div class="modal-footer border-secondary">
						<span class="text-muted small me-auto">{{ pickerSelected.size }} selected</span>
						<button class="btn btn-secondary btn-sm" @click="showPicker = false">Cancel</button>
						<button class="btn btn-primary btn-sm" :disabled="pickerSelected.size === 0" @click="addPickerSelection">
							Add {{ pickerSelected.size || '' }} Selected
						</button>
					</div>
				</div>
			</div>
		</div>
		<div v-if="showPicker" class="modal-backdrop fade show"></div>
	</div>
</template>

<script>
import { videosAPI, librariesAPI, getAssetURL } from '@/services/api'

const PICKER_PAGE_SIZE = 24

export default {
	name: 'VideoAnalyzerPage',

	data() {
		return {
			selectedVideos: [],
			isDragOver: false,
			comparing: false,
			result: null,
			confirmDelete: null,

			// Library picker
			showPicker: false,
			pickerSearch: '',
			pickerLibraryId: '',
			pickerVideos: [],
			pickerSelected: new Set(),
			pickerLoading: false,
			pickerPage: 1,
			pickerTotal: 0,
			libraries: [],
		}
	},

	computed: {
		pickerTotalPages() {
			return Math.max(1, Math.ceil(this.pickerTotal / PICKER_PAGE_SIZE))
		},
	},

	async mounted() {
		const res = await librariesAPI.getAll()
		this.libraries = res.data || res || []
	},

	watch: {
		showPicker(val) {
			if (val) {
				this.pickerSelected = new Set()
				this.loadPickerVideos(1)
			}
		},
	},

	methods: {
		// ── Picker ────────────────────────────────────────────────────────────

		debouncedSearch() {
			clearTimeout(this.searchTimer)
			this.searchTimer = setTimeout(() => this.loadPickerVideos(1), 300)
		},

		async loadPickerVideos(page) {
			this.pickerPage = page
			this.pickerLoading = true
			try {
				const params = { page, limit: PICKER_PAGE_SIZE }
				if (this.pickerSearch.trim()) params.query = this.pickerSearch.trim()
				if (this.pickerLibraryId) params.library_id = this.pickerLibraryId
				const res = await videosAPI.getAll(params)
				this.pickerVideos = res.data || []
				this.pickerTotal = res.total || 0
			} catch {
				this.pickerVideos = []
				this.pickerTotal = 0
			} finally {
				this.pickerLoading = false
			}
		},

		togglePickerSelect(id) {
			const next = new Set(this.pickerSelected)
			if (next.has(id)) {
				next.delete(id)
			} else {
				next.add(id)
			}
			this.pickerSelected = next
		},

		isAlreadyAdded(id) {
			return this.selectedVideos.some(v => v.source === 'library' && v.id === id)
		},

		addPickerSelection() {
			const toAdd = this.pickerVideos.filter(v => this.pickerSelected.has(v.id) && !this.isAlreadyAdded(v.id))
			for (const v of toAdd) {
				this.selectedVideos.push({
					key: 'lib_' + v.id,
					source: 'library',
					id: v.id,
					title: v.title,
					thumbnail_path: v.thumbnail_path || '',
					file_path: v.file_path || '',
					resolution: v.resolution || '',
					codec: v.codec || '',
					fps: v.fps || 0,
					bitrate: v.bitrate || 0,
					file_size: v.file_size || 0,
					duration: v.duration || 0,
				})
			}
			this.showPicker = false
		},

		// ── Upload ────────────────────────────────────────────────────────────

		onFilesSelected(e) {
			this.addFiles(Array.from(e.target.files))
			e.target.value = ''
		},

		onDrop(e) {
			this.isDragOver = false
			const files = Array.from(e.dataTransfer.files).filter(f => f.type.startsWith('video/') || /\.(mp4|mkv|avi|mov|wmv|flv|webm|m4v|mpg|mpeg|ts)$/i.test(f.name))
			this.addFiles(files)
		},

		addFiles(files) {
			for (const file of files) {
				const key = 'upload_' + file.name + '_' + file.size
				if (this.selectedVideos.some(v => v.key === key)) continue
				this.selectedVideos.push({
					key,
					source: 'upload',
					id: 0,
					title: file.name,
					thumbnail_path: '',
					file_path: file.name,
					resolution: '',
					codec: '',
					fps: 0,
					bitrate: 0,
					file_size: file.size,
					duration: 0,
					_file: file,
				})
			}
		},

		removeVideo(idx) {
			this.selectedVideos.splice(idx, 1)
		},

		// ── Compare ───────────────────────────────────────────────────────────

		async compareAll() {
			if (this.selectedVideos.length < 2) return
			this.comparing = true
			this.result = null
			try {
				const formData = new FormData()
				const libraryIds = this.selectedVideos.filter(v => v.source === 'library').map(v => v.id)
				if (libraryIds.length > 0) {
					formData.append('video_ids', JSON.stringify(libraryIds))
				}
				for (const v of this.selectedVideos) {
					if (v.source === 'upload' && v._file) {
						formData.append('file', v._file)
					}
				}
				const res = await videosAPI.analyze(formData)
				this.result = res.data || res
			} catch (err) {
				this.$toast.error('Analysis failed', err?.response?.data?.message || err?.message || '', 5000)
			} finally {
				this.comparing = false
			}
		},

		// ── Delete ───────────────────────────────────────────────────────────

		async deleteVideo(videoId) {
			try {
				await videosAPI.delete(videoId)
				this.result.videos = this.result.videos.filter(v => v.video_id !== videoId)
				this.selectedVideos = this.selectedVideos.filter(v => !(v.source === 'library' && v.id === videoId))
				this.$toast.success('Video deleted', '', 3000)
			} catch (err) {
				this.$toast.error('Delete failed', err?.response?.data?.message || '', 5000)
			} finally {
				this.confirmDelete = null
			}
		},

		// ── Helpers ───────────────────────────────────────────────────────────

		getThumbnailUrl(path) {
			if (!path) return ''
			const encoded = path.split('/').map(s => encodeURIComponent(s)).join('/')
			return getAssetURL(encoded)
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
			if (!bytes) return ''
			if (bytes >= 1_073_741_824) return (bytes / 1_073_741_824).toFixed(1) + ' GB'
			if (bytes >= 1_048_576) return (bytes / 1_048_576).toFixed(1) + ' MB'
			return (bytes / 1024).toFixed(0) + ' KB'
		},

		formatDuration(secs) {
			if (!secs) return ''
			const h = Math.floor(secs / 3600)
			const m = Math.floor((secs % 3600) / 60)
			const s = Math.floor(secs % 60)
			if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
			return `${m}:${String(s).padStart(2, '0')}`
		},

		formatFPS(fps) {
			if (!fps) return ''
			return fps % 1 === 0 ? fps + ' fps' : fps.toFixed(2) + ' fps'
		},

		formatBitrate(bps) {
			if (!bps) return ''
			if (bps >= 1_000_000) return (bps / 1_000_000).toFixed(1) + ' Mbps'
			return Math.round(bps / 1000) + ' Kbps'
		},
	},
}
</script>

<style scoped>
.drop-zone {
	min-height: 200px;
	transition: border-color 0.2s, background 0.2s;
}
.drop-zone-mini {
	min-height: 40px;
	transition: border-color 0.2s, background 0.2s;
}
.drop-zone.drag-over,
.drop-zone-mini.drag-over {
	border-color: #0dcaf0 !important;
	background: rgba(13, 202, 240, 0.05);
}
.border-dashed {
	border-style: dashed !important;
}
.selected-card:hover {
	border-color: #6c757d !important;
}
.picker-row:hover {
	background: #1e1e1e !important;
}
.min-width-0 {
	min-width: 0;
}
.result-row {
	transition: border-color 0.2s;
}
.result-action-btn {
	font-size: 0.72rem;
	padding: 2px 10px;
}
</style>
