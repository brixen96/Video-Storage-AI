<template>
	<div class="scraper-page">
		<div class="container-fluid py-4">
			<!-- Page Header -->
			<div class="page-header mb-4">
				<div class="d-flex justify-content-between align-items-center">
					<div>
						<h1>
							<font-awesome-icon :icon="['fas', 'spider']" class="me-3" />
							Web Scraper
						</h1>
						<p class="text-muted mb-0">Scrape content from multiple sources</p>
					</div>
				</div>
			</div>

			<!-- Tabs for different scrapers -->
			<div class="scraper-tabs mb-4">
				<!-- Simpcity Tab (Always present) -->
				<button
					class="scraper-tab"
					:class="{ active: activeTab === 'simpcity' }"
					@click="activeTab = 'simpcity'"
				>
					<font-awesome-icon :icon="['fas', 'comments']" class="me-2" />
					Simpcity Forum
				</button>

				<!-- Dynamic Custom Tabs -->
				<button
					v-for="tab in scraperTabs"
					:key="tab.id"
					class="scraper-tab"
					:class="{ active: activeTab === `tab-${tab.id}` }"
					:style="{ borderColor: tab.color }"
					@click="selectTab(tab.id)"
				>
					<font-awesome-icon :icon="['fas', tab.icon]" class="me-2" />
					{{ tab.name }}
					<button
						class="btn-close-tab"
						@click.stop="confirmDeleteTab(tab)"
						title="Delete tab"
					>
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
			<div v-if="activeTab === 'simpcity'" class="tab-content">
				<!-- Action Buttons -->
				<div class="d-flex gap-2 mb-4">
					<button class="btn btn-outline-light" @click="showAuthModal = true">
						<font-awesome-icon :icon="['fas', 'key']" class="me-2" />
						{{ sessionCookieSet ? 'Auth: Active' : 'Set Auth' }}
					</button>
					<button class="btn btn-success" @click="showForumScrapeModal = true">
						<font-awesome-icon :icon="['fas', 'globe']" class="me-2" />
						Scrape Entire Forum
					</button>
					<button class="btn btn-warning" @click="autoLinkPerformers">
						<font-awesome-icon :icon="['fas', 'link']" class="me-2" />
						Auto-Link Performers
					</button>
					<button class="btn btn-info" @click="checkLinkStatuses">
						<font-awesome-icon :icon="['fas', 'check-circle']" class="me-2" />
						Check Link Status
					</button>
					<button class="btn btn-primary" @click="showScrapeModal = true">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
						Single Thread
					</button>
				</div>

			<!-- Statistics Cards -->
			<div class="row g-3 mb-4">
				<div class="col-md-3">
					<StatCard :value="stats.total_threads || 0" label="Threads Scraped" :icon="['fas', 'comments']" icon-class="threads" />
				</div>
				<div class="col-md-3">
					<StatCard :value="stats.total_posts || 0" label="Posts Collected" :icon="['fas', 'comment-dots']" icon-class="posts" />
				</div>
				<div class="col-md-3">
					<StatCard :value="stats.total_download_links || 0" label="Download Links" :icon="['fas', 'link']" icon-class="links" />
				</div>
				<div class="col-md-3">
					<StatCard :value="stats.active_links || 0" label="Active Links" :icon="['fas', 'check-circle']" icon-class="active" />
				</div>
			</div>

			<!-- Provider Breakdown -->
			<div v-if="stats.provider_breakdown" class="card mb-4">
				<div class="card-header">
					<h5 class="mb-0">
						<font-awesome-icon :icon="['fas', 'chart-pie']" class="me-2" />
						Provider Breakdown
					</h5>
				</div>
				<div class="card-body">
					<div class="provider-tags">
						<span v-for="(count, provider) in stats.provider_breakdown" :key="provider" class="provider-tag">
							<font-awesome-icon :icon="['fas', 'server']" class="me-1" />
							{{ provider }}: <strong>{{ count }}</strong>
						</span>
					</div>
				</div>
			</div>

			<!-- Search and Filters -->
			<div class="filters-section mb-4">
				<div class="row g-3">
					<div class="col-md-12">
						<SearchBox v-model="searchQuery" placeholder="Search threads by title or author..." @input="searchThreads" />
					</div>
				</div>
				<div class="row g-3 mt-2">
					<div class="col-md-3">
						<label class="filter-label">Sort By</label>
						<select v-model="sortBy" @change="handleSortChange" class="form-select">
							<option value="date_desc">Date (Newest First)</option>
							<option value="date_asc">Date (Oldest First)</option>
							<option value="title_asc">Title (A-Z)</option>
							<option value="title_desc">Title (Z-A)</option>
							<option value="views_desc">Views (High to Low)</option>
							<option value="views_asc">Views (Low to High)</option>
							<option value="replies_desc">Replies (Most First)</option>
							<option value="downloads_desc">Downloads (Most First)</option>
						</select>
					</div>
					<div class="col-md-3">
						<label class="filter-label">Provider</label>
						<select v-model="filterProvider" @change="handleFilterChange" class="form-select">
							<option value="">All Providers</option>
							<option v-for="(count, provider) in stats.provider_breakdown" :key="provider" :value="provider">
								{{ provider }} ({{ count }})
							</option>
						</select>
					</div>
					<div class="col-md-3">
						<label class="filter-label">Content</label>
						<select v-model="filterContent" @change="handleFilterChange" class="form-select">
							<option value="">All Threads</option>
							<option value="has_downloads">Has Downloads</option>
							<option value="no_downloads">No Downloads</option>
						</select>
					</div>
					<div class="col-md-3">
						<label class="filter-label">View Mode</label>
						<select v-model="viewMode" class="form-select">
							<option value="grid">Grid View</option>
							<option value="list">List View</option>
							<option value="compact">Compact View</option>
						</select>
					</div>
				</div>
			</div>

			<!-- Bulk Action Toolbar -->
			<div v-if="threads.length > 0" class="bulk-actions-toolbar mb-4">
				<div class="d-flex justify-content-between align-items-center">
					<div class="d-flex gap-2 align-items-center">
						<button class="btn btn-sm" :class="selectionMode ? 'btn-primary' : 'btn-outline-primary'" @click="toggleSelectionMode">
							<font-awesome-icon :icon="['fas', selectionMode ? 'times' : 'check-square']" class="me-2" />
							{{ selectionMode ? 'Cancel Selection' : 'Select Threads' }}
						</button>
						<div v-if="selectionMode" class="d-flex gap-2">
							<button class="btn btn-sm btn-outline-secondary" @click="allSelectedOnPage ? deselectAllThreads() : selectAllThreads()">
								<font-awesome-icon :icon="['fas', allSelectedOnPage ? 'square' : 'check-square']" class="me-2" />
								{{ allSelectedOnPage ? 'Deselect All' : 'Select All' }}
							</button>
							<span v-if="selectedCount > 0" class="badge bg-primary align-self-center">{{ selectedCount }} selected</span>
						</div>
					</div>
					<div v-if="selectionMode || threads.length > 0" class="d-flex gap-2">
						<button v-if="selectionMode && selectedCount > 0" class="btn btn-sm btn-info" @click="bulkVerifyLinks" :disabled="bulkOperationInProgress">
							<font-awesome-icon :icon="['fas', bulkOperationInProgress ? 'spinner' : 'check-double']" :spin="bulkOperationInProgress" class="me-2" />
							Verify Selected
						</button>
						<button v-if="selectionMode && selectedCount > 0" class="btn btn-sm btn-warning" @click="bulkSendToJDownloader" :disabled="bulkOperationInProgress">
							<font-awesome-icon :icon="['fas', bulkOperationInProgress ? 'spinner' : 'cloud-download-alt']" :spin="bulkOperationInProgress" class="me-2" />
							Send to JD
						</button>
						<button v-if="selectionMode && selectedCount > 0" class="btn btn-sm btn-danger" @click="deleteSelectedThreads">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
							Delete Selected
						</button>
						<button class="btn btn-sm btn-outline-danger" @click="deleteAllThreads">
							<font-awesome-icon :icon="['fas', 'trash-alt']" class="me-2" />
							Delete All
						</button>
					</div>
				</div>
			</div>

			<!-- Loading State -->
			<LoadingState v-if="loading" show-text loading-text="Loading scraped threads..." />

			<!-- Empty State -->
			<EmptyState v-else-if="threads.length === 0 && !loading" :icon="['fas', 'inbox']" icon-size="4x" title="No Threads Found" message="Start scraping to collect thread data">
				<template #actions>
					<button class="btn btn-primary mt-3" @click="showScrapeModal = true">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
						Start Scraping
					</button>
				</template>
			</EmptyState>

			<!-- Threads Grid View -->
			<div v-else-if="viewMode === 'grid'" class="threads-grid">
				<div v-for="thread in threads" :key="thread.id" class="thread-card" :class="{ 'selected': isThreadSelected(thread.id) }" @click="openThread(thread)">
					<div v-if="selectionMode" class="thread-checkbox" @click="toggleThreadSelection(thread.id, $event)">
						<input type="checkbox" :checked="isThreadSelected(thread.id)" @click.stop>
					</div>
					<div v-if="thread.metadata?.thumbnail_url || thread.metadata?.thumbnail_urls?.length" class="thread-thumbnail">
						<img
							:src="thread.metadata.thumbnail_url"
							:data-fallback-urls="JSON.stringify(thread.metadata.thumbnail_urls || [])"
							alt="Thread thumbnail"
							@error="handleThumbnailError" />
					</div>
					<div v-else class="thread-thumbnail placeholder">
						<font-awesome-icon :icon="['fas', 'image']" size="3x" />
					</div>
					<div class="thread-content">
						<h5 class="thread-title">{{ thread.title }}</h5>
						<div class="thread-meta">
							<span class="meta-item">
								<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
								{{ thread.author }}
							</span>
							<span class="meta-item">
								<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
								{{ thread.category || 'General' }}
							</span>
						</div>
						<div class="thread-stats">
							<span class="stat-badge">
								<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
								{{ thread.view_count }}
							</span>
							<span class="stat-badge">
								<font-awesome-icon :icon="['fas', 'comment']" class="me-1" />
								{{ thread.post_count }}
							</span>
							<span class="stat-badge">
								<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
								{{ thread.download_count }}
							</span>
						</div>
						<div v-if="thread.metadata?.tags?.length" class="thread-tags">
							<span v-for="tag in thread.metadata.tags.slice(0, 3)" :key="tag" class="thread-tag">
								{{ tag }}
							</span>
						</div>
					</div>
				</div>
			</div>

			<!-- Threads List View -->
			<div v-else-if="viewMode === 'list'" class="threads-list">
				<div v-for="thread in threads" :key="thread.id" class="thread-list-item" @click="openThread(thread)">
					<div class="thread-list-content">
						<h5 class="thread-title">{{ thread.title }}</h5>
						<div class="thread-info">
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
								{{ thread.author }}
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
								{{ thread.category }}
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
								{{ thread.view_count }} views
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'comment']" class="me-1" />
								{{ thread.post_count }} posts
							</span>
							<span class="info-item">
								<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
								{{ thread.download_count }} links
							</span>
						</div>
					</div>
					<div class="thread-list-actions">
						<button class="btn btn-sm btn-outline-primary" @click.stop="rescrapeThread(thread)">
							<font-awesome-icon :icon="['fas', 'sync']" />
						</button>
						<button class="btn btn-sm btn-outline-secondary" @click.stop="openThreadURL(thread.url)">
							<font-awesome-icon :icon="['fas', 'external-link-alt']" />
						</button>
					</div>
				</div>
			</div>

			<!-- Pagination -->
			<div v-if="totalPages > 1" class="pagination-container">
				<nav>
					<ul class="pagination justify-content-center">
						<li class="page-item" :class="{ disabled: currentPage === 1 }">
							<button class="page-link" @click="goToPage(currentPage - 1)">Previous</button>
						</li>
						<li v-for="page in visiblePages" :key="page" class="page-item" :class="{ active: page === currentPage }">
							<button class="page-link" @click="goToPage(page)">{{ page }}</button>
						</li>
						<li class="page-item" :class="{ disabled: currentPage === totalPages }">
							<button class="page-link" @click="goToPage(currentPage + 1)">Next</button>
						</li>
					</ul>
				</nav>
			</div>
		</div>

		<!-- Authentication Modal -->
		<div v-if="showAuthModal" class="modal-overlay" @click="showAuthModal = false">
			<div class="modal-dialog" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'key']" class="me-2" />
							Set Authentication Cookie
						</h5>
						<button type="button" class="btn-close" @click="showAuthModal = false"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Session Cookie</label>
							<textarea
								v-model="authCookie"
								class="form-control"
								rows="4"
								placeholder="Paste your session cookie here..."
							></textarea>
							<small class="form-text text-muted">
								Get your cookie from browser Dev Tools (F12 → Application → Cookies → simpcity.is)
							</small>
						</div>
						<div class="alert alert-info">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
							<strong>Required cookies:</strong> ogaddgmetaprof_csrf, ogaddgmetaprof_session, ogaddgmetaprof_user,
							cucksed, cucksid
						</div>
						<div v-if="sessionCookieSet" class="alert alert-success">
							<font-awesome-icon :icon="['fas', 'check-circle']" class="me-2" />
							Authentication is currently <strong>active</strong>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="showAuthModal = false">Cancel</button>
						<button type="button" class="btn btn-primary" @click="setAuthCookie" :disabled="!authCookie">
							<font-awesome-icon :icon="['fas', 'save']" class="me-2" />
							Save Cookie
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Scrape Modal -->
		<div v-if="showScrapeModal" class="modal-overlay" @click="showScrapeModal = false">
			<div class="modal-dialog" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'spider']" class="me-2" />
							New Scrape Job
						</h5>
						<button type="button" class="btn-close" @click="showScrapeModal = false"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Thread URL</label>
							<input
								v-model="scrapeURL"
								type="text"
								class="form-control"
								placeholder="https://simpcity.is/threads/thread-name.123456/"
							/>
							<small class="form-text text-muted">Enter the full URL of the thread you want to scrape</small>
						</div>
						<div class="alert alert-info">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
							Scraping will run in the background. Check the Tasks page to monitor progress.
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="showScrapeModal = false">Cancel</button>
						<button type="button" class="btn btn-primary" @click="startScrape" :disabled="!scrapeURL">
							<font-awesome-icon :icon="['fas', 'play']" class="me-2" />
							Start Scraping
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Thread Detail Modal -->
		<div v-if="selectedThread" class="modal-overlay" @click="selectedThread = null">
			<div class="modal-dialog modal-lg" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">{{ selectedThread.title }}</h5>
						<button type="button" class="btn-close" @click="selectedThread = null"></button>
					</div>
					<div class="modal-body">
						<div class="thread-detail">
							<div class="detail-row">
								<strong>Author:</strong> {{ selectedThread.author }}
							</div>
							<div class="detail-row">
								<strong>Category:</strong> {{ selectedThread.category }}
							</div>
							<div class="detail-row">
								<strong>URL:</strong>
								<a :href="selectedThread.url" target="_blank">{{ selectedThread.url }}</a>
							</div>
							<div class="detail-row">
								<strong>Statistics:</strong>
								<div class="stats-inline">
									<span>{{ selectedThread.view_count }} views</span>
									<span>{{ selectedThread.post_count }} posts</span>
									<span>{{ selectedThread.download_count }} download links</span>
								</div>
							</div>
							<div v-if="selectedThread.metadata?.performer_names?.length" class="detail-row">
								<strong>Detected Performers:</strong>
								<div class="tags-container">
									<span v-for="performer in selectedThread.metadata.performer_names" :key="performer" class="tag-badge">
										{{ performer }}
									</span>
								</div>
							</div>
							<div v-if="selectedThread.metadata?.studio_names?.length" class="detail-row">
								<strong>Detected Studios:</strong>
								<div class="tags-container">
									<span v-for="studio in selectedThread.metadata.studio_names" :key="studio" class="tag-badge">
										{{ studio }}
									</span>
								</div>
							</div>
							<div v-if="selectedThread.metadata?.tags?.length" class="detail-row">
								<strong>Tags:</strong>
								<div class="tags-container">
									<span v-for="tag in selectedThread.metadata.tags" :key="tag" class="tag-badge">
										{{ tag }}
									</span>
								</div>
							</div>
							<div class="detail-row">
								<strong>Last Scraped:</strong> {{ formatDate(selectedThread.last_scraped_at) }}
							</div>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="openThreadURL(selectedThread.url)">
							<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-2" />
							Open in Browser
						</button>
						<button type="button" class="btn btn-primary" @click="rescrapeThread(selectedThread)">
							<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
							Rescrape
						</button>
					</div>
				</div>
			</div>
		</div>

		<!-- Forum Scrape Modal -->
		<div v-if="showForumScrapeModal" class="modal-overlay" @click="showForumScrapeModal = false">
			<div class="modal-dialog" @click.stop>
				<div class="modal-content">
					<div class="modal-header">
						<h5 class="modal-title">
							<font-awesome-icon :icon="['fas', 'globe']" class="me-2" />
							Scrape Entire Forum
						</h5>
						<button type="button" class="btn-close" @click="showForumScrapeModal = false"></button>
					</div>
					<div class="modal-body">
						<div class="mb-3">
							<label class="form-label">Forum Category URL</label>
							<input
								v-model="forumURL"
								type="text"
								class="form-control"
								placeholder="https://simpcity.is/forums/xxx-porn.50/"
							/>
							<small class="form-text text-muted">
								Enter the forum category URL (e.g., https://simpcity.is/forums/xxx-porn.50/)
							</small>
						</div>
						<div class="alert alert-warning">
							<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-2" />
							<strong>Warning:</strong> This will scrape ALL threads in the forum category. This may take several hours and consume significant bandwidth. The scraper will run in the background.
						</div>
						<div class="alert alert-info">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
							<ul class="mb-0">
								<li>The scraper will automatically handle pagination</li>
								<li>Each thread will be fully scraped with all posts and download links</li>
								<li>Performer names will be extracted from thread titles</li>
								<li>You can monitor progress in the server logs</li>
							</ul>
						</div>
					</div>
					<div class="modal-footer">
						<button type="button" class="btn btn-secondary" @click="showForumScrapeModal = false">Cancel</button>
						<button type="button" class="btn btn-success" @click="startForumScrape" :disabled="!forumURL">
							<font-awesome-icon :icon="['fas', 'play']" class="me-2" />
							Start Forum Scraping
						</button>
					</div>
				</div>
			</div>
		</div>
		<!-- End Simpcity Tab Content (modals are shared between tabs) -->

		<!-- Videos Tab Content -->
			<div v-if="activeTab === 'videos'" class="tab-content">
				<div class="card">
					<div class="card-header">
						<h5 class="mb-0">
							<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
							Video Site Scraper
						</h5>
					</div>
					<div class="card-body">
						<p class="text-muted mb-4">
							Configure and scrape videos from supported video hosting sites. Enter a URL or configure site-specific settings below.
						</p>

						<!-- Video Scraper Form -->
						<div class="row g-3">
							<div class="col-md-8">
								<label class="form-label">Video URL or Page URL</label>
								<input
									v-model="videoUrl"
									type="text"
									class="form-control"
									placeholder="https://example.com/video/12345 or https://example.com/category/name"
								/>
								<small class="text-muted">Enter a single video URL, category page, or channel/user page</small>
							</div>
							<div class="col-md-4">
								<label class="form-label">Site Type</label>
								<select v-model="videoSiteType" class="form-select">
									<option value="generic">Auto-Detect</option>
									<option value="youtube">YouTube</option>
									<option value="vimeo">Vimeo</option>
									<option value="custom">Custom Site</option>
								</select>
							</div>
						</div>

						<div class="row g-3 mt-3">
							<div class="col-md-12">
								<label class="form-label">Scrape Options</label>
								<div class="form-check">
									<input v-model="videoScrapeOptions.extractMetadata" class="form-check-input" type="checkbox" id="extractMetadata" />
									<label class="form-check-label" for="extractMetadata">
										Extract video metadata
									</label>
								</div>
								<div class="form-check">
									<input v-model="videoScrapeOptions.scrapeThumbnails" class="form-check-input" type="checkbox" id="scrapeThumbnails" />
									<label class="form-check-label" for="scrapeThumbnails">
										Download thumbnails
									</label>
								</div>
								<small class="text-muted d-block mt-2">The scraper will automatically crawl all pages until no more videos are found.</small>
							</div>
						</div>

						<div class="mt-4 d-flex gap-2">
							<button class="btn btn-primary" @click="scrapeVideo" :disabled="!videoUrl || videoScraping">
								<font-awesome-icon :icon="['fas', videoScraping ? 'spinner' : 'play']" :spin="videoScraping" class="me-2" />
								{{ videoScraping ? 'Scraping...' : 'Start Video Scrape' }}
							</button>
							<button class="btn btn-info" @click="testScraper" :disabled="!videoUrl">
								<font-awesome-icon :icon="['fas', 'vial']" class="me-2" />
								Test Scraper
							</button>
							<button v-if="scrapedVideos.length > 0" class="btn btn-danger" @click="clearAllVideos">
								<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
								Clear All Videos
							</button>
						</div>
					</div>
				</div>

				<!-- Scraped Videos List -->
				<div v-if="scrapedVideos.length > 0" class="card mt-4">
					<div class="card-header">
						<h5 class="mb-0">
							<font-awesome-icon :icon="['fas', 'list']" class="me-2" />
							Scraped Videos ({{ scrapedVideos.length }})
						</h5>
					</div>
					<div class="card-body">
						<div class="table-responsive">
							<table class="table table-hover">
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
											<a :href="video.url" target="_blank" class="text-truncate d-inline-block" style="max-width: 200px;">
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
				</div>
			</div>
			<!-- End Videos Tab Content -->

			<!-- Dynamic Custom Tab Content -->
			<div v-for="tab in scraperTabs" :key="`content-${tab.id}`">
				<div v-if="activeTab === `tab-${tab.id}`" class="tab-content">
					<div class="card">
						<div class="card-header d-flex justify-content-between align-items-center">
							<h5 class="mb-0">
								<font-awesome-icon :icon="['fas', tab.icon]" class="me-2" />
								{{ tab.name }} Scraper
							</h5>
							<button class="btn btn-sm btn-outline-primary" @click="editScraperTab(tab)">
								<font-awesome-icon :icon="['fas', 'edit']" class="me-2" />
								Edit Tab
							</button>
						</div>
						<div class="card-body">
							<!-- Scraper URL Input -->
							<div class="mb-3">
								<label class="form-label">URL to Scrape</label>
								<input
									v-model="customTabUrls[tab.id]"
									type="text"
									class="form-control"
									:placeholder="`Enter ${tab.name} URL...`"
								/>
								<small class="text-muted">Base URL: {{ tab.baseUrl }}</small>
							</div>

							<!-- Action Buttons -->
							<div class="d-flex gap-2 mb-4">
								<button
									class="btn btn-primary"
									@click="startCustomTabScrape(tab)"
									:disabled="!customTabUrls[tab.id] || customTabJobs[tab.id]?.status === 'running'"
								>
									<font-awesome-icon :icon="['fas', customTabJobs[tab.id]?.status === 'running' ? 'spinner' : 'play']" :spin="customTabJobs[tab.id]?.status === 'running'" class="me-2" />
									{{ customTabJobs[tab.id]?.status === 'running' ? 'Scraping...' : 'Start Scrape' }}
								</button>
								<button
									v-if="customTabJobs[tab.id] && (customTabJobs[tab.id].status === 'running' || customTabJobs[tab.id].status === 'pending')"
									class="btn btn-danger"
									@click="cancelJob(customTabJobs[tab.id])"
								>
									<font-awesome-icon :icon="['fas', 'stop']" class="me-2" />
									Cancel Job
								</button>
								<button
									v-if="customTabJobs[tab.id]"
									class="btn btn-info"
									@click="loadTabJobProgress(tab.id)"
								>
									<font-awesome-icon :icon="['fas', 'sync']" class="me-2" />
									Refresh Status
								</button>
							</div>

							<!-- Job Progress -->
							<div v-if="customTabJobs[tab.id]" class="card bg-dark mb-3">
								<div class="card-body">
									<h6>Job Status: <span :class="getJobStatusClass(customTabJobs[tab.id].status)">{{ customTabJobs[tab.id].status }}</span></h6>
									<div class="row g-2 mt-2">
										<div class="col-md-3">
											<small class="text-muted">Pages Completed:</small>
											<div><strong>{{ customTabJobs[tab.id].pagesCompleted }} / {{ customTabJobs[tab.id].totalPages || '?' }}</strong></div>
										</div>
										<div class="col-md-3">
											<small class="text-muted">Videos Found:</small>
											<div><strong>{{ customTabJobs[tab.id].videosFound }}</strong></div>
										</div>
										<div class="col-md-3">
											<small class="text-muted">Videos Saved:</small>
											<div><strong>{{ customTabJobs[tab.id].videosSaved }}</strong></div>
										</div>
										<div class="col-md-3">
											<small class="text-muted">Errors:</small>
											<div><strong>{{ customTabJobs[tab.id].errorsCount }}</strong></div>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>

					<!-- Scraped Videos List -->
					<div v-if="customTabVideos[tab.id] && customTabVideos[tab.id].length > 0" class="card mt-3">
						<div class="card-header">
							<div class="d-flex justify-content-between align-items-center flex-wrap gap-3 mb-3">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									Scraped Videos ({{ filteredTabVideos(tab.id).length }} / {{ customTabVideos[tab.id].length }})
								</h6>
								<button class="btn btn-sm btn-outline-secondary" @click="toggleVideoFilters(tab.id)">
									<font-awesome-icon :icon="['fas', showVideoFilters[tab.id] ? 'chevron-up' : 'filter']" class="me-2" />
									{{ showVideoFilters[tab.id] ? 'Hide Filters' : 'Show Filters' }}
								</button>
							</div>

							<!-- Advanced Filters (Collapsible) -->
							<div v-if="showVideoFilters[tab.id]" class="video-filters">
								<div class="row g-2">
									<!-- Search -->
									<div class="col-md-4">
										<label class="form-label small">Search</label>
										<input
											v-model="videoSearch[tab.id]"
											type="text"
											class="form-control form-control-sm"
											placeholder="Title, uploader, author..."
										/>
									</div>

									<!-- Sort -->
									<div class="col-md-3">
										<label class="form-label small">Sort By</label>
										<select v-model="videoSort[tab.id]" class="form-select form-select-sm">
											<option value="newest">Newest First</option>
											<option value="oldest">Oldest First</option>
											<option value="title-asc">Title (A-Z)</option>
											<option value="title-desc">Title (Z-A)</option>
											<option value="duration-desc">Longest Duration</option>
											<option value="duration-asc">Shortest Duration</option>
											<option value="views-desc">Most Views</option>
											<option value="views-asc">Least Views</option>
											<option value="likes-desc">Most Likes</option>
										</select>
									</div>

									<!-- Status Filter -->
									<div class="col-md-2">
										<label class="form-label small">Status</label>
										<select v-model="videoStatusFilter[tab.id]" class="form-select form-select-sm">
											<option value="">All</option>
											<option value="discovered">Discovered</option>
											<option value="scraped">Scraped</option>
											<option value="downloaded">Downloaded</option>
											<option value="failed">Failed</option>
										</select>
									</div>

									<!-- Resolution Filter -->
									<div class="col-md-2">
										<label class="form-label small">Resolution</label>
										<select v-model="videoResolutionFilter[tab.id]" class="form-select form-select-sm">
											<option value="">All</option>
											<option value="4K">4K / 2160p</option>
											<option value="1440p">1440p</option>
											<option value="1080p">1080p</option>
											<option value="720p">720p</option>
											<option value="480p">480p</option>
											<option value="360p">360p</option>
											<option value="240p">240p</option>
										</select>
									</div>

									<!-- Duration Filter -->
									<div class="col-md-3">
										<label class="form-label small">Duration</label>
										<select v-model="videoDurationFilter[tab.id]" class="form-select form-select-sm">
											<option value="">All</option>
											<option value="0-60">Under 1 min</option>
											<option value="60-300">1-5 min</option>
											<option value="300-600">5-10 min</option>
											<option value="600-1800">10-30 min</option>
											<option value="1800-3600">30-60 min</option>
											<option value="3600-">Over 1 hour</option>
										</select>
									</div>

									<!-- Has Thumbnail -->
									<div class="col-md-2">
										<label class="form-label small">Thumbnail</label>
										<select v-model="videoThumbnailFilter[tab.id]" class="form-select form-select-sm">
											<option value="">All</option>
											<option value="yes">Has Thumbnail</option>
											<option value="no">No Thumbnail</option>
										</select>
									</div>

									<!-- Reset Filters -->
									<div class="col-md-2 d-flex align-items-end">
										<button class="btn btn-sm btn-outline-warning w-100" @click="resetVideoFilters(tab.id)">
											<font-awesome-icon :icon="['fas', 'redo']" class="me-1" />
											Reset
										</button>
									</div>
								</div>
							</div>
						</div>
						<div class="card-body p-0">
							<div class="video-grid">
								<div
									v-for="video in paginatedTabVideos(tab.id)"
									:key="video.id"
									class="video-card"
								>
									<!-- Thumbnail -->
									<div class="video-thumbnail">
										<img
											v-if="video.thumbnail_url"
											:src="video.thumbnail_url"
											:alt="video.title"
											@error="$event.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'"
										/>
										<div v-else class="no-thumbnail">
											<font-awesome-icon :icon="['fas', 'video']" size="3x" />
										</div>
										<span v-if="video.duration" class="duration-badge">
											{{ formatDuration(video.duration) }}
										</span>
										<span v-if="getVideoResolution(video)" class="resolution-badge">
											{{ getVideoResolution(video) }}
										</span>
									</div>

									<!-- Video Info -->
									<div class="video-info">
										<h6 class="video-title" :title="video.title">{{ video.title }}</h6>
										<div class="video-meta">
											<span v-if="video.uploader || video.author" class="text-muted small d-block mb-1">
												<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
												{{ video.uploader || video.author }}
											</span>
											<div class="d-flex flex-wrap gap-2 mt-1">
												<span v-if="video.view_count > 0" class="text-muted small">
													<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
													{{ formatNumber(video.view_count) }}
												</span>
												<span v-if="video.like_count > 0" class="text-muted small">
													<font-awesome-icon :icon="['fas', 'heart']" class="me-1" />
													{{ formatNumber(video.like_count) }}
												</span>
												<span v-if="video.file_size > 0" class="text-muted small">
													<font-awesome-icon :icon="['fas', 'hdd']" class="me-1" />
													{{ formatFileSize(video.file_size) }}
												</span>
											</div>
										</div>

										<!-- Actions -->
										<div class="video-actions mt-2">
											<button @click="playScrapedVideo(video)" class="btn btn-sm btn-primary">
												<font-awesome-icon :icon="['fas', 'play']" class="me-1" />
												Play
											</button>
											<a :href="video.url" target="_blank" class="btn btn-sm btn-outline-primary">
												<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-1" />
												Open
											</a>
											<span class="badge" :class="getVideoStatusBadge(video.status)">
												{{ video.status }}
											</span>
										</div>
									</div>
								</div>
							</div>

							<!-- Pagination -->
							<div v-if="filteredTabVideos(tab.id).length > videosPerPage" class="pagination-controls p-3 border-top">
								<div class="d-flex justify-content-center align-items-center flex-column gap-3">
									<div class="btn-group">
										<button
											class="btn btn-sm btn-outline-secondary"
											:disabled="(videoPage[tab.id] || 1) === 1"
											@click="changeVideoPage(tab.id, -1)"
										>
											<font-awesome-icon :icon="['fas', 'chevron-left']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary disabled">
											Page {{ videoPage[tab.id] || 1 }} / {{ Math.ceil(filteredTabVideos(tab.id).length / videosPerPage) }}
										</button>
										<button
											class="btn btn-sm btn-outline-secondary"
											:disabled="(videoPage[tab.id] || 1) >= Math.ceil(filteredTabVideos(tab.id).length / videosPerPage)"
											@click="changeVideoPage(tab.id, 1)"
										>
											<font-awesome-icon :icon="['fas', 'chevron-right']" />
										</button>
									</div>
									<span class="text-muted small">
										Showing {{ ((videoPage[tab.id] || 1) - 1) * videosPerPage + 1 }} -
										{{ Math.min((videoPage[tab.id] || 1) * videosPerPage, filteredTabVideos(tab.id).length) }}
										of {{ filteredTabVideos(tab.id).length }} videos
									</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
			<!-- End Dynamic Custom Tab Content -->

			<!-- Add/Edit Tab Modal -->
			<div v-if="showAddTabModal || showEditTabModal" class="modal show d-block" tabindex="-1" style="background-color: rgba(0,0,0,0.5);">
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
							<div class="mb-3">
								<label class="form-label">Tab Color</label>
								<input v-model="newTab.color" type="color" class="form-control" />
							</div>
							<div class="mb-3">
								<label class="form-label">Scrape Options</label>
								<div class="form-check">
									<input v-model="newTab.scrapeOptions.extractMetadata" class="form-check-input" type="checkbox" id="newTabExtractMetadata" />
									<label class="form-check-label" for="newTabExtractMetadata">
										Extract video metadata
									</label>
								</div>
								<div class="form-check">
									<input v-model="newTab.scrapeOptions.scrapeThumbnails" class="form-check-input" type="checkbox" id="newTabScrapeThumbnails" />
									<label class="form-check-label" for="newTabScrapeThumbnails">
										Download thumbnails
									</label>
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
		<DeleteConfirmationModal
			:visible="showDeleteConfirmModal"
			title="Confirm Deletion"
			message="Are you sure you want to delete"
			:itemName="`${selectedCount} selected thread(s)`"
			warningMessage="This action cannot be undone. All posts and download links associated with these threads will also be deleted."
			:confirmText="`Delete ${selectedCount} Thread(s)`"
			:icon="['fas', 'trash']"
			@confirm="confirmDeleteSelected"
			@cancel="showDeleteConfirmModal = false"
		/>

		<!-- Delete All Confirmation Modal -->
		<DeleteConfirmationModal
			:visible="showDeleteAllConfirmModal"
			title="Confirm Delete All"
			message="Are you sure you want to delete"
			:itemName="`ALL ${totalThreads} scraped threads`"
			warningMessage="DANGER: This will permanently delete ALL scraped threads, posts, and download links from the database. This action cannot be undone!"
			confirmText="Yes, Delete Everything"
			:icon="['fas', 'trash-alt']"
			@confirm="confirmDeleteAll"
			@cancel="showDeleteAllConfirmModal = false"
		/>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted, computed, getCurrentInstance } from 'vue'
import { useRouter } from 'vue-router'
import { DeleteConfirmationModal, LoadingState, EmptyState, StatCard, SearchBox } from '@/components/shared'
import { useFormatters } from '@/composables/useFormatters'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const router = useRouter()
const { formatDateTime } = useFormatters()

// State
const loading = ref(false)
const stats = ref({})
const threads = ref([])
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
const authCookie = ref('')
const sessionCookieSet = ref(false)
const selectedThread = ref(null)
const selectedThreadIds = ref(new Set())
const selectionMode = ref(false)

// Tab state
const activeTab = ref('simpcity')

// Video scraper state
const videoUrl = ref('')
const videoSiteType = ref('generic')
const videoScraping = ref(false)
const scrapedVideos = ref([])
const videoScrapeOptions = ref({
	downloadVideo: false,
	extractMetadata: true,
	scrapeThumbnails: true,
	maxPages: 0 // 0 = unlimited, scrapes all pages automatically
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
	scrapeOptions: {
		extractMetadata: true,
		scrapeThumbnails: true,
		downloadVideo: false
	}
})
const customTabUrls = ref({}) // { tabId: url }
const customTabJobs = ref({}) // { tabId: job }
const customTabVideos = ref({}) // { tabId: [videos] }
const videoSearch = ref({}) // { tabId: searchQuery }
const videoSort = ref({}) // { tabId: sortType }
const videoPage = ref({}) // { tabId: pageNumber }
const videoStatusFilter = ref({}) // { tabId: status }
const videoResolutionFilter = ref({}) // { tabId: resolution }
const videoDurationFilter = ref({}) // { tabId: durationRange }
const videoThumbnailFilter = ref({}) // { tabId: 'yes'|'no'|'' }
const showVideoFilters = ref({}) // { tabId: boolean }
const videosPerPage = 24

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

const setAuthCookie = async () => {
	if (!authCookie.value) {
		toast.warning('Please enter a cookie')
		return
	}

	try {
		const response = await fetch('http://localhost:8080/api/v1/scraper/session', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				cookie: authCookie.value
			})
		})

		const data = await response.json()

		if (data.success) {
			toast.success('Authentication cookie set successfully!')
			sessionCookieSet.value = true
			showAuthModal.value = false
			authCookie.value = ''
		} else {
			toast.error(data.message || 'Failed to set cookie')
		}
	} catch (error) {
		console.error('Error setting cookie:', error)
		toast.error('Failed to set authentication cookie')
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
		let url = searchQuery.value
			? `http://localhost:8080/api/v1/scraper/threads/search?q=${encodeURIComponent(searchQuery.value)}`
			: `http://localhost:8080/api/v1/scraper/threads?`

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
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				url: scrapeURL.value
			})
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
			method: 'POST'
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
	window.open(url, '_blank')
}

// formatDate replaced with formatDateTime from useFormatters composable
const formatDate = formatDateTime

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
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				url: forumURL.value
			})
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
			method: 'POST'
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
			method: 'POST'
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
	threads.value.forEach(thread => {
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
	return threads.value.every(thread => selectedThreadIds.value.has(thread.id))
})

// Deletion methods
// Bulk Operations
const bulkOperationInProgress = ref(false)

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
					method: 'POST'
				})
				const data = await response.json()

				if (data.success) {
					completed++
					toast.success(`Verified thread ${completed}/${threadIds.length}`, {
						duration: 2000
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
						auto_start: false
					})
				})
				const data = await response.json()

				if (data.success) {
					completed++
					totalLinksSent += data.data.links_sent || 0
					toast.success(`Sent thread ${completed}/${threadIds.length} (${data.data.links_sent} links)`, {
						duration: 2000
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
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				thread_ids: threadIds
			})
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
			method: 'DELETE'
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
				options: videoScrapeOptions.value
			})
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
	// Navigate to video details or open in new tab
	window.open(video.url, '_blank')
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
			method: 'DELETE'
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
			data.data.forEach(tab => {
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
	videoPage.value[tabId] = 1
}

const confirmDeleteTab = async (tab) => {
	if (!confirm(`Are you sure you want to delete the "${tab.name}" tab? This cannot be undone.`)) {
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/tabs/${tab.id}`, {
			method: 'DELETE'
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

const editScraperTab = (tab) => {
	editingTab.value = tab
	newTab.value = {
		name: tab.name,
		baseUrl: tab.baseUrl,
		siteType: tab.siteType,
		icon: tab.icon,
		color: tab.color,
		scrapeOptions: { ...tab.scrapeOptions }
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
		scrapeOptions: {
			extractMetadata: true,
			scrapeThumbnails: true,
			downloadVideo: false
		}
	}
}

const saveTabConfig = async () => {
	if (!newTab.value.name || !newTab.value.baseUrl) {
		toast.error('Please fill in all required fields')
		return
	}

	try {
		const url = showEditTabModal.value
			? `http://localhost:8080/api/v1/scraper/tabs/${editingTab.value.id}`
			: 'http://localhost:8080/api/v1/scraper/tabs'

		const method = showEditTabModal.value ? 'PUT' : 'POST'

		const response = await fetch(url, {
			method,
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(newTab.value)
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
				url: url
			})
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
				setTimeout(() => loadTabJobProgress(tabId), 3000)
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
					setTimeout(checkProgress, 3000)
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

const cancelJob = async (job) => {
	if (!confirm('Are you sure you want to cancel this scraping job?')) {
		return
	}

	try {
		const response = await fetch(`http://localhost:8080/api/v1/scraper/jobs/${job.id}/cancel`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' }
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

const loadVideosForTab = async (tabId) => {
	try {
		const tab = scraperTabs.value.find(t => t.id === tabId)
		if (!tab) return

		const response = await fetch('http://localhost:8080/api/v1/scraper/videos')
		const data = await response.json()

		if (data.success && data.data) {
			// Filter videos by source matching the tab's base URL
			const tabSource = new URL(tab.baseUrl).host
			console.log(`Loading videos for tab ${tabId}, baseUrl: ${tab.baseUrl}, expected source: ${tabSource}`)
			console.log(`Found ${data.data.length} total videos in database`)

			// Filter by source
			const filtered = data.data.filter(v => {
				console.log(`Video source: ${v.source}, matches: ${v.source === tabSource}`)
				return v.source === tabSource
			})

			customTabVideos.value[tabId] = filtered
			console.log(`Showing ${filtered.length} videos for this tab`)
			if (filtered.length > 0) {
				console.log('Sample video data:', JSON.stringify(filtered[0], null, 2))
			}
		}
	} catch (error) {
		console.error('Error loading videos for tab:', error)
	}
}

// Video filtering, sorting and pagination
const filteredTabVideos = (tabId) => {
	let videos = customTabVideos.value[tabId] || []
	const searchTerm = (videoSearch.value[tabId] || '').toLowerCase()
	const sortType = videoSort.value[tabId] || 'newest'
	const statusFilter = videoStatusFilter.value[tabId] || ''
	const resolutionFilter = videoResolutionFilter.value[tabId] || ''
	const durationFilter = videoDurationFilter.value[tabId] || ''
	const thumbnailFilter = videoThumbnailFilter.value[tabId] || ''

	// Filter by search
	if (searchTerm) {
		videos = videos.filter(v =>
			v.title.toLowerCase().includes(searchTerm) ||
			(v.uploader || '').toLowerCase().includes(searchTerm) ||
			(v.author || '').toLowerCase().includes(searchTerm)
		)
	}

	// Filter by status
	if (statusFilter) {
		videos = videos.filter(v => v.status === statusFilter)
	}

	// Filter by resolution
	if (resolutionFilter) {
		videos = videos.filter(v => {
			const resolution = getVideoResolution(v)
			if (!resolution) return false
			// Match 4K to both "4K" and "2160p"
			if (resolutionFilter === '4K') {
				return resolution === '4K' || resolution === '2160p'
			}
			return resolution === resolutionFilter
		})
	}

	// Filter by duration
	if (durationFilter) {
		const [min, max] = durationFilter.split('-').map(Number)
		videos = videos.filter(v => {
			const duration = v.duration || 0
			if (max) {
				return duration >= min && duration < max
			} else {
				// "3600-" means 3600 and above
				return duration >= min
			}
		})
	}

	// Filter by thumbnail
	if (thumbnailFilter === 'yes') {
		videos = videos.filter(v => v.thumbnail_url && v.thumbnail_url.trim() !== '')
	} else if (thumbnailFilter === 'no') {
		videos = videos.filter(v => !v.thumbnail_url || v.thumbnail_url.trim() === '')
	}

	// Sort
	const sorted = [...videos]
	switch (sortType) {
		case 'newest':
			sorted.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
			break
		case 'oldest':
			sorted.sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
			break
		case 'title-asc':
			sorted.sort((a, b) => a.title.localeCompare(b.title))
			break
		case 'title-desc':
			sorted.sort((a, b) => b.title.localeCompare(a.title))
			break
		case 'duration-desc':
			sorted.sort((a, b) => (b.duration || 0) - (a.duration || 0))
			break
		case 'duration-asc':
			sorted.sort((a, b) => (a.duration || 0) - (b.duration || 0))
			break
		case 'views-desc':
			sorted.sort((a, b) => (b.view_count || 0) - (a.view_count || 0))
			break
		case 'views-asc':
			sorted.sort((a, b) => (a.view_count || 0) - (b.view_count || 0))
			break
		case 'likes-desc':
			sorted.sort((a, b) => (b.like_count || 0) - (a.like_count || 0))
			break
	}

	return sorted
}

const paginatedTabVideos = (tabId) => {
	const filtered = filteredTabVideos(tabId)
	const page = videoPage.value[tabId] || 1
	const start = (page - 1) * videosPerPage
	const end = start + videosPerPage
	return filtered.slice(start, end)
}

const changeVideoPage = (tabId, direction) => {
	const currentPage = videoPage.value[tabId] || 1
	const newPage = currentPage + direction
	const maxPage = Math.ceil(filteredTabVideos(tabId).length / videosPerPage)

	if (newPage >= 1 && newPage <= maxPage) {
		videoPage.value[tabId] = newPage
	}
}

const playScrapedVideo = (video) => {
	// Navigate to video player page with scraped video ID
	router.push(`/watch/scraped/${video.id}`)
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

const formatFileSize = (bytes) => {
	if (!bytes || bytes === 0) return ''
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
	const i = Math.floor(Math.log(bytes) / Math.log(1024))
	return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + sizes[i]
}

const getVideoResolution = (video) => {
	if (!video.metadata) return null

	try {
		const metadata = typeof video.metadata === 'string'
			? JSON.parse(video.metadata)
			: video.metadata

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

const getVideoStatusBadge = (status) => {
	switch (status) {
		case 'scraped': return 'bg-success'
		case 'downloaded': return 'bg-primary'
		case 'pending': return 'bg-warning'
		case 'failed': return 'bg-danger'
		default: return 'bg-secondary'
	}
}

const getJobStatusClass = (status) => {
	switch (status) {
		case 'completed': return 'text-success'
		case 'running': return 'text-info'
		case 'pending': return 'text-warning'
		case 'failed': return 'text-danger'
		case 'cancelled': return 'text-secondary'
		default: return ''
	}
}

// Lifecycle
onMounted(() => {
	checkAuthStatus()
	loadStats()
	loadThreads()
	loadScraperTabs()

	// Load scraped videos if on videos tab
	if (activeTab.value === 'videos') {
		loadScrapedVideos()
	}
})
</script>

<style scoped src="@/styles/pages/scraper_page.css"></style>
