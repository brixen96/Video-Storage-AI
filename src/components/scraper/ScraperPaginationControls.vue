<template>
	<div v-if="totalPages > 1" class="pagination-controls p-3">
		<div class="d-flex justify-content-center align-items-center flex-column gap-2">
			<div class="btn-group">
				<button
					class="btn btn-sm btn-outline-secondary"
					:disabled="currentPage <= 1"
					@click="emit('update:currentPage', currentPage - 1); scrollTop()"
				>
					<font-awesome-icon :icon="['fas', 'chevron-left']" />
				</button>
				<button class="btn btn-sm btn-outline-secondary disabled">
					Page {{ currentPage }} / {{ totalPages }}
				</button>
				<button
					class="btn btn-sm btn-outline-secondary"
					:disabled="currentPage >= totalPages"
					@click="emit('update:currentPage', currentPage + 1); scrollTop()"
				>
					<font-awesome-icon :icon="['fas', 'chevron-right']" />
				</button>
			</div>
			<span class="text-muted small">
				Showing {{ rangeStart }} – {{ rangeEnd }} of {{ totalItems }} videos
			</span>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
import { computed } from 'vue'

const props = defineProps({
	currentPage: { type: Number, required: true },
	totalPages:  { type: Number, required: true },
	totalItems:  { type: Number, required: true },
	pageSize:    { type: Number, default: 24    },
})

const emit = defineEmits(['update:currentPage'])

const rangeStart = computed(() => (props.currentPage - 1) * props.pageSize + 1)
const rangeEnd   = computed(() => Math.min(props.currentPage * props.pageSize, props.totalItems))

const scrollTop = () => window.scrollTo(0, 0)
</script>
