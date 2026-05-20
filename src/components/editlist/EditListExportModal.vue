<template>
	<div v-if="visible" class="modal-overlay" @click.self="emit('close')">
		<div class="list-modal">
			<div class="modal-header">
				<h4>Export List</h4>
				<button class="btn-close-modal" @click="emit('close')">
					<font-awesome-icon :icon="['fas', 'times']" />
				</button>
			</div>
			<div class="modal-body">
				<p>Export "{{ list?.name }}" to a file.</p>
				<div class="mb-3">
					<label class="form-label">Format</label>
					<select v-model="format" class="form-select">
						<option value="json">JSON (Full data)</option>
						<option value="csv">CSV (Spreadsheet)</option>
						<option value="txt">Text (Simple list)</option>
					</select>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" @click="emit('close')">Cancel</button>
				<button class="btn btn-primary" :disabled="exporting" @click="exportList">
					<font-awesome-icon v-if="exporting" :icon="['fas', 'spinner']" spin class="me-2" />
					<font-awesome-icon v-else :icon="['fas', 'download']" class="me-2" />
					Export
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, getCurrentInstance } from 'vue'

const props = defineProps({
	visible: { type: Boolean, default: false },
	list: { type: Object, default: null },
	items: { type: Array, default: () => [] },
})

const emit = defineEmits(['close'])

const { proxy } = getCurrentInstance()

const format = ref('json')
const exporting = ref(false)

const exportList = async () => {
	exporting.value = true
	try {
		const list = props.list
		const items = props.items
		let content = ''
		let filename = ''
		let mimeType = ''

		if (format.value === 'json') {
			const exportData = {
				list: { name: list.name, description: list.description, color: list.color, icon: list.icon },
				items: items.map(i => ({
					title: i.title, file_path: i.file_path, duration: i.duration,
					resolution: i.resolution, is_scraped: i.is_scraped, source: i.source,
					notes: i.notes, added_at: i.added_at,
				})),
				exported_at: new Date().toISOString(),
			}
			content = JSON.stringify(exportData, null, 2)
			filename = `${list.name.replace(/[^a-z0-9]/gi, '_')}.json`
			mimeType = 'application/json'
		} else if (format.value === 'csv') {
			const headers = ['Title', 'File Path', 'Duration', 'Resolution', 'Source', 'Notes', 'Added At']
			const rows = items.map(i => [
				`"${(i.title || '').replace(/"/g, '""')}"`,
				`"${(i.file_path || '').replace(/"/g, '""')}"`,
				i.duration || '',
				i.resolution || '',
				i.is_scraped ? i.source : 'Local',
				`"${(i.notes || '').replace(/"/g, '""')}"`,
				i.added_at || '',
			])
			content = [headers.join(','), ...rows.map(r => r.join(','))].join('\n')
			filename = `${list.name.replace(/[^a-z0-9]/gi, '_')}.csv`
			mimeType = 'text/csv'
		} else {
			content = `${list.name}\n${'='.repeat(list.name.length)}\n\n`
			content += items.map((i, idx) => `${idx + 1}. ${i.title}${i.notes ? ` - ${i.notes}` : ''}`).join('\n')
			filename = `${list.name.replace(/[^a-z0-9]/gi, '_')}.txt`
			mimeType = 'text/plain'
		}

		const blob = new Blob([content], { type: mimeType })
		const url = URL.createObjectURL(blob)
		const a = document.createElement('a')
		a.href = url
		a.download = filename
		document.body.appendChild(a)
		a.click()
		document.body.removeChild(a)
		URL.revokeObjectURL(url)

		proxy.$toast.success(`List exported as ${filename}`)
		emit('close')
	} catch (error) {
		console.error('Failed to export:', error)
		proxy.$toast.error('Failed to export list')
	} finally {
		exporting.value = false
	}
}
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

.list-modal .modal-body .form-select {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: white;
}

.list-modal .modal-body .form-select:focus {
	background: rgba(255, 255, 255, 0.08);
	border-color: #00d9ff;
	box-shadow: 0 0 0 0.25rem rgba(0, 217, 255, 0.25);
}

.list-modal .modal-body p {
	color: rgba(255, 255, 255, 0.9);
	margin-bottom: 0.5rem;
}

.list-modal .modal-footer {
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
	padding: 1.25rem 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
}
</style>
