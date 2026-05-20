<template>
	<div class="details-panel" @click.self="$emit('close')">
		<div class="panel-content">
			<!-- Carousel -->
			<div class="panel-carousel">
				<div v-if="getPerformerPreviews(performer).length > 0" class="carousel-container">
					<button v-if="carouselIndex > 0" class="carousel-btn prev" @click="carouselIndex--">
						<font-awesome-icon :icon="['fas', 'chevron-left']" />
					</button>
					<video
						:src="getPerformerPreviews(performer)[carouselIndex]"
						class="carousel-video"
						controls
						autoplay
						loop
						muted
						@contextmenu.prevent="openCarouselContextMenu($event, carouselIndex)"
					></video>
					<button v-if="carouselIndex < getPerformerPreviews(performer).length - 1" class="carousel-btn next" @click="carouselIndex++">
						<font-awesome-icon :icon="['fas', 'chevron-right']" />
					</button>
					<div class="carousel-indicator">{{ carouselIndex + 1 }} / {{ getPerformerPreviews(performer).length }}</div>
				</div>
				<div v-else class="no-carousel">
					<font-awesome-icon :icon="['fas', 'user']" size="5x" />
					<p class="mt-3">No preview videos available</p>
				</div>
			</div>

			<!-- Metadata Tabs -->
			<div class="panel-metadata">
				<ul class="nav nav-tabs">
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'basic' }]" @click="activeTab = 'basic'">Basic</a>
					</li>
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'appearance' }]" @click="activeTab = 'appearance'">Appearance</a>
					</li>
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'social_media' }]" @click="activeTab = 'social_media'">Social Media</a>
					</li>
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'platform' }]" @click="activeTab = 'platform'">Platform</a>
					</li>
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'tags' }]" @click="activeTab = 'tags'">Tags</a>
					</li>
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'bios' }]" @click="activeTab = 'bios'">Bios</a>
					</li>
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'external_links' }]" @click="activeTab = 'external_links'">Links</a>
					</li>
					<li class="nav-item">
						<a :class="['nav-link', { active: activeTab === 'videos' }]" @click="activeTab = 'videos'">Videos ({{ performer.video_count || 0 }})</a>
					</li>
				</ul>

				<div class="tab-content mt-3">
					<!-- Basic Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'basic' }]">
						<div class="metadata-grid">
							<div class="metadata-item">
								<span class="metadata-label">Name:</span>
								<span class="metadata-value">{{ performer.name }}</span>
							</div>
							<div v-if="getAge(performer)" class="metadata-item">
								<span class="metadata-label">Age:</span>
								<span class="metadata-value">{{ getAge(performer) }}</span>
							</div>
							<div v-if="performer.metadata?.birthdate" class="metadata-item">
								<span class="metadata-label">Birthdate:</span>
								<span class="metadata-value">{{ formatDate(performer.metadata.birthdate) }}</span>
							</div>
							<div v-if="performer.metadata?.birthplace" class="metadata-item">
								<span class="metadata-label">Birthplace:</span>
								<span class="metadata-value">{{ performer.metadata.birthplace }}</span>
							</div>
							<div v-if="performer.metadata?.career_start" class="metadata-item">
								<span class="metadata-label">Career Start:</span>
								<span class="metadata-value">{{ performer.metadata.career_start }}</span>
							</div>
							<div v-if="performer.metadata?.career_end" class="metadata-item">
								<span class="metadata-label">Career End:</span>
								<span class="metadata-value">{{ performer.metadata.career_end }}</span>
							</div>
							<div class="metadata-item">
								<span class="metadata-label">Video Count:</span>
								<span class="metadata-value">{{ performer.video_count || 0 }}</span>
							</div>
							<div v-if="performer.metadata?.aliases?.length" class="metadata-item full-width">
								<span class="metadata-label">Aliases:</span>
								<span class="metadata-value">{{ performer.metadata.aliases.join(', ') }}</span>
							</div>
							<div v-if="performer.category" class="metadata-item full-width">
								<span class="metadata-label">Category:</span>
								<span :class="['metadata-value', 'category-badge', 'category-' + performer.category]">{{ getCategoryName(performer.category) }}</span>
							</div>
						</div>
					</div>

					<!-- Appearance Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'appearance' }]">
						<div v-if="getAppearanceData(performer)" class="metadata-grid">
							<div v-for="(value, key) in getAppearanceData(performer)" :key="key" class="metadata-item">
								<span class="metadata-label">{{ formatLabel(key) }}:</span>
								<span class="metadata-value">{{ value }}</span>
							</div>
						</div>
						<div v-else class="empty-tab"><p>No appearance data available</p></div>
					</div>

					<!-- Social Media Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'social_media' }]">
						<div v-if="getSocialMediaData(performer)" class="metadata-grid">
							<div v-for="(value, key) in getSocialMediaData(performer)" :key="key" class="metadata-item">
								<span class="metadata-label">{{ formatLabel(key) }}:</span>
								<span class="metadata-value">
									<a v-if="isURL(value)" :href="value" target="_blank" rel="noopener">{{ value }}</a>
									<span v-else>{{ formatValue(value) }}</span>
								</span>
							</div>
						</div>
						<div v-else class="empty-tab"><p>No social media data available</p></div>
					</div>

					<!-- Platform Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'platform' }]">
						<div v-if="getPlatformData(performer)" class="metadata-grid">
							<div v-for="(value, key) in getPlatformData(performer)" :key="key" class="metadata-item">
								<span class="metadata-label">{{ formatLabel(key) }}:</span>
								<span class="metadata-value">{{ formatValue(value) }}</span>
							</div>
						</div>
						<div v-else class="empty-tab"><p>No platform data available</p></div>
					</div>

					<!-- Tags Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'tags' }]">
						<div class="tags-management">
							<div class="add-tag-section mb-3">
								<div class="input-group">
									<select v-model.number="selectedTagId" class="form-select">
										<option value="">Select a tag...</option>
										<option v-for="tag in availableTagsForPerformer" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
									</select>
									<button class="btn btn-primary" @click="addTagToPerformer" :disabled="!selectedTagId">
										<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
										Add Tag
									</button>
								</div>
							</div>
							<div v-if="performerMasterTags.length > 0" class="tags-container">
								<div v-for="tag in performerMasterTags" :key="tag.id" class="tag-badge-removable">
									<span>{{ tag.name }}</span>
									<button class="btn-remove-tag-small" @click="removeTagFromPerformer(tag.id)" title="Remove tag">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>
							</div>
							<div v-else class="empty-tab"><p>No master tags assigned</p></div>
							<div v-if="performerMasterTags.length > 0" class="mt-3">
								<button class="btn btn-outline-primary btn-sm w-100" @click="syncPerformerTags" :disabled="isSyncingTags">
									<font-awesome-icon :icon="['fas', isSyncingTags ? 'spinner' : 'sync']" :spin="isSyncingTags" class="me-2" />
									{{ isSyncingTags ? 'Syncing...' : `Sync Tags to ${performer.video_count || 0} Videos` }}
								</button>
							</div>
						</div>
					</div>

					<!-- Bios Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'bios' }]">
						<div v-if="getBiosData(performer)" class="bios-container">
							<div v-for="(bio, source) in getBiosData(performer)" :key="source" class="bio-item">
								<h6 class="bio-source">{{ formatLabel(source) }}</h6>
								<p class="bio-text">{{ bio }}</p>
							</div>
						</div>
						<div v-else class="empty-tab"><p>No bios available</p></div>
					</div>

					<!-- External Links Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'external_links' }]">
						<div v-if="getExternalLinksData(performer)?.length" class="links-container">
							<a
								v-for="(link, index) in getExternalLinksData(performer)"
								:key="index"
								:href="getLinkURL(link)"
								target="_blank"
								rel="noopener"
								class="external-link"
							>
								<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-2" />
								{{ getLinkTitle(link) }}
							</a>
						</div>
						<div v-else class="empty-tab"><p>No external links available</p></div>
					</div>

					<!-- Videos Tab -->
					<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'videos' }]">
						<div v-if="performerVideos.length > 0" class="videos-grid">
							<div v-for="video in performerVideos" :key="video.id" class="video-item" @click="openVideo(video)">
								<div class="video-thumbnail">
									<img v-if="video.thumbnail_path" :src="`http://localhost:8080/assets/${video.thumbnail_path}`" :alt="video.title" loading="lazy" />
									<div v-else class="no-thumbnail">
										<font-awesome-icon :icon="['fas', 'film']" size="2x" />
									</div>
									<div class="video-duration">{{ formatDuration(video.duration) }}</div>
								</div>
								<div class="video-info">
									<h6 class="video-title">{{ video.title }}</h6>
									<p class="video-meta">{{ formatFileSize(video.file_size) }}</p>
								</div>
							</div>
						</div>
						<div v-else-if="loadingVideos" class="empty-tab">
							<font-awesome-icon :icon="['fas', 'spinner']" spin size="2x" />
							<p class="mt-3">Loading videos...</p>
						</div>
						<div v-else class="empty-tab"><p>No videos found for this performer</p></div>
					</div>
				</div>
			</div>

			<!-- Action Buttons -->
			<div class="panel-actions">
				<button class="btn btn-primary" @click="$emit('fetch-metadata', performer)">
					<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
					Fetch Metadata
				</button>
				<button class="btn btn-secondary" @click="$emit('reset', performer)">
					<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
					Reset
				</button>
				<button class="btn btn-danger" @click="$emit('confirm-delete', performer)">
					<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
					Delete
				</button>
			</div>
		</div>
	</div>

	<!-- Carousel Context Menu -->
	<div v-if="carouselContextMenu.visible" class="context-menu" :style="{ top: carouselContextMenu.y + 'px', left: carouselContextMenu.x + 'px' }" @click.stop>
		<button class="context-menu-item" @click="setAsPrimaryPreview">
			<font-awesome-icon :icon="['fas', 'star']" class="me-2" />
			Set as Primary
		</button>
	</div>
</template>

<script>
import { performersAPI } from '@/services/api'
import { useFormatters } from '@/composables/useFormatters'

export default {
	name: 'PerformerDetailsPanel',
	props: {
		performer: { type: Object, required: true },
		allTags: { type: Array, default: () => [] },
	},
	emits: ['close', 'fetch-metadata', 'reset', 'confirm-delete', 'performer-updated'],
	data() {
		let cachedPreviews = {}
		try {
			const stored = localStorage.getItem('performerPreviews')
			if (stored) {
				const parsed = JSON.parse(stored)
				let needsUpdate = false
				for (const [performerId, paths] of Object.entries(parsed)) {
					if (Array.isArray(paths)) {
						const cleanedPaths = paths
							.map((path) => {
								if (!path || typeof path !== 'string') return null
								if (path.startsWith('http://') || path.startsWith('https://')) {
									try {
										return new URL(path).pathname
									} catch {
										return null
									}
								}
								if (path.startsWith('/../')) {
									needsUpdate = true
									return null
								}
								return path
							})
							.filter((path) => path !== null)
						if (cleanedPaths.length > 0) {
							cachedPreviews[performerId] = cleanedPaths
						} else {
							needsUpdate = true
						}
					}
				}
				if (needsUpdate) {
					try {
						localStorage.setItem('performerPreviews', JSON.stringify(cachedPreviews))
					} catch (e) {
						console.error('Failed to save cleaned cache:', e)
					}
				}
			}
		} catch (e) {
			console.error('Failed to load preview cache:', e)
			localStorage.removeItem('performerPreviews')
		}

		return {
			carouselIndex: 0,
			activeTab: 'basic',
			selectedTagId: null,
			performerMasterTags: [],
			isSyncingTags: false,
			performerVideos: [],
			loadingVideos: false,
			carouselContextMenu: { visible: false, x: 0, y: 0, previewIndex: 0 },
			performerPreviews: cachedPreviews,
		}
	},
	computed: {
		availableTagsForPerformer() {
			if (!this.performer) return []
			const performerCategory = this.performer.category || 'regular'
			if (!Array.isArray(this.performerMasterTags)) {
				return (this.allTags || []).filter((tag) => tag.category === performerCategory)
			}
			const assignedTagIds = new Set(this.performerMasterTags.map((t) => t.id))
			return (this.allTags || []).filter((tag) => tag.category === performerCategory && !assignedTagIds.has(tag.id))
		},
	},
	watch: {
		'performer.id'(newId) {
			if (newId) {
				this.carouselIndex = 0
				this.activeTab = 'basic'
				this.selectedTagId = null
				this.performerVideos = []
				this.loadPerformerPreviews(newId)
				this.loadPerformerTags(newId)
				this.loadPerformerVideos(newId)
			}
		},
	},
	created() {
		const formatters = useFormatters()
		this.formatDuration = formatters.formatDuration
		this.formatFileSize = formatters.formatFileSize
	},
	mounted() {
		if (this.performer?.id) {
			this.loadPerformerPreviews(this.performer.id)
			this.loadPerformerTags(this.performer.id)
			this.loadPerformerVideos(this.performer.id)
		}
		document.addEventListener('click', this.handleDocumentClick)
	},
	beforeUnmount() {
		document.removeEventListener('click', this.handleDocumentClick)
	},
	methods: {
		getCategoryName(category) {
			const names = { regular: 'Regular', zoo: 'Zoo', '3d': '3D' }
			return names[category] || 'Regular'
		},
		getAge(performer) {
			if (!performer.metadata?.birthdate) return null
			const birthDate = new Date(performer.metadata.birthdate)
			const today = new Date()
			let age = today.getFullYear() - birthDate.getFullYear()
			const monthDiff = today.getMonth() - birthDate.getMonth()
			if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birthDate.getDate())) age--
			return age
		},
		getPreviewUrl(path) {
			if (!path) return ''
			if (path.startsWith('http://') || path.startsWith('https://')) return path
			const pathParts = path.split('/').map((part) => encodeURIComponent(part))
			return `http://localhost:8080${pathParts.join('/')}`
		},
		getPerformerPreviews(performer) {
			if (this.performerPreviews[performer.id]) {
				return this.performerPreviews[performer.id].map((path) => this.getPreviewUrl(path))
			}
			if (performer.preview_path) {
				return [this.getPreviewUrl(performer.preview_path)]
			}
			return []
		},
		async loadPerformerPreviews(performerId) {
			if (Object.prototype.hasOwnProperty.call(this.performerPreviews, performerId) && this.performerPreviews[performerId]) {
				return
			}
			try {
				const response = await performersAPI.getPreviews(performerId)
				if (response.data && response.data.previews) {
					const previewPaths = response.data.previews.map((path) => {
						if (path.startsWith('http://') || path.startsWith('https://')) {
							return new URL(path).pathname
						}
						return path
					})
					this.performerPreviews[performerId] = previewPaths
					this.savePreviewCache()
				}
			} catch (err) {
				console.error('Failed to load performer previews:', err)
			}
		},
		savePreviewCache() {
			try {
				localStorage.setItem('performerPreviews', JSON.stringify(this.performerPreviews))
			} catch (e) {
				console.error('Failed to save preview cache:', e)
			}
		},
		handleDocumentClick() {
			if (this.carouselContextMenu.visible) {
				this.closeCarouselContextMenu()
			}
		},
		openCarouselContextMenu(event, previewIndex) {
			this.carouselContextMenu.visible = true
			this.carouselContextMenu.x = event.clientX
			this.carouselContextMenu.y = event.clientY
			this.carouselContextMenu.previewIndex = previewIndex
		},
		closeCarouselContextMenu() {
			this.carouselContextMenu.visible = false
		},
		async setAsPrimaryPreview() {
			const performer = this.performer
			const previews = await performersAPI.getPreviews(performer.id)
			const selectedPreviewPath = previews.data.previews[this.carouselContextMenu.previewIndex]
			this.closeCarouselContextMenu()
			try {
				await performersAPI.update(performer.id, { preview_path: selectedPreviewPath })
				this.$emit('performer-updated', performer.id)
				this.$toast.success('Primary Set', 'Primary preview updated successfully')
			} catch (err) {
				console.error('Failed to set primary preview:', err)
				this.$toast.error('Update Failed', 'Failed to set primary preview')
			}
		},
		async loadPerformerTags(performerId) {
			try {
				const response = await performersAPI.getTags(performerId)
				this.performerMasterTags = response.data || []
			} catch (error) {
				console.error('Failed to load performer tags:', error)
				this.performerMasterTags = []
			}
		},
		async loadPerformerVideos(performerId) {
			this.loadingVideos = true
			try {
				const response = await performersAPI.getVideos(performerId)
				this.performerVideos = response.data || []
			} catch (error) {
				console.error('Failed to load performer videos:', error)
				this.performerVideos = []
				this.$toast.error('Error', 'Failed to load performer videos')
			} finally {
				this.loadingVideos = false
			}
		},
		openVideo(video) {
			this.$router.push({ name: 'VideoPlayer', params: { id: video.id } })
		},
		async addTagToPerformer() {
			if (!this.selectedTagId || !this.performer) return
			const tagId = parseInt(this.selectedTagId)
			const performerId = this.performer.id
			try {
				await performersAPI.addTag(performerId, tagId)
				this.$toast.success('Tag Added', 'Master tag added to performer')
				await this.loadPerformerTags(performerId)
				this.selectedTagId = null
			} catch (error) {
				console.error('Failed to add tag:', error)
				this.$toast.error('Error', 'Failed to add master tag')
			}
		},
		async removeTagFromPerformer(tagId) {
			if (!this.performer) return
			if (!confirm('Remove this master tag from the performer?')) return
			try {
				await performersAPI.removeTag(this.performer.id, tagId)
				this.$toast.success('Tag Removed', 'Master tag removed from performer')
				await this.loadPerformerTags(this.performer.id)
			} catch (error) {
				console.error('Failed to remove tag:', error)
				this.$toast.error('Error', 'Failed to remove master tag')
			}
		},
		async syncPerformerTags() {
			if (!this.performer) return
			const videoCount = this.performer.video_count || 0
			if (!confirm(`Apply master tags to all ${videoCount} videos featuring this performer?`)) return
			this.isSyncingTags = true
			try {
				const response = await performersAPI.syncTags(this.performer.id)
				const videosUpdated = response.data?.videos_updated || 0
				this.$toast.success('Sync Complete', `Applied master tags to ${videosUpdated} videos`)
			} catch (error) {
				console.error('Failed to sync tags:', error)
				this.$toast.error('Error', 'Failed to sync master tags to videos')
			} finally {
				this.isSyncingTags = false
			}
		},
		getAppearanceData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.appearance || Object.keys(adlData.appearance).length === 0) return null
			return { ...adlData.appearance }
		},
		getSocialMediaData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.social_media || Object.keys(adlData.social_media).length === 0) return null
			return { ...adlData.social_media }
		},
		getPlatformData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData) return null
			const platformData = {}
			if (adlData.platform_views) Object.entries(adlData.platform_views).forEach(([k, v]) => (platformData[`${k}_views`] = v))
			if (adlData.platform_video_counts) Object.entries(adlData.platform_video_counts).forEach(([k, v]) => (platformData[`${k}_videos`] = v))
			if (adlData.platform_profile_counts) Object.entries(adlData.platform_profile_counts).forEach(([k, v]) => (platformData[`${k}_profiles`] = v))
			return Object.keys(platformData).length > 0 ? platformData : null
		},
		getBiosData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.bios || Object.keys(adlData.bios).length === 0) return null
			return { ...adlData.bios }
		},
		getExternalLinksData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.external_links) return []
			return [...adlData.external_links]
		},
		formatDate(dateString) {
			if (!dateString) return ''
			return new Date(dateString).toLocaleDateString()
		},
		formatLabel(key) {
			return key
				.split('_')
				.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
				.join(' ')
		},
		formatValue(value) {
			if (value === null || value === undefined) return 'N/A'
			if (typeof value === 'boolean') return value ? 'Yes' : 'No'
			if (typeof value === 'object') return JSON.stringify(value)
			return String(value)
		},
		isURL(value) {
			if (typeof value !== 'string') return false
			return value.startsWith('http://') || value.startsWith('https://')
		},
		getLinkURL(link) {
			if (typeof link === 'string') return link
			return link.url || link.link || '#'
		},
		getLinkTitle(link) {
			if (typeof link === 'string') return link
			return link.title || link.name || link.platform || link.url || 'External Link'
		},
	},
}
</script>
