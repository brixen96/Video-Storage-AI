<template>
	<div class="libraries-page">
		<div class="container-fluid">
			<div class="row">
				<div class="col d-flex justify-content-between align-items-center my-3">
					<span class="lead">Manage your video collections and directories</span>
					<div class="d-flex gap-2">
						<button v-if="libraries.length >= 2" class="btn btn-outline-warning" @click="openLinkFinder" title="Find matching Edited↔Backup video pairs">
							<font-awesome-icon :icon="['fas', 'link']" class="me-2" />
							Find Video Pairs
						</button>
						<button class="btn btn-primary" @click="showCreateModal = true">
							<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
							Add Library
						</button>
					</div>
				</div>
			</div>

			<!-- Loading State -->
			<LoadingState v-if="loading" spinner="fontawesome" show-text loading-text="Loading libraries..." />

			<!-- Libraries List -->
			<div v-else-if="libraries.length > 0" class="row g-4">
				<div v-for="library in libraries" :key="library.id" class="col-md-6 col-lg-4">
					<div class="library-card card h-100 text-primary">
						<div class="card-body">
							<div class="d-flex justify-content-between align-items-start mb-3">
								<h5 class="card-title mb-0">
									<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
									{{ library.name }}
									<span v-if="library.primary" class="badge bg-primary ms-2">Primary</span>
								</h5>
								<div class="btn-group btn-group-sm">
									<button class="btn btn-outline-secondary me-2 text-warning" @click="editLibrary(library)" title="Edit">
										<font-awesome-icon :icon="['fas', 'edit']" />
									</button>
									<button class="btn btn-outline-danger" @click="confirmDelete(library)" title="Delete">
										<font-awesome-icon :icon="['fas', 'trash']" />
									</button>
								</div>
							</div>

							<div class="library-path mb-3">
								<small class="">
									<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
									{{ library.path }}
								</small>
							</div>

							<div class="library-stats">
								<div class="stat-item">
									<font-awesome-icon :icon="['fas', 'video']" class="me-1" />
									<span>{{ library.video_count || 0 }} videos</span>
								</div>
							</div>
						</div>
						<div class="card-footer d-flex justify-content-between align-items-center">
							<small>Created {{ formatDate(library.created_at) }}</small>
							<button class="btn btn-sm btn-outline-info" @click="openAnalysis(library)" title="Analyze folder structure and auto-assign metadata">
								<font-awesome-icon :icon="['fas', 'sitemap']" class="me-1" />
								Analyze Structure
							</button>
						</div>
					</div>
				</div>
			</div>

			<!-- Empty State -->
			<EmptyState v-else :icon="['fas', 'folder']" icon-size="5x" title="No Libraries Yet" message="Get started by adding your first library">
				<template #actions>
					<button class="btn btn-primary btn-lg" @click="showCreateModal = true">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
						Add Your First Library
					</button>
				</template>
			</EmptyState>
		</div>

		<!-- Create/Edit Modal -->
		<div class="modal fade" :class="{ show: showCreateModal || showEditModal }" :style="{ display: showCreateModal || showEditModal ? 'block' : 'none' }" tabindex="-1">
			<div class="modal-dialog">
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							{{ showEditModal ? 'Edit Library' : 'Add New Library' }}
						</h5>
						<button type="button" class="btn-close" @click="closeModal"></button>
					</div>
					<div class="modal-body">
						<form @submit.prevent="saveLibrary">
							<div class="mb-3">
								<label for="libraryName" class="form-label">Library Name *</label>
								<input id="libraryName" v-model="form.name" type="text" class="form-control" placeholder="e.g., Movies, Personal Collection" required />
							</div>
							<div class="mb-3">
								<label for="libraryPath" class="form-label">Path *</label>
								<input id="libraryPath" v-model="form.path" type="text" class="form-control" placeholder="e.g., D:\Videos\Movies" required />
								<small class="">Full path to the directory containing your videos</small>
							</div>
							<div class="mb-3 form-check">
								<input id="libraryPrimary" v-model="form.primary" type="checkbox" class="form-check-input" />
								<label for="libraryPrimary" class="form-check-label">
									Set as primary library
									<small class="d-block text-muted">The primary library will be loaded by default in the browser</small>
								</label>
							</div>
						</form>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="closeModal">Cancel</button>
						<button type="button" class="btn btn-primary" @click="saveLibrary" :disabled="saving">
							<font-awesome-icon v-if="saving" :icon="['fas', 'spinner']" spin class="me-2" />
							{{ saving ? 'Saving...' : 'Save Library' }}
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Modal Backdrop (create/edit) -->
		<div v-if="showCreateModal || showEditModal" class="modal-backdrop fade show" @click="closeModal"></div>

		<!-- ── Video Pair Finder Modal ── -->
		<div v-if="showLinkModal" class="modal fade show" style="display:block" tabindex="-1">
			<div class="modal-dialog modal-xl modal-dialog-scrollable">
				<div class="modal-content analysis-modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'link']" class="me-2 text-warning" />
							Find Video Pairs — Edited ↔ Backup
						</h5>
						<button type="button" class="btn-close" @click="closeLinkFinder"></button>
					</div>
					<div class="modal-body">
						<!-- Library selector -->
						<div v-if="!linkSuggestions" class="row g-3 mb-4">
							<div class="col-md-5">
								<label class="form-label text-warning fw-bold">Edited Library</label>
								<select v-model="linkEditedLibID" class="form-select form-select-dark">
									<option value="">— select —</option>
									<option v-for="lib in libraries" :key="lib.id" :value="lib.id">{{ lib.name }}</option>
								</select>
							</div>
							<div class="col-md-2 d-flex align-items-end justify-content-center pb-1">
								<font-awesome-icon :icon="['fas', 'link']" class="text-muted fs-4" />
							</div>
							<div class="col-md-5">
								<label class="form-label text-info fw-bold">Backup Library</label>
								<select v-model="linkBackupLibID" class="form-select form-select-dark">
									<option value="">— select —</option>
									<option v-for="lib in libraries" :key="lib.id" :value="lib.id">{{ lib.name }}</option>
								</select>
							</div>
						</div>

						<!-- Suggestion results -->
						<div v-if="linkSuggestions">
							<!-- Tab bar: suggested / confirmed / rejected -->
							<div class="d-flex justify-content-between align-items-center mb-3">
								<ul class="nav nav-tabs flex-grow-1">
									<li class="nav-item">
										<button class="nav-link" :class="{ active: linkTab === 'suggested' }" @click="switchLinkTab('suggested')">
											<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
											To Review
											<span class="badge bg-warning text-dark ms-1">{{ countByStatus('suggested') }}</span>
										</button>
									</li>
									<li class="nav-item">
										<button class="nav-link" :class="{ active: linkTab === 'confirmed' }" @click="switchLinkTab('confirmed')">
											<font-awesome-icon :icon="['fas', 'check']" class="me-1" />
											Confirmed
											<span class="badge bg-success ms-1">{{ countByStatus('confirmed') }}</span>
										</button>
									</li>
									<li class="nav-item">
										<button class="nav-link" :class="{ active: linkTab === 'rejected' }" @click="switchLinkTab('rejected')">
											<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
											Rejected
											<span class="badge bg-secondary ms-1">{{ countByStatus('rejected') }}</span>
										</button>
									</li>
								</ul>
								<button class="btn btn-sm btn-outline-secondary ms-3" @click="rerunLinkFinder" :disabled="linkRunning">
									<font-awesome-icon v-if="linkRunning" :icon="['fas', 'spinner']" spin class="me-1" />
									<font-awesome-icon v-else :icon="['fas', 'sync']" class="me-1" />
									Re-run
								</button>
							</div>

							<!-- Suggested table -->
							<div v-if="linkTab === 'suggested'">
								<div v-if="tabLinks.length === 0" class="text-center py-4 text-muted">
									<font-awesome-icon :icon="['fas', 'check-circle']" size="2x" class="mb-2 text-success" />
									<p class="mb-0">All pairs have been reviewed.</p>
								</div>
								<div v-else>
									<div class="d-flex gap-2 mb-2 justify-content-end">
										<button class="btn btn-sm btn-success" @click="bulkAction('confirmed')" :disabled="selectedLinks.length === 0">
											<font-awesome-icon :icon="['fas', 'check']" class="me-1" />
											Confirm {{ selectedLinks.length > 0 ? selectedLinks.length : '' }}
										</button>
										<button class="btn btn-sm btn-danger" @click="bulkAction('rejected')" :disabled="selectedLinks.length === 0">
											<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
											Reject {{ selectedLinks.length > 0 ? selectedLinks.length : '' }}
										</button>
										<button class="btn btn-sm btn-outline-secondary" @click="toggleSelectAllLinks">
											{{ selectedLinks.length === tabLinks.length ? 'Deselect All' : 'Select All' }}
										</button>
									</div>
									<div class="analysis-table-wrap">
										<table class="table table-sm table-dark table-hover mb-0">
											<thead>
												<tr>
													<th style="width:2rem"></th>
													<th>Edited Video</th>
													<th>Backup Video</th>
													<th style="width:5rem">Score</th>
													<th style="width:6rem">Match</th>
													<th style="width:7rem"></th>
												</tr>
											</thead>
											<tbody>
												<tr v-for="link in tabLinks" :key="link.id">
													<td><input type="checkbox" class="form-check-input" :value="link.id" v-model="selectedLinks" /></td>
													<td>
														<div class="text-truncate" style="max-width:220px" :title="link.edited_video?.title">{{ link.edited_video?.title }}</div>
														<small class="text-muted">{{ formatDuration(link.edited_video?.duration) }}</small>
													</td>
													<td>
														<div class="text-truncate" style="max-width:220px" :title="link.backup_video?.title">{{ link.backup_video?.title }}</div>
														<small class="text-muted">{{ formatDuration(link.backup_video?.duration) }}</small>
													</td>
													<td>
														<span class="badge" :class="confidenceBadge(link.confidence)">{{ (link.confidence * 100).toFixed(0) }}%</span>
													</td>
													<td>
														<span v-for="r in parseReasons(link.match_reasons)" :key="r" class="badge bg-secondary me-1" style="font-size:0.65rem">{{ r }}</span>
													</td>
													<td class="text-end">
														<button class="btn btn-xs btn-success me-1" @click="updateSingleLink(link.id, 'confirmed')" title="Confirm">
															<font-awesome-icon :icon="['fas', 'check']" />
														</button>
														<button class="btn btn-xs btn-danger" @click="updateSingleLink(link.id, 'rejected')" title="Reject">
															<font-awesome-icon :icon="['fas', 'times']" />
														</button>
													</td>
												</tr>
											</tbody>
										</table>
									</div>
								</div>
							</div>

							<!-- Confirmed / Rejected tables -->
							<div v-if="linkTab === 'confirmed' || linkTab === 'rejected'">
								<div v-if="tabLinks.length === 0" class="text-center py-4 text-muted">No {{ linkTab }} pairs yet.</div>
								<div v-else class="analysis-table-wrap">
									<table class="table table-sm table-dark table-hover mb-0">
										<thead>
											<tr>
												<th>Edited Video</th>
												<th>Backup Video</th>
												<th style="width:5rem">Score</th>
												<th style="width:7rem"></th>
											</tr>
										</thead>
										<tbody>
											<tr v-for="link in tabLinks" :key="link.id">
												<td class="text-truncate" style="max-width:260px" :title="link.edited_video?.title">{{ link.edited_video?.title }}</td>
												<td class="text-truncate" style="max-width:260px" :title="link.backup_video?.title">{{ link.backup_video?.title }}</td>
												<td>
													<span class="badge" :class="confidenceBadge(link.confidence)">{{ (link.confidence * 100).toFixed(0) }}%</span>
												</td>
												<td class="text-end">
													<button
														v-if="linkTab === 'confirmed'"
														class="btn btn-xs btn-outline-danger"
														@click="updateSingleLink(link.id, 'rejected')"
														title="Undo confirmation"
													>Undo</button>
													<button
														v-else
														class="btn btn-xs btn-outline-success"
														@click="updateSingleLink(link.id, 'suggested')"
														title="Move back to review"
													>Restore</button>
												</td>
											</tr>
										</tbody>
									</table>
								</div>
							</div>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="closeLinkFinder">Close</button>
						<button
							v-if="!linkSuggestions"
							class="btn btn-warning"
							@click="runLinkFinder"
							:disabled="!linkEditedLibID || !linkBackupLibID || linkEditedLibID === linkBackupLibID || linkRunning"
						>
							<font-awesome-icon v-if="linkRunning" :icon="['fas', 'spinner']" spin class="me-2" />
							<font-awesome-icon v-else :icon="['fas', 'search']" class="me-2" />
							{{ linkRunning ? 'Searching…' : 'Find Pairs' }}
						</button>
					</div>
				</div>
			</div>
		</div>
		<div v-if="showLinkModal" class="modal-backdrop fade show" @click="closeLinkFinder"></div>

		<!-- ── Structure Analysis Modal ── -->
		<div v-if="showAnalysisModal" class="modal fade show" style="display:block" tabindex="-1">
			<div class="modal-dialog modal-xl modal-dialog-scrollable">
				<div class="modal-content analysis-modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'sitemap']" class="me-2 text-info" />
							Structure Analysis — <span class="text-info">{{ analysisLibrary?.name }}</span>
						</h5>
						<button type="button" class="btn-close" @click="closeAnalysis"></button>
					</div>

					<!-- Loading -->
					<div v-if="analysisLoading" class="modal-body text-center py-5">
						<font-awesome-icon :icon="['fas', 'spinner']" spin size="3x" class="text-info mb-3" />
						<p class="text-muted">Scanning folder structure…</p>
					</div>

					<!-- Results -->
					<div v-else-if="analysisPlan" class="modal-body p-0">
						<!-- Stats bar -->
						<div class="analysis-stats-bar d-flex gap-3 flex-wrap px-4 py-3 border-bottom">
							<div class="stat-pill">
								<span class="stat-num">{{ analysisPlan.stats.total_videos }}</span>
								<span class="stat-label">Videos scanned</span>
							</div>
							<div class="stat-pill text-warning">
								<span class="stat-num">{{ analysisPlan.stats.content_type_will_change }}</span>
								<span class="stat-label">Content type changes</span>
							</div>
							<div class="stat-pill text-success">
								<span class="stat-num">{{ analysisPlan.stats.tags_will_add }}</span>
								<span class="stat-label">Tags to assign</span>
							</div>
							<div class="stat-pill text-info">
								<span class="stat-num">{{ analysisPlan.stats.performers_will_link }}</span>
								<span class="stat-label">Performers to link</span>
							</div>
							<div v-if="analysisPlan.stats.videos_unknown_root > 0" class="stat-pill text-muted">
								<span class="stat-num">{{ analysisPlan.stats.videos_unknown_root }}</span>
								<span class="stat-label">Unknown root folder</span>
							</div>
						</div>

						<!-- Nothing to do -->
						<div v-if="nothingToDo" class="text-center py-5 text-muted">
							<font-awesome-icon :icon="['fas', 'check-circle']" size="3x" class="mb-3 text-success" />
							<p class="mb-0">Everything is already up to date. No changes needed.</p>
						</div>

						<!-- Tabs -->
						<div v-else class="px-4 pt-3">
							<ul class="nav nav-tabs mb-3">
								<li class="nav-item">
									<button
										class="nav-link"
										:class="{ active: analysisTab === 'types' }"
										@click="analysisTab = 'types'"
										:disabled="!analysisPlan.content_type_changes?.length"
									>
										<font-awesome-icon :icon="['fas', 'tags']" class="me-1" />
										Content Types
										<span class="badge bg-warning text-dark ms-1">{{ analysisPlan.content_type_changes?.length || 0 }}</span>
									</button>
								</li>
								<li class="nav-item">
									<button
										class="nav-link"
										:class="{ active: analysisTab === 'tags' }"
										@click="analysisTab = 'tags'"
										:disabled="!analysisPlan.tag_assignments?.length"
									>
										<font-awesome-icon :icon="['fas', 'tag']" class="me-1" />
										Tags
										<span class="badge bg-success ms-1">{{ analysisPlan.tag_assignments?.length || 0 }}</span>
									</button>
								</li>
								<li class="nav-item">
									<button
										class="nav-link"
										:class="{ active: analysisTab === 'performers' }"
										@click="analysisTab = 'performers'"
										:disabled="!analysisPlan.performer_links?.length"
									>
										<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
										Performers
										<span class="badge bg-info ms-1">{{ analysisPlan.performer_links?.length || 0 }}</span>
									</button>
								</li>
							</ul>

							<!-- Content Type Changes tab -->
							<div v-if="analysisTab === 'types'">
								<div class="d-flex justify-content-between align-items-center mb-2">
									<small class="text-muted">Select which content type changes to apply</small>
									<div class="d-flex gap-2">
										<button class="btn btn-sm btn-outline-secondary" @click="selectAll('types')">Select All</button>
										<button class="btn btn-sm btn-outline-secondary" @click="selectNone('types')">Deselect All</button>
									</div>
								</div>
								<div class="analysis-table-wrap">
									<table class="table table-sm table-dark table-hover mb-0">
										<thead>
											<tr>
												<th style="width:2rem"></th>
												<th>Title</th>
												<th>Current Type</th>
												<th>New Type</th>
											</tr>
										</thead>
										<tbody>
											<tr v-for="(item, i) in analysisPlan.content_type_changes" :key="i">
												<td>
													<input type="checkbox" v-model="selectedTypes[i]" class="form-check-input" />
												</td>
												<td class="text-truncate" style="max-width:320px" :title="item.title">{{ item.title }}</td>
												<td><span class="badge bg-secondary">{{ item.old_type }}</span></td>
												<td><span class="badge bg-warning text-dark">{{ item.new_type }}</span></td>
											</tr>
										</tbody>
									</table>
								</div>
							</div>

							<!-- Tag Assignments tab -->
							<div v-if="analysisTab === 'tags'">
								<div class="d-flex justify-content-between align-items-center mb-2">
									<small class="text-muted">Select which tag assignments to apply</small>
									<div class="d-flex gap-2">
										<button class="btn btn-sm btn-outline-secondary" @click="selectAll('tags')">Select All</button>
										<button class="btn btn-sm btn-outline-secondary" @click="selectNone('tags')">Deselect All</button>
									</div>
								</div>
								<div class="analysis-table-wrap">
									<table class="table table-sm table-dark table-hover mb-0">
										<thead>
											<tr>
												<th style="width:2rem"></th>
												<th>Title</th>
												<th>Tag</th>
											</tr>
										</thead>
										<tbody>
											<tr v-for="(item, i) in analysisPlan.tag_assignments" :key="i">
												<td>
													<input type="checkbox" v-model="selectedTags[i]" class="form-check-input" />
												</td>
												<td class="text-truncate" style="max-width:380px" :title="item.title">{{ item.title }}</td>
												<td><span class="badge bg-success">{{ item.tag_name }}</span></td>
											</tr>
										</tbody>
									</table>
								</div>
							</div>

							<!-- Performer Links tab -->
							<div v-if="analysisTab === 'performers'">
								<div class="d-flex justify-content-between align-items-center mb-2">
									<small class="text-muted">Select which performer links to apply</small>
									<div class="d-flex gap-2">
										<button class="btn btn-sm btn-outline-secondary" @click="selectAll('performers')">Select All</button>
										<button class="btn btn-sm btn-outline-secondary" @click="selectNone('performers')">Deselect All</button>
									</div>
								</div>
								<div class="analysis-table-wrap">
									<table class="table table-sm table-dark table-hover mb-0">
										<thead>
											<tr>
												<th style="width:2rem"></th>
												<th>Title</th>
												<th>Performer</th>
											</tr>
										</thead>
										<tbody>
											<tr v-for="(item, i) in analysisPlan.performer_links" :key="i">
												<td>
													<input type="checkbox" v-model="selectedPerformers[i]" class="form-check-input" />
												</td>
												<td class="text-truncate" style="max-width:380px" :title="item.title">{{ item.title }}</td>
												<td><span class="badge bg-info">{{ item.performer_name }}</span></td>
											</tr>
										</tbody>
									</table>
								</div>
							</div>
						</div>
					</div>

					<!-- Apply result message -->
					<div v-if="applyResult" class="px-4 py-2">
						<div class="alert alert-success mb-0 py-2">
							<font-awesome-icon :icon="['fas', 'check']" class="me-2" />
							Applied: <strong>{{ applyResult.content_type_changed }}</strong> content type changes,
							<strong>{{ applyResult.tags_added }}</strong> tags,
							<strong>{{ applyResult.performers_linked }}</strong> performers.
						</div>
					</div>

					<div class="modal-footer">
						<div class="me-auto text-muted small" v-if="selectedCount > 0">
							{{ selectedCount }} change{{ selectedCount !== 1 ? 's' : '' }} selected
						</div>
						<button type="button" class="btn btn-secondary" @click="closeAnalysis">Close</button>
						<button
							v-if="!nothingToDo"
							type="button"
							class="btn btn-primary"
							@click="applyPlan"
							:disabled="applying || selectedCount === 0"
						>
							<font-awesome-icon v-if="applying" :icon="['fas', 'spinner']" spin class="me-2" />
							{{ applying ? 'Applying…' : `Apply ${selectedCount} Change${selectedCount !== 1 ? 's' : ''}` }}
						</button>
					</div>
				</div>
			</div>
		</div>
		<div v-if="showAnalysisModal" class="modal-backdrop fade show" @click="closeAnalysis"></div>
	</div>
</template>

<script>
import { librariesAPI, videoLinksAPI } from '@/services/api'
import { LoadingState, EmptyState } from '@/components/shared'

export default {
	name: 'LibrariesPage',
	components: {
		LoadingState,
		EmptyState,
	},
	data() {
		return {
			libraries: [],
			loading: false,
			saving: false,
			showCreateModal: false,
			showEditModal: false,
			editingLibrary: null,
			form: {
				name: '',
				path: '',
				primary: false,
			},
			// Link Finder
			showLinkModal: false,
			linkEditedLibID: '',
			linkBackupLibID: '',
			linkRunning: false,
			linkSuggestions: null, // all links fetched from backend
			linkTab: 'suggested',
			selectedLinks: [],
			// Analysis
			showAnalysisModal: false,
			analysisLibrary: null,
			analysisLoading: false,
			analysisPlan: null,
			analysisTab: 'types',
			selectedTypes: [],
			selectedTags: [],
			selectedPerformers: [],
			applying: false,
			applyResult: null,
		}
	},
	computed: {
		tabLinks() {
			if (!this.linkSuggestions) return []
			return this.linkSuggestions.filter(l => l.status === this.linkTab)
		},
		nothingToDo() {
			if (!this.analysisPlan) return true
			return (
				!this.analysisPlan.content_type_changes?.length &&
				!this.analysisPlan.tag_assignments?.length &&
				!this.analysisPlan.performer_links?.length
			)
		},
		selectedCount() {
			const types = this.selectedTypes.filter(Boolean).length
			const tags = this.selectedTags.filter(Boolean).length
			const perfs = this.selectedPerformers.filter(Boolean).length
			return types + tags + perfs
		},
	},
	async mounted() {
		await this.loadLibraries()
	},
	onActivated() {
		this.loadLibraries()
	},
	methods: {
		async loadLibraries() {
			this.loading = true
			try {
				const response = await librariesAPI.getAll()
				this.libraries = response.data || []
			} catch (error) {
				console.error('Failed to load libraries:', error)
				alert('Failed to load libraries. Please try again.')
			} finally {
				this.loading = false
			}
		},
		editLibrary(library) {
			this.editingLibrary = library
			this.form.name = library.name
			this.form.path = library.path
			this.form.primary = library.primary || false
			this.showEditModal = true
		},
		async saveLibrary() {
			if (!this.form.name || !this.form.path) {
				alert('Please fill in all required fields')
				return
			}

			this.saving = true
			try {
				if (this.showEditModal && this.editingLibrary) {
					await librariesAPI.update(this.editingLibrary.id, this.form)
				} else {
					await librariesAPI.create(this.form)
				}

				await this.loadLibraries()
				this.closeModal()
			} catch (error) {
				console.error('Failed to save library:', error)
				alert(error.response?.data?.error || 'Failed to save library. Please check the path and try again.')
			} finally {
				this.saving = false
			}
		},
		confirmDelete(library) {
			if (confirm(`Are you sure you want to delete "${library.name}"?\n\nThis will not delete any files.`)) {
				this.deleteLibrary(library.id)
			}
		},
		async deleteLibrary(id) {
			try {
				await librariesAPI.delete(id)
				await this.loadLibraries()
			} catch (error) {
				console.error('Failed to delete library:', error)
				alert('Failed to delete library. Please try again.')
			}
		},
		closeModal() {
			this.showCreateModal = false
			this.showEditModal = false
			this.editingLibrary = null
			this.form = { name: '', path: '', primary: false }
		},
		formatDate(dateString) {
			const date = new Date(dateString)
			return date.toLocaleDateString()
		},

		// ── Video Pair Finder ────────────────────────────────────────────
		openLinkFinder() {
			this.showLinkModal = true
			this.linkSuggestions = null
			this.linkTab = 'suggested'
			this.selectedLinks = []
			// Auto-select by name if possible
			const edited = this.libraries.find(l => l.name === 'Edited')
			const backup = this.libraries.find(l => l.name === 'Backup')
			this.linkEditedLibID = edited?.id || ''
			this.linkBackupLibID = backup?.id || ''
		},
		async runLinkFinder() {
			this.linkRunning = true
			try {
				await videoLinksAPI.suggest(this.linkEditedLibID, this.linkBackupLibID)
				await this.fetchLinkSuggestions()
			} catch (err) {
				console.error('Link finder failed:', err)
				alert(err.response?.data?.error || 'Failed to find pairs.')
			} finally {
				this.linkRunning = false
			}
		},
		async rerunLinkFinder() {
			this.linkRunning = true
			try {
				await videoLinksAPI.suggest(this.linkEditedLibID, this.linkBackupLibID)
				await this.fetchLinkSuggestions()
			} catch (err) {
				console.error('Re-run failed:', err)
			} finally {
				this.linkRunning = false
			}
		},
		async fetchLinkSuggestions() {
			const res = await videoLinksAPI.getLinks(this.linkEditedLibID)
			this.linkSuggestions = res.data?.links ?? res?.links ?? []
			this.selectedLinks = []
		},
		async switchLinkTab(tab) {
			this.linkTab = tab
			this.selectedLinks = []
		},
		countByStatus(status) {
			if (!this.linkSuggestions) return 0
			return this.linkSuggestions.filter(l => l.status === status).length
		},
		toggleSelectAllLinks() {
			if (this.selectedLinks.length === this.tabLinks.length) {
				this.selectedLinks = []
			} else {
				this.selectedLinks = this.tabLinks.map(l => l.id)
			}
		},
		async bulkAction(status) {
			if (this.selectedLinks.length === 0) return
			try {
				await videoLinksAPI.bulkUpdateStatus(this.selectedLinks, status)
				this.selectedLinks.forEach(id => {
					const link = this.linkSuggestions.find(l => l.id === id)
					if (link) link.status = status
				})
				this.selectedLinks = []
			} catch (err) {
				console.error('Bulk action failed:', err)
			}
		},
		async updateSingleLink(id, status) {
			try {
				await videoLinksAPI.updateStatus(id, status)
				const link = this.linkSuggestions.find(l => l.id === id)
				if (link) link.status = status
				this.selectedLinks = this.selectedLinks.filter(sid => sid !== id)
			} catch (err) {
				console.error('Update link failed:', err)
			}
		},
		closeLinkFinder() {
			this.showLinkModal = false
		},
		parseReasons(json) {
			try { return JSON.parse(json) } catch { return [] }
		},
		confidenceBadge(conf) {
			if (conf >= 0.8) return 'bg-success'
			if (conf >= 0.55) return 'bg-warning text-dark'
			return 'bg-secondary'
		},
		formatDuration(seconds) {
			if (!seconds) return '—'
			const m = Math.floor(seconds / 60)
			const s = Math.floor(seconds % 60)
			return `${m}:${String(s).padStart(2, '0')}`
		},

		// ── Structure Analysis ──────────────────────────────────────────
		async openAnalysis(library) {
			this.analysisLibrary = library
			this.showAnalysisModal = true
			this.analysisPlan = null
			this.applyResult = null
			this.analysisLoading = true
			this.analysisTab = 'types'
			try {
				const res = await librariesAPI.analyzeStructure(library.id)
				this.analysisPlan = res.data || res
				this.initSelections()
				// Jump to first tab that has data
				if (this.analysisPlan.content_type_changes?.length) this.analysisTab = 'types'
				else if (this.analysisPlan.tag_assignments?.length) this.analysisTab = 'tags'
				else if (this.analysisPlan.performer_links?.length) this.analysisTab = 'performers'
			} catch (err) {
				console.error('Analysis failed:', err)
				alert(err.response?.data?.error || 'Failed to analyze library structure.')
				this.showAnalysisModal = false
			} finally {
				this.analysisLoading = false
			}
		},
		initSelections() {
			this.selectedTypes = (this.analysisPlan.content_type_changes || []).map(() => true)
			this.selectedTags = (this.analysisPlan.tag_assignments || []).map(() => true)
			this.selectedPerformers = (this.analysisPlan.performer_links || []).map(() => true)
		},
		selectAll(type) {
			if (type === 'types') this.selectedTypes = this.selectedTypes.map(() => true)
			else if (type === 'tags') this.selectedTags = this.selectedTags.map(() => true)
			else this.selectedPerformers = this.selectedPerformers.map(() => true)
		},
		selectNone(type) {
			if (type === 'types') this.selectedTypes = this.selectedTypes.map(() => false)
			else if (type === 'tags') this.selectedTags = this.selectedTags.map(() => false)
			else this.selectedPerformers = this.selectedPerformers.map(() => false)
		},
		async applyPlan() {
			if (this.selectedCount === 0) return
			this.applying = true
			this.applyResult = null
			try {
				const payload = {
					library_id: this.analysisLibrary.id,
					content_type_changes: (this.analysisPlan.content_type_changes || []).filter((_, i) => this.selectedTypes[i]),
					tag_assignments: (this.analysisPlan.tag_assignments || []).filter((_, i) => this.selectedTags[i]),
					performer_links: (this.analysisPlan.performer_links || []).filter((_, i) => this.selectedPerformers[i]),
				}
				const res = await librariesAPI.applyStructure(this.analysisLibrary.id, payload)
				this.applyResult = res.data || res
				// Remove applied items from the plan so they won't appear again
				this.analysisPlan.content_type_changes = (this.analysisPlan.content_type_changes || []).filter((_, i) => !this.selectedTypes[i])
				this.analysisPlan.tag_assignments = (this.analysisPlan.tag_assignments || []).filter((_, i) => !this.selectedTags[i])
				this.analysisPlan.performer_links = (this.analysisPlan.performer_links || []).filter((_, i) => !this.selectedPerformers[i])
				this.initSelections()
			} catch (err) {
				console.error('Apply failed:', err)
				alert(err.response?.data?.error || 'Failed to apply changes.')
			} finally {
				this.applying = false
			}
		},
		closeAnalysis() {
			this.showAnalysisModal = false
			this.analysisLibrary = null
			this.analysisPlan = null
			this.applyResult = null
		},
	},
}
</script>

<style scoped>
@import '@/styles/pages/libraries_page.css';

.analysis-modal-content {
	background: #1a1a2e;
	color: #fff;
	border: 1px solid rgba(0, 217, 255, 0.3);
	max-height: 90vh;
}

.analysis-stats-bar {
	background: rgba(0, 0, 0, 0.3);
}

.stat-pill {
	display: flex;
	flex-direction: column;
	align-items: center;
	min-width: 90px;
	background: rgba(255, 255, 255, 0.05);
	border-radius: 8px;
	padding: 0.4rem 0.75rem;
}

.stat-num {
	font-size: 1.4rem;
	font-weight: 700;
	line-height: 1;
}

.stat-label {
	font-size: 0.7rem;
	opacity: 0.7;
	text-align: center;
}

.analysis-table-wrap {
	max-height: 360px;
	overflow-y: auto;
	border-radius: 6px;
	border: 1px solid rgba(255, 255, 255, 0.1);
}

.nav-tabs {
	border-bottom-color: rgba(255, 255, 255, 0.15);
}

.nav-tabs .nav-link {
	color: rgba(255, 255, 255, 0.6);
	border-color: transparent;
}

.nav-tabs .nav-link.active {
	background: rgba(0, 217, 255, 0.15);
	border-color: rgba(0, 217, 255, 0.4);
	color: #fff;
}

.nav-tabs .nav-link:disabled {
	color: rgba(255, 255, 255, 0.25);
	pointer-events: none;
}

.table-dark {
	--bs-table-bg: transparent;
}

.table-dark thead th {
	color: rgba(255, 255, 255, 0.5);
	font-size: 0.75rem;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	border-bottom-color: rgba(255, 255, 255, 0.1);
}

.form-select-dark {
	background-color: rgba(255, 255, 255, 0.08);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: #fff;
}

.form-select-dark option {
	background: #1a1a2e;
	color: #fff;
}

.btn-xs {
	padding: 0.15rem 0.45rem;
	font-size: 0.75rem;
	border-radius: 4px;
}
</style>
