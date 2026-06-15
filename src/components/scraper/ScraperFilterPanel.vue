<!-- eslint-disable no-undef -->
<template>
	<div class="scraper-filter-panel" :class="{ 'is-collapsed': modelCollapsed }">
		<div class="sfp-inner">
			<!-- Header -->
			<div class="sfp-header">
				<div class="sfp-title">
					<font-awesome-icon :icon="['fas', 'sliders-h']" />
					<span>Filters &amp; Sort</span>
					<span v-if="activeCount > 0" class="sfp-active-badge">{{ activeCount }}</span>
				</div>
				<button class="sfp-close-btn" @click="$emit('update:modelCollapsed', true)" title="Hide filters">
					<font-awesome-icon :icon="['fas', 'chevron-left']" />
				</button>
			</div>

			<!-- Clear All -->
			<button v-if="activeCount > 0" class="sfp-clear-all" @click="clearAll">
				<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
				Clear All Filters
			</button>

			<!-- Sort -->
			<div class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'sort']" />
					Sort By
				</div>
				<select :value="sort" @change="$emit('update:sort', $event.target.value)" class="form-select form-select-sm sfp-select">
					<optgroup label="Date">
						<option value="newest">Newest First</option>
						<option value="oldest">Oldest First</option>
					</optgroup>
					<optgroup label="Popularity">
						<option value="views-desc">Most Views</option>
						<option value="views-asc">Least Views</option>
						<option value="likes-desc">Most Likes</option>
						<option value="likes-asc">Least Likes</option>
					</optgroup>
					<optgroup label="Quality">
						<option value="rating-desc">Highest Rated</option>
						<option value="rating-asc">Lowest Rated</option>
					</optgroup>
					<optgroup label="Length">
						<option value="duration-desc">Longest</option>
						<option value="duration-asc">Shortest</option>
					</optgroup>
					<optgroup label="Title">
						<option value="title-asc">Title A–Z</option>
						<option value="title-desc">Title Z–A</option>
					</optgroup>
					<optgroup v-if="showFileSizeSort" label="Size">
						<option value="filesize-desc">Largest</option>
						<option value="filesize-asc">Smallest</option>
					</optgroup>
				</select>
			</div>

			<!-- Duration Slider -->
			<div class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'clock']" />
					Duration
					<span v-if="minDuration > 0 || maxDuration > 0" class="sfp-duration-value">{{ durationLabel }}</span>
				</div>
				<div class="sfp-slider-group">
					<div class="sfp-slider-row">
						<span class="sfp-slider-label">Min</span>
						<input type="range" class="form-range sfp-slider" min="0" :max="maxSliderValue" step="5" :value="minDuration" @input="onMinDurationInput" />
						<span class="sfp-slider-val">{{ formatMinutes(minDuration) }}</span>
					</div>
					<div class="sfp-slider-row">
						<span class="sfp-slider-label">Max</span>
						<input type="range" class="form-range sfp-slider" min="0" :max="maxSliderValue" step="5" :value="maxDuration" @input="onMaxDurationInput" />
						<span class="sfp-slider-val">{{ maxDuration > 0 ? formatMinutes(maxDuration) : 'Any' }}</span>
					</div>
					<div v-if="minDuration > 0 || maxDuration > 0" class="sfp-slider-reset" @click="resetDuration">
						<font-awesome-icon :icon="['fas', 'times']" class="me-1" /> Clear
					</div>
				</div>
			</div>

			<!-- Date Range (optional) -->
			<div v-if="showDateRange" class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'calendar']" />
					Date Range
				</div>
				<div class="sfp-range-row">
					<input type="date" class="form-control form-control-sm sfp-input" :value="dateFrom" @input="$emit('update:dateFrom', $event.target.value)" />
					<span class="sfp-range-sep">–</span>
					<input type="date" class="form-control form-control-sm sfp-input" :value="dateTo" @input="$emit('update:dateTo', $event.target.value)" />
				</div>
			</div>

			<!-- File Size Filter (optional) -->
			<div v-if="showFileSizeFilter" class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'hdd']" />
					File Size
				</div>
				<select :value="fileSizeRange" @change="$emit('update:fileSizeRange', $event.target.value)" class="form-select form-select-sm sfp-select">
					<option value="">Any Size</option>
					<option value="tiny">Tiny (&lt;100 MB)</option>
					<option value="small">Small (100–500 MB)</option>
					<option value="medium">Medium (500 MB–2 GB)</option>
					<option value="large">Large (2–10 GB)</option>
					<option value="huge">Huge (&gt;10 GB)</option>
				</select>
			</div>

			<!-- Quick Filters -->
			<div class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'bolt']" />
					Quick Filters
				</div>
				<div
					class="sfp-toggle-row"
					:class="{ active: onlyDownloaded }"
					@click="$emit('update:onlyDownloaded', !onlyDownloaded)"
				>
					<div class="sfp-toggle-icon">
						<font-awesome-icon :icon="['fas', 'download']" />
					</div>
					<span class="sfp-toggle-label">Downloaded Only</span>
					<div class="sfp-toggle-switch" :class="{ on: onlyDownloaded }">
						<div class="sfp-toggle-knob" />
					</div>
				</div>
				<div v-if="showOnlyWithRating" class="sfp-toggle-row" :class="{ active: onlyWithRating }" @click="$emit('update:onlyWithRating', !onlyWithRating)">
					<div class="sfp-toggle-icon"><font-awesome-icon :icon="['fas', 'star']" /></div>
					<span class="sfp-toggle-label">Has Rating</span>
					<div class="sfp-toggle-switch" :class="{ on: onlyWithRating }"><div class="sfp-toggle-knob" /></div>
				</div>
				<div v-if="showOnlyWithAuthor" class="sfp-toggle-row" :class="{ active: onlyWithAuthor }" @click="$emit('update:onlyWithAuthor', !onlyWithAuthor)">
					<div class="sfp-toggle-icon"><font-awesome-icon :icon="['fas', 'user']" /></div>
					<span class="sfp-toggle-label">Has Author</span>
					<div class="sfp-toggle-switch" :class="{ on: onlyWithAuthor }"><div class="sfp-toggle-knob" /></div>
				</div>
			</div>

			<!-- Min Rating -->
			<div v-if="showMinRating" class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'star']" />
					Min Rating
				</div>
				<select :value="minRating" @change="$emit('update:minRating', parseInt($event.target.value))" class="form-select form-select-sm sfp-select">
					<option value="0">Any Rating</option>
					<option value="50">50%+</option>
					<option value="70">70%+</option>
					<option value="80">80%+</option>
					<option value="90">90%+</option>
				</select>
			</div>

			<!-- Min Views -->
			<div v-if="showMinViews" class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'eye']" />
					Min Views
				</div>
				<select :value="minViews" @change="$emit('update:minViews', parseInt($event.target.value))" class="form-select form-select-sm sfp-select">
					<option value="0">Any Views</option>
					<option value="1000">1K+</option>
					<option value="10000">10K+</option>
					<option value="100000">100K+</option>
					<option value="500000">500K+</option>
					<option value="1000000">1M+</option>
					<option value="10000000">10M+</option>
				</select>
			</div>

			<!-- Min Likes -->
			<div v-if="showMinLikes" class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'heart']" />
					Min Likes
				</div>
				<select :value="minLikes" @change="$emit('update:minLikes', parseInt($event.target.value))" class="form-select form-select-sm sfp-select">
					<option value="0">Any Likes</option>
					<option value="100">100+</option>
					<option value="1000">1K+</option>
					<option value="10000">10K+</option>
					<option value="100000">100K+</option>
				</select>
			</div>

			<!-- Performers (optional) -->
			<div v-if="performers.length > 0" class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'user']" />
					Performer
				</div>
				<select :value="performerId" @change="$emit('update:performerId', parseInt($event.target.value))" class="form-select form-select-sm sfp-select">
					<option :value="0">All Performers</option>
					<option v-for="p in performers" :key="p.id" :value="p.id">{{ p.name }}</option>
				</select>
			</div>

			<!-- Tags (optional) -->
			<div v-if="tags.length > 0" class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'tag']" />
					Tag
				</div>
				<select :value="tagId" @change="$emit('update:tagId', parseInt($event.target.value))" class="form-select form-select-sm sfp-select">
					<option :value="0">All Tags</option>
					<option v-for="t in tags" :key="t.id" :value="t.id">{{ t.name }}</option>
				</select>
			</div>

			<!-- Custom slot for per-scraper filters -->
			<slot />
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
import { computed } from 'vue'

const MAX_SLIDER_MINUTES = 120 // slider max = 2 hours

const props = defineProps({
	modelCollapsed:    { type: Boolean, default: false },
	sort:              { type: String,  default: 'newest' },
	onlyDownloaded:    { type: Boolean, default: false },
	performers:        { type: Array,   default: () => [] },
	performerId:       { type: Number,  default: 0 },
	tags:              { type: Array,   default: () => [] },
	tagId:             { type: Number,  default: 0 },
	minRating:         { type: Number,  default: 0 },
	minViews:          { type: Number,  default: 0 },
	minLikes:          { type: Number,  default: 0 },
	showMinRating:     { type: Boolean, default: false },
	showMinViews:      { type: Boolean, default: false },
	showMinLikes:      { type: Boolean, default: false },
	showFileSizeSort:  { type: Boolean, default: false },
	showFileSizeFilter:{ type: Boolean, default: false },
	fileSizeRange:     { type: String,  default: '' },
	showDateRange:     { type: Boolean, default: false },
	dateFrom:          { type: String,  default: '' },
	dateTo:            { type: String,  default: '' },
	minDuration:       { type: Number,  default: 0 },
	maxDuration:       { type: Number,  default: 0 },
	showOnlyWithRating:{ type: Boolean, default: false },
	onlyWithRating:    { type: Boolean, default: false },
	showOnlyWithAuthor:{ type: Boolean, default: false },
	onlyWithAuthor:    { type: Boolean, default: false },
})

const emit = defineEmits([
	'update:modelCollapsed',
	'update:sort',
	'update:onlyDownloaded',
	'update:performerId',
	'update:tagId',
	'update:minRating',
	'update:minViews',
	'update:minLikes',
	'update:fileSizeRange',
	'update:dateFrom',
	'update:dateTo',
	'update:minDuration',
	'update:maxDuration',
	'update:onlyWithRating',
	'update:onlyWithAuthor',
	'clear',
])

const maxSliderValue = computed(() => MAX_SLIDER_MINUTES)

const formatMinutes = (val) => {
	if (!val || val <= 0) return '0m'
	const h = Math.floor(val / 60)
	const m = val % 60
	if (h > 0 && m > 0) return `${h}h ${m}m`
	if (h > 0) return `${h}h`
	return `${m}m`
}

const durationLabel = computed(() => {
	const min = props.minDuration
	const max = props.maxDuration
	if (min > 0 && max > 0) return `${formatMinutes(min)} – ${formatMinutes(max)}`
	if (min > 0) return `${formatMinutes(min)}+`
	if (max > 0) return `≤ ${formatMinutes(max)}`
	return ''
})

const onMinDurationInput = (e) => {
	const val = parseInt(e.target.value) || 0
	emit('update:minDuration', val)
	// Auto-clamp: min can't exceed max
	if (props.maxDuration > 0 && val > props.maxDuration) {
		emit('update:maxDuration', val)
	}
}

const onMaxDurationInput = (e) => {
	const val = parseInt(e.target.value) || 0
	emit('update:maxDuration', val)
	// Auto-clamp: max can't be less than min
	if (val > 0 && props.minDuration > val) {
		emit('update:minDuration', val)
	}
}

const resetDuration = () => {
	emit('update:minDuration', 0)
	emit('update:maxDuration', 0)
}

const activeCount = computed(() => {
	let n = 0
	if (props.sort !== 'newest')           n++
	if (props.onlyDownloaded)              n++
	if (props.performerId !== 0)           n++
	if (props.tagId !== 0)                 n++
	if (props.minRating > 0)               n++
	if (props.minViews > 0)                n++
	if (props.minLikes > 0)                n++
	if (props.fileSizeRange !== '')        n++
	if (props.dateFrom !== '')             n++
	if (props.dateTo !== '')               n++
	if (props.minDuration > 0)             n++
	if (props.maxDuration > 0)             n++
	if (props.onlyWithRating)              n++
	if (props.onlyWithAuthor)              n++
	return n
})

const clearAll = () => {
	emit('update:sort',             'newest')
	emit('update:onlyDownloaded',   false)
	emit('update:performerId',      0)
	emit('update:tagId',            0)
	emit('update:minRating',        0)
	emit('update:minViews',         0)
	emit('update:minLikes',         0)
	emit('update:fileSizeRange',    '')
	emit('update:dateFrom',         '')
	emit('update:dateTo',           '')
	emit('update:minDuration',      0)
	emit('update:maxDuration',      0)
	emit('update:onlyWithRating',   false)
	emit('update:onlyWithAuthor',   false)
	emit('clear')
}
</script>
