<template>
	<div class="provider-priority-section mb-3">
		<button class="btn btn-outline-secondary btn-sm w-100 d-flex align-items-center justify-content-between" @click="showPriorityConfig = !showPriorityConfig">
			<span>
				<font-awesome-icon :icon="['fas', 'sort-amount-down']" class="me-2" />
				Download Provider Priority
			</span>
			<font-awesome-icon :icon="['fas', showPriorityConfig ? 'chevron-up' : 'chevron-down']" />
		</button>
		<div v-if="showPriorityConfig" class="priority-config-card mt-2">
			<p class="text-muted small mb-2">Set the priority order for download providers. Higher priority providers are listed first when sending links to JDownloader.</p>
			<div class="priority-list-jd">
				<div v-for="(provider, idx) in priorityList" :key="provider" class="priority-item-jd">
					<span class="priority-rank-jd">{{ idx + 1 }}</span>
					<span class="priority-name-jd">{{ provider }}</span>
					<div class="priority-actions-jd">
						<button @click="movePriorityUp(provider)" :disabled="idx === 0" class="btn btn-sm btn-outline-light priority-btn-jd" title="Move up">
							<font-awesome-icon :icon="['fas', 'chevron-up']" />
						</button>
						<button @click="movePriorityDown(provider)" :disabled="idx === priorityList.length - 1" class="btn btn-sm btn-outline-light priority-btn-jd" title="Move down">
							<font-awesome-icon :icon="['fas', 'chevron-down']" />
						</button>
					</div>
				</div>
			</div>
			<button class="btn btn-outline-secondary btn-sm mt-2" @click="resetPriority">
				<font-awesome-icon :icon="['fas', 'undo']" class="me-1" />
				Reset to Defaults
			</button>
		</div>
	</div>
</template>

<script setup>
import { ref, computed } from 'vue'

const DEFAULT_PROVIDER_PRIORITY = { pixeldrain: 1, gofile: 2, cyberfile: 3, bunkr: 4, bunkrr: 4, cyberdrop: 5, mediafire: 6, mega: 7 }
const showPriorityConfig = ref(false)

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

const priorityList = computed(() => {
	return Object.entries(providerPriority.value)
		.filter(([name]) => name !== 'bunkrr')
		.sort(([, a], [, b]) => a - b)
		.map(([name]) => name)
})

const savePriority = () => {
	localStorage.setItem('providerPriority', JSON.stringify(providerPriority.value))
}

const movePriorityUp = (provider) => {
	const sorted = priorityList.value
	const idx = sorted.indexOf(provider)
	if (idx <= 0) return
	const above = sorted[idx - 1]
	const temp = providerPriority.value[provider]
	providerPriority.value[provider] = providerPriority.value[above]
	providerPriority.value[above] = temp
	if (provider === 'bunkr' || above === 'bunkr') {
		providerPriority.value['bunkrr'] = providerPriority.value['bunkr']
	}
	savePriority()
}

const movePriorityDown = (provider) => {
	const sorted = priorityList.value
	const idx = sorted.indexOf(provider)
	if (idx >= sorted.length - 1) return
	const below = sorted[idx + 1]
	const temp = providerPriority.value[provider]
	providerPriority.value[provider] = providerPriority.value[below]
	providerPriority.value[below] = temp
	if (provider === 'bunkr' || below === 'bunkr') {
		providerPriority.value['bunkrr'] = providerPriority.value['bunkr']
	}
	savePriority()
}

const resetPriority = () => {
	providerPriority.value = { ...DEFAULT_PROVIDER_PRIORITY }
	savePriority()
}
</script>

<style scoped>
.priority-config-card {
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 8px;
	padding: 1rem;
}

.priority-list-jd {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.priority-item-jd {
	display: flex;
	align-items: center;
	padding: 0.5rem 0.75rem;
	background: rgba(255, 255, 255, 0.03);
	border: 1px solid rgba(255, 255, 255, 0.06);
	border-radius: 6px;
}

.priority-rank-jd {
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

.priority-name-jd {
	flex: 1;
	color: rgba(255, 255, 255, 0.85);
	font-size: 0.85rem;
	text-transform: capitalize;
}

.priority-actions-jd {
	display: flex;
	gap: 0.25rem;
}

.priority-btn-jd {
	padding: 0.15rem 0.4rem;
	font-size: 0.7rem;
}

.priority-btn-jd:disabled {
	opacity: 0.3;
}
</style>
