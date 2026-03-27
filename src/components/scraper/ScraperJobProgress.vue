<template>
	<div v-if="job" class="card bg-dark mb-3">
		<div class="card-body py-2">
			<div class="d-flex justify-content-between align-items-center">
				<span class="small">
					<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-danger" />
					Scraping {{ label }}...
				</span>
				<span class="small text-muted">
					<strong class="text-white">{{ job.pagesCompleted }}</strong> pages &nbsp;|&nbsp;
					<strong class="text-white">{{ job.videosFound }}</strong> found &nbsp;|&nbsp;
					<strong class="text-white">{{ job.videosSaved }}</strong> saved
					<span v-if="job.errorsCount > 0" class="text-danger ms-2">
						{{ job.errorsCount }} errors
					</span>
				</span>
			</div>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
defineProps({
	job:   { type: Object, default: null },
	label: { type: String, default: ''   },
})
</script>
