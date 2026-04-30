<template>
	<Teleport to="body">
		<Transition name="scraped-modal">
			<div v-if="video" class="scraped-modal-backdrop" @click.self="$emit('close')">
				<div class="scraped-modal" @click.stop>
					<!-- Header -->
					<div class="scraped-modal-header">
						<h5 class="scraped-modal-title" :title="video.title">{{ video.title }}</h5>
						<div class="scraped-modal-header-actions">
							<button class="btn btn-sm btn-outline-primary" @click="scrapersAPI.openURL(video.source_url || video.url).catch(() => window.open(video.source_url || video.url, '_blank'))">
								<font-awesome-icon :icon="['fab', 'firefox-browser']" class="me-1" />
								Open in Firefox
							</button>
							<button class="btn btn-sm btn-warning" @click="download" :disabled="jdBusy">
								<font-awesome-icon :icon="['fas', jdBusy ? 'spinner' : 'cloud-download-alt']" :spin="jdBusy" class="me-1" />
								Download
							</button>
							<button v-if="video.downloaded_at" class="btn btn-sm btn-outline-secondary" @click="unmarkDownloaded" :disabled="jdBusy" title="Remove downloaded mark">
								<font-awesome-icon :icon="['fas', 'undo']" class="me-1" />
								Unmark
							</button>
							<button class="scraped-modal-close" @click="$emit('close')">
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
						</div>
					</div>

					<!-- Player -->
					<div class="scraped-modal-player">
						<div v-if="extracting" class="modal-extracting">
							<font-awesome-icon :icon="['fas', 'spinner']" spin size="2x" />
							<span class="mt-2">Extracting video URL…</span>
						</div>
						<video
							v-else-if="directUrl"
							:src="directUrl"
							controls
							autoplay
							class="modal-video-el"
						></video>
						<iframe
							v-else-if="embedSrc"
							:src="embedSrc"
							class="modal-iframe-el"
							allowfullscreen
							frameborder="0"
							scrolling="no"
						></iframe>
						<div v-else class="modal-extracting">
							<font-awesome-icon :icon="['fas', 'exclamation-triangle']" size="2x" class="text-warning" />
							<span class="mt-2">No playable URL found</span>
						</div>
					</div>

					<!-- Meta footer -->
					<div class="scraped-modal-meta">
						<div class="modal-badges">
							<span v-if="video.duration" class="badge bg-dark">
								<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
								{{ formatDuration(video.duration) }}
							</span>
							<span v-if="video.view_count > 0" class="badge bg-secondary">
								<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
								{{ formatCompactNumber(video.view_count) }}
							</span>
							<span v-if="video.rating > 0" class="badge bg-success">
								<font-awesome-icon :icon="['fas', 'thumbs-up']" class="me-1" />
								{{ video.rating }}%
							</span>
							<span v-if="video.source" class="badge bg-info text-dark">
								{{ video.source }}
							</span>
						</div>

						<!-- Performers -->
						<div v-if="performers.length" class="scraped-performers mt-2">
							<router-link
								v-for="p in performers"
								:key="p.name"
								:to="p.id > 0 ? '/performers/' + p.id : '#'"
								:class="['performer-chip', { matched: p.id > 0 }]"
								@click.prevent="p.id > 0 ? $router.push('/performers/' + p.id) : null"
							>{{ p.name }}</router-link>
						</div>

						<!-- Matched tags -->
						<div v-if="matchedTags.length" class="scraped-matched-tags mt-1">
							<span
								v-for="t in matchedTags"
								:key="t.id"
								class="tag-chip"
								:style="{ backgroundColor: t.color || '#6c757d' }"
							>
								<font-awesome-icon v-if="t.icon" :icon="['fas', t.icon]" class="me-1" />
								{{ t.name }}
							</span>
						</div>

						<!-- Library Match Panel -->
						<div v-if="matchLoading || libraryMatches.length > 0" class="library-match-panel mt-3">
							<div class="match-panel-header">
								<font-awesome-icon :icon="['fas', 'database']" class="me-1" />
								<span v-if="matchLoading" class="text-muted">Checking library…</span>
								<span v-else class="text-warning fw-semibold">
									<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-1" />
									Possibly already in library ({{ libraryMatches.length }})
								</span>
							</div>
							<div v-if="!matchLoading" class="match-list">
								<div
									v-for="m in libraryMatches"
									:key="m.id"
									class="match-item"
									@click="openLibraryVideo(m.id)"
									title="Open in library"
								>
									<div class="match-title">{{ m.title }}</div>
									<div class="match-meta">
										<span class="match-lib">{{ m.library_name }}</span>
										<span v-if="m.duration" class="match-dur">{{ formatDuration(m.duration) }}</span>
										<span class="match-score" :class="m.score >= 0.7 ? 'score-high' : m.score >= 0.5 ? 'score-mid' : 'score-low'">
											{{ Math.round(m.score * 100) }}%
										</span>
										<span v-for="r in (m.reasons || [])" :key="r" class="match-reason">{{ r }}</span>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</Transition>
	</Teleport>
</template>

<script setup>
/* eslint-disable no-undef */
import { ref, computed, watch } from 'vue'
import { useFormatters } from '@/composables/useFormatters'
import { scrapersAPI, jdownloaderAPI, scraperMatchAPI } from '@/services/api'

const props = defineProps({
	video: { type: Object, default: null },
	tab:   { type: Object, default: null },
})
const emit = defineEmits(['close', 'downloaded', 'unmarked'])

const { formatDuration, formatCompactNumber } = useFormatters()

const directUrl = ref(null)
const iframeUrl = ref(null)
const extracting = ref(false)
const jdBusy = ref(false)

// ── Library match ─────────────────────────────────────────────────────────────
const libraryMatches = ref([])
const matchLoading = ref(false)

// ── Metadata ──────────────────────────────────────────────────────────────────
const parsedMeta = computed(() => {
	if (!props.video?.metadata) return {}
	try {
		return typeof props.video.metadata === 'string'
			? JSON.parse(props.video.metadata)
			: props.video.metadata
	} catch { return {} }
})
const performers  = computed(() => parsedMeta.value.performers   || [])
const matchedTags = computed(() => parsedMeta.value.matched_tags || [])

// ── Embed src (iframe fallback) ───────────────────────────────────────────────
const embedSrc = computed(() => {
	if (iframeUrl.value) return iframeUrl.value
	const url = props.video?.source_url || props.video?.url || ''
	if (!url) return ''

	// PornHub embed
	if (url.includes('pornhub.com')) {
		const m = url.match(/viewkey=([a-zA-Z0-9]+)/)
		if (m) return `https://www.pornhub.com/embed/${m[1]}`
	}
	// Rule34Video embed
	if (url.includes('rule34video.com')) {
		const m = url.match(/\/video\/(\d+)\//)
		if (m) return `https://rule34video.com/embed/${m[1]}`
	}
	// LuxureTV embed
	if (url.includes('luxuretv.com')) {
		const m = url.match(/\/videos\/([^/]+)\//)
		if (m) return `https://luxuretv.com/embed/${m[1]}`
	}

	// Generic: use source URL directly in iframe
	return url
})

// ── Extract direct URL when video changes ─────────────────────────────────────
watch(() => props.video, async (v) => {
	directUrl.value = null
	iframeUrl.value = null
	libraryMatches.value = []
	if (!v) return

	// Library match check (fire-and-forget, non-blocking)
	if (v.id) {
		matchLoading.value = true
		scraperMatchAPI.findLibraryMatch(v.id)
			.then(res => { libraryMatches.value = res?.data?.matches ?? res?.matches ?? [] })
			.catch(() => {})
			.finally(() => { matchLoading.value = false })
	}

	const url = v.source_url || v.url
	if (!url) return

	// Sites that embed well via iframe — skip extraction
	const skipSites = ['pornhub.com', 'rule34video.com', 'luxuretv.com', 'xvideos.com', 'xhamster.com']
	if (skipSites.some(s => url.includes(s))) return

	extracting.value = true
	try {
		const res = await scrapersAPI.extractVideoUrl(url)
		if (res?.direct_url) directUrl.value = res.direct_url
		else if (res?.iframe_url) iframeUrl.value = res.iframe_url
	} catch { /* fall through to iframe */ }
	finally { extracting.value = false }
}, { immediate: true })

// ── Download ──────────────────────────────────────────────────────────────────
const download = async () => {
	const url = directUrl.value || iframeUrl.value || props.video?.source_url || props.video?.url
	if (!url) return
	const scraperName = props.tab?.name || props.video?.source || 'Scraper'
	const videoTitle  = props.video?.title || ''

	jdBusy.value = true
	try {
		const result = await jdownloaderAPI.sendLinks([{
			url,
			packageName:  videoTitle,
			downloadPath: `/output/${scraperName}`,
			videoId:      props.video?.id ?? null,
		}])
		if (result?.successCount > 0) {
			emit('downloaded', props.video?.id)
		}
	} catch { /* silent */ }
	finally { jdBusy.value = false }
}

// ── Open library video in new tab ────────────────────────────────────────────
const openLibraryVideo = (videoId) => {
	window.open(`/watch/${videoId}`, '_blank')
}

// ── Unmark downloaded ────────────────────────────────────────────────────────
const unmarkDownloaded = async () => {
	if (!props.video?.id) return
	jdBusy.value = true
	try {
		await scrapersAPI.unmarkVideoDownloaded(props.video.id)
		emit('unmarked', props.video.id)
	} catch { /* silent */ }
	finally { jdBusy.value = false }
}
</script>

<style scoped>
/* Backdrop */
.scraped-modal-backdrop {
	position: fixed;
	inset: 0;
	z-index: 100000;
	background: rgba(0, 0, 0, 0.8);
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 1rem;
}

/* Modal box */
.scraped-modal {
	background: #141422;
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 12px;
	width: 100%;
	max-width: 1400px;
	max-height: 96vh;
	display: flex;
	flex-direction: column;
	overflow: hidden;
	box-shadow: 0 24px 64px rgba(0, 0, 0, 0.7);
}

/* Header */
.scraped-modal-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	padding: 0.875rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	flex-shrink: 0;
}
.scraped-modal-title {
	margin: 0;
	font-size: 1rem;
	font-weight: 600;
	color: #fff;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}
.scraped-modal-header-actions {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	flex-shrink: 0;
}
.scraped-modal-close {
	background: none;
	border: none;
	color: rgba(255, 255, 255, 0.5);
	font-size: 1.1rem;
	cursor: pointer;
	padding: 0.25rem 0.5rem;
	border-radius: 6px;
	transition: color 0.15s, background 0.15s;
}
.scraped-modal-close:hover {
	color: #fff;
	background: rgba(255, 255, 255, 0.1);
}

/* Player */
.scraped-modal-player {
	position: relative;
	width: 100%;
	aspect-ratio: 16 / 9;
	background: #000;
	flex-shrink: 0;
}
.modal-video-el,
.modal-iframe-el {
	position: absolute;
	inset: 0;
	width: 100%;
	height: 100%;
	object-fit: contain;
	border: none;
}
.modal-extracting {
	position: absolute;
	inset: 0;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	color: rgba(255, 255, 255, 0.6);
	font-size: 0.9rem;
}

/* Meta footer */
.scraped-modal-meta {
	padding: 0.875rem 1.25rem;
	border-top: 1px solid rgba(255, 255, 255, 0.08);
	flex-shrink: 0;
	overflow-y: auto;
}
.modal-badges {
	display: flex;
	flex-wrap: wrap;
	gap: 0.4rem;
}

/* Transition */
.scraped-modal-enter-active,
.scraped-modal-leave-active {
	transition: opacity 0.2s ease;
}
.scraped-modal-enter-active .scraped-modal,
.scraped-modal-leave-active .scraped-modal {
	transition: transform 0.2s ease, opacity 0.2s ease;
}
.scraped-modal-enter-from,
.scraped-modal-leave-to {
	opacity: 0;
}
.scraped-modal-enter-from .scraped-modal,
.scraped-modal-leave-to .scraped-modal {
	transform: scale(0.95);
	opacity: 0;
}

/* Library match panel */
.library-match-panel {
	background: rgba(255, 193, 7, 0.06);
	border: 1px solid rgba(255, 193, 7, 0.25);
	border-radius: 8px;
	padding: 0.6rem 0.75rem;
}

.match-panel-header {
	font-size: 0.8rem;
	margin-bottom: 0.4rem;
	opacity: 0.9;
}

.match-list {
	display: flex;
	flex-direction: column;
	gap: 0.35rem;
}

.match-item {
	background: rgba(255, 255, 255, 0.04);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 6px;
	padding: 0.4rem 0.6rem;
	cursor: pointer;
	transition: background 0.15s, border-color 0.15s;
}

.match-item:hover {
	background: rgba(255, 255, 255, 0.09);
	border-color: rgba(255, 193, 7, 0.4);
}

.match-title {
	font-size: 0.8rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	color: #fff;
}

.match-meta {
	display: flex;
	gap: 0.4rem;
	flex-wrap: wrap;
	align-items: center;
	margin-top: 0.2rem;
}

.match-lib {
	font-size: 0.68rem;
	color: #00d9ff;
	opacity: 0.8;
}

.match-dur {
	font-size: 0.68rem;
	color: rgba(255, 255, 255, 0.5);
}

.match-score {
	font-size: 0.68rem;
	font-weight: 600;
	padding: 0.05rem 0.35rem;
	border-radius: 4px;
}

.score-high { background: rgba(40, 167, 69, 0.3); color: #5cdd82; }
.score-mid  { background: rgba(255, 193,  7, 0.3); color: #ffc107; }
.score-low  { background: rgba(108, 117, 125, 0.3); color: #adb5bd; }

.match-reason {
	font-size: 0.65rem;
	background: rgba(255, 255, 255, 0.1);
	border-radius: 3px;
	padding: 0.05rem 0.3rem;
	color: rgba(255, 255, 255, 0.6);
}
</style>
