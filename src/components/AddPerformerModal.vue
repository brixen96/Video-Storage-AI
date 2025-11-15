<template>
	<div v-if="show" class="modal show d-block" tabindex="-1">
		<div class="modal-dialog modal-dialog-centered">
			<div class="modal-content text-bg-dark">
				<div class="modal-header">
					<h5 class="modal-title">
						<font-awesome-icon :icon="['fas', 'user-plus']" />
						Add New Performer
					</h5>
					<button type="button" class="btn-close" @click="close"></button>
				</div>
				<div class="modal-body">
					<form @submit.prevent="save">
						<!-- Name -->
						<div class="mb-3">
							<label class="form-label">Name *</label>
							<input v-model="formData.name" type="text" class="form-control" placeholder="Performer name" required />
						</div>

						<!-- Image Upload -->
						<div class="mb-3">
							<label class="form-label">Profile Image</label>
							<div class="image-upload-area">
								<div v-if="imagePreview" class="image-preview">
									<img :src="imagePreview" alt="Preview" />
									<button type="button" class="btn btn-sm btn-danger remove-image" @click="removeImage">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>
								<div v-else class="upload-placeholder" @click="$refs.imageInput.click()">
									<font-awesome-icon :icon="['fas', 'cloud-upload-alt']" size="2x" class="mb-2" />
									<p>Click to upload image</p>
									<small class="text-muted">PNG, JPG up to 10MB</small>
								</div>
								<input ref="imageInput" type="file" accept="image/*" class="d-none" @change="handleImageUpload" />
							</div>
						</div>

						<!-- Birthdate -->
						<div class="mb-3">
							<label class="form-label">Birthdate</label>
							<input v-model="formData.birthdate" type="date" class="form-control" />
						</div>

						<!-- Country -->
						<div class="mb-3">
							<label class="form-label">Country</label>
							<input v-model="formData.country" type="text" class="form-control" placeholder="Country" />
						</div>

						<!-- Height -->
						<div class="mb-3">
							<label class="form-label">Height (cm)</label>
							<input v-model.number="formData.height" type="number" class="form-control" placeholder="170" />
						</div>

						<!-- Weight -->
						<div class="mb-3">
							<label class="form-label">Weight (kg)</label>
							<input v-model.number="formData.weight" type="number" class="form-control" placeholder="60" />
						</div>

						<!-- Measurements -->
						<div class="mb-3">
							<label class="form-label">Measurements</label>
							<input v-model="formData.measurements" type="text" class="form-control" placeholder="90-60-90" />
						</div>

						<!-- Hair Color -->
						<div class="mb-3">
							<label class="form-label">Hair Color</label>
							<select v-model="formData.hairColor" class="form-select">
								<option value="">Select...</option>
								<option value="Blonde">Blonde</option>
								<option value="Brown">Brown</option>
								<option value="Black">Black</option>
								<option value="Red">Red</option>
								<option value="Other">Other</option>
							</select>
						</div>

						<!-- Eye Color -->
						<div class="mb-3">
							<label class="form-label">Eye Color</label>
							<select v-model="formData.eyeColor" class="form-select">
								<option value="">Select...</option>
								<option value="Blue">Blue</option>
								<option value="Brown">Brown</option>
								<option value="Green">Green</option>
								<option value="Hazel">Hazel</option>
								<option value="Other">Other</option>
							</select>
						</div>

						<!-- Bio -->
						<div class="mb-3">
							<label class="form-label">Bio</label>
							<textarea v-model="formData.bio" class="form-control" rows="3" placeholder="Brief biography..."></textarea>
						</div>

						<!-- Social Links -->
						<div class="mb-3">
							<label class="form-label">Social Links</label>
							<div class="input-group mb-2">
								<span class="input-group-text">
									<font-awesome-icon :icon="['fab', 'twitter']" />
								</span>
								<input v-model="formData.twitter" type="text" class="form-control" placeholder="Twitter username" />
							</div>
							<div class="input-group mb-2">
								<span class="input-group-text">
									<font-awesome-icon :icon="['fab', 'instagram']" />
								</span>
								<input v-model="formData.instagram" type="text" class="form-control" placeholder="Instagram username" />
							</div>
						</div>
					</form>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" @click="close">Cancel</button>
					<button type="button" class="btn btn-primary" @click="save">
						<font-awesome-icon :icon="['fas', 'save']" />
						Create Performer
					</button>
				</div>
			</div>
		</div>
	</div>
	<div v-if="show" class="modal-backdrop show"></div>
</template>

<script>
import { performersAPI } from '@/services/api'

export default {
	name: 'AddPerformerModal',
	props: {
		show: {
			type: Boolean,
			default: false,
		},
	},
	emits: ['close', 'performer-added'],
	data() {
		return {
			formData: {
				name: '',
				birthdate: '',
				country: '',
				height: null,
				weight: null,
				measurements: '',
				hairColor: '',
				eyeColor: '',
				bio: '',
				twitter: '',
				instagram: '',
			},
			imageFile: null,
			imagePreview: null,
		}
	},
	methods: {
		handleImageUpload(event) {
			const file = event.target.files[0]
			if (!file) return

			// Validate file size (10MB)
			if (file.size > 10 * 1024 * 1024) {
				this.$toast.error('Image size must be less than 10MB')
				return
			}

			// Validate file type
			if (!file.type.startsWith('image/')) {
				this.$toast.error('Please upload an image file')
				return
			}

			this.imageFile = file

			// Create preview
			const reader = new FileReader()
			reader.onload = (e) => {
				this.imagePreview = e.target.result
			}
			reader.readAsDataURL(file)
		},
		removeImage() {
			this.imageFile = null
			this.imagePreview = null
			if (this.$refs.imageInput) {
				this.$refs.imageInput.value = ''
			}
		},
		async save() {
			if (!this.formData.name.trim()) {
				this.$toast.error('Performer name is required')
				return
			}

			try {
				// Create metadata object
				const metadata = {
					birthdate: this.formData.birthdate || undefined,
					country: this.formData.country || undefined,
					height: this.formData.height || undefined,
					weight: this.formData.weight || undefined,
					measurements: this.formData.measurements || undefined,
					hair_color: this.formData.hairColor || undefined,
					eye_color: this.formData.eyeColor || undefined,
					bio: this.formData.bio || undefined,
					twitter: this.formData.twitter || undefined,
					instagram: this.formData.instagram || undefined,
				}

				// Create FormData for multipart upload
				const formData = new FormData()
				formData.append('name', this.formData.name)
				formData.append('metadata', JSON.stringify(metadata))

				if (this.imageFile) {
					formData.append('image', this.imageFile)
				}

				const response = await performersAPI.create(formData)
				this.$toast.success('Performer created successfully')
				this.$emit('performer-added', response.data)
				this.resetForm()
			} catch (error) {
				console.error('Failed to create performer:', error)
				this.$toast.error('Failed to create performer: ' + (error.response?.data?.error || error.message))
			}
		},
		close() {
			this.resetForm()
			this.$emit('close')
		},
		resetForm() {
			this.formData = {
				name: '',
				birthdate: '',
				country: '',
				height: null,
				weight: null,
				measurements: '',
				hairColor: '',
				eyeColor: '',
				bio: '',
				twitter: '',
				instagram: '',
			}
			this.removeImage()
		},
	},
}
</script>

<style scoped>
.modal-content {
	background: #1a1a2e;
	border: 1px solid #2d2d44;
	max-height: 90vh;
	overflow-y: auto;
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

.image-upload-area {
	border: 2px dashed #2d2d44;
	border-radius: 8px;
	overflow: hidden;
	background: #16213e;
}

.upload-placeholder {
	padding: 2rem;
	text-align: center;
	cursor: pointer;
	transition: all 0.2s;
}

.upload-placeholder:hover {
	background: #1a1a2e;
	border-color: #667eea;
}

.image-preview {
	position: relative;
	width: 100%;
	height: 300px;
}

.image-preview img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.remove-image {
	position: absolute;
	top: 10px;
	right: 10px;
	width: 32px;
	height: 32px;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
}

.input-group-text {
	background: #16213e;
	border-color: #2d2d44;
	color: #667eea;
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
