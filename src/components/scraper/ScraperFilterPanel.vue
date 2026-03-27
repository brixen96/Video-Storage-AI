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
					<option value="newest">Newest First</option>
					<option value="oldest">Oldest First</option>
					<option value="views-desc">Most Views</option>
					<option value="views-asc">Least Views</option>
					<option value="rating-desc">Highest Rated</option>
					<option value="rating-asc">Lowest Rated</option>
					<option value="duration-desc">Longest</option>
					<option value="duration-asc">Shortest</option>
					<option value="title-asc">Title A–Z</option>
				</select>
			</div>

			<!-- Duration -->
			<div class="sfp-section">
				<div class="sfp-section-label">
					<font-awesome-icon :icon="['fas', 'clock']" />
					Duration
				</div>
				<div class="sfp-btn-group">
					<button v-for="d in DURATIONS" :key="d.value" class="sfp-chip" :class="{ active: duration === d.value }" @click="$emit('update:duration', d.value)">
						{{ d.label }}
					</button>
				</div>
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

const DURATIONS = [
	{ value: 'all',    label: 'All'      },
	{ value: 'short',  label: '< 5 min'  },
	{ value: 'medium', label: '5–20 min' },
	{ value: 'long',   label: '> 20 min' },
]

const props = defineProps({
	modelCollapsed:  { type: Boolean, default: false },
	sort:            { type: String,  default: 'newest' },
	duration:        { type: String,  default: 'all' },
	onlyDownloaded:  { type: Boolean, default: false },
	performers:      { type: Array,   default: () => [] },
	performerId:     { type: Number,  default: 0 },
	tags:            { type: Array,   default: () => [] },
	tagId:           { type: Number,  default: 0 },
	minRating:       { type: Number,  default: 0 },
	minViews:        { type: Number,  default: 0 },
	showMinRating:   { type: Boolean, default: false },
	showMinViews:    { type: Boolean, default: false },
})

const emit = defineEmits([
	'update:modelCollapsed',
	'update:sort',
	'update:duration',
	'update:onlyDownloaded',
	'update:performerId',
	'update:tagId',
	'update:minRating',
	'update:minViews',
	'clear',
])

const activeCount = computed(() => {
	let n = 0
	if (props.sort !== 'newest')     n++
	if (props.duration !== 'all')    n++
	if (props.onlyDownloaded)        n++
	if (props.performerId !== 0)     n++
	if (props.tagId !== 0)           n++
	if (props.minRating > 0)         n++
	if (props.minViews > 0)          n++
	return n
})

const clearAll = () => {
	emit('update:sort',           'newest')
	emit('update:duration',       'all')
	emit('update:onlyDownloaded', false)
	emit('update:performerId',    0)
	emit('update:tagId',          0)
	emit('update:minRating',      0)
	emit('update:minViews',       0)
	emit('clear')
}
</script>
