<template>
	<span class="tag-badge" :style="{ backgroundColor: tag.color || '#6c757d' }" :class="{ clickable: clickable }" @click="handleClick">
		<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" class="me-1" />
		{{ tag.name }}
		<button v-if="removable" class="btn-remove" @click.stop="$emit('remove', tag.id)">
			<font-awesome-icon :icon="['fas', 'times']" />
		</button>
	</span>
</template>

<script>
export default {
	name: 'TagBadge',
	props: {
		tag: {
			type: Object,
			required: true,
		},
		clickable: {
			type: Boolean,
			default: false,
		},
		removable: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['click', 'remove'],
	methods: {
		handleClick() {
			if (this.clickable) {
				this.$emit('click', this.tag)
			}
		},
	},
}
</script>

<style scoped>
@import '@/styles/components/tag_badge.css';
</style>
