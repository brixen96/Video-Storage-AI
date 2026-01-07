<template>
	<div class="stat-card" :class="cardClass">
		<div v-if="icon" class="stat-icon" :class="iconClass">
			<font-awesome-icon :icon="icon" :size="iconSize" />
		</div>
		<div class="stat-content">
			<div class="stat-value" :class="valueClass">{{ formattedValue }}</div>
			<div class="stat-label" :class="labelClass">{{ label }}</div>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
import { computed } from 'vue'

const props = defineProps({
	// Stat value (number or string)
	value: {
		type: [Number, String],
		default: 0,
	},
	// Stat label/description
	label: {
		type: String,
		required: true,
	},
	// Icon (FontAwesome icon array)
	icon: {
		type: Array,
		default: null,
	},
	// Icon size
	iconSize: {
		type: String,
		default: '1x',
	},
	// Icon color class (e.g., 'threads', 'posts', 'primary', 'success')
	iconClass: {
		type: String,
		default: '',
	},
	// Additional card classes
	cardClass: {
		type: String,
		default: '',
	},
	// Additional value classes
	valueClass: {
		type: String,
		default: '',
	},
	// Additional label classes
	labelClass: {
		type: String,
		default: '',
	},
	// Auto-format number with thousands separators
	formatNumber: {
		type: Boolean,
		default: false,
	},
})

const formattedValue = computed(() => {
	if (props.formatNumber && typeof props.value === 'number') {
		return props.value.toLocaleString()
	}
	return props.value
})
</script>

<style scoped>
.stat-card {
	display: flex;
	align-items: center;
	padding: 1.5rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
	transition: transform 0.2s, box-shadow 0.2s;
}

.stat-card:hover {
	transform: translateY(-2px);
	box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
	background: rgba(255, 255, 255, 0.08);
}

.stat-icon {
	display: flex;
	align-items: center;
	justify-content: center;
	margin-right: 1rem;
	flex-shrink: 0;
}

/* Icon color variations */
.stat-icon.threads { color: #60a5fa; }
.stat-icon.posts { color: #34d399; }
.stat-icon.links { color: #fbbf24; }
.stat-icon.active { color: #4ade80; }
.stat-icon.primary { color: #60a5fa; }
.stat-icon.success { color: #4ade80; }
.stat-icon.warning { color: #fbbf24; }
.stat-icon.info { color: #22d3ee; }

.stat-content {
	flex: 1;
	min-width: 0;
}

.stat-value {
	font-size: 1.75rem;
	font-weight: 700;
	line-height: 1.2;
	margin-bottom: 0.25rem;
	color: #f8f9fa;
}

.stat-label {
	font-size: 0.875rem;
	color: #adb5bd;
	font-weight: 500;
}
</style>
