<template>
	<div class="search-filter-bar mb-3">
		<div class="row g-2">
			<!-- Search -->
			<div class="col-md-6">
				<div class="input-group">
					<span class="input-group-text bg-dark border-secondary">
						<font-awesome-icon :icon="['fas', 'search']" />
					</span>
					<input
						:value="searchQuery"
						@input="$emit('update:searchQuery', $event.target.value)"
						type="text"
						class="form-control bg-dark text-white border-secondary"
						placeholder="Search videos..."
					/>
					<button v-if="searchQuery" class="btn btn-outline-secondary" @click="$emit('update:searchQuery', '')">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
			</div>

			<!-- Filters -->
			<div class="col-md-6">
				<div class="d-flex gap-2 filter-buttons">
					<!-- Filter Type -->
					<div class="btn-group" role="group">
						<button type="button" class="btn btn-sm btn-outline-secondary" :class="{ active: filterType === 'all' }" @click="$emit('update:filterType', 'all')">
							All
						</button>
						<button
							type="button"
							class="btn btn-sm btn-outline-secondary"
							:class="{ active: filterType === 'videos' }"
							@click="$emit('update:filterType', 'videos')"
						>
							Videos
						</button>
						<button
							type="button"
							class="btn btn-sm btn-outline-secondary"
							:class="{ active: filterType === 'folders' }"
							@click="$emit('update:filterType', 'folders')"
						>
							Folders
						</button>
					</div>

					<!-- Sort -->
					<select :value="sortBy" class="form-select form-select-sm bg-dark text-white border-secondary" @change="$emit('update:sortBy', $event.target.value)">
						<option value="name">Name</option>
						<option value="date">Date</option>
						<option value="size">Size</option>
						<option value="duration">Duration</option>
					</select>
					<button class="btn btn-sm btn-outline-secondary" @click="toggleSortOrder" :title="sortOrder">
						<font-awesome-icon :icon="['fas', sortOrder === 'asc' ? 'arrow-up' : 'arrow-down']" />
					</button>

					<!-- Mark Filters -->
					<button class="btn btn-sm btn-outline-danger" :class="{ active: showNotInterested }" @click="toggleShowNotInterested" title="Show Not Interested">
						<font-awesome-icon :icon="['fas', 'times-circle']" />
					</button>
					<button class="btn btn-sm btn-outline-success" :class="{ active: showEditList }" @click="toggleShowEditList" title="Show Edit List">
						<font-awesome-icon :icon="['fas', 'list']" />
					</button>

					<div class="vr d-none d-md-block"></div>
					<button class="btn btn-sm btn-outline-primary" @click="$emit('refresh')" title="Refresh folder">
						<font-awesome-icon :icon="['fas', 'sync']" :class="{ 'fa-spin': isLoading }" />
						<span class="ms-1 d-none d-lg-inline">Refresh</span>
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
const props = defineProps({
	searchQuery: {
		type: String,
		default: '',
	},
	filterType: {
		type: String,
		default: 'all',
		validator: (value) => ['all', 'videos', 'folders'].includes(value),
	},
	sortBy: {
		type: String,
		default: 'name',
		validator: (value) => ['name', 'date', 'size', 'duration'].includes(value),
	},
	sortOrder: {
		type: String,
		default: 'asc',
		validator: (value) => ['asc', 'desc'].includes(value),
	},
	showNotInterested: {
		type: Boolean,
		default: false,
	},
	showEditList: {
		type: Boolean,
		default: false,
	},
	isLoading: {
		type: Boolean,
		default: false,
	},
})

const emit = defineEmits(['update:searchQuery', 'update:filterType', 'update:sortBy', 'update:sortOrder', 'update:showNotInterested', 'update:showEditList', 'refresh'])

// Toggle methods
const toggleSortOrder = () => {
	emit('update:sortOrder', props.sortOrder === 'asc' ? 'desc' : 'asc')
}

const toggleShowNotInterested = () => {
	emit('update:showNotInterested', !props.showNotInterested)
}

const toggleShowEditList = () => {
	emit('update:showEditList', !props.showEditList)
}
</script>

<style scoped>
/* Component uses styles from parent browser_page.css */
</style>
