<template>
	<div :class="['empty-state', `py-${padding}`, `text-${align}`]">
		<font-awesome-icon v-if="icon" :icon="icon" :size="iconSize" :class="['mb-3', iconClass]" />
		<h3 v-if="title">{{ title }}</h3>
		<p :class="['text-muted', messageClass]">{{ message }}</p>
		<slot name="actions"></slot>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
defineProps({
	// Icon to display (FontAwesome icon array)
	icon: {
		type: Array,
		default: null,
	},
	// Icon size
	iconSize: {
		type: String,
		default: '3x',
	},
	// Icon color class
	iconClass: {
		type: String,
		default: 'text-muted',
	},
	// Title text
	title: {
		type: String,
		default: '',
	},
	// Message text
	message: {
		type: String,
		default: 'No items found',
	},
	// Message additional classes
	messageClass: {
		type: String,
		default: '',
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
})
</script>

<style scoped>
.empty-state {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
}
</style>
