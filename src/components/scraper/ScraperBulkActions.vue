<template>
	<div
		v-if="selectionMode && selectedCount > 0"
		class="bulk-actions-toolbar mb-3"
	>
		<div class="d-flex justify-content-between align-items-center">
			<span class="badge bg-primary">{{ selectedCount }} selected</span>
			<div class="d-flex gap-2">
				<button class="btn btn-sm btn-outline-secondary" @click="emit('select-all')">
					Select All
				</button>
				<button class="btn btn-sm btn-outline-primary" @click="emit('send-to-jd')" :disabled="sendingToJD" title="Send selected to JDownloader">
					<font-awesome-icon :icon="['fas', sendingToJD ? 'spinner' : 'cloud-download-alt']" :spin="sendingToJD" class="me-1" />
					{{ sendingToJD ? 'Sending…' : 'Send to JD' }}
				</button>
				<button class="btn btn-sm btn-outline-danger" @click="emit('delete-selected')">
					<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
					Delete
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
defineProps({
	selectionMode:  { type: Boolean, required: true },
	selectedCount:  { type: Number,  required: true },
	sendingToJD:    { type: Boolean, default: false },
})

const emit = defineEmits(['select-all', 'delete-selected', 'send-to-jd'])
</script>
