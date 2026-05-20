<template>
	<div class="modal-backdrop" @click.self="$emit('close')">
		<div class="modal-content add-links-modal">
			<div class="modal-header">
				<h5 class="modal-title">
					<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
					Add Links to JDownloader
				</h5>
				<button class="btn-close" @click="$emit('close')"></button>
			</div>
			<div class="modal-body">
				<div class="mb-3">
					<label class="form-label">Paste URLs (one per line)</label>
					<textarea
						v-model="newLinksText"
						class="form-control"
						rows="8"
						placeholder="https://example.com/video1.mp4&#10;https://example.com/video2.mp4&#10;..."
					></textarea>
				</div>
				<div class="mb-3" style="position: relative">
					<label class="form-label">Download Path</label>
					<div class="input-group">
						<span class="input-group-text">/output/</span>
						<input
							v-model="newLinksPath"
							type="text"
							class="form-control"
							placeholder="subfolder (leave empty for root)"
							@focus="showPathDropdown = true"
							@blur="hidePathDropdownDelayed"
						/>
						<button v-if="newLinksPath" class="btn btn-outline-secondary" @click="newLinksPath = ''" title="Clear">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
					<div v-if="showPathDropdown && downloadPathHistory.length > 0" class="path-history-dropdown">
						<div class="path-history-header">
							<span class="text-muted small">Recent paths</span>
							<button class="btn btn-link btn-sm text-danger p-0" @mousedown.prevent="clearPathHistory">clear</button>
						</div>
						<div v-for="path in filteredPathHistory" :key="path" class="path-history-item" @mousedown.prevent="selectPath(path)">
							<font-awesome-icon :icon="['fas', 'folder']" class="me-2 text-muted" />
							/output/{{ path }}
						</div>
					</div>
					<small class="text-muted">Downloads to /output/ by default. Add subfolder name to customize.</small>
				</div>
				<div class="mb-3">
					<label class="form-label">Package Name (optional, leave empty for auto)</label>
					<input v-model="newLinksPackageName" type="text" class="form-control" placeholder="Auto-detect from links" />
				</div>
				<div class="form-check">
					<input v-model="newLinksAutoStart" type="checkbox" class="form-check-input" id="addLinksAutoStart" />
					<label class="form-check-label" for="addLinksAutoStart">Auto-start downloads</label>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
				<button class="btn btn-primary" @click="addLinks" :disabled="!newLinksText.trim() || addingLinks">
					<font-awesome-icon v-if="addingLinks" :icon="['fas', 'spinner']" spin class="me-1" />
					<font-awesome-icon v-else :icon="['fas', 'plus']" class="me-1" />
					Add {{ linksCount }} Link{{ linksCount !== 1 ? 's' : '' }}
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { jdownloaderAPI } from '@/services/api'

const emit = defineEmits(['close', 'links-added', 'error'])

const newLinksText = ref('')
const newLinksPath = ref('')
const newLinksPackageName = ref('')
const newLinksAutoStart = ref(true)
const addingLinks = ref(false)

const STORAGE_KEY = 'jd-download-path-history'
const downloadPathHistory = ref(JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]'))
const showPathDropdown = ref(false)

const linksCount = computed(() => {
	if (!newLinksText.value.trim()) return 0
	return newLinksText.value.trim().split('\n').filter((l) => l.trim()).length
})

const filteredPathHistory = computed(() => {
	const query = (newLinksPath.value || '').toLowerCase()
	if (!query) return downloadPathHistory.value
	return downloadPathHistory.value.filter((p) => p.toLowerCase().includes(query))
})

const savePathToHistory = (path) => {
	if (!path) return
	const history = downloadPathHistory.value.filter((p) => p !== path)
	history.unshift(path)
	if (history.length > 20) history.length = 20
	downloadPathHistory.value = history
	localStorage.setItem(STORAGE_KEY, JSON.stringify(history))
}

const selectPath = (path) => {
	newLinksPath.value = path
	showPathDropdown.value = false
}

const clearPathHistory = () => {
	downloadPathHistory.value = []
	localStorage.removeItem(STORAGE_KEY)
	showPathDropdown.value = false
}

const hidePathDropdownDelayed = () => {
	setTimeout(() => {
		showPathDropdown.value = false
	}, 150)
}

const addLinks = async () => {
	if (addingLinks.value) return
	addingLinks.value = true
	try {
		const urls = newLinksText.value.trim().split('\n').filter((l) => l.trim())
		const downloadPath = newLinksPath.value.trim() ? `/output/${newLinksPath.value.trim().replace(/^\/+/, '')}` : '/output'
		const links = urls.map((url) => ({
			url: url.trim(),
			packageName: newLinksPackageName.value || undefined,
			downloadPath,
			autoStart: newLinksAutoStart.value,
		}))
		await jdownloaderAPI.sendLinks(links)
		savePathToHistory(newLinksPath.value.trim())
		newLinksText.value = ''
		newLinksPath.value = ''
		newLinksPackageName.value = ''
		emit('links-added', links.length)
		emit('close')
	} catch (error) {
		emit('error', 'Failed to add links: ' + error.message)
	} finally {
		addingLinks.value = false
	}
}
</script>

<style scoped>
.modal-backdrop {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1050;
}

.modal-content {
	background: #1a1a2e;
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	width: 90%;
	max-width: 600px;
	max-height: 90vh;
	overflow: auto;
}

.modal-header {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.modal-title {
	margin: 0;
	color: #fff;
}

.btn-close {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	font-size: 1.5rem;
	cursor: pointer;
	padding: 0;
	line-height: 1;
}

.btn-close:hover {
	color: #fff;
}

.btn-close::before {
	content: '×';
}

.modal-body {
	padding: 1.25rem;
}

.modal-footer {
	padding: 1rem 1.25rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: flex-end;
	gap: 0.5rem;
}

.form-label {
	color: rgba(255, 255, 255, 0.8);
}

.form-control {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: #fff;
}

.form-control:focus {
	background: rgba(255, 255, 255, 0.15);
	border-color: #2196f3;
	color: #fff;
	box-shadow: 0 0 0 0.25rem rgba(33, 150, 243, 0.25);
}

.form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.form-check-label {
	color: rgba(255, 255, 255, 0.8);
}

.path-history-dropdown {
	position: absolute;
	top: 100%;
	left: 0;
	right: 0;
	z-index: 1050;
	background: #1e1e2e;
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-top: none;
	border-radius: 0 0 6px 6px;
	max-height: 200px;
	overflow-y: auto;
}

.path-history-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.4rem 0.75rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.path-history-item {
	padding: 0.4rem 0.75rem;
	cursor: pointer;
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.8);
	transition: background 0.15s;
}

.path-history-item:hover {
	background: rgba(255, 255, 255, 0.1);
}
</style>
