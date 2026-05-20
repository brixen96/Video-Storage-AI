<template>
	<div v-if="visible" class="modal-overlay" @click.self="handleClose">
		<div class="list-modal">
			<div class="modal-header">
				<h4>Edit Notes</h4>
				<button class="btn-close-modal" @click="handleClose">
					<font-awesome-icon :icon="['fas', 'times']" />
				</button>
			</div>
			<div class="modal-body">
				<p class="text-muted mb-2">{{ item?.title }}</p>
				<textarea
					v-model="notes"
					class="form-control"
					rows="4"
					placeholder="Add notes about this video..."
				></textarea>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" @click="handleClose">Cancel</button>
				<button class="btn btn-primary" :disabled="saving" @click="save">
					<font-awesome-icon v-if="saving" :icon="['fas', 'spinner']" spin class="me-2" />
					Save Notes
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, watch, getCurrentInstance } from 'vue'

const API_BASE = 'http://localhost:8080/api/v1'

const props = defineProps({
	visible: { type: Boolean, default: false },
	item: { type: Object, default: null },
	listId: { type: Number, default: null },
})

const emit = defineEmits(['close', 'saved'])

const { proxy } = getCurrentInstance()

const notes = ref('')
const saving = ref(false)

const handleClose = () => {
	emit('close')
}

const save = async () => {
	saving.value = true
	try {
		const response = await fetch(`${API_BASE}/lists/${props.listId}/items/${props.item.id}/notes`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ notes: notes.value }),
		})
		const data = await response.json()
		if (data.success) {
			proxy.$toast.success('Notes saved')
			emit('saved', props.item.id, notes.value)
		} else {
			throw new Error(data.message)
		}
	} catch (error) {
		console.error('Failed to save notes:', error)
		proxy.$toast.error(error.message || 'Failed to save notes')
	} finally {
		saving.value = false
	}
}

watch(() => props.visible, (newVal) => {
	if (newVal) {
		notes.value = props.item?.notes || ''
	}
})
</script>

<style scoped>
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.8);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
}

.list-modal {
	background: #1a1a2e;
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 16px;
	width: 100%;
	max-width: 500px;
	overflow: hidden;
}

.list-modal .modal-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1.25rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.list-modal .modal-header h4 {
	margin: 0;
	color: white;
	font-weight: 600;
}

.btn-close-modal {
	background: rgba(255, 255, 255, 0.1);
	border: none;
	color: rgba(255, 255, 255, 0.7);
	width: 32px;
	height: 32px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	cursor: pointer;
	transition: all 0.2s;
}

.btn-close-modal:hover {
	background: rgba(255, 255, 255, 0.2);
	color: white;
}

.list-modal .modal-body {
	padding: 1.5rem;
}

.list-modal .modal-body .form-label {
	color: rgba(255, 255, 255, 0.9);
	font-weight: 500;
	margin-bottom: 0.5rem;
}

.list-modal .modal-body .form-control {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: white;
}

.list-modal .modal-body .form-control:focus {
	background: rgba(255, 255, 255, 0.08);
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.25rem rgba(0, 217, 255, 0.25);
}

.list-modal .modal-body .form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.list-modal .modal-body p {
	color: rgba(255, 255, 255, 0.9);
	margin-bottom: 0.5rem;
}

.list-modal .modal-body .text-muted {
	color: rgba(255, 255, 255, 0.5) !important;
	font-size: 0.875rem;
}

.list-modal .modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
	padding: 1.25rem 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}
</style>
