<template>
	<div class="scraper-page">
		<div class="container-fluid py-3">
			<!-- Page Header -->
			<div class="page-header mb-3">
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
			<div v-if="activeTab === 'simpcity'" class="tab-content">
				<div class="scraper-layout" :class="{ 'sidebar-collapsed': collapsedSections.simpcity }">
					<!-- Main Content (Left) -->
					<div class="scraper-main">
						<!-- Search and Filters -->
						<div class="filters-section mb-3">
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
								<option v-for="(count, provider) in stats.provider_breakdown" :key="provider" :value="provider">{{ provider }} ({{ count }})</option>
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
				<EmptyState
					v-else-if="threads.length === 0 && !loading"
					:icon="['fas', 'inbox']"
					icon-size="4x"
					title="No Threads Found"
					message="Start scraping to collect thread data"
				>
					<template #actions>
						<button class="btn btn-primary mt-3" @click="showScrapeModal = true">
							<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
							Start Scraping
						</button>
					</template>
				</EmptyState>

				<!-- Threads Grid View -->
				<div v-else-if="viewMode === 'grid'" class="threads-grid">
					<div
						v-for="thread in threads"
						:key="thread.id"
						class="thread-card"
						:class="{ selected: isThreadSelected(thread.id), watched: thread.is_watched }"
						@click="openThread(thread)"
					>
						<div class="thread-actions-overlay">
							<button
								class="watch-btn"
								:class="{ active: thread.is_watched }"
								@click.stop="toggleThreadWatch(thread)"
								:title="thread.is_watched ? 'Stop watching' : 'Watch for updates'"
							>
								<font-awesome-icon :icon="['fas', thread.is_watched ? 'bell' : 'bell-slash']" />
							</button>
							<button class="verify-btn" @click.stop="verifyThreadLinks(thread)" title="Verify download links">
								<font-awesome-icon :icon="['fas', 'check-double']" />
							</button>
							<div v-if="selectionMode" class="thread-checkbox" @click="toggleThreadSelection(thread.id, $event)">
								<input type="checkbox" :checked="isThreadSelected(thread.id)" @click.stop />
							</div>
						</div>
						<div v-if="thread.metadata?.thumbnail_url || thread.metadata?.thumbnail_urls?.length" class="thread-thumbnail">
							<img
								:src="thread.metadata.thumbnail_url"
								:data-fallback-urls="JSON.stringify(thread.metadata.thumbnail_urls || [])"
								alt="Thread thumbnail"
								@error="handleThumbnailError"
							/>
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
								<!-- Link Health Badge -->
								<span
									v-if="threadLinkStats[thread.id]"
									class="stat-badge"
									:class="getThreadLinkHealthClass(threadLinkStats[thread.id])"
									:title="getThreadLinkHealthTitle(threadLinkStats[thread.id])"
								>
									<font-awesome-icon :icon="['fas', 'link']" class="me-1" />
									{{ threadLinkStats[thread.id].working }}/{{ threadLinkStats[thread.id].total }}
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
					<div v-for="thread in threads" :key="thread.id" class="thread-list-item" :class="{ watched: thread.is_watched }" @click="openThread(thread)">
						<div class="thread-list-content">
							<h5 class="thread-title">
								<font-awesome-icon v-if="thread.is_watched" :icon="['fas', 'bell']" class="watched-indicator me-2" />
								{{ thread.title }}
							</h5>
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
							<button
								class="btn btn-sm"
								:class="thread.is_watched ? 'btn-warning' : 'btn-outline-warning'"
								@click.stop="toggleThreadWatch(thread)"
								:title="thread.is_watched ? 'Stop watching' : 'Watch for updates'"
							>
								<font-awesome-icon :icon="['fas', thread.is_watched ? 'bell' : 'bell-slash']" />
							</button>
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
					<!-- End Main Content -->

					<!-- Sidebar (Right) -->
					<div class="scraper-sidebar" :class="{ collapsed: collapsedSections.simpcity }">
						<button class="collapse-toggle-btn mb-3 w-100" @click="toggleCollapse('simpcity')">
							<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
							Hide Panel
						</button>

						<!-- Auto-Update Panel -->
						<AutoUpdatePanel scraper-type="simpcity" />

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
									<StatCard :value="stats.total_threads || 0" label="Threads" :icon="['fas', 'comments']" icon-class="threads" />
									<StatCard :value="stats.total_posts || 0" label="Posts" :icon="['fas', 'comment-dots']" icon-class="posts" />
									<StatCard :value="stats.total_download_links || 0" label="Links" :icon="['fas', 'link']" icon-class="links" />
									<StatCard :value="stats.active_links || 0" label="Active" :icon="['fas', 'check-circle']" icon-class="active" />
								</div>
							</div>
						</div>

						<!-- Quick Actions -->
						<div class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'bolt']" class="me-2" />
									Quick Actions
								</h6>
							</div>
							<div class="card-body">
								<div class="d-grid gap-2">
									<button class="btn btn-sm btn-outline-light" @click="showAuthModal = true">
										<font-awesome-icon :icon="['fas', 'key']" class="me-2" />
										{{ sessionCookieSet ? 'Auth: Active' : 'Set Auth' }}
									</button>
									<button class="btn btn-sm btn-success" @click="showForumScrapeModal = true">
										<font-awesome-icon :icon="['fas', 'globe']" class="me-2" />
										Scrape Forum
									</button>
									<button class="btn btn-sm btn-primary" @click="showScrapeModal = true">
										<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
										Single Thread
									</button>
									<button class="btn btn-sm btn-warning" @click="autoLinkPerformers">
										<font-awesome-icon :icon="['fas', 'link']" class="me-2" />
										Auto-Link
									</button>
									<button class="btn btn-sm btn-info" @click="checkLinkStatuses">
										<font-awesome-icon :icon="['fas', 'check-circle']" class="me-2" />
										Check Links
									</button>
								</div>
							</div>
						</div>

						<!-- Provider Breakdown -->
						<div v-if="stats.provider_breakdown" class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'chart-pie']" class="me-2" />
									Providers
								</h6>
							</div>
							<div class="card-body">
								<div class="provider-tags">
									<span v-for="(count, provider) in stats.provider_breakdown" :key="provider" class="provider-tag small">
										{{ provider }}: <strong>{{ count }}</strong>
									</span>
								</div>
							</div>
						</div>

						<!-- Auto-update info -->
						<div class="alert alert-info small mb-0">
							<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
							Auto-update checks <strong>watched threads</strong> only.
						</div>
					</div>
					<!-- End Sidebar -->
				</div>
				<!-- End scraper-layout -->

				<!-- Sidebar Toggle Button (shown when collapsed) -->
				<button
					v-if="collapsedSections.simpcity"
					class="sidebar-toggle"
					@click="toggleCollapse('simpcity')"
					title="Show Controls Panel"
				>
					<font-awesome-icon :icon="['fas', 'cog']" />
				</button>
			</div>

			<!-- Authentication Modal -->
			<div v-if="showAuthModal" class="modal-overlay" @click="showAuthModal = false">
				<div class="modal-dialog modal-lg" @click.stop>
					<div class="modal-content">
						<div class="modal-header">
							<h5 class="modal-title">
								<font-awesome-icon :icon="['fas', 'key']" class="me-2" />
								Simpcity Authentication
							</h5>
							<button type="button" class="btn-close" @click="showAuthModal = false"></button>
						</div>
						<div class="modal-body">
							<!-- Login Helper Section -->
							<div class="card bg-dark mb-3">
								<div class="card-body">
									<h6 class="card-title mb-3">
										<font-awesome-icon :icon="['fas', 'magic']" class="me-2 text-primary" />
										Easy Login (Recommended)
									</h6>
									<p class="text-muted small mb-3">
										Click the button below to open Simpcity in Firefox. Log in, then follow the instructions to copy your cookies.
									</p>
									<button type="button" class="btn btn-primary btn-lg w-100 mb-3" @click="openLoginHelper" :disabled="loginHelperLoading">
										<font-awesome-icon :icon="loginHelperLoading ? ['fas', 'spinner'] : ['fas', 'external-link-alt']" :spin="loginHelperLoading" class="me-2" />
										{{ loginHelperLoading ? 'Opening Browser...' : 'Open Simpcity Login (Firefox)' }}
									</button>
									<div v-if="loginHelperInstructions.length > 0" class="alert alert-info small mb-0">
										<strong>After logging in:</strong>
										<ol class="mb-0 mt-2 ps-3">
											<li v-for="(instruction, idx) in loginHelperInstructions" :key="idx">{{ instruction }}</li>
										</ol>
									</div>
								</div>
							</div>

							<!-- Divider -->
							<div class="text-center text-muted my-3">
								<span class="px-3 bg-dark">or paste cookies manually</span>
							</div>

							<!-- Manual Cookie Entry -->
							<div class="mb-3">
								<label class="form-label">Session Cookie</label>
								<textarea
									v-model="authCookie"
									class="form-control"
									rows="4"
									placeholder="Paste cookie table from DevTools (Ctrl+A in cookie list) or cookie header string..."
								></textarea>
								<small class="form-text text-muted">
									Paste the cookie table directly from DevTools (F12 → Application → Cookies → simpcity.is → Ctrl+A → Ctrl+C)
								</small>
							</div>
							<div class="alert alert-info small">
								<font-awesome-icon :icon="['fas', 'lightbulb']" class="me-2" />
								<strong>Tip:</strong> In Chrome DevTools, click on any cookie, press Ctrl+A to select all, then Ctrl+C to copy. The app will automatically parse the
								table format.
							</div>
							<div v-if="sessionCookieSet" class="alert alert-success">
								<font-awesome-icon :icon="['fas', 'check-circle']" class="me-2" />
								Authentication is currently <strong>active</strong>
								<span v-if="verifiedUsername">
									(logged in as <strong>{{ verifiedUsername }}</strong
									>)</span
								>
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
								<input v-model="scrapeURL" type="text" class="form-control" placeholder="https://simpcity.is/threads/thread-name.123456/" />
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
								<div class="detail-row"><strong>Author:</strong> {{ selectedThread.author }}</div>
								<div class="detail-row"><strong>Category:</strong> {{ selectedThread.category }}</div>
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
								<!-- Link Health Status -->
								<div v-if="threadLinkStats[selectedThread.id]" class="detail-row">
									<strong>Link Health:</strong>
									<div class="d-flex gap-3 mt-1">
										<span class="badge bg-success">
											<font-awesome-icon :icon="['fas', 'check']" class="me-1" />
											{{ threadLinkStats[selectedThread.id].working }} Working
										</span>
										<span class="badge bg-danger">
											<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
											{{ threadLinkStats[selectedThread.id].dead }} Dead
										</span>
										<span class="badge bg-secondary">
											<font-awesome-icon :icon="['fas', 'question']" class="me-1" />
											{{ threadLinkStats[selectedThread.id].unknown }} Unknown
										</span>
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
								<div class="detail-row"><strong>Last Scraped:</strong> {{ formatDate(selectedThread.last_scraped_at) }}</div>
							</div>
						</div>
						<div class="modal-footer">
							<button type="button" class="btn btn-secondary" @click="openThreadURL(selectedThread.url)">
								<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-2" />
								Open in Browser
							</button>
							<button type="button" class="btn btn-info" @click="verifyThreadLinks(selectedThread)">
								<font-awesome-icon :icon="['fas', 'check-double']" class="me-2" />
								Verify Links
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
							<!-- Predefined Forum Categories -->
							<div class="mb-4">
								<label class="form-label">Quick Select Forum</label>
								<div class="keyword-tags">
									<button
										v-for="forum in simpcityForums"
										:key="forum.url"
										class="keyword-tag"
										:class="{ active: forumURL === forum.url }"
										@click="forumURL = forum.url"
									>
										{{ forum.name }}
									</button>
								</div>
							</div>

							<div class="mb-3">
								<label class="form-label">Forum Category URL</label>
								<input v-model="forumURL" type="text" class="form-control" placeholder="https://simpcity.is/forums/xxx-porn.50/" />
								<small class="form-text text-muted"> Or enter a custom forum category URL </small>
							</div>
							<div class="alert alert-warning">
								<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-2" />
								<strong>Warning:</strong> This will scrape ALL threads in the forum category. This may take several hours and consume significant bandwidth. The
								scraper will run in the background.
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

			<!-- PornHub Tab Content -->
			<div v-if="activeTab === 'pornhub'" class="tab-content">
				<div class="scraper-layout" :class="{ 'sidebar-collapsed': collapsedSections.pornhub }">
					<!-- Main Content (Left) -->
					<div class="scraper-main">
						<!-- Search and Filters (Always Visible) -->
						<div class="filter-bar mb-3">
							<div class="row g-2 align-items-end">
								<div class="col-md-4">
									<input v-model="pornHubVideoSearch" type="text" class="form-control form-control-sm" placeholder="Search videos..." />
								</div>
								<div class="col-md-3">
									<select v-model="pornHubVideoSort" class="form-select form-select-sm">
										<option value="newest">Newest First</option>
										<option value="oldest">Oldest First</option>
										<option value="views-desc">Most Views</option>
										<option value="duration-desc">Longest</option>
									</select>
								</div>
								<div class="col-md-3">
									<select v-model="pornHubKeywordFilter" class="form-select form-select-sm">
										<option value="">All Keywords</option>
										<option v-for="kw in Object.keys(pornHubStats.keyword_counts || {})" :key="kw" :value="kw">
											{{ kw }} ({{ pornHubStats.keyword_counts[kw] }})
										</option>
									</select>
								</div>
								<div class="col-md-2">
									<button
										class="btn btn-sm w-100"
										:class="pornHubSelectionMode ? 'btn-primary' : 'btn-outline-secondary'"
										@click="togglePornHubSelectionMode"
									>
										<font-awesome-icon :icon="['fas', pornHubSelectionMode ? 'times' : 'check-square']" class="me-1" />
										Select
									</button>
								</div>
							</div>
						</div>

						<!-- Bulk Actions (when selecting) -->
						<div v-if="pornHubSelectionMode && pornHubSelectedVideos.size > 0" class="bulk-actions-toolbar mb-3">
							<div class="d-flex justify-content-between align-items-center">
								<span class="badge bg-primary">{{ pornHubSelectedVideos.size }} selected</span>
								<div class="d-flex gap-2">
									<button class="btn btn-sm btn-outline-secondary" @click="selectAllPornHubVideos">Select All</button>
									<button class="btn btn-sm btn-outline-danger" @click="deleteSelectedPornHubVideos">
										<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />Delete
									</button>
								</div>
							</div>
						</div>

						<!-- Loading State -->
						<LoadingState v-if="pornHubLoading" show-text loading-text="Loading PornHub videos..." />

						<!-- PornHub Videos Grid -->
						<div v-else-if="pornHubVideos.length > 0">
							<div class="d-flex justify-content-between align-items-center mb-2">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									PornHub Videos ({{ filteredPornHubVideos.length }} / {{ pornHubVideos.length }})
								</h6>
								<div class="d-flex gap-2">
									<button class="btn btn-sm btn-outline-info" @click="loadPornHubVideos" title="Refresh" :disabled="pornHubLoading">
										<font-awesome-icon :icon="['fas', pornHubLoading ? 'spinner' : 'sync-alt']" :spin="pornHubLoading" />
									</button>
									<button class="btn btn-sm btn-outline-danger" @click="clearPornHubVideos" title="Clear all">
										<font-awesome-icon :icon="['fas', 'trash-alt']" />
									</button>
								</div>
							</div>
						<div class="video-grid">
							<div
								v-for="video in paginatedPornHubVideos"
								:key="video.id"
								class="video-card"
								:class="{ selected: pornHubSelectedVideos.has(video.id) }"
								@mouseenter="startThumbnailPreview(video)"
								@mouseleave="stopThumbnailPreview(video)"
								@click="pornHubSelectionMode && togglePornHubVideoSelection(video.id)"
							>
								<!-- Selection Checkbox -->
								<div v-if="pornHubSelectionMode" class="video-checkbox" @click.stop="togglePornHubVideoSelection(video.id)">
									<input type="checkbox" :checked="pornHubSelectedVideos.has(video.id)" @change.stop />
								</div>
								<!-- Thumbnail -->
								<div class="video-thumbnail">
									<img
										v-if="video.thumbnail_url"
										:src="getCurrentThumbnail(video)"
										:alt="video.title"
										@error="$event.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'"
									/>
									<div v-else class="no-thumbnail">
										<font-awesome-icon :icon="['fas', 'video']" size="3x" />
									</div>
									<span v-if="video.duration" class="duration-badge">
										{{ formatDuration(video.duration) }}
									</span>
								</div>

								<!-- Video Info -->
								<div class="video-info">
									<h6 class="video-title" :title="video.title">{{ video.title }}</h6>
									<div class="video-meta">
										<span v-if="video.uploader" class="text-muted small d-block mb-1">
											<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
											{{ video.uploader }}
										</span>
										<div class="d-flex flex-wrap gap-2 mt-1">
											<span v-if="video.view_count > 0" class="text-muted small">
												<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
												{{ formatNumber(video.view_count) }}
											</span>
											<span v-if="video.rating > 0" class="text-success small">
												<font-awesome-icon :icon="['fas', 'thumbs-up']" class="me-1" />
												{{ video.rating }}%
											</span>
										</div>
										<!-- Tags (keywords) -->
										<div v-if="video.tags && video.tags.length" class="mt-1">
											<span v-for="tag in video.tags.slice(0, 3)" :key="tag" class="badge bg-secondary me-1 small">
												{{ tag }}
											</span>
										</div>
									</div>

									<!-- Actions -->
									<div class="video-actions mt-2">
										<button @click="playScrapedVideo(video)" class="btn btn-sm btn-primary">
											<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-1" />
											Open
										</button>
										<button
											v-if="jdAvailable && video.source_url"
											@click="sendToJDownloader(video)"
											class="btn btn-sm btn-warning"
											:disabled="jdSending[video.id]"
											title="Send to JDownloader"
										>
											<font-awesome-icon :icon="['fas', jdSending[video.id] ? 'spinner' : 'download']" :spin="jdSending[video.id]" />
										</button>
									</div>
								</div>
							</div>
						</div>

							<!-- Pagination -->
							<div v-if="filteredPornHubVideos.length > videosPerPage" class="pagination-controls p-3">
								<div class="d-flex justify-content-center align-items-center flex-column gap-2">
									<div class="btn-group">
										<button class="btn btn-sm btn-outline-secondary" :disabled="pornHubVideoPage === 1" @click="pornHubVideoPage--; window.scrollTo({ top: 0, behavior: 'smooth' })">
											<font-awesome-icon :icon="['fas', 'chevron-left']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary disabled">
											Page {{ pornHubVideoPage }} / {{ Math.ceil(filteredPornHubVideos.length / videosPerPage) }}
										</button>
										<button
											class="btn btn-sm btn-outline-secondary"
											:disabled="pornHubVideoPage >= Math.ceil(filteredPornHubVideos.length / videosPerPage)"
											@click="pornHubVideoPage++; window.scrollTo({ top: 0, behavior: 'smooth' })"
										>
											<font-awesome-icon :icon="['fas', 'chevron-right']" />
										</button>
									</div>
									<span class="text-muted small">
										Showing {{ (pornHubVideoPage - 1) * videosPerPage + 1 }} -
										{{ Math.min(pornHubVideoPage * videosPerPage, filteredPornHubVideos.length) }}
										of {{ filteredPornHubVideos.length }} videos
									</span>
								</div>
							</div>
						</div>

						<!-- Empty State -->
						<EmptyState
							v-else-if="!pornHubScraping"
							:icon="['fas', 'fire']"
							icon-size="4x"
							title="No PornHub Videos Yet"
							message="Start a keyword search to scrape videos from PornHub"
						/>
					</div>
					<!-- End Main Content -->

					<!-- Sidebar (Right) -->
					<div class="scraper-sidebar" :class="{ collapsed: collapsedSections.pornhub }">
						<button class="collapse-toggle-btn mb-3 w-100" @click="toggleCollapse('pornhub')">
							<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
							Hide Panel
						</button>

						<!-- Auto-Update Panel -->
						<AutoUpdatePanel scraper-type="pornhub" endpoint-type="keywords" />

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
									<StatCard :value="pornHubStats.total_videos || 0" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
									<StatCard :value="Object.keys(pornHubStats.keyword_counts || {}).length" label="Keywords" :icon="['fas', 'tags']" icon-class="keywords" />
									<StatCard :value="pornHubActiveJob ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
									<StatCard :value="pornHubActiveJob?.videosFound || 0" label="Found" :icon="['fas', 'search']" icon-class="found" />
								</div>
							</div>
						</div>

						<!-- Scraper Controls -->
						<div class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'fire']" class="me-2 text-warning" />
									Keyword Scraper
								</h6>
							</div>
							<div class="card-body">
								<div class="mb-2">
									<label class="form-label small">Search Keyword</label>
									<input v-model="pornHubKeyword" type="text" class="form-control form-control-sm" placeholder="Enter keyword..." @keyup.enter="startPornHubScrape" />
								</div>
								<div class="mb-2">
									<label class="form-label small">Max Pages</label>
									<input v-model.number="pornHubMaxPages" type="number" class="form-control form-control-sm" min="1" max="9999" placeholder="0 = unlimited" />
								</div>
								<button class="btn btn-primary btn-sm w-100" @click="startPornHubScrape" :disabled="!pornHubKeyword || pornHubScraping">
									<font-awesome-icon :icon="['fas', pornHubScraping ? 'spinner' : 'search']" :spin="pornHubScraping" class="me-2" />
									{{ pornHubScraping ? 'Scraping...' : 'Start Scrape' }}
								</button>
							</div>
						</div>

						<!-- Suggested Keywords -->
						<div class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'lightbulb']" class="me-2" />
									Quick Keywords
								</h6>
							</div>
							<div class="card-body">
								<div class="keyword-tags">
									<button
										v-for="keyword in pornHubSuggestedKeywords"
										:key="keyword"
										class="keyword-tag"
										:class="{ active: pornHubKeyword === keyword }"
										@click="pornHubKeyword = keyword"
									>
										{{ keyword }}
									</button>
								</div>
							</div>
						</div>

						<!-- Active Job Progress -->
						<div v-if="pornHubActiveJob" class="card bg-dark">
							<div class="card-body">
								<h6 class="mb-2 small">
									<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-primary" />
									"{{ pornHubActiveJob.keyword }}"
								</h6>
								<div class="d-flex flex-wrap gap-2 small">
									<span><strong>{{ pornHubActiveJob.pagesCompleted }}</strong> pages</span>
									<span><strong>{{ pornHubActiveJob.videosFound }}</strong> found</span>
									<span><strong>{{ pornHubActiveJob.videosSaved }}</strong> saved</span>
								</div>
							</div>
						</div>
					</div>
					<!-- End Sidebar -->
				</div>
				<!-- End scraper-layout -->

				<!-- Sidebar Toggle Button (shown when collapsed) -->
				<button
					v-if="collapsedSections.pornhub"
					class="sidebar-toggle"
					@click="toggleCollapse('pornhub')"
					title="Show Controls Panel"
				>
					<font-awesome-icon :icon="['fas', 'cog']" />
				</button>
			</div>
			<!-- End PornHub Tab Content -->

			<!-- LuxureTV Tab Content -->
			<div v-if="activeTab === 'luxuretv'" class="tab-content">
				<div class="scraper-layout" :class="{ 'sidebar-collapsed': collapsedSections.luxuretv }">
					<!-- Main Content (Left) -->
					<div class="scraper-main">
						<!-- Search and Filters (Always Visible) -->
						<div class="filter-bar mb-3">
							<div class="row g-2 align-items-end">
								<div class="col-md-4">
									<input v-model="luxureTVVideoSearch" type="text" class="form-control form-control-sm" placeholder="Search videos..." />
								</div>
								<div class="col-md-3">
									<select v-model="luxureTVVideoSort" class="form-select form-select-sm">
										<option value="newest">Newest First</option>
										<option value="oldest">Oldest First</option>
										<option value="views-desc">Most Views</option>
										<option value="rating-desc">Highest Rated</option>
										<option value="duration-desc">Longest</option>
									</select>
								</div>
								<div class="col-md-3">
									<button
										class="btn btn-sm w-100"
										:class="luxureTVSelectionMode ? 'btn-primary' : 'btn-outline-secondary'"
										@click="toggleLuxureTVSelectionMode"
									>
										<font-awesome-icon :icon="['fas', luxureTVSelectionMode ? 'times' : 'check-square']" class="me-1" />
										Select
									</button>
								</div>
								<div class="col-md-2">
									<button class="btn btn-sm btn-outline-info w-100" @click="loadLuxureTVVideos" :disabled="luxureTVLoading">
										<font-awesome-icon :icon="['fas', luxureTVLoading ? 'spinner' : 'sync-alt']" :spin="luxureTVLoading" />
									</button>
								</div>
							</div>
						</div>

						<!-- Bulk Actions (when selecting) -->
						<div v-if="luxureTVSelectionMode && luxureTVSelectedVideos.size > 0" class="bulk-actions-toolbar mb-3">
							<div class="d-flex justify-content-between align-items-center">
								<span class="badge bg-primary">{{ luxureTVSelectedVideos.size }} selected</span>
								<div class="d-flex gap-2">
									<button class="btn btn-sm btn-outline-secondary" @click="selectAllLuxureTVVideos">Select All</button>
									<button class="btn btn-sm btn-outline-danger" @click="deleteSelectedLuxureTVVideos">
										<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />Delete
									</button>
								</div>
							</div>
						</div>

						<!-- Active Job Progress -->
						<div v-if="luxureTVActiveJob" class="card bg-dark mb-3">
							<div class="card-body py-2">
								<div class="d-flex justify-content-between align-items-center">
									<span class="small">
										<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-primary" />
										Scraping...
									</span>
									<span class="small">
										<strong>{{ luxureTVActiveJob.pagesCompleted }}</strong> pages |
										<strong>{{ luxureTVActiveJob.videosFound }}</strong> found |
										<strong>{{ luxureTVActiveJob.videosSaved }}</strong> saved
									</span>
								</div>
							</div>
						</div>

						<!-- Loading State -->
						<LoadingState v-if="luxureTVLoading" show-text loading-text="Loading LuxureTV videos..." />

						<!-- LuxureTV Videos Grid -->
						<div v-else-if="luxureTVVideos.length > 0">
							<div class="d-flex justify-content-between align-items-center mb-2">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'video']" class="me-2" />
									LuxureTV Videos ({{ filteredLuxureTVVideos.length }} / {{ luxureTVVideos.length }})
								</h6>
								<button class="btn btn-sm btn-outline-danger" @click="clearLuxureTVVideos" title="Clear all">
									<font-awesome-icon :icon="['fas', 'trash-alt']" />
								</button>
							</div>
						<div class="video-grid">
							<div
								v-for="video in paginatedLuxureTVVideos"
								:key="video.id"
								class="video-card"
								:class="{ selected: luxureTVSelectedVideos.has(video.id) }"
								@mouseenter="startThumbnailPreview(video)"
								@mouseleave="stopThumbnailPreview(video)"
								@click="luxureTVSelectionMode && toggleLuxureTVVideoSelection(video.id)"
							>
								<!-- Selection Checkbox -->
								<div v-if="luxureTVSelectionMode" class="video-checkbox" @click.stop="toggleLuxureTVVideoSelection(video.id)">
									<input type="checkbox" :checked="luxureTVSelectedVideos.has(video.id)" @change.stop />
								</div>
								<!-- Thumbnail -->
								<div class="video-thumbnail">
									<img
										v-if="video.thumbnail_url"
										:src="getCurrentThumbnail(video)"
										:alt="video.title"
										@error="$event.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'"
									/>
									<div v-else class="no-thumbnail">
										<font-awesome-icon :icon="['fas', 'video']" size="3x" />
									</div>
									<span v-if="video.duration" class="duration-badge">
										{{ formatDuration(video.duration) }}
									</span>
								</div>

								<!-- Video Info -->
								<div class="video-info">
									<h6 class="video-title" :title="video.title">{{ video.title }}</h6>
									<div class="video-meta">
										<div class="d-flex flex-wrap gap-2 mt-1">
											<span v-if="video.view_count > 0" class="text-muted small">
												<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
												{{ formatNumber(video.view_count) }}
											</span>
											<span v-if="video.rating > 0" class="text-success small">
												<font-awesome-icon :icon="['fas', 'star']" class="me-1" />
												{{ video.rating }}%
											</span>
										</div>
										<!-- Tags -->
										<div v-if="video.tags && video.tags.length" class="mt-1">
											<span v-for="tag in video.tags.slice(0, 3)" :key="tag" class="badge bg-secondary me-1 small">
												{{ tag }}
											</span>
										</div>
									</div>

									<!-- Actions -->
									<div class="video-actions mt-2">
										<button v-if="getEmbedUrl(video)" @click="playEmbeddedVideo(video)" class="btn btn-sm btn-success me-1" title="Play video in app">
											<font-awesome-icon :icon="['fas', 'play']" class="me-1" />
											Play
										</button>
										<button @click="playScrapedVideo(video)" class="btn btn-sm btn-primary">
											<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-1" />
											Open
										</button>
										<button
											v-if="jdAvailable && video.source_url"
											@click="sendToJDownloader(video)"
											class="btn btn-sm btn-warning"
											:disabled="jdSending[video.id]"
											title="Send to JDownloader"
										>
											<font-awesome-icon :icon="['fas', jdSending[video.id] ? 'spinner' : 'download']" :spin="jdSending[video.id]" />
										</button>
									</div>
								</div>
							</div>
						</div>

							<!-- Pagination -->
							<div v-if="filteredLuxureTVVideos.length > videosPerPage" class="pagination-controls p-3">
								<div class="d-flex justify-content-center align-items-center flex-column gap-2">
									<div class="btn-group">
										<button class="btn btn-sm btn-outline-secondary" :disabled="luxureTVVideoPage === 1" @click="luxureTVVideoPage--; window.scrollTo({ top: 0, behavior: 'smooth' })">
											<font-awesome-icon :icon="['fas', 'chevron-left']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary disabled">
											Page {{ luxureTVVideoPage }} / {{ Math.ceil(filteredLuxureTVVideos.length / videosPerPage) }}
										</button>
										<button
											class="btn btn-sm btn-outline-secondary"
											:disabled="luxureTVVideoPage >= Math.ceil(filteredLuxureTVVideos.length / videosPerPage)"
											@click="luxureTVVideoPage++; window.scrollTo({ top: 0, behavior: 'smooth' })"
										>
											<font-awesome-icon :icon="['fas', 'chevron-right']" />
										</button>
									</div>
									<span class="text-muted small">
										Showing {{ (luxureTVVideoPage - 1) * videosPerPage + 1 }} -
										{{ Math.min(luxureTVVideoPage * videosPerPage, filteredLuxureTVVideos.length) }}
										of {{ filteredLuxureTVVideos.length }} videos
									</span>
								</div>
							</div>
						</div>

						<!-- Empty State -->
						<EmptyState
							v-else-if="!luxureTVScraping"
							:icon="['fas', 'paw']"
							icon-size="4x"
							title="No LuxureTV Videos Yet"
							message="Use the sidebar controls to start scraping"
						/>
					</div>
					<!-- End Main Content -->

					<!-- Sidebar (Right) -->
					<div class="scraper-sidebar" :class="{ collapsed: collapsedSections.luxuretv }">
						<button class="collapse-toggle-btn mb-3 w-100" @click="toggleCollapse('luxuretv')">
							<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
							Hide Panel
						</button>

						<!-- Auto-Update Panel -->
						<AutoUpdatePanel scraper-type="luxuretv" endpoint-type="categories" />

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
									<StatCard :value="luxureTVVideos.length" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
									<StatCard :value="luxureTVActiveJob ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
									<StatCard :value="luxureTVActiveJob?.videosFound || 0" label="Found" :icon="['fas', 'search']" icon-class="found" />
									<StatCard :value="luxureTVActiveJob?.videosSaved || 0" label="Saved" :icon="['fas', 'save']" icon-class="links" />
								</div>
							</div>
						</div>

						<!-- Scraper Controls -->
						<div class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'paw']" class="me-2 text-info" />
									Scraper
								</h6>
							</div>
							<div class="card-body">
								<button
									class="btn btn-success btn-sm w-100 mb-2"
									:disabled="luxureTVScraping"
									@click="startLuxureTVScrapeCategory(luxureTVCategories[0])"
								>
									<font-awesome-icon :icon="['fas', luxureTVScraping ? 'spinner' : 'play']" :spin="luxureTVScraping" class="me-1" />
									{{ luxureTVScraping ? 'Scraping...' : 'Start Scrape' }}
								</button>
								<button
									class="btn btn-outline-info btn-sm w-100"
									@click="fetchLuxureTVEmbedIDs"
									:disabled="luxureTVFetchingEmbeds"
								>
									<font-awesome-icon :icon="['fas', luxureTVFetchingEmbeds ? 'spinner' : 'code']" :spin="luxureTVFetchingEmbeds" class="me-1" />
									{{ luxureTVFetchingEmbeds ? 'Fetching...' : 'Fetch Embeds' }}
								</button>
							</div>
						</div>
					</div>
					<!-- End Sidebar -->
				</div>
				<!-- End scraper-layout -->

				<!-- Sidebar Toggle Button (shown when collapsed) -->
				<button
					v-if="collapsedSections.luxuretv"
					class="sidebar-toggle"
					@click="toggleCollapse('luxuretv')"
					title="Show Controls Panel"
				>
					<font-awesome-icon :icon="['fas', 'cog']" />
				</button>
			</div>
			<!-- End LuxureTV Tab Content -->

			<!-- Sexuria Tab Content -->
			<div v-if="activeTab === 'sexuria'" class="tab-content">
				<div class="scraper-layout" :class="{ 'sidebar-collapsed': collapsedSections.sexuria }">
					<!-- Main Content (Left) -->
					<div class="scraper-main">
						<!-- Search and Filters (Always Visible) -->
						<div class="filter-bar mb-3">
							<div class="row g-2 align-items-end">
								<div class="col-md-3">
									<input v-model="sexuriaItemSearch" type="text" class="form-control form-control-sm" placeholder="Search items..." />
								</div>
								<div class="col-md-2">
									<select v-model="sexuriaHostFilter" class="form-select form-select-sm">
										<option value="">All Hosts</option>
										<option v-for="(count, host) in sexuriaStats.host_counts" :key="host" :value="host">{{ host }} ({{ count }})</option>
									</select>
								</div>
								<div class="col-md-2">
									<select v-model="sexuriaItemSort" class="form-select form-select-sm">
										<option value="newest">Newest First</option>
										<option value="oldest">Oldest First</option>
										<option value="title-asc">Title A-Z</option>
										<option value="title-desc">Title Z-A</option>
									</select>
								</div>
								<div class="col-md-2">
									<button
										class="btn btn-sm w-100"
										:class="sexuriaSelectionMode ? 'btn-primary' : 'btn-outline-secondary'"
										@click="toggleSexuriaSelectionMode"
									>
										<font-awesome-icon :icon="['fas', sexuriaSelectionMode ? 'times' : 'check-square']" class="me-1" />
										Select
									</button>
								</div>
								<div class="col-md-3">
									<!-- Tags/Pornstars dropdowns -->
									<div class="d-flex gap-1">
										<div class="dropdown flex-grow-1" v-if="availableSexuriaTags.length > 0">
											<button class="btn btn-sm btn-outline-secondary w-100 dropdown-toggle" type="button" @click="showSexuriaTagDropdown = !showSexuriaTagDropdown">
												Tags {{ sexuriaTagFilter.length > 0 ? `(${sexuriaTagFilter.length})` : '' }}
											</button>
											<div class="dropdown-menu multi-select-dropdown" :class="{ show: showSexuriaTagDropdown }">
												<div class="dropdown-header d-flex justify-content-between align-items-center">
													<span>Tags</span>
													<button class="btn btn-sm btn-link p-0" @click="sexuriaTagFilter = []">Clear</button>
												</div>
												<div class="dropdown-items-container">
													<label v-for="tag in availableSexuriaTags" :key="tag" class="dropdown-item" @click.stop>
														<input type="checkbox" :checked="sexuriaTagFilter.includes(tag)" @change="toggleSexuriaTagFilter(tag)" class="me-2" />
														{{ tag }}
													</label>
												</div>
											</div>
										</div>
										<div class="dropdown flex-grow-1" v-if="availableSexuriaPornstars.length > 0">
											<button class="btn btn-sm btn-outline-secondary w-100 dropdown-toggle" type="button" @click="showSexuriaPornstarDropdown = !showSexuriaPornstarDropdown">
												Stars {{ sexuriaPornstarFilter.length > 0 ? `(${sexuriaPornstarFilter.length})` : '' }}
											</button>
											<div class="dropdown-menu multi-select-dropdown" :class="{ show: showSexuriaPornstarDropdown }">
												<div class="dropdown-header d-flex justify-content-between align-items-center">
													<span>Pornstars</span>
													<button class="btn btn-sm btn-link p-0" @click="sexuriaPornstarFilter = []">Clear</button>
												</div>
												<div class="dropdown-items-container">
													<label v-for="star in availableSexuriaPornstars" :key="star" class="dropdown-item" @click.stop>
														<input type="checkbox" :checked="sexuriaPornstarFilter.includes(star)" @change="toggleSexuriaPornstarFilter(star)" class="me-2" />
														{{ star }}
													</label>
												</div>
											</div>
										</div>
									</div>
								</div>
							</div>
						</div>

						<!-- Bulk Actions (when selecting) -->
						<div v-if="sexuriaSelectionMode && sexuriaSelectedItems.size > 0" class="bulk-actions-toolbar mb-3">
							<div class="d-flex justify-content-between align-items-center">
								<span class="badge bg-primary">{{ sexuriaSelectedItems.size }} selected</span>
								<div class="d-flex gap-2">
									<button class="btn btn-sm btn-outline-secondary" @click="selectAllSexuriaItems">Select All</button>
									<button class="btn btn-sm btn-outline-danger" @click="deleteSelectedSexuriaItems">
										<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />Delete
									</button>
								</div>
							</div>
						</div>

						<!-- Active Job Progress -->
						<div v-if="sexuriaActiveJob" class="card bg-dark mb-3">
							<div class="card-body py-2">
								<div class="d-flex justify-content-between align-items-center">
									<span class="small">
										<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-primary" />
										Scraping Sexuria...
									</span>
									<span class="small">
										<strong>{{ sexuriaActiveJob.pagesCompleted }}</strong> pages |
										<strong>{{ sexuriaActiveJob.itemsFound }}</strong> found |
										<strong>{{ sexuriaActiveJob.itemsSaved }}</strong> saved
									</span>
								</div>
							</div>
						</div>

						<!-- Scraper Config Card (Collapsible in main area) -->
						<div class="card mb-3">
					<div class="card-header d-flex justify-content-between align-items-center">
						<h5 class="mb-0">
							<font-awesome-icon :icon="['fas', 'download']" class="me-2 text-info" />
							Sexuria Download Links Scraper
						</h5>
						<!-- Mode Toggle -->
						<div class="btn-group btn-group-sm">
							<button class="btn" :class="sexuriaAdvancedMode ? 'btn-outline-primary' : 'btn-primary'" @click="sexuriaAdvancedMode = false">
								<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
								Category
							</button>
							<button class="btn" :class="sexuriaAdvancedMode ? 'btn-primary' : 'btn-outline-primary'" @click="sexuriaAdvancedMode = true">
								<font-awesome-icon :icon="['fas', 'sliders-h']" class="me-1" />
								Advanced Search
							</button>
						</div>
					</div>
					<div class="card-body">
						<p class="text-muted mb-4">
							Scrape download links from Sexuria.net. This site shares links to file hosting services like FastFile.cc, Rapidgator.net, Keep2share, etc.
						</p>

						<!-- Simple Category Mode -->
						<div v-if="!sexuriaAdvancedMode">
							<!-- Category URL Input -->
							<div class="row g-3 mb-4">
								<div class="col-md-6">
									<label class="form-label">Category URL</label>
									<input
										v-model="sexuriaCategoryURL"
										type="text"
										class="form-control"
										placeholder="https://sexuria.net/movies/"
										@keyup.enter="startSexuriaScrape"
									/>
									<small class="text-muted">Paste the category page URL from Sexuria</small>
								</div>
								<div class="col-md-3">
									<label class="form-label">Max Pages</label>
									<input v-model.number="sexuriaMaxPages" type="number" class="form-control" min="1" max="9999" placeholder="0 = unlimited" />
									<small class="text-muted">Leave at 0 for unlimited pages</small>
								</div>
								<div class="col-md-3 d-flex align-items-end">
									<button class="btn btn-primary w-100" @click="startSexuriaScrape" :disabled="!sexuriaCategoryURL || sexuriaScraping">
										<font-awesome-icon :icon="['fas', sexuriaScraping ? 'spinner' : 'search']" :spin="sexuriaScraping" class="me-2" />
										{{ sexuriaScraping ? 'Scraping...' : 'Start Scrape' }}
									</button>
								</div>
							</div>

							<!-- Available Categories -->
							<div class="mb-4">
								<label class="form-label">Available Categories</label>
								<div class="keyword-tags">
									<button
										v-for="cat in sexuriaCategories"
										:key="cat.url"
										class="keyword-tag"
										:class="{ active: sexuriaCategoryURL === cat.url }"
										@click="sexuriaCategoryURL = cat.url"
									>
										{{ cat.name }}
									</button>
								</div>
							</div>
						</div>

						<!-- Advanced Search Mode -->
						<div v-else>
							<!-- Search Categories -->
							<div class="mb-4">
								<label class="form-label">
									<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
									Categories
								</label>
								<div class="keyword-tags">
									<button
										v-for="cat in sexuriaSearchOptions.categories"
										:key="cat.id"
										class="keyword-tag"
										:class="{ active: sexuriaAdvancedSearch.categories.includes(cat.id) }"
										@click="toggleSexuriaSearchCategory(cat.id)"
									>
										{{ cat.name }}
									</button>
								</div>
								<small class="text-muted">Select one or more categories to search in</small>
							</div>

							<!-- Text Search Fields -->
							<div class="row g-3 mb-4">
								<div class="col-md-6">
									<label class="form-label">
										<font-awesome-icon :icon="['fas', 'heading']" class="me-1" />
										Title
									</label>
									<input v-model="sexuriaAdvancedSearch.title" type="text" class="form-control" placeholder="Search by title..." />
								</div>
								<div class="col-md-6">
									<label class="form-label">
										<font-awesome-icon :icon="['fas', 'align-left']" class="me-1" />
										Description
									</label>
									<input v-model="sexuriaAdvancedSearch.description" type="text" class="form-control" placeholder="Search in description..." />
								</div>
							</div>

							<div class="row g-3 mb-4">
								<div class="col-md-4">
									<label class="form-label">
										<font-awesome-icon :icon="['fas', 'star']" class="me-1" />
										Pornstars
									</label>
									<input v-model="sexuriaAdvancedSearch.pornstars" type="text" class="form-control" placeholder="e.g. Angela White" />
									<small class="text-muted">Search by performer name</small>
								</div>
								<div class="col-md-4">
									<label class="form-label">
										<font-awesome-icon :icon="['fas', 'building']" class="me-1" />
										Production / Studio
									</label>
									<input v-model="sexuriaAdvancedSearch.production" type="text" class="form-control" placeholder="e.g. Brazzers" />
									<small class="text-muted">Search by production company</small>
								</div>
								<div class="col-md-4">
									<label class="form-label">
										<font-awesome-icon :icon="['fas', 'tags']" class="me-1" />
										Tags
									</label>
									<input v-model="sexuriaAdvancedSearch.tags" type="text" class="form-control" placeholder="e.g. anal, threesome" />
									<small class="text-muted">Comma-separated tags</small>
								</div>
							</div>

							<!-- Resolutions -->
							<div class="mb-4">
								<label class="form-label">
									<font-awesome-icon :icon="['fas', 'tv']" class="me-1" />
									Resolutions
								</label>
								<div class="keyword-tags">
									<button
										v-for="res in sexuriaSearchOptions.resolutions"
										:key="res"
										class="keyword-tag"
										:class="{ active: sexuriaAdvancedSearch.resolutions.includes(res) }"
										@click="toggleSexuriaSearchResolution(res)"
									>
										{{ res }}
									</button>
								</div>
							</div>

							<!-- File Hosts -->
							<div class="mb-4">
								<label class="form-label">
									<font-awesome-icon :icon="['fas', 'server']" class="me-1" />
									File Hosts
								</label>
								<div class="keyword-tags">
									<button
										v-for="host in sexuriaSearchOptions.fileHosts"
										:key="host.id"
										class="keyword-tag"
										:class="{ active: sexuriaAdvancedSearch.fileHosts.includes(host.id) }"
										@click="toggleSexuriaSearchFileHost(host.id)"
									>
										{{ host.name }}
									</button>
								</div>
								<small class="text-muted">Filter by file hosting services</small>
							</div>

							<!-- Max Pages & Actions -->
							<div class="row g-3 mb-4">
								<div class="col-md-3">
									<label class="form-label">Max Pages</label>
									<input v-model.number="sexuriaAdvancedSearch.maxPages" type="number" class="form-control" min="0" max="9999" placeholder="0 = unlimited" />
									<small class="text-muted">Leave at 0 for unlimited pages</small>
								</div>
								<div class="col-md-9 d-flex align-items-end gap-2">
									<button class="btn btn-outline-secondary" @click="resetSexuriaAdvancedSearch" :disabled="sexuriaScraping">
										<font-awesome-icon :icon="['fas', 'undo']" class="me-1" />
										Reset
									</button>
									<button class="btn btn-primary flex-grow-1" @click="startSexuriaAdvancedSearch" :disabled="sexuriaScraping">
										<font-awesome-icon :icon="['fas', sexuriaScraping ? 'spinner' : 'search']" :spin="sexuriaScraping" class="me-2" />
										{{ sexuriaScraping ? 'Searching...' : 'Start Advanced Search' }}
									</button>
								</div>
							</div>
						</div>

						<!-- Host Stats (shown in both modes) -->
						<div v-if="sexuriaStats.host_counts && Object.keys(sexuriaStats.host_counts).length > 0" class="mb-4">
							<label class="form-label">File Hosts Distribution</label>
							<div class="provider-tags">
								<span v-for="(count, host) in sexuriaStats.host_counts" :key="host" class="provider-tag">
									<font-awesome-icon :icon="['fas', 'server']" class="me-1" />
									{{ host }}: <strong>{{ count }}</strong>
								</span>
							</div>
						</div>

					</div>
				</div>

						<!-- Loading State -->
						<LoadingState v-if="sexuriaLoading" show-text loading-text="Loading Sexuria items..." />

						<!-- Sexuria Items Grid -->
						<div v-else-if="sexuriaItems.length > 0">
							<div class="d-flex justify-content-between align-items-center mb-2">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'file-video']" class="me-2" />
									Sexuria Items ({{ filteredSexuriaItems.length }} / {{ sexuriaItems.length }})
								</h6>
								<div class="d-flex gap-2">
									<button class="btn btn-sm btn-outline-info" @click="loadSexuriaItems" :disabled="sexuriaLoading">
										<font-awesome-icon :icon="['fas', sexuriaLoading ? 'spinner' : 'sync-alt']" :spin="sexuriaLoading" />
									</button>
									<button class="btn btn-sm btn-outline-danger" @click="clearSexuriaItems" title="Clear all">
										<font-awesome-icon :icon="['fas', 'trash-alt']" />
									</button>
								</div>
							</div>

							<div class="video-grid">
							<div
								v-for="item in paginatedSexuriaItems"
								:key="item.id"
								class="video-card sexuria-card"
								:class="{ selected: sexuriaSelectedItems.has(item.id) }"
								@mouseenter="handleSexuriaItemHover(item, $event)"
								@mouseleave="hideSexuriaDetails"
								@click="sexuriaSelectionMode && toggleSexuriaItemSelection(item.id)"
							>
								<!-- Selection Checkbox -->
								<div v-if="sexuriaSelectionMode" class="video-checkbox" @click.stop="toggleSexuriaItemSelection(item.id)">
									<input type="checkbox" :checked="sexuriaSelectedItems.has(item.id)" @change.stop />
								</div>
								<!-- Thumbnail -->
								<div class="video-thumbnail">
									<img
										v-if="item.thumbnail_url"
										:src="item.thumbnail_url"
										:alt="item.title"
										@error="$event.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'"
									/>
									<div v-else class="no-thumbnail">
										<font-awesome-icon :icon="['fas', 'file-video']" size="3x" />
									</div>
								</div>

								<!-- Item Info -->
								<div class="video-info">
									<h6 class="video-title" :title="item.title">{{ item.title }}</h6>
									<div class="video-meta">
										<!-- File Size -->
										<div v-if="getSexuriaFileSize(item)" class="text-muted small">
											<font-awesome-icon :icon="['fas', 'hdd']" class="me-1" />
											{{ getSexuriaFileSize(item) }}
										</div>
										<!-- Author -->
										<div v-if="item.uploader" class="text-muted small">
											<font-awesome-icon :icon="['fas', 'user']" class="me-1" />
											{{ item.uploader }}
										</div>
										<!-- File Hosts (from tags) -->
										<div v-if="item.tags && item.tags.length" class="mt-1">
											<span v-for="host in item.tags.slice(0, 4)" :key="host" class="badge bg-info me-1 small">
												{{ host }}
											</span>
											<span v-if="item.tags.length > 4" class="badge bg-secondary small"> +{{ item.tags.length - 4 }} </span>
										</div>
										<!-- Link Health Indicator -->
										<div v-if="item.links_extracted" class="mt-1">
											<span
												v-if="sexuriaLinkStats[item.id]"
												class="badge small"
												:class="getLinkHealthBadgeClass(sexuriaLinkStats[item.id])"
												:title="getLinkHealthTitle(sexuriaLinkStats[item.id])"
											>
												<font-awesome-icon :icon="['fas', 'link']" class="me-1" />
												{{ sexuriaLinkStats[item.id].total }} links
											</span>
											<span v-else class="badge bg-secondary small">
												<font-awesome-icon :icon="['fas', 'link']" class="me-1" />
												Links extracted
											</span>
										</div>
									</div>

									<!-- Actions -->
									<div class="video-actions mt-2">
										<button @click="openSexuriaItem(item)" class="btn btn-sm btn-primary">
											<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-1" />
											Open
										</button>
										<button
											v-if="!item.links_extracted"
											@click.stop="extractSexuriaItemLinks(item)"
											class="btn btn-sm btn-outline-info"
											:disabled="extractingLinks[item.id]"
											title="Extract download links from this item"
										>
											<font-awesome-icon :icon="extractingLinks[item.id] ? ['fas', 'spinner'] : ['fas', 'download']" :spin="extractingLinks[item.id]" />
										</button>
										<button v-else @click.stop="showSexuriaItemLinks(item)" class="btn btn-sm btn-outline-success" title="View download links">
											<font-awesome-icon :icon="['fas', 'link']" />
										</button>
									</div>
								</div>
							</div>
						</div>

						<!-- Sexuria Details Hover Card -->
						<Teleport to="body">
							<div
								v-if="sexuriaHoverDetails.visible"
								class="sexuria-details-card"
								:style="sexuriaHoverDetails.style"
								@mouseenter="keepSexuriaDetailsVisible"
								@mouseleave="hideSexuriaDetails"
							>
								<div class="details-header">
									<h6>{{ sexuriaHoverDetails.item?.title }}</h6>
									<button class="btn-close-details" @click="hideSexuriaDetails">
										<font-awesome-icon :icon="['fas', 'times']" />
									</button>
								</div>

								<!-- Loading State -->
								<div v-if="sexuriaHoverDetails.loading" class="details-loading">
									<font-awesome-icon :icon="['fas', 'spinner']" spin size="2x" />
									<p>Loading details...</p>
								</div>

								<!-- Details Content -->
								<div v-else-if="sexuriaHoverDetails.data" class="details-content">
									<!-- Screenshots -->
									<div v-if="sexuriaHoverDetails.data.screenshots?.length" class="details-section">
										<h6 class="section-title">
											<font-awesome-icon :icon="['fas', 'images']" class="me-2" />
											Screenshots ({{ sexuriaHoverDetails.data.screenshots.length }})
										</h6>

										<div class="screenshots-grid">
											<img
												v-for="(screenshot, idx) in sexuriaHoverDetails.data.screenshots.slice(0, 6)"
												:key="idx"
												:src="screenshot"
												:alt="`Screenshot ${idx + 1}`"
												class="screenshot-thumb"
												:class="{ active: sexuriaHoveredScreenshot === screenshot }"
												@mouseenter="sexuriaHoveredScreenshot = screenshot"
												@mouseleave="sexuriaHoveredScreenshot = null"
												@click="openScreenshot(screenshot)"
												@error="$event.target.style.display = 'none'"
											/>
										</div>
										<small v-if="sexuriaHoverDetails.data.screenshots.length > 6" class="text-muted">
											+{{ sexuriaHoverDetails.data.screenshots.length - 6 }} more
										</small>
									</div>

									<!-- Tags -->
									<div v-if="sexuriaHoverDetails.data.tags?.length" class="details-section">
										<h6 class="section-title">
											<font-awesome-icon :icon="['fas', 'tags']" class="me-2" />
											Tags
										</h6>
										<div class="tags-list">
											<span v-for="tag in sexuriaHoverDetails.data.tags" :key="tag" class="detail-tag">
												{{ tag }}
											</span>
										</div>
									</div>

									<!-- Porn Stars -->
									<div v-if="sexuriaHoverDetails.data.porn_stars?.length" class="details-section">
										<h6 class="section-title">
											<font-awesome-icon :icon="['fas', 'star']" class="me-2" />
											Porn Stars ({{ sexuriaHoverDetails.data.porn_stars.length }})
										</h6>
										<div class="stars-list">
											<span v-for="star in sexuriaHoverDetails.data.porn_stars.slice(0, 10)" :key="star" class="star-tag">
												{{ star }}
											</span>
											<span v-if="sexuriaHoverDetails.data.porn_stars.length > 10" class="star-tag more">
												+{{ sexuriaHoverDetails.data.porn_stars.length - 10 }} more
											</span>
										</div>
									</div>

									<!-- No Details -->
									<div
										v-if="
											!sexuriaHoverDetails.data.screenshots?.length && !sexuriaHoverDetails.data.tags?.length && !sexuriaHoverDetails.data.porn_stars?.length
										"
										class="no-details"
									>
										<font-awesome-icon :icon="['fas', 'info-circle']" class="me-2" />
										No additional details available
									</div>
								</div>

								<!-- Error State -->
								<div v-else-if="sexuriaHoverDetails.error" class="details-error">
									<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-2" />
									{{ sexuriaHoverDetails.error }}
								</div>
							</div>
						</Teleport>

						<!-- Sexuria Screenshot Modal -->
						<Teleport to="body">
							<div v-if="sexuriaScreenshotModal.visible" class="screenshot-modal-backdrop" @click="closeSexuriaScreenshotModal"></div>
							<div v-if="sexuriaScreenshotModal.visible" class="screenshot-modal" @click.self="closeSexuriaScreenshotModal">
								<button class="screenshot-modal-close" @click="closeSexuriaScreenshotModal">
									<font-awesome-icon :icon="['fas', 'times']" />
								</button>

								<!-- Navigation arrows -->
								<button v-if="sexuriaScreenshotModal.screenshots.length > 1" class="screenshot-modal-nav screenshot-modal-prev" @click.stop="prevSexuriaScreenshot">
									<font-awesome-icon :icon="['fas', 'chevron-left']" />
								</button>
								<button v-if="sexuriaScreenshotModal.screenshots.length > 1" class="screenshot-modal-nav screenshot-modal-next" @click.stop="nextSexuriaScreenshot">
									<font-awesome-icon :icon="['fas', 'chevron-right']" />
								</button>

								<!-- Screenshot image -->
								<img v-if="currentSexuriaScreenshot" :src="currentSexuriaScreenshot" alt="Screenshot" class="screenshot-modal-image" @click.stop />

								<!-- Counter -->
								<div v-if="sexuriaScreenshotModal.screenshots.length > 1" class="screenshot-modal-counter">
									{{ sexuriaScreenshotModal.currentIndex + 1 }} / {{ sexuriaScreenshotModal.screenshots.length }}
								</div>
							</div>
						</Teleport>

						<!-- Sexuria Links Modal -->
						<Teleport to="body">
							<div v-if="sexuriaLinksModal.visible" class="modal-backdrop fade show" @click="closeSexuriaLinksModal"></div>
							<div v-if="sexuriaLinksModal.visible" class="modal fade show d-block" tabindex="-1">
								<div class="modal-dialog modal-lg modal-dialog-centered modal-dialog-scrollable">
									<div class="modal-content bg-dark">
										<div class="modal-header border-secondary">
											<h5 class="modal-title">
												<font-awesome-icon :icon="['fas', 'link']" class="me-2" />
												Download Links
											</h5>
											<button type="button" class="btn-close btn-close-white" @click="closeSexuriaLinksModal"></button>
										</div>
										<div class="modal-body">
											<!-- Item Title -->
											<div v-if="sexuriaLinksModal.item" class="mb-3">
												<h6 class="text-truncate" :title="sexuriaLinksModal.item.title">
													{{ sexuriaLinksModal.item.title }}
												</h6>
											</div>

											<!-- Loading -->
											<div v-if="sexuriaLinksModal.loading" class="text-center py-4">
												<font-awesome-icon :icon="['fas', 'spinner']" spin size="2x" class="text-primary" />
												<p class="mt-2 text-muted">Loading links...</p>
											</div>

											<!-- Links List -->
											<div v-else-if="sexuriaLinksModal.links.length > 0" class="links-list">
												<div
													v-for="link in sexuriaLinksModal.links"
													:key="link.id"
													class="link-item d-flex align-items-center p-2 mb-2 rounded"
													:class="{
														'bg-success bg-opacity-10 border-success': link.status === 'working',
														'bg-danger bg-opacity-10 border-danger': link.status === 'dead',
														'bg-secondary bg-opacity-10': link.status === 'unknown',
													}"
												>
													<!-- Provider Badge -->
													<span class="badge bg-info me-2" style="min-width: 80px">{{ link.provider }}</span>

													<!-- Status Badge -->
													<span class="badge me-2" :class="getLinkStatusBadgeClass(link.status)" style="min-width: 60px">
														{{ link.status }}
													</span>

													<!-- URL (truncated) -->
													<a :href="link.url" target="_blank" class="text-decoration-none text-light flex-grow-1 text-truncate" :title="link.url">
														{{ link.url }}
													</a>

													<!-- Copy Button -->
													<button class="btn btn-sm btn-outline-secondary ms-2" @click="copyLinkToClipboard(link.url)" title="Copy link">
														<font-awesome-icon :icon="['fas', 'copy']" />
													</button>
												</div>

												<!-- Summary -->
												<div class="mt-3 p-2 bg-dark rounded border border-secondary">
													<div class="d-flex justify-content-around text-center">
														<div>
															<div class="text-success fw-bold">{{ sexuriaLinksModal.links.filter((l) => l.status === 'working').length }}</div>
															<small class="text-muted">Working</small>
														</div>
														<div>
															<div class="text-danger fw-bold">{{ sexuriaLinksModal.links.filter((l) => l.status === 'dead').length }}</div>
															<small class="text-muted">Dead</small>
														</div>
														<div>
															<div class="text-secondary fw-bold">{{ sexuriaLinksModal.links.filter((l) => l.status === 'unknown').length }}</div>
															<small class="text-muted">Unknown</small>
														</div>
													</div>
												</div>
											</div>

											<!-- No Links -->
											<div v-else class="text-center py-4 text-muted">
												<font-awesome-icon :icon="['fas', 'link-slash']" size="2x" />
												<p class="mt-2">No download links found</p>
											</div>
										</div>
										<div class="modal-footer border-secondary">
											<button
												class="btn btn-primary"
												@click="verifySexuriaItemLinks"
												:disabled="sexuriaLinksModal.verifying || sexuriaLinksModal.links.length === 0"
											>
												<font-awesome-icon
													:icon="sexuriaLinksModal.verifying ? ['fas', 'spinner'] : ['fas', 'check-double']"
													:spin="sexuriaLinksModal.verifying"
													class="me-1"
												/>
												{{ sexuriaLinksModal.verifying ? 'Verifying...' : 'Verify All Links' }}
											</button>
											<button class="btn btn-secondary" @click="closeSexuriaLinksModal">Close</button>
										</div>
									</div>
								</div>
							</div>
						</Teleport>

							<!-- Pagination -->
							<div v-if="filteredSexuriaItems.length > itemsPerPage" class="pagination-controls p-3">
								<div class="d-flex justify-content-center align-items-center flex-column gap-2">
									<div class="btn-group">
										<button class="btn btn-sm btn-outline-secondary" :disabled="sexuriaItemPage === 1" @click="sexuriaItemPage--; window.scrollTo({ top: 0, behavior: 'smooth' })">
											<font-awesome-icon :icon="['fas', 'chevron-left']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary disabled">
											Page {{ sexuriaItemPage }} / {{ Math.ceil(filteredSexuriaItems.length / itemsPerPage) }}
										</button>
										<button
											class="btn btn-sm btn-outline-secondary"
											:disabled="sexuriaItemPage >= Math.ceil(filteredSexuriaItems.length / itemsPerPage)"
											@click="sexuriaItemPage++; window.scrollTo({ top: 0, behavior: 'smooth' })"
										>
											<font-awesome-icon :icon="['fas', 'chevron-right']" />
										</button>
									</div>
									<span class="text-muted small">
										Showing {{ (sexuriaItemPage - 1) * itemsPerPage + 1 }} -
										{{ Math.min(sexuriaItemPage * itemsPerPage, filteredSexuriaItems.length) }}
										of {{ filteredSexuriaItems.length }} items
									</span>
								</div>
							</div>
						</div>

						<!-- Empty State -->
						<EmptyState
							v-else-if="!sexuriaScraping"
							:icon="['fas', 'download']"
							icon-size="4x"
							title="No Sexuria Items Yet"
							message="Use the sidebar controls to start scraping"
						/>
					</div>
					<!-- End Main Content -->

					<!-- Sidebar (Right) -->
					<div class="scraper-sidebar" :class="{ collapsed: collapsedSections.sexuria }">
						<button class="collapse-toggle-btn mb-3 w-100" @click="toggleCollapse('sexuria')">
							<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
							Hide Panel
						</button>

						<!-- Auto-Update Panel -->
						<AutoUpdatePanel scraper-type="sexuria" endpoint-type="categories" />

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
									<StatCard :value="sexuriaStats.total_items || 0" label="Items" :icon="['fas', 'file-video']" icon-class="videos" />
									<StatCard :value="Object.keys(sexuriaStats.host_counts || {}).length" label="Hosts" :icon="['fas', 'server']" icon-class="posts" />
									<StatCard :value="sexuriaActiveJob ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
									<StatCard :value="sexuriaItems.length" label="Loaded" :icon="['fas', 'database']" icon-class="links" />
								</div>
							</div>
						</div>

						<!-- Quick Scrape -->
						<div class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'bolt']" class="me-2" />
									Quick Scrape
								</h6>
							</div>
							<div class="card-body">
								<div class="keyword-tags mb-2">
									<button
										v-for="cat in sexuriaCategories.slice(0, 6)"
										:key="cat.url"
										class="keyword-tag"
										:class="{ active: sexuriaCategoryURL === cat.url }"
										@click="sexuriaCategoryURL = cat.url"
									>
										{{ cat.name }}
									</button>
								</div>
								<button class="btn btn-primary btn-sm w-100" @click="startSexuriaScrape" :disabled="!sexuriaCategoryURL || sexuriaScraping">
									<font-awesome-icon :icon="['fas', sexuriaScraping ? 'spinner' : 'search']" :spin="sexuriaScraping" class="me-2" />
									{{ sexuriaScraping ? 'Scraping...' : 'Start Scrape' }}
								</button>
							</div>
						</div>

						<!-- Host Distribution -->
						<div v-if="sexuriaStats.host_counts && Object.keys(sexuriaStats.host_counts).length > 0" class="card">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'server']" class="me-2" />
									File Hosts
								</h6>
							</div>
							<div class="card-body p-2">
								<div class="d-flex flex-wrap gap-1">
									<span v-for="(count, host) in sexuriaStats.host_counts" :key="host" class="badge bg-info">
										{{ host }}: {{ count }}
									</span>
								</div>
							</div>
						</div>
					</div>
					<!-- End Sidebar -->
				</div>
				<!-- End scraper-layout -->

				<!-- Sidebar Toggle Button (shown when collapsed) -->
				<button
					v-if="collapsedSections.sexuria"
					class="sidebar-toggle"
					@click="toggleCollapse('sexuria')"
					title="Show Controls Panel"
				>
					<font-awesome-icon :icon="['fas', 'cog']" />
				</button>
			</div>
			<!-- End Sexuria Tab Content -->

			<!-- Rule34Video Tab Content -->
			<div v-if="activeTab === 'rule34video'" class="tab-content">
				<div class="scraper-layout" :class="{ 'sidebar-collapsed': collapsedSections.rule34video }">
					<!-- Main Content (Left) -->
					<div class="scraper-main">
						<!-- Search and Filters (Always Visible) -->
						<div class="filter-bar mb-3">
							<div class="row g-2 align-items-end">
								<div class="col-md-4">
									<input v-model="rule34VideoVideoSearch" type="text" class="form-control form-control-sm" placeholder="Search videos..." />
								</div>
								<div class="col-md-3">
									<select v-model="rule34VideoVideoSort" class="form-select form-select-sm">
										<option value="newest">Newest First</option>
										<option value="oldest">Oldest First</option>
										<option value="views-desc">Most Views</option>
										<option value="rating-desc">Top Rated</option>
										<option value="duration-desc">Longest</option>
									</select>
								</div>
								<div class="col-md-3">
									<button
										class="btn btn-sm w-100"
										:class="rule34VideoSelectionMode ? 'btn-primary' : 'btn-outline-secondary'"
										@click="toggleRule34VideoSelectionMode"
									>
										<font-awesome-icon :icon="['fas', rule34VideoSelectionMode ? 'times' : 'check-square']" class="me-1" />
										Select
									</button>
								</div>
								<div class="col-md-2">
									<button class="btn btn-sm btn-outline-info w-100" @click="loadRule34VideoVideos" :disabled="rule34VideoLoading">
										<font-awesome-icon :icon="['fas', rule34VideoLoading ? 'spinner' : 'sync-alt']" :spin="rule34VideoLoading" />
									</button>
								</div>
							</div>
						</div>

						<!-- Bulk Actions (when selecting) -->
						<div v-if="rule34VideoSelectionMode && rule34VideoSelectedVideos.size > 0" class="bulk-actions-toolbar mb-3">
							<div class="d-flex justify-content-between align-items-center">
								<span class="badge bg-primary">{{ rule34VideoSelectedVideos.size }} selected</span>
								<div class="d-flex gap-2">
									<button class="btn btn-sm btn-outline-secondary" @click="selectAllRule34VideoVideos">Select All</button>
									<button class="btn btn-sm btn-outline-danger" @click="deleteSelectedRule34VideoVideos">
										<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />Delete
									</button>
								</div>
							</div>
						</div>

						<!-- Active Job Progress -->
						<div v-if="rule34VideoActiveJob" class="card bg-dark mb-3">
							<div class="card-body py-2">
								<div class="d-flex justify-content-between align-items-center">
									<span class="small">
										<font-awesome-icon :icon="['fas', 'spinner']" spin class="me-2 text-primary" />
										Scraping Rule34Video...
									</span>
									<span class="small">
										<strong>{{ rule34VideoActiveJob.pagesCompleted }}</strong> pages |
										<strong>{{ rule34VideoActiveJob.videosFound }}</strong> found |
										<strong>{{ rule34VideoActiveJob.videosSaved }}</strong> saved
									</span>
								</div>
							</div>
						</div>

						<!-- Loading State -->
						<LoadingState v-if="rule34VideoLoading" show-text loading-text="Loading Rule34Video videos..." />

						<!-- Rule34Video Videos Grid -->
						<div v-else-if="rule34VideoVideos.length > 0">
							<div class="d-flex justify-content-between align-items-center mb-2">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'cube']" class="me-2" />
									Rule34Videos ({{ filteredRule34VideoVideos.length }} / {{ rule34VideoVideos.length }})
								</h6>
								<button class="btn btn-sm btn-outline-danger" @click="clearRule34VideoVideos" title="Clear all">
									<font-awesome-icon :icon="['fas', 'trash-alt']" />
								</button>
							</div>
						<div class="video-grid">
							<div
								v-for="video in paginatedRule34VideoVideos"
								:key="video.id"
								class="video-card"
								:class="{ selected: rule34VideoSelectedVideos.has(video.id) }"
								@mouseenter="startThumbnailPreview(video)"
								@mouseleave="stopThumbnailPreview(video)"
								@click="rule34VideoSelectionMode && toggleRule34VideoVideoSelection(video.id)"
							>
								<!-- Selection Checkbox -->
								<div v-if="rule34VideoSelectionMode" class="video-checkbox" @click.stop="toggleRule34VideoVideoSelection(video.id)">
									<input type="checkbox" :checked="rule34VideoSelectedVideos.has(video.id)" @change.stop />
								</div>
								<!-- Thumbnail -->
								<div class="video-thumbnail">
									<img
										v-if="video.thumbnail_url"
										:src="getCurrentThumbnail(video)"
										:alt="video.title"
										@error="$event.target.src = 'https://via.placeholder.com/320x180?text=No+Thumbnail'"
									/>
									<div v-else class="no-thumbnail">
										<font-awesome-icon :icon="['fas', 'cube']" size="3x" />
									</div>
									<span v-if="video.duration" class="duration-badge">
										{{ formatDuration(video.duration) }}
									</span>
								</div>

								<!-- Video Info -->
								<div class="video-info">
									<h6 class="video-title" :title="video.title">{{ video.title }}</h6>
									<div class="video-meta">
										<div class="d-flex flex-wrap gap-2 mt-1">
											<span v-if="video.view_count > 0" class="text-muted small">
												<font-awesome-icon :icon="['fas', 'eye']" class="me-1" />
												{{ formatNumber(video.view_count) }}
											</span>
											<span v-if="video.rating > 0" class="text-success small">
												<font-awesome-icon :icon="['fas', 'thumbs-up']" class="me-1" />
												{{ video.rating }}%
											</span>
										</div>
										<!-- Tags -->
										<div v-if="video.tags && video.tags.length" class="mt-1">
											<span v-for="tag in video.tags.slice(0, 3)" :key="tag" class="badge bg-info me-1 small">
												{{ tag }}
											</span>
										</div>
									</div>

									<!-- Actions -->
									<div class="video-actions mt-2">
										<button @click="playScrapedVideo(video)" class="btn btn-sm btn-primary">
											<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-1" />
											Open
										</button>
										<button
											v-if="jdAvailable && video.source_url"
											@click="sendToJDownloader(video)"
											class="btn btn-sm btn-warning"
											:disabled="jdSending[video.id]"
											title="Send to JDownloader"
										>
											<font-awesome-icon :icon="['fas', jdSending[video.id] ? 'spinner' : 'download']" :spin="jdSending[video.id]" />
										</button>
									</div>
								</div>
							</div>
						</div>

							<!-- Pagination -->
							<div v-if="filteredRule34VideoVideos.length > videosPerPage" class="pagination-controls p-3">
								<div class="d-flex justify-content-center align-items-center flex-column gap-2">
									<div class="btn-group">
										<button class="btn btn-sm btn-outline-secondary" :disabled="rule34VideoVideoPage === 1" @click="rule34VideoVideoPage--; window.scrollTo({ top: 0, behavior: 'smooth' })">
											<font-awesome-icon :icon="['fas', 'chevron-left']" />
										</button>
										<button class="btn btn-sm btn-outline-secondary disabled">
											Page {{ rule34VideoVideoPage }} / {{ Math.ceil(filteredRule34VideoVideos.length / videosPerPage) }}
										</button>
										<button
											class="btn btn-sm btn-outline-secondary"
											:disabled="rule34VideoVideoPage >= Math.ceil(filteredRule34VideoVideos.length / videosPerPage)"
											@click="rule34VideoVideoPage++; window.scrollTo({ top: 0, behavior: 'smooth' })"
										>
											<font-awesome-icon :icon="['fas', 'chevron-right']" />
										</button>
									</div>
									<span class="text-muted small">
										Showing {{ (rule34VideoVideoPage - 1) * videosPerPage + 1 }} -
										{{ Math.min(rule34VideoVideoPage * videosPerPage, filteredRule34VideoVideos.length) }}
										of {{ filteredRule34VideoVideos.length }} videos
									</span>
								</div>
							</div>
						</div>

						<!-- Empty State -->
						<EmptyState
							v-else-if="!rule34VideoScraping"
							:icon="['fas', 'cube']"
							icon-size="4x"
							title="No Rule34Videos Yet"
							message="Use the sidebar controls to start scraping"
						/>
					</div>
					<!-- End Main Content -->

					<!-- Sidebar (Right) -->
					<div class="scraper-sidebar" :class="{ collapsed: collapsedSections.rule34video }">
						<button class="collapse-toggle-btn mb-3 w-100" @click="toggleCollapse('rule34video')">
							<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
							Hide Panel
						</button>

						<!-- Auto-Update Panel -->
						<AutoUpdatePanel scraper-type="rule34video" />

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
									<StatCard :value="rule34VideoStats.total_videos || 0" label="Videos" :icon="['fas', 'video']" icon-class="videos" />
									<StatCard :value="rule34VideoStats.avg_rating || 0" label="Avg Rating" :icon="['fas', 'star']" icon-class="keywords" />
									<StatCard :value="rule34VideoActiveJob ? 1 : 0" label="Active" :icon="['fas', 'spinner']" icon-class="active" />
									<StatCard :value="rule34VideoActiveJob?.videosFound || 0" label="Found" :icon="['fas', 'search']" icon-class="found" />
								</div>
							</div>
						</div>

						<!-- Scraper Controls -->
						<div class="card mb-3">
							<div class="card-header">
								<h6 class="mb-0">
									<font-awesome-icon :icon="['fas', 'cube']" class="me-2 text-info" />
									Scraper
								</h6>
							</div>
							<div class="card-body">
								<div class="mb-2">
									<label class="form-label small">Sort By</label>
									<select v-model="rule34VideoSortBy" class="form-select form-select-sm">
										<option v-for="opt in rule34VideoSortOptions" :key="opt.value" :value="opt.value">
											{{ opt.label }}
										</option>
									</select>
								</div>
								<div class="mb-2">
									<label class="form-label small">Max Pages</label>
									<input v-model.number="rule34VideoMaxPages" type="number" class="form-control form-control-sm" min="1" max="9999" placeholder="0 = unlimited" />
								</div>
								<button class="btn btn-primary btn-sm w-100" @click="startRule34VideoScrape" :disabled="rule34VideoScraping">
									<font-awesome-icon :icon="['fas', rule34VideoScraping ? 'spinner' : 'search']" :spin="rule34VideoScraping" class="me-2" />
									{{ rule34VideoScraping ? 'Scraping...' : 'Start Scrape' }}
								</button>
								<div class="alert alert-info mt-2 mb-0 py-1 px-2 small">
									<font-awesome-icon :icon="['fas', 'filter']" class="me-1" />
									Filter: Straight 3D content
								</div>
							</div>
						</div>
					</div>
					<!-- End Sidebar -->
				</div>
				<!-- End scraper-layout -->

				<!-- Sidebar Toggle Button (shown when collapsed) -->
				<button
					v-if="collapsedSections.rule34video"
					class="sidebar-toggle"
					@click="toggleCollapse('rule34video')"
					title="Show Controls Panel"
				>
					<font-awesome-icon :icon="['fas', 'cog']" />
				</button>
			</div>
			<!-- End Rule34Video Tab Content -->

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
						<EmptyState
							v-else
							:icon="['fas', 'video']"
							icon-size="4x"
							title="No Videos Scraped Yet"
							message="Use the sidebar controls to scrape videos from any site"
						/>
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
							<!-- Search and Filters (Always Visible) -->
							<div class="filter-bar mb-3">
								<div class="row g-2 align-items-end">
									<div class="col-md-4">
										<input v-model="videoSearch[tab.id]" type="text" class="form-control form-control-sm" placeholder="Search videos..." />
									</div>
									<div class="col-md-3">
										<select v-model="videoSort[tab.id]" class="form-select form-select-sm">
											<option value="newest">Newest First</option>
											<option value="oldest">Oldest First</option>
											<option value="views-desc">Most Views</option>
											<option value="duration-desc">Longest</option>
											<option value="title-asc">Title (A-Z)</option>
											<option value="likes-desc">Most Likes</option>
										</select>
									</div>
									<div class="col-md-2">
										<select v-model="videoStatusFilter[tab.id]" class="form-select form-select-sm">
											<option value="">All Status</option>
											<option value="discovered">Discovered</option>
											<option value="scraped">Scraped</option>
											<option value="downloaded">Downloaded</option>
										</select>
									</div>
									<div class="col-md-3">
										<button class="btn btn-sm btn-outline-secondary w-100" @click="toggleVideoFilters(tab.id)">
											<font-awesome-icon :icon="['fas', 'filter']" class="me-1" />
											More Filters
										</button>
									</div>
								</div>
								<!-- Advanced Filters (Collapsible) -->
								<div v-if="showVideoFilters[tab.id]" class="mt-2 pt-2 border-top border-secondary">
									<div class="row g-2">
										<div class="col-md-3">
											<select v-model="videoResolutionFilter[tab.id]" class="form-select form-select-sm">
												<option value="">All Resolutions</option>
												<option value="4K">4K / 2160p</option>
												<option value="1080p">1080p</option>
												<option value="720p">720p</option>
												<option value="480p">480p</option>
											</select>
										</div>
										<div class="col-md-3">
											<select v-model="videoDurationFilter[tab.id]" class="form-select form-select-sm">
												<option value="">All Durations</option>
												<option value="0-60">Under 1 min</option>
												<option value="60-300">1-5 min</option>
												<option value="300-600">5-10 min</option>
												<option value="600-1800">10-30 min</option>
												<option value="1800-">Over 30 min</option>
											</select>
										</div>
										<div class="col-md-3">
											<select v-model="videoThumbnailFilter[tab.id]" class="form-select form-select-sm">
												<option value="">All</option>
												<option value="yes">Has Thumbnail</option>
												<option value="no">No Thumbnail</option>
											</select>
										</div>
										<div class="col-md-3">
											<button class="btn btn-sm btn-outline-warning w-100" @click="resetVideoFilters(tab.id)">
												<font-awesome-icon :icon="['fas', 'redo']" class="me-1" />
												Reset
											</button>
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
											<strong>{{ customTabJobs[tab.id].pagesCompleted }}</strong> pages |
											<strong>{{ customTabJobs[tab.id].videosFound }}</strong> found |
											<strong>{{ customTabJobs[tab.id].videosSaved }}</strong> saved
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
										{{ tab.name }} Videos ({{ filteredTabVideos(tab.id).length }} / {{ customTabVideos[tab.id].length }})
									</h6>
								</div>

								<div class="video-grid">
									<div
										v-for="video in paginatedTabVideos(tab.id)"
										:key="video.id"
										class="video-card"
										@mouseenter="startThumbnailPreview(video)"
										@mouseleave="stopThumbnailPreview(video)"
									>
										<!-- Thumbnail -->
										<div class="video-thumbnail">
											<img
												v-if="video.thumbnail_url"
												:src="getCurrentThumbnail(video)"
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
													<span v-if="video.rating > 0" class="text-success small">
														<font-awesome-icon :icon="['fas', 'thumbs-up']" class="me-1" />
														{{ video.rating }}%
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
												<button
													v-if="jdAvailable && video.source_url"
													@click="sendToJDownloader(video)"
													class="btn btn-sm btn-warning"
													:disabled="jdSending[video.id]"
													title="Send to JDownloader"
												>
													<font-awesome-icon :icon="['fas', jdSending[video.id] ? 'spinner' : 'download']" :spin="jdSending[video.id]" />
												</button>
											</div>
										</div>
									</div>
								</div>

								<!-- Pagination -->
								<div v-if="filteredTabVideos(tab.id).length > videosPerPage" class="pagination-controls mt-3">
									<div class="d-flex justify-content-center align-items-center flex-column gap-2">
										<div class="btn-group">
											<button class="btn btn-sm btn-outline-secondary" :disabled="(videoPage[tab.id] || 1) === 1" @click="changeVideoPage(tab.id, -1)">
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

							<!-- Empty State -->
							<EmptyState
								v-else
								:icon="['fas', 'video']"
								icon-size="4x"
								:title="`No ${tab.name} Videos Yet`"
								message="Use the sidebar to configure and start scraping"
							/>
						</div>
						<!-- End Main Content -->

						<!-- Sidebar (Right) -->
						<div class="scraper-sidebar" :class="{ collapsed: collapsedSections[`tab-${tab.id}`] }">
							<!-- Collapse Toggle -->
							<button class="btn btn-sm btn-outline-secondary w-100 mb-3" @click="toggleCollapse(`tab-${tab.id}`)">
								<font-awesome-icon :icon="['fas', 'chevron-right']" class="me-2" />
								Hide Panel
							</button>

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
										<button class="btn btn-outline-secondary btn-sm w-100" @click="openEditTabModal(tab)">
											<font-awesome-icon :icon="['fas', 'edit']" class="me-2" />
											Edit Tab
										</button>
									</div>
								</div>
							</div>
						</div>
						<!-- End Sidebar -->
					</div>
					<!-- End scraper-layout -->

					<!-- Sidebar Toggle Button (shown when collapsed) -->
					<button
						v-if="collapsedSections[`tab-${tab.id}`]"
						class="sidebar-toggle"
						@click="toggleCollapse(`tab-${tab.id}`)"
						title="Show Controls Panel"
					>
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
import { ref, shallowRef, onMounted, onBeforeUnmount, computed, getCurrentInstance, watch } from 'vue'
import { useRouter } from 'vue-router'
import { DeleteConfirmationModal, LoadingState, EmptyState, StatCard, SearchBox } from '@/components/shared'
import AutoUpdatePanel from '@/components/scraper/AutoUpdatePanel.vue'
import { useFormatters } from '@/composables/useFormatters'
import { jdownloaderAPI } from '@/services/api'

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

const sendToJDownloader = async (video, packageName = '') => {
	if (!video?.source_url && !video?.embed_url) {
		toast.warning('No URL available for this video')
		return
	}

	const url = video.source_url || video.embed_url
	const key = video.id || url
	jdSending.value[key] = true

	try {
		const result = await jdownloaderAPI.sendLink(
			url,
			packageName || video.title || '',
			video.id ? parseInt(video.id) : null,
			false
		)

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

// State
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
const videoThumbnailFilter = ref({}) // { tabId: 'yes'|'no'|'' }
const showVideoFilters = ref({}) // { tabId: boolean }
const videosPerPage = 24

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
	const start = (pornHubVideoPage.value - 1) * videosPerPage
	const end = start + videosPerPage
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
	const start = (luxureTVVideoPage.value - 1) * videosPerPage
	const end = start + videosPerPage
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
	const start = (rule34VideoVideoPage.value - 1) * videosPerPage
	const end = start + videosPerPage
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
			loginHelperInstructions.value = [
				'Log in to your Simpcity account',
				'Press F12 to open Developer Tools',
				'Go to Application tab → Cookies → simpcity.is',
				'Select all cookies (Ctrl+A) and copy their values',
				'Paste them in the text area below',
			]
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
		let url = searchQuery.value
			? `http://localhost:8080/api/v1/scraper/threads/search?q=${encodeURIComponent(searchQuery.value)}&`
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
	window.open(url, '_blank')
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
		// Open embed URL in a new window/tab for now
		// Could be enhanced to show in a modal within the app
		window.open(embedUrl, '_blank', 'width=900,height=600')
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
	const hasSearchCriteria =
		search.categories.length > 0 ||
		search.title.trim() ||
		search.description.trim() ||
		search.pornstars.trim() ||
		search.production.trim() ||
		search.tags.trim() ||
		search.resolutions.length > 0 ||
		search.fileHosts.length > 0

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
		window.open(item.url, '_blank')
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
	videoPage.value[tabId] = 1
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

const loadVideosForTab = async (tabId) => {
	customTabLoading.value[tabId] = true
	try {
		const tab = scraperTabs.value.find((t) => t.id === tabId)
		if (!tab) return

		// Get the source from the tab's baseUrl
		const tabSource = new URL(tab.baseUrl).host

		// Pass source parameter to filter on backend for better performance
		const response = await fetch(`http://localhost:8080/api/v1/scraper/videos?source=${encodeURIComponent(tabSource)}`)
		const data = await response.json()

		if (data.success && data.data) {
			customTabVideos.value[tabId] = data.data
			console.log(`Loaded ${data.data.length} videos for tab ${tabId} (source: ${tabSource})`)
		}
	} catch (error) {
		console.error('Error loading videos for tab:', error)
	} finally {
		customTabLoading.value[tabId] = false
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
		videos = videos.filter(
			(v) => v.title.toLowerCase().includes(searchTerm) || (v.uploader || '').toLowerCase().includes(searchTerm) || (v.author || '').toLowerCase().includes(searchTerm)
		)
	}

	// Filter by status
	if (statusFilter) {
		videos = videos.filter((v) => v.status === statusFilter)
	}

	// Filter by resolution
	if (resolutionFilter) {
		videos = videos.filter((v) => {
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
		videos = videos.filter((v) => {
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
		videos = videos.filter((v) => v.thumbnail_url && v.thumbnail_url.trim() !== '')
	} else if (thumbnailFilter === 'no') {
		videos = videos.filter((v) => !v.thumbnail_url || v.thumbnail_url.trim() === '')
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
		window.scrollTo({ top: 0, behavior: 'smooth' })
	}
}

const playScrapedVideo = (video) => {
	const videoUrl = video.url || video.source_url
	// Sites that block iframe embedding completely - open in new tab
	// Note: PornHub now supports embedding via /embed/{viewkey}
	if (videoUrl && (videoUrl.includes('xvideos.com') || videoUrl.includes('xhamster.com'))) {
		window.open(videoUrl, '_blank')
		return
	}
	// Navigate to video player page with scraped video ID
	// PornHub and LuxureTV videos will use their embed URLs in the player
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

<style scoped src="@/styles/pages/scraper_page.css"></style>
