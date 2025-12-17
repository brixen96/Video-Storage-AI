<template>
	<div class="performers-page">
		<div class="container-fluid py-2">
			<!-- Controls Bar -->
			<div class="controls-bar mb-2">
				<div class="row g-3">
					<!-- Search -->
					<div class="col-md-4">
						<div class="search-box">
							<font-awesome-icon :icon="['fas', 'search']" class="search-icon" />
							<input v-model="searchQuery" type="text" class="form-control" placeholder="Search performers..." />
							<button v-if="searchQuery" class="btn-clear-search" @click="searchQuery = ''">
								<font-awesome-icon :icon="['fas', 'times-circle']" />
							</button>
						</div>
					</div>

					<!-- Sort -->
					<div class="col-md-3">
						<select v-model="sortBy" class="form-select">
							<option value="name">Sort by Name</option>
							<option value="age">Sort by Age</option>
							<option value="breast">Sort by Breast Size</option>
							<option value="height">Sort by Height</option>
							<option value="scenes">Sort by Scene Count</option>
						</select>
					</div>

					<!-- Zoo Quick Filter -->
					<div class="col-md-2">
						<div class="btn-group w-100" role="group">
							<button
								:class="['btn', 'btn-sm', filters.zooFilter === 'all' ? 'btn-primary' : 'btn-outline-primary']"
								@click="filters.zooFilter = 'all'"
								title="Show All Performers"
							>
								All
							</button>
							<button
								:class="['btn', 'btn-sm', filters.zooFilter === 'zoo-only' ? 'btn-danger' : 'btn-outline-danger']"
								@click="filters.zooFilter = 'zoo-only'"
								title="Show Zoo Only"
							>
								<font-awesome-icon :icon="['fas', 'dog']" />
							</button>
							<button
								:class="['btn', 'btn-sm', filters.zooFilter === 'non-zoo' ? 'btn-success' : 'btn-outline-success']"
								@click="filters.zooFilter = 'non-zoo'"
								title="Hide Zoo"
							>
								<font-awesome-icon :icon="['fas', 'ban']" />
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
					<div class="col-md-2">
						<button class="btn btn-outline-secondary w-100" @click="showFilters = !showFilters">
							<font-awesome-icon :icon="['fas', 'filter']" class="me-2" />
							{{ showFilters ? 'Hide' : 'Filters' }}
						</button>
					</div>
				</div>

				<!-- Advanced Filters -->
				<div v-if="showFilters" class="filters-panel mt-3">
					<div class="row g-3">
						<!-- Zoo Filter -->
						<div class="col-md-3">
							<div class="filter-group">
								<label class="form-label">
									<font-awesome-icon :icon="['fas', 'dog']" class="me-2" />
									Zoo Filter
								</label>
								<select v-model="filters.zooFilter" class="form-select form-select-sm">
									<option value="all">Show All</option>
									<option value="zoo-only">Zoo Only</option>
									<option value="non-zoo">Non-Zoo Only</option>
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
					v-for="performer in filteredPerformers"
					:key="performer.id"
					class="performer-card text-center"
					@click="openDetails(performer)"
					@contextmenu.prevent="openContextMenu($event, performer)"
				>
					<!-- Video Preview -->
					<div class="card-preview">
						<video
							v-if="performer.preview_path"
							:key="`preview-${performer.id}-${videoRefreshKey}`"
							class="preview-video"
							:src="getPreviewUrl(performer.preview_path)"
							loop
							muted
							playsinline
							preload="metadata"
							@mouseenter="playPreview"
							@mouseleave="pausePreview"
							@error="handleVideoError"
						></video>
						<div v-else class="no-preview">
							<font-awesome-icon :icon="['fas', 'user']" size="3x" />
						</div>

						<!-- Scene Count Badge -->
						<div class="scene-badge">
							<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
							{{ performer.scene_count || 0 }}
						</div>

						<!-- Zoo Badge -->
						<div v-if="performer.zoo" class="zoo-badge" title="Zoo Content">
							<font-awesome-icon :icon="['fas', 'dog']" />
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
					v-for="performer in filteredPerformers"
					:key="performer.id"
					class="list-item"
					@click="openDetails(performer)"
					@contextmenu.prevent="openContextMenu($event, performer)"
				>
					<div class="list-preview">
						<video
							v-if="performer.preview_path"
							:key="`preview-${performer.id}-${videoRefreshKey}`"
							class="preview-video-small"
							:src="getPreviewUrl(performer.preview_path)"
							loop
							muted
							playsinline
							preload="metadata"
							@loadedmetadata="onVideoLoaded"
							@error="handleVideoError"
						></video>
						<div v-else class="no-preview-small">
							<font-awesome-icon :icon="['fas', 'user']" />
						</div>
					</div>
					<div class="list-content">
						<h5 class="performer-name">
							{{ performer.name }}
							<font-awesome-icon v-if="performer.zoo" :icon="['fas', 'dog']" class="zoo-icon ms-2" title="Zoo Content" />
						</h5>
						<div class="performer-details">
							<span v-if="getAge(performer)" class="detail-item">Age: {{ getAge(performer) }}</span>
							<span v-if="performer.metadata?.measurements" class="detail-item">Measurements: {{ performer.metadata.measurements }}</span>
							<span v-if="performer.metadata?.height" class="detail-item">Height: {{ performer.metadata.height }}</span>
							<span v-if="performer.metadata?.weight" class="detail-item">Weight: {{ performer.metadata.weight }}</span>
							<span class="detail-item">Scenes: {{ performer.scene_count || 0 }}</span>
						</div>
					</div>
					<div class="list-actions">
						<button class="btn btn-sm btn-outline-primary" @click.stop="openDetails(performer)">
							<font-awesome-icon :icon="['fas', 'eye']" />
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Context Menu -->
		<div v-if="contextMenu.visible" class="context-menu" :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }" @click="closeContextMenu">
			<button class="context-menu-item" @click="toggleZoo(contextMenu.performer)">
				<font-awesome-icon :icon="['fas', 'dog']" class="me-2" />
				{{ contextMenu.performer?.zoo ? 'Unmark Zoo' : 'Mark as Zoo' }}
			</button>
			<button class="context-menu-item" @click="fetchMetadata(contextMenu.performer)">
				<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
				Fetch Metadata
			</button>
			<button class="context-menu-item" @click="resetPerformer(contextMenu.performer)">
				<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
				Reset Performer
			</button>
			<button class="context-menu-item danger" @click="confirmDelete(contextMenu.performer)">
				<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
				Delete Performer
			</button>
		</div>

		<!-- Details Panel (Side Drawer) -->
		<div v-if="detailsPanel.visible" class="details-panel" @click.self="closeDetails">
			<div class="panel-content">
				<!-- Carousel -->
				<div class="panel-carousel">
					<div v-if="getPerformerPreviews(detailsPanel.performer).length > 0" class="carousel-container">
						<button v-if="carouselIndex > 0" class="carousel-btn prev" @click="carouselIndex--">
							<font-awesome-icon :icon="['fas', 'chevron-left']" />
						</button>
						<video
							:src="getPerformerPreviews(detailsPanel.performer)[carouselIndex]"
							class="carousel-video"
							controls
							autoplay
							loop
							muted
							@contextmenu.prevent="openCarouselContextMenu($event, carouselIndex)"
						></video>
						<button v-if="carouselIndex < getPerformerPreviews(detailsPanel.performer).length - 1" class="carousel-btn next" @click="carouselIndex++">
							<font-awesome-icon :icon="['fas', 'chevron-right']" />
						</button>
						<div class="carousel-indicator">{{ carouselIndex + 1 }} / {{ getPerformerPreviews(detailsPanel.performer).length }}</div>
					</div>
					<div v-else class="no-carousel">
						<font-awesome-icon :icon="['fas', 'user']" size="5x" />
						<p class="mt-3">No preview videos available</p>
					</div>
				</div>

				<!-- Metadata Tabs -->
				<div class="panel-metadata">
					<!-- Tab Navigation -->
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
					</ul>

					<!-- Tab Content -->
					<div class="tab-content mt-3">
						<!-- Basic Tab -->
						<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'basic' }]">
							<div class="metadata-grid">
								<div class="metadata-item">
									<span class="metadata-label">Name:</span>
									<span class="metadata-value">{{ detailsPanel.performer.name }}</span>
								</div>
								<div v-if="getAge(detailsPanel.performer)" class="metadata-item">
									<span class="metadata-label">Age:</span>
									<span class="metadata-value">{{ getAge(detailsPanel.performer) }}</span>
								</div>
								<div v-if="detailsPanel.performer.metadata?.birthdate" class="metadata-item">
									<span class="metadata-label">Birthdate:</span>
									<span class="metadata-value">{{ formatDate(detailsPanel.performer.metadata.birthdate) }}</span>
								</div>
								<div v-if="detailsPanel.performer.metadata?.birthplace" class="metadata-item">
									<span class="metadata-label">Birthplace:</span>
									<span class="metadata-value">{{ detailsPanel.performer.metadata.birthplace }}</span>
								</div>
								<div v-if="detailsPanel.performer.metadata?.career_start" class="metadata-item">
									<span class="metadata-label">Career Start:</span>
									<span class="metadata-value">{{ detailsPanel.performer.metadata.career_start }}</span>
								</div>
								<div v-if="detailsPanel.performer.metadata?.career_end" class="metadata-item">
									<span class="metadata-label">Career End:</span>
									<span class="metadata-value">{{ detailsPanel.performer.metadata.career_end }}</span>
								</div>
								<div class="metadata-item">
									<span class="metadata-label">Scene Count:</span>
									<span class="metadata-value">{{ detailsPanel.performer.scene_count || 0 }}</span>
								</div>
								<div v-if="detailsPanel.performer.metadata?.aliases?.length" class="metadata-item full-width">
									<span class="metadata-label">Aliases:</span>
									<span class="metadata-value">{{ detailsPanel.performer.metadata.aliases.join(', ') }}</span>
								</div>
								<div v-if="detailsPanel.performer.zoo" class="metadata-item full-width">
									<span class="metadata-label">Content Type:</span>
									<span class="metadata-value zoo-indicator">Zoo Content</span>
								</div>
							</div>
						</div>

						<!-- Appearance Tab -->
						<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'appearance' }]">
							<div v-if="getAppearanceData(detailsPanel.performer)" class="metadata-grid">
								<div v-for="(value, key) in getAppearanceData(detailsPanel.performer)" :key="key" class="metadata-item">
									<span class="metadata-label">{{ formatLabel(key) }}:</span>
									<span class="metadata-value">{{ value }}</span>
								</div>
							</div>
							<div v-else class="empty-tab">
								<p>No appearance data available</p>
							</div>
						</div>

						<!-- Social Media Tab -->
						<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'social_media' }]">
							<div v-if="getSocialMediaData(detailsPanel.performer)" class="metadata-grid">
								<div v-for="(value, key) in getSocialMediaData(detailsPanel.performer)" :key="key" class="metadata-item">
									<span class="metadata-label">{{ formatLabel(key) }}:</span>
									<span class="metadata-value">
										<a v-if="isURL(value)" :href="value" target="_blank" rel="noopener">{{ value }}</a>
										<span v-else>{{ formatValue(value) }}</span>
									</span>
								</div>
							</div>
							<div v-else class="empty-tab">
								<p>No social media data available</p>
							</div>
						</div>

						<!-- Platform Tab -->
						<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'platform' }]">
							<div v-if="getPlatformData(detailsPanel.performer)" class="metadata-grid">
								<div v-for="(value, key) in getPlatformData(detailsPanel.performer)" :key="key" class="metadata-item">
									<span class="metadata-label">{{ formatLabel(key) }}:</span>
									<span class="metadata-value">{{ formatValue(value) }}</span>
								</div>
							</div>
							<div v-else class="empty-tab">
								<p>No platform data available</p>
							</div>
						</div>

						<!-- Tags Tab -->
						<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'tags' }]">
							<div class="tags-management">
								<!-- Add Tag Section -->
								<div class="add-tag-section mb-3">
									<div class="input-group">
										<select v-model.number="selectedTagId" class="form-select">
											<option value="">Select a tag...</option>
											<option v-for="tag in availableTagsForPerformer" :key="tag.id" :value="tag.id">
												{{ tag.name }}
											</option>
										</select>
										<button class="btn btn-primary" @click="addTagToPerformer" :disabled="!selectedTagId">
											<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
											Add Tag
										</button>
									</div>
								</div>

								<!-- Current Tags -->
								<div v-if="performerMasterTags.length > 0" class="tags-container">
									<div v-for="tag in performerMasterTags" :key="tag.id" class="tag-badge-removable">
										<span>{{ tag.name }}</span>
										<button class="btn-remove-tag-small" @click="removeTagFromPerformer(tag.id)" title="Remove tag">
											<font-awesome-icon :icon="['fas', 'times']" />
										</button>
									</div>
								</div>
								<div v-else class="empty-tab">
									<p>No master tags assigned</p>
								</div>

								<!-- Sync Button -->
								<div v-if="performerMasterTags.length > 0" class="mt-3">
									<button class="btn btn-outline-primary btn-sm w-100" @click="syncPerformerTags" :disabled="isSyncingTags">
										<font-awesome-icon :icon="['fas', isSyncingTags ? 'spinner' : 'sync']" :spin="isSyncingTags" class="me-2" />
										{{ isSyncingTags ? 'Syncing...' : `Sync Tags to ${detailsPanel.performer.scene_count || 0} Videos` }}
									</button>
								</div>
							</div>
						</div>

						<!-- Bios Tab -->
						<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'bios' }]">
							<div v-if="getBiosData(detailsPanel.performer)" class="bios-container">
								<div v-for="(bio, source) in getBiosData(detailsPanel.performer)" :key="source" class="bio-item">
									<h6 class="bio-source">{{ formatLabel(source) }}</h6>
									<p class="bio-text">{{ bio }}</p>
								</div>
							</div>
							<div v-else class="empty-tab">
								<p>No bios available</p>
							</div>
						</div>

						<!-- External Links Tab -->
						<div :class="['tab-pane', 'fade', { 'show active': activeTab === 'external_links' }]">
							<div v-if="getExternalLinksData(detailsPanel.performer)?.length" class="links-container">
								<a
									v-for="(link, index) in getExternalLinksData(detailsPanel.performer)"
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
							<div v-else class="empty-tab">
								<p>No external links available</p>
							</div>
						</div>
					</div>
				</div>

				<!-- Action Buttons -->
				<div class="panel-actions">
					<button class="btn btn-primary" @click="fetchMetadata(detailsPanel.performer)">
						<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
						Fetch Metadata
					</button>
					<button class="btn btn-secondary" @click="resetPerformer(detailsPanel.performer)">
						<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
						Reset
					</button>
					<button class="btn btn-danger" @click="confirmDelete(detailsPanel.performer)">
						<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
						Delete
					</button>
				</div>
			</div>
		</div>

		<!-- Delete Confirmation Modal -->
		<div v-if="deleteModal.visible" class="modal-overlay" @click="deleteModal.visible = false">
			<div class="modal-dialog" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">Confirm Delete</h5>
						<button class="btn-close-modal" @click="deleteModal.visible = false">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
					<div class="modal-body">
						<p>
							Are you sure you want to delete
							<strong>{{ deleteModal.performer?.name }}</strong
							>?
						</p>
						<p class="text-muted">This action cannot be undone.</p>
					</div>
					<div class="modal-footer">
						<button class="btn btn-secondary" @click="deleteModal.visible = false">Cancel</button>
						<button class="btn btn-danger" @click="deletePerformer">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
							Delete
						</button>
					</div>
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
	</div>
</template>

<script>
import { performersAPI } from '@/services/api'
import { tagsAPI } from '@/services/api'
import settingsService from '@/services/settingsService'

export default {
	name: 'PerformersPage',
	data() {
		// Load preview cache from localStorage
		let cachedPreviews = {}
		try {
			const stored = localStorage.getItem('performerPreviews')
			if (stored) {
				const parsed = JSON.parse(stored)
				// Validate and clean cache - ensure all paths are relative and properly formatted
				let needsUpdate = false
				for (const [performerId, paths] of Object.entries(parsed)) {
					if (Array.isArray(paths)) {
						const cleanedPaths = paths
							.map((path) => {
								if (!path || typeof path !== 'string') return null

								// If path is a full URL, extract just the pathname
								if (path.startsWith('http://') || path.startsWith('https://')) {
									try {
										return new URL(path).pathname
									} catch {
										return null
									}
								}

								// Fix paths with /../ at the start (old bug)
								if (path.startsWith('/../')) {
									needsUpdate = true
									return null // Will be reloaded from API
								}

								return path
							})
							.filter((path) => path !== null)

						// Only keep this performer's cache if we have valid paths
						if (cleanedPaths.length > 0) {
							cachedPreviews[performerId] = cleanedPaths
						} else {
							needsUpdate = true // Need to reload this performer's previews
						}
					}
				}

				// If we found invalid paths, save the cleaned cache
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
			// Clear corrupted cache
			localStorage.removeItem('performerPreviews')
		}

		// Get settings
		const settings = settingsService.getSettings()

		return {
			performers: [],
			loading: false,
			error: null,
			searchQuery: '',
			sortBy: 'name',
			viewMode: settings.defaultViewMode || 'grid',
			showFilters: false,
			filters: {
				zooFilter: settings.defaultZooFilter || 'all', // 'all', 'zoo-only', 'non-zoo'
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
			carouselIndex: 0,
			activeTab: 'basic', // Tab in details panel: basic, appearance, performances, social_media, platform, tags, bios, external_links
			deleteModal: {
				visible: false,
				performer: null,
			},
			performerPreviews: cachedPreviews, // Cache for performer previews: { performerId: [urls] }
			videoRefreshKey: 0, // Key to force video elements to recreate
			carouselContextMenu: {
				visible: false,
				x: 0,
				y: 0,
				previewIndex: 0,
			},
			// Tag Management
			allTags: [],
			performerMasterTags: [],
			selectedTagId: null,
			isSyncingTags: false,
			performerTags: {}, // Cache for performer tags: { performerId: [tags] }
		}
	},

	computed: {
		filteredPerformers() {
			// Ensure performers is always an array
			if (!Array.isArray(this.performers)) {
				console.warn('performers is not an array:', this.performers)
				return []
			}

			let result = this.performers

			// Search filter
			if (this.searchQuery) {
				const query = this.searchQuery.toLowerCase()
				result = result.filter((p) => p && p.name && p.name.toLowerCase().includes(query))
			}

			// Zoo filter
			if (this.filters.zooFilter === 'zoo-only') {
				result = result.filter((p) => p && p.zoo === true)
			} else if (this.filters.zooFilter === 'non-zoo') {
				result = result.filter((p) => p && p.zoo !== true)
			}
			// If 'all', no filtering needed

			// Age filter
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

			// Height filter (parse height from metadata string like "5'7" or "170 cm")
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

			// Breast size filter (basic implementation)
			if (this.filters.breastMin) {
				// Convert breast sizes to numeric for comparison (simplified)
				result = result.filter((p) => {
					if (!p || !p.breast_size) return false
					return p.breast_size >= this.filters.breastMin
				})
			}
			if (this.filters.breastMax) {
				result = result.filter((p) => {
					if (!p || !p.breast_size) return false
					return p.breast_size <= this.filters.breastMax
				})
			}

			// Sort
			result = [...result].sort((a, b) => {
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
						return (b.scene_count || 0) - (a.scene_count || 0)
					default:
						return 0
				}
			})

			return result
		},
		availableTagsForPerformer() {
			// Filter out tags that are already assigned to the performer
			if (!Array.isArray(this.performerMasterTags)) {
				return this.allTags || []
			}
			const assignedTagIds = new Set(this.performerMasterTags.map((t) => t.id))
			return (this.allTags || []).filter((tag) => !assignedTagIds.has(tag.id))
		},
	},
	methods: {
		// Calculate age from birthdate
		getAge(performer) {
			if (!performer.metadata?.birthdate) return null
			const birthDate = new Date(performer.metadata.birthdate)
			const today = new Date()
			let age = today.getFullYear() - birthDate.getFullYear()
			const monthDiff = today.getMonth() - birthDate.getMonth()
			if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birthDate.getDate())) {
				age--
			}
			return age
		},
		// Parse height from various formats to cm
		parseHeight(heightStr) {
			if (!heightStr) return null
			// Try to extract numeric value (assumes cm if just a number)
			const match = heightStr.match(/(\d+)/)
			return match ? parseInt(match[1]) : null
		},

		async loadPerformers() {
			this.loading = true
			this.error = null
			try {
				const response = await performersAPI.getAll()

				// Ensure we always have an array
				if (response.data) {
					// Handle different response structures
					if (Array.isArray(response.data)) {
						this.performers = response.data
					} else if (response.data.performers && Array.isArray(response.data.performers)) {
						this.performers = response.data.performers
					} else if (response.data.data && Array.isArray(response.data.data)) {
						this.performers = response.data.data
					} else {
						console.warn('Unexpected response structure:', response.data)
						this.performers = []
					}
				} else {
					this.performers = []
				}

				// Note: Previews are now loaded on-demand when opening details panel
				// This prevents API spam and improves initial page load performance

				// Load tags for all performers
				this.loadAllPerformerTags()
			} catch (err) {
				console.error('Failed to load performers:', err)
				this.error = 'Failed to load performers. Please try again.'
				this.performers = [] // Ensure it's always an array even on error
			} finally {
				this.loading = false
				console.log(this.performers)
			}
		},

		async loadAllPerformerTags() {
			// Load tags for all performers in batches to avoid API spam
			const batchSize = 5
			const performers = this.performers || []

			for (let i = 0; i < performers.length; i += batchSize) {
				const batch = performers.slice(i, i + batchSize)
				await Promise.all(
					batch.map(async (performer) => {
						try {
							const response = await performersAPI.getTags(performer.id)
							// Access response.data to get the actual tags array
							this.performerTags[performer.id] = response.data || []
						} catch (error) {
							console.error(`Failed to load tags for performer ${performer.id}:`, error)
							this.performerTags[performer.id] = []
						}
					})
				)
				// Small delay between batches to avoid overwhelming the API
				if (i + batchSize < performers.length) {
					await new Promise((resolve) => setTimeout(resolve, 100))
				}
			}
		},

		getPreviewUrl(path) {
			if (!path) {
				console.warn('getPreviewUrl called with empty path')
				return ''
			}
			// If path already includes the full URL, return as-is
			if (path.startsWith('http://') || path.startsWith('https://')) {
				return path
			}
			// Split path into segments and encode each segment (to handle spaces and special chars)
			const pathParts = path.split('/').map((part) => encodeURIComponent(part))
			const encodedPath = pathParts.join('/')
			// Otherwise, add the base URL
			const url = `http://localhost:8080${encodedPath}`
			return url
		},
		getPerformerPreviews(performer) {
			// Return cached preview URLs or fallback to primary preview
			if (this.performerPreviews[performer.id]) {
				// Convert relative paths to full URLs
				return this.performerPreviews[performer.id].map((path) => this.getPreviewUrl(path))
			}
			// Fallback to primary preview while loading
			if (performer.preview_path) {
				return [this.getPreviewUrl(performer.preview_path)]
			}
			return []
		},
		async loadPerformerPreviews(performerId) {
			// Skip if already in cache (check for existence, not just truthiness)
			if (Object.prototype.hasOwnProperty.call(this.performerPreviews, performerId) && this.performerPreviews[performerId]) {
				return
			}

			try {
				const response = await performersAPI.getPreviews(performerId)
				if (response.data && response.data.previews) {
					// Store relative paths (not full URLs) to avoid caching issues
					const previewPaths = response.data.previews.map((path) => {
						// Ensure we store clean relative paths
						if (path.startsWith('http://') || path.startsWith('https://')) {
							// Extract path from full URL if accidentally passed
							return new URL(path).pathname
						}
						return path
					})
					this.performerPreviews[performerId] = previewPaths

					// Save to localStorage
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
		playPreview(event) {
			const video = event.target
			video.play().catch(() => {
				// Ignore autoplay errors
			})
		},
		pausePreview(event) {
			const video = event.target
			video.pause()
		},
		handleVideoError(event) {
			const video = event.target
			console.error('Video failed to load:', {
				src: video.src,
				error: video.error,
				networkState: video.networkState,
				readyState: video.readyState,
			})
		},
		async openDetails(performer) {
			this.detailsPanel.visible = true
			this.detailsPanel.performer = performer
			this.carouselIndex = 0
			this.activeTab = 'basic' // Reset to basic tab
			this.selectedTagId = null // Reset selected tag
			// Load all previews for this performer
			await this.loadPerformerPreviews(performer.id)
			// Load performer master tags
			await this.loadPerformerTags(performer.id)
		},
		closeDetails() {
			this.detailsPanel.visible = false
			this.detailsPanel.performer = null
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
			if (!this.detailsPanel.performer) return

			const performer = this.detailsPanel.performer
			const previews = await performersAPI.getPreviews(performer.id)
			const selectedPreviewPath = previews.data.previews[this.carouselContextMenu.previewIndex]

			this.closeCarouselContextMenu()

			try {
				// Update performer with new primary preview path
				await performersAPI.update(performer.id, {
					preview_path: selectedPreviewPath,
				})

				// Reload performers
				await this.loadPerformers()

				// Update details panel
				const updatedPerformer = this.performers.find((p) => p.id === performer.id)
				if (updatedPerformer) {
					this.detailsPanel.performer = updatedPerformer
				}

				this.$toast.success('Primary Set', 'Primary preview updated successfully')
			} catch (err) {
				console.error('Failed to set primary preview:', err)
				this.$toast.error('Update Failed', 'Failed to set primary preview')
			}
		},
		async fetchMetadata(performer) {
			this.closeContextMenu()
			try {
				// Call API to fetch metadata from AdultDataLink
				const response = await performersAPI.fetchMetadata(performer.id)
				console.log('Metadata fetch response:', response)
				// Reload performers to get updated data
				await this.loadPerformers()
				// Reload the current performer in details panel
				if (this.detailsPanel.visible && this.detailsPanel.performer?.id === performer.id) {
					const updatedPerformer = this.performers.find((p) => p.id === performer.id)
					if (updatedPerformer) {
						this.detailsPanel.performer = updatedPerformer
					}
				}
				this.$toast.success('Metadata Fetched', `Successfully fetched metadata for ${performer.name}`)
			} catch (err) {
				console.error('Failed to fetch metadata:', err)
				this.$toast.error('Fetch Failed', err.response?.data?.error || 'Failed to fetch metadata from AdultDataLink')
			}
		},
		async toggleZoo(performer) {
			this.closeContextMenu()
			try {
				const newZooValue = !performer.zoo
				const response = await performersAPI.update(performer.id, { zoo: newZooValue })

				// Update local state with response from server
				if (response && response.zoo !== undefined) {
					performer.zoo = response.zoo
				} else {
					performer.zoo = newZooValue
				}

				// Force re-render
				this.$forceUpdate()

				this.$toast.success('Updated', `${performer.name} ${newZooValue ? 'marked as Zoo' : 'unmarked as Zoo'}`)
			} catch (err) {
				console.error('Failed to toggle zoo:', err)
				this.$toast.error('Update Failed', 'Failed to update zoo status')
			}
		},
		async resetPerformer(performer) {
			this.closeContextMenu()
			try {
				// Call API to reset performer metadata
				await performersAPI.resetMetadata(performer.id)
				// Reload performers
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
				// Remove from local list
				this.performers = this.performers.filter((p) => p.id !== performer.id)
				this.$toast.success('Deleted', `${performer.name} has been deleted`)
			} catch (err) {
				console.error('Failed to delete performer:', err)
				this.$toast.error('Delete Failed', 'Failed to delete performer')
			}
		},
		clearFilters() {
			this.filters = {
				showZoo: true,
				ageMin: null,
				ageMax: null,
				breastMin: '',
				breastMax: '',
				heightMin: null,
				heightMax: null,
			}
		},
		formatDate(dateString) {
			if (!dateString) return ''
			const date = new Date(dateString)
			return date.toLocaleDateString()
		},
		// Helper methods for tabbed metadata display
		getAppearanceData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.appearance) {
				return null
			}
			// Check if it's an empty object
			if (Object.keys(adlData.appearance).length === 0) {
				return null
			}
			// Convert to plain object to avoid reactivity issues with v-for
			return { ...adlData.appearance }
		},
		getPerformancesData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.performances) return null
			if (Object.keys(adlData.performances).length === 0) return null
			return { ...adlData.performances }
		},
		getSocialMediaData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.social_media) return null
			if (Object.keys(adlData.social_media).length === 0) return null
			return { ...adlData.social_media }
		},
		getPlatformData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData) return null
			// Combine platform_views, platform_video_counts, platform_profile_counts
			const platformData = {}
			if (adlData.platform_views) {
				Object.entries(adlData.platform_views).forEach(([key, value]) => {
					platformData[`${key}_views`] = value
				})
			}
			if (adlData.platform_video_counts) {
				Object.entries(adlData.platform_video_counts).forEach(([key, value]) => {
					platformData[`${key}_videos`] = value
				})
			}
			if (adlData.platform_profile_counts) {
				Object.entries(adlData.platform_profile_counts).forEach(([key, value]) => {
					platformData[`${key}_profiles`] = value
				})
			}
			return Object.keys(platformData).length > 0 ? platformData : null
		},
		// Tag Management Methods
		async loadPerformerTags(performerId) {
			try {
				const response = await performersAPI.getTags(performerId)
				// Response interceptor unwraps to {success, message, data}
				// We need to access .data to get the actual tags array
				this.performerMasterTags = response.data || []
			} catch (error) {
				console.error('Failed to load performer tags:', error)
				this.performerMasterTags = []
			}
		},
		async addTagToPerformer() {
			if (!this.selectedTagId || !this.detailsPanel.performer) return

			const tagId = parseInt(this.selectedTagId)
			const performerId = this.detailsPanel.performer.id

			try {
				await performersAPI.addTag(performerId, tagId)
				this.$toast.success('Tag Added', 'Master tag added to performer')
				await this.loadPerformerTags(performerId)
				this.selectedTagId = null
			} catch (error) {
				console.error('Failed to add tag:', error)
				if (error.response) {
					console.error('Error response:', error.response.data)
				}
				this.$toast.error('Error', 'Failed to add master tag')
			}
		},
		async removeTagFromPerformer(tagId) {
			if (!this.detailsPanel.performer) return
			if (!confirm('Remove this master tag from the performer?')) return

			try {
				await performersAPI.removeTag(this.detailsPanel.performer.id, tagId)
				this.$toast.success('Tag Removed', 'Master tag removed from performer')
				await this.loadPerformerTags(this.detailsPanel.performer.id)
			} catch (error) {
				console.error('Failed to remove tag:', error)
				this.$toast.error('Error', 'Failed to remove master tag')
			}
		},
		async syncPerformerTags() {
			if (!this.detailsPanel.performer) return
			const sceneCount = this.detailsPanel.performer.scene_count || 0
			if (!confirm(`Apply master tags to all ${sceneCount} videos featuring this performer?`)) return

			this.isSyncingTags = true
			try {
				const response = await performersAPI.syncTags(this.detailsPanel.performer.id)
				// Response interceptor unwraps to {success, message, data}
				const videosUpdated = response.data?.videos_updated || 0
				this.$toast.success('Sync Complete', `Applied master tags to ${videosUpdated} videos`)
			} catch (error) {
				console.error('Failed to sync tags:', error)
				this.$toast.error('Error', 'Failed to sync master tags to videos')
			} finally {
				this.isSyncingTags = false
			}
		},
		async loadAllTags() {
			let storeTags = await this.$store.dispatch('fetchTags')
			if (storeTags.length !== 0) {
				this.allTags = storeTags
				return
			} else {
				const response = await tagsAPI.getAll()
				this.allTags = response || []
			}
		},
		getBiosData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.bios) return null
			if (Object.keys(adlData.bios).length === 0) return null
			return { ...adlData.bios }
		},
		getExternalLinksData(performer) {
			const adlData = performer.metadata?.adult_data_link_response
			if (!adlData || !adlData.external_links) return []
			// Return plain array
			return [...adlData.external_links]
		},
		formatLabel(key) {
			// Convert snake_case to Title Case
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
	mounted() {
		this.loadPerformers()
		this.loadAllTags()

		// Close context menus on click outside
		document.addEventListener('click', () => {
			if (this.contextMenu.visible) {
				this.closeContextMenu()
			}
			if (this.carouselContextMenu.visible) {
				this.closeCarouselContextMenu()
			}
		})
	},
	activated() {
		// Called when component is activated (navigated to) in keep-alive
		// Wait for performers to be loaded, then reload videos
		let unwatch = null
		unwatch = this.$watch(
			'performers',
			(newVal) => {
				if (newVal && newVal.length > 0) {
					// Wait for DOM to update with the performer data
					this.$nextTick(() => {
						const videos = this.$el.querySelectorAll('video')
						videos.forEach((video) => {
							if (video.src) {
								// Force browser to reload the video by resetting src
								const currentSrc = video.src
								video.src = ''
								video.load()
								video.src = currentSrc
								video.load()

								// Force opacity and position inline styles to fix transition issue
								video.style.opacity = '1'
								video.style.position = 'static'
							}
						})
					})
					// Stop watching after first trigger
					if (unwatch) unwatch()
				}
			},
			{ immediate: true }
		)
	},
	beforeUnmount() {
		// Cleanup if needed
	},
}
</script>

<style scoped>
@import '@/styles/pages/performers_page.css';
</style>
