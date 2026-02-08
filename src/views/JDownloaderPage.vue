<template>
	<div class="jdownloader-page" @keydown="handleKeyDown" tabindex="0" ref="pageRef">
		<div class="container-fluid py-3">
			<!-- Header -->
			<div class="page-header mb-4">
				<div class="d-flex justify-content-between align-items-center">
					<div class="d-flex align-items-center gap-3">
						<h1 class="page-title mb-0">
							<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
							JDownloader 2
						</h1>
						<span :class="['connection-badge', connectionStatus.connected ? 'connected' : 'disconnected']">
							<font-awesome-icon :icon="['fas', 'circle']" class="me-1" />
							{{ connectionStatus.connected ? 'Connected' : 'Disconnected' }}
						</span>
					</div>
					<div class="d-flex gap-2">
						<button class="btn btn-primary btn-sm" @click="showAddLinksModal = true" :disabled="!connectionStatus.connected">
							<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
							Add Links
						</button>
						<button class="btn btn-outline-info btn-sm" @click="exportLinks" :disabled="activeTab === 'history'" title="Export Links (Ctrl+E)">
							<font-awesome-icon :icon="['fas', 'file-export']" class="me-1" />
							Export
						</button>
						<button class="btn btn-outline-secondary btn-sm" @click="refreshAll" :disabled="loading">
							<font-awesome-icon :icon="['fas', loading ? 'spinner' : 'sync']" :spin="loading" class="me-1" />
							Refresh
						</button>
						<router-link to="/settings" class="btn btn-outline-light btn-sm">
							<font-awesome-icon :icon="['fas', 'cog']" class="me-1" />
							Settings
						</router-link>
					</div>
				</div>
			</div>

			<!-- Connection Error -->
			<div v-if="!connectionStatus.connected && !loading" class="alert alert-warning mb-4">
				<font-awesome-icon :icon="['fas', 'exclamation-triangle']" class="me-2" />
				JDownloader is not connected. Please check your settings and ensure JDownloader 2 is running.
				<router-link to="/settings" class="alert-link ms-2">Go to Settings</router-link>
			</div>

			<!-- Stats Row -->
			<div class="row g-3 mb-4">
				<div class="col-6 col-lg-2">
					<div class="stat-card">
						<div class="stat-icon download">
							<font-awesome-icon :icon="['fas', 'arrow-down']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ formatSpeed(currentSpeed) }}</div>
							<div class="stat-label">Download Speed</div>
						</div>
					</div>
				</div>
				<div class="col-6 col-lg-2">
					<div class="stat-card">
						<div class="stat-icon queue">
							<font-awesome-icon :icon="['fas', 'list']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ linkGrabberPackages.length }}</div>
							<div class="stat-label">LinkGrabber</div>
						</div>
					</div>
				</div>
				<div class="col-6 col-lg-2">
					<div class="stat-card">
						<div class="stat-icon active">
							<font-awesome-icon :icon="['fas', 'download']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ activeDownloads }}</div>
							<div class="stat-label">Active Downloads</div>
						</div>
					</div>
				</div>
				<div class="col-6 col-lg-2">
					<div class="stat-card">
						<div class="stat-icon limit">
							<font-awesome-icon :icon="['fas', 'tachometer-alt']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ speedLimit > 0 ? formatSpeed(speedLimit) : '∞' }}</div>
							<div class="stat-label">Speed Limit</div>
						</div>
					</div>
				</div>
				<div class="col-6 col-lg-2">
					<div class="stat-card">
						<div class="stat-icon" style="background: linear-gradient(135deg, #17a2b8, #117a8b)">
							<font-awesome-icon :icon="['fas', 'database']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ formatSize(totalDownloadSize) }}</div>
							<div class="stat-label">Total Size</div>
						</div>
					</div>
				</div>
				<div class="col-6 col-lg-2">
					<div class="stat-card">
						<div class="stat-icon" style="background: linear-gradient(135deg, #28a745, #1e7e34)">
							<font-awesome-icon :icon="['fas', 'check-circle']" />
						</div>
						<div class="stat-content">
							<div class="stat-value">{{ completedDownloads }}</div>
							<div class="stat-label">Completed</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Overall Progress (when downloading) -->
			<div v-if="activeDownloads > 0" class="overall-progress mb-4">
				<div class="d-flex justify-content-between align-items-center mb-2">
					<span class="text-muted">Overall Progress</span>
					<span class="text-light">
						{{ formatSize(totalBytesLoaded) }} / {{ formatSize(totalBytesTotal) }}
						<span v-if="overallETA" class="ms-2 text-info">
							<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
							ETA: {{ overallETA }}
						</span>
					</span>
				</div>
				<div class="progress" style="height: 8px">
					<div class="progress-bar bg-success progress-bar-striped progress-bar-animated" :style="{ width: overallProgress + '%' }"></div>
				</div>
			</div>

			<!-- Control Bar -->
			<div class="control-bar mb-4">
				<div class="d-flex flex-wrap gap-2 align-items-center">
					<div class="btn-group">
						<button class="btn btn-success" @click="startDownloads" :disabled="!connectionStatus.connected" title="Start (Ctrl+S)">
							<font-awesome-icon :icon="['fas', 'play']" class="me-1" />
							Start
						</button>
						<button class="btn btn-danger" @click="stopDownloads" :disabled="!connectionStatus.connected" title="Stop (Ctrl+X)">
							<font-awesome-icon :icon="['fas', 'stop']" class="me-1" />
							Stop
						</button>
						<button class="btn btn-warning" @click="pauseDownloads(true)" :disabled="!connectionStatus.connected" title="Pause (Space)">
							<font-awesome-icon :icon="['fas', 'pause']" class="me-1" />
							Pause
						</button>
					</div>

					<div class="vr d-none d-md-block"></div>

					<div class="speed-limit-control d-flex align-items-center gap-2">
						<label class="text-muted small mb-0">Speed Limit:</label>
						<select v-model="selectedSpeedLimit" class="form-select form-select-sm" style="width: auto" @change="setSpeedLimit">
							<option :value="0">Unlimited</option>
							<option :value="512 * 1024">512 KB/s</option>
							<option :value="1024 * 1024">1 MB/s</option>
							<option :value="2 * 1024 * 1024">2 MB/s</option>
							<option :value="5 * 1024 * 1024">5 MB/s</option>
							<option :value="10 * 1024 * 1024">10 MB/s</option>
							<option :value="20 * 1024 * 1024">20 MB/s</option>
						</select>
					</div>

					<div class="vr d-none d-md-block"></div>

					<div class="refresh-control d-flex align-items-center gap-2">
						<label class="text-muted small mb-0">Refresh:</label>
						<select v-model="refreshIntervalSeconds" class="form-select form-select-sm" style="width: auto" @change="updateRefreshInterval">
							<option :value="1">1s</option>
							<option :value="2">2s</option>
							<option :value="3">3s</option>
							<option :value="5">5s</option>
							<option :value="10">10s</option>
							<option :value="0">Off</option>
						</select>
					</div>

					<div class="vr d-none d-md-block"></div>

					<button class="btn btn-outline-secondary btn-sm" @click="cleanupFinished" :disabled="!connectionStatus.connected">
						<font-awesome-icon :icon="['fas', 'broom']" class="me-1" />
						Cleanup Finished
					</button>

					<div class="ms-auto d-none d-lg-block">
						<span class="keyboard-hints"> <kbd>Space</kbd> Pause <kbd>Del</kbd> Remove <kbd>Ctrl+A</kbd> Select All <kbd>Ctrl+E</kbd> Export </span>
					</div>
				</div>
			</div>

			<!-- Main Content Tabs -->
			<ul class="nav nav-tabs mb-3" role="tablist">
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'linkgrabber' }]" @click="activeTab = 'linkgrabber'" type="button">
						<font-awesome-icon :icon="['fas', 'link']" class="me-2" />
						LinkGrabber
						<span class="badge bg-secondary ms-2">{{ filteredLinkGrabberPackages.length }}</span>
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'downloads' }]" @click="activeTab = 'downloads'" type="button">
						<font-awesome-icon :icon="['fas', 'download']" class="me-2" />
						Downloads
						<span class="badge bg-secondary ms-2">{{ filteredDownloadPackages.length }}</span>
					</button>
				</li>
				<li class="nav-item" role="presentation">
					<button :class="['nav-link', { active: activeTab === 'history' }]" @click=";(activeTab = 'history'), loadHistory()" type="button">
						<font-awesome-icon :icon="['fas', 'history']" class="me-2" />
						History
					</button>
				</li>
			</ul>

			<!-- Filter Bar -->
			<div v-if="activeTab !== 'history'" class="filter-bar mb-3">
				<div class="row g-2 align-items-center">
					<div class="col-lg-3 col-md-6">
						<div class="input-group input-group-sm">
							<span class="input-group-text">
								<font-awesome-icon :icon="['fas', 'search']" />
							</span>
							<input v-model="searchQuery" type="text" class="form-control" placeholder="Search packages..." @keydown.escape="searchQuery = ''" />
							<button v-if="searchQuery" class="btn btn-outline-secondary" @click="searchQuery = ''">
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
						</div>
					</div>
					<div class="col-lg-2 col-md-3">
						<select v-model="statusFilter" class="form-select form-select-sm">
							<option value="">All Status</option>
							<option value="running">Running</option>
							<option value="queued">Queued</option>
							<option value="finished">Finished</option>
							<option value="failed">Failed</option>
							<option value="paused">Paused</option>
						</select>
					</div>
					<div class="col-lg-2 col-md-3">
						<select v-model="hostFilter" class="form-select form-select-sm">
							<option value="">All Hosts</option>
							<option v-for="host in availableHosts" :key="host" :value="host">{{ host }}</option>
						</select>
					</div>
					<div class="col-lg-2 col-md-3">
						<select v-model="enabledFilter" class="form-select form-select-sm">
							<option value="">All Items</option>
							<option value="enabled">Enabled Only</option>
							<option value="disabled">Disabled Only</option>
						</select>
					</div>
					<div class="col-lg-2 col-md-3">
						<select v-model="sortBy" class="form-select form-select-sm">
							<option value="added-desc">Newest First</option>
							<option value="added-asc">Oldest First</option>
							<option value="name-asc">Name A-Z</option>
							<option value="name-desc">Name Z-A</option>
							<option value="size-desc">Size (Largest)</option>
							<option value="size-asc">Size (Smallest)</option>
							<option value="links-desc">Links (Most)</option>
							<option value="links-asc">Links (Fewest)</option>
							<option value="progress-desc">Progress (Most)</option>
							<option value="progress-asc">Progress (Least)</option>
						</select>
					</div>
					<div class="col-lg-1 col-md-3 text-end d-flex gap-1 justify-content-end">
						<button v-if="activeFilterCount > 0" class="btn btn-outline-warning btn-sm" @click="clearAllFilters" title="Clear all filters">
							<font-awesome-icon :icon="['fas', 'times-circle']" />
							<span class="badge bg-warning text-dark ms-1">{{ activeFilterCount }}</span>
						</button>
						<button class="btn btn-outline-light btn-sm" @click="selectAll" title="Select All (Ctrl+A)">
							<font-awesome-icon :icon="['fas', 'check-double']" />
						</button>
					</div>
				</div>
			</div>

			<!-- Tab Content -->
			<div class="tab-content">
				<!-- LinkGrabber Tab -->
				<div v-if="activeTab === 'linkgrabber'" class="tab-pane active">
					<!-- LinkGrabber Actions -->
					<div class="d-flex flex-wrap gap-2 mb-3">
						<button class="btn btn-primary btn-sm" @click="moveAllToDownloads" :disabled="filteredLinkGrabberPackages.length === 0">
							<font-awesome-icon :icon="['fas', 'arrow-right']" class="me-1" />
							Move All to Downloads
						</button>
						<button class="btn btn-outline-primary btn-sm" @click="moveSelectedToDownloads" :disabled="selectedLinkGrabberItems.length === 0">
							<font-awesome-icon :icon="['fas', 'arrow-right']" class="me-1" />
							Move Selected ({{ selectedLinkGrabberItems.length }})
						</button>
						<button class="btn btn-outline-danger btn-sm" @click="removeSelectedFromLinkGrabber" :disabled="selectedLinkGrabberItems.length === 0" title="Delete (Del)">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
							Remove Selected
						</button>
						<button class="btn btn-outline-warning btn-sm" @click="clearLinkGrabber" :disabled="linkGrabberPackages.length === 0">
							<font-awesome-icon :icon="['fas', 'times']" class="me-1" />
							Clear All
						</button>
					</div>

					<!-- LinkGrabber Packages -->
					<div v-if="filteredLinkGrabberPackages.length === 0" class="empty-state">
						<font-awesome-icon :icon="['fas', 'inbox']" class="empty-icon" />
						<p>{{ searchQuery || statusFilter || hostFilter ? 'No packages match your filters' : 'No links in LinkGrabber' }}</p>
						<p class="text-muted small">Add links from scraped videos or paste URLs directly</p>
						<button
							v-if="!searchQuery && !statusFilter && !hostFilter"
							class="btn btn-primary mt-2"
							@click="showAddLinksModal = true"
							:disabled="!connectionStatus.connected"
						>
							<font-awesome-icon :icon="['fas', 'plus']" class="me-1" />
							Add Links
						</button>
					</div>

					<div v-else class="packages-list">
						<div v-for="pkg in filteredLinkGrabberPackages" :key="pkg.uuid" class="package-card" @contextmenu.prevent="showContextMenu($event, 'lg', pkg)">
							<div class="package-header" @click="togglePackage('lg', pkg.uuid)">
								<div class="d-flex align-items-center gap-2">
									<input
										type="checkbox"
										class="form-check-input"
										:checked="isPackageSelected('lg', pkg.uuid)"
										@click.stop="togglePackageSelection('lg', pkg.uuid)"
									/>
									<font-awesome-icon :icon="['fas', expandedPackages.lg[pkg.uuid] ? 'chevron-down' : 'chevron-right']" class="expand-icon" />

									<!-- Editable Package Name -->
									<template v-if="editingPackage === pkg.uuid">
										<input
											v-model="editingPackageName"
											type="text"
											class="form-control form-control-sm package-name-input"
											@click.stop
											@keydown.enter="savePackageName('lg', pkg.uuid)"
											@keydown.escape="cancelEditPackage"
											@blur="savePackageName('lg', pkg.uuid)"
											ref="packageNameInput"
										/>
									</template>
									<template v-else>
										<span class="package-name" @dblclick.stop="startEditPackage(pkg)">
											{{ pkg.name || 'Unnamed Package' }}
										</span>
									</template>

									<span class="badge bg-info">{{ pkg.childCount || 0 }} links</span>
									<span v-if="pkg.bytesTotal > 0" class="badge bg-secondary">{{ formatSize(pkg.bytesTotal) }}</span>
								</div>
								<div class="package-actions" @click.stop>
									<!-- Priority Dropdown -->
									<div class="dropdown d-inline-block">
										<button class="btn btn-sm btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" @click.stop>
											<font-awesome-icon :icon="['fas', 'sort']" />
										</button>
										<ul class="dropdown-menu dropdown-menu-dark">
											<li>
												<a class="dropdown-item" href="#" @click.prevent="setPriority('lg', pkg.uuid, 'HIGHEST')">
													<font-awesome-icon :icon="['fas', 'angle-double-up']" class="me-2 text-danger" />Highest
												</a>
											</li>
											<li>
												<a class="dropdown-item" href="#" @click.prevent="setPriority('lg', pkg.uuid, 'HIGHER')">
													<font-awesome-icon :icon="['fas', 'angle-up']" class="me-2 text-warning" />Higher
												</a>
											</li>
											<li>
												<a class="dropdown-item" href="#" @click.prevent="setPriority('lg', pkg.uuid, 'HIGH')">
													<font-awesome-icon :icon="['fas', 'arrow-up']" class="me-2 text-info" />High
												</a>
											</li>
											<li>
												<a class="dropdown-item" href="#" @click.prevent="setPriority('lg', pkg.uuid, 'DEFAULT')">
													<font-awesome-icon :icon="['fas', 'minus']" class="me-2" />Default
												</a>
											</li>
											<li>
												<a class="dropdown-item" href="#" @click.prevent="setPriority('lg', pkg.uuid, 'LOW')">
													<font-awesome-icon :icon="['fas', 'arrow-down']" class="me-2 text-secondary" />Low
												</a>
											</li>
										</ul>
									</div>
									<button class="btn btn-sm btn-outline-success" @click="togglePackageEnabled('lg', pkg)" :title="pkg.enabled === false ? 'Enable' : 'Disable'">
										<font-awesome-icon :icon="['fas', pkg.enabled === false ? 'eye-slash' : 'eye']" />
									</button>
									<button class="btn btn-sm btn-outline-primary" @click="movePackageToDownloads(pkg.uuid)" title="Move to Downloads">
										<font-awesome-icon :icon="['fas', 'arrow-right']" />
									</button>
									<button class="btn btn-sm btn-outline-danger" @click="removePackage('lg', pkg.uuid)" title="Remove">
										<font-awesome-icon :icon="['fas', 'trash']" />
									</button>
								</div>
							</div>
							<div v-if="expandedPackages.lg[pkg.uuid]" class="package-links">
								<div
									v-for="link in getLinkGrabberLinksForPackage(pkg.uuid)"
									:key="link.uuid"
									class="link-item"
									:class="{ 'link-disabled': link.enabled === false }"
									@contextmenu.prevent="showContextMenu($event, 'lg-link', link)"
								>
									<input type="checkbox" class="form-check-input" :checked="isLinkSelected('lg', link.uuid)" @change="toggleLinkSelection('lg', link.uuid)" />
									<button class="btn btn-sm btn-link p-0 me-2" @click="toggleLinkEnabled('lg', link)" :title="link.enabled === false ? 'Enable' : 'Disable'">
										<font-awesome-icon
											:icon="['fas', link.enabled === false ? 'eye-slash' : 'eye']"
											:class="link.enabled === false ? 'text-muted' : 'text-success'"
										/>
									</button>
									<div class="link-info">
										<div class="link-name">{{ link.name || 'Unknown' }}</div>
										<div class="link-meta">
											<span class="host">{{ link.host || 'Unknown Host' }}</span>
											<span v-if="link.bytesTotal > 0" class="size">{{ formatSize(link.bytesTotal) }}</span>
											<span :class="['availability', getAvailabilityClass(link.availability)]">
												{{ link.availability || 'Unknown' }}
											</span>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Downloads Tab -->
				<div v-if="activeTab === 'downloads'" class="tab-pane active">
					<!-- Downloads Actions -->
					<div class="d-flex flex-wrap gap-2 mb-3">
						<button class="btn btn-success btn-sm" @click="forceDownloadSelected" :disabled="selectedDownloadItems.length === 0">
							<font-awesome-icon :icon="['fas', 'bolt']" class="me-1" />
							Force Download ({{ selectedDownloadItems.length }})
						</button>
						<button class="btn btn-outline-warning btn-sm" @click="resetSelectedLinks" :disabled="selectedDownloadItems.length === 0">
							<font-awesome-icon :icon="['fas', 'redo']" class="me-1" />
							Reset Selected
						</button>
						<button class="btn btn-outline-danger btn-sm" @click="removeSelectedFromDownloads" :disabled="selectedDownloadItems.length === 0" title="Delete (Del)">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
							Remove Selected
						</button>
					</div>

					<!-- Pagination Info -->
					<div v-if="filteredDownloadPackages.length > downloadsPerPage" class="d-flex justify-content-between align-items-center mb-3 pagination-info">
						<span class="text-muted small">
							Showing {{ (downloadCurrentPage - 1) * downloadsPerPage + 1 }}-{{
								Math.min(downloadCurrentPage * downloadsPerPage, filteredDownloadPackages.length)
							}}
							of {{ filteredDownloadPackages.length }} packages
						</span>
						<div class="d-flex align-items-center gap-2">
							<label class="text-muted small mb-0">Per page:</label>
							<select v-model="downloadsPerPage" class="form-select form-select-sm" style="width: auto">
								<option :value="25">25</option>
								<option :value="50">50</option>
								<option :value="100">100</option>
								<option :value="200">200</option>
							</select>
						</div>
					</div>

					<!-- Download Packages -->
					<div v-if="filteredDownloadPackages.length === 0" class="empty-state">
						<font-awesome-icon :icon="['fas', 'download']" class="empty-icon" />
						<p>{{ searchQuery || statusFilter || hostFilter ? 'No packages match your filters' : 'No active downloads' }}</p>
						<p class="text-muted small">Move links from LinkGrabber to start downloading</p>
					</div>

					<div v-else class="packages-list">
						<div
							v-for="pkg in paginatedDownloadPackages"
							:key="pkg.uuid"
							class="package-card"
							:class="{ 'package-finished': pkg.finished }"
							@contextmenu.prevent="showContextMenu($event, 'dl', pkg)"
						>
							<div class="package-header" @click="togglePackage('dl', pkg.uuid)">
								<div class="d-flex align-items-center gap-2">
									<input
										type="checkbox"
										class="form-check-input"
										:checked="isPackageSelected('dl', pkg.uuid)"
										@click.stop="togglePackageSelection('dl', pkg.uuid)"
									/>
									<font-awesome-icon :icon="['fas', expandedPackages.dl[pkg.uuid] ? 'chevron-down' : 'chevron-right']" class="expand-icon" />

									<!-- Editable Package Name -->
									<template v-if="editingPackage === pkg.uuid">
										<input
											v-model="editingPackageName"
											type="text"
											class="form-control form-control-sm package-name-input"
											@click.stop
											@keydown.enter="savePackageName('dl', pkg.uuid)"
											@keydown.escape="cancelEditPackage"
											@blur="savePackageName('dl', pkg.uuid)"
										/>
									</template>
									<template v-else>
										<span class="package-name" @dblclick.stop="startEditPackage(pkg)">
											{{ pkg.name || 'Unnamed Package' }}
										</span>
									</template>

									<!-- Download Path -->
									<span v-if="pkg.saveTo" class="package-path text-muted small" :title="pkg.saveTo">
										<font-awesome-icon :icon="['fas', 'folder']" class="me-1" />
										{{ formatPath(pkg.saveTo) }}
									</span>

									<span class="badge bg-info">{{ pkg.childCount || 0 }} links</span>
									<span v-if="pkg.finished" class="badge bg-success">
										<font-awesome-icon :icon="['fas', 'check']" class="me-1" />
										Finished
									</span>
									<span v-else-if="pkg.speed > 0" class="badge bg-primary">
										<font-awesome-icon :icon="['fas', 'arrow-down']" class="me-1" />
										{{ formatSpeed(pkg.speed) }}
									</span>
									<span v-if="pkg.eta > 0 && !pkg.finished" class="badge bg-secondary">
										<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
										{{ formatETA(pkg.eta) }}
									</span>
								</div>
								<div class="package-progress-info d-flex align-items-center gap-3">
									<span v-if="pkg.bytesTotal > 0" class="text-muted small">
										{{ formatSize(pkg.bytesLoaded || 0) }} / {{ formatSize(pkg.bytesTotal) }}
										<span class="ms-1">({{ getProgressPercent(pkg) }}%)</span>
									</span>
									<div class="package-actions" @click.stop>
										<!-- Priority Dropdown -->
										<div class="dropdown d-inline-block">
											<button class="btn btn-sm btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" @click.stop>
												<font-awesome-icon :icon="['fas', 'sort']" />
											</button>
											<ul class="dropdown-menu dropdown-menu-dark">
												<li>
													<a class="dropdown-item" href="#" @click.prevent="setPriority('dl', pkg.uuid, 'HIGHEST')">
														<font-awesome-icon :icon="['fas', 'angle-double-up']" class="me-2 text-danger" />Highest
													</a>
												</li>
												<li>
													<a class="dropdown-item" href="#" @click.prevent="setPriority('dl', pkg.uuid, 'HIGHER')">
														<font-awesome-icon :icon="['fas', 'angle-up']" class="me-2 text-warning" />Higher
													</a>
												</li>
												<li>
													<a class="dropdown-item" href="#" @click.prevent="setPriority('dl', pkg.uuid, 'HIGH')">
														<font-awesome-icon :icon="['fas', 'arrow-up']" class="me-2 text-info" />High
													</a>
												</li>
												<li>
													<a class="dropdown-item" href="#" @click.prevent="setPriority('dl', pkg.uuid, 'DEFAULT')">
														<font-awesome-icon :icon="['fas', 'minus']" class="me-2" />Default
													</a>
												</li>
												<li>
													<a class="dropdown-item" href="#" @click.prevent="setPriority('dl', pkg.uuid, 'LOW')">
														<font-awesome-icon :icon="['fas', 'arrow-down']" class="me-2 text-secondary" />Low
													</a>
												</li>
											</ul>
										</div>
										<button
											class="btn btn-sm btn-outline-success"
											@click="togglePackageEnabled('dl', pkg)"
											:title="pkg.enabled === false ? 'Enable' : 'Disable'"
										>
											<font-awesome-icon :icon="['fas', pkg.enabled === false ? 'eye-slash' : 'eye']" />
										</button>
										<button class="btn btn-sm btn-outline-danger" @click="removePackage('dl', pkg.uuid)" title="Remove">
											<font-awesome-icon :icon="['fas', 'trash']" />
										</button>
									</div>
								</div>
							</div>
							<!-- Package Progress Bar -->
							<div v-if="pkg.bytesTotal > 0" class="package-progress">
								<div class="progress">
									<div class="progress-bar" :style="{ width: getProgressPercent(pkg) + '%' }" :class="getProgressClass(pkg)"></div>
								</div>
							</div>
							<div v-if="expandedPackages.dl[pkg.uuid]" class="package-links">
								<div
									v-for="link in getDownloadLinksForPackage(pkg.uuid)"
									:key="link.uuid"
									class="link-item"
									:class="{ 'link-finished': link.finished, 'link-disabled': link.enabled === false }"
									@contextmenu.prevent="showContextMenu($event, 'dl-link', link)"
								>
									<input type="checkbox" class="form-check-input" :checked="isLinkSelected('dl', link.uuid)" @change="toggleLinkSelection('dl', link.uuid)" />
									<button class="btn btn-sm btn-link p-0 me-2" @click="toggleLinkEnabled('dl', link)" :title="link.enabled === false ? 'Enable' : 'Disable'">
										<font-awesome-icon
											:icon="['fas', link.enabled === false ? 'eye-slash' : 'eye']"
											:class="link.enabled === false ? 'text-muted' : 'text-success'"
										/>
									</button>
									<div class="link-info flex-grow-1">
										<div class="d-flex justify-content-between">
											<div class="link-name">{{ link.name || 'Unknown' }}</div>
											<div class="d-flex align-items-center gap-2">
												<span v-if="link.eta > 0 && !link.finished" class="text-muted small">
													<font-awesome-icon :icon="['fas', 'clock']" class="me-1" />
													{{ formatETA(link.eta) }}
												</span>
												<span :class="['status-badge', getStatusClass(link)]">
													{{ link.finished ? 'Finished' : link.running ? 'Downloading' : link.status || 'Queued' }}
												</span>
											</div>
										</div>
										<div class="link-meta">
											<span class="host">{{ link.host || 'Unknown Host' }}</span>
											<span v-if="link.bytesTotal > 0" class="size"> {{ formatSize(link.bytesLoaded || 0) }} / {{ formatSize(link.bytesTotal) }} </span>
											<span v-if="link.speed > 0" class="speed">
												<font-awesome-icon :icon="['fas', 'arrow-down']" />
												{{ formatSpeed(link.speed) }}
											</span>
										</div>
										<!-- Link Progress -->
										<div v-if="link.bytesTotal > 0" class="link-progress mt-1">
											<div class="progress" style="height: 4px">
												<div class="progress-bar" :style="{ width: getLinkProgressPercent(link) + '%' }" :class="getProgressClass(link)"></div>
											</div>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>

					<!-- Pagination Controls -->
					<div v-if="totalDownloadPages > 1" class="pagination-controls mt-3 d-flex justify-content-center align-items-center gap-2">
						<button class="btn btn-outline-light btn-sm" @click="downloadCurrentPage = 1" :disabled="downloadCurrentPage === 1">
							<font-awesome-icon :icon="['fas', 'angle-double-left']" />
						</button>
						<button class="btn btn-outline-light btn-sm" @click="downloadCurrentPage--" :disabled="downloadCurrentPage === 1">
							<font-awesome-icon :icon="['fas', 'angle-left']" />
						</button>
						<span class="text-muted"> Page {{ downloadCurrentPage }} of {{ totalDownloadPages }} </span>
						<button class="btn btn-outline-light btn-sm" @click="downloadCurrentPage++" :disabled="downloadCurrentPage === totalDownloadPages">
							<font-awesome-icon :icon="['fas', 'angle-right']" />
						</button>
						<button class="btn btn-outline-light btn-sm" @click="downloadCurrentPage = totalDownloadPages" :disabled="downloadCurrentPage === totalDownloadPages">
							<font-awesome-icon :icon="['fas', 'angle-double-right']" />
						</button>
					</div>
				</div>

				<!-- History Tab -->
				<div v-if="activeTab === 'history'" class="tab-pane active">
					<div class="d-flex justify-content-between align-items-center mb-3">
						<span class="text-muted">Links sent to JDownloader from this app</span>
						<button class="btn btn-outline-danger btn-sm" @click="clearHistory" :disabled="history.length === 0">
							<font-awesome-icon :icon="['fas', 'trash']" class="me-1" />
							Clear History
						</button>
					</div>

					<div v-if="history.length === 0" class="empty-state">
						<font-awesome-icon :icon="['fas', 'history']" class="empty-icon" />
						<p>No history yet</p>
						<p class="text-muted small">Links sent to JDownloader will appear here</p>
					</div>

					<div v-else class="history-list">
						<div v-for="item in history" :key="item.id" class="history-item">
							<div class="history-info">
								<div class="history-title">{{ item.videoTitle || item.linkUrl }}</div>
								<div class="history-meta">
									<span class="package">{{ item.packageName || 'No package' }}</span>
									<span class="time">{{ formatDate(item.sentAt) }}</span>
								</div>
							</div>
							<span :class="['status-badge', getHistoryStatusClass(item.status)]">{{ item.status }}</span>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Context Menu -->
		<div v-if="contextMenu.show" class="context-menu" :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }" @click.stop>
			<div class="context-menu-item" @click="contextMenuAction('rename')">
				<font-awesome-icon :icon="['fas', 'edit']" class="me-2" />
				Rename
			</div>
			<div class="context-menu-item" @click="contextMenuAction('copy-url')">
				<font-awesome-icon :icon="['fas', 'copy']" class="me-2" />
				Copy URL
			</div>
			<div class="context-menu-divider"></div>
			<div class="context-menu-item" @click="contextMenuAction('priority-highest')">
				<font-awesome-icon :icon="['fas', 'angle-double-up']" class="me-2 text-danger" />
				Priority: Highest
			</div>
			<div class="context-menu-item" @click="contextMenuAction('priority-high')">
				<font-awesome-icon :icon="['fas', 'angle-up']" class="me-2 text-warning" />
				Priority: High
			</div>
			<div class="context-menu-item" @click="contextMenuAction('priority-default')">
				<font-awesome-icon :icon="['fas', 'minus']" class="me-2" />
				Priority: Default
			</div>
			<div class="context-menu-item" @click="contextMenuAction('priority-low')">
				<font-awesome-icon :icon="['fas', 'angle-down']" class="me-2 text-secondary" />
				Priority: Low
			</div>
			<div class="context-menu-divider"></div>
			<div v-if="contextMenu.type.includes('lg')" class="context-menu-item" @click="contextMenuAction('move')">
				<font-awesome-icon :icon="['fas', 'arrow-right']" class="me-2 text-primary" />
				Move to Downloads
			</div>
			<div v-if="contextMenu.type === 'dl' || contextMenu.type === 'dl-link'" class="context-menu-item" @click="contextMenuAction('force')">
				<font-awesome-icon :icon="['fas', 'bolt']" class="me-2 text-warning" />
				Force Download
			</div>
			<div class="context-menu-item" @click="contextMenuAction('toggle-enabled')">
				<font-awesome-icon :icon="['fas', 'eye']" class="me-2" />
				{{ contextMenu.item?.enabled === false ? 'Enable' : 'Disable' }}
			</div>
			<div class="context-menu-divider"></div>
			<div class="context-menu-item text-danger" @click="contextMenuAction('remove')">
				<font-awesome-icon :icon="['fas', 'trash']" class="me-2" />
				Remove
			</div>
		</div>

		<!-- Add Links Modal -->
		<div v-if="showAddLinksModal" class="modal-backdrop" @click.self="showAddLinksModal = false">
			<div class="modal-content add-links-modal">
				<div class="modal-header">
					<h5 class="modal-title">
						<font-awesome-icon :icon="['fas', 'plus']" class="me-2" />
						Add Links to JDownloader
					</h5>
					<button class="btn-close" @click="showAddLinksModal = false"></button>
				</div>
				<div class="modal-body">
					<div class="mb-3">
						<label class="form-label">Paste URLs (one per line)</label>
						<textarea
							v-model="newLinksText"
							class="form-control"
							rows="8"
							placeholder="https://example.com/video1.mp4&#10;https://example.com/video2.mp4&#10;..."
						></textarea>
					</div>
					<div class="mb-3" style="position: relative">
						<label class="form-label">Download Path</label>
						<div class="input-group">
							<span class="input-group-text">/output/</span>
							<input
								v-model="newLinksPath"
								type="text"
								class="form-control"
								placeholder="subfolder (leave empty for root)"
								@focus="showPathDropdown = true"
								@blur="hidePathDropdownDelayed"
							/>
							<button
								v-if="newLinksPath"
								class="btn btn-outline-secondary"
								@click="newLinksPath = ''"
								title="Clear"
							>
								<font-awesome-icon :icon="['fas', 'times']" />
							</button>
						</div>
						<div v-if="showPathDropdown && downloadPathHistory.length > 0" class="path-history-dropdown">
							<div class="path-history-header">
								<span class="text-muted small">Recent paths</span>
								<button class="btn btn-link btn-sm text-danger p-0" @mousedown.prevent="clearPathHistory">clear</button>
							</div>
							<div
								v-for="path in filteredPathHistory"
								:key="path"
								class="path-history-item"
								@mousedown.prevent="selectPath(path)"
							>
								<font-awesome-icon :icon="['fas', 'folder']" class="me-2 text-muted" />
								/output/{{ path }}
							</div>
						</div>
						<small class="text-muted">Downloads to /output/ by default. Add subfolder name to customize.</small>
					</div>
					<div class="mb-3">
						<label class="form-label">Package Name (optional, leave empty for auto)</label>
						<input v-model="newLinksPackageName" type="text" class="form-control" placeholder="Auto-detect from links" />
					</div>
					<div class="form-check">
						<input v-model="newLinksAutoStart" type="checkbox" class="form-check-input" id="autoStart" />
						<label class="form-check-label" for="autoStart">Auto-start downloads</label>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showAddLinksModal = false">Cancel</button>
					<button class="btn btn-primary" @click="addLinks" :disabled="!newLinksText.trim() || addingLinks">
						<font-awesome-icon v-if="addingLinks" :icon="['fas', 'spinner']" spin class="me-1" />
						<font-awesome-icon v-else :icon="['fas', 'plus']" class="me-1" />
						Add {{ linksCount }} Link{{ linksCount !== 1 ? 's' : '' }}
					</button>
				</div>
			</div>
		</div>

		<!-- Export Modal -->
		<div v-if="showExportModal" class="modal-backdrop" @click.self="showExportModal = false">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">
						<font-awesome-icon :icon="['fas', 'file-export']" class="me-2" />
						Export Links
					</h5>
					<button class="btn-close" @click="showExportModal = false"></button>
				</div>
				<div class="modal-body">
					<div class="mb-3">
						<label class="form-label">Exported URLs</label>
						<textarea v-model="exportedLinksText" class="form-control" rows="10" readonly></textarea>
					</div>
					<div class="d-flex gap-2">
						<button class="btn btn-primary" @click="copyExportedLinks">
							<font-awesome-icon :icon="['fas', 'copy']" class="me-1" />
							Copy to Clipboard
						</button>
						<button class="btn btn-outline-primary" @click="downloadExportedLinks">
							<font-awesome-icon :icon="['fas', 'download']" class="me-1" />
							Download as File
						</button>
					</div>
				</div>
				<div class="modal-footer">
					<button class="btn btn-secondary" @click="showExportModal = false">Close</button>
				</div>
			</div>
		</div>

		<!-- Toast Notifications -->
		<div class="toast-container">
			<transition-group name="toast">
				<div v-for="toast in toasts" :key="toast.id" :class="['toast-notification', toast.type]">
					<font-awesome-icon :icon="['fas', getToastIcon(toast.type)]" class="me-2" />
					{{ toast.message }}
				</div>
			</transition-group>
		</div>
	</div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { jdownloaderAPI } from '@/services/api'

// Refs
const pageRef = ref(null)

// State
const loading = ref(false)
const activeTab = ref('linkgrabber')
const connectionStatus = ref({ connected: false })
const currentSpeed = ref(0)
const speedLimit = ref(0)
const selectedSpeedLimit = ref(0)
const refreshIntervalSeconds = ref(3)

// Filter state
const searchQuery = ref('')
const statusFilter = ref('')
const hostFilter = ref('')
const sortBy = ref('added-desc')
const enabledFilter = ref('')

// LinkGrabber state
const linkGrabberPackages = ref([])
const linkGrabberLinks = ref([])
const selectedLinkGrabberPackages = ref(new Set())
const selectedLinkGrabberLinks = ref(new Set())

// Downloads state
const downloadPackages = ref([])
const downloadLinks = ref([])
const selectedDownloadPackages = ref(new Set())
const selectedDownloadLinks = ref(new Set())

// History state
const history = ref([])

// Locally-tracked package add timestamps (package_name → Unix ms)
const packageTimestamps = ref({})

// Expanded packages
const expandedPackages = reactive({
	lg: {},
	dl: {},
})

// Package rename state
const editingPackage = ref(null)
const editingPackageName = ref('')

// Context menu state
const contextMenu = reactive({
	show: false,
	x: 0,
	y: 0,
	type: '',
	item: null,
})

// Add Links Modal state
const showAddLinksModal = ref(false)
const newLinksText = ref('')
const newLinksPath = ref('')
const newLinksPackageName = ref('')
const newLinksAutoStart = ref(true)
const addingLinks = ref(false)

// Download path history (persisted in localStorage)
const STORAGE_KEY = 'jd-download-path-history'
const downloadPathHistory = ref(JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]'))
const showPathDropdown = ref(false)

// Export Modal state
const showExportModal = ref(false)
const exportedLinksText = ref('')

// Toast notifications
const toasts = ref([])
let toastId = 0

// Track previously finished downloads for notifications
const previouslyFinishedPackages = ref(new Set())
const isInitialLoad = ref(true)

// Pagination for downloads (to handle 1000+ items)
const downloadsPerPage = ref(50)
const downloadCurrentPage = ref(1)

// Auto refresh interval
let refreshInterval = null

// Computed - Filtering
const filteredLinkGrabberPackages = computed(() => {
	return filterPackages(linkGrabberPackages.value, linkGrabberLinks.value)
})

const filteredDownloadPackages = computed(() => {
	return filterPackages(downloadPackages.value, downloadLinks.value)
})

// Paginated downloads to handle 1000+ items
const paginatedDownloadPackages = computed(() => {
	const start = (downloadCurrentPage.value - 1) * downloadsPerPage.value
	const end = start + downloadsPerPage.value
	return filteredDownloadPackages.value.slice(start, end)
})

const totalDownloadPages = computed(() => {
	return Math.ceil(filteredDownloadPackages.value.length / downloadsPerPage.value)
})

const filterPackages = (packages, links) => {
	let result = packages.filter((pkg) => {
		// Search filter
		if (searchQuery.value) {
			const query = searchQuery.value.toLowerCase()
			if (!(pkg.name || '').toLowerCase().includes(query)) {
				return false
			}
		}

		// Status filter
		if (statusFilter.value) {
			const pkgLinks = links.filter((l) => l.packageUUID === pkg.uuid)
			switch (statusFilter.value) {
				case 'running':
					if (!pkgLinks.some((l) => l.running && !l.finished)) return false
					break
				case 'finished':
					if (!pkg.finished && !pkgLinks.every((l) => l.finished)) return false
					break
				case 'queued':
					if (pkg.finished || pkgLinks.some((l) => l.running || l.finished)) return false
					break
				case 'failed':
					if (!pkgLinks.some((l) => (l.status || '').toLowerCase().includes('error'))) return false
					break
				case 'paused':
					if (!pkgLinks.some((l) => l.skipped)) return false
					break
			}
		}

		// Host filter
		if (hostFilter.value) {
			const pkgLinks = links.filter((l) => l.packageUUID === pkg.uuid)
			if (!pkgLinks.some((l) => l.host === hostFilter.value)) return false
		}

		// Enabled filter
		if (enabledFilter.value === 'enabled' && pkg.enabled === false) return false
		if (enabledFilter.value === 'disabled' && pkg.enabled !== false) return false

		return true
	})

	// Sort
	const [sortField, sortDir] = sortBy.value.split('-')
	result.sort((a, b) => {
		let valA, valB
		switch (sortField) {
			case 'name':
				valA = (a.name || '').toLowerCase()
				valB = (b.name || '').toLowerCase()
				return sortDir === 'asc' ? valA.localeCompare(valB) : valB.localeCompare(valA)
			case 'size':
				valA = a.bytesTotal || 0
				valB = b.bytesTotal || 0
				break
			case 'links':
				valA = a.childCount || 0
				valB = b.childCount || 0
				break
			case 'progress':
				valA = a.bytesTotal > 0 ? (a.bytesLoaded || 0) / a.bytesTotal : 0
				valB = b.bytesTotal > 0 ? (b.bytesLoaded || 0) / b.bytesTotal : 0
				break
			case 'added':
				valA = a.addedDate || a.localAddedAt || 0
				valB = b.addedDate || b.localAddedAt || 0
				break
			default:
				return 0
		}
		return sortDir === 'asc' ? valA - valB : valB - valA
	})

	return result
}

// Computed - Available hosts for filter
const availableHosts = computed(() => {
	const hosts = new Set()
	const allLinks = activeTab.value === 'linkgrabber' ? linkGrabberLinks.value : downloadLinks.value
	allLinks.forEach((link) => {
		if (link.host) hosts.add(link.host)
	})
	return Array.from(hosts).sort()
})

const activeFilterCount = computed(() => {
	let count = 0
	if (searchQuery.value) count++
	if (statusFilter.value) count++
	if (hostFilter.value) count++
	if (enabledFilter.value) count++
	if (sortBy.value !== 'added-desc') count++
	return count
})

const clearAllFilters = () => {
	searchQuery.value = ''
	statusFilter.value = ''
	hostFilter.value = ''
	enabledFilter.value = ''
	sortBy.value = 'added-desc'
}

// Used in template tab badges
// eslint-disable-next-line no-unused-vars
const totalLinkGrabberLinks = computed(() => linkGrabberLinks.value.length)
// eslint-disable-next-line no-unused-vars
const totalDownloadLinks = computed(() => downloadLinks.value.length)

const selectedLinkGrabberItems = computed(() => {
	return [...selectedLinkGrabberPackages.value, ...selectedLinkGrabberLinks.value]
})

const selectedDownloadItems = computed(() => {
	return [...selectedDownloadPackages.value, ...selectedDownloadLinks.value]
})

const activeDownloads = computed(() => {
	return downloadLinks.value.filter((l) => l.running && !l.finished).length
})

const completedDownloads = computed(() => {
	return downloadLinks.value.filter((l) => l.finished).length
})

const totalDownloadSize = computed(() => {
	return downloadPackages.value.reduce((sum, pkg) => sum + (pkg.bytesTotal || 0), 0)
})

const totalBytesLoaded = computed(() => {
	return downloadPackages.value.reduce((sum, pkg) => sum + (pkg.bytesLoaded || 0), 0)
})

const totalBytesTotal = computed(() => {
	return downloadPackages.value.reduce((sum, pkg) => sum + (pkg.bytesTotal || 0), 0)
})

const overallProgress = computed(() => {
	if (totalBytesTotal.value === 0) return 0
	return Math.round((totalBytesLoaded.value / totalBytesTotal.value) * 100)
})

const overallETA = computed(() => {
	if (currentSpeed.value <= 0) return null
	const remaining = totalBytesTotal.value - totalBytesLoaded.value
	if (remaining <= 0) return null
	const seconds = remaining / currentSpeed.value
	return formatETA(seconds * 1000) // Convert to ms
})

const linksCount = computed(() => {
	if (!newLinksText.value.trim()) return 0
	return newLinksText.value
		.trim()
		.split('\n')
		.filter((l) => l.trim()).length
})

// Methods
const showToast = (message, type = 'info') => {
	// Limit max toasts to 5
	if (toasts.value.length >= 5) {
		toasts.value.shift()
	}
	const id = ++toastId
	toasts.value.push({ id, message, type })
	setTimeout(() => {
		toasts.value = toasts.value.filter((t) => t.id !== id)
	}, 5000)
}

const getToastIcon = (type) => {
	switch (type) {
		case 'success':
			return 'check-circle'
		case 'error':
			return 'exclamation-circle'
		case 'warning':
			return 'exclamation-triangle'
		default:
			return 'info-circle'
	}
}

const refreshAll = async () => {
	loading.value = true
	try {
		await Promise.all([checkConnection(), loadPackageTimestamps(), loadSpeed()])
		await Promise.all([loadLinkGrabber(), loadDownloads()])
	} finally {
		loading.value = false
	}
}

const checkConnection = async () => {
	try {
		const status = await jdownloaderAPI.getStatus()
		connectionStatus.value = status
	} catch (error) {
		connectionStatus.value = { connected: false }
	}
}

const loadSpeed = async () => {
	try {
		const speed = await jdownloaderAPI.getSpeed()
		currentSpeed.value = speed.speed || 0
		speedLimit.value = speed.limit || 0
		selectedSpeedLimit.value = speed.limit || 0
	} catch (error) {
		console.error('Failed to load speed:', error)
	}
}

// Merge locally-tracked timestamps into packages by matching package name
const mergeTimestamps = (packages) => {
	const ts = packageTimestamps.value
	return packages.map((pkg) => {
		if (pkg.name && ts[pkg.name]) {
			return { ...pkg, localAddedAt: ts[pkg.name] }
		}
		return pkg
	})
}

const loadPackageTimestamps = async () => {
	try {
		const res = await jdownloaderAPI.getPackageTimestamps()
		packageTimestamps.value = res || {}
	} catch {
		// Non-critical, ignore
	}
}

const loadLinkGrabber = async () => {
	try {
		const [packagesRes, linksRes] = await Promise.all([jdownloaderAPI.getLinkGrabberPackages(), jdownloaderAPI.getLinkGrabberLinks()])
		linkGrabberPackages.value = mergeTimestamps((packagesRes?.data || []).filter((p) => p && p.uuid))
		linkGrabberLinks.value = (linksRes?.data || []).filter((l) => l && l.uuid)
	} catch (error) {
		console.error('Failed to load LinkGrabber:', error)
		linkGrabberPackages.value = []
		linkGrabberLinks.value = []
	}
}

const loadDownloads = async () => {
	try {
		const [packagesRes, linksRes] = await Promise.all([jdownloaderAPI.getDownloadPackages(), jdownloaderAPI.getDownloadLinks()])
		const newPackages = mergeTimestamps((packagesRes?.data || []).filter((p) => p && p.uuid))
		const newLinks = (linksRes?.data || []).filter((l) => l && l.uuid)

		// On initial load, just populate the finished set without showing toasts
		if (isInitialLoad.value) {
			newPackages.forEach((pkg) => {
				if (pkg.finished) {
					previouslyFinishedPackages.value.add(pkg.uuid)
				}
			})
			isInitialLoad.value = false
		} else {
			// Check for newly finished packages (only show toast for truly new completions)
			newPackages.forEach((pkg) => {
				if (pkg.finished && !previouslyFinishedPackages.value.has(pkg.uuid)) {
					showToast(`Download complete: ${pkg.name || 'Package'}`, 'success')
					previouslyFinishedPackages.value.add(pkg.uuid)
				}
			})
		}

		downloadPackages.value = newPackages
		downloadLinks.value = newLinks
	} catch (error) {
		console.error('Failed to load Downloads:', error)
		downloadPackages.value = []
		downloadLinks.value = []
	}
}

const loadHistory = async () => {
	try {
		const result = await jdownloaderAPI.getHistory(100, 0)
		history.value = result.items || result || []
	} catch (error) {
		console.error('Failed to load history:', error)
		history.value = []
	}
}

const updateRefreshInterval = () => {
	if (refreshInterval) {
		clearInterval(refreshInterval)
		refreshInterval = null
	}
	if (refreshIntervalSeconds.value > 0) {
		refreshInterval = setInterval(async () => {
			if (connectionStatus.value.connected) {
				await Promise.all([loadSpeed(), loadDownloads()])
			}
		}, refreshIntervalSeconds.value * 1000)
	}
}

// Add Links
const addLinks = async () => {
	if (!newLinksText.value.trim()) return

	addingLinks.value = true
	try {
		const urls = newLinksText.value
			.trim()
			.split('\n')
			.filter((l) => l.trim())
		// Build the full download path
		const downloadPath = newLinksPath.value.trim() ? `/output/${newLinksPath.value.trim().replace(/^\/+/, '')}` : '/output'

		const links = urls.map((url) => ({
			url: url.trim(),
			packageName: newLinksPackageName.value || undefined,
			downloadPath: downloadPath,
			autoStart: newLinksAutoStart.value,
		}))

		await jdownloaderAPI.sendLinks(links)
		showToast(`Added ${links.length} link(s) to JDownloader`, 'success')

		// Save path to history before clearing
		savePathToHistory(newLinksPath.value.trim())

		newLinksText.value = ''
		newLinksPath.value = ''
		newLinksPackageName.value = ''
		showAddLinksModal.value = false

		await loadPackageTimestamps()
		await loadLinkGrabber()
	} catch (error) {
		console.error('Failed to add links:', error)
		showToast('Failed to add links: ' + error.message, 'error')
	} finally {
		addingLinks.value = false
	}
}

// Download path history helpers
const filteredPathHistory = computed(() => {
	const query = (newLinksPath.value || '').toLowerCase()
	if (!query) return downloadPathHistory.value
	return downloadPathHistory.value.filter((p) => p.toLowerCase().includes(query))
})

const savePathToHistory = (path) => {
	if (!path) return
	const history = downloadPathHistory.value.filter((p) => p !== path)
	history.unshift(path) // Most recent first
	if (history.length > 20) history.length = 20
	downloadPathHistory.value = history
	localStorage.setItem(STORAGE_KEY, JSON.stringify(history))
}

const selectPath = (path) => {
	newLinksPath.value = path
	showPathDropdown.value = false
}

const clearPathHistory = () => {
	downloadPathHistory.value = []
	localStorage.removeItem(STORAGE_KEY)
	showPathDropdown.value = false
}

const hidePathDropdownDelayed = () => {
	setTimeout(() => {
		showPathDropdown.value = false
	}, 150)
}

// Export Links
const exportLinks = () => {
	const links = activeTab.value === 'linkgrabber' ? linkGrabberLinks.value : downloadLinks.value
	const urls = links.map((l) => l.url).filter(Boolean)
	exportedLinksText.value = urls.join('\n')
	showExportModal.value = true
}

const copyExportedLinks = async () => {
	try {
		await navigator.clipboard.writeText(exportedLinksText.value)
		showToast('Links copied to clipboard', 'success')
	} catch (error) {
		showToast('Failed to copy links', 'error')
	}
}

const downloadExportedLinks = () => {
	const blob = new Blob([exportedLinksText.value], { type: 'text/plain' })
	const url = URL.createObjectURL(blob)
	const a = document.createElement('a')
	a.href = url
	a.download = `jdownloader-links-${new Date().toISOString().slice(0, 10)}.txt`
	a.click()
	URL.revokeObjectURL(url)
	showToast('Links downloaded', 'success')
}

// Download Controls
const startDownloads = async () => {
	try {
		await jdownloaderAPI.startDownloads()
		showToast('Downloads started', 'success')
		await loadDownloads()
	} catch (error) {
		console.error('Failed to start downloads:', error)
		showToast('Failed to start downloads', 'error')
	}
}

const stopDownloads = async () => {
	try {
		await jdownloaderAPI.stopDownloads()
		showToast('Downloads stopped', 'info')
		await loadDownloads()
	} catch (error) {
		console.error('Failed to stop downloads:', error)
		showToast('Failed to stop downloads', 'error')
	}
}

const pauseDownloads = async (pause) => {
	try {
		await jdownloaderAPI.pauseDownloads(pause)
		showToast(pause ? 'Downloads paused' : 'Downloads resumed', 'info')
		await loadDownloads()
	} catch (error) {
		console.error('Failed to pause downloads:', error)
	}
}

const setSpeedLimit = async () => {
	try {
		await jdownloaderAPI.setSpeedLimit(selectedSpeedLimit.value)
		speedLimit.value = selectedSpeedLimit.value
		showToast(selectedSpeedLimit.value > 0 ? `Speed limit set to ${formatSpeed(selectedSpeedLimit.value)}` : 'Speed limit removed', 'info')
	} catch (error) {
		console.error('Failed to set speed limit:', error)
	}
}

const cleanupFinished = async () => {
	try {
		await jdownloaderAPI.cleanupFinished()
		showToast('Finished downloads cleaned up', 'success')
		previouslyFinishedPackages.value.clear()
		await loadDownloads()
	} catch (error) {
		console.error('Failed to cleanup finished:', error)
	}
}

// Priority
const setPriority = async (type, packageId, priority) => {
	try {
		await jdownloaderAPI.setLinkPriority(priority, [], [packageId], type === 'dl')
		showToast(`Priority set to ${priority}`, 'info')
		if (type === 'lg') {
			await loadLinkGrabber()
		} else {
			await loadDownloads()
		}
	} catch (error) {
		console.error('Failed to set priority:', error)
		showToast('Failed to set priority', 'error')
	}
}

// Enable/Disable
const togglePackageEnabled = async (type, pkg) => {
	try {
		const enabled = pkg.enabled === false
		await jdownloaderAPI.setLinkEnabled(enabled, [], [pkg.uuid], type === 'dl')
		showToast(enabled ? 'Package enabled' : 'Package disabled', 'info')
		if (type === 'lg') {
			await loadLinkGrabber()
		} else {
			await loadDownloads()
		}
	} catch (error) {
		console.error('Failed to toggle enabled:', error)
	}
}

const toggleLinkEnabled = async (type, link) => {
	try {
		const enabled = link.enabled === false
		await jdownloaderAPI.setLinkEnabled(enabled, [link.uuid], [], type === 'dl')
		if (type === 'lg') {
			await loadLinkGrabber()
		} else {
			await loadDownloads()
		}
	} catch (error) {
		console.error('Failed to toggle link enabled:', error)
	}
}

// Package rename
const startEditPackage = (pkg) => {
	editingPackage.value = pkg.uuid
	editingPackageName.value = pkg.name || ''
	nextTick(() => {
		const input = document.querySelector('.package-name-input')
		if (input) input.focus()
	})
}

const savePackageName = async (type, packageId) => {
	if (!editingPackageName.value.trim()) {
		cancelEditPackage()
		return
	}

	try {
		await jdownloaderAPI.renamePackage(packageId, editingPackageName.value, type === 'dl')
		showToast('Package renamed', 'success')
		if (type === 'lg') {
			await loadLinkGrabber()
		} else {
			await loadDownloads()
		}
	} catch (error) {
		console.error('Failed to rename package:', error)
		showToast('Failed to rename package', 'error')
	} finally {
		editingPackage.value = null
		editingPackageName.value = ''
	}
}

const cancelEditPackage = () => {
	editingPackage.value = null
	editingPackageName.value = ''
}

// Context Menu
const showContextMenu = (event, type, item) => {
	contextMenu.show = true
	contextMenu.x = event.clientX
	contextMenu.y = event.clientY
	contextMenu.type = type
	contextMenu.item = item
}

const hideContextMenu = () => {
	contextMenu.show = false
	contextMenu.item = null
}

const contextMenuAction = async (action) => {
	const item = contextMenu.item
	const type = contextMenu.type
	hideContextMenu()

	if (!item) return

	const isLink = type.includes('-link')
	const isDownload = type.includes('dl')

	switch (action) {
		case 'rename':
			if (!isLink) startEditPackage(item)
			break
		case 'copy-url':
			if (item.url) {
				await navigator.clipboard.writeText(item.url)
				showToast('URL copied', 'success')
			}
			break
		case 'priority-highest':
		case 'priority-high':
		case 'priority-default':
		case 'priority-low': {
			const priority = action.split('-')[1].toUpperCase()
			if (isLink) {
				await jdownloaderAPI.setLinkPriority(priority, [item.uuid], [], isDownload)
			} else {
				await jdownloaderAPI.setLinkPriority(priority, [], [item.uuid], isDownload)
			}
			showToast(`Priority set to ${priority}`, 'info')
			await (isDownload ? loadDownloads() : loadLinkGrabber())
			break
		}
		case 'move':
			if (isLink) {
				await jdownloaderAPI.moveToDownloads([item.uuid], [])
			} else {
				await jdownloaderAPI.moveToDownloads([], [item.uuid])
			}
			await Promise.all([loadLinkGrabber(), loadDownloads()])
			break
		case 'force':
			if (isLink) {
				await jdownloaderAPI.forceDownload([item.uuid], [])
			} else {
				await jdownloaderAPI.forceDownload([], [item.uuid])
			}
			showToast('Force download started', 'info')
			await loadDownloads()
			break
		case 'toggle-enabled':
			if (isLink) {
				await toggleLinkEnabled(isDownload ? 'dl' : 'lg', item)
			} else {
				await togglePackageEnabled(isDownload ? 'dl' : 'lg', item)
			}
			break
		case 'remove':
			if (isDownload) {
				if (isLink) {
					await jdownloaderAPI.removeDownloadLinks([item.uuid], [])
				} else {
					await jdownloaderAPI.removeDownloadLinks([], [item.uuid])
				}
				await loadDownloads()
			} else {
				if (isLink) {
					await jdownloaderAPI.removeLinkGrabberLinks([item.uuid], [])
				} else {
					await jdownloaderAPI.removeLinkGrabberLinks([], [item.uuid])
				}
				await loadLinkGrabber()
			}
			break
	}
}

// Keyboard shortcuts
const handleKeyDown = (event) => {
	// Ignore if typing in input
	if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') return

	const ctrlOrMeta = event.ctrlKey || event.metaKey

	switch (event.key) {
		case ' ': // Space - Pause
			event.preventDefault()
			pauseDownloads(true)
			break
		case 'Delete':
			event.preventDefault()
			if (activeTab.value === 'linkgrabber' && selectedLinkGrabberItems.value.length > 0) {
				removeSelectedFromLinkGrabber()
			} else if (activeTab.value === 'downloads' && selectedDownloadItems.value.length > 0) {
				removeSelectedFromDownloads()
			}
			break
		case 'a':
			if (ctrlOrMeta) {
				event.preventDefault()
				selectAll()
			}
			break
		case 'e':
			if (ctrlOrMeta) {
				event.preventDefault()
				exportLinks()
			}
			break
		case 's':
			if (ctrlOrMeta) {
				event.preventDefault()
				startDownloads()
			}
			break
		case 'x':
			if (ctrlOrMeta) {
				event.preventDefault()
				stopDownloads()
			}
			break
		case 'Escape':
			hideContextMenu()
			cancelEditPackage()
			break
	}
}

// Select all
const selectAll = () => {
	if (activeTab.value === 'linkgrabber') {
		filteredLinkGrabberPackages.value.forEach((pkg) => {
			selectedLinkGrabberPackages.value.add(pkg.uuid)
		})
	} else if (activeTab.value === 'downloads') {
		filteredDownloadPackages.value.forEach((pkg) => {
			selectedDownloadPackages.value.add(pkg.uuid)
		})
	}
}

// LinkGrabber Actions
const moveAllToDownloads = async () => {
	try {
		const packageIds = filteredLinkGrabberPackages.value.map((p) => p.uuid)
		await jdownloaderAPI.moveToDownloads([], packageIds)
		showToast('All packages moved to downloads', 'success')
		await Promise.all([loadLinkGrabber(), loadDownloads()])
	} catch (error) {
		console.error('Failed to move to downloads:', error)
	}
}

const moveSelectedToDownloads = async () => {
	try {
		const linkIds = [...selectedLinkGrabberLinks.value]
		const packageIds = [...selectedLinkGrabberPackages.value]
		await jdownloaderAPI.moveToDownloads(linkIds, packageIds)
		selectedLinkGrabberLinks.value.clear()
		selectedLinkGrabberPackages.value.clear()
		showToast('Selected items moved to downloads', 'success')
		await Promise.all([loadLinkGrabber(), loadDownloads()])
	} catch (error) {
		console.error('Failed to move selected to downloads:', error)
	}
}

const movePackageToDownloads = async (packageId) => {
	try {
		await jdownloaderAPI.moveToDownloads([], [packageId])
		await Promise.all([loadLinkGrabber(), loadDownloads()])
	} catch (error) {
		console.error('Failed to move package to downloads:', error)
	}
}

const removeSelectedFromLinkGrabber = async () => {
	try {
		const linkIds = [...selectedLinkGrabberLinks.value]
		const packageIds = [...selectedLinkGrabberPackages.value]
		await jdownloaderAPI.removeLinkGrabberLinks(linkIds, packageIds)
		selectedLinkGrabberLinks.value.clear()
		selectedLinkGrabberPackages.value.clear()
		showToast('Items removed', 'info')
		await loadLinkGrabber()
	} catch (error) {
		console.error('Failed to remove from LinkGrabber:', error)
	}
}

const removePackage = async (type, packageId) => {
	try {
		if (type === 'lg') {
			await jdownloaderAPI.removeLinkGrabberLinks([], [packageId])
			await loadLinkGrabber()
		} else {
			await jdownloaderAPI.removeDownloadLinks([], [packageId])
			await loadDownloads()
		}
	} catch (error) {
		console.error('Failed to remove package:', error)
	}
}

const clearLinkGrabber = async () => {
	if (!confirm('Are you sure you want to clear all items from LinkGrabber?')) return
	try {
		await jdownloaderAPI.clearLinkGrabber()
		showToast('LinkGrabber cleared', 'info')
		await loadLinkGrabber()
	} catch (error) {
		console.error('Failed to clear LinkGrabber:', error)
	}
}

// Downloads Actions
const forceDownloadSelected = async () => {
	try {
		const linkIds = [...selectedDownloadLinks.value]
		const packageIds = [...selectedDownloadPackages.value]
		await jdownloaderAPI.forceDownload(linkIds, packageIds)
		selectedDownloadLinks.value.clear()
		selectedDownloadPackages.value.clear()
		showToast('Forcing download of selected items', 'info')
		await loadDownloads()
	} catch (error) {
		console.error('Failed to force download:', error)
	}
}

const resetSelectedLinks = async () => {
	try {
		const linkIds = [...selectedDownloadLinks.value]
		const packageIds = [...selectedDownloadPackages.value]
		await jdownloaderAPI.resetLinks(linkIds, packageIds)
		selectedDownloadLinks.value.clear()
		selectedDownloadPackages.value.clear()
		showToast('Links reset', 'info')
		await loadDownloads()
	} catch (error) {
		console.error('Failed to reset links:', error)
	}
}

const removeSelectedFromDownloads = async () => {
	try {
		const linkIds = [...selectedDownloadLinks.value]
		const packageIds = [...selectedDownloadPackages.value]
		await jdownloaderAPI.removeDownloadLinks(linkIds, packageIds)
		selectedDownloadLinks.value.clear()
		selectedDownloadPackages.value.clear()
		showToast('Items removed', 'info')
		await loadDownloads()
	} catch (error) {
		console.error('Failed to remove from Downloads:', error)
	}
}

// History
const clearHistory = async () => {
	if (!confirm('Are you sure you want to clear the history?')) return
	try {
		await jdownloaderAPI.clearHistory()
		history.value = []
		showToast('History cleared', 'info')
	} catch (error) {
		console.error('Failed to clear history:', error)
	}
}

// Selection helpers
const togglePackage = (type, packageId) => {
	if (type === 'lg') {
		expandedPackages.lg[packageId] = !expandedPackages.lg[packageId]
	} else {
		expandedPackages.dl[packageId] = !expandedPackages.dl[packageId]
	}
}

const isPackageSelected = (type, packageId) => {
	if (type === 'lg') {
		return selectedLinkGrabberPackages.value.has(packageId)
	}
	return selectedDownloadPackages.value.has(packageId)
}

const togglePackageSelection = (type, packageId) => {
	if (type === 'lg') {
		if (selectedLinkGrabberPackages.value.has(packageId)) {
			selectedLinkGrabberPackages.value.delete(packageId)
		} else {
			selectedLinkGrabberPackages.value.add(packageId)
		}
	} else {
		if (selectedDownloadPackages.value.has(packageId)) {
			selectedDownloadPackages.value.delete(packageId)
		} else {
			selectedDownloadPackages.value.add(packageId)
		}
	}
}

const isLinkSelected = (type, linkId) => {
	if (type === 'lg') {
		return selectedLinkGrabberLinks.value.has(linkId)
	}
	return selectedDownloadLinks.value.has(linkId)
}

const toggleLinkSelection = (type, linkId) => {
	if (type === 'lg') {
		if (selectedLinkGrabberLinks.value.has(linkId)) {
			selectedLinkGrabberLinks.value.delete(linkId)
		} else {
			selectedLinkGrabberLinks.value.add(linkId)
		}
	} else {
		if (selectedDownloadLinks.value.has(linkId)) {
			selectedDownloadLinks.value.delete(linkId)
		} else {
			selectedDownloadLinks.value.add(linkId)
		}
	}
}

const getLinkGrabberLinksForPackage = (packageId) => {
	return linkGrabberLinks.value.filter((link) => link.packageUUID === packageId)
}

const getDownloadLinksForPackage = (packageId) => {
	return downloadLinks.value.filter((link) => link.packageUUID === packageId)
}

// Formatting helpers
const formatSpeed = (bytesPerSecond) => {
	if (!bytesPerSecond || bytesPerSecond === 0) return '0 B/s'
	const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
	let i = 0
	let speed = bytesPerSecond
	while (speed >= 1024 && i < units.length - 1) {
		speed /= 1024
		i++
	}
	return `${speed.toFixed(1)} ${units[i]}`
}

const formatSize = (bytes) => {
	if (!bytes || bytes === 0) return '0 B'
	const units = ['B', 'KB', 'MB', 'GB', 'TB']
	let i = 0
	let size = bytes
	while (size >= 1024 && i < units.length - 1) {
		size /= 1024
		i++
	}
	return `${size.toFixed(1)} ${units[i]}`
}

const formatDate = (dateString) => {
	if (!dateString) return ''
	const date = new Date(dateString)
	return date.toLocaleString()
}

const formatETA = (milliseconds) => {
	if (!milliseconds || milliseconds <= 0) return ''
	const seconds = Math.floor(milliseconds / 1000)
	if (seconds < 60) return `${seconds}s`
	const minutes = Math.floor(seconds / 60)
	if (minutes < 60) return `${minutes}m ${seconds % 60}s`
	const hours = Math.floor(minutes / 60)
	if (hours < 24) return `${hours}h ${minutes % 60}m`
	const days = Math.floor(hours / 24)
	return `${days}d ${hours % 24}h`
}

const formatPath = (path) => {
	if (!path) return ''
	// Show last part of path for brevity
	const parts = path.replace(/\\/g, '/').split('/')
	if (parts.length <= 2) return path
	return '.../' + parts.slice(-2).join('/')
}

const getProgressPercent = (item) => {
	if (!item.bytesTotal || item.bytesTotal === 0) return 0
	return Math.round(((item.bytesLoaded || 0) / item.bytesTotal) * 100)
}

const getLinkProgressPercent = (link) => {
	if (!link.bytesTotal || link.bytesTotal === 0) return 0
	return Math.round(((link.bytesLoaded || 0) / link.bytesTotal) * 100)
}

const getProgressClass = (item) => {
	if (item.finished) return 'bg-success'
	const percent = getProgressPercent(item)
	if (percent >= 100) return 'bg-success'
	if (item.speed > 0 || item.running) return 'bg-primary progress-bar-striped progress-bar-animated'
	return 'bg-info'
}

const getAvailabilityClass = (availability) => {
	if (!availability) return ''
	const lower = availability.toLowerCase()
	if (lower.includes('online') || lower.includes('available')) return 'online'
	if (lower.includes('offline') || lower.includes('unavailable')) return 'offline'
	return 'unknown'
}

const getStatusClass = (link) => {
	if (link.finished) return 'success'
	if (link.running) return 'primary'
	if (link.skipped) return 'warning'
	const status = link.status?.toLowerCase() || ''
	if (status.includes('error') || status.includes('failed')) return 'danger'
	if (status.includes('waiting') || status.includes('queued')) return 'warning'
	return 'secondary'
}

const getHistoryStatusClass = (status) => {
	if (!status) return 'secondary'
	const lower = status.toLowerCase()
	if (lower === 'sent') return 'success'
	if (lower === 'failed') return 'danger'
	if (lower === 'queued') return 'warning'
	return 'secondary'
}

// Lifecycle
onMounted(async () => {
	await refreshAll()
	updateRefreshInterval()

	// Focus page for keyboard shortcuts
	if (pageRef.value) {
		pageRef.value.focus()
	}

	// Close context menu on click outside
	document.addEventListener('click', hideContextMenu)
})

onUnmounted(() => {
	if (refreshInterval) {
		clearInterval(refreshInterval)
	}
	document.removeEventListener('click', hideContextMenu)
})

// Reset download pagination when filters change
watch([searchQuery, statusFilter, hostFilter, enabledFilter, sortBy], () => {
	downloadCurrentPage.value = 1
})

// Watch for refresh interval changes
watch(refreshIntervalSeconds, updateRefreshInterval)
</script>

<style scoped>
@import '@/styles/pages/jdownloader_page.css';

/* Filter Bar */
.filter-bar {
	background: rgba(255, 255, 255, 0.05);
	padding: 0.75rem 1rem;
	border-radius: 8px;
	border: 1px solid rgba(255, 255, 255, 0.1);
}

.filter-bar .input-group-text {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.2);
	color: rgba(255, 255, 255, 0.6);
}

.filter-bar .form-control,
.filter-bar .form-select {
	background: rgba(255, 255, 255, 0.1);
	border-color: rgba(255, 255, 255, 0.2);
	color: #fff;
}

.filter-bar .form-control:focus,
.filter-bar .form-select:focus {
	background: rgba(255, 255, 255, 0.15);
	border-color: #2196f3;
	box-shadow: none;
}

/* Keyboard hints */
.keyboard-hints {
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
}

.keyboard-hints kbd {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.2);
	border-radius: 4px;
	padding: 0.1rem 0.4rem;
	margin-right: 0.25rem;
	font-size: 0.7rem;
}

/* Package name input */
.package-name-input {
	width: 300px;
	background: rgba(255, 255, 255, 0.15) !important;
	border-color: #2196f3 !important;
	color: #fff !important;
}

/* Package path display */
.package-path {
	max-width: 200px;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
	font-size: 0.75rem;
	color: rgba(255, 255, 255, 0.5);
}

/* Path history dropdown */
.path-history-dropdown {
	position: absolute;
	top: 100%;
	left: 0;
	right: 0;
	z-index: 1050;
	background: #1e1e2e;
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-top: none;
	border-radius: 0 0 6px 6px;
	max-height: 200px;
	overflow-y: auto;
}

.path-history-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 0.4rem 0.75rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.path-history-item {
	padding: 0.4rem 0.75rem;
	cursor: pointer;
	font-size: 0.85rem;
	color: rgba(255, 255, 255, 0.8);
	transition: background 0.15s;
}

.path-history-item:hover {
	background: rgba(255, 255, 255, 0.1);
}

/* Pagination */
.pagination-controls {
	padding: 0.75rem;
	background: rgba(255, 255, 255, 0.03);
	border-radius: 8px;
}

.pagination-info {
	background: rgba(255, 255, 255, 0.03);
	padding: 0.5rem 1rem;
	border-radius: 8px;
}

/* Link disabled state */
.link-disabled {
	opacity: 0.5;
}

.link-disabled .link-name {
	text-decoration: line-through;
}

/* Context Menu */
.context-menu {
	position: fixed;
	z-index: 1060;
	background: #1e1e2e;
	border: 1px solid rgba(255, 255, 255, 0.15);
	border-radius: 8px;
	min-width: 200px;
	box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
	padding: 0.5rem 0;
}

.context-menu-item {
	padding: 0.5rem 1rem;
	cursor: pointer;
	color: rgba(255, 255, 255, 0.9);
	font-size: 0.875rem;
	transition: background 0.15s;
}

.context-menu-item:hover {
	background: rgba(255, 255, 255, 0.1);
}

.context-menu-item.text-danger {
	color: #f44336;
}

.context-menu-divider {
	height: 1px;
	background: rgba(255, 255, 255, 0.1);
	margin: 0.5rem 0;
}

/* Dropdown menu dark */
.dropdown-menu-dark {
	background: #1e1e2e;
	border-color: rgba(255, 255, 255, 0.15);
	z-index: 9999;
}

.dropdown-menu-dark .dropdown-item {
	color: rgba(255, 255, 255, 0.9);
}

.dropdown-menu-dark .dropdown-item:hover {
	background: rgba(255, 255, 255, 0.1);
}

/* Modal Styles */
.modal-backdrop {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.7);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1050;
}

.modal-content {
	background: #1a1a2e;
	border-radius: 12px;
	border: 1px solid rgba(255, 255, 255, 0.1);
	width: 90%;
	max-width: 600px;
	max-height: 90vh;
	overflow: auto;
}

.modal-header {
	padding: 1rem 1.25rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.modal-title {
	margin: 0;
	color: #fff;
}

.btn-close {
	background: transparent;
	border: none;
	color: rgba(255, 255, 255, 0.6);
	font-size: 1.5rem;
	cursor: pointer;
	padding: 0;
	line-height: 1;
}

.btn-close:hover {
	color: #fff;
}

.btn-close::before {
	content: '×';
}

.modal-body {
	padding: 1.25rem;
}

.modal-footer {
	padding: 1rem 1.25rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	display: flex;
	justify-content: flex-end;
	gap: 0.5rem;
}

.form-label {
	color: rgba(255, 255, 255, 0.8);
}

.form-control {
	background: rgba(255, 255, 255, 0.1);
	border: 1px solid rgba(255, 255, 255, 0.2);
	color: #fff;
}

.form-control:focus {
	background: rgba(255, 255, 255, 0.15);
	border-color: #2196f3;
	color: #fff;
	box-shadow: 0 0 0 0.25rem rgba(33, 150, 243, 0.25);
}

.form-control::placeholder {
	color: rgba(255, 255, 255, 0.4);
}

.form-check-label {
	color: rgba(255, 255, 255, 0.8);
}

/* Toast Notifications */
.toast-container {
	position: fixed;
	bottom: 20px;
	right: 20px;
	z-index: 1100;
	display: flex;
	flex-direction: column;
	gap: 10px;
}

.toast-notification {
	padding: 12px 20px;
	border-radius: 8px;
	color: #fff;
	display: flex;
	align-items: center;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
	min-width: 250px;
}

.toast-notification.success {
	background: linear-gradient(135deg, #28a745, #1e7e34);
}

.toast-notification.error {
	background: linear-gradient(135deg, #dc3545, #bd2130);
}

.toast-notification.warning {
	background: linear-gradient(135deg, #ffc107, #d39e00);
	color: #212529;
}

.toast-notification.info {
	background: linear-gradient(135deg, #17a2b8, #117a8b);
}

.toast-enter-active,
.toast-leave-active {
	transition: all 0.3s ease;
}

.toast-enter-from,
.toast-leave-to {
	opacity: 0;
	transform: translateX(100px);
}

/* Overall Progress */
.overall-progress {
	background: rgba(255, 255, 255, 0.05);
	padding: 1rem;
	border-radius: 8px;
	border: 1px solid rgba(255, 255, 255, 0.1);
}

/* Package finished state */
.package-finished {
	border-color: rgba(40, 167, 69, 0.3) !important;
}

.package-finished .package-header {
	background: rgba(40, 167, 69, 0.1);
}

.link-finished {
	background: rgba(40, 167, 69, 0.05);
}

/* Focus outline for keyboard nav */
.jdownloader-page:focus {
	outline: none;
}
</style>
