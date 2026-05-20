<template>
	<div class="jd-modal-backdrop" @click.self="$emit('close')">
		<div class="jd-modal-content">
			<div class="jd-modal-header">
				<h5 class="jd-modal-title">
					<font-awesome-icon :icon="['fas', 'cloud-download-alt']" class="me-2" />
					Send to JDownloader
				</h5>
				<button class="jd-btn-close" @click="$emit('close')">&times;</button>
			</div>
			<div class="jd-modal-body">
				<div v-if="jdDedupInfo" class="jd-dedup-info">
					<font-awesome-icon :icon="['fas', 'info-circle']" />
					<span>{{ jdDedupInfo }}</span>
				</div>

				<!-- Provider Priority Config -->
				<div class="priority-config-section">
					<button class="priority-toggle-btn" @click="showPriorityConfig = !showPriorityConfig">
						<font-awesome-icon :icon="['fas', 'sort-amount-down']" class="me-1" />
						Provider Priority
						<font-awesome-icon :icon="['fas', showPriorityConfig ? 'chevron-up' : 'chevron-down']" class="ms-auto" />
					</button>
					<div v-if="showPriorityConfig" class="priority-list">
						<div v-for="(provider, idx) in priorityList" :key="provider" class="priority-item">
							<span class="priority-rank">{{ idx + 1 }}</span>
							<span class="priority-name">{{ provider }}</span>
							<div class="priority-actions">
								<button @click="movePriorityUp(provider)" :disabled="idx === 0" class="priority-btn" title="Move up">
									<font-awesome-icon :icon="['fas', 'chevron-up']" />
								</button>
								<button @click="movePriorityDown(provider)" :disabled="idx === priorityList.length - 1" class="priority-btn" title="Move down">
									<font-awesome-icon :icon="['fas', 'chevron-down']" />
								</button>
							</div>
						</div>
						<button class="priority-reset-btn" @click="resetPriority">
							<font-awesome-icon :icon="['fas', 'undo']" class="me-1" />Reset to defaults
						</button>
					</div>
				</div>

				<div class="jd-form-group">
					<label class="jd-form-label">Links ({{ jdLinksCount }} link{{ jdLinksCount !== 1 ? 's' : '' }})</label>
					<textarea v-model="jdLinksText" class="jd-form-control jd-textarea" rows="10" placeholder="No active links found..."></textarea>
				</div>

				<div class="jd-form-group">
					<label class="jd-form-label">Package Name</label>
					<input v-model="jdPackageName" type="text" class="jd-form-control" placeholder="Package name for JDownloader" />
				</div>

				<div class="jd-form-group">
					<label class="jd-form-label">Download Path</label>
					<div class="jd-input-group">
						<span class="jd-input-prefix">/output/</span>
						<input v-model="jdDownloadPath" type="text" class="jd-form-control jd-input-with-prefix" placeholder="subfolder (leave empty for root)" />
					</div>
				</div>

				<div class="jd-form-check">
					<input v-model="jdAutoStart" type="checkbox" class="jd-check-input" id="jdAutoStart" />
					<label class="jd-check-label" for="jdAutoStart">Auto-start downloads</label>
				</div>
			</div>
			<div class="jd-modal-footer">
				<button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
				<button class="btn btn-jdownloader" @click="confirmSendToJD" :disabled="!jdLinksText.trim() || sendingToJD">
					<font-awesome-icon v-if="sendingToJD" :icon="['fas', 'spinner']" spin class="me-1" />
					<font-awesome-icon v-else :icon="['fas', 'cloud-download-alt']" class="me-1" />
					Send {{ jdLinksCount }} Link{{ jdLinksCount !== 1 ? 's' : '' }}
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, computed, watch, getCurrentInstance } from 'vue'

const props = defineProps({
	links: { type: Array, default: () => [] },
	defaultPackageName: { type: String, default: '' },
})
const emit = defineEmits(['close', 'sent'])

const { proxy } = getCurrentInstance()
const toast = proxy.$toast

// JD form state
const jdLinksText = ref('')
const jdPackageName = ref('')
const jdDownloadPath = ref('')
const jdAutoStart = ref(false)
const jdDedupInfo = ref('')
const sendingToJD = ref(false)
const showPriorityConfig = ref(false)

// Provider priority
const DEFAULT_PROVIDER_PRIORITY = {
	pixeldrain: 1,
	gofile: 2,
	cyberfile: 3,
	bunkr: 4,
	bunkrr: 4,
	cyberdrop: 5,
	mediafire: 6,
	mega: 7,
}

const loadProviderPriority = () => {
	try {
		const saved = localStorage.getItem('providerPriority')
		if (saved) {
			const parsed = JSON.parse(saved)
			for (const [provider, priority] of Object.entries(DEFAULT_PROVIDER_PRIORITY)) {
				if (!(provider in parsed)) parsed[provider] = priority
			}
			const sorted = Object.entries(parsed).sort(([, a], [, b]) => a - b)
			const values = sorted.map(([, v]) => v)
			if (new Set(values).size !== values.length) {
				const fixed = {}
				sorted.forEach(([name], idx) => { fixed[name] = idx + 1 })
				localStorage.setItem('providerPriority', JSON.stringify(fixed))
				return fixed
			}
			return parsed
		}
	} catch (e) { /* ignore */ }
	return { ...DEFAULT_PROVIDER_PRIORITY }
}

const providerPriority = ref(loadProviderPriority())

const savePriority = () => localStorage.setItem('providerPriority', JSON.stringify(providerPriority.value))

const priorityList = computed(() =>
	Object.entries(providerPriority.value)
		.filter(([name]) => name !== 'bunkrr')
		.sort(([, a], [, b]) => a - b)
		.map(([name]) => name)
)

const movePriorityUp = (provider) => {
	const list = priorityList.value
	const idx = list.indexOf(provider)
	if (idx <= 0) return
	const above = list[idx - 1]
	const temp = providerPriority.value[provider]
	providerPriority.value[provider] = providerPriority.value[above]
	providerPriority.value[above] = temp
	if (provider === 'bunkr' || above === 'bunkr') providerPriority.value['bunkrr'] = providerPriority.value['bunkr']
	savePriority()
}

const movePriorityDown = (provider) => {
	const list = priorityList.value
	const idx = list.indexOf(provider)
	if (idx < 0 || idx >= list.length - 1) return
	const below = list[idx + 1]
	const temp = providerPriority.value[provider]
	providerPriority.value[provider] = providerPriority.value[below]
	providerPriority.value[below] = temp
	if (provider === 'bunkr' || below === 'bunkr') providerPriority.value['bunkrr'] = providerPriority.value['bunkr']
	savePriority()
}

const resetPriority = () => {
	providerPriority.value = { ...DEFAULT_PROVIDER_PRIORITY }
	savePriority()
}

const getProviderPriority = (provider) => providerPriority.value[provider.toLowerCase()] || 0

const jdLinksCount = computed(() => {
	if (!jdLinksText.value.trim()) return 0
	return jdLinksText.value.trim().split('\n').filter((l) => l.trim()).length
})

const getDeduplicatedActiveLinks = () => {
	const activeLinks = props.links.filter((l) => l.status === 'active')
	const totalActive = activeLinks.length
	const byPost = {}
	for (const link of activeLinks) {
		if (!byPost[link.post_id]) byPost[link.post_id] = []
		byPost[link.post_id].push(link)
	}
	const deduplicated = []
	for (const postLinks of Object.values(byPost)) {
		if (postLinks.length === 1) { deduplicated.push(postLinks[0]); continue }
		postLinks.sort((a, b) => getProviderPriority(a.provider) - getProviderPriority(b.provider))
		deduplicated.push(postLinks[0])
	}
	return { links: deduplicated.map((l) => l.url), removedCount: totalActive - deduplicated.length, totalActive }
}

// Initialize form when modal opens
watch(() => props.links, () => {
	const result = getDeduplicatedActiveLinks()
	jdLinksText.value = result.links.join('\n')
	jdPackageName.value = props.defaultPackageName
	jdDownloadPath.value = ''
	jdAutoStart.value = false
	if (result.removedCount > 0) {
		jdDedupInfo.value = `${result.removedCount} duplicate link${result.removedCount !== 1 ? 's' : ''} removed (${result.totalActive} active → ${result.links.length}). Priority: ${priorityList.value.join(' > ')}`
	} else if (result.totalActive > 0) {
		jdDedupInfo.value = `${result.links.length} verified active link${result.links.length !== 1 ? 's' : ''} ready to send.`
	} else {
		jdDedupInfo.value = 'No active (verified) links found. Try verifying links first.'
	}
}, { immediate: true })

const confirmSendToJD = async () => {
	if (!jdLinksText.value.trim()) return
	sendingToJD.value = true
	try {
		const urls = jdLinksText.value.trim().split('\n').filter((l) => l.trim())
		const downloadPath = jdDownloadPath.value.trim() ? `/output/${jdDownloadPath.value.trim().replace(/^\/+/, '')}` : ''
		const links = urls.map((url) => ({
			url: url.trim(),
			packageName: jdPackageName.value || undefined,
			downloadPath: downloadPath || undefined,
			autoStart: jdAutoStart.value,
		}))
		const response = await fetch('http://localhost:8080/api/v1/jdownloader/send', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ links }),
		})
		const data = await response.json()
		if (data.successCount > 0) toast.success(`Sent ${data.successCount} link${data.successCount !== 1 ? 's' : ''} to JDownloader!`)
		if (data.failCount > 0) toast.error(`${data.failCount} link${data.failCount !== 1 ? 's' : ''} failed to send`)
		if (!data.successCount && !data.failCount) toast.info('No links were sent')
		emit('sent')
		emit('close')
	} catch (error) {
		console.error('Error sending to JDownloader:', error)
		toast.error('Failed to send links: ' + error.message)
	} finally {
		sendingToJD.value = false
	}
}
</script>

<style scoped>
.jd-modal-backdrop {
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

.jd-modal-content {
	background: #1a1a2e;
	border-radius: 16px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	width: 90%;
	max-width: 650px;
	max-height: 90vh;
	overflow: auto;
	box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
}

.jd-modal-header {
	padding: 1.25rem 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.jd-modal-title {
	margin: 0;
	color: #fff;
	font-size: 1.2rem;
	font-weight: 600;
	display: flex;
	align-items: center;
}

.jd-btn-close {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	font-size: 1.5rem;
	cursor: pointer;
	padding: 0;
	line-height: 1;
}

.jd-btn-close:hover { color: #fff; }

.jd-modal-body { padding: 1.5rem; }

.jd-modal-footer {
	padding: 1rem 1.5rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: flex-end;
	gap: 0.75rem;
}

.jd-form-group { margin-bottom: 1.25rem; }

.jd-form-label {
	display: block;
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.9rem;
	font-weight: 500;
	margin-bottom: 0.5rem;
}

.jd-form-control {
	width: 100%;
	padding: 0.75rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-radius: 10px;
	color: #fff;
	font-size: 0.9rem;
	transition: border-color 0.2s;
}

.jd-form-control:focus {
	outline: none;
	border-color: rgba(74, 171, 247, 0.5);
	background: rgba(255, 255, 255, 0.08);
}

.jd-form-control::placeholder { color: rgba(255, 255, 255, 0.3); }

.jd-textarea {
	resize: vertical;
	min-height: 120px;
	font-family: 'Fira Code', 'Consolas', monospace;
	font-size: 0.8rem;
	line-height: 1.5;
}

.jd-input-group { display: flex; }

.jd-input-prefix {
	padding: 0.75rem 0.75rem;
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-right: none;
	border-radius: 10px 0 0 10px;
	color: rgba(255, 255, 255, 0.6);
	font-size: 0.9rem;
	display: flex;
	align-items: center;
}

.jd-input-with-prefix { border-radius: 0 10px 10px 0; }

.jd-form-check { display: flex; align-items: center; gap: 0.5rem; }
.jd-check-input { width: 18px; height: 18px; accent-color: #4dabf7; cursor: pointer; }
.jd-check-label { color: rgba(255, 255, 255, 0.8); font-size: 0.9rem; cursor: pointer; }

.jd-dedup-info {
	display: flex;
	align-items: flex-start;
	gap: 0.75rem;
	padding: 0.75rem 1rem;
	border-radius: 10px;
	background: rgba(74, 171, 247, 0.1);
	border: 1px solid rgba(74, 171, 247, 0.2);
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.85rem;
	margin-bottom: 1.25rem;
	line-height: 1.4;
}

.jd-dedup-info svg { color: #4dabf7; font-size: 1rem; margin-top: 0.1rem; flex-shrink: 0; }

.priority-config-section { margin-bottom: 1rem; }

.priority-toggle-btn {
	display: flex;
	align-items: center;
	width: 100%;
	padding: 0.6rem 1rem;
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	color: rgba(255, 255, 255, 0.8);
	font-size: 0.85rem;
	cursor: pointer;
	transition: background 0.2s;
}

.priority-toggle-btn:hover { background: rgba(255, 255, 255, 0.1); }

.priority-list {
	margin-top: 0.5rem;
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	overflow: hidden;
}

.priority-item {
	display: flex;
	align-items: center;
	padding: 0.5rem 0.75rem;
	background: rgba(255, 255, 255, 0.03);
	border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.priority-item:last-of-type { border-bottom: none; }

.priority-rank {
	width: 1.5rem;
	height: 1.5rem;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(139, 92, 246, 0.2);
	color: #a78bfa;
	border-radius: 50%;
	font-size: 0.75rem;
	font-weight: 600;
	margin-right: 0.75rem;
	flex-shrink: 0;
}

.priority-name { flex: 1; color: rgba(255, 255, 255, 0.85); font-size: 0.85rem; text-transform: capitalize; }
.priority-actions { display: flex; gap: 0.25rem; }

.priority-btn {
	width: 1.6rem;
	height: 1.6rem;
	display: flex;
	align-items: center;
	justify-content: center;
	background: rgba(255, 255, 255, 0.06);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 4px;
	color: rgba(255, 255, 255, 0.6);
	cursor: pointer;
	font-size: 0.7rem;
	transition: all 0.2s;
}

.priority-btn:hover:not(:disabled) { background: rgba(139, 92, 246, 0.2); color: #a78bfa; border-color: rgba(139, 92, 246, 0.3); }
.priority-btn:disabled { opacity: 0.3; cursor: not-allowed; }

.priority-reset-btn {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 100%;
	padding: 0.45rem;
	background: rgba(255, 255, 255, 0.03);
	border: none;
	border-top: 1px solid rgba(255, 255, 255, 0.06);
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.8rem;
	cursor: pointer;
	transition: all 0.2s;
}

.priority-reset-btn:hover { background: rgba(255, 255, 255, 0.06); color: rgba(255, 255, 255, 0.8); }

.btn-jdownloader {
	background: linear-gradient(135deg, #ff6b6b 0%, #ee5a52 100%) !important;
}

.btn-jdownloader:hover:not(:disabled) {
	box-shadow: 0 4px 12px rgba(255, 107, 107, 0.4) !important;
}
</style>
