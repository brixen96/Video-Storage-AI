<template>
	<div v-if="show" class="modal show d-block" tabindex="-1">
		<div class="modal-dialog modal-dialog-centered">
			<div class="modal-content text-bg-dark">
				<div class="modal-header">
					<h5 class="modal-title">
						<font-awesome-icon :icon="['fas', 'tag']" />
						Add Tags
					</h5>
					<button type="button" class="btn-close" @click="close"></button>
				</div>
				<div class="modal-body">
					<!-- Selected Tags -->
					<div v-if="selectedTags.length > 0" class="selected-tags mb-3">
						<h6 class="text-muted mb-2">Selected Tags:</h6>
						<div class="tags-list">
							<span v-for="tag in selectedTags" :key="tag.id" class="tag-chip" :style="{ backgroundColor: tag.color || '#6c757d' }">
								<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
								{{ tag.name }}
								<button type="button" class="btn-close btn-close-white ms-2" @click="removeTag(tag)"></button>
							</span>
						</div>
					</div>

					<!-- Available Tags -->
					<div class="available-tags">
						<h6 class="text-muted mb-2">Available Tags:</h6>
						<div class="search-box mb-3">
							<input v-model="searchQuery" type="text" class="form-control" placeholder="Search tags..." />
						</div>

						<div class="tags-grid">
							<div
								v-for="tag in filteredTags"
								:key="tag.id"
								class="tag-item"
								:class="{ selected: isSelected(tag) }"
								:style="{ borderColor: tag.color || '#6c757d' }"
								@click="toggleTag(tag)"
							>
								<div class="tag-icon" :style="{ backgroundColor: tag.color || '#6c757d' }">
									<font-awesome-icon v-if="tag.icon" :icon="['fas', tag.icon]" />
									<font-awesome-icon v-else :icon="['fas', 'tag']" />
								</div>
								<span class="tag-name">{{ tag.name }}</span>
								<font-awesome-icon v-if="isSelected(tag)" :icon="['fas', 'check-circle']" class="check-icon" />
							</div>
						</div>

						<!-- No tags found -->
						<div v-if="filteredTags.length === 0" class="text-center text-muted py-4">
							<font-awesome-icon :icon="['fas', 'tag']" size="2x" class="mb-2" />
							<p>No tags found</p>
						</div>
					</div>

					<!-- Create New Tag -->
					<div class="mt-3">
						<button class="btn btn-sm btn-outline-success w-100" @click="showCreateTag = !showCreateTag">
							<font-awesome-icon :icon="['fas', 'plus']" />
							Create New Tag
						</button>

						<!-- Create Tag Form -->
						<div v-if="showCreateTag" class="create-tag-form mt-3 p-3">
							<div class="mb-2">
								<input v-model="newTag.name" type="text" class="form-control form-control-sm" placeholder="Tag name" />
							</div>
							<div class="mb-2">
								<label class="form-label small">Color</label>
								<input v-model="newTag.color" type="color" class="form-control form-control-color" />
							</div>
							<div class="mb-2">
								<label class="form-label small">Icon (optional)</label>
								<input v-model="newTag.icon" type="text" class="form-control form-control-sm" placeholder="e.g., heart, star, fire" />
							</div>
							<button class="btn btn-sm btn-success" @click="createTag">Create</button>
							<button class="btn btn-sm btn-secondary ms-2" @click="showCreateTag = false">Cancel</button>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" @click="close">Cancel</button>
					<button type="button" class="btn btn-primary" @click="save">
						<font-awesome-icon :icon="['fas', 'save']" />
						Save Tags
					</button>
				</div>
			</div>
		</div>
	</div>
	<div v-if="show" class="modal-backdrop show"></div>
</template>

<script>
import { videosAPI, tagsAPI } from '@/services/api'

export default {
	name: 'AddTagModal',
	props: {
		show: {
			type: Boolean,
			default: false,
		},
		video: {
			type: Object,
			default: null,
		},
		videos: {
			type: Array,
			default: () => [],
		},
	},
	emits: ['close', 'saved'],
	data() {
		return {
			tags: [],
			selectedTags: [],
			searchQuery: '',
			showCreateTag: false,
			newTag: {
				name: '',
				color: '#667eea',
				icon: '',
			},
		}
	},
	computed: {
		filteredTags() {
			if (!this.searchQuery) return this.tags

			const query = this.searchQuery.toLowerCase()
			return this.tags.filter((tag) => tag.name.toLowerCase().includes(query))
		},
	},
	watch: {
		video: {
			immediate: true,
			handler(newVideo) {
				if (newVideo && newVideo.tags) {
					this.selectedTags = [...newVideo.tags]
				}
			},
		},
	},
	async mounted() {
		await this.loadTags()
	},
	methods: {
		async loadTags() {
			try {
				const response = await tagsAPI.getAll()
				this.tags = response || []
			} catch (error) {
				console.error('Failed to load tags:', error)
				this.$toast.error('Failed to load tags')
			}
		},
		isSelected(tag) {
			return this.selectedTags.some((t) => t.id === tag.id)
		},
		toggleTag(tag) {
			if (this.isSelected(tag)) {
				this.removeTag(tag)
			} else {
				this.selectedTags.push(tag)
			}
		},
		removeTag(tag) {
			const index = this.selectedTags.findIndex((t) => t.id === tag.id)
			if (index > -1) {
				this.selectedTags.splice(index, 1)
			}
		},
		async createTag() {
			if (!this.newTag.name.trim()) {
				this.$toast.error('Tag name is required')
				return
			}

			try {
				const response = await tagsAPI.create({
					name: this.newTag.name,
					color: this.newTag.color,
					icon: this.newTag.icon || undefined,
				})

				this.$toast.success('Tag created successfully')
				await this.loadTags()
				this.selectedTags.push(response.data)
				this.showCreateTag = false
				this.newTag = { name: '', color: '#667eea', icon: '' }
			} catch (error) {
				console.error('Failed to create tag:', error)
				this.$toast.error('Failed to create tag')
			}
		},
		async save() {
			try {
				const tagIds = this.selectedTags.map((t) => t.id)

				if (this.video) {
					// Single video
					await videosAPI.update(this.video.id, { tag_ids: tagIds })
					this.$toast.success('Tags updated successfully')
				} else if (this.videos.length > 0) {
					// Multiple videos
					for (const videoId of this.videos) {
						await videosAPI.update(videoId, { tag_ids: tagIds })
					}
					this.$toast.success(`Tags updated for ${this.videos.length} videos`)
				}

				this.$emit('saved')
				this.close()
			} catch (error) {
				console.error('Failed to save tags:', error)
				this.$toast.error('Failed to save tags')
			}
		},
		close() {
			this.showCreateTag = false
			this.$emit('close')
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

.selected-tags {
	padding: 1rem;
	background: #16213e;
	border-radius: 6px;
	border: 1px solid #2d2d44;
}

.tags-list {
	display: flex;
	flex-wrap: wrap;
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

.search-box input {
	background: #16213e;
	border: 1px solid #2d2d44;
	color: #ffffff;
}

.search-box input:focus {
	background: #1a1a2e;
	border-color: #667eea;
	color: #ffffff;
	box-shadow: 0 0 0 0.2rem rgba(102, 126, 234, 0.25);
}

.tags-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
	gap: 0.75rem;
	max-height: 400px;
	overflow-y: auto;
	padding: 0.5rem;
}

.tag-item {
	background: #16213e;
	border: 2px solid;
	border-radius: 8px;
	padding: 1rem;
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 0.5rem;
	cursor: pointer;
	transition: all 0.2s;
	position: relative;
}

.tag-item:hover {
	transform: translateY(-2px);
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.tag-item.selected {
	background: rgba(102, 126, 234, 0.2);
	border-width: 3px;
}

.tag-icon {
	width: 48px;
	height: 48px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 1.25rem;
}

.tag-name {
	font-size: 0.875rem;
	font-weight: 500;
	text-align: center;
	color: #ffffff;
}

.check-icon {
	position: absolute;
	top: 8px;
	right: 8px;
	color: #4caf50;
	font-size: 1.25rem;
}

.create-tag-form {
	background: #16213e;
	border: 1px solid #2d2d44;
	border-radius: 6px;
}

.form-control,
.form-control-color {
	background: #0f0c29;
	border: 1px solid #2d2d44;
	color: #ffffff;
}

.form-control:focus,
.form-control-color:focus {
	background: #1a1a2e;
	border-color: #667eea;
	color: #ffffff;
	box-shadow: 0 0 0 0.2rem rgba(102, 126, 234, 0.25);
}

.form-label {
	color: #a0a0c0;
	margin-bottom: 0.25rem;
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
