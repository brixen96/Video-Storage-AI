<template>
	<div class="performers-page">
		<div class="container-fluid py-2">
			<!-- Controls Bar -->
			<div class="controls-bar mb-2">
				<div class="row g-3">
					<!-- Search -->
					<div class="col-md-3">
						<SearchBox v-model="searchQuery" placeholder="Search performers..." />
					</div>

					<!-- Sort -->
					<div class="col-md-1">
						<select v-model="sortBy" class="form-select">
							<option value="name">Sort by Name</option>
							<option value="age">Sort by Age</option>
							<option value="breast">Sort by Breast Size</option>
							<option value="height">Sort by Height</option>
							<option value="scenes">Sort by Video Count</option>
						</select>
					</div>

					<!-- Category Quick Filter -->
					<div class="col-md-3">
						<div class="btn-group w-100" role="group">
							<button
								:class="['btn', 'btn-sm', filters.categoryFilter === 'all' ? 'btn-primary' : 'btn-outline-primary']"
								@click="filters.categoryFilter = 'all'"
								title="Show All Performers"
							>
								All
							</button>
							<button
								:class="['btn', 'btn-sm', filters.categoryFilter === 'regular' ? 'btn-info' : 'btn-outline-info']"
								@click="filters.categoryFilter = 'regular'"
								title="Regular Only"
							>
								<font-awesome-icon :icon="['fas', 'user']" />
							</button>
							<button
								:class="['btn', 'btn-sm', filters.categoryFilter === 'zoo' ? 'btn-success' : 'btn-outline-success']"
								@click="filters.categoryFilter = 'zoo'"
								title="Zoo Only"
							>
								<font-awesome-icon :icon="['fas', 'dog']" />
							</button>
							<button
								:class="['btn', 'btn-sm', filters.categoryFilter === '3d' ? 'btn-warning' : 'btn-outline-warning']"
								@click="filters.categoryFilter = '3d'"
								title="3D Only"
							>
								<font-awesome-icon :icon="['fas', 'cube']" />
							</button>
						</div>
					</div>

					<!-- View Toggle -->
					<div class="col-md-1">
						<div class="btn-group w-100" role="group">
							<button :class="['btn', 'btn-outline-primary', { active: viewMode === 'grid' }]" @click="viewMode = 'grid'">
								<font-awesome-icon :icon="['fas', 'th']" />
							</button>
							<button :class="['btn', 'btn-outline-primary', { active: viewMode === 'list' }]" @click="viewMode = 'list'">
								<font-awesome-icon :icon="['fas', 'list']" />
							</button>
						</div>
					</div>

					<!-- Advanced Filter Toggle -->
					<div class="col-md-1">
						<button class="btn btn-outline-secondary w-100" @click="showFilters = !showFilters">
							<font-awesome-icon :icon="['fas', 'filter']" class="me-2" />
							{{ showFilters ? 'Hide' : 'Filters' }}
						</button>
					</div>

					<!-- Cleanup Button -->
					<div class="col-md-1">
						<button class="btn btn-outline-danger w-100" @click="cleanupEmptyPerformers" :disabled="isCleaningUp" title="Delete empty performers">
							<font-awesome-icon :icon="['fas', isCleaningUp ? 'spinner' : 'trash-alt']" :spin="isCleaningUp" />
						</button>
					</div>
				</div>

				<!-- Advanced Filters -->
				<div v-if="showFilters" class="filters-panel mt-3">
					<div class="row g-3">
						<!-- Category Filter -->
						<div class="col-md-3">
							<div class="filter-group">
								<label class="form-label">
									<font-awesome-icon :icon="['fas', 'list']" class="me-2" />
									Category Filter
								</label>
								<select v-model="filters.categoryFilter" class="form-select form-select-sm">
									<option value="all">Show All</option>
									<option value="regular">Regular Only</option>
									<option value="zoo">Zoo Only</option>
									<option value="3d">3D Only</option>
								</select>
							</div>
						</div>

						<!-- Age Range -->
						<div class="col-md-3">
							<div class="filter-group">
								<label class="form-label">Age Range</label>
								<div class="range-inputs">
									<input v-model.number="filters.ageMin" type="number" class="form-control form-control-sm" placeholder="Min" min="18" />
									<span class="range-separator">-</span>
									<input v-model.number="filters.ageMax" type="number" class="form-control form-control-sm" placeholder="Max" />
								</div>
							</div>
						</div>

						<!-- Breast Size Range -->
						<div class="col-md-3">
							<div class="filter-group">
								<label class="form-label">Breast Size</label>
								<div class="range-inputs">
									<input v-model="filters.breastMin" type="text" class="form-control form-control-sm" placeholder="Min (e.g., A)" maxlength="2" />
									<span class="range-separator">-</span>
									<input v-model="filters.breastMax" type="text" class="form-control form-control-sm" placeholder="Max (e.g., DD)" maxlength="3" />
								</div>
							</div>
						</div>

						<!-- Height Range -->
						<div class="col-md-3">
							<div class="filter-group">
								<label class="form-label">Height (cm)</label>
								<div class="range-inputs">
									<input v-model.number="filters.heightMin" type="number" class="form-control form-control-sm" placeholder="Min" />
									<span class="range-separator">-</span>
									<input v-model.number="filters.heightMax" type="number" class="form-control form-control-sm" placeholder="Max" />
								</div>
							</div>
						</div>
					</div>

					<!-- Clear Filters -->
					<div class="row mt-2">
						<div class="col-12">
							<button class="btn btn-sm btn-outline-danger" @click="clearFilters">
								<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
								Clear Filters
							</button>
						</div>
					</div>
				</div>
			</div>

			<!-- Results / Pagination Bar -->
			<div v-if="!loading && !error && filteredPerformers.length > 0" class="results-bar d-flex justify-content-between align-items-center mb-2">
				<span class="text-muted small">
					Showing {{ (currentPage - 1) * pageSize + 1 }}–{{ Math.min(currentPage * pageSize, filteredPerformers.length) }}
					of {{ filteredPerformers.length }} performers
				</span>
				<div v-if="totalPages > 1" class="d-flex align-items-center gap-2">
					<button class="btn btn-sm btn-outline-secondary" :disabled="currentPage === 1" @click="goToPage(currentPage - 1)">
						<font-awesome-icon :icon="['fas', 'chevron-left']" />
					</button>
					<span class="text-muted small">Page {{ currentPage }} / {{ totalPages }}</span>
					<button class="btn btn-sm btn-outline-secondary" :disabled="currentPage === totalPages" @click="goToPage(currentPage + 1)">
						<font-awesome-icon :icon="['fas', 'chevron-right']" />
					</button>
				</div>
			</div>

			<!-- Loading State -->
			<div v-if="loading" class="loading-state">
				<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" />
				<p class="mt-3">Loading performers...</p>
			</div>

			<!-- Error State -->
			<div v-else-if="error" class="error-state">
				<font-awesome-icon :icon="['fas', 'exclamation-triangle']" size="3x" />
				<p class="mt-3">{{ error }}</p>
				<button class="btn btn-primary mt-2" @click="loadPerformers">
					<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
					Retry
				</button>
			</div>

			<!-- Empty State -->
			<div v-else-if="filteredPerformers.length === 0" class="empty-state">
				<font-awesome-icon :icon="['fas', 'users']" size="3x" />
				<p class="mt-3">
					{{ performers.length === 0 ? 'No performers found.' : 'No performers match your filters.' }}
				</p>
			</div>

			<!-- Performers Grid View -->
			<div v-else-if="viewMode === 'grid'" class="performers-grid">
				<div
					v-for="performer in pagedPerformers"
					:key="performer.id"
					class="performer-card text-center"
					@click="openDetails(performer)"
					@contextmenu.prevent="openContextMenu($event, performer)"
				>
					<!-- Video Preview with Two-Tier Loading -->
					<div class="card-preview" @mouseenter="startVideoPreview(performer)" @mouseleave="stopVideoPreview(performer)">
						<!-- Static Thumbnail (shown when not hovered) -->
						<img
							v-if="performer.thumbnail_path"
							v-show="!hoveredPerformer[performer.id]"
							:src="getThumbnailUrl(performer.thumbnail_path)"
							class="preview-thumbnail"
							:alt="performer.name"
							loading="lazy"
						/>
						<!-- Video Preview (only created on hover to avoid hundreds of video elements in DOM) -->
						<video
							v-if="performer.preview_path && hoveredPerformer[performer.id]"
							class="preview-video"
							:src="getPreviewUrl(performer.preview_path)"
							autoplay
							loop
							muted
							playsinline
							@error="handleVideoError"
						></video>
						<!-- Fallback for No Preview -->
						<div v-if="!performer.preview_path && !performer.thumbnail_path && !hoveredPerformer[performer.id]" class="no-preview">
							<font-awesome-icon :icon="['fas', 'user']" size="3x" />
						</div>

						<!-- Video Count Badge -->
						<div class="scene-badge">
							<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
							{{ performer.video_count || 0 }}
						</div>

						<!-- Category Badge -->
						<div
							v-if="performer.category && performer.category !== 'regular'"
							:class="['category-badge', 'category-' + performer.category]"
							:title="getCategoryName(performer.category)"
						>
							<font-awesome-icon :icon="['fas', getCategoryIcon(performer.category)]" />
						</div>
					</div>

					<!-- Card Info -->
					<div class="card-info">
						<span class="performer-name">{{ performer.name }}</span>
						<div class="d-flex flex-row justify-content-between">
							<div class="performer-meta d-flex flex-row">
								<span v-if="getAge(performer)" class="meta-item">
									<font-awesome-icon :icon="['fas', 'birthday-cake']" class="me-1" />
									{{ getAge(performer) }}
								</span>
								<span v-if="performer.metadata?.measurements" class="meta-item">
									{{ performer.metadata.measurements }}
								</span>
								<span v-if="performer.metadata?.height" class="meta-item"> {{ performer.metadata.height }}</span>
							</div>
							<!-- Tags -->
							<div v-if="performerTags[performer.id] && performerTags[performer.id].length > 0" class="performer-tags d-flex justify-content-end">
								<span v-for="tag in performerTags[performer.id].slice(0, 5)" :key="tag.id" class="tag-chip" :style="{ backgroundColor: tag.color }">
									{{ tag.name }}
								</span>
								<span v-if="performerTags[performer.id].length > 5" class="tag-more">+{{ performerTags[performer.id].length - 5 }}</span>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Performers List View -->
			<div v-else class="performers-list">
				<div
					v-for="performer in pagedPerformers"
					:key="performer.id"
					class="list-item"
					@click="openDetails(performer)"
					@contextmenu.prevent="openContextMenu($event, performer)"
				>
					<div class="list-preview" @mouseenter="startVideoPreview(performer)" @mouseleave="stopVideoPreview(performer)">
						<!-- Static Thumbnail (shown when not hovered) -->
						<img
							v-if="performer.thumbnail_path"
							v-show="!hoveredPerformer[performer.id]"
							:src="getThumbnailUrl(performer.thumbnail_path)"
							class="preview-thumbnail-small"
							:alt="performer.name"
							loading="lazy"
						/>
						<!-- Video Preview (only created on hover) -->
						<video
							v-if="performer.preview_path && hoveredPerformer[performer.id]"
							class="preview-video-small"
							:src="getPreviewUrl(performer.preview_path)"
							autoplay
							loop
							muted
							playsinline
							@error="handleVideoError"
						></video>
						<!-- Fallback for No Preview -->
						<div v-if="!performer.preview_path && !performer.thumbnail_path && !hoveredPerformer[performer.id]" class="no-preview-small">
							<font-awesome-icon :icon="['fas', 'user']" />
						</div>
					</div>
					<div class="list-content">
						<h5 class="performer-name">
							{{ performer.name }}
							<font-awesome-icon
								v-if="performer.category && performer.category !== 'regular'"
								:icon="['fas', getCategoryIcon(performer.category)]"
								:class="['category-icon', 'ms-2', 'category-' + performer.category]"
								:title="getCategoryName(performer.category)"
							/>
						</h5>
						<div class="performer-details">
							<span v-if="getAge(performer)" class="detail-item">Age: {{ getAge(performer) }}</span>
							<span v-if="performer.metadata?.measurements" class="detail-item">Measurements: {{ performer.metadata.measurements }}</span>
							<span v-if="performer.metadata?.height" class="detail-item">Height: {{ performer.metadata.height }}</span>
							<span v-if="performer.metadata?.weight" class="detail-item">Weight: {{ performer.metadata.weight }}</span>
							<span class="detail-item">Videos: {{ performer.video_count || 0 }}</span>
						</div>
					</div>
					<div class="list-actions">
						<button class="btn btn-sm btn-outline-primary" @click.stop="openDetails(performer)">
							<font-awesome-icon :icon="['fas', 'eye']" />
						</button>
					</div>
				</div>
			</div>

			<!-- Bottom Pagination -->
			<div v-if="!loading && !error && totalPages > 1" class="d-flex justify-content-center mt-3 mb-2">
				<div class="d-flex align-items-center gap-2">
					<button class="btn btn-sm btn-outline-secondary" :disabled="currentPage === 1" @click="goToPage(1)">
						<font-awesome-icon :icon="['fas', 'angle-double-left']" />
					</button>
					<button class="btn btn-sm btn-outline-secondary" :disabled="currentPage === 1" @click="goToPage(currentPage - 1)">
						<font-awesome-icon :icon="['fas', 'chevron-left']" />
					</button>
					<span class="px-2 text-muted">Page {{ currentPage }} / {{ totalPages }}</span>
					<button class="btn btn-sm btn-outline-secondary" :disabled="currentPage === totalPages" @click="goToPage(currentPage + 1)">
						<font-awesome-icon :icon="['fas', 'chevron-right']" />
					</button>
					<button class="btn btn-sm btn-outline-secondary" :disabled="currentPage === totalPages" @click="goToPage(totalPages)">
						<font-awesome-icon :icon="['fas', 'angle-double-right']" />
					</button>
				</div>
			</div>
		</div>

		<!-- Context Menu -->
		<PerformerContextMenu
			:visible="contextMenu.visible"
			:x="contextMenu.x"
			:y="contextMenu.y"
			:performer="contextMenu.performer"
			@set-category="updateCategory(contextMenu.performer, $event)"
			@go-to-details="goToPerformerDetails(contextMenu.performer)"
			@fetch-metadata="fetchMetadata(contextMenu.performer)"
			@reset-performer="resetPerformer(contextMenu.performer)"
			@confirm-delete="confirmDelete(contextMenu.performer)"
			@close="closeContextMenu"
		/>

		<!-- Details Panel -->
		<PerformerDetailsPanel
			v-if="detailsPanel.visible"
			:performer="detailsPanel.performer"
			:all-tags="allTags"
			@close="closeDetails"
			@fetch-metadata="fetchMetadata"
			@reset="resetPerformer"
			@confirm-delete="confirmDelete"
			@performer-updated="onPerformerUpdated"
		/>

		<!-- Delete Confirmation Modal -->
		<DeleteConfirmationModal
			:visible="deleteModal.visible"
			title="Confirm Delete"
			message="Are you sure you want to delete"
			:itemName="deleteModal.performer?.name"
			@confirm="deletePerformer"
			@cancel="deleteModal.visible = false"
		/>
	</div>
</template>

<script>
import { performersAPI } from '@/services/api'
import { tagsAPI } from '@/services/api'
import { API_ORIGIN } from '@/services/api'
import settingsService from '@/services/settingsService'
import { DeleteConfirmationModal, SearchBox } from '@/components/shared'
import { PerformerContextMenu } from '@/components/performers'
import PerformerDetailsPanel from '@/components/PerformerDetailsPanel.vue'

export default {
	name: 'PerformersPage',
	components: {
		DeleteConfirmationModal,
		PerformerContextMenu,
		PerformerDetailsPanel,
		SearchBox,
	},
	data() {
		const settings = settingsService.getSettings()
		return {
			performers: [],
			loading: false,
			error: null,
			isCleaningUp: false,
			searchQuery: '',
			sortBy: 'name',
			viewMode: settings.defaultViewMode || 'grid',
			showFilters: false,
			filters: {
				categoryFilter: settings.defaultCategoryFilter || 'all',
				ageMin: null,
				ageMax: null,
				breastMin: '',
				breastMax: '',
				heightMin: null,
				heightMax: null,
			},
			contextMenu: {
				visible: false,
				x: 0,
				y: 0,
				performer: null,
			},
			detailsPanel: {
				visible: false,
				performer: null,
			},
			deleteModal: {
				visible: false,
				performer: null,
			},
			allTags: [],
			performerTags: {},
			hoveredPerformer: {},
			currentPage: 1,
			pageSize: 48,
		}
	},
	computed: {
		filteredPerformers() {
			if (!Array.isArray(this.performers)) return []

			let result = this.performers

			if (this.searchQuery) {
				const query = this.searchQuery.toLowerCase()
				result = result.filter((p) => p && p.name && p.name.toLowerCase().includes(query))
			}

			if (this.filters.categoryFilter !== 'all') {
				result = result.filter((p) => p && p.category === this.filters.categoryFilter)
			}

			if (this.filters.ageMin) {
				result = result.filter((p) => {
					const age = this.getAge(p)
					return age && age >= this.filters.ageMin
				})
			}
			if (this.filters.ageMax) {
				result = result.filter((p) => {
					const age = this.getAge(p)
					return age && age <= this.filters.ageMax
				})
			}

			if (this.filters.heightMin) {
				result = result.filter((p) => {
					const height = this.parseHeight(p.metadata?.height)
					return height && height >= this.filters.heightMin
				})
			}
			if (this.filters.heightMax) {
				result = result.filter((p) => {
					const height = this.parseHeight(p.metadata?.height)
					return height && height <= this.filters.heightMax
				})
			}

			if (this.filters.breastMin) {
				result = result.filter((p) => p && p.breast_size && p.breast_size >= this.filters.breastMin)
			}
			if (this.filters.breastMax) {
				result = result.filter((p) => p && p.breast_size && p.breast_size <= this.filters.breastMax)
			}

			return [...result].sort((a, b) => {
				if (!a || !b) return 0
				switch (this.sortBy) {
					case 'name':
						return (a.name || '').localeCompare(b.name || '')
					case 'age':
						return (this.getAge(a) || 0) - (this.getAge(b) || 0)
					case 'breast':
						return (a.metadata?.measurements || '').localeCompare(b.metadata?.measurements || '')
					case 'height':
						return (this.parseHeight(a.metadata?.height) || 0) - (this.parseHeight(b.metadata?.height) || 0)
					case 'scenes':
						return (b.video_count || 0) - (a.video_count || 0)
					default:
						return 0
				}
			})
		},
		pagedPerformers() {
			const start = (this.currentPage - 1) * this.pageSize
			return this.filteredPerformers.slice(start, start + this.pageSize)
		},
		totalPages() {
			return Math.max(1, Math.ceil(this.filteredPerformers.length / this.pageSize))
		},
	},
	watch: {
		searchQuery() {
			this.currentPage = 1
		},
		sortBy() {
			this.currentPage = 1
		},
		filters: {
			deep: true,
			handler() {
				this.currentPage = 1
			},
		},
	},
	methods: {
		getCategoryIcon(category) {
			const icons = { regular: 'user', zoo: 'dog', '3d': 'cube' }
			return icons[category] || 'user'
		},
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
		parseHeight(heightStr) {
			if (!heightStr) return null
			const match = heightStr.match(/(\d+)/)
			return match ? parseInt(match[1]) : null
		},
		getThumbnailUrl(thumbnailPath) {
			if (!thumbnailPath) return null
			if (thumbnailPath.startsWith('http://') || thumbnailPath.startsWith('https://')) return thumbnailPath
			if (thumbnailPath.startsWith('/assets/')) return `${API_ORIGIN}${thumbnailPath}`
			return `${API_ORIGIN}/assets/${thumbnailPath}`
		},
		getPreviewUrl(path) {
			if (!path) return ''
			if (path.startsWith('http://') || path.startsWith('https://')) return path
			const pathParts = path.split('/').map((part) => encodeURIComponent(part))
			return `${API_ORIGIN}${pathParts.join('/')}`
		},
		startVideoPreview(performer) {
			this.hoveredPerformer[performer.id] = true
		},
		stopVideoPreview(performer) {
			this.hoveredPerformer[performer.id] = false
		},
		handleVideoError(event) {
			const video = event.target
			console.error('Video failed to load:', { src: video.src, error: video.error })
		},
		async loadPerformers() {
			this.loading = true
			this.error = null
			try {
				const response = await performersAPI.getAll()
				if (response.data) {
					if (Array.isArray(response.data)) {
						this.performers = response.data
					} else if (response.data.performers && Array.isArray(response.data.performers)) {
						this.performers = response.data.performers
					} else if (response.data.data && Array.isArray(response.data.data)) {
						this.performers = response.data.data
					} else {
						this.performers = []
					}
				} else {
					this.performers = []
				}
				this.loadTagsForPage(this.pagedPerformers)
			} catch (err) {
				console.error('Failed to load performers:', err)
				this.error = 'Failed to load performers. Please try again.'
				this.performers = []
			} finally {
				this.loading = false
			}
		},
		async loadTagsForPage(performers) {
			const toLoad = performers.filter((p) => !Object.prototype.hasOwnProperty.call(this.performerTags, p.id))
			if (toLoad.length === 0) return
			await Promise.all(
				toLoad.map(async (performer) => {
					try {
						const response = await performersAPI.getTags(performer.id)
						this.performerTags[performer.id] = response.data || []
					} catch {
						this.performerTags[performer.id] = []
					}
				})
			)
		},
		goToPage(page) {
			if (page < 1 || page > this.totalPages) return
			this.currentPage = page
			this.loadTagsForPage(this.pagedPerformers)
			window.scrollTo({ top: 0, behavior: 'smooth' })
		},
		openDetails(performer) {
			this.detailsPanel.visible = true
			this.detailsPanel.performer = performer
		},
		closeDetails() {
			this.detailsPanel.visible = false
			this.detailsPanel.performer = null
		},
		async onPerformerUpdated() {
			await this.loadPerformers()
			const updated = this.performers.find((p) => p.id === this.detailsPanel.performer?.id)
			if (updated) {
				this.detailsPanel.performer = updated
			}
		},
		openContextMenu(event, performer) {
			this.contextMenu.visible = true
			this.contextMenu.x = event.clientX
			this.contextMenu.y = event.clientY
			this.contextMenu.performer = performer
		},
		closeContextMenu() {
			this.contextMenu.visible = false
			this.contextMenu.performer = null
		},
		handleDocumentClick() {
			if (this.contextMenu.visible) {
				this.closeContextMenu()
			}
		},
		goToPerformerDetails(performer) {
			this.closeContextMenu()
			this.$router.push({ name: 'PerformerDetails', params: { id: performer.id } })
		},
		async fetchMetadata(performer) {
			this.closeContextMenu()
			try {
				await performersAPI.fetchMetadata(performer.id)
				await this.loadPerformers()
				if (this.detailsPanel.visible && this.detailsPanel.performer?.id === performer.id) {
					const updatedPerformer = this.performers.find((p) => p.id === performer.id)
					if (updatedPerformer) this.detailsPanel.performer = updatedPerformer
				}
				this.$toast.success('Metadata Fetched', `Successfully fetched metadata for ${performer.name}`)
			} catch (err) {
				console.error('Failed to fetch metadata:', err)
				this.$toast.error('Fetch Failed', err.response?.data?.error || 'Failed to fetch metadata from AdultDataLink')
			}
		},
		async updateCategory(performer, newCategory) {
			this.closeContextMenu()
			try {
				const response = await performersAPI.update(performer.id, { category: newCategory })
				if (response && response.category !== undefined) {
					performer.category = response.category
				} else {
					performer.category = newCategory
				}
				this.$forceUpdate()
				this.$toast.success('Updated', `${performer.name} set to ${this.getCategoryName(newCategory)}`)
			} catch (err) {
				console.error('Failed to update category:', err)
				this.$toast.error('Update Failed', 'Failed to update category')
			}
		},
		async resetPerformer(performer) {
			this.closeContextMenu()
			try {
				await performersAPI.resetMetadata(performer.id)
				await this.loadPerformers()
				this.$toast.success('Metadata Reset', `Metadata for ${performer.name} has been reset`)
			} catch (err) {
				console.error('Failed to reset performer:', err)
				this.$toast.error('Reset Failed', 'Failed to reset metadata')
			}
		},
		confirmDelete(performer) {
			this.closeContextMenu()
			this.closeDetails()
			this.deleteModal.visible = true
			this.deleteModal.performer = performer
		},
		async deletePerformer() {
			const performer = this.deleteModal.performer
			this.deleteModal.visible = false
			try {
				await performersAPI.delete(performer.id)
				this.performers = this.performers.filter((p) => p.id !== performer.id)
				this.$toast.success('Deleted', `${performer.name} has been deleted`)
			} catch (err) {
				console.error('Failed to delete performer:', err)
				this.$toast.error('Delete Failed', 'Failed to delete performer')
			}
		},
		async cleanupEmptyPerformers() {
			if (!confirm('This will delete all performers without preview videos or metadata. Are you sure?')) return
			this.isCleaningUp = true
			try {
				const response = await fetch(`${API_ORIGIN}/api/v1/performers/cleanup`, { method: 'POST' })
				const data = await response.json()
				if (data.success) {
					this.$toast.success(`Cleaned up ${data.data.deleted_count} empty performers`)
					await this.loadPerformers()
				} else {
					this.$toast.error('Cleanup Failed', data.message || 'Failed to cleanup performers')
				}
			} catch (err) {
				console.error('Failed to cleanup performers:', err)
				this.$toast.error('Cleanup Failed', 'Failed to cleanup performers')
			} finally {
				this.isCleaningUp = false
			}
		},
		clearFilters() {
			this.filters = {
				categoryFilter: 'all',
				ageMin: null,
				ageMax: null,
				breastMin: '',
				breastMax: '',
				heightMin: null,
				heightMax: null,
			}
		},
		async loadAllTags() {
			let storeTags = await this.$store.dispatch('fetchTags')
			if (storeTags.length !== 0) {
				this.allTags = storeTags
			} else {
				const response = await tagsAPI.getAll()
				this.allTags = response || []
			}
		},
	},
	mounted() {
		this.loadPerformers()
		this.loadAllTags()
		document.addEventListener('click', this.handleDocumentClick)
	},
	activated() {
		let unwatch = null
		unwatch = this.$watch(
			'performers',
			(newVal) => {
				if (newVal && newVal.length > 0) {
					this.$nextTick(() => {
						const videos = this.$el.querySelectorAll('video')
						videos.forEach((video) => {
							if (video.src) {
								const currentSrc = video.src
								video.src = ''
								video.load()
								video.src = currentSrc
								video.load()
								video.style.opacity = '1'
								video.style.position = 'static'
							}
						})
					})
					if (unwatch) unwatch()
				}
			},
			{ immediate: true }
		)
	},
	beforeUnmount() {
		document.removeEventListener('click', this.handleDocumentClick)
	},
}
</script>

<style scoped>
@import '@/styles/pages/performers_page.css';
</style>
