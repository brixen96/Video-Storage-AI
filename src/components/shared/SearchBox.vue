<template>
	<div class="search-box">
		<font-awesome-icon :icon="['fas', 'search']" class="search-icon" />
		<input
			:value="modelValue"
			type="text"
			class="form-control"
			:placeholder="placeholder"
			@input="handleInput"
		/>
		<button v-if="modelValue && clearable" class="btn-clear-search" @click="handleClear">
			<font-awesome-icon :icon="['fas', 'times-circle']" />
		</button>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
/* eslint-disable no-unused-vars */
const props = defineProps({
	// v-model binding for search query
	modelValue: {
		type: String,
		default: '',
	},
	// Placeholder text
	placeholder: {
		type: String,
		default: 'Search...',
	},
	// Show clear button
	clearable: {
		type: Boolean,
		default: true,
	},
})

const emit = defineEmits(['update:modelValue', 'input', 'clear'])

const handleInput = (event) => {
	const value = event.target.value
	emit('update:modelValue', value)
	emit('input', event)
}

const handleClear = () => {
	emit('update:modelValue', '')
	emit('clear')
}
</script>

<style scoped>
.search-box {
	position: relative;
	display: flex;
	align-items: center;
}

.search-icon {
	position: absolute;
	left: 12px;
	color: rgba(255, 255, 255, 0.5);
	pointer-events: none;
	z-index: 1;
}

.search-box .form-control {
	padding-left: 38px;
	padding-right: 38px;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: #fff;
	transition: all 0.3s ease;
}

.search-box .form-control:focus {
	background: rgba(255, 255, 255, 0.08);
	border-color: rgba(0, 217, 255, 0.5);
	box-shadow: 0 0 0 0.2rem rgba(0, 217, 255, 0.15);
	color: #fff;
}

.search-box .form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.btn-clear-search {
	position: absolute;
	right: 8px;
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.5);
	cursor: pointer;
	padding: 4px 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	transition: color 0.2s ease;
	z-index: 1;
}

.btn-clear-search:hover {
	color: rgba(255, 255, 255, 0.9);
}

.btn-clear-search:focus {
	outline: none;
}
</style>
