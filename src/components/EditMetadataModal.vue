<template>
	<div v-if="show" class="modal show d-block" tabindex="-1">
		<div class="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable">
			<div class="modal-content text-bg-dark">
				<div class="modal-header">
					<h5 class="modal-title">
						<font-awesome-icon :icon="['fas', 'edit']" />
						Edit Metadata
					</h5>
					<button type="button" class="btn-close" @click="close"></button>
				</div>
				<div class="modal-body">
					<form @submit.prevent="save">
						<!-- Title -->
						<div class="mb-3">
							<label class="form-label">Title</label>
							<input v-model="formData.title" type="text" class="form-control" required />
						</div>

						<!-- Performers -->
						<div class="mb-3">
							<label class="form-label">Performers</label>
							<div class="performers-selector">
								<div class="selected-performers mb-2">
									<span v-for="performer in selectedPerformers" :key="performer.id" class="badge bg-primary me-2 mb-2">
										<font-awesome-icon v-if="performer.zoo" :icon="['fas', 'dog']" class="me-1" title="Zoo Content" />
										{{ performer.name }}
										<button type="button" class="btn-close btn-close-white ms-2" @click="removePerformer(performer)"></button>
									</span>
								</div>
								<div class="input-group">
									<input v-model="performerSearch" type="text" class="form-control" placeholder="Search performers..." @input="searchPerformers" />
									<button type="button" class="btn btn-outline-success" @click="showAddPerformerModal = true">
										<font-awesome-icon :icon="['fas', 'plus']" />
										New
									</button>
								</div>
								<div v-if="performerResults.length > 0" class="search-results mt-2">
									<div v-for="performer in performerResults" :key="performer.id" class="search-result-item" @click="addPerformer(performer)">
										<img v-if="performer.image_path" :src="getAssetURL(performer.image_path)" :alt="performer.name" class="performer-thumb" />
										<div v-else class="performer-thumb-placeholder">
											<font-awesome-icon :icon="['fas', 'user']" />
										</div>
										<span>
											<font-awesome-icon v-if="performer.zoo" :icon="['fas', 'dog']" class="text-danger me-1" title="Zoo Content" />
											{{ performer.name }}
										</span>
									</div>
								</div>
							</div>
						</div>

						<!-- Studios -->
						<div class="mb-3">
							<label class="form-label">Studio</label>
							<select v-model="formData.studioId" class="form-select" @change="onStudioChange">
								<option :value="null">None</option>
								<option v-for="studio in studios" :key="studio.id" :value="studio.id">{{ studio.name }}</option>
							</select>
						</div>

						<!-- Groups -->
						<div v-if="filteredGroups.length > 0" class="mb-3">
							<label class="form-label">Group</label>
							<select v-model="formData.groupId" class="form-select">
								<option :value="null">None</option>
								<option v-for="group in filteredGroups" :key="group.id" :value="group.id">{{ group.name }}</option>
							</select>
						</div>

						<!-- Tags -->
						<div class="mb-3">
							<label class="form-label">Tags</label>
							<div class="tags-selector">
								<div class="selected-tags mb-2">
									<span v-for="tag in selectedTags" :key="tag.id" class="tag-chip me-2 mb-2" :style="{ backgroundColor: tag.color || '#6c757d' }">
										<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
										{{ tag.name }}
										<button type="button" class="btn-close btn-close-white ms-2" @click="removeTag(tag)"></button>
									</span>
								</div>
								<select v-model="selectedTagId" class="form-select" @change="addTag">
									<option :value="null">Select a tag...</option>
									<option v-for="tag in availableTags" :key="tag.id" :value="tag.id">{{ tag.name }}</option>
								</select>
							</div>
						</div>

						<!-- Date -->
						<div class="mb-3">
							<label class="form-label">Release Date</label>
							<input v-model="formData.date" type="date" class="form-control" />
						</div>

						<!-- Rating -->
						<div class="mb-3">
							<label class="form-label">Rating</label>
							<div class="rating-selector">
								<button
									v-for="star in 5"
									:key="star"
									type="button"
									class="btn btn-sm"
									:class="star <= formData.rating ? 'btn-warning' : 'btn-outline-secondary'"
									@click="formData.rating = star"
								>
									<font-awesome-icon :icon="['fas', 'star']" />
								</button>
								<button type="button" class="btn btn-sm btn-outline-secondary ms-2" @click="formData.rating = 0">Clear</button>
							</div>
						</div>

						<!-- Description -->
						<div class="mb-3">
							<label class="form-label">Description</label>
							<textarea v-model="formData.description" class="form-control" rows="4"></textarea>
						</div>

						<!-- Checkboxes -->
						<div class="mb-3">
							<div class="form-check">
								<input id="favorite" v-model="formData.isFavorite" type="checkbox" class="form-check-input" />
								<label class="form-check-label" for="favorite">
									<font-awesome-icon :icon="['fas', 'heart']" />
									Mark as Favorite
								</label>
							</div>
							<div class="form-check">
								<input id="pinned" v-model="formData.isPinned" type="checkbox" class="form-check-input" />
								<label class="form-check-label" for="pinned">
									<font-awesome-icon :icon="['fas', 'thumbtack']" />
									Pin to Top
								</label>
							</div>
						</div>
					</form>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" @click="close">Cancel</button>
					<button type="button" class="btn btn-primary" @click="save">
						<font-awesome-icon :icon="['fas', 'save']" />
						Save Changes
					</button>
				</div>
			</div>
		</div>
	</div>
	<div v-if="show" class="modal-backdrop show"></div>

	<!-- Add Performer Modal -->
	<AddPerformerModal :show="showAddPerformerModal" @close="showAddPerformerModal = false" @performer-added="onPerformerAdded" />
</template>

<script>
import { videosAPI, performersAPI, getAssetURL } from '@/services/api'
import AddPerformerModal from './AddPerformerModal.vue'

export default {
	name: 'EditMetadataModal',
	components: {
		AddPerformerModal,
	},
	props: {
		show: {
			type: Boolean,
			default: false,
		},
		video: {
			type: Object,
			default: null,
		},
	},
	emits: ['close', 'saved'],
	data() {
		return {
			formData: {
				title: '',
				studioId: null,
				groupId: null,
				date: '',
				rating: 0,
				description: '',
				isFavorite: false,
				isPinned: false,
			},
			selectedPerformers: [],
			selectedTags: [],
			performerSearch: '',
			performerResults: [],
			studios: [],
			groups: [],
			tags: [],
			selectedTagId: null,
			showAddPerformerModal: false,
		}
	},
	computed: {
		availableTags() {
			return this.tags.filter((tag) => !this.selectedTags.find((t) => t.id === tag.id))
		},
		filteredGroups() {
			if (!this.formData.studioId) return this.groups
			return this.groups.filter((g) => g.studio_id === this.formData.studioId)
		},
	},
	watch: {
		video: {
			immediate: true,
			handler(newVideo) {
				if (newVideo) {
					this.loadFormData()
				}
			},
		},
	},
	async mounted() {
		await this.loadStudios()
		await this.loadGroups()
		await this.loadTags()
	},
	methods: {
		getAssetURL,
		loadFormData() {
			if (!this.video) return

			this.formData = {
				title: this.video.title || '',
				studioId: this.video.studios?.[0]?.id || null,
				groupId: this.video.groups?.[0]?.id || null,
				date: this.video.date || '',
				rating: this.video.rating || 0,
				description: this.video.description || '',
				isFavorite: this.video.is_favorite || false,
				isPinned: this.video.is_pinned || false,
			}

			this.selectedPerformers = this.video.performers ? [...this.video.performers] : []
			this.selectedTags = this.video.tags ? [...this.video.tags] : []
		},
		async loadStudios() {
			try {
				this.studios = await this.$store.dispatch('fetchStudios')
			} catch (error) {
				console.error('Failed to load studios:', error)
			}
		},
		async loadGroups() {
			try {
				this.groups = await this.$store.dispatch('fetchGroups')
			} catch (error) {
				console.error('Failed to load groups:', error)
			}
		},
		onStudioChange() {
			// Reset group when studio changes
			this.formData.groupId = null
		},
		async loadTags() {
			try {
				this.tags = await this.$store.dispatch('fetchTags')
			} catch (error) {
				console.error('Failed to load tags:', error)
			}
		},
		async searchPerformers() {
			if (this.performerSearch.length < 2) {
				this.performerResults = []
				return
			}

			try {
				const response = await performersAPI.search({ search: this.performerSearch, limit: 10 })
				this.performerResults = response.data || []
			} catch (error) {
				console.error('Failed to search performers:', error)
			}
		},
		addPerformer(performer) {
			if (!this.selectedPerformers.find((p) => p.id === performer.id)) {
				this.selectedPerformers.push(performer)
			}
			this.performerSearch = ''
			this.performerResults = []
		},
		removePerformer(performer) {
			const index = this.selectedPerformers.findIndex((p) => p.id === performer.id)
			if (index > -1) {
				this.selectedPerformers.splice(index, 1)
			}
		},
		addTag() {
			if (!this.selectedTagId) return

			const tag = this.tags.find((t) => t.id === this.selectedTagId)
			if (tag && !this.selectedTags.find((t) => t.id === tag.id)) {
				this.selectedTags.push(tag)
			}
			this.selectedTagId = null
		},
		removeTag(tag) {
			const index = this.selectedTags.findIndex((t) => t.id === tag.id)
			if (index > -1) {
				this.selectedTags.splice(index, 1)
			}
		},
		async save() {
			try {
				const updateData = {
					title: this.formData.title,
					studio_id: this.formData.studioId,
					group_id: this.formData.groupId,
					performer_ids: this.selectedPerformers.map((p) => p.id),
					tag_ids: this.selectedTags.map((t) => t.id),
					date: this.formData.date || undefined,
					rating: this.formData.rating || undefined,
					description: this.formData.description || undefined,
					is_favorite: this.formData.isFavorite,
					is_pinned: this.formData.isPinned,
				}

				await videosAPI.update(this.video.id, updateData)
				this.$toast.success('Metadata updated successfully')
				this.$emit('saved')
				this.close()
			} catch (error) {
				console.error('Failed to update metadata:', error)
				this.$toast.error('Failed to update metadata: ' + (error.response?.data?.error || error.message))
			}
		},
		close() {
			this.$emit('close')
		},
		async onPerformerAdded(performer) {
			this.showAddPerformerModal = false
			// Reload performers and add to selection
			await this.searchPerformers()
			this.addPerformer(performer)
		},
	},
}
</script>

<style scoped>
.modal-content {
	background: #1a1a2e;
	border: 1px solid #2d2d44;
}

.modal-header {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	color: white;
	border-bottom: 1px solid #2d2d44;
}

.modal-title {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.btn-close {
	filter: brightness(0) invert(1);
}

.form-label {
	color: #a0a0c0;
	font-weight: 500;
	margin-bottom: 0.5rem;
}

.form-control,
.form-select {
	background: #16213e;
	border: 1px solid #2d2d44;
	color: #ffffff;
}

.form-control:focus,
.form-select:focus {
	background: #1a1a2e;
	border-color: #667eea;
	color: #ffffff;
	box-shadow: 0 0 0 0.2rem rgba(102, 126, 234, 0.25);
}

.performers-selector,
.tags-selector {
	background: #16213e;
	padding: 1rem;
	border-radius: 6px;
	border: 1px solid #2d2d44;
}

.selected-performers,
.selected-tags {
	min-height: 40px;
}

.badge {
	padding: 0.5rem 0.75rem;
	font-size: 0.875rem;
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
}

.tag-chip {
	padding: 0.5rem 0.75rem;
	border-radius: 20px;
	font-size: 0.875rem;
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	color: white;
}

.search-results {
	max-height: 200px;
	overflow-y: auto;
	background: #0f0c29;
	border: 1px solid #2d2d44;
	border-radius: 4px;
}

.search-result-item {
	padding: 0.75rem;
	display: flex;
	align-items: center;
	gap: 0.75rem;
	cursor: pointer;
	transition: all 0.2s;
}

.search-result-item:hover {
	background: #16213e;
}

.performer-thumb,
.performer-thumb-placeholder {
	width: 40px;
	height: 40px;
	border-radius: 50%;
	object-fit: cover;
	display: flex;
	align-items: center;
	justify-content: center;
	background: #2d2d44;
	color: #667eea;
}

.rating-selector {
	display: flex;
	gap: 0.5rem;
	align-items: center;
}

.form-check-label {
	color: #a0a0c0;
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.btn-primary {
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
	border: none;
}

.btn-primary:hover {
	background: linear-gradient(135deg, #764ba2 0%, #667eea 100%);
	transform: translateY(-2px);
	box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}
</style>
