<template>
	<div class="scraper-page">
		<div class="container-fluid py-3">
			<!-- Tabs for different scrapers -->
			<div class="scraper-tabs mb-3">
				<!-- Simpcity Tab (Always present) -->
				<button class="scraper-tab" :class="{ active: activeTab === 'simpcity' }" @click="activeTab = 'simpcity'">
					<font-awesome-icon :icon="['fas', 'comments']" class="me-2" />
					Simpcity Forum
				</button>

				<!-- PornHub Tab (Always present) -->
				<button class="scraper-tab pornhub-tab" :class="{ active: activeTab === 'pornhub' }" @click="activeTab = 'pornhub'">
					<font-awesome-icon :icon="['fas', 'fire']" class="me-2" />
					PornHub
				</button>

				<!-- LuxureTV Tab (Always present) -->
				<button class="scraper-tab luxuretv-tab" :class="{ active: activeTab === 'luxuretv' }" @click="activeTab = 'luxuretv'">
					<font-awesome-icon :icon="['fas', 'paw']" class="me-2" />
					LuxureTV
				</button>

				<!-- Sexuria Tab (Always present) -->
				<button class="scraper-tab sexuria-tab" :class="{ active: activeTab === 'sexuria' }" @click="activeTab = 'sexuria'">
					<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
					Sexuria
				</button>

				<!-- Rule34Video Tab (Always present) -->
				<button class="scraper-tab rule34video-tab" :class="{ active: activeTab === 'rule34video' }" @click="activeTab = 'rule34video'">
					<font-awesome-icon :icon="['fas', 'cube']" class="me-2" />
					Rule34Videos
				</button>

				<!-- Motherless Tab -->
				<button class="scraper-tab motherless-tab" :class="{ active: activeTab === 'motherless' }" @click="activeTab = 'motherless'">
					<font-awesome-icon :icon="['fas', 'film']" class="me-2" />
					Motherless
				</button>

				<!-- HQPorner Tab -->
				<button class="scraper-tab hqporner-tab" :class="{ active: activeTab === 'hqporner' }" @click="activeTab = 'hqporner'">
					<font-awesome-icon :icon="['fas', 'bolt']" class="me-2" />
					HQPorner
				</button>

				<!-- LeakVids Tab -->
				<button class="scraper-tab leakvids-tab" :class="{ active: activeTab === 'leakvids' }" @click="activeTab = 'leakvids'">
					<font-awesome-icon :icon="['fas', 'eye-slash']" class="me-2" />
					LeakVids
				</button>

				<!-- Dynamic Custom Tabs -->
				<button v-for="tab in scraperTabs" :key="tab.id" class="scraper-tab" :class="{ active: activeTab === `tab-${tab.id}` }" :style="{ borderColor: tab.color }" @click="selectTab(tab.id)">
					<font-awesome-icon :icon="['fas', tab.icon]" class="me-2" />
					{{ tab.name }}
					<button class="btn-close-tab" @click.stop="confirmDeleteTab(tab)" title="Delete tab">
						<font-awesome-icon :icon="['fas', 'times']" />
					</button>
				</button>

				<!-- Add New Tab Button -->
				<button class="scraper-tab add-tab" @click="showAddTabModal = true">
					<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
					Add Tab
				</button>
			</div>

			<!-- Simpcity Tab Content -->
			<div v-if="activeTab === 'simpcity'">
				<SimpcityTab />
			</div>
			<!-- End Simpcity Tab Content (modals are shared between tabs) -->

			<!-- PornHub Tab Content -->
			<div v-if="activeTab === 'pornhub'" class="tab-content">
				<PornHubTab />
			</div>
			<!-- End PornHub Tab Content -->

			<!-- LuxureTV Tab Content -->
			<div v-if="activeTab === 'luxuretv'" class="tab-content">
				<LuxureTVTab />
			</div>
			<!-- End LuxureTV Tab Content -->

			<!-- Sexuria Tab Content -->
			<div v-if="activeTab === 'sexuria'" class="tab-content">
				<SexuriaTab />
			</div>
			<!-- End Sexuria Tab Content -->

			<!-- Rule34Video Tab Content -->
			<div v-if="activeTab === 'rule34video'" class="tab-content">
				<Rule34VideoTab />
			</div>
			<!-- End Rule34Video Tab Content -->
			<!-- Motherless Tab Content -->
			<div v-if="activeTab === 'motherless'" class="tab-content">
				<MotherlessTab />
			</div>

			<!-- HQPorner Tab Content -->
			<div v-if="activeTab === 'hqporner'" class="tab-content">
				<HQPornerTab />
			</div>

			<!-- LeakVids Tab Content -->
			<div v-if="activeTab === 'leakvids'" class="tab-content">
				<LeakVidsTab />
			</div>

			<!-- Videos Tab Content -->
			<div v-if="activeTab === 'videos'" class="tab-content">
				<div class="scraper-layout">
					<!-- Main Content (Left) -->
					<div class="scraper-main">
						<!-- Scraped Videos List -->
						<div v-if="scrapedVideos.length > 0">
							<div class="d-flex justify-content-between align-items-center mb-2">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'list']" class="me-2" />
									Scraped Videos ({{ scrapedVideos.length }})
								</h6>
								<button class="btn btn-sm btn-outline-danger" @click="clearAllVideos">
									<font-awesome-icon :icon="['fas', 'trash']" />
								</button>
							</div>
							<div class="table-responsive">
								<table class="table table-hover table-dark">
									<thead>
										<tr>
											<th>Title</th>
											<th>URL</th>
											<th>Duration</th>
											<th>Size</th>
											<th>Status</th>
											<th>Actions</th>
										</tr>
									</thead>
									<tbody>
										<tr v-for="video in scrapedVideos" :key="video.id">
											<td>{{ video.title }}</td>
											<td>
												<a :href="video.url" target="_blank" class="text-truncate d-inline-block" style="max-width: 200px">
													{{ video.url }}
												</a>
											</td>
											<td>{{ video.duration || 'N/A' }}</td>
											<td>{{ video.fileSize || 'N/A' }}</td>
											<td>
												<span class="badge" :class="video.status === 'completed' ? 'bg-success' : 'bg-warning'">
													{{ video.status }}
												</span>
											</td>
											<td>
												<button class="btn btn-sm btn-outline-primary" @click="viewVideo(video)">
													<font-awesome-icon :icon="['fas', 'eye']" />
												</button>
											</td>
										</tr>
									</tbody>
								</table>
							</div>
						</div>

						<!-- Empty State -->
						<EmptyState v-else :icon="['fas', 'video']" icon-size="4x" title="No Videos Scraped Yet" message="Use the sidebar controls to scrape videos from any site" />
					</div>
					<!-- End Main Content -->

					<!-- Sidebar (Right) -->
					<div class="scraper-sidebar">
						<!-- Scraper Controls -->
						<div class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									Video Scraper
								</h6>
							</div>
							<div class="card-body">
								<div class="mb-2">
									<label class="form-label small">URL to Scrape</label>
									<input v-model="videoUrl" type="text" class="form-control form-control-sm" placeholder="Enter video or page URL..." />
								</div>
								<div class="mb-2">
									<label class="form-label small">Site Type</label>
									<select v-model="videoSiteType" class="form-select form-select-sm">
										<option value="generic">Auto-Detect</option>
										<option value="youtube">YouTube</option>
										<option value="vimeo">Vimeo</option>
										<option value="custom">Custom Site</option>
									</select>
								</div>
								<div class="mb-3">
									<div class="form-check">
										<input v-model="videoScrapeOptions.extractMetadata" class="form-check-input" type="checkbox" id="extractMetadata" />
										<label class="form-check-label small" for="extractMetadata">Extract metadata</label>
									</div>
									<div class="form-check">
										<input v-model="videoScrapeOptions.scrapeThumbnails" class="form-check-input" type="checkbox" id="scrapeThumbnails" />
										<label class="form-check-label small" for="scrapeThumbnails">Download thumbnails</label>
									</div>
								</div>
								<div class="d-flex flex-column gap-2">
									<button class="btn btn-primary btn-sm w-100" @click="scrapeVideo" :disabled="!videoUrl || videoScraping">
										<font-awesome-icon :icon="['fas', videoScraping ? 'spinner' : 'play']" :spin="videoScraping" class="me-2" />
										{{ videoScraping ? 'Scraping...' : 'Start Scrape' }}
									</button>
									<button class="btn btn-outline-info btn-sm w-100" @click="testScraper" :disabled="!videoUrl">
										<font-awesome-icon :icon="['fas', 'vial']" class="me-2" />
										Test Scraper
									</button>
								</div>
							</div>
						</div>

						<!-- Statistics -->
						<div class="card">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />
									Statistics
								</h6>
							</div>
							<div class="card-body p-2">
								<div class="sidebar-stats-grid">
									<StatCard :value="scrapedVideos.length" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
									<StatCard :value="videoScraping ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
								</div>
							</div>
						</div>
					</div>
					<!-- End Sidebar -->
				</div>
				<!-- End scraper-layout -->
			</div>
			<!-- End Videos Tab Content -->

			<!-- Dynamic Custom Tab Content -->
			<div v-for="tab in scraperTabs" :key="`content-${tab.id}`">
				<div v-if="activeTab === `tab-${tab.id}`" class="tab-content">
					<div class="scraper-layout" :class="{ 'sidebar-collapsed': collapsedSections[`tab-${tab.id}`] }">
						<!-- Main Content (Left) -->
						<div class="scraper-main">
							<!-- Search and Filters -->
							<div class="filter-bar mb-3">
								<div class="row g-2 align-items-end">
									<div class="col-md-4">
										<input v-model="videoSearch[tab.id]" type="text" class="form-control form-control-sm" placeholder="Search videos..." @input="onTabSearchInput(tab.id)" />
									</div>
									<div class="col-md-3">
										<select v-model="videoSort[tab.id]" class="form-select form-select-sm" @change="onTabFilterChange(tab.id)">
											<option value="newest">Newest First</option>
											<option value="oldest">Oldest First</option>
											<option value="views-desc">Most Views</option>
											<option value="duration-desc">Longest</option>
											<option value="duration-asc">Shortest</option>
											<option value="title-asc">Title (A-Z)</option>
											<option value="title-desc">Title (Z-A)</option>
											<option value="likes-desc">Most Likes</option>
										</select>
									</div>
									<div class="col-md-2">
										<select v-model="videoDurationFilter[tab.id]" class="form-select form-select-sm" @change="onTabFilterChange(tab.id)">
											<option value="">Any Duration</option>
											<option value="short">Short (&lt;5 min)</option>
											<option value="medium">Medium (5–20 min)</option>
											<option value="long">Long (&gt;20 min)</option>
										</select>
									</div>
									<div class="col-md-3 d-flex gap-1">
										<button class="btn btn-sm btn-outline-secondary flex-grow-1" @click="toggleVideoFilters(tab.id)">
											<font-awesome-icon :icon="['fas', 'filter']" class="me-1" />
											More
											<span v-if="activeTabFilterCount(tab.id) > 0" class="badge bg-primary ms-1">{{ activeTabFilterCount(tab.id) }}</span>
										</button>
										<button class="btn btn-sm btn-outline-warning" @click="resetVideoFilters(tab.id)" title="Reset all filters">
											<font-awesome-icon :icon="['fas', 'redo']" />
										</button>
									</div>
								</div>
								<!-- Advanced Filters (Collapsible) -->
								<div v-if="showVideoFilters[tab.id]" class="mt-2 pt-2 border-top border-secondary">
									<div class="row g-2 align-items-center">
										<div class="col-md-4">
											<input v-model="videoAuthorFilter[tab.id]" type="text" class="form-control form-control-sm" placeholder="Author / uploader..." @input="onTabSearchInput(tab.id)" />
										</div>
										<div class="col-md-3">
											<input v-model.number="videoMinViews[tab.id]" type="number" min="0" class="form-control form-control-sm" placeholder="Min views (0 = any)" @change="onTabFilterChange(tab.id)" />
										</div>
										<div class="col-md-3 d-flex align-items-center">
											<div class="form-check mb-0">
												<input :id="`only-dl-${tab.id}`" v-model="videoOnlyDownloaded[tab.id]" class="form-check-input" type="checkbox" @change="onTabFilterChange(tab.id)" />
												<label :for="`only-dl-${tab.id}`" class="form-check-label small">Downloaded only</label>
											</div>
										</div>
									</div>
								</div>
							</div>

							<!-- Active Job Progress -->
							<div v-if="customTabJobs[tab.id]" class="card bg-dark mb-3">
								<div class="card-body py-2">
									<div class="d-flex justify-content-between align-items-center">
										<span class="small">
											<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-primary" />
											{{ customTabJobs[tab.id].status }}
										</span>
										<span class="small">
											<strong>{{ customTabJobs[tab.id].pagesCompleted }}</strong> pages | <strong>{{ customTabJobs[tab.id].videosFound }}</strong> found | <strong>{{ customTabJobs[tab.id].videosSaved }}</strong> saved
										</span>
									</div>
								</div>
							</div>

							<!-- Loading State -->
							<LoadingState v-if="customTabLoading[tab.id]" show-text :loading-text="`Loading ${tab.name} videos...`" />

							<!-- Scraped Videos Grid -->
							<div v-else-if="customTabVideos[tab.id] && customTabVideos[tab.id].length > 0">
								<div class="d-flex justify-content-between align-items-center mb-2">
									<h6 class="mb-0">
										<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
										{{ tab.name }} Videos ({{ (tabVideoTotal[tab.id] || 0).toLocaleString() }})
									</h6>
									<div class="d-flex gap-1">
										<button
											class="btn btn-sm"
											:class="customTabSelectionMode[tab.id] ? 'btn-primary' : 'btn-outline-secondary'"
											@click="toggleTabSelectionMode(tab.id)"
											title="Toggle selection mode"
										>
											<font-awesome-icon :icon="['fas', 'check-square']" class="me-1" />
											{{ customTabSelectionMode[tab.id] ? `${(customTabSelectedVideos[tab.id]?.size || 0)} selected` : 'Select' }}
										</button>
										<button class="btn btn-sm btn-outline-secondary" @click="loadVideosForTab(tab.id)" title="Reload videos">
											<font-awesome-icon :icon="['fas', 'sync']" />
										</button>
										<button class="btn btn-sm btn-outline-danger" @click="clearCustomTabVideos(tab)" title="Delete all videos for this scraper">
											<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
											Delete All
										</button>
									</div>
								</div>

								<!-- Bulk Actions Bar -->
								<ScraperBulkActions
									:selectionMode="customTabSelectionMode[tab.id] || false"
									:selectedCount="customTabSelectedVideos[tab.id]?.size || 0"
									:sendingToJD="customTabJDSending[tab.id] || false"
									@select-all="selectAllTabVideos(tab.id)"
									@send-to-jd="sendTabSelectedToJD(tab)"
									@delete-selected="deleteTabSelectedVideos(tab)"
								/>

								<div class="video-grid">
									<ScrapedVideoCard
										v-for="video in customTabVideos[tab.id]"
										:key="video.id"
										:video="video"
										:selectionMode="customTabSelectionMode[tab.id] || false"
										:isSelected="customTabSelectedVideos[tab.id]?.has(video.id) || false"
										@toggle-selection="toggleTabVideoSelection(tab.id, $event)"
									>
										<template #actions="{ video: v }">
											<button @click="playScrapedVideo(v, tab)" class="btn btn-sm btn-primary">
												<font-awesome-icon :icon="['fas', 'play']" class="me-1" />
												Play
											</button>
											<button @click="scrapersAPI.openURL(v.source_url || v.url).catch(() => {})" class="btn btn-sm btn-outline-primary" title="Open in Firefox">
												<font-awesome-icon :icon="['fab', 'firefox-browser']" />
											</button>
											<button v-if="jdAvailable && (v.source_url || v.embed_url)" @click="sendToJDownloader(v, tab)" class="btn btn-sm btn-warning" :disabled="jdSending[v.id]" title="Send to JDownloader">
												<font-awesome-icon :icon="['fas', jdSending[v.id] ? 'spinner' : 'cloud-download-alt']" :spin="jdSending[v.id]" />
											</button>
										</template>
									</ScrapedVideoCard>
								</div>

								<!-- Pagination -->
								<div class="d-flex align-items-center justify-content-between mt-3">
									<!-- Left: count + per-page selector -->
									<div class="d-flex align-items-center gap-2">
										<span class="text-muted small">
											Showing {{ ((videoPage[tab.id] || 1) - 1) * videosPerPage.value + 1 }}–{{ Math.min((videoPage[tab.id] || 1) * videosPerPage.value, tabVideoTotal[tab.id] || 0) }} of {{ (tabVideoTotal[tab.id] || 0).toLocaleString() }}
										</span>
										<select class="form-select form-select-sm" style="width: auto" @change="changeVideosPerPage($event.target.value, tab.id)">
											<option :value="12" :selected="videosPerPage.value === 12">12/page</option>
											<option :value="24" :selected="videosPerPage.value === 24">24/page</option>
											<option :value="48" :selected="videosPerPage.value === 48">48/page</option>
											<option :value="96" :selected="videosPerPage.value === 96">96/page</option>
										</select>
									</div>
									<!-- Center: nav buttons -->
									<div v-if="(tabVideoTotalPages[tab.id] || 1) > 1" class="btn-group">
										<button class="btn btn-sm btn-outline-secondary" :disabled="(videoPage[tab.id] || 1) === 1" @click="changeVideoPage(tab.id, -1)">
											<font-awesome-icon :icon="['fas', 'chevron-left']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary disabled">Page {{ videoPage[tab.id] || 1 }} / {{ tabVideoTotalPages[tab.id] || 1 }}</button>
										<button class="btn btn-sm btn-outline-secondary" :disabled="(videoPage[tab.id] || 1) >= (tabVideoTotalPages[tab.id] || 1)" @click="changeVideoPage(tab.id, 1)">
											<font-awesome-icon :icon="['fas', 'chevron-right']" />
										</button>
									</div>
									<div v-else></div>
									<!-- Right spacer -->
									<div style="min-width: 120px"></div>
								</div>
							</div>

							<!-- Empty State -->
							<EmptyState v-else :icon="['fas', 'video']" icon-size="4x" :title="`No ${tab.name} Videos Yet`" message="Use the sidebar to configure and start scraping" />
						</div>
						<!-- End Main Content -->

						<!-- Sidebar (Right) -->
						<div class="scraper-sidebar" :class="{ collapsed: collapsedSections[`tab-${tab.id}`] }">
							<!-- Collapse Toggle -->
							<button class="btn btn-sm btn-outline-secondary w-100 mb-3" @click="toggleCollapse(`tab-${tab.id}`)">
								<font-awesome-icon :icon="['fas', 'chevron-right']" class="me-2" />
								Hide Panel
							</button>

							<!-- Auto-Update Panel -->
							<div class="mb-3">
								<AutoUpdatePanel :scraper-type="`custom_${tab.id}`" />
							</div>

							<!-- Statistics -->
							<div class="card mb-3">
								<div class="card-header">
									<h6 class="mb-0">
										<font-awesome-icon :icon="['fas', 'chart-bar']" class="me-2" />
										Statistics
									</h6>
								</div>
								<div class="card-body p-2">
									<div class="sidebar-stats-grid">
										<StatCard :value="customTabVideos[tab.id]?.length || 0" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
										<StatCard :value="customTabJobs[tab.id] ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
									</div>
								</div>
							</div>

							<!-- Scraper Controls -->
							<div class="card mb-3">
								<div class="card-header">
									<h6 class="mb-0">
										<font-awesome-icon :icon="['fas', tab.icon || 'globe']" class="me-2" :style="{ color: tab.color }" />
										Scraper
									</h6>
								</div>
								<div class="card-body">
									<div class="mb-2">
										<label class="form-label small">Base URL</label>
										<input :value="tab.baseUrl" type="text" class="form-control form-control-sm" disabled />
									</div>
									<div class="mb-2">
										<label class="form-label small">Site Type</label>
										<input :value="tab.siteType || 'Generic'" type="text" class="form-control form-control-sm" disabled />
									</div>
									<div class="d-flex flex-column gap-2">
										<button class="btn btn-primary btn-sm w-100" @click="startCustomTabScrape(tab)" :disabled="customTabJobs[tab.id]">
											<font-awesome-icon :icon="['fas', customTabJobs[tab.id] ? 'spinner' : 'play']" :spin="customTabJobs[tab.id]" class="me-2" />
											{{ customTabJobs[tab.id] ? 'Scraping...' : 'Start Scrape' }}
										</button>
										<button class="btn btn-outline-secondary btn-sm w-100" @click="editScraperTab(tab)">
											<font-awesome-icon :icon="['fas', 'edit']" class="me-2" />
											Edit Tab
										</button>
										<button class="btn btn-outline-danger btn-sm w-100" @click="clearCustomTabVideos(tab)">
											<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
											Delete All Videos
										</button>
									</div>
								</div>
							</div>
						</div>
						<!-- End Sidebar -->
					</div>
					<!-- End scraper-layout -->

					<!-- Sidebar Toggle Button (shown when collapsed) -->
					<button v-if="collapsedSections[`tab-${tab.id}`]" class="sidebar-toggle" @click="toggleCollapse(`tab-${tab.id}`)" title="Show Controls Panel">
						<font-awesome-icon :icon="['fas', 'cog']" />
					</button>
				</div>
			</div>
			<!-- End Dynamic Custom Tab Content -->

			<!-- Add/Edit Tab Modal -->
			<div v-if="showAddTabModal || showEditTabModal" class="modal show d-block" tabindex="-1" style="background-color: rgba(0, 0, 0, 0.5)">
				<div class="modal-dialog modal-lg">
					<div class="modal-content bg-dark text-light">
						<div class="modal-header">
							<h5 class="modal-title">
								<font-awesome-icon :icon="['fas', showEditTabModal ? 'edit' : 'plus']" class="me-2" />
								{{ showEditTabModal ? 'Edit' : 'Add New' }} Scraper Tab
							</h5>
							<button type="button" class="btn-close btn-close-white" @click="closeTabModal"></button>
						</div>
						<div class="modal-body">
							<div class="mb-3">
								<label class="form-label">Tab Name</label>
								<input v-model="newTab.name" type="text" class="form-control" placeholder="e.g., My Favorite Site" />
							</div>
							<div class="mb-3">
								<label class="form-label">Base URL</label>
								<input v-model="newTab.baseUrl" type="text" class="form-control" placeholder="https://example.com/videos" />
								<small class="text-muted">The base URL pattern for this site's video listings</small>
							</div>
							<div class="row g-3 mb-3">
								<div class="col-md-6">
									<label class="form-label">Site Type</label>
									<select v-model="newTab.siteType" class="form-select">
										<option value="generic">Generic</option>
										<option value="youtube">YouTube</option>
										<option value="vimeo">Vimeo</option>
										<option value="custom">Custom</option>
									</select>
								</div>
								<div class="col-md-6">
									<label class="form-label">Icon</label>
									<select v-model="newTab.icon" class="form-select">
										<option value="video">Video</option>
										<option value="film">Film</option>
										<option value="play-circle">Play Circle</option>
										<option value="youtube">YouTube</option>
										<option value="vimeo">Vimeo</option>
										<option value="camera">Camera</option>
										<option value="globe">Globe</option>
									</select>
								</div>
							</div>
							<div class="row g-3 mb-3">
								<div class="col-md-6">
									<label class="form-label">Tab Color</label>
									<input v-model="newTab.color" type="color" class="form-control" />
								</div>
								<div class="col-md-6">
									<label class="form-label">Content Type</label>
									<select v-model="newTab.contentType" class="form-select">
										<option value="regular">Regular</option>
										<option value="zoo">Zoo</option>
										<option value="3d">3D / Animation</option>
									</select>
									<small class="text-muted">Used for filtering in Videos page</small>
								</div>
							</div>
							<div class="mb-3">
								<label class="form-label">Scrape Options</label>
								<div class="form-check">
									<input v-model="newTab.scrapeOptions.extractMetadata" class="form-check-input" type="checkbox" id="newTabExtractMetadata" />
									<label class="form-check-label" for="newTabExtractMetadata"> Extract video metadata </label>
								</div>
								<div class="form-check">
									<input v-model="newTab.scrapeOptions.scrapeThumbnails" class="form-check-input" type="checkbox" id="newTabScrapeThumbnails" />
									<label class="form-check-label" for="newTabScrapeThumbnails"> Download thumbnails </label>
								</div>
							</div>
						</div>
						<div class="modal-footer">
							<button type="button" class="btn btn-secondary" @click="closeTabModal">Cancel</button>
							<button type="button" class="btn btn-primary" @click="saveTabConfig">
								<font-awesome-icon :icon="['fas', 'save']" class="me-2" />
								Save Tab
							</button>
						</div>
					</div>
				</div>
			</div>

			<!-- Delete Selected Confirmation Modal -->
			<DeleteConfirmationModal :visible="showDeleteConfirmModal" title="Confirm Deletion" message="Are you sure you want to delete" :itemName="`${selectedCount} selected thread(s)`" warningMessage="This action cannot be undone. All posts and download links associated with these threads will also be deleted." :confirmText="`Delete ${selectedCount} Thread(s)`" :icon="['fas', 'trash']" @confirm="confirmDeleteSelected" @cancel="showDeleteConfirmModal = false" />

			<!-- Delete All Confirmation Modal -->
			<DeleteConfirmationModal :visible="showDeleteAllConfirmModal" title="Confirm Delete All" message="Are you sure you want to delete" :itemName="`ALL ${totalThreads} scraped threads`" warningMessage="DANGER: This will permanently delete ALL scraped threads, posts, and download links from the database. This action cannot be undone!" confirmText="Yes, Delete Everything" :icon="['fas', 'trash-alt']" @confirm="confirmDeleteAll" @cancel="showDeleteAllConfirmModal = false" />
		</div>
	</div>

	<ScrapedVideoModal :video="modalVideo" :tab="modalTab" @close=";(modalVideo = null), (modalTab = null)" @downloaded="onVideoDownloaded" @unmarked="onVideoUnmarked" />
</template>

<script setup>
/* eslint-disable no-unused-vars */
import { ref, shallowRef, onMounted, onBeforeUnmount, computed, getCurrentInstance, watch } from 'vue'
import { useRouter } from 'vue-router'
import { DeleteConfirmationModal, LoadingState, EmptyState, StatCard, SearchBox } from '@/components/shared'
import AutoUpdatePanel from '@/components/scraper/AutoUpdatePanel.vue'
import ScrapedVideoCard from '@/components/scraper/ScrapedVideoCard.vue'
import ScrapedVideoModal from '@/components/scraper/ScrapedVideoModal.vue'
import ScraperBulkActions from '@/components/scraper/ScraperBulkActions.vue'
import SimpcityTab from '@/components/scraper/tabs/SimpcityTab.vue'
import LuxureTVTab from '@/components/scraper/tabs/LuxureTVTab.vue'
import PornHubTab from '@/components/scraper/tabs/PornHubTab.vue'
import SexuriaTab from '@/components/scraper/tabs/SexuriaTab.vue'
import Rule34VideoTab from '@/components/scraper/tabs/Rule34VideoTab.vue'
import MotherlessTab from '@/components/scraper/tabs/MotherlessTab.vue'
import HQPornerTab from '@/components/scraper/tabs/HQPornerTab.vue'
import LeakVidsTab from '@/components/scraper/tabs/LeakVidsTab.vue'
import { useFormatters } from '@/composables/useFormatters'
import { jdownloaderAPI, scrapersAPI, deleteVideosBySource } from '@/services/api'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const router = useRouter()
const { formatDateTime } = useFormatters()

// JDownloader integration
const jdAvailable = ref(false)
const jdSending = ref({}) // Track sending state per video ID

const checkJDownloaderStatus = async () => {
	try {
		const result = await jdownloaderAPI.isAvailable()
		jdAvailable.value = result?.available || false
	} catch {
		jdAvailable.value = false
	}
}

const sendToJDownloader = async (video, tab = null) => {
	if (!video?.source_url && !video?.embed_url) {
		toast.warning('No URL available for this video')
		return
	}

	const key = video.id || video.source_url
	jdSending.value[key] = true

	try {
		let url = video.source_url || video.embed_url
		const videoTitle = video.title || ''
		const scraperName = tab?.name || video.source || 'Scraper'

		// Try to extract direct video URL (MP4/etc) from the page
		try {
			const extracted = await scrapersAPI.extractVideoUrl(url)
			if (extracted?.direct_url) {
				url = extracted.direct_url
			}
		} catch {
			// Fall back to source URL if extraction fails
		}

		const result = await jdownloaderAPI.sendLinks([
			{
				url,
				packageName: videoTitle,
				downloadPath: `/output/${scraperName}`,
				videoId: video?.id ?? null,
			},
		])

		if (result?.successCount > 0) {
			toast.success('Link sent to JDownloader')
		} else if (result?.results?.[0]?.message) {
			toast.error(result.results[0].message)
		} else {
			toast.error('Failed to send link')
		}
	} catch (error) {
		console.error('JDownloader send error:', error)
		toast.error('Failed to send to JDownloader')
	} finally {
		jdSending.value[key] = false
	}
}

// Scraped video modal
const modalVideo = ref(null)
const modalTab = ref(null)
const loading = ref(false)
const stats = ref({})
const threads = ref([])
const threadLinkStats = ref({}) // { threadId: { total, working, dead, unknown } }
const searchQuery = ref('')
const viewMode = ref('grid')
const currentPage = ref(1)
const totalPages = ref(1)
const totalThreads = ref(0)
const limit = 20

// Filters and sorting
const sortBy = ref('date_desc')
const filterProvider = ref('')
const filterContent = ref('')

const showScrapeModal = ref(false)
const showAuthModal = ref(false)
const showForumScrapeModal = ref(false)
const showDeleteConfirmModal = ref(false)
const showDeleteAllConfirmModal = ref(false)
const scrapeURL = ref('')
const forumURL = ref('https://simpcity.is/forums/xxx-porn.50/')
const simpcityForums = ref([
	{ name: 'XXX Porn', url: 'https://simpcity.is/forums/xxx-porn.50/' },
	{ name: 'OnlyFans', url: 'https://simpcity.is/forums/onlyfans.8/' },
	{ name: 'Hotwives / QoS', url: 'https://simpcity.is/forums/hotwives-qos.96/' },
	{ name: 'Fansly', url: 'https://simpcity.is/forums/fansly.58/' },
	{ name: 'Patreon', url: 'https://simpcity.is/forums/patreon.9/' },
	{ name: 'MYM Fans', url: 'https://simpcity.is/forums/mym-fans.56/' },
	{ name: 'Snapchat', url: 'https://simpcity.is/forums/snapchat.12/' },
	{ name: 'Cosplayers', url: 'https://simpcity.is/forums/cosplayers.45/' },
	{ name: 'TikTok', url: 'https://simpcity.is/forums/tiktok.14/' },
	{ name: 'Twitch', url: 'https://simpcity.is/forums/twitch.18/' },
	{ name: 'Instagram', url: 'https://simpcity.is/forums/instagram.19/' },
	{ name: 'YouTube', url: 'https://simpcity.is/forums/youtube.20/' },
])
const authCookie = ref('')
const sessionCookieSet = ref(false)
const verifiedUsername = ref('')
const loginHelperLoading = ref(false)
const loginHelperInstructions = ref([])
const selectedThread = ref(null)
const selectedThreadIds = ref(new Set())
const selectionMode = ref(false)

// Tab state
const activeTab = ref('simpcity')

// Collapsible controls state (collapsed by default to focus on content)
const collapsedSections = ref({
	simpcity: true,
	pornhub: true,
	luxuretv: true,
	sexuria: true,
	rule34video: true,
})

const toggleCollapse = (tab) => {
	collapsedSections.value[tab] = !collapsedSections.value[tab]
}

// Video scraper state
const videoUrl = ref('')
const videoSiteType = ref('generic')
const videoScraping = ref(false)
const scrapedVideos = ref([])
const videoScrapeOptions = ref({
	downloadVideo: false,
	extractMetadata: true,
	scrapeThumbnails: true,
	maxPages: 0, // 0 = unlimited, scrapes all pages automatically
})

// Scraper tabs state
const scraperTabs = ref([])
const showAddTabModal = ref(false)
const showEditTabModal = ref(false)
const editingTab = ref(null)
const newTab = ref({
	name: '',
	baseUrl: '',
	siteType: 'generic',
	icon: 'video',
	color: '#007bff',
	contentType: 'regular',
	scrapeOptions: {
		extractMetadata: true,
		scrapeThumbnails: true,
		downloadVideo: false,
	},
})
const customTabUrls = ref({}) // { tabId: url }
const customTabJobs = ref({}) // { tabId: job }
const customTabVideos = ref({}) // { tabId: [videos] }
const customTabLoading = ref({}) // { tabId: boolean } - Loading state for each custom tab
const videoSearch = ref({}) // { tabId: searchQuery }
const videoSort = ref({}) // { tabId: sortType }
const videoPage = ref({}) // { tabId: pageNumber }
const videoStatusFilter = ref({}) // { tabId: status }
const videoResolutionFilter = ref({}) // { tabId: resolution }
const videoDurationFilter = ref({}) // { tabId: durationRange }
const videoThumbnailFilter = ref({}) // { tabId: 'yes'|'no'|'' } — kept for compat, not used in server query
const videoOnlyDownloaded = ref({}) // { tabId: boolean }
const videoAuthorFilter = ref({}) // { tabId: string }
const videoMinViews = ref({}) // { tabId: number }
const tabVideoTotal = ref({}) // { tabId: number } — server-side total count
const tabVideoTotalPages = ref({}) // { tabId: number } — server-side total pages
const showVideoFilters = ref({}) // { tabId: boolean }
const tabSearchTimeouts = {} // debounce per tab
// Per-tab selection state (bulk actions)
const customTabSelectionMode = ref({}) // { tabId: boolean }
const customTabSelectedVideos = ref({}) // { tabId: Set<number> }
const customTabJDSending = ref({}) // { tabId: boolean } — loading state for bulk JD send
const videosPerPage = ref(parseInt(localStorage.getItem('scraper-page-size') || '24'))

// Timeout tracking for cleanup
const activeTimeouts = ref([])

// Thumbnail preview animation state
const thumbnailPreviewState = ref({}) // { videoId: { currentFrame: number, interval: intervalId } }

// PornHub scraper state
const pornHubKeyword = ref('')
const pornHubMaxPages = ref(0)
const pornHubScraping = ref(false)
const pornHubLoading = ref(false) // Loading state for video list
const pornHubVideos = shallowRef([]) // Using shallowRef for large array - better performance
const pornHubStats = ref({ total_videos: 0, keyword_counts: {} })
const pornHubSuggestedKeywords = ref([])
const pornHubActiveJob = ref(null)
const pornHubVideoSearch = ref('')
const pornHubVideoSort = ref('newest')
const pornHubKeywordFilter = ref('')
const pornHubVideoPage = ref(1)
const pornHubSelectedVideos = ref(new Set()) // Selected video IDs for batch operations
const pornHubSelectionMode = ref(false) // Toggle selection mode

// LuxureTV scraper state
const luxureTVCategoryURL = ref('')
const luxureTVMaxPages = ref(0)
const luxureTVScraping = ref(false)
const luxureTVLoading = ref(false) // Loading state for video list
const luxureTVVideos = shallowRef([]) // Using shallowRef for large array - better performance
const luxureTVStats = ref({ total_videos: 0, total_views: 0, avg_rating: 0 })
const luxureTVCategories = ref([])
const luxureTVActiveJob = ref(null)
const luxureTVVideoSearch = ref('')
const luxureTVSelectedVideos = ref(new Set()) // Selected video IDs for batch operations
const luxureTVSelectionMode = ref(false) // Toggle selection mode
const luxureTVVideoSort = ref('newest')
const luxureTVVideoPage = ref(1)
const luxureTVFetchingEmbeds = ref(false)

// Sexuria scraper state
const sexuriaCategoryURL = ref('')
const sexuriaMaxPages = ref(0)
const sexuriaScraping = ref(false)
const sexuriaLoading = ref(false) // Loading state for item list
const sexuriaItems = shallowRef([]) // Using shallowRef for large array - better performance
const sexuriaStats = ref({ total_items: 0, host_counts: {} })
const sexuriaCategories = ref([])
const sexuriaActiveJob = ref(null)
const sexuriaItemSearch = ref('')
const sexuriaHostFilter = ref('')
const sexuriaItemSort = ref('newest')
const sexuriaItemPage = ref(1)
const sexuriaSelectedItems = ref(new Set()) // Selected item IDs for batch operations
const sexuriaSelectionMode = ref(false) // Toggle selection mode
const itemsPerPage = 24

// Sexuria Link Management state
const sexuriaLinkStats = ref({}) // { itemId: { total, working, dead, unknown } }
const extractingLinks = ref({}) // { itemId: true/false } - track which items are being extracted
const sexuriaLinksModal = ref({
	visible: false,
	item: null,
	links: [],
	loading: false,
	verifying: false,
})

// Rule34Video scraper state
const rule34VideoSortBy = ref('post_date')
const rule34VideoMaxPages = ref(0)
const rule34VideoScraping = ref(false)
const rule34VideoLoading = ref(false)
const rule34VideoVideos = shallowRef([])
const rule34VideoStats = ref({ total_videos: 0, avg_rating: 0 })
const rule34VideoSortOptions = ref([
	{ value: 'post_date', label: 'Most Recent' },
	{ value: 'video_viewed', label: 'Most Viewed' },
	{ value: 'rating', label: 'Top Rated' },
	{ value: 'video_viewed_today', label: 'Most Viewed Today' },
	{ value: 'video_viewed_week', label: 'Most Viewed This Week' },
	{ value: 'video_viewed_month', label: 'Most Viewed This Month' },
])
const rule34VideoActiveJob = ref(null)
const rule34VideoVideoSearch = ref('')
const rule34VideoVideoSort = ref('newest')
const rule34VideoVideoPage = ref(1)
const rule34VideoSelectedVideos = ref(new Set())
const rule34VideoSelectionMode = ref(false)

// Debounced search values for performance
const debouncedPornHubSearch = ref('')
const debouncedLuxureTVSearch = ref('')
const debouncedSexuriaSearch = ref('')
const debouncedRule34VideoSearch = ref('')

// Debounce search inputs to reduce re-renders
let pornHubSearchTimeout = null
let luxureTVSearchTimeout = null
let sexuriaSearchTimeout = null
let rule34VideoSearchTimeout = null

watch(pornHubVideoSearch, (val) => {
	clearTimeout(pornHubSearchTimeout)
	pornHubSearchTimeout = setTimeout(() => {
		debouncedPornHubSearch.value = val
	}, 300)
})

watch(luxureTVVideoSearch, (val) => {
	clearTimeout(luxureTVSearchTimeout)
	luxureTVSearchTimeout = setTimeout(() => {
		debouncedLuxureTVSearch.value = val
	}, 300)
})

watch(sexuriaItemSearch, (val) => {
	clearTimeout(sexuriaSearchTimeout)
	sexuriaSearchTimeout = setTimeout(() => {
		debouncedSexuriaSearch.value = val
	}, 300)
})

watch(rule34VideoVideoSearch, (val) => {
	clearTimeout(rule34VideoSearchTimeout)
	rule34VideoSearchTimeout = setTimeout(() => {
		debouncedRule34VideoSearch.value = val
	}, 300)
})

// Sexuria details cache (for hover card)
const sexuriaDetailsCache = ref({}) // Cache fetched details by URL

// Sexuria hover card state
const sexuriaHoverDetails = ref({
	visible: false,
	item: null,
	data: null,
	loading: false,
	error: null,
	style: {},
})
const sexuriaHoveredScreenshot = ref(null)
let sexuriaHoverTimeout = null
let sexuriaHideTimeout = null

// Sexuria screenshot modal state (for viewing screenshots in full size)
const sexuriaScreenshotModal = ref({
	visible: false,
	screenshots: [],
	currentIndex: 0,
})

// Sexuria multi-select filtering
const sexuriaTagFilter = ref([]) // Selected tags to filter by
const sexuriaPornstarFilter = ref([]) // Selected pornstars to filter by
const showSexuriaTagDropdown = ref(false)
const showSexuriaPornstarDropdown = ref(false)

// Sexuria Advanced Search state
const sexuriaAdvancedMode = ref(false) // Toggle between simple category scrape and advanced search
const sexuriaSearchOptions = ref({
	categories: [],
	fileHosts: [],
	resolutions: [],
})
const sexuriaAdvancedSearch = ref({
	categories: [],
	title: '',
	description: '',
	pornstars: '',
	production: '',
	tags: '',
	resolutions: [],
	fileHosts: [],
	maxPages: 0,
})

// Computed
const visiblePages = computed(() => {
	const pages = []
	const start = Math.max(1, currentPage.value - 2)
	const end = Math.min(totalPages.value, currentPage.value + 2)

	for (let i = start; i <= end; i++) {
		pages.push(i)
	}
	return pages
})

// PornHub computed properties
const filteredPornHubVideos = computed(() => {
	let videos = [...pornHubVideos.value]

	// Filter by keyword
	if (pornHubKeywordFilter.value) {
		videos = videos.filter((v) => v.tags && v.tags.includes(pornHubKeywordFilter.value))
	}

	// Filter by search
	if (debouncedPornHubSearch.value) {
		const search = debouncedPornHubSearch.value.toLowerCase()
		videos = videos.filter((v) => (v.title && v.title.toLowerCase().includes(search)) || (v.uploader && v.uploader.toLowerCase().includes(search)))
	}

	// Sort
	switch (pornHubVideoSort.value) {
		case 'newest':
			videos.sort((a, b) => new Date(b.discovered_at) - new Date(a.discovered_at))
			break
		case 'oldest':
			videos.sort((a, b) => new Date(a.discovered_at) - new Date(b.discovered_at))
			break
		case 'views-desc':
			videos.sort((a, b) => (b.view_count || 0) - (a.view_count || 0))
			break
		case 'duration-desc':
			videos.sort((a, b) => (b.duration || 0) - (a.duration || 0))
			break
	}

	return videos
})

const paginatedPornHubVideos = computed(() => {
	const start = (pornHubVideoPage.value - 1) * videosPerPage.value
	const end = start + videosPerPage.value
	return filteredPornHubVideos.value.slice(start, end)
})

// LuxureTV computed properties
const filteredLuxureTVVideos = computed(() => {
	let videos = [...luxureTVVideos.value]

	// Filter by search
	if (debouncedLuxureTVSearch.value) {
		const search = debouncedLuxureTVSearch.value.toLowerCase()
		videos = videos.filter((v) => (v.title && v.title.toLowerCase().includes(search)) || (v.tags && v.tags.some((tag) => tag.toLowerCase().includes(search))))
	}

	// Sort
	switch (luxureTVVideoSort.value) {
		case 'newest':
			videos.sort((a, b) => new Date(b.discovered_at) - new Date(a.discovered_at))
			break
		case 'oldest':
			videos.sort((a, b) => new Date(a.discovered_at) - new Date(b.discovered_at))
			break
		case 'views-desc':
			videos.sort((a, b) => (b.view_count || 0) - (a.view_count || 0))
			break
		case 'rating-desc':
			videos.sort((a, b) => (b.rating || 0) - (a.rating || 0))
			break
		case 'duration-desc':
			videos.sort((a, b) => (b.duration || 0) - (a.duration || 0))
			break
	}

	return videos
})

const paginatedLuxureTVVideos = computed(() => {
	const start = (luxureTVVideoPage.value - 1) * videosPerPage.value
	const end = start + videosPerPage.value
	return filteredLuxureTVVideos.value.slice(start, end)
})

// Sexuria computed properties

// Get unique tags from cached details
const availableSexuriaTags = computed(() => {
	const tagsSet = new Set()
	Object.values(sexuriaDetailsCache.value).forEach((details) => {
		if (details && details.tags) {
			details.tags.forEach((tag) => tagsSet.add(tag))
		}
	})
	return Array.from(tagsSet).sort()
})

// Get unique pornstars from cached details
const availableSexuriaPornstars = computed(() => {
	const starsSet = new Set()
	Object.values(sexuriaDetailsCache.value).forEach((details) => {
		if (details && details.porn_stars) {
			details.porn_stars.forEach((star) => starsSet.add(star))
		}
	})
	return Array.from(starsSet).sort()
})

const filteredSexuriaItems = computed(() => {
	let items = [...sexuriaItems.value]

	// Filter by search (using debounced value)
	if (debouncedSexuriaSearch.value) {
		const search = debouncedSexuriaSearch.value.toLowerCase()
		items = items.filter((item) => (item.title && item.title.toLowerCase().includes(search)) || (item.uploader && item.uploader.toLowerCase().includes(search)))
	}

	// Filter by host
	if (sexuriaHostFilter.value) {
		items = items.filter((item) => item.tags && item.tags.includes(sexuriaHostFilter.value))
	}

	// Filter by selected tags (from cached details)
	if (sexuriaTagFilter.value.length > 0) {
		items = items.filter((item) => {
			const cachedDetails = sexuriaDetailsCache.value[item.url]
			if (!cachedDetails || !cachedDetails.tags) return false
			// Item must have ALL selected tags
			return sexuriaTagFilter.value.every((tag) => cachedDetails.tags.includes(tag))
		})
	}

	// Filter by selected pornstars (from cached details)
	if (sexuriaPornstarFilter.value.length > 0) {
		items = items.filter((item) => {
			const cachedDetails = sexuriaDetailsCache.value[item.url]
			if (!cachedDetails || !cachedDetails.porn_stars) return false
			// Item must have at least ONE of the selected pornstars
			return sexuriaPornstarFilter.value.some((star) => cachedDetails.porn_stars.includes(star))
		})
	}

	// Sort
	switch (sexuriaItemSort.value) {
		case 'newest':
			items.sort((a, b) => new Date(b.discovered_at) - new Date(a.discovered_at))
			break
		case 'oldest':
			items.sort((a, b) => new Date(a.discovered_at) - new Date(b.discovered_at))
			break
		case 'title-asc':
			items.sort((a, b) => (a.title || '').localeCompare(b.title || ''))
			break
		case 'title-desc':
			items.sort((a, b) => (b.title || '').localeCompare(a.title || ''))
			break
	}

	return items
})

const paginatedSexuriaItems = computed(() => {
	const start = (sexuriaItemPage.value - 1) * itemsPerPage
	const end = start + itemsPerPage
	return filteredSexuriaItems.value.slice(start, end)
})

// Rule34Video computed properties
const filteredRule34VideoVideos = computed(() => {
	let videos = [...rule34VideoVideos.value]

	// Filter by search
	if (debouncedRule34VideoSearch.value) {
		const search = debouncedRule34VideoSearch.value.toLowerCase()
		videos = videos.filter((v) => (v.title && v.title.toLowerCase().includes(search)) || (v.tags && v.tags.some((tag) => tag.toLowerCase().includes(search))))
	}

	// Sort
	switch (rule34VideoVideoSort.value) {
		case 'newest':
			videos.sort((a, b) => new Date(b.discovered_at) - new Date(a.discovered_at))
			break
		case 'oldest':
			videos.sort((a, b) => new Date(a.discovered_at) - new Date(b.discovered_at))
			break
		case 'views-desc':
			videos.sort((a, b) => (b.view_count || 0) - (a.view_count || 0))
			break
		case 'rating-desc':
			videos.sort((a, b) => (b.rating || 0) - (a.rating || 0))
			break
		case 'duration-desc':
			videos.sort((a, b) => (b.duration || 0) - (a.duration || 0))
			break
	}

	return videos
})

const paginatedRule34VideoVideos = computed(() => {
	const start = (rule34VideoVideoPage.value - 1) * videosPerPage.value
	const end = start + videosPerPage.value
	return filteredRule34VideoVideos.value.slice(start, end)
})

// Methods
const checkAuthStatus = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/session')
		const data = await response.json()
		if (data.success && data.data) {
			sessionCookieSet.value = data.data.is_set
		}
	} catch (error) {
		console.error('Error checking auth status:', error)
	}
}

// Parse cookie table format from DevTools into HTTP header format
const parseCookieTable = (input) => {
	// Check if it's already in header format (name=value; name=value)
	if (input.includes('=') && !input.includes('\t') && input.split(';').length > 1) {
		return input.trim()
	}

	// Parse table format: each row is "name\tvalue\tdomain\tpath\t..."
	const lines = input.trim().split('\n')
	const cookies = []

	for (const line of lines) {
		const parts = line.split('\t')
		if (parts.length >= 2) {
			const name = parts[0].trim()
			const value = parts[1].trim()
			// Skip empty names or header rows
			if (name && value && name !== 'Name' && !name.startsWith('__')) {
				// Skip tracking cookies (__ddg*, etc) - focus on session cookies
				cookies.push(`${name}=${value}`)
			} else if (name && value && name.startsWith('__')) {
				// Include ddg cookies too as they might be needed for anti-bot
				cookies.push(`${name}=${value}`)
			}
		}
	}

	return cookies.join('; ')
}

const setAuthCookie = async () => {
	if (!authCookie.value) {
		toast.warning('Please enter a cookie')
		return
	}

	// Parse the cookie input (handles both table format and header format)
	const parsedCookie = parseCookieTable(authCookie.value)

	if (!parsedCookie) {
		toast.error('Could not parse cookie data. Please try copying again.')
		return
	}

	console.log('Parsed cookie:', parsedCookie.substring(0, 100) + '...')

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/session', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				cookie: parsedCookie,
			}),
		})

		const data = await response.json()

		if (data.success && data.data) {
			const result = data.data
			if (result.verified) {
				verifiedUsername.value = result.username || ''
				const username = result.username ? ` as ${result.username}` : ''
				toast.success(`Cookie verified successfully${username}!`)
				sessionCookieSet.value = true
				showAuthModal.value = false
				authCookie.value = ''
			} else {
				// Cookie was saved but verification failed
				verifiedUsername.value = ''
				toast.warning(result.message || 'Cookie saved but verification failed. Please check the cookie is correct.')
				sessionCookieSet.value = false
			}
		} else {
			toast.error(data.message || 'Failed to set cookie')
		}
	} catch (error) {
		console.error('Error setting cookie:', error)
		toast.error('Failed to set authentication cookie')
	}
}

const openLoginHelper = async () => {
	loginHelperLoading.value = true
	loginHelperInstructions.value = []

	try {
		const response = await fetch('http://localhost:8080/api/v1/system/open-login-helper', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ site: 'simpcity' }),
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Login page opened in Firefox')
			// Show instructions after successful browser open
			loginHelperInstructions.value = ['Log in to your Simpcity account', 'Press F12 to open Developer Tools', 'Go to Application tab → Cookies → simpcity.is', 'Select all cookies (Ctrl+A) and copy their values', 'Paste them in the text area below']
		} else {
			toast.error(data.message || 'Failed to open browser')
		}
	} catch (error) {
		console.error('Error opening login helper:', error)
		toast.error('Failed to open login page')
	} finally {
		loginHelperLoading.value = false
	}
}

const loadStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/stats')
		const data = await response.json()
		if (data.success) {
			stats.value = data.data
		}
	} catch (error) {
		console.error('Error loading stats:', error)
	}
}

const loadThreads = async () => {
	loading.value = true
	try {
		let url = searchQuery.value ? `http://localhost:8080/api/v1/scraper/threads/search?q=${encodeURIComponent(searchQuery.value)}&` : `http://localhost:8080/api/v1/scraper/threads?`

		// Add pagination
		url += `page=${currentPage.value}&limit=${limit}`

		// Add sorting
		if (sortBy.value) {
			url += `&sort=${sortBy.value}`
		}

		// Add filters
		if (filterProvider.value) {
			url += `&provider=${encodeURIComponent(filterProvider.value)}`
		}
		if (filterContent.value) {
			url += `&filter=${filterContent.value}`
		}

		const response = await fetch(url)
		const data = await response.json()

		if (data.success) {
			threads.value = data.data || []
			totalPages.value = data.pagination.total_pages
			totalThreads.value = data.pagination.total
		}
	} catch (error) {
		console.error('Error loading threads:', error)
		toast.error('Failed to load threads')
	} finally {
		loading.value = false
	}
}

const searchThreads = () => {
	currentPage.value = 1
	loadThreads()
}

const handleSortChange = () => {
	currentPage.value = 1
	loadThreads()
}

const handleFilterChange = () => {
	currentPage.value = 1
	loadThreads()
}

const goToPage = (page) => {
	if (page >= 1 && page <= totalPages.value) {
		currentPage.value = page
		loadThreads()
		window.scrollTo({ top: 0, behavior: 'smooth' })
	}
}

const startScrape = async () => {
	if (!scrapeURL.value) {
		toast.warning('Please enter a URL')
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/threads/scrape', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				url: scrapeURL.value,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Scraping started! Check the Tasks page for progress.')
			showScrapeModal.value = false
			scrapeURL.value = ''

			// Reload threads after a short delay
			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 2000)
		} else {
			toast.error(data.message || 'Failed to start scraping')
		}
	} catch (error) {
		console.error('Error starting scrape:', error)
		toast.error('Failed to start scraping')
	}
}

const rescrapeThread = async (thread) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${thread.id}/rescrape`, {
			method: 'POST',
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Rescaping started! Check the Tasks page for progress.')
			selectedThread.value = null

			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 2000)
		} else {
			toast.error(data.message || 'Failed to start rescraping')
		}
	} catch (error) {
		console.error('Error rescraping thread:', error)
		toast.error('Failed to start rescraping')
	}
}

const openThread = (thread) => {
	router.push(`/scraper/${thread.id}`)
}

const openThreadURL = (url) => {
	scrapersAPI.openURL(url).catch(() => window.open(url, '_blank'))
}

const toggleThreadWatch = async (thread) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${thread.id}/watch`, {
			method: 'POST',
		})

		const data = await response.json()

		if (data.success) {
			// Update local state
			thread.is_watched = data.data.is_watched
			const action = thread.is_watched ? 'now watching' : 'stopped watching'
			toast.success(`You are ${action} "${thread.title}"`)
		} else {
			toast.error(data.message || 'Failed to toggle watch status')
		}
	} catch (error) {
		console.error('Error toggling thread watch:', error)
		toast.error('Failed to toggle watch status')
	}
}

// formatDate replaced with formatDateTime from useFormatters composable
const formatDate = formatDateTime

// PornHub scraper methods
const loadPornHubKeywords = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/pornhub/keywords')
		const data = await response.json()
		if (data.success && data.data) {
			pornHubSuggestedKeywords.value = data.data
		}
	} catch (error) {
		console.error('Error loading PornHub keywords:', error)
	}
}

const loadPornHubStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/pornhub/stats')
		const data = await response.json()
		if (data.success && data.data) {
			pornHubStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading PornHub stats:', error)
	}
}

const loadPornHubVideos = async () => {
	pornHubLoading.value = true
	try {
		// Fetch with high limit to get all videos (server now supports pagination)
		const response = await fetch('http://localhost:8080/api/v1/scraper/pornhub/videos?limit=1000000')
		const data = await response.json()
		if (data.success && data.data) {
			pornHubVideos.value = data.data.videos || []
		}
	} catch (error) {
		console.error('Error loading PornHub videos:', error)
	} finally {
		pornHubLoading.value = false
	}
}

const clearPornHubVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${pornHubVideos.value.length} PornHub videos? This cannot be undone.`)) {
		return
	}
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/site/pornhub', {
			method: 'DELETE',
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} PornHub videos`)
			pornHubVideos.value = []
			await loadPornHubStats()
		} else {
			toast.error(data.message || 'Failed to delete videos')
		}
	} catch (error) {
		console.error('Error deleting PornHub videos:', error)
		toast.error('Failed to delete videos')
	}
}

// PornHub batch operations
const togglePornHubSelectionMode = () => {
	pornHubSelectionMode.value = !pornHubSelectionMode.value
	if (!pornHubSelectionMode.value) {
		pornHubSelectedVideos.value.clear()
	}
}

const togglePornHubVideoSelection = (videoId) => {
	if (pornHubSelectedVideos.value.has(videoId)) {
		pornHubSelectedVideos.value.delete(videoId)
	} else {
		pornHubSelectedVideos.value.add(videoId)
	}
	// Force reactivity update
	pornHubSelectedVideos.value = new Set(pornHubSelectedVideos.value)
}

const selectAllPornHubVideos = () => {
	paginatedPornHubVideos.value.forEach((v) => pornHubSelectedVideos.value.add(v.id))
	pornHubSelectedVideos.value = new Set(pornHubSelectedVideos.value)
}

const deleteSelectedPornHubVideos = async () => {
	if (pornHubSelectedVideos.value.size === 0) return
	if (!confirm(`Are you sure you want to delete ${pornHubSelectedVideos.value.size} selected videos?`)) return

	try {
		const ids = Array.from(pornHubSelectedVideos.value)
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/bulk-delete', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ ids }),
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} videos`)
			pornHubSelectedVideos.value.clear()
			pornHubSelectionMode.value = false
			await loadPornHubVideos()
			await loadPornHubStats()
		} else {
			toast.error(data.message || 'Failed to delete videos')
		}
	} catch (error) {
		console.error('Error deleting selected videos:', error)
		toast.error('Failed to delete selected videos')
	}
}

// LuxureTV batch operations
const toggleLuxureTVSelectionMode = () => {
	luxureTVSelectionMode.value = !luxureTVSelectionMode.value
	if (!luxureTVSelectionMode.value) {
		luxureTVSelectedVideos.value.clear()
	}
}

const toggleLuxureTVVideoSelection = (videoId) => {
	if (luxureTVSelectedVideos.value.has(videoId)) {
		luxureTVSelectedVideos.value.delete(videoId)
	} else {
		luxureTVSelectedVideos.value.add(videoId)
	}
	luxureTVSelectedVideos.value = new Set(luxureTVSelectedVideos.value)
}

const selectAllLuxureTVVideos = () => {
	paginatedLuxureTVVideos.value.forEach((v) => luxureTVSelectedVideos.value.add(v.id))
	luxureTVSelectedVideos.value = new Set(luxureTVSelectedVideos.value)
}

const deleteSelectedLuxureTVVideos = async () => {
	if (luxureTVSelectedVideos.value.size === 0) return
	if (!confirm(`Are you sure you want to delete ${luxureTVSelectedVideos.value.size} selected videos?`)) return

	try {
		const ids = Array.from(luxureTVSelectedVideos.value)
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/bulk-delete', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ ids }),
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} videos`)
			luxureTVSelectedVideos.value.clear()
			luxureTVSelectionMode.value = false
			await loadLuxureTVVideos()
			await loadLuxureTVStats()
		} else {
			toast.error(data.message || 'Failed to delete videos')
		}
	} catch (error) {
		console.error('Error deleting selected videos:', error)
		toast.error('Failed to delete selected videos')
	}
}

// Sexuria batch operations
const toggleSexuriaSelectionMode = () => {
	sexuriaSelectionMode.value = !sexuriaSelectionMode.value
	if (!sexuriaSelectionMode.value) {
		sexuriaSelectedItems.value.clear()
	}
}

const toggleSexuriaItemSelection = (itemId) => {
	if (sexuriaSelectedItems.value.has(itemId)) {
		sexuriaSelectedItems.value.delete(itemId)
	} else {
		sexuriaSelectedItems.value.add(itemId)
	}
	sexuriaSelectedItems.value = new Set(sexuriaSelectedItems.value)
}

const selectAllSexuriaItems = () => {
	paginatedSexuriaItems.value.forEach((v) => sexuriaSelectedItems.value.add(v.id))
	sexuriaSelectedItems.value = new Set(sexuriaSelectedItems.value)
}

const deleteSelectedSexuriaItems = async () => {
	if (sexuriaSelectedItems.value.size === 0) return
	if (!confirm(`Are you sure you want to delete ${sexuriaSelectedItems.value.size} selected items?`)) return

	try {
		const ids = Array.from(sexuriaSelectedItems.value)
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/bulk-delete', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ ids }),
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} items`)
			sexuriaSelectedItems.value.clear()
			sexuriaSelectionMode.value = false
			await loadSexuriaItems()
			await loadSexuriaStats()
		} else {
			toast.error(data.message || 'Failed to delete items')
		}
	} catch (error) {
		console.error('Error deleting selected items:', error)
		toast.error('Failed to delete selected items')
	}
}

const startPornHubScrape = async () => {
	if (!pornHubKeyword.value) {
		toast.warning('Please enter a keyword')
		return
	}

	pornHubScraping.value = true

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/pornhub/scrape', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				keyword: pornHubKeyword.value,
				maxPages: pornHubMaxPages.value || 0,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Started PornHub scrape for keyword: ${pornHubKeyword.value}`)

			// Set up active job tracking
			pornHubActiveJob.value = {
				id: data.data.job_id,
				keyword: pornHubKeyword.value,
				status: 'running',
				pagesCompleted: 0,
				videosFound: 0,
				videosSaved: 0,
				errorsCount: 0,
			}

			// Poll for job progress
			pollPornHubJobProgress(data.data.job_id)
		} else {
			toast.error(data.message || 'Failed to start PornHub scrape')
			pornHubScraping.value = false
		}
	} catch (error) {
		console.error('Error starting PornHub scrape:', error)
		toast.error('Failed to start PornHub scrape')
		pornHubScraping.value = false
	}
}

const pollPornHubJobProgress = async (jobId) => {
	const pollInterval = setInterval(async () => {
		try {
			const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${jobId}`)
			const data = await response.json()

			if (data.success && data.data) {
				const job = data.data
				pornHubActiveJob.value = {
					id: job.id,
					keyword: pornHubKeyword.value,
					status: job.status,
					pagesCompleted: job.pagesCompleted || 0,
					videosFound: job.videosFound || 0,
					videosSaved: job.videosSaved || 0,
					errorsCount: job.errorsCount || 0,
				}

				// Check if job is complete
				if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
					clearInterval(pollInterval)
					pornHubScraping.value = false

					if (job.status === 'completed') {
						toast.success(`PornHub scrape complete! Found ${job.videosFound} videos`)
					} else if (job.status === 'failed') {
						toast.error('PornHub scrape failed')
					}

					// Reload videos and stats
					await loadPornHubVideos()
					await loadPornHubStats()

					// Clear active job after a delay
					setTimeout(() => {
						pornHubActiveJob.value = null
					}, 5000)
				}
			}
		} catch (error) {
			console.error('Error polling job progress:', error)
		}
	}, 2000) // Poll every 2 seconds

	// Store interval ID for cleanup
	activeTimeouts.value.push(pollInterval)
}

// LuxureTV scraper methods
const loadLuxureTVCategories = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/luxuretv/categories')
		const data = await response.json()
		if (data.success && data.data) {
			luxureTVCategories.value = data.data
		}
	} catch (error) {
		console.error('Error loading LuxureTV categories:', error)
	}
}

const loadLuxureTVStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/luxuretv/stats')
		const data = await response.json()
		if (data.success && data.data) {
			luxureTVStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading LuxureTV stats:', error)
	}
}

const loadLuxureTVVideos = async () => {
	luxureTVLoading.value = true
	try {
		// Fetch with high limit to get all videos (server now supports pagination)
		const response = await fetch('http://localhost:8080/api/v1/scraper/luxuretv/videos?limit=1000000')
		const data = await response.json()
		if (data.success && data.data) {
			luxureTVVideos.value = data.data.videos || []
		}
	} catch (error) {
		console.error('Error loading LuxureTV videos:', error)
	} finally {
		luxureTVLoading.value = false
	}
}

const clearLuxureTVVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${luxureTVVideos.value.length} LuxureTV videos? This cannot be undone.`)) {
		return
	}
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/site/luxuretv', {
			method: 'DELETE',
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} LuxureTV videos`)
			luxureTVVideos.value = []
			await loadLuxureTVStats()
		} else {
			toast.error(data.message || 'Failed to delete videos')
		}
	} catch (error) {
		console.error('Error deleting LuxureTV videos:', error)
		toast.error('Failed to delete videos')
	}
}

// Helper function to start LuxureTV scrape from category button
const startLuxureTVScrapeCategory = async (category) => {
	luxureTVCategoryURL.value = category.url
	await startLuxureTVScrape()
}

const startLuxureTVScrape = async () => {
	if (!luxureTVCategoryURL.value) {
		toast.warning('Please select a category')
		return
	}

	luxureTVScraping.value = true

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/luxuretv/scrape', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				categoryUrl: luxureTVCategoryURL.value,
				maxPages: luxureTVMaxPages.value || 0,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Started LuxureTV scrape for: ${luxureTVCategoryURL.value}`)

			// Set up active job tracking
			luxureTVActiveJob.value = {
				id: data.data.job_id,
				status: 'running',
				pagesCompleted: 0,
				videosFound: 0,
				videosSaved: 0,
				errorsCount: 0,
			}

			// Poll for job progress
			pollLuxureTVJobProgress(data.data.job_id)
		} else {
			toast.error(data.message || 'Failed to start LuxureTV scrape')
			luxureTVScraping.value = false
		}
	} catch (error) {
		console.error('Error starting LuxureTV scrape:', error)
		toast.error('Failed to start LuxureTV scrape')
		luxureTVScraping.value = false
	}
}

const pollLuxureTVJobProgress = async (jobId) => {
	const pollInterval = setInterval(async () => {
		try {
			const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${jobId}`)
			const data = await response.json()

			if (data.success && data.data) {
				const job = data.data
				luxureTVActiveJob.value = {
					id: job.id,
					status: job.status,
					pagesCompleted: job.pagesCompleted || 0,
					videosFound: job.videosFound || 0,
					videosSaved: job.videosSaved || 0,
					errorsCount: job.errorsCount || 0,
				}

				// Check if job is complete
				if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
					clearInterval(pollInterval)
					luxureTVScraping.value = false

					if (job.status === 'completed') {
						toast.success(`LuxureTV scrape complete! Found ${job.videosFound} videos`)
					} else if (job.status === 'failed') {
						toast.error('LuxureTV scrape failed')
					}

					// Reload videos and stats
					await loadLuxureTVVideos()
					await loadLuxureTVStats()

					// Clear active job after a delay
					setTimeout(() => {
						luxureTVActiveJob.value = null
					}, 5000)
				}
			}
		} catch (error) {
			console.error('Error polling LuxureTV job progress:', error)
		}
	}, 2000) // Poll every 2 seconds

	// Store interval ID for cleanup
	activeTimeouts.value.push(pollInterval)
}

const fetchLuxureTVEmbedIDs = async () => {
	luxureTVFetchingEmbeds.value = true

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/luxuretv/fetch-embeds', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Started fetching embed IDs for ${data.data.videos_to_process} videos. Check server logs for progress.`)
			// Keep the spinner for a bit to indicate work is happening
			setTimeout(() => {
				luxureTVFetchingEmbeds.value = false
				loadLuxureTVVideos() // Reload to get updated metadata
			}, 5000)
		} else {
			toast.error(data.message || 'Failed to start fetching embed IDs')
			luxureTVFetchingEmbeds.value = false
		}
	} catch (error) {
		console.error('Error fetching embed IDs:', error)
		toast.error('Failed to fetch embed IDs')
		luxureTVFetchingEmbeds.value = false
	}
}

// Helper to get embed URL from video metadata
const getEmbedUrl = (video) => {
	if (!video.metadata || video.metadata === '{}') return null
	try {
		const meta = JSON.parse(video.metadata)
		return meta.embed_url || null
	} catch {
		return null
	}
}

// Play video in embedded player modal
const playEmbeddedVideo = (video) => {
	const embedUrl = getEmbedUrl(video)
	if (embedUrl) {
		scrapersAPI.openURL(embedUrl).catch(() => window.open(embedUrl, '_blank', 'width=900,height=600'))
	}
}

// Sexuria scraper methods
const loadSexuriaCategories = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/sexuria/categories')
		const data = await response.json()
		if (data.success && data.data) {
			sexuriaCategories.value = data.data
		}
	} catch (error) {
		console.error('Error loading Sexuria categories:', error)
	}
}

const loadSexuriaStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/sexuria/stats')
		const data = await response.json()
		if (data.success && data.data) {
			sexuriaStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading Sexuria stats:', error)
	}
}

const loadSexuriaItems = async () => {
	sexuriaLoading.value = true
	try {
		// Fetch with high limit to get all items (server now supports pagination)
		const response = await fetch('http://localhost:8080/api/v1/scraper/sexuria/items?limit=1000000')
		const data = await response.json()
		if (data.success && data.data) {
			sexuriaItems.value = data.data.items || []
		}
	} catch (error) {
		console.error('Error loading Sexuria items:', error)
	} finally {
		sexuriaLoading.value = false
	}
}

const clearSexuriaItems = async () => {
	if (!confirm(`Are you sure you want to delete all ${sexuriaItems.value.length} Sexuria items? This cannot be undone.`)) {
		return
	}
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/site/sexuria', {
			method: 'DELETE',
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} Sexuria items`)
			sexuriaItems.value = []
			await loadSexuriaStats()
		} else {
			toast.error(data.message || 'Failed to delete items')
		}
	} catch (error) {
		console.error('Error deleting Sexuria items:', error)
		toast.error('Failed to delete items')
	}
}

const startSexuriaScrape = async () => {
	if (!sexuriaCategoryURL.value) {
		toast.warning('Please enter a category URL')
		return
	}

	sexuriaScraping.value = true

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/sexuria/scrape', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				categoryUrl: sexuriaCategoryURL.value,
				maxPages: sexuriaMaxPages.value || 0,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Started Sexuria scrape for: ${sexuriaCategoryURL.value}`)

			// Set up active job tracking
			sexuriaActiveJob.value = {
				id: data.data.job_id,
				status: 'running',
				pagesCompleted: 0,
				itemsFound: 0,
				itemsSaved: 0,
				errorsCount: 0,
			}

			// Poll for job progress
			pollSexuriaJobProgress(data.data.job_id)
		} else {
			toast.error(data.message || 'Failed to start Sexuria scrape')
			sexuriaScraping.value = false
		}
	} catch (error) {
		console.error('Error starting Sexuria scrape:', error)
		toast.error('Failed to start Sexuria scrape')
		sexuriaScraping.value = false
	}
}

const pollSexuriaJobProgress = async (jobId) => {
	const pollInterval = setInterval(async () => {
		try {
			const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${jobId}`)
			const data = await response.json()

			if (data.success && data.data) {
				const job = data.data
				sexuriaActiveJob.value = {
					id: job.id,
					status: job.status,
					pagesCompleted: job.pagesCompleted || 0,
					itemsFound: job.videosFound || 0,
					itemsSaved: job.videosSaved || 0,
					errorsCount: job.errorsCount || 0,
				}

				// Check if job is complete
				if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
					clearInterval(pollInterval)
					sexuriaScraping.value = false

					if (job.status === 'completed') {
						toast.success(`Sexuria scrape complete! Found ${job.videosFound} items`)
					} else if (job.status === 'failed') {
						toast.error('Sexuria scrape failed')
					}

					// Reload items and stats
					await loadSexuriaItems()
					await loadSexuriaStats()

					// Clear active job after a delay
					setTimeout(() => {
						sexuriaActiveJob.value = null
					}, 5000)
				}
			}
		} catch (error) {
			console.error('Error polling Sexuria job progress:', error)
		}
	}, 2000) // Poll every 2 seconds

	// Store interval ID for cleanup
	activeTimeouts.value.push(pollInterval)
}

// Load Sexuria advanced search options (categories, file hosts, resolutions)
const loadSexuriaSearchOptions = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/sexuria/search-options')
		const data = await response.json()
		if (data.success && data.data) {
			sexuriaSearchOptions.value = {
				categories: data.data.categories || [],
				fileHosts: data.data.fileHosts || [],
				resolutions: data.data.resolutions || [],
			}
		}
	} catch (error) {
		console.error('Error loading Sexuria search options:', error)
	}
}

// Start Sexuria advanced search
const startSexuriaAdvancedSearch = async () => {
	// Validate that at least one search criterion is provided
	const search = sexuriaAdvancedSearch.value
	const hasSearchCriteria = search.categories.length > 0 || search.title.trim() || search.description.trim() || search.pornstars.trim() || search.production.trim() || search.tags.trim() || search.resolutions.length > 0 || search.fileHosts.length > 0

	if (!hasSearchCriteria) {
		toast.warning('Please provide at least one search criterion')
		return
	}

	sexuriaScraping.value = true

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/sexuria/advanced-search', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				categories: search.categories,
				title: search.title,
				description: search.description,
				pornstars: search.pornstars,
				production: search.production,
				tags: search.tags,
				resolutions: search.resolutions,
				fileHosts: search.fileHosts,
				maxPages: search.maxPages || 0,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Started Sexuria advanced search')

			// Set up active job tracking
			sexuriaActiveJob.value = {
				id: data.data.job_id,
				status: 'running',
				pagesCompleted: 0,
				itemsFound: 0,
				itemsSaved: 0,
				errorsCount: 0,
			}

			// Poll for job progress
			pollSexuriaJobProgress(data.data.job_id)
		} else {
			toast.error(data.message || 'Failed to start advanced search')
			sexuriaScraping.value = false
		}
	} catch (error) {
		console.error('Error starting Sexuria advanced search:', error)
		toast.error('Failed to start advanced search')
		sexuriaScraping.value = false
	}
}

// Reset advanced search form
const resetSexuriaAdvancedSearch = () => {
	sexuriaAdvancedSearch.value = {
		categories: [],
		title: '',
		description: '',
		pornstars: '',
		production: '',
		tags: '',
		resolutions: [],
		fileHosts: [],
		maxPages: 0,
	}
}

// Toggle category selection for advanced search
const toggleSexuriaSearchCategory = (categoryId) => {
	const idx = sexuriaAdvancedSearch.value.categories.indexOf(categoryId)
	if (idx >= 0) {
		sexuriaAdvancedSearch.value.categories.splice(idx, 1)
	} else {
		sexuriaAdvancedSearch.value.categories.push(categoryId)
	}
}

// Toggle file host selection for advanced search
const toggleSexuriaSearchFileHost = (hostId) => {
	const idx = sexuriaAdvancedSearch.value.fileHosts.indexOf(hostId)
	if (idx >= 0) {
		sexuriaAdvancedSearch.value.fileHosts.splice(idx, 1)
	} else {
		sexuriaAdvancedSearch.value.fileHosts.push(hostId)
	}
}

// Toggle resolution selection for advanced search
const toggleSexuriaSearchResolution = (resolution) => {
	const idx = sexuriaAdvancedSearch.value.resolutions.indexOf(resolution)
	if (idx >= 0) {
		sexuriaAdvancedSearch.value.resolutions.splice(idx, 1)
	} else {
		sexuriaAdvancedSearch.value.resolutions.push(resolution)
	}
}

// Helper to get file size from Sexuria item metadata
const getSexuriaFileSize = (item) => {
	if (!item.metadata || item.metadata === '{}') return null
	try {
		const meta = JSON.parse(item.metadata)
		return meta.file_size || null
	} catch {
		return null
	}
}

// Open Sexuria item in browser
const openSexuriaItem = (item) => {
	if (item.url) {
		scrapersAPI.openURL(item.url).catch(() => window.open(item.url, '_blank'))
	}
}

// Sexuria Hover Card Methods
const handleSexuriaItemHover = (item, event) => {
	// Clear any pending hide timeout
	if (sexuriaHideTimeout) {
		clearTimeout(sexuriaHideTimeout)
		sexuriaHideTimeout = null
	}

	// Clear any pending show timeout
	if (sexuriaHoverTimeout) {
		clearTimeout(sexuriaHoverTimeout)
	}

	// Delay showing the hover card
	sexuriaHoverTimeout = setTimeout(() => {
		showSexuriaDetails(item, event)
	}, 300)
}

const showSexuriaDetails = async (item, event) => {
	// Position the hover card near the cursor
	const rect = event.target.getBoundingClientRect()
	const viewportWidth = window.innerWidth
	const viewportHeight = window.innerHeight

	// Card dimensions (approximate)
	const cardWidth = 350
	const cardHeight = 400

	// Calculate position - prefer right side, but flip if not enough space
	let left = rect.right + 10
	let top = rect.top

	if (left + cardWidth > viewportWidth) {
		left = rect.left - cardWidth - 10
	}
	if (top + cardHeight > viewportHeight) {
		top = viewportHeight - cardHeight - 10
	}
	if (top < 10) {
		top = 10
	}

	sexuriaHoverDetails.value = {
		visible: true,
		item: item,
		data: null,
		loading: true,
		error: null,
		style: {
			left: `${left}px`,
			top: `${top}px`,
		},
	}

	// Check cache first
	if (sexuriaDetailsCache.value[item.url]) {
		sexuriaHoverDetails.value.data = sexuriaDetailsCache.value[item.url]
		sexuriaHoverDetails.value.loading = false
		return
	}

	// Fetch details
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/sexuria/item-details', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ url: item.url }),
		})

		if (!response.ok) throw new Error('Failed to fetch details')

		const data = await response.json()
		if (data.success && data.data) {
			// Cache the result
			sexuriaDetailsCache.value[item.url] = data.data
			// Only update if this is still the same item
			if (sexuriaHoverDetails.value.item?.url === item.url) {
				sexuriaHoverDetails.value.data = data.data
			}
		}
	} catch (error) {
		console.error('Failed to fetch Sexuria details:', error)
		if (sexuriaHoverDetails.value.item?.url === item.url) {
			sexuriaHoverDetails.value.error = 'Failed to load details'
		}
	} finally {
		if (sexuriaHoverDetails.value.item?.url === item.url) {
			sexuriaHoverDetails.value.loading = false
		}
	}
}

const keepSexuriaDetailsVisible = () => {
	// Clear any pending hide timeout when entering the card
	if (sexuriaHideTimeout) {
		clearTimeout(sexuriaHideTimeout)
		sexuriaHideTimeout = null
	}
}

const hideSexuriaDetails = () => {
	// Clear any pending show timeout
	if (sexuriaHoverTimeout) {
		clearTimeout(sexuriaHoverTimeout)
		sexuriaHoverTimeout = null
	}

	// Delay hiding to allow mouse to enter the card
	sexuriaHideTimeout = setTimeout(() => {
		sexuriaHoverDetails.value.visible = false
		sexuriaHoverDetails.value.item = null
		sexuriaHoverDetails.value.data = null
		sexuriaHoveredScreenshot.value = null
	}, 200)
}

const openScreenshot = (url) => {
	// Open screenshot in modal with all screenshots from the hover card
	const screenshots = sexuriaHoverDetails.value.data?.screenshots || []
	const index = screenshots.indexOf(url)
	sexuriaScreenshotModal.value = {
		visible: true,
		screenshots: screenshots,
		currentIndex: index >= 0 ? index : 0,
	}
}

const closeSexuriaScreenshotModal = () => {
	sexuriaScreenshotModal.value.visible = false
}

const prevSexuriaScreenshot = () => {
	const { screenshots, currentIndex } = sexuriaScreenshotModal.value
	if (screenshots.length === 0) return
	sexuriaScreenshotModal.value.currentIndex = currentIndex <= 0 ? screenshots.length - 1 : currentIndex - 1
}

const nextSexuriaScreenshot = () => {
	const { screenshots, currentIndex } = sexuriaScreenshotModal.value
	if (screenshots.length === 0) return
	sexuriaScreenshotModal.value.currentIndex = currentIndex >= screenshots.length - 1 ? 0 : currentIndex + 1
}

const currentSexuriaScreenshot = computed(() => {
	const { screenshots, currentIndex } = sexuriaScreenshotModal.value
	return screenshots[currentIndex] || null
})

// Sexuria Link Management Methods
const extractSexuriaItemLinks = async (item) => {
	if (!item.url) return

	extractingLinks.value[item.id] = true
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/sexuria/items/${item.id}/extract-links`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ url: item.url }),
		})

		if (!response.ok) throw new Error('Failed to extract links')

		const data = await response.json()
		if (data.success) {
			// Update the item to show links were extracted
			item.links_extracted = true

			// Store link stats
			const links = data.data.links || []
			const stats = {
				total: links.length,
				working: links.filter((l) => l.status === 'working').length,
				dead: links.filter((l) => l.status === 'dead').length,
				unknown: links.filter((l) => l.status === 'unknown').length,
			}
			sexuriaLinkStats.value[item.id] = stats

			toast.success(`Extracted ${links.length} download links`)
		}
	} catch (error) {
		console.error('Failed to extract links:', error)
		toast.error('Failed to extract links')
	} finally {
		extractingLinks.value[item.id] = false
	}
}

const showSexuriaItemLinks = async (item) => {
	sexuriaLinksModal.value.visible = true
	sexuriaLinksModal.value.item = item
	sexuriaLinksModal.value.loading = true
	sexuriaLinksModal.value.links = []

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/sexuria/items/${item.id}/links`)
		if (!response.ok) throw new Error('Failed to get links')

		const data = await response.json()
		if (data.success) {
			sexuriaLinksModal.value.links = data.data.links || []

			// Update link stats
			const links = sexuriaLinksModal.value.links
			sexuriaLinkStats.value[item.id] = {
				total: links.length,
				working: links.filter((l) => l.status === 'working').length,
				dead: links.filter((l) => l.status === 'dead').length,
				unknown: links.filter((l) => l.status === 'unknown').length,
			}
		}
	} catch (error) {
		console.error('Failed to load links:', error)
		toast.error('Failed to load links')
	} finally {
		sexuriaLinksModal.value.loading = false
	}
}

const verifySexuriaItemLinks = async () => {
	const item = sexuriaLinksModal.value.item
	if (!item) return

	sexuriaLinksModal.value.verifying = true
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/sexuria/items/${item.id}/verify-links`, {
			method: 'POST',
		})

		if (!response.ok) throw new Error('Failed to verify links')

		const data = await response.json()
		if (data.success) {
			toast.success('Links verified')
			// Reload links to get updated statuses
			await showSexuriaItemLinks(item)
		}
	} catch (error) {
		console.error('Failed to verify links:', error)
		toast.error('Failed to verify links')
	} finally {
		sexuriaLinksModal.value.verifying = false
	}
}

const closeSexuriaLinksModal = () => {
	sexuriaLinksModal.value.visible = false
	sexuriaLinksModal.value.item = null
	sexuriaLinksModal.value.links = []
}

const getLinkHealthBadgeClass = (stats) => {
	if (!stats || stats.total === 0) return 'bg-secondary'
	const workingPct = (stats.working / stats.total) * 100
	const deadPct = (stats.dead / stats.total) * 100

	if (deadPct > 50) return 'bg-danger'
	if (deadPct > 25) return 'bg-warning text-dark'
	if (workingPct > 75) return 'bg-success'
	if (stats.unknown > 0) return 'bg-info'
	return 'bg-secondary'
}

const getLinkHealthTitle = (stats) => {
	if (!stats) return 'No link data'
	return `${stats.working} working, ${stats.dead} dead, ${stats.unknown} unknown`
}

const getLinkStatusBadgeClass = (status) => {
	switch (status) {
		case 'working':
			return 'bg-success'
		case 'dead':
			return 'bg-danger'
		case 'unknown':
			return 'bg-secondary'
		default:
			return 'bg-secondary'
	}
}

const copyLinkToClipboard = (url) => {
	navigator.clipboard.writeText(url)
	toast.success('Link copied to clipboard')
}

// Toggle tag filter selection
const toggleSexuriaTagFilter = (tag) => {
	const idx = sexuriaTagFilter.value.indexOf(tag)
	if (idx >= 0) {
		sexuriaTagFilter.value.splice(idx, 1)
	} else {
		sexuriaTagFilter.value.push(tag)
	}
	// Reset to page 1 when filter changes
	sexuriaItemPage.value = 1
}

// Toggle pornstar filter selection
const toggleSexuriaPornstarFilter = (star) => {
	const idx = sexuriaPornstarFilter.value.indexOf(star)
	if (idx >= 0) {
		sexuriaPornstarFilter.value.splice(idx, 1)
	} else {
		sexuriaPornstarFilter.value.push(star)
	}
	// Reset to page 1 when filter changes
	sexuriaItemPage.value = 1
}

// Rule34Video scraper methods
const loadRule34VideoStats = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/rule34video/stats')
		const data = await response.json()
		if (data.success) {
			rule34VideoStats.value = data.data
		}
	} catch (error) {
		console.error('Error loading Rule34Video stats:', error)
	}
}

const loadRule34VideoVideos = async () => {
	rule34VideoLoading.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/rule34video/videos?limit=1000000')
		const data = await response.json()
		if (data.success) {
			rule34VideoVideos.value = data.data.videos || []
		}
	} catch (error) {
		console.error('Error loading Rule34Video videos:', error)
	} finally {
		rule34VideoLoading.value = false
	}
}

const clearRule34VideoVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${rule34VideoVideos.value.length} Rule34Video videos? This cannot be undone.`)) {
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/site/rule34video', {
			method: 'DELETE',
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} Rule34Video videos`)
			rule34VideoVideos.value = []
			await loadRule34VideoStats()
		}
	} catch (error) {
		console.error('Error deleting Rule34Video videos:', error)
		toast.error('Failed to delete videos')
	}
}

// Rule34Video batch operations
const toggleRule34VideoSelectionMode = () => {
	rule34VideoSelectionMode.value = !rule34VideoSelectionMode.value
	if (!rule34VideoSelectionMode.value) {
		rule34VideoSelectedVideos.value.clear()
	}
}

const toggleRule34VideoVideoSelection = (videoId) => {
	if (rule34VideoSelectedVideos.value.has(videoId)) {
		rule34VideoSelectedVideos.value.delete(videoId)
	} else {
		rule34VideoSelectedVideos.value.add(videoId)
	}
	rule34VideoSelectedVideos.value = new Set(rule34VideoSelectedVideos.value) // Force reactivity
}

const selectAllRule34VideoVideos = () => {
	paginatedRule34VideoVideos.value.forEach((v) => rule34VideoSelectedVideos.value.add(v.id))
	rule34VideoSelectedVideos.value = new Set(rule34VideoSelectedVideos.value)
}

const deleteSelectedRule34VideoVideos = async () => {
	if (rule34VideoSelectedVideos.value.size === 0) return
	if (!confirm(`Delete ${rule34VideoSelectedVideos.value.size} selected videos?`)) return

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos/bulk-delete', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ ids: Array.from(rule34VideoSelectedVideos.value) }),
		})
		const data = await response.json()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} videos`)
			rule34VideoSelectedVideos.value.clear()
			rule34VideoSelectionMode.value = false
			await loadRule34VideoVideos()
			await loadRule34VideoStats()
		}
	} catch (error) {
		console.error('Error deleting videos:', error)
		toast.error('Failed to delete videos')
	}
}

const startRule34VideoScrape = async () => {
	rule34VideoScraping.value = true

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/rule34video/scrape', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				maxPages: rule34VideoMaxPages.value || 0,
				sortBy: rule34VideoSortBy.value,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Started Rule34Video scrape (${rule34VideoSortBy.value}, Straight filter)`)

			// Set active job for UI
			rule34VideoActiveJob.value = {
				id: data.data.job_id,
				status: 'running',
				sortBy: data.data.sort_by,
				pagesCompleted: 0,
				videosFound: 0,
				videosSaved: 0,
				errorsCount: 0,
			}

			// Poll for progress
			pollRule34VideoJobProgress(data.data.job_id)
		} else {
			toast.error(data.message || 'Failed to start Rule34Video scrape')
			rule34VideoScraping.value = false
		}
	} catch (error) {
		console.error('Error starting Rule34Video scrape:', error)
		toast.error('Failed to start Rule34Video scrape')
		rule34VideoScraping.value = false
	}
}

const pollRule34VideoJobProgress = async (jobId) => {
	const pollInterval = setInterval(async () => {
		try {
			const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${jobId}`)
			const data = await response.json()

			if (data.success) {
				const job = data.data
				rule34VideoActiveJob.value = {
					...rule34VideoActiveJob.value,
					status: job.status,
					pagesCompleted: job.pagesCompleted,
					videosFound: job.videosFound,
					videosSaved: job.videosSaved,
					errorsCount: job.errorsCount,
				}

				if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
					clearInterval(pollInterval)
					rule34VideoScraping.value = false
					if (job.status === 'completed') {
						toast.success(`Rule34Video scrape complete! Found ${job.videosFound} videos`)
					} else if (job.status === 'failed') {
						toast.error('Rule34Video scrape failed')
					}
					// Clear active job after a delay
					setTimeout(() => {
						rule34VideoActiveJob.value = null
					}, 3000)
					await loadRule34VideoVideos()
					await loadRule34VideoStats()
				}
			}
		} catch (error) {
			console.error('Error polling Rule34Video job progress:', error)
		}
	}, 2000)
}

const startForumScrape = async () => {
	if (!forumURL.value) {
		toast.warning('Please enter a forum URL')
		return
	}

	if (!sessionCookieSet.value) {
		toast.warning('Please set authentication cookie first')
		showForumScrapeModal.value = false
		showAuthModal.value = true
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/forum/scrape-all', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				url: forumURL.value,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Forum scraping started! This will take a while. Check server logs for progress.')
			showForumScrapeModal.value = false

			// Reload threads periodically
			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 5000)
		} else {
			toast.error(data.message || 'Failed to start forum scraping')
		}
	} catch (error) {
		console.error('Error starting forum scrape:', error)
		toast.error('Failed to start forum scraping')
	}
}

const autoLinkPerformers = async () => {
	try {
		toast.info('Auto-linking threads to performers...')

		const response = await fetch('http://localhost:8080/api/v1/scraper/performers/auto-link', {
			method: 'POST',
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Auto-linking started! Check server logs for progress. Performers will be created/linked automatically.')

			// Reload threads after a delay
			setTimeout(() => {
				loadThreads()
				loadStats()
			}, 3000)
		} else {
			toast.error(data.message || 'Failed to start auto-linking')
		}
	} catch (error) {
		console.error('Error auto-linking performers:', error)
		toast.error('Failed to start auto-linking')
	}
}

const checkLinkStatuses = async () => {
	try {
		toast.info('Checking link statuses...')

		const response = await fetch('http://localhost:8080/api/v1/scraper/links/check-status', {
			method: 'POST',
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Link status check started! This will take a while. Check server logs for progress.')
		} else {
			toast.error(data.message || 'Failed to start link check')
		}
	} catch (error) {
		console.error('Error checking link statuses:', error)
		toast.error('Failed to start link status check')
	}
}

// Selection methods
const toggleSelectionMode = () => {
	selectionMode.value = !selectionMode.value
	if (!selectionMode.value) {
		selectedThreadIds.value.clear()
	}
}

const toggleThreadSelection = (threadId, event) => {
	event.stopPropagation()
	if (selectedThreadIds.value.has(threadId)) {
		selectedThreadIds.value.delete(threadId)
	} else {
		selectedThreadIds.value.add(threadId)
	}
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const selectAllThreads = () => {
	threads.value.forEach((thread) => {
		selectedThreadIds.value.add(thread.id)
	})
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const deselectAllThreads = () => {
	selectedThreadIds.value.clear()
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const isThreadSelected = (threadId) => {
	return selectedThreadIds.value.has(threadId)
}

const selectedCount = computed(() => selectedThreadIds.value.size)

const allSelectedOnPage = computed(() => {
	if (threads.value.length === 0) return false
	return threads.value.every((thread) => selectedThreadIds.value.has(thread.id))
})

// Deletion methods
// Bulk Operations
const bulkOperationInProgress = ref(false)

// Thread Link Health Methods
const getThreadLinkHealthClass = (stats) => {
	if (!stats || stats.total === 0) return ''
	const workingPct = (stats.working / stats.total) * 100
	const deadPct = (stats.dead / stats.total) * 100

	if (deadPct > 50) return 'text-danger'
	if (deadPct > 25) return 'text-warning'
	if (workingPct > 75) return 'text-success'
	return ''
}

const getThreadLinkHealthTitle = (stats) => {
	if (!stats) return 'No link data'
	return `${stats.working} working, ${stats.dead} dead, ${stats.unknown} unchecked`
}

const fetchThreadLinkStats = async (threadId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${threadId}/link-stats`)
		if (!response.ok) return

		const data = await response.json()
		if (data.success && data.data) {
			threadLinkStats.value[threadId] = {
				total: data.data.total || 0,
				working: data.data.working || 0,
				dead: data.data.dead || 0,
				unknown: data.data.unchecked || data.data.unknown || 0,
			}
		}
	} catch (error) {
		console.error(`Failed to fetch link stats for thread ${threadId}:`, error)
	}
}

const verifyThreadLinks = async (thread) => {
	try {
		toast.info('Starting link verification...')
		const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${thread.id}/verify-links`, {
			method: 'POST',
		})

		if (!response.ok) throw new Error('Failed to verify links')

		const data = await response.json()
		if (data.success) {
			toast.success('Link verification started in background')
			// Refresh stats after a delay
			setTimeout(() => fetchThreadLinkStats(thread.id), 5000)
		}
	} catch (error) {
		console.error('Failed to verify thread links:', error)
		toast.error('Failed to start link verification')
	}
}

const bulkVerifyLinks = async () => {
	if (selectedThreadIds.value.size === 0) {
		toast.warning('No threads selected')
		return
	}

	bulkOperationInProgress.value = true
	const threadIds = Array.from(selectedThreadIds.value)
	let completed = 0

	try {
		toast.info(`Starting verification for ${threadIds.length} threads...`)

		// Verify each thread sequentially with progress updates
		for (const threadId of threadIds) {
			try {
				const response = await fetch(`http://localhost:8080/api/v1/scraper/threads/${threadId}/verify-links`, {
					method: 'POST',
				})
				const data = await response.json()

				if (data.success) {
					completed++
					toast.success(`Verified thread ${completed}/${threadIds.length}`, {
						duration: 2000,
					})
				}
			} catch (error) {
				console.error(`Failed to verify thread ${threadId}:`, error)
			}
		}

		toast.success(`✅ Completed! Verified ${completed}/${threadIds.length} threads`)
		selectedThreadIds.value.clear()
		selectionMode.value = false
	} catch (error) {
		console.error('Bulk verify error:', error)
		toast.error('Bulk verification failed: ' + error.message)
	} finally {
		bulkOperationInProgress.value = false
	}
}

const bulkSendToJDownloader = async () => {
	if (selectedThreadIds.value.size === 0) {
		toast.warning('No threads selected')
		return
	}

	bulkOperationInProgress.value = true
	const threadIds = Array.from(selectedThreadIds.value)
	let completed = 0
	let totalLinksSent = 0

	try {
		// First check if JDownloader is available
		const jdStatusResponse = await fetch('http://localhost:8080/api/v1/jdownloader/status')
		const jdStatusData = await jdStatusResponse.json()

		if (!jdStatusData.success || !jdStatusData.data.available) {
			toast.error('JDownloader is not running! Please start JDownloader first.')
			return
		}

		toast.info(`Sending ${threadIds.length} threads to JDownloader...`)

		// Send each thread to JDownloader
		for (const threadId of threadIds) {
			try {
				const response = await fetch(`http://localhost:8080/api/v1/jdownloader/threads/${threadId}/send`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						only_active: true,
						only_pending: true,
						auto_start: false,
					}),
				})
				const data = await response.json()

				if (data.success) {
					completed++
					totalLinksSent += data.data.links_sent || 0
					toast.success(`Sent thread ${completed}/${threadIds.length} (${data.data.links_sent} links)`, {
						duration: 2000,
					})
				}
			} catch (error) {
				console.error(`Failed to send thread ${threadId}:`, error)
			}
		}

		toast.success(`🎉 Sent ${totalLinksSent} links from ${completed}/${threadIds.length} threads to JDownloader!`)
		selectedThreadIds.value.clear()
		selectionMode.value = false
	} catch (error) {
		console.error('Bulk JDownloader send error:', error)
		toast.error('Bulk send failed: ' + error.message)
	} finally {
		bulkOperationInProgress.value = false
	}
}

const deleteSelectedThreads = async () => {
	if (selectedThreadIds.value.size === 0) {
		toast.warning('No threads selected')
		return
	}
	showDeleteConfirmModal.value = true
}

const confirmDeleteSelected = async () => {
	const threadIds = Array.from(selectedThreadIds.value)

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/threads', {
			method: 'DELETE',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				thread_ids: threadIds,
			}),
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Successfully deleted ${threadIds.length} thread(s)`)
			selectedThreadIds.value.clear()
			selectionMode.value = false
			showDeleteConfirmModal.value = false
			loadThreads()
			loadStats()
		} else {
			toast.error(data.message || 'Failed to delete threads')
		}
	} catch (error) {
		console.error('Error deleting threads:', error)
		toast.error('Failed to delete selected threads')
	}
}

const deleteAllThreads = async () => {
	showDeleteAllConfirmModal.value = true
}

const confirmDeleteAll = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/threads/all', {
			method: 'DELETE',
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Successfully deleted all threads')
			selectedThreadIds.value.clear()
			selectionMode.value = false
			showDeleteAllConfirmModal.value = false
			loadThreads()
			loadStats()
		} else {
			toast.error(data.message || 'Failed to delete all threads')
		}
	} catch (error) {
		console.error('Error deleting all threads:', error)
		toast.error('Failed to delete all threads')
	}
}

// Thumbnail fallback handler
const handleThumbnailError = (event) => {
	const img = event.target
	const fallbackURLs = JSON.parse(img.dataset.fallbackUrls || '[]')

	// Get current src index in fallback array
	const currentIndex = fallbackURLs.indexOf(img.src)

	// Try next fallback URL
	if (currentIndex < fallbackURLs.length - 1) {
		img.src = fallbackURLs[currentIndex + 1]
	} else {
		// All fallbacks failed, hide image and show placeholder
		img.style.display = 'none'
		const placeholder = img.parentElement.nextElementSibling
		if (placeholder && placeholder.classList.contains('placeholder')) {
			placeholder.style.display = 'flex'
		}
	}
}

// Video scraper functions
const scrapeVideo = async () => {
	if (!videoUrl.value) {
		toast.error('Please enter a video URL')
		return
	}

	videoScraping.value = true
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/video', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				url: videoUrl.value,
				site_type: videoSiteType.value,
				options: videoScrapeOptions.value,
			}),
		})

		const data = await response.json()

		if (data.success) {
			// Check if it's a page scrape (multiple videos) or single video
			if (data.data.videos && Array.isArray(data.data.videos)) {
				// Page scrape result
				const result = data.data
				toast.success(`Scraped ${result.total_scraped} videos from ${result.pages_crawled} page(s)!`)
				scrapedVideos.value.push(...result.videos)
			} else {
				// Single video result
				toast.success(`Video scraped successfully!`)
				scrapedVideos.value.push(data.data)
			}
			videoUrl.value = ''

			// Load all scraped videos to refresh the list
			loadScrapedVideos()
		} else {
			toast.error(data.message || 'Failed to scrape video')
		}
	} catch (error) {
		console.error('Error scraping video:', error)
		toast.error('Failed to scrape video: ' + error.message)
	} finally {
		videoScraping.value = false
	}
}

const viewVideo = (video) => {
	const url = video.url
	if (url) scrapersAPI.openURL(url).catch(() => window.open(url, '_blank'))
}

const loadScrapedVideos = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos')
		const data = await response.json()

		if (data.success && data.data) {
			scrapedVideos.value = data.data
		}
	} catch (error) {
		console.error('Error loading scraped videos:', error)
	}
}

const testScraper = async () => {
	if (!videoUrl.value) {
		toast.error('Please enter a URL')
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/video/test?url=${encodeURIComponent(videoUrl.value)}`)
		const data = await response.json()

		if (data.success) {
			const result = data.data
			console.log('Test scraper result:', result)

			// Show results in console and toast
			toast.success(`Test found ${result.total_found} videos. Check console for details.`)

			// Log selectors tried
			console.log('Selectors tried:', result.selectors_tried)
			console.log('Videos found:', result.videos_found)
		} else {
			toast.error(data.message || 'Test failed')
		}
	} catch (error) {
		console.error('Error testing scraper:', error)
		toast.error('Failed to test scraper: ' + error.message)
	}
}

const clearAllVideos = async () => {
	if (!confirm(`Are you sure you want to delete all ${scrapedVideos.value.length} scraped videos? This cannot be undone.`)) {
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/videos', {
			method: 'DELETE',
		})
		const data = await response.json()

		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} videos`)
			scrapedVideos.value = []
		} else {
			toast.error(data.message || 'Failed to delete videos')
		}
	} catch (error) {
		console.error('Error deleting videos:', error)
		toast.error('Failed to delete videos: ' + error.message)
	}
}

// Scraper Tabs Functions
const loadScraperTabs = async () => {
	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/tabs')
		const data = await response.json()

		if (data.success && data.data) {
			scraperTabs.value = data.data
			// Initialize URLs, jobs, and video controls for each tab
			data.data.forEach((tab) => {
				if (!customTabUrls.value[tab.id]) {
					customTabUrls.value[tab.id] = tab.baseUrl
				}
				// Initialize video controls
				if (!videoSort.value[tab.id]) {
					videoSort.value[tab.id] = 'newest'
				}
				if (!videoPage.value[tab.id]) {
					videoPage.value[tab.id] = 1
				}
				if (!videoSearch.value[tab.id]) {
					videoSearch.value[tab.id] = ''
				}
				// Start collapsed by default (same as built-in scrapers)
				const key = `tab-${tab.id}`
				if (collapsedSections.value[key] === undefined) {
					collapsedSections.value[key] = true
				}
			})
		}
	} catch (error) {
		console.error('Error loading scraper tabs:', error)
	}
}

const selectTab = (tabId) => {
	activeTab.value = `tab-${tabId}`
	loadTabJobProgress(tabId)
	loadVideosForTab(tabId)
}

// Helper to get the endpoint type for auto-update based on site type
// eslint-disable-next-line no-unused-vars
const getEndpointTypeForSiteType = (siteType) => {
	switch (siteType) {
		case 'pornhub':
			return 'keywords'
		// All other tabs (LuxureTV, Sexuria, generic) use their baseUrl directly
		// No additional endpoint configuration needed
		default:
			return null
	}
}

const toggleVideoFilters = (tabId) => {
	showVideoFilters.value[tabId] = !showVideoFilters.value[tabId]
}

const resetVideoFilters = (tabId) => {
	videoSearch.value[tabId] = ''
	videoSort.value[tabId] = 'newest'
	videoStatusFilter.value[tabId] = ''
	videoResolutionFilter.value[tabId] = ''
	videoDurationFilter.value[tabId] = ''
	videoThumbnailFilter.value[tabId] = ''
	videoOnlyDownloaded.value[tabId] = false
	videoAuthorFilter.value[tabId] = ''
	videoMinViews.value[tabId] = 0
	videoPage.value[tabId] = 1
	loadVideosForTab(tabId)
}

const confirmDeleteTab = async (tab) => {
	if (!confirm(`Are you sure you want to delete the "${tab.name}" tab? This cannot be undone.`)) {
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/tabs/${tab.id}`, {
			method: 'DELETE',
		})
		const data = await response.json()

		if (data.success) {
			toast.success(`Tab "${tab.name}" deleted successfully`)

			// Switch to simpcity tab if we're deleting the active tab
			if (activeTab.value === `tab-${tab.id}`) {
				activeTab.value = 'simpcity'
			}

			// Reload tabs
			await loadScraperTabs()
		} else {
			toast.error(data.message || 'Failed to delete tab')
		}
	} catch (error) {
		console.error('Error deleting tab:', error)
		toast.error('Failed to delete tab: ' + error.message)
	}
}

// eslint-disable-next-line no-unused-vars
const editScraperTab = (tab) => {
	editingTab.value = tab
	newTab.value = {
		name: tab.name,
		baseUrl: tab.baseUrl,
		siteType: tab.siteType,
		icon: tab.icon,
		color: tab.color,
		contentType: tab.contentType || 'regular',
		scrapeOptions: { ...tab.scrapeOptions },
		isActive: tab.isActive !== false, // Preserve active state, default to true
		position: tab.position || 0, // Preserve position
	}
	showEditTabModal.value = true
}

const closeTabModal = () => {
	showAddTabModal.value = false
	showEditTabModal.value = false
	editingTab.value = null
	newTab.value = {
		name: '',
		baseUrl: '',
		siteType: 'generic',
		icon: 'video',
		color: '#007bff',
		contentType: 'regular',
		scrapeOptions: {
			extractMetadata: true,
			scrapeThumbnails: true,
			downloadVideo: false,
		},
	}
}

const saveTabConfig = async () => {
	if (!newTab.value.name || !newTab.value.baseUrl) {
		toast.error('Please fill in all required fields')
		return
	}

	try {
		const url = showEditTabModal.value ? `http://localhost:8080/api/v1/scraper/tabs/${editingTab.value.id}` : 'http://localhost:8080/api/v1/scraper/tabs'

		const method = showEditTabModal.value ? 'PUT' : 'POST'

		const response = await fetch(url, {
			method,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(newTab.value),
		})

		const data = await response.json()

		if (data.success) {
			toast.success(`Tab ${showEditTabModal.value ? 'updated' : 'created'} successfully`)
			closeTabModal()
			await loadScraperTabs()
		} else {
			toast.error(data.message || `Failed to ${showEditTabModal.value ? 'update' : 'create'} tab`)
		}
	} catch (error) {
		console.error('Error saving tab:', error)
		toast.error('Failed to save tab: ' + error.message)
	}
}

// ── Per-tab selection helpers ─────────────────────────────────────────────────

const toggleTabSelectionMode = (tabId) => {
	customTabSelectionMode.value[tabId] = !customTabSelectionMode.value[tabId]
	if (!customTabSelectionMode.value[tabId]) {
		customTabSelectedVideos.value[tabId] = new Set()
	}
}

const toggleTabVideoSelection = (tabId, videoId) => {
	const current = customTabSelectedVideos.value[tabId] || new Set()
	const next = new Set(current)
	if (next.has(videoId)) next.delete(videoId)
	else next.add(videoId)
	customTabSelectedVideos.value[tabId] = next
}

const selectAllTabVideos = (tabId) => {
	const videos = customTabVideos.value[tabId] || []
	customTabSelectedVideos.value[tabId] = new Set(videos.map((v) => v.id))
}

const sendTabSelectedToJD = async (tab) => {
	const tabId = tab.id
	const selectedIds = customTabSelectedVideos.value[tabId]
	if (!selectedIds || selectedIds.size === 0) {
		toast.warning('No videos selected')
		return
	}

	const allVideos = customTabVideos.value[tabId] || []
	const links = allVideos
		.filter((v) => selectedIds.has(v.id) && (v.source_url || v.url))
		.map((v) => ({
			url: v.source_url || v.url,
			packageName: v.title || '',
			downloadPath: `/output/${tab.name}`,
			videoId: v.id,
		}))

	if (links.length === 0) {
		toast.warning('No valid URLs in selection')
		return
	}

	customTabJDSending.value[tabId] = true
	try {
		const result = await jdownloaderAPI.sendLinks(links)
		if (result?.successCount > 0) {
			toast.success(`Sent ${result.successCount} link(s) to JDownloader`)
			customTabSelectedVideos.value[tabId] = new Set()
			customTabSelectionMode.value[tabId] = false
		} else {
			toast.error(result?.results?.[0]?.message || 'Failed to send links')
		}
	} catch (err) {
		console.error('Bulk JD send error:', err)
		toast.error('Failed to send to JDownloader')
	} finally {
		customTabJDSending.value[tabId] = false
	}
}

const deleteTabSelectedVideos = async (tab) => {
	const tabId = tab.id
	const selectedIds = customTabSelectedVideos.value[tabId]
	if (!selectedIds || selectedIds.size === 0) return

	const ids = Array.from(selectedIds)
	if (!confirm(`Delete ${ids.length} selected video(s)? This cannot be undone.`)) return

	try {
		await scrapersAPI.bulkDelete(ids)
		toast.success(`Deleted ${ids.length} video(s)`)
		customTabSelectedVideos.value[tabId] = new Set()
		customTabSelectionMode.value[tabId] = false
		await loadVideosForTab(tabId)
	} catch (err) {
		console.error('Error deleting selected videos:', err)
		toast.error('Failed to delete videos')
	}
}

const clearCustomTabVideos = async (tab) => {
	if (!confirm(`Delete ALL scraped videos for "${tab.name}"? This cannot be undone.`)) return
	try {
		const source = new URL(tab.baseUrl).host
		await deleteVideosBySource(source)
		customTabVideos.value[tab.id] = []
		toast.success(`Deleted all videos for ${tab.name}`)
	} catch (err) {
		console.error('Failed to delete videos:', err)
		toast.error('Failed to delete videos')
	}
}

const startCustomTabScrape = async (tab) => {
	const url = customTabUrls.value[tab.id]

	if (!url) {
		toast.error('Please enter a URL to scrape')
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/jobs/start', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				tabId: tab.id,
				url: url,
			}),
		})

		const data = await response.json()

		if (data.success) {
			customTabJobs.value[tab.id] = data.data
			toast.success('Scraping job started! It will run in the background.')

			// Poll for progress
			pollJobProgress(tab.id, data.data.id)
		} else {
			toast.error(data.message || 'Failed to start scraping job')
		}
	} catch (error) {
		console.error('Error starting scrape:', error)
		toast.error('Failed to start scraping: ' + error.message)
	}
}

const loadTabJobProgress = async (tabId) => {
	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs?tabId=${tabId}`)
		const data = await response.json()

		if (data.success && data.data && data.data.length > 0) {
			// Get the most recent job
			const latestJob = data.data[0]

			// Only show job status if it's actually running or pending
			// Don't show completed/failed jobs to avoid confusion
			if (latestJob.status === 'running' || latestJob.status === 'pending') {
				customTabJobs.value[tabId] = latestJob
				// Continue polling
				const timeoutId = setTimeout(() => loadTabJobProgress(tabId), 3000)
				activeTimeouts.value.push(timeoutId)
			} else {
				// Clear the job status if it's completed/failed
				customTabJobs.value[tabId] = null
			}
		} else {
			// No jobs found, clear status
			customTabJobs.value[tabId] = null
		}
	} catch (error) {
		console.error('Error loading job progress:', error)
	}
}

const pollJobProgress = async (tabId, jobId) => {
	const checkProgress = async () => {
		try {
			const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${jobId}`)
			const data = await response.json()

			if (data.success && data.data) {
				customTabJobs.value[tabId] = data.data

				// Continue polling if still running
				if (data.data.status === 'running' || data.data.status === 'pending') {
					const timeoutId = setTimeout(checkProgress, 3000)
					activeTimeouts.value.push(timeoutId)
				} else if (data.data.status === 'completed') {
					toast.success(`Scraping completed! Found ${data.data.videosFound} videos.`)
					loadVideosForTab(tabId)
				} else if (data.data.status === 'failed') {
					toast.error('Scraping job failed. Check the error messages for details.')
				}
			}
		} catch (error) {
			console.error('Error polling job progress:', error)
		}
	}

	checkProgress()
}

// eslint-disable-next-line no-unused-vars
const cancelJob = async (job) => {
	if (!confirm('Are you sure you want to cancel this scraping job?')) {
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${job.id}/cancel`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Job cancelled successfully')
			// Clear the job from display
			customTabJobs.value[job.tabId] = null
		} else {
			toast.error(data.message || 'Failed to cancel job')
		}
	} catch (error) {
		console.error('Error cancelling job:', error)
		toast.error('Failed to cancel job: ' + error.message)
	}
}

// Maps UI sort value → { sort_by, sort_order } for backend
const tabSortMap = {
	newest:       { sort_by: 'discovered_at', sort_order: 'desc' },
	oldest:       { sort_by: 'discovered_at', sort_order: 'asc' },
	'views-desc': { sort_by: 'view_count',    sort_order: 'desc' },
	'views-asc':  { sort_by: 'view_count',    sort_order: 'asc' },
	'duration-desc': { sort_by: 'duration',   sort_order: 'desc' },
	'duration-asc':  { sort_by: 'duration',   sort_order: 'asc' },
	'title-asc':  { sort_by: 'title',         sort_order: 'asc' },
	'title-desc': { sort_by: 'title',         sort_order: 'desc' },
	'likes-desc': { sort_by: 'like_count',    sort_order: 'desc' },
}

const loadVideosForTab = async (tabId) => {
	customTabLoading.value[tabId] = true
	try {
		const tab = scraperTabs.value.find((t) => t.id === tabId)
		if (!tab) return

		const source = new URL(tab.baseUrl).host
		const sort = videoSort.value[tabId] || 'newest'
		const { sort_by, sort_order } = tabSortMap[sort] || tabSortMap.newest

		const params = new URLSearchParams({
			source,
			page:       String(videoPage.value[tabId] || 1),
			limit:      String(videosPerPage.value),
			sort_by,
			sort_order,
		})

		const search = (videoSearch.value[tabId] || '').trim()
		if (search) params.set('search', search)

		const duration = videoDurationFilter.value[tabId] || ''
		if (duration) params.set('duration_filter', duration)

		if (videoOnlyDownloaded.value[tabId]) params.set('downloaded_only', 'true')

		const author = (videoAuthorFilter.value[tabId] || '').trim()
		if (author) params.set('author', author)

		const minViews = videoMinViews.value[tabId] || 0
		if (minViews > 0) params.set('min_views', String(minViews))

		const response = await fetch(`http://localhost:8080/api/v1/scraper/videos?${params}`)
		const data = await response.json()

		if (data.success && data.data) {
			const result = data.data
			customTabVideos.value[tabId] = result.videos || []
			tabVideoTotal.value[tabId] = result.total || 0
			tabVideoTotalPages.value[tabId] = result.totalPages || 1
		}
	} catch (error) {
		console.error('Error loading videos for tab:', error)
	} finally {
		customTabLoading.value[tabId] = false
	}
}

// Called by filter dropdowns / checkboxes — resets to page 1 then reloads
const onTabFilterChange = (tabId) => {
	videoPage.value[tabId] = 1
	customTabSelectedVideos.value[tabId] = new Set()
	loadVideosForTab(tabId)
}

// Called by search/author inputs — debounced
const onTabSearchInput = (tabId) => {
	clearTimeout(tabSearchTimeouts[tabId])
	tabSearchTimeouts[tabId] = setTimeout(() => {
		videoPage.value[tabId] = 1
		loadVideosForTab(tabId)
	}, 400)
}

// Returns number of active (non-default) advanced filters for a tab
const activeTabFilterCount = (tabId) => {
	let n = 0
	if (videoDurationFilter.value[tabId]) n++
	if (videoOnlyDownloaded.value[tabId]) n++
	if (videoAuthorFilter.value[tabId]) n++
	if (videoMinViews.value[tabId] > 0) n++
	return n
}

// Video filtering, sorting and pagination

const changeVideoPage = (tabId, direction) => {
	const currentPage = videoPage.value[tabId] || 1
	const totalPages = tabVideoTotalPages.value[tabId] || 1
	const newPage = currentPage + direction

	if (newPage >= 1 && newPage <= totalPages) {
		videoPage.value[tabId] = newPage
		loadVideosForTab(tabId)
		window.scrollTo({ top: 0, behavior: 'smooth' })
	}
}

const changeVideosPerPage = (newSize, tabId = null) => {
	videosPerPage.value = parseInt(newSize) || 24
	localStorage.setItem('scraper-page-size', String(videosPerPage.value))
	if (tabId != null) {
		videoPage.value[tabId] = 1
		loadVideosForTab(tabId)
	} else {
		videoPage.value = {}
	}
}

const playScrapedVideo = (video, tab = null) => {
	modalVideo.value = video
	modalTab.value = tab
}

const onVideoDownloaded = (videoId) => {
	// Mark the video as downloaded in all local lists so the badge appears immediately
	const now = new Date().toISOString()
	const markInList = (list) => {
		if (!list) return
		const v = list.find((v) => v.id === videoId)
		if (v) v.downloaded_at = now
	}
	markInList(customTabVideos.value)
	markInList(pornHubVideos.value)
	markInList(luxureTVVideos.value)
	markInList(rule34VideoVideos.value)
	markInList(sexuriaItems.value)
}

const onVideoUnmarked = (videoId) => {
	// Clear downloaded_at from all local lists
	const clearInList = (list) => {
		if (!list) return
		const v = list.find((v) => v.id === videoId)
		if (v) v.downloaded_at = null
	}
	clearInList(customTabVideos.value)
	clearInList(pornHubVideos.value)
	clearInList(luxureTVVideos.value)
	clearInList(rule34VideoVideos.value)
	clearInList(sexuriaItems.value)
}

const formatDuration = (seconds) => {
	if (!seconds) return '0:00'
	const hours = Math.floor(seconds / 3600)
	const minutes = Math.floor((seconds % 3600) / 60)
	const secs = seconds % 60

	if (hours > 0) {
		return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
	}
	return `${minutes}:${secs.toString().padStart(2, '0')}`
}

const formatNumber = (num) => {
	if (num >= 1000000) {
		return (num / 1000000).toFixed(1) + 'M'
	}
	if (num >= 1000) {
		return (num / 1000).toFixed(1) + 'K'
	}
	return num.toString()
}

const getVideoResolution = (video) => {
	if (!video.metadata) return null

	try {
		const metadata = typeof video.metadata === 'string' ? JSON.parse(video.metadata) : video.metadata

		// Return the max resolution if available
		if (metadata.max_resolution) {
			return metadata.max_resolution
		}

		// Or the first resolution from the list
		if (metadata.resolutions && metadata.resolutions.length > 0) {
			return metadata.resolutions[0]
		}
	} catch (e) {
		console.error('Error parsing video metadata:', e)
	}

	return null
}

// Thumbnail preview animation
const getThumbnailFrames = (video) => {
	if (!video.metadata) return []

	try {
		const metadata = typeof video.metadata === 'string' ? JSON.parse(video.metadata) : video.metadata
		return metadata.thumbnail_frames || []
	} catch (e) {
		return []
	}
}

const getCurrentThumbnail = (video) => {
	const frames = getThumbnailFrames(video)
	if (frames.length === 0) {
		return video.thumbnail_url
	}

	const state = thumbnailPreviewState.value[video.id]
	if (!state) {
		return video.thumbnail_url
	}

	return frames[state.currentFrame] || video.thumbnail_url
}

const startThumbnailPreview = (video) => {
	const frames = getThumbnailFrames(video)
	if (frames.length <= 1) return

	// Initialize state
	thumbnailPreviewState.value[video.id] = {
		currentFrame: 0,
		interval: null,
	}

	// Cycle through frames every 300ms
	const interval = setInterval(() => {
		const state = thumbnailPreviewState.value[video.id]
		if (state) {
			state.currentFrame = (state.currentFrame + 1) % frames.length
		}
	}, 300)

	thumbnailPreviewState.value[video.id].interval = interval
}

const stopThumbnailPreview = (video) => {
	const state = thumbnailPreviewState.value[video.id]
	if (state && state.interval) {
		clearInterval(state.interval)
		delete thumbnailPreviewState.value[video.id]
	}
}

// eslint-disable-next-line no-unused-vars
const getJobStatusClass = (status) => {
	switch (status) {
		case 'completed':
			return 'text-success'
		case 'running':
			return 'text-info'
		case 'pending':
			return 'text-warning'
		case 'failed':
			return 'text-danger'
		case 'cancelled':
			return 'text-secondary'
		default:
			return ''
	}
}

// Lifecycle
onMounted(() => {
	checkAuthStatus()
	loadStats()
	loadThreads()
	loadScraperTabs()
	checkJDownloaderStatus() // Check JDownloader availability

	// Load scraped videos if on videos tab
	if (activeTab.value === 'videos') {
		loadScrapedVideos()
	}

	// Load PornHub data
	loadPornHubKeywords()
	loadPornHubStats()
	loadPornHubVideos()

	// Load LuxureTV data
	loadLuxureTVCategories()
	loadLuxureTVStats()
	loadLuxureTVVideos()

	// Load Sexuria data
	loadSexuriaCategories()
	loadSexuriaStats()
	loadSexuriaItems()
	loadSexuriaSearchOptions()

	// Load Rule34Video data
	loadRule34VideoStats()
	loadRule34VideoVideos()
})

// Cleanup all active polling timeouts when component unmounts
onBeforeUnmount(() => {
	// Clear all active timeouts to prevent memory leaks
	activeTimeouts.value.forEach((timeoutId) => {
		clearTimeout(timeoutId)
	})
	activeTimeouts.value = []

	// Clear all thumbnail preview intervals
	Object.values(thumbnailPreviewState.value).forEach((state) => {
		if (state.interval) {
			clearInterval(state.interval)
		}
	})
	thumbnailPreviewState.value = {}
})
</script>

<style src="@/styles/pages/scraper_page.css"></style>
