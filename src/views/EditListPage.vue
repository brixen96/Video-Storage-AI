<template>
	<div class="edit-list-page">
		<div class="container-fluid py-4">
			<!-- Page Header -->
			<div class="page-header mb-4">
				<div class="header-content">
					<h1>
						<font-awesome-icon :icon="['fas', 'list-check']" class="me-3" />
						Edit List
					</h1>
					<p class="lead">Manage videos queued for editing and processing</p>
				</div>
				<div class="header-actions">
					<button v-if="selectedVideos.length > 0" class="btn btn-danger me-2" @click="removeSelectedFromList">
						<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
						Remove Selected ({{ selectedVideos.length }})
					</button>
					<button v-if="videos.length > 0" class="btn btn-warning me-2" @click="clearEditList">
						<font-awesome-icon :icon="['fas', 'broom']" class="me-2" />
						Clear All
					</button>
					<button class="btn btn-primary" @click="refreshList">
						<font-awesome-icon :icon="['fas', 'sync']" :spin="loading" class="me-2" />
						Refresh
					</button>
				</div>
			</div>

			<!-- Stats Summary -->
			<div class="stats-summary mb-4">
				<div class="row g-3">
					<div class="col-md-3">
						<div class="stat-card">
							<div class="stat-icon bg-primary">
								<font-awesome-icon :icon="['fas', 'video']" />
							</div>
							<div class="stat-content">
								<div class="stat-value">{{ videos.length }}</div>
								<div class="stat-label">Videos in List</div>
							</div>
						</div>
					</div>
					<div class="col-md-3">
						<div class="stat-card">
							<div class="stat-icon bg-success">
								<font-awesome-icon :icon="['fas', 'check-circle']" />
							</div>
							<div class="stat-content">
								<div class="stat-value">{{ selectedVideos.length }}</div>
								<div class="stat-label">Selected</div>
							</div>
						</div>
					</div>
					<div class="col-md-3">
						<div class="stat-card">
							<div class="stat-icon bg-info">
								<font-awesome-icon :icon="['fas', 'hdd']" />
							</div>
							<div class="stat-content">
								<div class="stat-value">{{ formatTotalSize(totalSize) }}</div>
								<div class="stat-label">Total Size</div>
							</div>
						</div>
					</div>
					<div class="col-md-3">
						<div class="stat-card">
							<div class="stat-icon bg-warning">
								<font-awesome-icon :icon="['fas', 'clock']" />
							</div>
							<div class="stat-content">
								<div class="stat-value">{{ formatTotalDuration(totalDuration) }}</div>
								<div class="stat-label">Total Duration</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Filters and Controls -->
			<div class="filters-bar mb-4">
				<div class="row g-3">
					<div class="col-md-4">
						<div class="input-group">
							<span class="input-group-text">
								<font-awesome-icon :icon="['fas', 'search']" />
							</span>
							<input v-model="searchQuery" type="text" class="form-control" placeholder="Search videos..." @input="filterVideos" />
							<button v-if="searchQuery" class="btn btn-outline-secondary" @click="searchQuery = ''; filterVideos()">
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
						</div>
					</div>
					<div class="col-md-2">
						<select v-model="sortBy" class="form-select" @change="sortVideos">
							<option value="added_date">Date Added</option>
							<option value="title">Title</option>
							<option value="duration">Duration</option>
							<option value="size">File Size</option>
							<option value="rating">Rating</option>
						</select>
					</div>
					<div class="col-md-2">
						<select v-model="sortOrder" class="form-select" @change="sortVideos">
							<option value="desc">Descending</option>
							<option value="asc">Ascending</option>
						</select>
					</div>
					<div class="col-md-4 text-end">
						<div class="btn-group" role="group">
							<input id="viewGrid" v-model="viewMode" type="radio" class="btn-check" value="grid" />
							<label class="btn btn-outline-primary" for="viewGrid">
								<font-awesome-icon :icon="['fas', 'th']" />
							</label>
							<input id="viewList" v-model="viewMode" type="radio" class="btn-check" value="list" />
							<label class="btn btn-outline-primary" for="viewList">
								<font-awesome-icon :icon="['fas', 'list']" />
							</label>
						</div>
					</div>
				</div>
			</div>

			<!-- Bulk Selection -->
			<div v-if="filteredVideos.length > 0" class="bulk-actions mb-3">
				<div class="form-check">
					<input id="selectAll" v-model="selectAll" type="checkbox" class="form-check-input" @change="toggleSelectAll" />
					<label class="form-check-label" for="selectAll"> Select All ({{ filteredVideos.length }}) </label>
				</div>
			</div>

			<!-- Videos Display -->
			<div v-if="loading" class="loading-state">
				<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" class="text-primary" />
				<p class="mt-3">Loading edit list...</p>
			</div>

			<div v-else-if="filteredVideos.length === 0" class="empty-state">
				<font-awesome-icon :icon="['fas', 'list-check']" size="5x" class="text-muted mb-3" />
				<h3>{{ searchQuery ? 'No videos found' : 'Edit List is Empty' }}</h3>
				<p class="text-muted">{{ searchQuery ? 'Try adjusting your search query' : 'Add videos to your edit list from the Videos or Browser page' }}</p>
			</div>

			<!-- Grid View -->
			<div v-else-if="viewMode === 'grid'" class="videos-grid">
				<div
					v-for="video in filteredVideos"
					:key="video.id"
					class="video-card"
					:class="{ selected: isSelected(video.id) }"
					@click="toggleSelection(video.id)"
				>
					<div class="selection-checkbox">
						<input type="checkbox" :checked="isSelected(video.id)" @click.stop="toggleSelection(video.id)" />
					</div>

					<div class="video-thumbnail">
						<img v-if="video.thumbnail_path" :src="getThumbnailURL(video)" :alt="video.title" loading="lazy" />
						<div v-else class="thumbnail-placeholder">
							<font-awesome-icon :icon="['fas', 'video']" size="3x" />
						</div>
						<div v-if="video.duration" class="badge-duration">{{ formatDuration(video.duration) }}</div>
					</div>

					<div class="video-info">
						<h3 class="video-title" :title="video.title">{{ video.title }}</h3>

						<div class="video-badges">
							<span v-if="video.resolution" class="badge bg-primary">{{ video.resolution }}</span>
							<span v-if="video.file_size" class="badge bg-secondary">{{ formatFileSize(video.file_size) }}</span>
							<span v-if="video.rating" class="badge bg-warning">
								<font-awesome-icon :icon="['fas', 'star']" />
								{{ video.rating }}
							</span>
						</div>

						<div class="video-actions mt-2">
							<button class="btn btn-sm btn-primary me-1" @click.stop="openVideo(video)">
								<font-awesome-icon :icon="['fas', 'play']" />
							</button>
							<button class="btn btn-sm btn-danger" @click.stop="removeFromList(video.id)">
								<font-awesome-icon :icon="['fas', 'trash']" />
							</button>
						</div>
					</div>
				</div>
			</div>

			<!-- List View -->
			<div v-else class="videos-list">
				<div class="list-header">
					<div class="col-select"></div>
					<div class="col-title">Title</div>
					<div class="col-duration">Duration</div>
					<div class="col-size">Size</div>
					<div class="col-rating">Rating</div>
					<div class="col-added">Added</div>
					<div class="col-actions">Actions</div>
				</div>
				<div
					v-for="video in filteredVideos"
					:key="video.id"
					class="list-item"
					:class="{ selected: isSelected(video.id) }"
					@click="toggleSelection(video.id)"
				>
					<div class="col-select">
						<input type="checkbox" :checked="isSelected(video.id)" @click.stop="toggleSelection(video.id)" />
					</div>
					<div class="col-title">
						<img v-if="video.thumbnail_path" :src="getThumbnailURL(video)" :alt="video.title" class="list-thumbnail" />
						<div v-else class="list-thumbnail-placeholder">
							<font-awesome-icon :icon="['fas', 'video']" />
						</div>
						<span class="title-text" :title="video.title">{{ video.title }}</span>
					</div>
					<div class="col-duration">{{ video.duration ? formatDuration(video.duration) : '-' }}</div>
					<div class="col-size">{{ video.file_size ? formatFileSize(video.file_size) : '-' }}</div>
					<div class="col-rating">
						<span v-if="video.rating" class="badge bg-warning">
							<font-awesome-icon :icon="['fas', 'star']" />
							{{ video.rating }}
						</span>
						<span v-else>-</span>
					</div>
					<div class="col-added">{{ formatDate(video.created_at) }}</div>
					<div class="col-actions">
						<button class="btn btn-sm btn-primary me-1" @click.stop="openVideo(video)">
							<font-awesome-icon :icon="['fas', 'play']" />
						</button>
						<button class="btn btn-sm btn-danger" @click.stop="removeFromList(video.id)">
							<font-awesome-icon :icon="['fas', 'trash']" />
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Video Player Modal -->
		<VideoPlayer :visible="playerVisible" :video="selectedVideo" @close="playerVisible = false" />
	</div>
</template>

<script>
import { videosAPI } from '@/services/api'
import { getAssetURL } from '@/services/api'
import VideoPlayer from '@/components/VideoPlayer.vue'

export default {
	name: 'EditListPage',
	components: {
		VideoPlayer,
	},
	data() {
		return {
			videos: [],
			filteredVideos: [],
			loading: false,
			searchQuery: '',
			sortBy: 'added_date',
			sortOrder: 'desc',
			viewMode: 'grid',
			selectedVideos: [],
			selectAll: false,
			playerVisible: false,
			selectedVideo: null,
		}
	},
	computed: {
		totalSize() {
			return this.videos.reduce((sum, v) => sum + (v.file_size || 0), 0)
		},
		totalDuration() {
			return this.videos.reduce((sum, v) => sum + (v.duration || 0), 0)
		},
	},
	async mounted() {
		await this.loadEditList()
	},
	methods: {
		async loadEditList() {
			this.loading = true
			try {
				const response = await videosAPI.getAll({ in_edit_list: true, per_page: 1000 })
				this.videos = Array.isArray(response.data) ? response.data : response.data?.data || []
				this.filterVideos()
			} catch (error) {
				console.error('Failed to load edit list:', error)
				this.$toast.error('Load Failed', 'Could not load edit list')
			} finally {
				this.loading = false
			}
		},
		filterVideos() {
			if (!this.searchQuery.trim()) {
				this.filteredVideos = [...this.videos]
			} else {
				const query = this.searchQuery.toLowerCase()
				this.filteredVideos = this.videos.filter((v) => v.title.toLowerCase().includes(query) || v.file_path.toLowerCase().includes(query))
			}
			this.sortVideos()
		},
		sortVideos() {
			const order = this.sortOrder === 'asc' ? 1 : -1
			this.filteredVideos.sort((a, b) => {
				let aVal, bVal
				switch (this.sortBy) {
					case 'title':
						aVal = a.title.toLowerCase()
						bVal = b.title.toLowerCase()
						return aVal.localeCompare(bVal) * order
					case 'duration':
						aVal = a.duration || 0
						bVal = b.duration || 0
						return (aVal - bVal) * order
					case 'size':
						aVal = a.file_size || 0
						bVal = b.file_size || 0
						return (aVal - bVal) * order
					case 'rating':
						aVal = a.rating || 0
						bVal = b.rating || 0
						return (aVal - bVal) * order
					case 'added_date':
					default:
						aVal = new Date(a.created_at || 0)
						bVal = new Date(b.created_at || 0)
						return (aVal - bVal) * order
				}
			})
		},
		isSelected(videoId) {
			return this.selectedVideos.includes(videoId)
		},
		toggleSelection(videoId) {
			const index = this.selectedVideos.indexOf(videoId)
			if (index > -1) {
				this.selectedVideos.splice(index, 1)
			} else {
				this.selectedVideos.push(videoId)
			}
			this.updateSelectAll()
		},
		toggleSelectAll() {
			if (this.selectAll) {
				this.selectedVideos = this.filteredVideos.map((v) => v.id)
			} else {
				this.selectedVideos = []
			}
		},
		updateSelectAll() {
			this.selectAll = this.filteredVideos.length > 0 && this.selectedVideos.length === this.filteredVideos.length
		},
		async removeFromList(videoId) {
			if (!confirm('Remove this video from the edit list?')) return

			try {
				const video = this.videos.find((v) => v.id === videoId)
				await videosAPI.update(videoId, { in_edit_list: false })
				this.videos = this.videos.filter((v) => v.id !== videoId)
				this.selectedVideos = this.selectedVideos.filter((id) => id !== videoId)
				this.filterVideos()
				this.$toast.success('Removed', `"${video.title}" removed from edit list`)
			} catch (error) {
				console.error('Failed to remove from edit list:', error)
				this.$toast.error('Remove Failed', 'Could not remove video from edit list')
			}
		},
		async removeSelectedFromList() {
			if (!confirm(`Remove ${this.selectedVideos.length} videos from the edit list?`)) return

			try {
				for (const videoId of this.selectedVideos) {
					await videosAPI.update(videoId, { in_edit_list: false })
				}
				this.videos = this.videos.filter((v) => !this.selectedVideos.includes(v.id))
				const count = this.selectedVideos.length
				this.selectedVideos = []
				this.filterVideos()
				this.$toast.success('Removed', `${count} videos removed from edit list`)
			} catch (error) {
				console.error('Failed to remove videos:', error)
				this.$toast.error('Remove Failed', 'Could not remove videos from edit list')
			}
		},
		async clearEditList() {
			if (!confirm(`Clear all ${this.videos.length} videos from the edit list?`)) return

			try {
				for (const video of this.videos) {
					await videosAPI.update(video.id, { in_edit_list: false })
				}
				const count = this.videos.length
				this.videos = []
				this.selectedVideos = []
				this.filterVideos()
				this.$toast.success('Cleared', `Edit list cleared (${count} videos removed)`)
			} catch (error) {
				console.error('Failed to clear edit list:', error)
				this.$toast.error('Clear Failed', 'Could not clear edit list')
			}
		},
		async refreshList() {
			await this.loadEditList()
			this.$toast.success('Refreshed', 'Edit list reloaded')
		},
		openVideo(video) {
			this.selectedVideo = video
			this.playerVisible = true
		},
		getThumbnailURL(video) {
			if (video.thumbnail_path) {
				return getAssetURL(video.thumbnail_path)
			}
			return `http://localhost:8080/api/v1/videos/${video.id}/thumbnail`
		},
		formatDuration(seconds) {
			const mins = Math.floor(seconds / 60)
			const secs = Math.floor(seconds % 60)
			return `${mins}:${secs.toString().padStart(2, '0')}`
		},
		formatFileSize(bytes) {
			if (bytes < 1024) return bytes + ' B'
			if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
			if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
			return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
		},
		formatTotalSize(bytes) {
			if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(0) + ' MB'
			return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
		},
		formatTotalDuration(seconds) {
			const hours = Math.floor(seconds / 3600)
			const mins = Math.floor((seconds % 3600) / 60)
			if (hours > 0) return `${hours}h ${mins}m`
			return `${mins}m`
		},
		formatDate(dateString) {
			if (!dateString) return '-'
			const date = new Date(dateString)
			return date.toLocaleDateString()
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/edit_list_page.css';
</style>
