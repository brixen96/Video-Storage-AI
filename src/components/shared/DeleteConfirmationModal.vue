<template>
	<div v-if="visible" class="modal-overlay" @click="$emit('cancel')">
		<div class="modal-dialog" @click.stop>
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">{{ title }}</h5>
					<button class="btn-close-modal" @click="$emit('cancel')">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>
				<div class="modal-body">
					<p>
						{{ message }}
						<strong v-if="itemName">{{ itemName }}</strong
						>?
					</p>
					<p class="text-muted">{{ warningMessage }}</p>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="$emit('cancel')">{{ cancelText }}</button>
					<button :class="['btn', isDangerous ? 'btn-danger' : 'btn-primary']" @click="$emit('confirm')">
						<font-awesome-icon v-if="icon" :icon="icon" class="me-2" />
						{{ confirmText }}
					</button>
				</div>
			</div>
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
	title: {
		type: String,
		default: 'Confirm Action',
	},
	message: {
		type: String,
		default: 'Are you sure you want to proceed with',
	},
	itemName: {
		type: String,
		default: null,
	},
	warningMessage: {
		type: String,
		default: 'This action cannot be undone.',
	},
	confirmText: {
		type: String,
		default: 'Confirm',
	},
	cancelText: {
		type: String,
		default: 'Cancel',
	},
	isDangerous: {
		type: Boolean,
		default: true,
	},
	icon: {
		type: Array,
		default: () => ['fas', 'trash'],
	},
})

defineEmits(['confirm', 'cancel'])
</script>

<style scoped>
/* Modal uses styles from parent pages */
</style>
