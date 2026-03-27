<!-- eslint-disable no-undef -->
<template>
	<div class="tab-content">
		<div class="scraper-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed, 'filter-panel-collapsed': filterPanelCollapsed }">
			<!-- Left Filter Panel -->
			<div class="scraper-filter-panel" :class="{ 'is-collapsed': filterPanelCollapsed }">
				<div class="sfp-inner">
					<div class="sfp-header">
						<div class="sfp-title">
							<font-awesome-icon :icon="['fas', 'filter']" class="me-2" />
							Filters
							<span v-if="activeFilterCount > 0" class="sfp-active-badge ms-2">{{ activeFilterCount }}</span>
						</div>
						<button class="sfp-close-btn" @click="filterPanelCollapsed = true">
							<font-awesome-icon :icon="['fas', 'times']" />
						</button>
					</div>
					<button v-if="activeFilterCount > 0" class="sfp-clear-all mt-2 mb-3" @click="clearAllFilters">
						<font-awesome-icon :icon="['fas', 'times-circle']" class="me-1" />
						Clear All Filters
					</button>

					<!-- Sort -->
					<div class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'sort']" class="me-2" />
							Sort By
						</div>
						<select v-model="sortBy" @change="handleSortChange" class="sfp-select form-select form-select-sm">
							<option value="date_desc">Date (Newest First)</option>
							<option value="date_asc">Date (Oldest First)</option>
							<option value="views_desc">Views (High to Low)</option>
							<option value="views_asc">Views (Low to High)</option>
							<option value="replies_desc">Replies (Most First)</option>
							<option value="replies_asc">Replies (Least First)</option>
							<option value="downloads_desc">Downloads (Most First)</option>
							<option value="downloads_asc">Downloads (Least First)</option>
							<option value="title_asc">Title (A-Z)</option>
							<option value="title_desc">Title (Z-A)</option>
						</select>
					</div>

					<!-- Watched toggle -->
					<div class="sfp-section">
						<div
							class="sfp-toggle-row"
							:class="{ active: filterContent === 'watched' }"
							@click="filterContent = filterContent === 'watched' ? '' : 'watched'; handleFilterChange()"
						>
							<div class="sfp-toggle-icon">
								<font-awesome-icon :icon="['fas', 'bell']" />
							</div>
							<span class="sfp-toggle-label">Watched Only</span>
							<div class="sfp-toggle-switch" :class="{ on: filterContent === 'watched' }">
								<div class="sfp-toggle-knob"></div>
							</div>
						</div>
					</div>

					<!-- Performers in DB toggle -->
					<div class="sfp-section">
						<div
							class="sfp-toggle-row"
							:class="{ active: filterContent === 'performer_in_db' }"
							@click="filterContent = filterContent === 'performer_in_db' ? '' : 'performer_in_db'; handleFilterChange()"
						>
							<div class="sfp-toggle-icon">
								<font-awesome-icon :icon="['fas', 'user-check']" />
							</div>
							<span class="sfp-toggle-label">Performers in Library</span>
							<div class="sfp-toggle-switch" :class="{ on: filterContent === 'performer_in_db' }">
								<div class="sfp-toggle-knob"></div>
							</div>
						</div>
					</div>

					<!-- Content Filter -->
					<div class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
							Download Links
						</div>
						<select v-model="filterDownloads" @change="handleFilterChange" class="sfp-select form-select form-select-sm">
							<option value="">All Threads</option>
							<option value="has_downloads">Has Downloads</option>
							<option value="no_downloads">No Downloads</option>
						</select>
					</div>

					<!-- Provider Filter -->
					<div class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'server']" class="me-2" />
							Provider
						</div>
						<select v-model="filterProvider" @change="handleFilterChange" class="sfp-select form-select form-select-sm">
							<option value="">All Providers</option>
							<option v-for="(count, provider) in stats.provider_breakdown" :key="provider" :value="provider">{{ provider }} ({{ count }})</option>
						</select>
					</div>

					<!-- Author Filter -->
					<div class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'user']" class="me-2" />
							Author
						</div>
						<input
							v-model="filterAuthor"
							@input="handleFilterChange"
							type="text"
							class="sfp-select form-control form-control-sm"
							placeholder="Filter by author..."
						/>
					</div>

					<!-- Category Filter -->
					<div class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'folder']" class="me-2" />
							Category
						</div>
						<select v-model="filterCategory" @change="handleFilterChange" class="sfp-select form-select form-select-sm">
							<option value="">All Categories</option>
							<option v-for="(count, cat) in stats.category_breakdown" :key="cat" :value="cat">{{ cat }} ({{ count }})</option>
						</select>
					</div>
				</div>
			</div>

			<!-- Filter Panel Reopen Button -->
			<button v-if="filterPanelCollapsed" class="filter-panel-toggle" @click="filterPanelCollapsed = false" title="Show Filters">
				<font-awesome-icon :icon="['fas', 'filter']" />
				<span v-if="activeFilterCount > 0" class="filter-panel-toggle-badge">{{ activeFilterCount }}</span>
			</button>

			<!-- Main Content -->
			<div class="scraper-main">
				<!-- Top Bar -->
				<div class="d-flex align-items-center gap-2 mb-3">
					<button class="btn btn-sm btn-outline-secondary" @click="filterPanelCollapsed = !filterPanelCollapsed">
						<font-awesome-icon :icon="['fas', 'filter']" class="me-1" />
						Filters
						<span v-if="activeFilterCount > 0" class="badge bg-primary ms-1">{{ activeFilterCount }}</span>
					</button>
					<div class="flex-grow-1">
						<SearchBox v-model="searchQuery" placeholder="Search threads by title or author..." />
					</div>
					<div class="d-flex gap-1 flex-shrink-0">
						<button class="btn btn-sm" :class="viewMode === 'grid' ? 'btn-primary' : 'btn-outline-secondary'" @click="viewMode = 'grid'" title="Grid View">
							<font-awesome-icon :icon="['fas', 'th']" />
						</button>
						<button class="btn btn-sm" :class="viewMode === 'list' ? 'btn-primary' : 'btn-outline-secondary'" @click="viewMode = 'list'" title="List View">
							<font-awesome-icon :icon="['fas', 'list']" />
						</button>
						<button class="btn btn-sm" :class="viewMode === 'compact' ? 'btn-primary' : 'btn-outline-secondary'" @click="viewMode = 'compact'" title="Compact View">
							<font-awesome-icon :icon="['fas', 'bars']" />
						</button>
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
					<div v-for="thread in threads" :key="thread.id" class="thread-card" :class="{ selected: isThreadSelected(thread.id), watched: thread.is_watched }" @click="openThread(thread)">
						<div class="thread-actions-overlay">
							<button class="watch-btn" :class="{ active: thread.is_watched }" @click.stop="toggleThreadWatch(thread)" :title="thread.is_watched ? 'Stop watching' : 'Watch for updates'">
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
							<img :src="thread.metadata.thumbnail_url" :data-fallback-urls="JSON.stringify(thread.metadata.thumbnail_urls || [])" alt="Thread thumbnail" @error="handleThumbnailError" />
						</div>
						<div v-else class="thread-thumbnail placeholder">
							<font-awesome-icon :icon="['fas', 'image']" size="3x" />
						</div>
						<!-- Verification Progress Bar -->
						<div v-if="verifyingThreads[thread.id]" class="thread-verify-progress">
							<div class="verify-progress-bar">
								<div class="verify-progress-fill" :style="{ width: verifyingThreads[thread.id].progress + '%' }"></div>
							</div>
							<span class="verify-progress-text">{{ verifyingThreads[thread.id].progress }}%</span>
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
								<span v-if="threadLinkStats[thread.id]" class="stat-badge" :class="getThreadLinkHealthClass(threadLinkStats[thread.id])" :title="getThreadLinkHealthTitle(threadLinkStats[thread.id])">
									<font-awesome-icon :icon="['fas', 'link']" class="me-1" />
									{{ threadLinkStats[thread.id].working }}/{{ threadLinkStats[thread.id].total }}
								</span>
							</div>
							<!-- Last Verified Timer -->
							<div v-if="threadLinkStats[thread.id]?.last_checked" class="thread-last-verified">
								<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
								Verified {{ timeAgo(threadLinkStats[thread.id].last_checked) }}
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
							<button class="btn btn-sm" :class="thread.is_watched ? 'btn-warning' : 'btn-outline-warning'" @click.stop="toggleThreadWatch(thread)" :title="thread.is_watched ? 'Stop watching' : 'Watch for updates'">
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
			<div class="scraper-sidebar" :class="{ collapsed: sidebarCollapsed }">
				<button class="collapse-toggle-btn mb-3 w-100" @click="sidebarCollapsed = !sidebarCollapsed">
					<font-awesome-icon :icon="['fas', 'times']" class="me-2" />
					Hide Panel
				</button>
				<!-- Site Status -->
				<div class="card mb-3">
					<div class="card-body py-2 px-3 d-flex align-items-center justify-content-between">
						<div class="d-flex align-items-center gap-2">
							<span
								style="display:inline-block;width:10px;height:10px;border-radius:50%;flex-shrink:0"
								:style="{ background: siteStatus === 'online' ? '#28a745' : siteStatus === 'offline' ? '#dc3545' : siteStatus === 'checking' ? '#ffc107' : '#6c757d' }"
							></span>
							<span class="small">
								<strong>simpcity.su</strong>
								<span v-if="siteStatus === 'online'" class="text-success ms-1">Online</span>
								<span v-else-if="siteStatus === 'offline'" class="text-danger ms-1">Offline</span>
								<span v-else-if="siteStatus === 'checking'" class="text-warning ms-1">Checking...</span>
								<span v-else class="text-muted ms-1">Unknown</span>
							</span>
						</div>
						<div class="d-flex align-items-center gap-2">
							<span v-if="siteStatus === 'online' && siteLatency" class="text-muted small">{{ siteLatency }}ms</span>
							<button class="btn btn-sm btn-outline-secondary py-0 px-2" @click="checkSiteHealth" :disabled="siteStatus === 'checking'">
								<font-awesome-icon :icon="['fas', 'sync']" :spin="siteStatus === 'checking'" style="font-size:0.7rem" />
							</button>
						</div>
					</div>
					<div v-if="siteStatus === 'offline'" class="px-3 pb-2">
						<small class="text-danger">
							<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-1" />
							Site unreachable — scraping will likely fail
						</small>
					</div>
				</div>
				<!-- Auto-Update Panel -->
				<AutoUpdatePanel scraper-type="simpcity" endpoint-type="urls" />
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
					Auto-updates <strong>watched threads</strong> automatically, plus any forum URLs configured in Auto-Update settings.
				</div>
			</div>
			<!-- End Sidebar -->
		</div>
		<!-- End scraper-layout -->
		<!-- Sidebar Toggle Button (shown when collapsed) -->
		<button v-if="sidebarCollapsed" class="sidebar-toggle" @click="sidebarCollapsed = !sidebarCollapsed" title="Show Controls Panel">
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
							<p class="text-muted small mb-3">Click the button below to open Simpcity in Firefox. Log in, then follow the instructions to copy your cookies.</p>
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
						<textarea v-model="authCookie" class="form-control" rows="4" placeholder="Paste cookie table from DevTools (Ctrl+A in cookie list) or cookie header string..."></textarea>
						<small class="form-text text-muted"> Paste the cookie table directly from DevTools (F12 → Application → Cookies → simpcity.is → Ctrl+A → Ctrl+C) </small>
					</div>
					<div class="alert alert-info small">
						<font-awesome-icon :icon="['fas', 'lightbulb']" class="me-2" />
						<strong>Tip:</strong> In Chrome DevTools, click on any cookie, press Ctrl+A to select all, then Ctrl+C to copy. The app will automatically parse the table format.
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
						<div class="detail-row"><strong>Last Scraped:</strong> {{ formatDateTime(selectedThread.last_scraped_at) }}</div>
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
							<button v-for="forum in simpcityForums" :key="forum.url" class="keyword-tag" :class="{ active: forumURL === forum.url }" @click="forumURL = forum.url">
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
	<!-- Delete Selected Confirmation Modal -->
	<DeleteConfirmationModal :visible="showDeleteConfirmModal" title="Confirm Deletion" message="Are you sure you want to delete" :itemName="`${selectedCount} selected thread(s)`" warningMessage="This action cannot be undone. All posts and download links associated with these threads will also be deleted." :confirmText="`Delete ${selectedCount} Thread(s)`" :icon="['fas', 'trash']" @confirm="confirmDeleteSelected" @cancel="showDeleteConfirmModal = false" />
	<!-- Delete All Confirmation Modal -->
	<DeleteConfirmationModal :visible="showDeleteAllConfirmModal" title="Confirm Delete All" message="Are you sure you want to delete" :itemName="`ALL ${totalThreads} scraped threads`" warningMessage="DANGER: This will permanently delete ALL scraped threads, posts, and download links from the database. This action cannot be undone!" confirmText="Yes, Delete Everything" :icon="['fas', 'trash-alt']" @confirm="confirmDeleteAll" @cancel="showDeleteAllConfirmModal = false" />
</template>

<script setup>
import { ref, shallowRef, watch, onMounted, onBeforeUnmount, computed, getCurrentInstance } from 'vue'
import { useFormatters } from '@/composables/useFormatters'
import { useOpenURL } from '@/composables/useOpenURL'
import { useScraperSidebar } from '@/composables/useScraperSidebar'
import { useDebouncedRef } from '@/composables/useDebouncedRef'
import { DeleteConfirmationModal, LoadingState, EmptyState, StatCard, SearchBox } from '@/components/shared'
import AutoUpdatePanel from '@/components/scraper/AutoUpdatePanel.vue'
import { scrapersAPI, jdownloaderAPI, siteHealthAPI } from '@/services/api'
import api from '@/services/api'
import websocketService from '@/services/websocket'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const { formatDateTime, timeAgo } = useFormatters()
const { openInFirefox } = useOpenURL()
const { sidebarCollapsed } = useScraperSidebar()

// ── State ─────────────────────────────────────────────────────────────────────
const loading = ref(false)
const stats = ref({})
const threads = shallowRef([])
const threadLinkStats = ref({})
const verifyingThreads = ref({})
const searchQuery          = ref('')
const debouncedSearchQuery = useDebouncedRef(searchQuery, 350)
const viewMode = ref('grid')
const currentPage = ref(1)
const totalPages = ref(1)
const totalThreads = ref(0)
const limit = 20

const sortBy = ref('date_desc')
const filterProvider = ref('')
const filterContent = ref('')
const filterDownloads = ref('')
const filterAuthor = ref('')
const filterCategory = ref('')
const filterPanelCollapsed = ref(false)

// ── Site Health ────────────────────────────────────────────────────────────────
const siteStatus = ref('unknown') // 'online' | 'offline' | 'unknown' | 'checking'
const siteLatency = ref(0)

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
const bulkOperationInProgress = ref(false)

// ── Computed ──────────────────────────────────────────────────────────────────
const visiblePages = computed(() => {
	const pages = []
	const start = Math.max(1, currentPage.value - 2)
	const end = Math.min(totalPages.value, currentPage.value + 2)
	for (let i = start; i <= end; i++) {
		pages.push(i)
	}
	return pages
})

const selectedCount = computed(() => selectedThreadIds.value.size)

const allSelectedOnPage = computed(() => {
	if (threads.value.length === 0) return false
	return threads.value.every((thread) => selectedThreadIds.value.has(thread.id))
})

const activeFilterCount = computed(() => {
	let n = 0
	if (sortBy.value !== 'date_desc') n++
	if (filterProvider.value) n++
	if (filterContent.value === 'watched' || filterContent.value === 'performer_in_db') n++
	if (filterDownloads.value) n++
	if (filterAuthor.value) n++
	if (filterCategory.value) n++
	return n
})

const clearAllFilters = () => {
	sortBy.value = 'date_desc'
	filterProvider.value = ''
	filterContent.value = ''
	filterDownloads.value = ''
	filterAuthor.value = ''
	filterCategory.value = ''
	currentPage.value = 1
	loadThreads()
}

// ── Auth ──────────────────────────────────────────────────────────────────────
const checkAuthStatus = async () => {
	try {
		const data = await scrapersAPI.getSession()
		if (data.success && data.data) {
			sessionCookieSet.value = data.data.is_set
		}
	} catch (error) {
		console.error('Error checking auth status:', error)
	}
}

const parseCookieTable = (input) => {
	if (input.includes('=') && !input.includes('\t') && input.split(';').length > 1) {
		return input.trim()
	}
	const lines = input.trim().split('\n')
	const cookies = []
	for (const line of lines) {
		const parts = line.split('\t')
		if (parts.length >= 2) {
			const name = parts[0].trim()
			const value = parts[1].trim()
			if (name && value && name !== 'Name' && !name.startsWith('__')) {
				cookies.push(`${name}=${value}`)
			} else if (name && value && name.startsWith('__')) {
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
	const parsedCookie = parseCookieTable(authCookie.value)
	if (!parsedCookie) {
		toast.error('Could not parse cookie data. Please try copying again.')
		return
	}
	console.log('Parsed cookie:', parsedCookie.substring(0, 100) + '...')
	try {
		const data = await scrapersAPI.setSession({ cookie: parsedCookie })
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
		const data = await scrapersAPI.openLoginHelper('simpcity')
		if (data.success) {
			toast.success('Login page opened in Firefox')
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

// ── Site Health ────────────────────────────────────────────────────────────────
const checkSiteHealth = async () => {
	siteStatus.value = 'checking'
	try {
		const data = await siteHealthAPI.check('https://simpcity.su')
		if (data.data?.online) {
			siteStatus.value = 'online'
			siteLatency.value = data.data.latency_ms
		} else {
			siteStatus.value = 'offline'
		}
	} catch {
		siteStatus.value = 'offline'
	}
}

// ── Data Loading ──────────────────────────────────────────────────────────────
const loadStats = async () => {
	try {
		const data = await scrapersAPI.getStats()
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
		const params = { page: currentPage.value, limit }
		if (debouncedSearchQuery.value) params.q        = debouncedSearchQuery.value
		if (sortBy.value)               params.sort     = sortBy.value
		if (filterProvider.value)       params.provider = filterProvider.value
		// Combine watched toggle + download filter into a single `filter` param
		if (filterContent.value === 'watched')            params.filter = 'watched'
		else if (filterContent.value === 'performer_in_db') params.filter = 'performer_in_db'
		else if (filterDownloads.value)               params.filter = filterDownloads.value
		if (filterAuthor.value)         params.author   = filterAuthor.value
		if (filterCategory.value)       params.category = filterCategory.value

		const endpoint = debouncedSearchQuery.value ? '/scraper/threads/search' : '/scraper/threads'
		const data = await api.get(endpoint, { params })

		if (data.success) {
			threads.value = data.data || []
			totalPages.value = data.pagination.total_pages
			totalThreads.value = data.pagination.total
			threads.value.forEach((thread) => fetchThreadLinkStats(thread.id))
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

watch(debouncedSearchQuery, searchThreads)

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

// ── Scraping ──────────────────────────────────────────────────────────────────
const startScrape = async () => {
	if (!scrapeURL.value) {
		toast.warning('Please enter a URL')
		return
	}
	try {
		const data = await scrapersAPI.startScrape({ url: scrapeURL.value })
		if (data.success) {
			toast.success('Scraping started! Check the Tasks page for progress.')
			showScrapeModal.value = false
			scrapeURL.value = ''
			setTimeout(() => { loadThreads(); loadStats() }, 2000)
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
		const data = await scrapersAPI.rescrapeThread(thread.id)
		if (data.success) {
			toast.success('Rescaping started! Check the Tasks page for progress.')
			selectedThread.value = null
			setTimeout(() => { loadThreads(); loadStats() }, 2000)
		} else {
			toast.error(data.message || 'Failed to start rescraping')
		}
	} catch (error) {
		console.error('Error rescraping thread:', error)
		toast.error('Failed to start rescraping')
	}
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
		const data = await scrapersAPI.scrapeForumAll({ url: forumURL.value })
		if (data.success) {
			toast.success('Forum scraping started! This will take a while. Check server logs for progress.')
			showForumScrapeModal.value = false
			setTimeout(() => { loadThreads(); loadStats() }, 5000)
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
		const data = await scrapersAPI.autoLinkPerformers()
		if (data.success) {
			toast.success('Auto-linking started! Check server logs for progress. Performers will be created/linked automatically.')
			setTimeout(() => { loadThreads(); loadStats() }, 3000)
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
		const data = await scrapersAPI.checkLinkStatuses()
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

// ── Thread Navigation ─────────────────────────────────────────────────────────
const openThread = (thread) => {
	window.open(`/scraper/${thread.id}`, '_blank')
}

const openThreadURL = (url) => {
	openInFirefox(url)
}

const toggleThreadWatch = async (thread) => {
	try {
		const data = await scrapersAPI.watchThread(thread.id)
		if (data.success) {
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

// ── Thread Thumbnail ──────────────────────────────────────────────────────────
const handleThumbnailError = (event) => {
	const img = event.target
	const fallbackURLs = JSON.parse(img.dataset.fallbackUrls || '[]')
	const currentIndex = fallbackURLs.indexOf(img.src)
	if (currentIndex < fallbackURLs.length - 1) {
		img.src = fallbackURLs[currentIndex + 1]
	} else {
		img.style.display = 'none'
		const placeholder = img.parentElement.nextElementSibling
		if (placeholder && placeholder.classList.contains('placeholder')) {
			placeholder.style.display = 'flex'
		}
	}
}

// ── Selection ─────────────────────────────────────────────────────────────────
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
	threads.value.forEach((thread) => selectedThreadIds.value.add(thread.id))
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const deselectAllThreads = () => {
	selectedThreadIds.value.clear()
	selectedThreadIds.value = new Set(selectedThreadIds.value)
}

const isThreadSelected = (threadId) => selectedThreadIds.value.has(threadId)

// ── Link Health ───────────────────────────────────────────────────────────────
const getThreadLinkHealthClass = (linkStats) => {
	if (!linkStats || linkStats.total === 0) return ''
	const workingPct = (linkStats.working / linkStats.total) * 100
	const deadPct = (linkStats.dead / linkStats.total) * 100
	if (deadPct > 50) return 'text-danger'
	if (deadPct > 25) return 'text-warning'
	if (workingPct > 75) return 'text-success'
	return ''
}

const getThreadLinkHealthTitle = (linkStats) => {
	if (!linkStats) return 'No link data'
	return `${linkStats.working} working, ${linkStats.dead} dead, ${linkStats.unknown} unchecked`
}

const fetchThreadLinkStats = async (threadId) => {
	try {
		const data = await scrapersAPI.getThreadLinkStats(threadId)
		if (data.success && data.data) {
			threadLinkStats.value[threadId] = {
				total:        data.data.total || 0,
				working:      data.data.working || 0,
				dead:         data.data.dead || 0,
				unknown:      data.data.unchecked || data.data.unknown || 0,
				last_checked: data.data.last_checked || null,
			}
		}
	} catch (error) {
		console.error(`Failed to fetch link stats for thread ${threadId}:`, error)
	}
}

const verifyThreadLinks = async (thread) => {
	try {
		verifyingThreads.value = { ...verifyingThreads.value, [thread.id]: { progress: 0, message: 'Starting...' } }
		const data = await scrapersAPI.verifyThreadLinks(thread.id)
		if (!data.success) {
			const updated = { ...verifyingThreads.value }
			delete updated[thread.id]
			verifyingThreads.value = updated
		}
	} catch (error) {
		console.error('Failed to verify thread links:', error)
		toast.error('Failed to start link verification')
		const updated = { ...verifyingThreads.value }
		delete updated[thread.id]
		verifyingThreads.value = updated
	}
}

// ── Bulk Operations ───────────────────────────────────────────────────────────
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
		for (const threadId of threadIds) {
			try {
				const data = await scrapersAPI.verifyThreadLinks(threadId)
				if (data.success) {
					completed++
					toast.success(`Verified thread ${completed}/${threadIds.length}`, { duration: 2000 })
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
		const jdStatusData = await jdownloaderAPI.getStatus()
		if (!jdStatusData.success || !jdStatusData.data.available) {
			toast.error('JDownloader is not running! Please start JDownloader first.')
			return
		}
		toast.info(`Sending ${threadIds.length} threads to JDownloader...`)
		for (const threadId of threadIds) {
			try {
				const data = await jdownloaderAPI.sendThreadLinks(threadId, { only_active: true, only_pending: true, auto_start: false })
				if (data.success) {
					completed++
					totalLinksSent += data.data.links_sent || 0
					toast.success(`Sent thread ${completed}/${threadIds.length} (${data.data.links_sent} links)`, { duration: 2000 })
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

const deleteSelectedThreads = () => {
	if (selectedThreadIds.value.size === 0) {
		toast.warning('No threads selected')
		return
	}
	showDeleteConfirmModal.value = true
}

const confirmDeleteSelected = async () => {
	const threadIds = Array.from(selectedThreadIds.value)
	try {
		const data = await scrapersAPI.deleteThreads({ thread_ids: threadIds })
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

const deleteAllThreads = () => {
	showDeleteAllConfirmModal.value = true
}

const confirmDeleteAll = async () => {
	try {
		const data = await scrapersAPI.deleteAllThreads()
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

// ── WebSocket ─────────────────────────────────────────────────────────────────
let unsubVerification = null
let unsubActivity = null

// ── Lifecycle ─────────────────────────────────────────────────────────────────
onMounted(() => {
	checkAuthStatus()
	checkSiteHealth()
	loadStats()
	loadThreads()

	unsubVerification = websocketService.on('link_verification_complete', (data) => {
		const tid = data.thread_id
		fetchThreadLinkStats(tid)
		const updated = { ...verifyingThreads.value }
		delete updated[tid]
		verifyingThreads.value = updated
		toast.success(`Verification complete: ${data.active} active, ${data.dead} dead, ${data.expired} expired (${data.total} total)`)
	})

	unsubActivity = websocketService.on('activity_update', (activity) => {
		if (activity.task_type === 'link_verification' && activity.details?.thread_id) {
			const threadId = activity.details.thread_id
			if (activity.status === 'completed' || activity.status === 'failed') {
				const updated = { ...verifyingThreads.value }
				delete updated[threadId]
				verifyingThreads.value = updated
				if (activity.status === 'completed') {
					fetchThreadLinkStats(threadId)
				}
			} else if (activity.status === 'in_progress' || activity.status === 'running') {
				verifyingThreads.value = {
					...verifyingThreads.value,
					[threadId]: {
						progress: activity.progress || 0,
						message:  activity.message || 'Verifying...',
					},
				}
				if (activity.progress >= 100) {
					setTimeout(() => {
						if (verifyingThreads.value[threadId]) {
							const cleaned = { ...verifyingThreads.value }
							delete cleaned[threadId]
							verifyingThreads.value = cleaned
							fetchThreadLinkStats(threadId)
						}
					}, 5000)
				}
			}
		}
	})
})

onBeforeUnmount(() => {
	if (unsubVerification) unsubVerification()
	if (unsubActivity) unsubActivity()
})
</script>
