<template>
	<div class="category-selector">
		<div :class="['category-option', { active: modelValue === 'regular' }]" @click="selectCategory('regular')">
			<font-awesome-icon :icon="['fas', 'user']" :size="iconSize" />
			<span>Regular</span>
		</div>
		<div :class="['category-option', { active: modelValue === 'zoo' }]" @click="selectCategory('zoo')">
			<font-awesome-icon :icon="['fas', 'dog']" :size="iconSize" />
			<span>Zoo</span>
		</div>
		<div :class="['category-option', { active: modelValue === '3d' }]" @click="selectCategory('3d')">
			<font-awesome-icon :icon="['fas', 'cube']" :size="iconSize" />
			<span>3D</span>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
/* eslint-disable no-unused-vars */
const props = defineProps({
	// v-model binding for selected category
	modelValue: {
		type: String,
		default: '',
	},
	// Icon size
	iconSize: {
		type: String,
		default: '2x',
	},
})

const emit = defineEmits(['update:modelValue'])

const selectCategory = (category) => {
	emit('update:modelValue', category)
}
</script>

<style scoped>
.category-selector {
	display: flex;
	gap: 1rem;
	justify-content: space-evenly;
	flex-wrap: wrap;
}

.category-option {
	flex: 1;
	min-width: 120px;
	padding: 1.5rem 1rem;
	border: 2px solid rgba(255, 255, 255, 0.2);
	border-radius: 8px;
	background: rgba(255, 255, 255, 0.05);
	cursor: pointer;
	transition: all 0.3s ease;
	text-align: center;
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 0.75rem;
}

.category-option:hover {
	transform: translateY(-2px);
	border-color: rgba(0, 217, 255, 0.5);
	background: rgba(255, 255, 255, 0.1);
}

.category-option.active {
	border-color: #00d9ff;
	background: rgba(0, 217, 255, 0.2);
	box-shadow: 0 0 20px rgba(0, 217, 255, 0.3);
}

.category-option span {
	font-weight: 600;
	font-size: 0.9rem;
	color: rgba(255, 255, 255, 0.9);
}

.category-option svg {
	color: #00d9ff;
}
</style>
