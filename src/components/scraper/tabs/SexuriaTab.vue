<!-- eslint-disable no-undef -->
<template>
	<div class="tab-content">
		<!-- Top Bar -->
		<div class="scraper-tab-topbar">
			<button
				class="scraper-filter-toggle-btn"
				:class="{ 'is-active': !filterCollapsed }"
				@click="filterCollapsed = !filterCollapsed"
				title="Toggle filters"
			>
				<font-awesome-icon :icon="['fas', 'sliders-h']" />
				Filters
				<span v-if="activeFilterCount > 0" class="scraper-filter-toggle-badge">{{ activeFilterCount }}</span>
			</button>
			<input v-model="sexuriaItemSearch" class="scraper-tab-search" type="text" placeholder="Search Sexuria items..." />
			<div class="scraper-tab-topbar-actions">
				<span v-if="sexuriaTotalCount > 0" class="scraper-video-count">{{ sexuriaTotalCount.toLocaleString() }} items</span>
				<button class="btn btn-sm" :class="sexuriaSelectionMode ? 'btn-primary' : 'btn-outline-secondary'" @click="toggleSexuriaSelectionMode" title="Select mode">
					<font-awesome-icon :icon="['fas', sexuriaSelectionMode ? 'times' : 'check-square']" />
				</button>
				<button class="btn btn-sm btn-outline-info" @click="loadSexuriaItems" :disabled="sexuriaLoading" title="Refresh">
					<font-awesome-icon :icon="['fas', sexuriaLoading ? 'spinner' : 'sync-alt']" :spin="sexuriaLoading" />
				</button>
			</div>
		</div>

		<div class="scraper-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
			<!-- Left: Filter Panel -->
			<div class="scraper-filter-panel" :class="{ 'is-collapsed': filterCollapsed }">
				<div class="sfp-inner">
					<div class="sfp-header">
						<div class="sfp-title">
							<font-awesome-icon :icon="['fas', 'sliders-h']" />
							<span>Filters &amp; Sort</span>
							<span v-if="activeFilterCount > 0" class="sfp-active-badge">{{ activeFilterCount }}</span>
						</div>
						<button class="sfp-close-btn" @click="filterCollapsed = true" title="Hide filters">
							<font-awesome-icon :icon="['fas', 'chevron-left']" />
						</button>
					</div>
					<button v-if="activeFilterCount > 0" class="sfp-clear-all" @click="clearAllFilters">
						<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
						Clear All Filters
					</button>
					<!-- Sort -->
					<div class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'sort']" />
							Sort By
						</div>
						<select v-model="sexuriaItemSort" class="form-select form-select-sm sfp-select">
							<option value="newest">Newest First</option>
							<option value="oldest">Oldest First</option>
							<option value="title-asc">Title A–Z</option>
							<option value="title-desc">Title Z–A</option>
						</select>
					</div>
					<!-- Category -->
					<div v-if="availableSexuriaCategories.length > 0" class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'folder']" />
							Category
						</div>
						<select v-model="sexuriaCategoryFilter" class="form-select form-select-sm sfp-select">
							<option value="">All Categories</option>
							<option v-for="cat in availableSexuriaCategories" :key="cat.name" :value="cat.name">{{ cat.name }} ({{ cat.count }})</option>
						</select>
					</div>
					<!-- Host -->
					<div v-if="Object.keys(sexuriaStats.host_counts || {}).length > 0" class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'server']" />
							File Host
						</div>
						<select v-model="sexuriaHostFilter" class="form-select form-select-sm sfp-select">
							<option value="">All Hosts</option>
							<option v-for="(count, host) in sexuriaStats.host_counts" :key="host" :value="host">{{ host }} ({{ count }})</option>
						</select>
					</div>
					<!-- Tags -->
					<div v-if="availableSexuriaTags.length > 0" class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'tags']" />
							Tags
							<button v-if="sexuriaTagFilter.length > 0" class="btn btn-link btn-sm p-0 ms-auto text-muted" @click="sexuriaTagFilter = []">Clear</button>
						</div>
						<div class="sfp-multi-list">
							<label v-for="tag in availableSexuriaTags" :key="tag" class="sfp-check-item" @click.stop>
								<input type="checkbox" :checked="sexuriaTagFilter.includes(tag)" @change="toggleSexuriaTagFilter(tag)" class="me-2" />
								{{ tag }}
							</label>
						</div>
					</div>
					<!-- Pornstars -->
					<div v-if="availableSexuriaPornstars.length > 0" class="sfp-section">
						<div class="sfp-section-label">
							<font-awesome-icon :icon="['fas', 'star']" />
							Pornstars
							<button v-if="sexuriaPornstarFilter.length > 0" class="btn btn-link btn-sm p-0 ms-auto text-muted" @click="sexuriaPornstarFilter = []">Clear</button>
						</div>
						<div class="sfp-multi-list">
							<label v-for="star in availableSexuriaPornstars" :key="star" class="sfp-check-item" @click.stop>
								<input type="checkbox" :checked="sexuriaPornstarFilter.includes(star)" @change="toggleSexuriaPornstarFilter(star)" class="me-2" />
								{{ star }}
							</label>
						</div>
					</div>
				</div>
			</div>

			<!-- Center: Main Content -->
			<div class="scraper-main">
				<!-- Bulk Actions (when selecting) -->
				<ScraperBulkActions
					:selectionMode="sexuriaSelectionMode"
					:selectedCount="sexuriaSelectedItems.size"
					:sendingToJD="sendingToJD"
					@select-all="selectAllSexuriaItems"
					@delete-selected="deleteSelectedSexuriaItems"
					@send-to-jd="sendSelectedToJD"
				/>
				<!-- Active Job Progress -->
				<ScraperJobProgress :job="sexuriaActiveJob" label="Sexuria" />
				<!-- Loading State -->
				<LoadingState v-if="sexuriaLoading" show-text loading-text="Loading Sexuria items..." />
				<!-- Sexuria Items Grid -->
				<div v-else-if="sexuriaItems.length > 0">
					<div class="d-flex justify-content-between align-items-center mb-2">
						<h6 class="mb-0">
							<font-awesome-icon :icon="['fas', 'file-video']" class="me-2" />
							Sexuria Items ({{ sexuriaTotalCount.toLocaleString() }})
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
						<div v-for="item in paginatedSexuriaItems" :key="item.id" v-memo="[item.id, item.downloaded_at, sexuriaSelectedItems.has(item.id), sexuriaSelectionMode]" class="video-card sexuria-card" :class="{ selected: sexuriaSelectedItems.has(item.id) }" @mouseenter="handleSexuriaItemHover(item, $event)" @mouseleave="hideSexuriaDetails" @click="sexuriaSelectionMode && toggleSexuriaItemSelection(item.id)">
							<!-- Selection Checkbox -->
							<div v-if="sexuriaSelectionMode" class="video-checkbox" @click.stop="toggleSexuriaItemSelection(item.id)">
								<input type="checkbox" :checked="sexuriaSelectedItems.has(item.id)" @change.stop />
							</div>
							<!-- Thumbnail -->
							<div class="video-thumbnail">
								<img v-if="item.thumbnail_url && !sexuriaImgErrors[item.id]" :src="item.thumbnail_url" :alt="item.title" referrerpolicy="no-referrer" @error="sexuriaImgErrors[item.id] = true" />
								<div v-if="!item.thumbnail_url || sexuriaImgErrors[item.id]" class="no-thumbnail">
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
										<span v-if="sexuriaLinkStats[item.id]" class="badge small" :class="getLinkHealthBadgeClass(sexuriaLinkStats[item.id])" :title="getLinkHealthTitle(sexuriaLinkStats[item.id])">
											<font-awesome-icon :icon="['fas', 'link']" class="me-1" />
											{{ sexuriaLinkStats[item.id].total }} links
										</span>
										<span v-else class="badge bg-secondary small">
											<font-awesome-icon :icon="['fas', 'link']" class="me-1" />
											Links extracted
										</span>
									</div>
								</div>
								<!-- Matched Performers & Tags (HQPorner) -->
								<div v-if="getVideoPerformers(item).length || getVideoMatchedTags(item).length" class="mt-1">
									<div v-if="getVideoPerformers(item).length" class="scraped-performers">
										<router-link v-for="p in getVideoPerformers(item)" :key="p.name" :to="p.id > 0 ? '/performers/' + p.id : '#'" :class="['performer-chip', { matched: p.id > 0 }]" @click.prevent="p.id > 0 ? $router.push('/performers/' + p.id) : null">{{ p.name }}</router-link>
									</div>
									<div v-if="getVideoMatchedTags(item).length" class="scraped-matched-tags">
										<span v-for="t in getVideoMatchedTags(item)" :key="t.id" class="tag-chip" :style="{ backgroundColor: t.color || '#6c757d' }">
											<font-awesome-icon v-if="t.icon" :icon="['fas', t.icon]" class="me-1" />
											{{ t.name }}
										</span>
									</div>
								</div>
								<!-- Actions -->
								<div class="video-actions mt-2">
									<button @click="openSexuriaItem(item)" class="btn btn-sm btn-primary">
										<font-awesome-icon :icon="['fas', 'external-link-alt']" class="me-1" />
										Open
									</button>
									<button v-if="!item.links_extracted" @click.stop="extractSexuriaItemLinks(item)" class="btn btn-sm btn-outline-info" :disabled="extractingLinks[item.id]" title="Extract download links from this item">
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
						<div v-if="sexuriaHoverDetails.visible" class="sexuria-details-card" :style="sexuriaHoverDetails.style" @mouseenter="keepSexuriaDetailsVisible" @mouseleave="hideSexuriaDetails">
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
										<img v-for="(screenshot, idx) in sexuriaHoverDetails.data.screenshots.slice(0, 6)" :key="idx" :src="screenshot" :alt="`Screenshot ${idx + 1}`" class="screenshot-thumb" :class="{ active: sexuriaHoveredScreenshot === screenshot }" @mouseenter="sexuriaHoveredScreenshot = screenshot" @mouseleave="sexuriaHoveredScreenshot = null" @click="openScreenshot(screenshot)" @error="$event.target.style.display = 'none'" />
									</div>
									<small v-if="sexuriaHoverDetails.data.screenshots.length > 6" class="text-muted"> +{{ sexuriaHoverDetails.data.screenshots.length - 6 }} more </small>
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
										<span v-if="sexuriaHoverDetails.data.porn_stars.length > 10" class="star-tag more"> +{{ sexuriaHoverDetails.data.porn_stars.length - 10 }} more </span>
									</div>
								</div>
								<!-- No Details -->
								<div v-if="!sexuriaHoverDetails.data.screenshots?.length && !sexuriaHoverDetails.data.tags?.length && !sexuriaHoverDetails.data.porn_stars?.length" class="no-details">
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
							<div v-if="sexuriaScreenshotModal.screenshots.length > 1" class="screenshot-modal-counter">{{ sexuriaScreenshotModal.currentIndex + 1 }} / {{ sexuriaScreenshotModal.screenshots.length }}</div>
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
										<button class="btn btn-primary" @click="verifySexuriaItemLinks" :disabled="sexuriaLinksModal.verifying || sexuriaLinksModal.links.length === 0">
											<font-awesome-icon :icon="sexuriaLinksModal.verifying ? ['fas', 'spinner'] : ['fas', 'check-double']" :spin="sexuriaLinksModal.verifying" class="me-1" />
											{{ sexuriaLinksModal.verifying ? 'Verifying...' : 'Verify All Links' }}
										</button>
										<button class="btn btn-secondary" @click="closeSexuriaLinksModal">Close</button>
									</div>
								</div>
							</div>
						</div>
					</Teleport>
					<!-- Pagination -->
					<div v-if="sexuriaTotalPages > 1" class="pagination-controls p-3">
						<div class="d-flex justify-content-center align-items-center flex-column gap-2">
							<div class="btn-group">
								<button
									class="btn btn-sm btn-outline-secondary"
									:disabled="sexuriaItemPage === 1"
									@click="sexuriaItemPage--; window.scrollTo({ top: 0, behavior: 'smooth' })"
								>
									<font-awesome-icon :icon="['fas', 'chevron-left']" />
								</button>
								<button class="btn btn-sm btn-outline-secondary disabled">Page {{ sexuriaItemPage }} / {{ sexuriaTotalPages }}</button>
								<button
									class="btn btn-sm btn-outline-secondary"
									:disabled="sexuriaItemPage >= sexuriaTotalPages"
									@click="sexuriaItemPage++; window.scrollTo({ top: 0, behavior: 'smooth' })"
								>
									<font-awesome-icon :icon="['fas', 'chevron-right']" />
								</button>
							</div>
							<span class="text-muted small">
								Showing {{ (sexuriaItemPage - 1) * itemsPerPage + 1 }} -
								{{ Math.min(sexuriaItemPage * itemsPerPage, sexuriaTotalCount) }}
								of {{ sexuriaTotalCount.toLocaleString() }} items
							</span>
						</div>
					</div>
				</div>
				<!-- Empty State -->
				<EmptyState v-else-if="!sexuriaScraping" :icon="['fas', 'download']" icon-size="4x" title="No Sexuria Items Yet" message="Use the sidebar controls to start scraping" />
			</div>
			<!-- End Main Content -->
			<!-- Sidebar (Right) -->
			<div class="scraper-sidebar" :class="{ collapsed: sidebarCollapsed }">
				<button class="collapse-toggle-btn mb-3 w-100" @click="sidebarCollapsed = true">
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
							<StatCard :value="sexuriaTotalCount" label="Loaded" :icon="['fas', 'database']" icon-class="links" />
						</div>
					</div>
				</div>
				<!-- Scraper Controls -->
				<div class="card mb-3">
					<div class="card-header d-flex justify-content-between align-items-center">
						<h6 class="mb-0">
							<font-awesome-icon :icon="['fas', 'download']" class="me-2 text-info" />
							Scraper
						</h6>
						<div class="btn-group btn-group-sm">
							<button class="btn btn-sm" :class="sexuriaAdvancedMode ? 'btn-outline-secondary' : 'btn-primary'" @click="sexuriaAdvancedMode = false" title="Category mode">
								<font-awesome-icon :icon="['fas', 'folder']" />
							</button>
							<button class="btn btn-sm" :class="sexuriaAdvancedMode ? 'btn-primary' : 'btn-outline-secondary'" @click="sexuriaAdvancedMode = true" title="Advanced search">
								<font-awesome-icon :icon="['fas', 'sliders-h']" />
							</button>
						</div>
					</div>
					<div class="card-body">
						<!-- Simple Category Mode -->
						<div v-if="!sexuriaAdvancedMode">
							<div class="mb-2">
								<label class="form-label small">Category URL</label>
								<input v-model="sexuriaCategoryURL" type="text" class="form-control form-control-sm" placeholder="https://sexuria.net/movies/" @keyup.enter="startSexuriaScrape" />
							</div>
							<div class="mb-2">
								<label class="form-label small">Max Pages</label>
								<input v-model.number="sexuriaMaxPages" type="number" class="form-control form-control-sm" min="1" max="9999" placeholder="0 = unlimited" />
							</div>
							<div v-if="sexuriaCategories.length > 0" class="mb-2">
								<label class="form-label small">Quick Categories</label>
								<div class="keyword-tags">
									<button v-for="cat in sexuriaCategories" :key="cat.url" class="keyword-tag" :class="{ active: sexuriaCategoryURL === cat.url }" @click="sexuriaCategoryURL = cat.url">
										{{ cat.name }}
									</button>
								</div>
							</div>
							<button class="btn btn-primary btn-sm w-100" @click="startSexuriaScrape" :disabled="!sexuriaCategoryURL || sexuriaScraping">
								<font-awesome-icon :icon="['fas', sexuriaScraping ? 'spinner' : 'search']" :spin="sexuriaScraping" class="me-2" />
								{{ sexuriaScraping ? 'Scraping...' : 'Start Scrape' }}
							</button>
						</div>
						<!-- Advanced Search Mode -->
						<div v-else>
							<div v-if="sexuriaSearchOptions.categories.length > 0" class="mb-2">
								<label class="form-label small">Categories</label>
								<div class="keyword-tags">
									<button v-for="cat in sexuriaSearchOptions.categories" :key="cat.id" class="keyword-tag" :class="{ active: sexuriaAdvancedSearch.categories.includes(cat.id) }" @click="toggleSexuriaSearchCategory(cat.id)">
										{{ cat.name }}
									</button>
								</div>
							</div>
							<div class="mb-2">
								<label class="form-label small">Title</label>
								<input v-model="sexuriaAdvancedSearch.title" type="text" class="form-control form-control-sm" placeholder="Search by title..." />
							</div>
							<div class="mb-2">
								<label class="form-label small">Pornstars</label>
								<input v-model="sexuriaAdvancedSearch.pornstars" type="text" class="form-control form-control-sm" placeholder="e.g. Angela White" />
							</div>
							<div class="mb-2">
								<label class="form-label small">Production / Studio</label>
								<input v-model="sexuriaAdvancedSearch.production" type="text" class="form-control form-control-sm" placeholder="e.g. Brazzers" />
							</div>
							<div class="mb-2">
								<label class="form-label small">Tags</label>
								<input v-model="sexuriaAdvancedSearch.tags" type="text" class="form-control form-control-sm" placeholder="e.g. anal, threesome" />
							</div>
							<div v-if="sexuriaSearchOptions.resolutions.length > 0" class="mb-2">
								<label class="form-label small">Resolutions</label>
								<div class="keyword-tags">
									<button v-for="res in sexuriaSearchOptions.resolutions" :key="res" class="keyword-tag" :class="{ active: sexuriaAdvancedSearch.resolutions.includes(res) }" @click="toggleSexuriaSearchResolution(res)">
										{{ res }}
									</button>
								</div>
							</div>
							<div v-if="sexuriaSearchOptions.fileHosts.length > 0" class="mb-2">
								<label class="form-label small">File Hosts</label>
								<div class="keyword-tags">
									<button v-for="host in sexuriaSearchOptions.fileHosts" :key="host.id" class="keyword-tag" :class="{ active: sexuriaAdvancedSearch.fileHosts.includes(host.id) }" @click="toggleSexuriaSearchFileHost(host.id)">
										{{ host.name }}
									</button>
								</div>
							</div>
							<div class="mb-2">
								<label class="form-label small">Max Pages</label>
								<input v-model.number="sexuriaAdvancedSearch.maxPages" type="number" class="form-control form-control-sm" min="0" max="9999" placeholder="0 = unlimited" />
							</div>
							<div class="d-flex gap-2">
								<button class="btn btn-outline-secondary btn-sm" @click="resetSexuriaAdvancedSearch" :disabled="sexuriaScraping">
									<font-awesome-icon :icon="['fas', 'undo']" />
								</button>
								<button class="btn btn-primary btn-sm flex-grow-1" @click="startSexuriaAdvancedSearch" :disabled="sexuriaScraping">
									<font-awesome-icon :icon="['fas', sexuriaScraping ? 'spinner' : 'search']" :spin="sexuriaScraping" class="me-2" />
									{{ sexuriaScraping ? 'Searching...' : 'Start Search' }}
								</button>
							</div>
						</div>
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
							<span v-for="(count, host) in sexuriaStats.host_counts" :key="host" class="badge bg-info"> {{ host }}: {{ count }} </span>
						</div>
					</div>
				</div>
			</div>
			<!-- End Sidebar -->
		</div>
		<!-- End scraper-layout -->
		<button v-if="sidebarCollapsed" class="sidebar-toggle" @click="sidebarCollapsed = false" title="Show Controls Panel">
			<font-awesome-icon :icon="['fas', 'cog']" />
		</button>
		<button class="filter-panel-toggle" :class="{ show: filterCollapsed }" @click="filterCollapsed = false" title="Show Filters">
			<font-awesome-icon :icon="['fas', 'sliders-h']" />
		</button>
	</div>
</template>

<script setup>
/* eslint-disable no-unused-vars */
import { ref, reactive, shallowRef, computed, onMounted, onBeforeUnmount, watch, getCurrentInstance } from 'vue'
import { scrapersAPI } from '@/services/api'
import { useScraperJobPolling } from '@/composables/useScraperJobPolling'
import { useVideoSelection }    from '@/composables/useVideoSelection'
import { useOpenURL }           from '@/composables/useOpenURL'
import { useScraperSidebar }    from '@/composables/useScraperSidebar'
import { useScraperAutoRefresh } from '@/composables/useScraperAutoRefresh'
import { useDebouncedRef }      from '@/composables/useDebouncedRef'
import { SCRAPER_PAGE_SIZE }    from '@/utils/scraperConstants'
import { getVideoPerformers, getVideoMatchedTags } from '@/utils/scraperMetadata'
import { LoadingState, EmptyState, StatCard } from '@/components/shared'
import AutoUpdatePanel from '@/components/scraper/AutoUpdatePanel.vue'
import ScraperJobProgress from '@/components/scraper/ScraperJobProgress.vue'
import ScraperBulkActions from '@/components/scraper/ScraperBulkActions.vue'
import { useSendToJD } from '@/composables/useSendToJD'

const { proxy } = getCurrentInstance()
const toast = proxy.$toast
const { sendingToJD, sendToJD } = useSendToJD(toast, 'sexuria')
const sendSelectedToJD = () => sendToJD(sexuriaItems.value, sexuriaSelectedItems.value)
const { pollJob } = useScraperJobPolling()
const {
	selectionMode:        sexuriaSelectionMode,
	selectedVideos:       sexuriaSelectedItems,
	toggleSelectionMode:  toggleSexuriaSelectionMode,
	toggleVideoSelection: toggleSexuriaItemSelection,
	selectAll:            _selectAllSexuria,
} = useVideoSelection()
const { openInFirefox } = useOpenURL()
const { sidebarCollapsed, activeTimeouts } = useScraperSidebar()

// ── Constants ─────────────────────────────────────────────────────────────────
const itemsPerPage   = SCRAPER_PAGE_SIZE
const filterCollapsed = ref(false)

// ── State ─────────────────────────────────────────────────────────────────────
const sexuriaCategoryURL     = ref('')
const sexuriaMaxPages        = ref(0)
const sexuriaScraping        = ref(false)
const sexuriaLoading         = ref(false)
const sexuriaItems           = shallowRef([])
const sexuriaStats           = ref({ total_items: 0, host_counts: {} })
const sexuriaCategories      = ref([])
const sexuriaActiveJob       = ref(null)
const sexuriaItemSearch      = ref('')
const sexuriaHostFilter      = ref('')
const sexuriaCategoryFilter  = ref('')
const sexuriaItemSort        = ref('newest')
const sexuriaItemPage        = ref(1)
const sexuriaTotalCount      = ref(0)
const sexuriaTotalPages      = ref(1)
const sexuriaImgErrors       = reactive({})

const selectAllSexuriaItems = () => _selectAllSexuria(sexuriaItems.value)

// ── Filter helpers ────────────────────────────────────────────────────────────
const activeFilterCount = computed(() => {
	let n = 0
	if (sexuriaItemSort.value !== 'newest')         n++
	if (sexuriaCategoryFilter.value !== '')          n++
	if (sexuriaHostFilter.value !== '')              n++
	if (sexuriaTagFilter.value.length > 0)           n++
	if (sexuriaPornstarFilter.value.length > 0)      n++
	return n
})

const clearAllFilters = () => {
	sexuriaItemSort.value     = 'newest'
	sexuriaCategoryFilter.value = ''
	sexuriaHostFilter.value   = ''
	sexuriaTagFilter.value    = []
	sexuriaPornstarFilter.value = []
	sexuriaItemPage.value     = 1
}

// ── Link Management state ─────────────────────────────────────────────────────
const sexuriaLinkStats = ref({})
const extractingLinks  = ref({})
const sexuriaLinksModal = ref({
	visible: false,
	item: null,
	links: [],
	loading: false,
	verifying: false,
})

// ── Details hover card ────────────────────────────────────────────────────────
const sexuriaDetailsCache = ref({})
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
let sexuriaHideTimeout  = null

// ── Screenshot modal ──────────────────────────────────────────────────────────
const sexuriaScreenshotModal = ref({
	visible: false,
	screenshots: [],
	currentIndex: 0,
})

// ── Multi-select filtering ────────────────────────────────────────────────────
const sexuriaTagFilter          = ref([])
const sexuriaPornstarFilter     = ref([])
const showSexuriaTagDropdown    = ref(false)
const showSexuriaPornstarDropdown = ref(false)

// ── Advanced Search state ─────────────────────────────────────────────────────
const sexuriaAdvancedMode = ref(false)
const sexuriaSearchOptions = ref({ categories: [], fileHosts: [], resolutions: [] })
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

// ── Debounced search ──────────────────────────────────────────────────────────
const debouncedSexuriaSearch = useDebouncedRef(sexuriaItemSearch)

// Reload from server whenever any filter or sort changes
watch(debouncedSexuriaSearch, () => { sexuriaItemPage.value = 1; loadSexuriaItems() })
watch(sexuriaCategoryFilter,  () => { sexuriaItemPage.value = 1; loadSexuriaItems() })
watch(sexuriaHostFilter,      () => { sexuriaItemPage.value = 1; loadSexuriaItems() })
watch(sexuriaItemSort,        () => { sexuriaItemPage.value = 1; loadSexuriaItems() })
watch(sexuriaItemPage,        () => loadSexuriaItems())

// ── Computed ──────────────────────────────────────────────────────────────────
const availableSexuriaTags = computed(() => {
	const tagsSet = new Set()
	Object.values(sexuriaDetailsCache.value).forEach((details) => {
		if (details && details.tags) details.tags.forEach((tag) => tagsSet.add(tag))
	})
	return Array.from(tagsSet).sort()
})

const availableSexuriaPornstars = computed(() => {
	const starsSet = new Set()
	Object.values(sexuriaDetailsCache.value).forEach((details) => {
		if (details && details.porn_stars) details.porn_stars.forEach((star) => starsSet.add(star))
	})
	return Array.from(starsSet).sort()
})

const availableSexuriaCategories = computed(() => {
	const catCounts = {}
	sexuriaItems.value.forEach((item) => {
		const cat = item.category || ''
		if (cat) catCounts[cat] = (catCounts[cat] || 0) + 1
	})
	return Object.entries(catCounts)
		.sort((a, b) => b[1] - a[1])
		.map(([name, count]) => ({ name, count }))
})

const filteredSexuriaItems = computed(() => {
	// Search, category, host, and sort are handled server-side.
	// Only apply client-side filters that require cached detail data.
	let items = [...sexuriaItems.value]
	if (sexuriaTagFilter.value.length > 0) {
		items = items.filter((item) => {
			const cachedDetails = sexuriaDetailsCache.value[item.url]
			if (!cachedDetails || !cachedDetails.tags) return false
			return sexuriaTagFilter.value.every((tag) => cachedDetails.tags.includes(tag))
		})
	}
	if (sexuriaPornstarFilter.value.length > 0) {
		items = items.filter((item) => {
			const cachedDetails = sexuriaDetailsCache.value[item.url]
			if (!cachedDetails || !cachedDetails.porn_stars) return false
			return sexuriaPornstarFilter.value.some((star) => cachedDetails.porn_stars.includes(star))
		})
	}
	return items
})

const paginatedSexuriaItems = computed(() => filteredSexuriaItems.value)

const currentSexuriaScreenshot = computed(() => {
	const { screenshots, currentIndex } = sexuriaScreenshotModal.value
	return screenshots[currentIndex] || null
})

// ── API ───────────────────────────────────────────────────────────────────────
const loadSexuriaCategories = async () => {
	try {
		const data = await scrapersAPI.sexuria.getCategories()
		if (data.success && data.data) sexuriaCategories.value = data.data
	} catch (e) { console.error('Error loading Sexuria categories:', e) }
}

const loadSexuriaStats = async () => {
	try {
		const data = await scrapersAPI.sexuria.getStats()
		if (data.success && data.data) sexuriaStats.value = data.data
	} catch (e) { console.error('Error loading Sexuria stats:', e) }
}

const loadSexuriaItems = async () => {
	sexuriaLoading.value = true
	try {
		const sortMap = {
			'newest':     { sort: 'discovered_at', order: 'desc' },
			'oldest':     { sort: 'discovered_at', order: 'asc' },
			'title-asc':  { sort: 'title',         order: 'asc' },
			'title-desc': { sort: 'title',         order: 'desc' },
		}
		const { sort, order } = sortMap[sexuriaItemSort.value] || sortMap['newest']
		const data = await scrapersAPI.sexuria.getItems({
			page:     sexuriaItemPage.value,
			limit:    itemsPerPage,
			search:   debouncedSexuriaSearch.value || undefined,
			category: sexuriaCategoryFilter.value || undefined,
			host:     sexuriaHostFilter.value || undefined,
			sort,
			order,
		})
		if (data.success && data.data) {
			sexuriaItems.value     = data.data.items || []
			sexuriaTotalCount.value = data.data.total || 0
			sexuriaTotalPages.value = data.data.totalPages || Math.ceil((data.data.total || 0) / itemsPerPage) || 1
		}
	} catch (e) { console.error('Error loading Sexuria items:', e) }
	finally { sexuriaLoading.value = false }
}

const clearSexuriaItems = async () => {
	if (!confirm(`Are you sure you want to delete all ${sexuriaTotalCount.value} Sexuria items? This cannot be undone.`)) return
	try {
		const data = await scrapersAPI.sexuria.clearItems()
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} Sexuria items`)
			sexuriaItems.value = []
			await loadSexuriaStats()
		} else {
			toast.error(data.message || 'Failed to delete items')
		}
	} catch (e) { console.error('Error deleting Sexuria items:', e); toast.error('Failed to delete items') }
}

const deleteSelectedSexuriaItems = async () => {
	if (sexuriaSelectedItems.value.size === 0) return
	if (!confirm(`Are you sure you want to delete ${sexuriaSelectedItems.value.size} selected items?`)) return
	try {
		const ids = Array.from(sexuriaSelectedItems.value)
		const data = await scrapersAPI.bulkDelete(ids)
		if (data.success) {
			toast.success(`Deleted ${data.data.deleted} items`)
			sexuriaSelectedItems.value.clear()
			sexuriaSelectionMode.value = false
			await loadSexuriaItems()
			await loadSexuriaStats()
		} else {
			toast.error(data.message || 'Failed to delete items')
		}
	} catch (e) { console.error('Error deleting selected items:', e); toast.error('Failed to delete selected items') }
}

const startSexuriaScrape = async () => {
	if (!sexuriaCategoryURL.value) { toast.warning('Please enter a category URL'); return }
	sexuriaScraping.value = true
	try {
		const data = await scrapersAPI.sexuria.scrape({ categoryUrl: sexuriaCategoryURL.value, maxPages: sexuriaMaxPages.value || 0 })
		if (data.success) {
			toast.success(`Started Sexuria scrape for: ${sexuriaCategoryURL.value}`)
			sexuriaActiveJob.value = { id: data.data.job_id, status: 'running', pagesCompleted: 0, itemsFound: 0, itemsSaved: 0, errorsCount: 0 }
			pollSexuriaJobProgress(data.data.job_id)
		} else {
			toast.error(data.message || 'Failed to start Sexuria scrape')
			sexuriaScraping.value = false
		}
	} catch (e) { console.error('Error starting Sexuria scrape:', e); toast.error('Failed to start Sexuria scrape'); sexuriaScraping.value = false }
}

const pollSexuriaJobProgress = (jobId) => {
	pollJob(jobId, {
		activeJob:    sexuriaActiveJob,
		isScraping:   sexuriaScraping,
		activeTimeouts,
		onSuccess: async (job) => {
			toast.success(`Sexuria scrape complete! Found ${job.videosFound} items`)
			await loadSexuriaItems()
			await loadSexuriaStats()
		},
		onFail: () => toast.error('Sexuria scrape failed'),
	})
}

const loadSexuriaSearchOptions = async () => {
	try {
		const data = await scrapersAPI.sexuria.getSearchOptions()
		if (data.success && data.data) {
			sexuriaSearchOptions.value = {
				categories: data.data.categories || [],
				fileHosts:  data.data.fileHosts || [],
				resolutions: data.data.resolutions || [],
			}
		}
	} catch (e) { console.error('Error loading Sexuria search options:', e) }
}

const startSexuriaAdvancedSearch = async () => {
	const search = sexuriaAdvancedSearch.value
	const hasSearchCriteria = search.categories.length > 0 || search.title.trim() || search.description.trim() || search.pornstars.trim() || search.production.trim() || search.tags.trim() || search.resolutions.length > 0 || search.fileHosts.length > 0
	if (!hasSearchCriteria) { toast.warning('Please provide at least one search criterion'); return }
	sexuriaScraping.value = true
	try {
		const data = await scrapersAPI.sexuria.advancedSearch(search)
		if (data.success) {
			toast.success('Started Sexuria advanced search')
			sexuriaActiveJob.value = { id: data.data.job_id, status: 'running', pagesCompleted: 0, itemsFound: 0, itemsSaved: 0, errorsCount: 0 }
			pollSexuriaJobProgress(data.data.job_id)
		} else {
			toast.error(data.message || 'Failed to start advanced search')
			sexuriaScraping.value = false
		}
	} catch (e) { console.error('Error starting Sexuria advanced search:', e); toast.error('Failed to start advanced search'); sexuriaScraping.value = false }
}

const resetSexuriaAdvancedSearch = () => {
	sexuriaAdvancedSearch.value = { categories: [], title: '', description: '', pornstars: '', production: '', tags: '', resolutions: [], fileHosts: [], maxPages: 0 }
}

const toggleSexuriaSearchCategory = (categoryId) => {
	const idx = sexuriaAdvancedSearch.value.categories.indexOf(categoryId)
	if (idx >= 0) sexuriaAdvancedSearch.value.categories.splice(idx, 1)
	else sexuriaAdvancedSearch.value.categories.push(categoryId)
}

const toggleSexuriaSearchFileHost = (hostId) => {
	const idx = sexuriaAdvancedSearch.value.fileHosts.indexOf(hostId)
	if (idx >= 0) sexuriaAdvancedSearch.value.fileHosts.splice(idx, 1)
	else sexuriaAdvancedSearch.value.fileHosts.push(hostId)
}

const toggleSexuriaSearchResolution = (resolution) => {
	const idx = sexuriaAdvancedSearch.value.resolutions.indexOf(resolution)
	if (idx >= 0) sexuriaAdvancedSearch.value.resolutions.splice(idx, 1)
	else sexuriaAdvancedSearch.value.resolutions.push(resolution)
}

const getSexuriaFileSize = (item) => {
	if (!item.metadata || item.metadata === '{}') return null
	try { return JSON.parse(item.metadata).file_size || null } catch { return null }
}

const openSexuriaItem = (item) => {
	if (item.url) openInFirefox(item.url)
}

// ── Hover Card ────────────────────────────────────────────────────────────────
const handleSexuriaItemHover = (item, event) => {
	if (sexuriaHideTimeout) { clearTimeout(sexuriaHideTimeout); sexuriaHideTimeout = null }
	if (sexuriaHoverTimeout) clearTimeout(sexuriaHoverTimeout)
	sexuriaHoverTimeout = setTimeout(() => { showSexuriaDetails(item, event) }, 300)
}

const showSexuriaDetails = async (item, event) => {
	const rect          = event.target.getBoundingClientRect()
	const viewportWidth = window.innerWidth
	const viewportHeight = window.innerHeight
	const cardWidth  = 350
	const cardHeight = 400
	let left = rect.right + 10
	let top  = rect.top
	if (left + cardWidth > viewportWidth) left = rect.left - cardWidth - 10
	if (top + cardHeight > viewportHeight) top = viewportHeight - cardHeight - 10
	if (top < 10) top = 10

	sexuriaHoverDetails.value = { visible: true, item, data: null, loading: true, error: null, style: { left: `${left}px`, top: `${top}px` } }

	if (sexuriaDetailsCache.value[item.url]) {
		sexuriaHoverDetails.value.data = sexuriaDetailsCache.value[item.url]
		sexuriaHoverDetails.value.loading = false
		return
	}
	try {
		const data = await scrapersAPI.sexuria.getItemDetails({ url: item.url })
		if (data.success && data.data) {
			sexuriaDetailsCache.value[item.url] = data.data
			if (sexuriaHoverDetails.value.item?.url === item.url) sexuriaHoverDetails.value.data = data.data
		}
	} catch (e) {
		console.error('Failed to fetch Sexuria details:', e)
		if (sexuriaHoverDetails.value.item?.url === item.url) sexuriaHoverDetails.value.error = 'Failed to load details'
	} finally {
		if (sexuriaHoverDetails.value.item?.url === item.url) sexuriaHoverDetails.value.loading = false
	}
}

const keepSexuriaDetailsVisible = () => {
	if (sexuriaHideTimeout) { clearTimeout(sexuriaHideTimeout); sexuriaHideTimeout = null }
}

const hideSexuriaDetails = () => {
	if (sexuriaHoverTimeout) { clearTimeout(sexuriaHoverTimeout); sexuriaHoverTimeout = null }
	sexuriaHideTimeout = setTimeout(() => {
		sexuriaHoverDetails.value.visible = false
		sexuriaHoverDetails.value.item = null
		sexuriaHoverDetails.value.data = null
		sexuriaHoveredScreenshot.value = null
	}, 200)
}

// ── Screenshot Modal ──────────────────────────────────────────────────────────
const openScreenshot = (url) => {
	const screenshots = sexuriaHoverDetails.value.data?.screenshots || []
	const index = screenshots.indexOf(url)
	sexuriaScreenshotModal.value = { visible: true, screenshots, currentIndex: index >= 0 ? index : 0 }
}

const closeSexuriaScreenshotModal = () => { sexuriaScreenshotModal.value.visible = false }

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

// ── Link Management ───────────────────────────────────────────────────────────
const extractSexuriaItemLinks = async (item) => {
	if (!item.url) return
	extractingLinks.value[item.id] = true
	try {
		const data = await scrapersAPI.sexuria.extractItemLinks(item.id, item.url)
		if (data.success) {
			item.links_extracted = true
			const links = data.data.links || []
			sexuriaLinkStats.value[item.id] = {
				total:   links.length,
				working: links.filter((l) => l.status === 'working').length,
				dead:    links.filter((l) => l.status === 'dead').length,
				unknown: links.filter((l) => l.status === 'unknown').length,
			}
			toast.success(`Extracted ${links.length} download links`)
		}
	} catch (e) { console.error('Failed to extract links:', e); toast.error('Failed to extract links') }
	finally { extractingLinks.value[item.id] = false }
}

const showSexuriaItemLinks = async (item) => {
	sexuriaLinksModal.value.visible = true
	sexuriaLinksModal.value.item = item
	sexuriaLinksModal.value.loading = true
	sexuriaLinksModal.value.links = []
	try {
		const data = await scrapersAPI.sexuria.getItemLinks(item.id)
		if (data.success) {
			sexuriaLinksModal.value.links = data.data.links || []
			const links = sexuriaLinksModal.value.links
			sexuriaLinkStats.value[item.id] = {
				total:   links.length,
				working: links.filter((l) => l.status === 'working').length,
				dead:    links.filter((l) => l.status === 'dead').length,
				unknown: links.filter((l) => l.status === 'unknown').length,
			}
		}
	} catch (e) { console.error('Failed to load links:', e); toast.error('Failed to load links') }
	finally { sexuriaLinksModal.value.loading = false }
}

const verifySexuriaItemLinks = async () => {
	const item = sexuriaLinksModal.value.item
	if (!item) return
	sexuriaLinksModal.value.verifying = true
	try {
		const data = await scrapersAPI.sexuria.verifyItemLinks(item.id)
		if (data.success) { toast.success('Links verified'); await showSexuriaItemLinks(item) }
	} catch (e) { console.error('Failed to verify links:', e); toast.error('Failed to verify links') }
	finally { sexuriaLinksModal.value.verifying = false }
}

const closeSexuriaLinksModal = () => {
	sexuriaLinksModal.value.visible = false
	sexuriaLinksModal.value.item = null
	sexuriaLinksModal.value.links = []
}

const getLinkHealthBadgeClass = (stats) => {
	if (!stats || stats.total === 0) return 'bg-secondary'
	const workingPct = (stats.working / stats.total) * 100
	const deadPct    = (stats.dead / stats.total) * 100
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
		case 'working': return 'bg-success'
		case 'dead':    return 'bg-danger'
		default:        return 'bg-secondary'
	}
}

const copyLinkToClipboard = (url) => {
	navigator.clipboard.writeText(url)
	toast.success('Link copied to clipboard')
}

const toggleSexuriaTagFilter = (tag) => {
	const idx = sexuriaTagFilter.value.indexOf(tag)
	if (idx >= 0) sexuriaTagFilter.value.splice(idx, 1)
	else sexuriaTagFilter.value.push(tag)
	sexuriaItemPage.value = 1
}

const toggleSexuriaPornstarFilter = (star) => {
	const idx = sexuriaPornstarFilter.value.indexOf(star)
	if (idx >= 0) sexuriaPornstarFilter.value.splice(idx, 1)
	else sexuriaPornstarFilter.value.push(star)
	sexuriaItemPage.value = 1
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────
useScraperAutoRefresh('sexuria', () => { sexuriaItemPage.value = 1; loadSexuriaItems() })
onMounted(() => {
	loadSexuriaCategories()
	loadSexuriaStats()
	loadSexuriaItems()
	loadSexuriaSearchOptions()
})

onBeforeUnmount(() => {
	if (sexuriaHoverTimeout) clearTimeout(sexuriaHoverTimeout)
	if (sexuriaHideTimeout)  clearTimeout(sexuriaHideTimeout)
})
</script>
