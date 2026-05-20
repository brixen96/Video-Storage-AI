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
			<button v-if="item.type === 'video'" class="context-menu-item" @click="$emit('toggle-watched')">
				<font-awesome-icon :icon="['fas', 'eye']" class="me-2" />
				{{ item.watched ? 'Mark as Unwatched' : 'Mark as Watched' }}
			</button>
			<button v-if="item.type === 'video'" class="context-menu-item" @click="$emit('toggle-not-interested')">
				<font-awesome-icon :icon="['fas', item.not_interested ? 'check' : 'times-circle']" class="me-2" />
				{{ item.not_interested ? 'Remove from' : 'Mark as' }} Not Interested
			</button>
			<button v-if="item.type === 'video'" class="context-menu-item" @click="$emit('add-to-list')">
				<font-awesome-icon :icon="['fas', 'folder-plus']" class="me-2" />
				Add to List...
			</button>
			<!-- Remove from List Submenu -->
			<div v-if="item.type === 'video' && item.lists && item.lists.length > 0" class="context-menu-submenu">
				<button class="context-menu-item has-submenu">
					<font-awesome-icon :icon="['fas', 'folder-minus']" class="me-2" />
					Remove from List
					<font-awesome-icon :icon="['fas', 'chevron-right']" class="submenu-arrow" />
				</button>
				<div class="context-submenu">
					<button
						v-for="list in item.lists"
						:key="list.id"
						class="context-menu-item"
						@click="$emit('remove-from-list', list)"
					>
						<span class="list-color-dot me-2" :style="{ backgroundColor: list.color || '#6366f1' }"></span>
						{{ list.name }}
					</button>
				</div>
			</div>
			<div class="context-menu-divider"></div>
			<!-- Content Type Submenu -->
			<div class="context-menu-submenu">
				<button class="context-menu-item has-submenu">
					<font-awesome-icon :icon="['fas', 'tags']" class="me-2" />
					Set Content Type
					<font-awesome-icon :icon="['fas', 'chevron-right']" class="submenu-arrow" />
				</button>
				<div class="context-submenu">
					<button
						class="context-menu-item"
						:class="{ active: item.content_type === 'regular' || !item.content_type }"
						@click="$emit('set-content-type', 'regular')"
					>
						<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
						Regular
						<font-awesome-icon v-if="item.content_type === 'regular' || !item.content_type" :icon="['fas', 'check']" class="ms-auto" />
					</button>
					<button
						class="context-menu-item"
						:class="{ active: item.content_type === 'zoo' }"
						@click="$emit('set-content-type', 'zoo')"
					>
						<font-awesome-icon :icon="['fas', 'paw']" class="me-2" />
						Zoo
						<font-awesome-icon v-if="item.content_type === 'zoo'" :icon="['fas', 'check']" class="ms-auto" />
					</button>
					<button
						class="context-menu-item"
						:class="{ active: item.content_type === '3d' }"
						@click="$emit('set-content-type', '3d')"
					>
						<font-awesome-icon :icon="['fas', 'cube']" class="me-2" />
						3D
						<font-awesome-icon v-if="item.content_type === '3d'" :icon="['fas', 'check']" class="ms-auto" />
					</button>
				</div>
			</div>
			<div class="context-menu-divider"></div>
			<!-- Assign Performer (folder or video) -->
			<button class="context-menu-item" @click="$emit('assign-performer')">
				<font-awesome-icon :icon="['fas', 'user-tag']" class="me-2" />
				{{ item.type === 'folder' ? 'Assign Performer to Folder...' : 'Assign Performer...' }}
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

defineEmits(['play', 'open', 'toggle-watched', 'toggle-not-interested', 'add-to-list', 'remove-from-list', 'copy-path', 'close', 'set-content-type', 'assign-performer'])
</script>

<style>
/* Styles are in browser_page.css */
</style>
