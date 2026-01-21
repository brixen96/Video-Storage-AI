<template>
	<div
		v-if="visible"
		class="context-menu"
		:style="{
			top: `${y}px`,
			left: `${x}px`,
		}"
		@click.stop
	>
		<div v-if="item" class="context-menu-items">
			<button v-if="item.type === 'video'" class="context-menu-item" @click="$emit('play')">
				<font-awesome-icon :icon="['fas', 'play']" class="me-2" />
				Play Video
			</button>
			<button v-if="item.type === 'folder'" class="context-menu-item" @click="$emit('open')">
				<font-awesome-icon :icon="['fas', 'folder-open']" class="me-2" />
				Open Folder
			</button>
			<div class="context-menu-divider"></div>
			<button v-if="item.type === 'video'" class="context-menu-item" @click="$emit('toggle-not-interested')">
				<font-awesome-icon :icon="['fas', item.not_interested ? 'check' : 'times-circle']" class="me-2" />
				{{ item.not_interested ? 'Remove from' : 'Mark as' }} Not Interested
			</button>
			<button v-if="item.type === 'video'" class="context-menu-item" @click="$emit('toggle-edit-list')">
				<font-awesome-icon :icon="['fas', item.in_edit_list ? 'check' : 'list']" class="me-2" />
				{{ item.in_edit_list ? 'Remove from' : 'Add to' }} Edit List
			</button>
			<div class="context-menu-divider"></div>
			<button class="context-menu-item" @click="$emit('copy-path')">
				<font-awesome-icon :icon="['fas', 'copy']" class="me-2" />
				Copy Path
			</button>
		</div>
	</div>
</template>

<script setup>
/* eslint-disable no-undef */
defineProps({
	visible: {
		type: Boolean,
		default: false,
	},
	x: {
		type: Number,
		default: 0,
	},
	y: {
		type: Number,
		default: 0,
	},
	item: {
		type: Object,
		default: null,
	},
})

defineEmits(['play', 'open', 'toggle-not-interested', 'toggle-edit-list', 'copy-path', 'close'])
</script>

<style scoped>
/* Component uses styles from parent browser_page.css */
</style>
