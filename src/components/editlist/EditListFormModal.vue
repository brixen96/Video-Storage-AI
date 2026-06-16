<template>
	<div v-if="visible" class="modal-overlay" @click.self="handleClose">
		<div class="list-modal">
			<div class="modal-header">
				<h4>{{ isEditMode ? 'Edit List' : 'Create New List' }}</h4>
				<button class="btn-close-modal" @click="handleClose">
					<font-awesome-icon :icon="['fas', 'times']" />
				</button>
			</div>
			<div class="modal-body">
				<div class="mb-3">
					<label class="form-label">List Name *</label>
					<input v-model="form.name" type="text" class="form-control" placeholder="Enter list name" />
				</div>
				<div class="mb-3">
					<label class="form-label">Description</label>
					<textarea v-model="form.description" class="form-control" rows="2" placeholder="Optional description"></textarea>
				</div>
				<div class="mb-3">
					<label class="form-label">Color</label>
					<div class="color-picker">
						<div
							v-for="color in availableColors"
							:key="color"
							class="color-option"
							:class="{ selected: form.color === color }"
							:style="{ backgroundColor: color }"
							@click="form.color = color"
						></div>
					</div>
				</div>
				<div class="mb-3">
					<label class="form-label">Icon</label>
					<div class="icon-picker">
						<div
							v-for="icon in availableIcons"
							:key="icon"
							class="icon-option"
							:class="{ selected: form.icon === icon }"
							@click="form.icon = icon"
						>
							<font-awesome-icon :icon="['fas', icon]" />
						</div>
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" @click="handleClose">Cancel</button>
				<button class="btn btn-primary" :disabled="!form.name" @click="save">
					{{ isEditMode ? 'Save Changes' : 'Create List' }}
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, watch, getCurrentInstance } from 'vue'
import { API_ORIGIN } from '@/services/api'

const API_BASE = `${API_ORIGIN}/api/v1`

const props = defineProps({
	visible: { type: Boolean, default: false },
	list: { type: Object, default: null },
})

const emit = defineEmits(['close', 'saved'])

const { proxy } = getCurrentInstance()

const isEditMode = computed(() => props.list !== null)

const defaultForm = () => ({
	name: '',
	description: '',
	color: '#007bff',
	icon: 'list',
})

const form = ref(defaultForm())

const availableColors = [
	'#007bff', '#6c757d', '#28a745', '#dc3545', '#ffc107',
	'#17a2b8', '#6f42c1', '#e83e8c', '#fd7e14', '#20c997',
]

const availableIcons = [
	'list', 'star', 'heart', 'bookmark', 'folder',
	'film', 'video', 'play-circle', 'fire', 'clock',
	'eye', 'thumbs-up', 'trophy', 'gem', 'crown',
]

const handleClose = () => {
	emit('close')
}

const save = async () => {
	try {
		if (isEditMode.value) {
			const response = await fetch(`${API_BASE}/lists/${props.list.id}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(form.value),
			})
			const data = await response.json()
			if (data.success) {
				proxy.$toast.success('List updated')
				emit('saved', null)
			} else {
				throw new Error(data.message)
			}
		} else {
			const response = await fetch(`${API_BASE}/lists`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(form.value),
			})
			const data = await response.json()
			if (data.success) {
				proxy.$toast.success('List created')
				emit('saved', data.data?.id || null)
			} else {
				throw new Error(data.message)
			}
		}
	} catch (error) {
		console.error('Failed to save list:', error)
		proxy.$toast.error(error.message || 'Failed to save list')
	}
}

watch(() => props.visible, (newVal) => {
	if (newVal) {
		if (props.list) {
			form.value = {
				name: props.list.name,
				description: props.list.description || '',
				color: props.list.color || '#007bff',
				icon: props.list.icon || 'list',
			}
		} else {
			form.value = defaultForm()
		}
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

.list-modal .modal-body .form-control,
.list-modal .modal-body .form-select {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: white;
}

.list-modal .modal-body .form-control:focus,
.list-modal .modal-body .form-select:focus {
	background: rgba(255, 255, 255, 0.08);
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.25rem rgba(0, 217, 255, 0.25);
}

.list-modal .modal-body .form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.list-modal .modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
	padding: 1.25rem 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.color-picker {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.color-option {
	width: 32px;
	height: 32px;
	border-radius: 8px;
	cursor: pointer;
	border: 2px solid transparent;
	transition: all 0.2s;
}

.color-option:hover {
	transform: scale(1.1);
}

.color-option.selected {
	border-color: white;
	box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.3);
}

.icon-picker {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.icon-option {
	width: 40px;
	height: 40px;
	border-radius: 8px;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(255, 255, 255, 0.05);
	color: rgba(255, 255, 255, 0.7);
	cursor: pointer;
	border: 2px solid transparent;
	transition: all 0.2s;
}

.icon-option:hover {
	background: rgba(255, 255, 255, 0.1);
	color: white;
}

.icon-option.selected {
	border-color: #00d9ff;
	color: #00d9ff;
	background: rgba(0, 217, 255, 0.1);
}
</style>
