<template>
	<div :class="['loading-state', `py-${padding}`]" :style="{ textAlign: align }">
		<div v-if="spinner === 'bootstrap'" class="spinner-border text-primary" role="status">
			<span class="visually-hidden">{{ loadingText }}</span>
		</div>
		<font-awesome-icon v-else-if="spinner === 'fontawesome'" :icon="['fas', 'spinner']" spin :size="iconSize" class="text-primary" />
		<p v-if="showText" :class="['mt-3', textClass]">{{ loadingText }}</p>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
defineProps({
	// Loading message text
	loadingText: {
		type: String,
		default: 'Loading...',
	},
	// Show text below spinner
	showText: {
		type: Boolean,
		default: false,
	},
	// Spinner type: 'bootstrap' or 'fontawesome'
	spinner: {
		type: String,
		default: 'bootstrap',
		validator: (value) => ['bootstrap', 'fontawesome'].includes(value),
	},
	// Icon size for FontAwesome spinner
	iconSize: {
		type: String,
		default: '3x',
	},
	// Text alignment
	align: {
		type: String,
		default: 'center',
	},
	// Vertical padding (Bootstrap spacing: 0-5)
	padding: {
		type: Number,
		default: 5,
		validator: (value) => value >= 0 && value <= 5,
	},
	// Additional text classes
	textClass: {
		type: String,
		default: '',
	},
})
</script>

<style scoped>
.loading-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
}
</style>
