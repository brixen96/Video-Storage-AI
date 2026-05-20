<template>
	<div class="context-menu" :style="{ top: y + 'px', left: x + 'px' }" @click.stop>
		<div class="context-menu-item" @click="$emit('action', 'rename')">
			<font-awesome-icon :icon="['fas', 'edit']" class="me-2" />
			Rename
		</div>
		<div class="context-menu-item" @click="$emit('action', 'copy-url')">
			<font-awesome-icon :icon="['fas', 'copy']" class="me-2" />
			Copy URL
		</div>
		<div class="context-menu-divider"></div>
		<div class="context-menu-item" @click="$emit('action', 'priority-highest')">
			<font-awesome-icon :icon="['fas', 'angle-double-up']" class="me-2 text-danger" />
			Priority: Highest
		</div>
		<div class="context-menu-item" @click="$emit('action', 'priority-high')">
			<font-awesome-icon :icon="['fas', 'angle-up']" class="me-2 text-warning" />
			Priority: High
		</div>
		<div class="context-menu-item" @click="$emit('action', 'priority-default')">
			<font-awesome-icon :icon="['fas', 'minus']" class="me-2" />
			Priority: Default
		</div>
		<div class="context-menu-item" @click="$emit('action', 'priority-low')">
			<font-awesome-icon :icon="['fas', 'angle-down']" class="me-2 text-secondary" />
			Priority: Low
		</div>
		<div class="context-menu-divider"></div>
		<div v-if="type.includes('lg')" class="context-menu-item" @click="$emit('action', 'move')">
			<font-awesome-icon :icon="['fas', 'arrow-right']" class="me-2 text-primary" />
			Move to Downloads
		</div>
		<div v-if="type === 'dl' || type === 'dl-link'" class="context-menu-item" @click="$emit('action', 'force')">
			<font-awesome-icon :icon="['fas', 'bolt']" class="me-2 text-warning" />
			Force Download
		</div>
		<div class="context-menu-item" @click="$emit('action', 'toggle-enabled')">
			<font-awesome-icon :icon="['fas', 'eye']" class="me-2" />
			{{ itemEnabled === false ? 'Enable' : 'Disable' }}
		</div>
		<div class="context-menu-divider"></div>
		<div class="context-menu-item text-danger" @click="$emit('action', 'remove')">
			<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
			Remove
		</div>
	</div>
</template>

<script setup>
defineProps({
	x: { type: Number, required: true },
	y: { type: Number, required: true },
	type: { type: String, default: '' },
	itemEnabled: { default: undefined },
})
defineEmits(['action'])
</script>

<style scoped>
.context-menu {
	position: fixed;
	z-index: 1060;
	background: #1e1e2e;
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-radius: 8px;
	min-width: 200px;
	box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
	padding: 0.5rem 0;
}

.context-menu-item {
	padding: 0.5rem 1rem;
	cursor: pointer;
	color: rgba(255, 255, 255, 0.9);
	font-size: 0.875rem;
	transition: background 0.15s;
}

.context-menu-item:hover {
	background: rgba(255, 255, 255, 0.1);
}

.context-menu-item.text-danger {
	color: #f44336;
}

.context-menu-divider {
	height: 1px;
	background: rgba(255, 255, 255, 0.1);
	margin: 0.5rem 0;
}
</style>
