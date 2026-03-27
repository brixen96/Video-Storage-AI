<template>
	<Teleport to="body">
		<Transition name="gs-fade">
			<div v-if="open" class="gs-backdrop" @click.self="close">
				<div class="gs-modal" role="dialog" aria-modal="true">
					<!-- Input Row -->
					<div class="gs-input-row">
						<font-awesome-icon :icon="['fas', 'search']" class="gs-search-icon" />
						<input
							ref="inputEl"
							v-model="query"
							class="gs-input"
							placeholder="Search videos, performers, studios…"
							autocomplete="off"
							spellcheck="false"
							@keydown.esc.prevent="close"
							@keydown.down.prevent="moveDown"
							@keydown.up.prevent="moveUp"
							@keydown.enter.prevent="selectActive"
						/>
						<kbd class="gs-esc-hint">ESC</kbd>
					</div>

					<!-- Results -->
					<div v-if="query.length >= 2" class="gs-results">
						<div v-if="loading" class="gs-status">
							<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2" />
							Searching…
						</div>
						<div v-else-if="allResults.length === 0" class="gs-status">No results for "{{ query }}"</div>
						<template v-else>
							<!-- Videos -->
							<div v-if="videoResults.length > 0">
								<div class="gs-section-header">
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									Videos
								</div>
								<div
									v-for="(result, idx) in videoResults"
									:key="`v-${result.id}`"
									class="gs-result-item"
									:class="{ active: videoOffset + idx === activeIndex }"
									@click="navigateTo(result, 'video')"
									@mouseenter="activeIndex = videoOffset + idx"
								>
									<img v-if="result.thumbnail_path" :src="getVideoThumb(result)" class="gs-thumb" loading="lazy" />
									<div v-else class="gs-thumb gs-thumb-placeholder">
										<font-awesome-icon :icon="['fas', 'video']" />
									</div>
									<div class="gs-result-info">
										<div class="gs-result-title">{{ result.title }}</div>
										<div class="gs-result-meta">
											<span v-if="result.duration">{{ formatDuration(result.duration) }}</span>
											<span v-if="result.performers && result.performers.length">· {{ result.performers[0].name }}</span>
										</div>
									</div>
								</div>
							</div>

							<!-- Performers -->
							<div v-if="performerResults.length > 0">
								<div class="gs-section-header">
									<font-awesome-icon :icon="['fas', 'users']" class="me-2" />
									Performers
								</div>
								<div
									v-for="(result, idx) in performerResults"
									:key="`p-${result.id}`"
									class="gs-result-item"
									:class="{ active: performerOffset + idx === activeIndex }"
									@click="navigateTo(result, 'performer')"
									@mouseenter="activeIndex = performerOffset + idx"
								>
									<div class="gs-thumb gs-thumb-avatar">
										<img v-if="result.thumbnail_path" :src="getPerformerThumb(result)" loading="lazy" />
										<font-awesome-icon v-else :icon="['fas', 'user']" />
									</div>
									<div class="gs-result-info">
										<div class="gs-result-title">{{ result.name }}</div>
										<div class="gs-result-meta">{{ result.video_count || 0 }} videos</div>
									</div>
								</div>
							</div>

							<!-- Studios -->
							<div v-if="studioResults.length > 0">
								<div class="gs-section-header">
									<font-awesome-icon :icon="['fas', 'building']" class="me-2" />
									Studios
								</div>
								<div
									v-for="(result, idx) in studioResults"
									:key="`s-${result.id}`"
									class="gs-result-item"
									:class="{ active: studioOffset + idx === activeIndex }"
									@click="navigateTo(result, 'studio')"
									@mouseenter="activeIndex = studioOffset + idx"
								>
									<div class="gs-thumb gs-thumb-avatar">
										<img v-if="result.thumbnail_path" :src="getStudioThumb(result)" loading="lazy" />
										<font-awesome-icon v-else :icon="['fas', 'building']" />
									</div>
									<div class="gs-result-info">
										<div class="gs-result-title">{{ result.name }}</div>
										<div class="gs-result-meta">{{ result.video_count || 0 }} videos<span v-if="result.country"> · {{ result.country }}</span></div>
									</div>
								</div>
							</div>
						</template>
					</div>

					<!-- Idle hint -->
					<div v-else class="gs-hint">
						<span>Type at least 2 characters · </span>
						<span>↑↓ navigate · Enter select</span>
					</div>
				</div>
			</div>
		</Transition>
	</Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { videosAPI, performersAPI, studiosAPI, getAssetURL } from '@/services/api'
import { useFormatters } from '@/composables/useFormatters'
import { debounce } from '@/utils/debounce'

const router = useRouter()
const { formatDuration } = useFormatters()

const open = ref(false)
const query = ref('')
const loading = ref(false)
const videoResults = ref([])
const performerResults = ref([])
const studioResults = ref([])
const activeIndex = ref(0)
const inputEl = ref(null)

// Flat offsets for keyboard navigation
const videoOffset = computed(() => 0)
const performerOffset = computed(() => videoResults.value.length)
const studioOffset = computed(() => videoResults.value.length + performerResults.value.length)
const allResults = computed(() => [...videoResults.value, ...performerResults.value, ...studioResults.value])

const openSearch = () => {
	open.value = true
	nextTick(() => inputEl.value?.focus())
}

const close = () => {
	open.value = false
	query.value = ''
	videoResults.value = []
	performerResults.value = []
	studioResults.value = []
	activeIndex.value = 0
}

const moveDown = () => {
	if (activeIndex.value < allResults.value.length - 1) activeIndex.value++
}

const moveUp = () => {
	if (activeIndex.value > 0) activeIndex.value--
}

const selectActive = () => {
	const idx = activeIndex.value
	if (idx < videoResults.value.length) {
		navigateTo(videoResults.value[idx], 'video')
	} else if (idx < videoResults.value.length + performerResults.value.length) {
		navigateTo(performerResults.value[idx - videoResults.value.length], 'performer')
	} else {
		const sIdx = idx - videoResults.value.length - performerResults.value.length
		if (sIdx < studioResults.value.length) {
			navigateTo(studioResults.value[sIdx], 'studio')
		}
	}
}

const navigateTo = (result, type) => {
	close()
	if (type === 'video') {
		const routeName = result.source ? 'ScrapedVideoPlayer' : 'VideoPlayer'
		router.push({ name: routeName, params: { id: result.id } })
	} else if (type === 'performer') {
		router.push(`/performers/${result.id}`)
	} else {
		router.push({ path: '/studios', query: { search: result.name } })
	}
}

const getVideoThumb = (video) => {
	if (video.thumbnail_path?.startsWith('http')) return video.thumbnail_path
	return video.thumbnail_path ? getAssetURL(video.thumbnail_path) : ''
}

const getPerformerThumb = (performer) => {
	if (!performer.thumbnail_path) return ''
	if (performer.thumbnail_path.startsWith('http')) return performer.thumbnail_path
	return getAssetURL(performer.thumbnail_path)
}

const getStudioThumb = (studio) => {
	if (!studio.thumbnail_path) return ''
	if (studio.thumbnail_path.startsWith('http')) return studio.thumbnail_path
	return getAssetURL(studio.thumbnail_path)
}

// Debounced search
const doSearch = debounce(async (q) => {
	if (q.length < 2) {
		videoResults.value = []
		performerResults.value = []
		studioResults.value = []
		loading.value = false
		return
	}
	loading.value = true
	activeIndex.value = 0
	try {
		const [vRes, pRes, sRes] = await Promise.allSettled([
			videosAPI.getAll({ query: q, per_page: 5 }),
			performersAPI.getAll(q),
			studiosAPI.getAll(),
		])

		// Videos
		if (vRes.status === 'fulfilled') {
			const raw = vRes.value
			const arr = Array.isArray(raw) ? raw : (raw?.data ?? [])
			videoResults.value = arr.slice(0, 5)
		}

		// Performers
		if (pRes.status === 'fulfilled') {
			const raw = pRes.value
			const arr = Array.isArray(raw) ? raw : (raw?.data ?? [])
			performerResults.value = arr.slice(0, 5)
		}

		// Studios (client-side filter)
		if (sRes.status === 'fulfilled') {
			const raw = sRes.value
			const arr = Array.isArray(raw) ? raw : (raw?.data ?? [])
			const ql = q.toLowerCase()
			studioResults.value = arr
				.filter((s) => s.name?.toLowerCase().includes(ql))
				.slice(0, 4)
		}
	} catch {
		// silent
	} finally {
		loading.value = false
	}
}, 250)

watch(query, (q) => {
	if (q.length >= 2) {
		loading.value = true
		doSearch(q)
	} else {
		videoResults.value = []
		performerResults.value = []
		studioResults.value = []
		loading.value = false
	}
})

// Global keyboard shortcut
const handleKeydown = (e) => {
	// Ctrl+K or Cmd+K
	if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
		e.preventDefault()
		open.value ? close() : openSearch()
		return
	}
	// `/` when not in a text field
	if (e.key === '/' && !open.value) {
		const tag = document.activeElement?.tagName?.toLowerCase()
		if (tag === 'input' || tag === 'textarea' || document.activeElement?.isContentEditable) return
		e.preventDefault()
		openSearch()
	}
}

onMounted(() => {
	document.addEventListener('keydown', handleKeydown)
	document.addEventListener('open-global-search', openSearch)
})
onBeforeUnmount(() => {
	document.removeEventListener('keydown', handleKeydown)
	document.removeEventListener('open-global-search', openSearch)
})

// Expose open for external trigger (e.g. NavBar button)
defineExpose({ openSearch })
</script>

<style scoped>
.gs-backdrop {
	position: fixed;
	inset: 0;
	background: rgba(0, 0, 0, 0.6);
	backdrop-filter: blur(4px);
	z-index: 9999;
	display: flex;
	align-items: flex-start;
	justify-content: center;
	padding-top: 12vh;
}

.gs-modal {
	background: #1a1a2e;
	border: 1px solid rgba(102, 126, 234, 0.4);
	border-radius: 14px;
	width: min(640px, 90vw);
	box-shadow: 0 24px 64px rgba(0, 0, 0, 0.6), 0 0 0 1px rgba(102, 126, 234, 0.2);
	overflow: hidden;
}

.gs-input-row {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.9rem 1.1rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.gs-search-icon {
	color: rgba(255, 255, 255, 0.4);
	font-size: 1rem;
	flex-shrink: 0;
}

.gs-input {
	flex: 1;
	background: transparent;
	border: none;
	outline: none;
	color: #fff;
	font-size: 1.05rem;
}

.gs-input::placeholder {
	color: rgba(255, 255, 255, 0.3);
}

.gs-esc-hint {
	font-size: 0.7rem;
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-radius: 4px;
	padding: 2px 6px;
	color: rgba(255, 255, 255, 0.4);
	flex-shrink: 0;
}

.gs-results {
	max-height: 420px;
	overflow-y: auto;
}

.gs-status {
	padding: 1.5rem;
	text-align: center;
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.9rem;
}

.gs-section-header {
	padding: 0.5rem 1rem;
	font-size: 0.72rem;
	font-weight: 600;
	text-transform: uppercase;
	letter-spacing: 0.08em;
	color: rgba(102, 126, 234, 0.8);
	background: rgba(102, 126, 234, 0.05);
	border-bottom: 1px solid rgba(102, 126, 234, 0.1);
}

.gs-result-item {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.6rem 1rem;
	cursor: pointer;
	transition: background 0.15s;
	border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.gs-result-item:hover,
.gs-result-item.active {
	background: rgba(102, 126, 234, 0.12);
}

.gs-thumb {
	width: 52px;
	height: 36px;
	border-radius: 6px;
	object-fit: cover;
	flex-shrink: 0;
	background: rgba(255, 255, 255, 0.05);
	display: flex;
	align-items: center;
	justify-content: center;
	color: rgba(255, 255, 255, 0.3);
	font-size: 0.9rem;
}

.gs-thumb-placeholder {
	background: rgba(255, 255, 255, 0.05);
}

.gs-thumb-avatar {
	width: 36px;
	height: 36px;
	border-radius: 50%;
}

.gs-thumb-avatar img {
	width: 100%;
	height: 100%;
	object-fit: cover;
	border-radius: 50%;
}

.gs-result-info {
	flex: 1;
	min-width: 0;
}

.gs-result-title {
	font-size: 0.9rem;
	color: #fff;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.gs-result-meta {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.4);
	margin-top: 2px;
}

.gs-hint {
	padding: 1rem 1.1rem;
	font-size: 0.8rem;
	color: rgba(255, 255, 255, 0.3);
	display: flex;
	justify-content: space-between;
}

/* Transition */
.gs-fade-enter-active,
.gs-fade-leave-active {
	transition: opacity 0.15s ease;
}
.gs-fade-enter-active .gs-modal,
.gs-fade-leave-active .gs-modal {
	transition: transform 0.15s ease, opacity 0.15s ease;
}
.gs-fade-enter-from,
.gs-fade-leave-to {
	opacity: 0;
}
.gs-fade-enter-from .gs-modal {
	transform: translateY(-12px) scale(0.97);
}
.gs-fade-leave-to .gs-modal {
	transform: translateY(-8px) scale(0.98);
	opacity: 0;
}
</style>
