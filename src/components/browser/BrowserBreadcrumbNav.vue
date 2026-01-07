<template>
	<div>
		<!-- Breadcrumb Navigation -->
		<nav v-if="pathSegments.length > 0" aria-label="breadcrumb" class="mb-2">
			<ol class="breadcrumb bg-dark border border-secondary rounded p-2 mb-0">
				<li class="breadcrumb-item">
					<button class="btn btn-sm btn-link text-white p-0" @click="$emit('back')" :disabled="pathSegments.length === 0" title="Go back">
						<font-awesome-icon :icon="['fas', 'arrow-left']" class="me-2" />
					</button>
				</li>
				<li class="breadcrumb-item">
					<button class="btn btn-sm btn-link text-decoration-none p-0" @click="$emit('navigate-to', -1)">
						<font-awesome-icon :icon="['fas', 'home']" class="me-1" />
						Root
					</button>
				</li>
				<li v-for="(segment, index) in pathSegments" :key="index" class="breadcrumb-item" :class="{ active: index === pathSegments.length - 1 }">
					<button
						v-if="index < pathSegments.length - 1"
						class="btn btn-sm btn-link text-decoration-none p-0"
						@click="$emit('navigate-to', index)"
					>
						{{ segment }}
					</button>
					<span v-else class="text-light">{{ segment }}</span>
				</li>
			</ol>
		</nav>

		<!-- Filter Alert -->
		<div v-if="showFilterAlert" class="alert alert-info alert-sm mb-2 py-1 px-2" role="alert">
			<small>
				<font-awesome-icon :icon="['fas', 'info-circle']" class="me-1" />
				{{ filterAlertMessage }}
			</small>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
import { computed } from 'vue'

const props = defineProps({
	pathSegments: {
		type: Array,
		default: () => [],
	},
	showNotInterested: {
		type: Boolean,
		default: false,
	},
	showEditList: {
		type: Boolean,
		default: false,
	},
})

// eslint-disable-next-line no-unused-vars
const emit = defineEmits(['navigate-to', 'back'])

// Computed properties
const showFilterAlert = computed(() => {
	return props.showNotInterested || props.showEditList
})

const filterAlertMessage = computed(() => {
	if (props.showNotInterested && props.showEditList) {
		return 'Showing "Not Interested" and "Edit List" videos'
	} else if (props.showNotInterested) {
		return 'Showing "Not Interested" videos'
	} else if (props.showEditList) {
		return 'Showing "Edit List" videos'
	}
	return ''
})
</script>

<style scoped>
/* Component uses styles from parent browser_page.css */
</style>
