<template>
	<div v-if="visible" class="assign-performer-overlay" @click.self="$emit('close')">
		<div class="assign-performer-dialog" @click.stop>
			<!-- Header -->
			<div class="apm-header">
				<div class="apm-header-icon">
					<font-awesome-icon :icon="['fas', 'user-tag']" />
				</div>
				<div class="apm-header-info">
					<h4 class="apm-title">Assign Performer</h4>
					<p class="apm-subtitle">
						<template v-if="item?.type === 'folder'">
							Link all videos in <strong>{{ item.name }}</strong> and subfolders
						</template>
						<template v-else>
							Link <strong>{{ item?.name }}</strong> to a performer
						</template>
					</p>
				</div>
				<button class="apm-close-btn" @click="$emit('close')">
					<font-awesome-icon :icon="['fas', 'times']" />
				</button>
			</div>

			<div class="apm-body">
				<!-- Selected Performer Preview -->
				<transition name="apm-preview-fade">
					<div v-if="selectedPerformer" class="apm-selected-preview">
						<div class="apm-preview-image">
							<img
								v-if="selectedPerformer.thumbnail_path"
											:src="`${API_ORIGIN}${selectedPerformer.thumbnail_path}`"
								:alt="selectedPerformer.name"
							/>
							<div v-else class="apm-preview-placeholder">
								<font-awesome-icon :icon="['fas', 'user']" />
							</div>
						</div>
						<div class="apm-preview-details">
							<div class="apm-preview-name">{{ selectedPerformer.name }}</div>
							<div class="apm-preview-meta">
								<span class="apm-meta-badge">
									<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
									{{ selectedPerformer.video_count || 0 }} videos
								</span>
								<span v-if="selectedPerformer.category" class="apm-meta-badge apm-meta-category">
									{{ selectedPerformer.category }}
								</span>
							</div>
						</div>
						<button class="apm-preview-clear" @click="selectedId = null; selectedPerformer = null" title="Clear selection">
							<font-awesome-icon :icon="['fas', 'times-circle']" />
						</button>
					</div>
				</transition>

				<!-- Search Input -->
				<div class="apm-search-wrapper">
					<font-awesome-icon :icon="['fas', 'search']" class="apm-search-icon" />
					<input
						v-model="search"
						@input="searchPerformers"
						type="text"
						class="apm-search-input"
						placeholder="Search performers by name..."
						autofocus
					/>
					<button v-if="search" class="apm-search-clear" @click="search = ''; searchPerformers()">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</div>

				<!-- Performer Grid -->
				<div class="apm-performer-grid">
					<button
						v-for="p in results"
						:key="p.id"
						class="apm-performer-card"
						:class="{ selected: selectedId === p.id }"
						@click="selectedId = p.id; selectedPerformer = p"
					>
						<div class="apm-card-image">
							<img
								v-if="p.thumbnail_path"
								:src="`${API_ORIGIN}${p.thumbnail_path}`"
								:alt="p.name"
							/>
							<div v-else class="apm-card-placeholder">
								<font-awesome-icon :icon="['fas', 'user']" />
							</div>
							<div v-if="selectedId === p.id" class="apm-card-check">
								<font-awesome-icon :icon="['fas', 'check']" />
							</div>
						</div>
						<div class="apm-card-info">
							<div class="apm-card-name" :title="p.name">{{ p.name }}</div>
							<div class="apm-card-count">{{ p.video_count || 0 }} videos</div>
						</div>
					</button>
					<div v-if="results.length === 0" class="apm-empty">
						<font-awesome-icon :icon="['fas', 'search']" class="mb-2" />
						<span>{{ search ? 'No performers found' : 'Start typing to search...' }}</span>
					</div>
				</div>
			</div>

			<!-- Footer -->
			<div class="apm-footer">
				<button class="apm-btn-cancel" @click="$emit('close')">Cancel</button>
				<button
					class="apm-btn-confirm"
					:disabled="!selectedId || loading"
					@click="confirmAssign"
				>
					<font-awesome-icon v-if="loading" :icon="['fas', 'spinner']" spin class="me-2" />
					<font-awesome-icon v-else :icon="['fas', 'user-tag']" class="me-2" />
					Assign Performer
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, watch, getCurrentInstance } from 'vue'
import { performersAPI, videosAPI, API_ORIGIN } from '@/services/api'

const props = defineProps({
	visible: { type: Boolean, default: false },
	item: { type: Object, default: null },
	tab: { type: Object, default: null },
})

const emit = defineEmits(['close', 'assigned'])

const { proxy } = getCurrentInstance()

const search = ref('')
const results = ref([])
const selectedId = ref(null)
const selectedPerformer = ref(null)
const loading = ref(false)

const resetState = () => {
	search.value = ''
	results.value = []
	selectedId.value = null
	selectedPerformer.value = null
	loading.value = false
}

const loadInitialPerformers = async () => {
	try {
		const data = await performersAPI.getAll('')
		results.value = (data.data || []).slice(0, 48)
	} catch {
		results.value = []
	}
}

const searchPerformers = async () => {
	try {
		const data = await performersAPI.getAll(search.value)
		results.value = data.data || []
	} catch {
		results.value = []
	}
}

const confirmAssign = async () => {
	const { item } = props
	if (!selectedId.value || !item) return
	loading.value = true
	try {
		if (item.type === 'folder') {
			const response = await fetch(`${API_ORIGIN}/api/v1/videos/performer/folder`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ folder_path: item.full_path, performer_id: selectedId.value }),
			})
			const data = await response.json()
			if (data.success) {
				proxy.$toast.success(`Linked performer to ${data.data.linked} videos in folder`)
				emit('assigned', props.tab)
				emit('close')
			} else {
				throw new Error(data.message || 'Failed')
			}
		} else {
			if (!item.id) {
				proxy.$toast.warning('Video must be scanned before assigning a performer')
				return
			}
			await videosAPI.update(item.id, { performer_ids: [selectedId.value] })
			proxy.$toast.success('Performer assigned to video')
			emit('close')
		}
	} catch (error) {
		console.error('Failed to assign performer:', error)
		proxy.$toast.error('Failed to assign performer: ' + error.message)
	} finally {
		loading.value = false
	}
}

watch(() => props.visible, (newVal) => {
	if (newVal) {
		resetState()
		loadInitialPerformers()
	}
})
</script>

<style scoped>
.assign-performer-overlay {
	position: fixed;
	inset: 0;
	background: rgba(0, 0, 0, 0.75);
	backdrop-filter: blur(4px);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 9999;
}

.assign-performer-dialog {
	background: #1a1a2e;
	border: 1px solid rgba(103, 126, 234, 0.3);
	border-radius: 16px;
	width: 680px;
	max-width: 95vw;
	max-height: 90vh;
	display: flex;
	flex-direction: column;
	overflow: hidden;
	box-shadow: 0 25px 60px rgba(0, 0, 0, 0.6);
}

.apm-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.08);
	background: linear-gradient(135deg, rgba(103, 126, 234, 0.15), rgba(118, 75, 162, 0.1));
}

.apm-header-icon {
	width: 44px;
	height: 44px;
	background: linear-gradient(135deg, #667eea, #764ba2);
	border-radius: 12px;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 1.2rem;
	flex-shrink: 0;
}

.apm-header-info {
	flex: 1;
	min-width: 0;
}

.apm-title {
	margin: 0;
	color: #e0e0e0;
	font-size: 1.1rem;
	font-weight: 700;
}

.apm-subtitle {
	margin: 0.2rem 0 0;
	color: #aaa;
	font-size: 0.85rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.apm-subtitle strong {
	color: #e0e0e0;
}

.apm-close-btn {
	background: none;
	border: none;
	color: #888;
	font-size: 1.1rem;
	cursor: pointer;
	padding: 0.4rem;
	border-radius: 6px;
	transition: all 0.2s;
}
.apm-close-btn:hover { color: #e0e0e0; background: rgba(255,255,255,0.08); }

.apm-body {
	padding: 1.25rem;
	overflow-y: auto;
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.apm-selected-preview {
	display: flex;
	align-items: center;
	gap: 1rem;
	background: linear-gradient(135deg, rgba(103, 126, 234, 0.15), rgba(118, 75, 162, 0.1));
	border: 1px solid rgba(103, 126, 234, 0.4);
	border-radius: 12px;
	padding: 1rem;
	position: relative;
}

.apm-preview-image {
	width: 72px;
	height: 90px;
	border-radius: 8px;
	overflow: hidden;
	flex-shrink: 0;
	background: rgba(255,255,255,0.05);
}

.apm-preview-image img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.apm-preview-placeholder {
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 2rem;
	color: #555;
}

.apm-preview-details {
	flex: 1;
}

.apm-preview-name {
	font-size: 1.1rem;
	font-weight: 700;
	color: #e0e0e0;
	margin-bottom: 0.5rem;
}

.apm-preview-meta {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
}

.apm-meta-badge {
	background: rgba(255,255,255,0.1);
	color: #bbb;
	font-size: 0.8rem;
	padding: 0.25rem 0.6rem;
	border-radius: 6px;
	display: inline-flex;
	align-items: center;
}

.apm-meta-category {
	background: rgba(103, 126, 234, 0.2);
	color: #667eea;
	text-transform: capitalize;
}

.apm-preview-clear {
	position: absolute;
	top: 0.5rem;
	right: 0.5rem;
	background: none;
	border: none;
	color: #888;
	font-size: 1rem;
	cursor: pointer;
	padding: 0.2rem;
	border-radius: 4px;
	transition: color 0.2s;
}
.apm-preview-clear:hover { color: #f44336; }

.apm-preview-fade-enter-active, .apm-preview-fade-leave-active { transition: all 0.2s ease; }
.apm-preview-fade-enter-from, .apm-preview-fade-leave-to { opacity: 0; transform: translateY(-8px); }

.apm-search-wrapper {
	position: relative;
	display: flex;
	align-items: center;
}

.apm-search-icon {
	position: absolute;
	left: 0.85rem;
	color: #888;
	font-size: 0.9rem;
	pointer-events: none;
}

.apm-search-input {
	width: 100%;
	background: rgba(255,255,255,0.05);
	border: 1px solid rgba(255,255,255,0.12);
	border-radius: 10px;
	padding: 0.65rem 2.2rem 0.65rem 2.4rem;
	color: #e0e0e0;
	font-size: 0.95rem;
	transition: border-color 0.2s;
}
.apm-search-input:focus {
	outline: none;
	border-color: rgba(103, 126, 234, 0.6);
	background: rgba(255,255,255,0.07);
}
.apm-search-input::placeholder { color: #666; }

.apm-search-clear {
	position: absolute;
	right: 0.7rem;
	background: none;
	border: none;
	color: #888;
	cursor: pointer;
	font-size: 0.85rem;
	padding: 0.2rem;
	border-radius: 4px;
	transition: color 0.2s;
}
.apm-search-clear:hover { color: #e0e0e0; }

.apm-performer-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
	gap: 0.75rem;
	max-height: 340px;
	overflow-y: auto;
	padding-right: 0.25rem;
}

.apm-performer-grid::-webkit-scrollbar { width: 4px; }
.apm-performer-grid::-webkit-scrollbar-track { background: transparent; }
.apm-performer-grid::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.15); border-radius: 2px; }

.apm-performer-card {
	background: rgba(255,255,255,0.04);
	border: 2px solid rgba(255,255,255,0.08);
	border-radius: 10px;
	padding: 0;
	cursor: pointer;
	transition: all 0.2s ease;
	overflow: hidden;
	text-align: left;
	display: flex;
	flex-direction: column;
}
.apm-performer-card:hover {
	background: rgba(255,255,255,0.07);
	border-color: rgba(103, 126, 234, 0.5);
	transform: translateY(-2px);
}
.apm-performer-card.selected {
	border-color: #667eea;
	background: rgba(103, 126, 234, 0.15);
}

.apm-card-image {
	width: 100%;
	aspect-ratio: 2/3;
	background: rgba(255,255,255,0.05);
	position: relative;
	overflow: hidden;
}

.apm-card-image img {
	width: 100%;
	height: 100%;
	object-fit: cover;
	transition: transform 0.3s ease;
}
.apm-performer-card:hover .apm-card-image img { transform: scale(1.05); }

.apm-card-placeholder {
	width: 100%;
	height: 100%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 2.5rem;
	color: #444;
}

.apm-card-check {
	position: absolute;
	inset: 0;
	background: rgba(103, 126, 234, 0.6);
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 1.8rem;
	color: white;
}

.apm-card-info {
	padding: 0.5rem 0.6rem;
}

.apm-card-name {
	font-size: 0.8rem;
	font-weight: 600;
	color: #e0e0e0;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	margin-bottom: 0.15rem;
}

.apm-card-count {
	font-size: 0.72rem;
	color: #888;
}

.apm-empty {
	grid-column: 1 / -1;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 3rem 1rem;
	color: #666;
	font-size: 0.9rem;
	gap: 0.5rem;
}

.apm-footer {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	gap: 0.75rem;
	padding: 1rem 1.5rem;
	border-top: 1px solid rgba(255,255,255,0.08);
}

.apm-btn-cancel {
	background: rgba(255,255,255,0.08);
	border: 1px solid rgba(255,255,255,0.12);
	color: #bbb;
	padding: 0.6rem 1.2rem;
	border-radius: 8px;
	cursor: pointer;
	font-size: 0.9rem;
	transition: all 0.2s;
}
.apm-btn-cancel:hover { background: rgba(255,255,255,0.12); color: #e0e0e0; }

.apm-btn-confirm {
	background: linear-gradient(135deg, #667eea, #764ba2);
	border: none;
	color: white;
	padding: 0.6rem 1.4rem;
	border-radius: 8px;
	cursor: pointer;
	font-size: 0.9rem;
	font-weight: 600;
	transition: all 0.2s;
	display: flex;
	align-items: center;
}
.apm-btn-confirm:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 4px 15px rgba(103,126,234,0.4); }
.apm-btn-confirm:disabled { opacity: 0.45; cursor: not-allowed; }
</style>
