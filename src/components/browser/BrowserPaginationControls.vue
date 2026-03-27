<template>
	<div v-if="totalItems > 0" class="pagination-controls">
		<div class="pagination-info">
			Showing {{ startItem }}&ndash;{{ endItem }} of {{ totalItems }} items
		</div>
		<div v-if="totalPages > 1" class="pagination-buttons">
			<button class="btn btn-sm btn-outline-primary" :disabled="currentPage === 1" @click="goToFirstPage" title="First page">
				<font-awesome-icon :icon="['fas', 'chevron-left']" />
				<font-awesome-icon :icon="['fas', 'chevron-left']" />
			</button>
			<button class="btn btn-sm btn-outline-primary" :disabled="currentPage === 1" @click="previousPage" title="Previous page">
				<font-awesome-icon :icon="['fas', 'chevron-left']" />
			</button>
			<span class="pagination-current">Page {{ currentPage }} of {{ totalPages }}</span>
			<button class="btn btn-sm btn-outline-primary" :disabled="currentPage === totalPages" @click="nextPage" title="Next page">
				<font-awesome-icon :icon="['fas', 'chevron-right']" />
			</button>
			<button class="btn btn-sm btn-outline-primary" :disabled="currentPage === totalPages" @click="goToLastPage" title="Last page">
				<font-awesome-icon :icon="['fas', 'chevron-right']" />
				<font-awesome-icon :icon="['fas', 'chevron-right']" />
			</button>
			<!-- Go-to-page -->
			<form class="pagination-goto" @submit.prevent="applyGoTo">
				<input
					v-model.number="goToPage"
					type="number"
					class="form-control form-control-sm bg-dark text-white border-secondary"
					:min="1"
					:max="totalPages"
					placeholder="Go to"
					title="Go to page"
					style="width: 70px"
				/>
				<button type="submit" class="btn btn-sm btn-outline-secondary" title="Go">
					<font-awesome-icon :icon="['fas', 'arrow-right']" />
				</button>
			</form>
		</div>
		<div class="pagination-size">
			<select :value="itemsPerPage" class="form-select form-select-sm bg-dark text-white border-secondary" @change="handleItemsPerPageChange">
				<option :value="24">24 per page</option>
				<option :value="48">48 per page</option>
				<option :value="60">60 per page</option>
				<option :value="100">100 per page</option>
				<option :value="200">200 per page</option>
				<option :value="500">500 per page</option>
			</select>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
import { computed, ref } from 'vue'

const props = defineProps({
	currentPage: {
		type: Number,
		required: true,
	},
	itemsPerPage: {
		type: Number,
		required: true,
	},
	totalItems: {
		type: Number,
		required: true,
	},
})

const emit = defineEmits(['update:currentPage', 'update:itemsPerPage'])

const goToPage = ref(null)

// Computed properties
const totalPages = computed(() => {
	return Math.ceil(props.totalItems / props.itemsPerPage)
})

const startItem = computed(() => {
	return (props.currentPage - 1) * props.itemsPerPage + 1
})

const endItem = computed(() => {
	return Math.min(props.currentPage * props.itemsPerPage, props.totalItems)
})

// Navigation methods
const goToFirstPage = () => {
	emit('update:currentPage', 1)
}

const previousPage = () => {
	if (props.currentPage > 1) {
		emit('update:currentPage', props.currentPage - 1)
	}
}

const nextPage = () => {
	if (props.currentPage < totalPages.value) {
		emit('update:currentPage', props.currentPage + 1)
	}
}

const goToLastPage = () => {
	emit('update:currentPage', totalPages.value)
}

const applyGoTo = () => {
	const page = parseInt(goToPage.value)
	if (page >= 1 && page <= totalPages.value) {
		emit('update:currentPage', page)
		goToPage.value = null
	}
}

const handleItemsPerPageChange = (event) => {
	const newItemsPerPage = parseInt(event.target.value)
	emit('update:itemsPerPage', newItemsPerPage)
	emit('update:currentPage', 1)
}
</script>

<style scoped>
.pagination-goto {
	display: flex;
	align-items: center;
	gap: 0.25rem;
}
</style>
