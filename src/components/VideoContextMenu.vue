<template>
	<div v-if="show" class="vp-context-menu" :style="{ top: y + 'px', left: x + 'px' }">
		<!-- Scraped video options -->
		<template v-if="video && video.is_scraped">
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'play', video })">
				<font-awesome-icon :icon="['fas', 'play']" />
				Watch
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'firefox', video })">
				<font-awesome-icon :icon="['fab', 'firefox-browser']" />
				Open in Firefox
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'download', video })">
				<font-awesome-icon :icon="['fas', 'cloud-download-alt']" />
				Download
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'copy-url', video })">
				<font-awesome-icon :icon="['fas', 'copy']" />
				Copy URL
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'add-to-list', video })">
				<font-awesome-icon :icon="['fas', 'folder-plus']" />
				Add to List...
			</div>
			<div class="vp-context-menu-item text-muted" style="font-size: 0.8em; pointer-events: none;">
				<font-awesome-icon :icon="['fas', 'globe']" />
				Source: {{ video.source }}
			</div>
		</template>
		<!-- Regular video options -->
		<template v-else>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'play', video })">
				<font-awesome-icon :icon="['fas', 'play']" />
				Play
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'favorite', video }); $emit('hide')">
				<font-awesome-icon :icon="[video?.is_favorite ? 'fas' : 'far', 'heart']" />
				{{ video?.is_favorite ? 'Unfavorite' : 'Favorite' }}
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'edit', video })">
				<font-awesome-icon :icon="['fas', 'edit']" />
				Edit Metadata
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'fetch-metadata', video })">
				<font-awesome-icon :icon="['fas', 'download']" />
				Fetch Metadata
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'tag', video })">
				<font-awesome-icon :icon="['fas', 'tag']" />
				Add Tags
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'add-to-list', video })">
				<font-awesome-icon :icon="['fas', 'folder-plus']" />
				Add to List...
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'explorer', video }); $emit('hide')">
				<font-awesome-icon :icon="['fas', 'folder-open']" />
				Open in Explorer
			</div>
			<div class="vp-context-menu-item" @click="$emit('action', { type: 'analyze-path', video })">
				<font-awesome-icon :icon="['fas', 'user-slash']" />
				Analyze Path
			</div>
			<div
				v-if="video?.linked_video_id"
				class="vp-context-menu-item text-info"
				@click="$emit('action', { type: 'open-linked', video })"
			>
				<font-awesome-icon :icon="['fas', 'link']" />
				Open Linked Video
			</div>
			<!-- Missing file actions -->
			<template v-if="video?.is_missing">
				<div class="vp-context-menu-divider"></div>
				<div class="vp-context-menu-item text-warning" @click="$emit('action', { type: 'locate', video })">
					<font-awesome-icon :icon="['fas', 'search-location']" />
					Locate File...
				</div>
				<div class="vp-context-menu-item text-warning" @click="$emit('action', { type: 'remap', video })">
					<font-awesome-icon :icon="['fas', 'route']" />
					Remap Path...
				</div>
				<div class="vp-context-menu-item danger" @click="$emit('action', { type: 'delete', video })">
					<font-awesome-icon :icon="['fas', 'trash']" />
					Delete Record
				</div>
			</template>
			<div class="vp-context-menu-divider"></div>
			<div class="vp-context-menu-rate">
				<span class="rate-label">Rate</span>
				<font-awesome-icon
					v-for="star in 5"
					:key="star"
					:icon="[star <= (hoverRating || video?.rating || 0) ? 'fas' : 'far', 'star']"
					class="ctx-star"
					:class="{ 'ctx-star-filled': star <= (hoverRating || video?.rating || 0) }"
					@mouseenter="hoverRating = star"
					@mouseleave="hoverRating = 0"
					@click.stop="$emit('action', { type: 'rate', video, star })"
				/>
			</div>
			<div class="vp-context-menu-item danger" @click="$emit('action', { type: 'delete', video })">
				<font-awesome-icon :icon="['fas', 'trash']" />
				Delete
			</div>
		</template>
	</div>
</template>

<script>
export default {
	name: 'VideoContextMenu',
	props: {
		show: { type: Boolean, default: false },
		x: { type: Number, default: 0 },
		y: { type: Number, default: 0 },
		video: { type: Object, default: null },
	},
	emits: ['action', 'hide'],
	data() {
		return {
			hoverRating: 0,
		}
	},
}
</script>
