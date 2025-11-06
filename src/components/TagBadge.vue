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
.tag-badge {
	display: inline-flex;
	align-items: center;
	gap: 0.25rem;
	padding: 0.375rem 0.75rem;
	border-radius: 0.5rem;
	color: #fff;
	font-size: 0.875rem;
	font-weight: 500;
	white-space: nowrap;
	transition: all 0.2s;
	margin: 0.25rem;
}

.tag-badge.clickable {
	cursor: pointer;
}

.tag-badge.clickable:hover {
	transform: translateY(-2px);
	box-shadow: 0 4px 8px rgba(0, 0, 0, 0.3);
}

.btn-remove {
	background: none;
	border: none;
	color: #fff;
	padding: 0;
	margin-left: 0.25rem;
	cursor: pointer;
	opacity: 0.7;
	transition: opacity 0.2s;
}

.btn-remove:hover {
	opacity: 1;
}
</style>
