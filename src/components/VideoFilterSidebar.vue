<template>
	<!-- eslint-disable vue/no-mutating-props -->
	<div class="vp-filter-sidebar">
		<div class="vp-filter-panel">
			<div class="vp-filter-header">
				<div class="vp-filter-title-group">
					<h3>
						<font-awesome-icon :icon="['fas', 'sliders-h']" />
						Filters &amp; Sort
					</h3>
					<span v-if="activeFiltersCount > 0" class="vp-active-filters-badge">{{ activeFiltersCount }}</span>
				</div>
				<button class="vp-clear-filters-btn" @click="$emit('clear')" title="Clear all filters">
					<font-awesome-icon :icon="['fas', 'times-circle']" />
				</button>
			</div>

			<!-- Sort Section -->
			<div class="vp-filter-section">
				<div class="vp-section-header">
					<font-awesome-icon :icon="['fas', 'sort']" />
					<span>Sort</span>
				</div>
				<div class="vp-sort-controls">
					<select :value="sortBy" class="form-select form-select-sm" @change="$emit('update:sort-by', $event.target.value); $emit('change')">
						<option value="created_at">📅 Date Added</option>
						<option value="title">🔤 Title</option>
						<option value="duration">⏱️ Duration</option>
						<option value="play_count">👁️ Views</option>
					</select>
					<div class="vp-sort-order-toggle">
						<button :class="{ active: sortOrder === 'desc' }" @click="$emit('update:sort-order', 'desc'); $emit('change')" title="Descending">
							<font-awesome-icon :icon="['fas', 'sort-amount-down']" />
						</button>
						<button :class="{ active: sortOrder === 'asc' }" @click="$emit('update:sort-order', 'asc'); $emit('change')" title="Ascending">
							<font-awesome-icon :icon="['fas', 'sort-amount-up']" />
						</button>
					</div>
				</div>
			</div>

			<!-- Content Filters -->
			<div class="vp-filter-section">
				<div class="vp-section-header">
					<font-awesome-icon :icon="['fas', 'filter']" />
					<span>Content</span>
				</div>

				<div v-if="performers.length > 0" class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'user']" />
						Performers
						<span v-if="filteredPerformers.length !== performers.length" class="vp-filter-note">({{ filteredPerformers.length }})</span>
					</label>
					<select v-model="filters.performerId" class="form-select form-select-sm" @change="$emit('change')">
						<option :value="null">All Performers</option>
						<option v-for="performer in filteredPerformers" :key="performer.id" :value="performer.id">{{ performer.name }}</option>
					</select>
				</div>

				<div class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'building']" />
						Studios
					</label>
					<select v-model="filters.studioId" class="form-select form-select-sm" @change="onStudioChange">
						<option :value="null">All Studios</option>
						<option v-for="studio in filteredStudios" :key="studio.id" :value="studio.id">{{ studio.name }}</option>
					</select>
				</div>

				<div v-if="filteredGroups.length > 0" class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'layer-group']" />
						Groups
					</label>
					<select v-model="filters.groupId" class="form-select form-select-sm" @change="$emit('change')">
						<option :value="null">All Groups</option>
						<option v-for="group in filteredGroups" :key="group.id" :value="group.id">{{ group.name }}</option>
					</select>
				</div>

				<div v-if="filteredTags.length > 0" class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'tags']" />
						Tags
						<span v-if="filters.selectedTags.length > 0" class="vp-filter-value">{{ filters.selectedTags.length }} selected</span>
					</label>
					<div class="vp-tags-multiselect">
						<div
							v-for="tag in filteredTags"
							:key="tag.id"
							class="vp-tag-item"
							:class="{ selected: filters.selectedTags.includes(tag.id) }"
							@click="toggleTag(tag.id)"
						>
							<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
							<span>{{ tag.name }}</span>
						</div>
					</div>
				</div>

				<div class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'list']" />
						Content Type
					</label>
					<select v-model="filters.contentType" class="form-select form-select-sm" @change="$emit('change')">
						<option :value="null">All Types</option>
						<option value="regular">Regular</option>
						<option value="zoo">Zoo</option>
						<option value="3d">3D</option>
					</select>
				</div>

				<div class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'calendar']" />
						Date Range
					</label>
					<input v-model="filters.dateFrom" type="date" class="form-control form-control-sm mb-2" placeholder="From" @change="$emit('change')" />
					<input v-model="filters.dateTo" type="date" class="form-control form-control-sm" placeholder="To" @change="$emit('change')" />
				</div>
			</div>

			<!-- Media Properties -->
			<div class="vp-filter-section">
				<div class="vp-section-header">
					<font-awesome-icon :icon="['fas', 'cog']" />
					<span>Media Properties</span>
				</div>

				<div class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'clock']" />
						Duration
						<span v-if="filters.minDuration || filters.maxDuration" class="vp-filter-value">
							{{ filters.minDuration || 0 }}m - {{ filters.maxDuration || 180 }}m
						</span>
					</label>
					<div class="vp-range-slider">
						<input v-model.number="filters.minDuration" type="range" min="0" max="180" step="5" class="vp-slider vp-slider-min" @change="$emit('change')" />
						<input v-model.number="filters.maxDuration" type="range" min="0" max="180" step="5" class="vp-slider vp-slider-max" @change="$emit('change')" />
						<div class="vp-slider-track"></div>
					</div>
				</div>

				<div class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'expand']" />
						Resolution
					</label>
					<select v-model="filters.resolution" class="form-select form-select-sm" @change="$emit('change')">
						<option value="">All Resolutions</option>
						<option value="3840x2160">🎬 4K (2160p)</option>
						<option value="1920x1080">📺 1080p (Full HD)</option>
						<option value="1280x720">📱 720p (HD)</option>
					</select>
				</div>

				<div class="vp-filter-group">
					<label>
						<font-awesome-icon :icon="['fas', 'hdd']" />
						File Size
						<span v-if="filters.minSize || filters.maxSize" class="vp-filter-value"> {{ filters.minSize || 0 }}MB - {{ filters.maxSize || 10000 }}MB </span>
					</label>
					<div class="vp-range-slider">
						<input v-model.number="filters.minSize" type="range" min="0" max="10000" step="100" class="vp-slider vp-slider-min" @change="$emit('change')" />
						<input v-model.number="filters.maxSize" type="range" min="0" max="10000" step="100" class="vp-slider vp-slider-max" @change="$emit('change')" />
						<div class="vp-slider-track"></div>
					</div>
				</div>

				<div class="vp-filter-group">
					<label class="vp-toggle-label" style="cursor:pointer;" @click="filters.showMissing = !filters.showMissing; $emit('change')">
						<font-awesome-icon :icon="['fas', 'video-slash']" :class="filters.showMissing ? 'text-danger' : 'text-muted'" />
						Show Missing Files
						<span v-if="filters.showMissing" class="badge bg-danger ms-1">ON</span>
					</label>
					<small class="text-muted d-block" style="font-size:0.75em;">Files no longer found on disk</small>
				</div>

				<div class="vp-filter-group">
					<label class="vp-toggle-label" style="cursor:pointer;" @click="filters.noPerformer = !filters.noPerformer; $emit('change')">
						<font-awesome-icon :icon="['fas', 'user-slash']" :class="filters.noPerformer ? 'text-warning' : 'text-muted'" />
						No Performer Linked
						<span v-if="filters.noPerformer" class="badge bg-warning text-dark ms-1">ON</span>
					</label>
					<small class="text-muted d-block" style="font-size:0.75em;">Videos with no performer assigned</small>
				</div>
			</div>
		</div>
	</div>
</template>

<script>
/* eslint-disable vue/no-mutating-props */
export default {
	name: 'VideoFilterSidebar',
	props: {
		performers: { type: Array, default: () => [] },
		filteredPerformers: { type: Array, default: () => [] },
		studios: { type: Array, default: () => [] },
		filteredStudios: { type: Array, default: () => [] },
		groups: { type: Array, default: () => [] },
		filteredGroups: { type: Array, default: () => [] },
		tags: { type: Array, default: () => [] },
		filteredTags: { type: Array, default: () => [] },
		filters: { type: Object, required: true },
		activeFiltersCount: { type: Number, default: 0 },
		sortBy: { type: String, default: 'created_at' },
		sortOrder: { type: String, default: 'desc' },
	},
	emits: ['update:sort-by', 'update:sort-order', 'change', 'clear'],
	methods: {
		toggleTag(tagId) {
			const index = this.filters.selectedTags.indexOf(tagId)
			if (index > -1) {
				this.filters.selectedTags.splice(index, 1)
			} else {
				this.filters.selectedTags.push(tagId)
			}
			this.$emit('change')
		},
		onStudioChange() {
			this.filters.groupId = null
			this.$emit('change')
		},
	},
}
</script>
