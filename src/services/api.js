import axios from 'axios'

// Resolve the API base URL. In production the web server injects window.VSAI_API_URL
// into index.html before the bundle loads. In development it falls back to localhost:8080.
export const API_ORIGIN = window.VSAI_API_URL || 'http://localhost:8080'

// Helper to build full API URLs (use for non-axios requests like fetch or <img src>)
export const getApiBase = () => API_ORIGIN

// Create axios instance with base configuration
const api = axios.create({
	baseURL: `${API_ORIGIN}/api/v1`,
	headers: {
		'Content-Type': 'application/json',
	},
	timeout: 60000, // 60 seconds
	paramsSerializer: {
		indexes: null, // Use repeat format for arrays: tag_ids=1&tag_ids=2
	},
})

// Request interceptor
api.interceptors.request.use(
	(config) => {
		// Add any auth tokens here if needed
		return config
	},
	(error) => {
		return Promise.reject(error)
	}
)

// Response interceptor
api.interceptors.response.use(
	(response) => {
		return response.data
	},
	(error) => {
		console.error('API Error:', error)
		return Promise.reject(error)
	}
)

// API endpoints
export const librariesAPI = {
	getAll: (signal) => api.get('/libraries', { signal }),
	getPrimary: (signal) => api.get('/libraries/primary', { signal }),
	getById: (id, signal) => api.get(`/libraries/${id}`, { signal }),
	create: (data, signal) => api.post('/libraries', data, { signal }),
	update: (id, data, signal) => api.put(`/libraries/${id}`, data, { signal }),
	delete: (id, signal) => api.delete(`/libraries/${id}`, { signal }),
	browse: (id, path = '', metadata = false, signal) => api.get(`/libraries/${id}/browse`, { params: { path, metadata: metadata ? 'true' : 'false' }, signal }),
	generateThumbnails: (id, path = '', signal) => api.post(`/libraries/${id}/generate-thumbnails`, null, { params: { path }, signal }),
	analyzeStructure: (id, signal) => api.get(`/libraries/${id}/analyze-structure`, { signal }),
	applyStructure: (id, plan, signal) => api.post(`/libraries/${id}/apply-structure`, plan, { signal }),
}

export const performersAPI = {
	getAll: (searchTerm = '') => {
		const params = searchTerm ? { search: searchTerm } : {}
		return api.get('/performers', { params })
	},
	search: (params) => api.get('/performers', { params }),
	getById: (id) => api.get(`/performers/${id}`),
	getPreviews: (id) => api.get(`/performers/${id}/previews`),
	create: (data) => api.post('/performers', data),
	scan: () => api.post('/performers/scan'),
	update: (id, data) => api.put(`/performers/${id}`, data),
	delete: (id) => api.delete(`/performers/${id}`),
	fetchMetadata: (id) => api.post(`/performers/${id}/fetch-metadata`),
	resetMetadata: (id) => api.post(`/performers/${id}/reset-metadata`),
	resetPreviews: (id) => api.post(`/performers/${id}/reset-previews`),
	generateThumbnails: () => api.post('/performers/generate-thumbnails'),
	generateThumbnail: (id) => api.post(`/performers/${id}/generate-thumbnail`),
	getTags: (id) => api.get(`/performers/${id}/tags`),
	addTag: (id, tagId) => api.post(`/performers/${id}/tags`, { tag_id: tagId }),
	removeTag: (id, tagId) => api.delete(`/performers/${id}/tags/${tagId}`),
	syncTags: (id) => api.post(`/performers/${id}/sync-tags`),
	getVideos: (id) => api.get(`/performers/${id}/videos`),
}

export const videosAPI = {
	getAll: (params, signal) => api.get('/videos', { params, signal }),
	getById: (id, signal) => api.get(`/videos/${id}`, { signal }),
	search: (params, signal) => api.get('/videos/search', { params, signal }),
	create: (data, signal) => api.post('/videos', data, { signal }),
	update: (id, data, signal) => api.put(`/videos/${id}`, data, { signal }),
	delete: (id, signal) => api.delete(`/videos/${id}`, { signal }),
	scan: (libraryId, signal) => api.post('/videos/scan', { library_id: libraryId }, { signal }),
	scanAllParallel: (config = {}, signal) => api.post('/videos/scan-all-parallel', config, { signal }),
	generatePreviews: (config = {}, signal) => api.post('/videos/generate-previews', config, { signal }),
	generateThumbnails: (signal) => api.post('/videos/generate-thumbnails', {}, { signal }),
	runPathAnalysis: (libraryId = 0, onlyUnlinked = false, dryRun = false, signal) =>
		api.post('/videos/path-analysis', { library_id: libraryId, only_unlinked: onlyUnlinked, dry_run: dryRun }, { signal }),
	analyzePath: (id, signal) => api.post(`/videos/${id}/analyze-path`, {}, { signal }),
	fetchMetadata: (id, signal) => api.post(`/videos/${id}/fetch`, {}, { signal }),
	addTags: (id, tagIds, signal) => api.post(`/videos/${id}/tags`, { tag_ids: tagIds }, { signal }),
	removeTags: (id, tagIds, signal) => api.delete(`/videos/${id}/tags`, { data: { tag_ids: tagIds }, signal }),
	bulk: (operation, videoIds, data = {}, signal) => api.post('/videos/bulk', { operation, video_ids: videoIds, ...data }, { signal }),
	getThumbnail: (id) => `${API_ORIGIN}/api/v1/videos/${id}/thumbnail`,
	openInExplorer: (id, signal) => api.post(`/videos/${id}/open-in-explorer`, {}, { signal }),
	updateVideoMarksByPath: (filePath, marks, signal) => api.patch('/videos/marks-by-path', { file_path: filePath, ...marks }, { signal }),
	locateFile: (id) => api.get(`/videos/${id}/locate`),
	remapPath: (id, filePath) => api.patch(`/videos/${id}/remap`, { file_path: filePath }),
	cleanupMissing: (libraryId) => api.post('/videos/missing/cleanup', { library_id: libraryId || 0 }),
	getDuplicates: () => api.get('/videos/duplicates'),
	qualityCompare: (videoIds) => api.post('/videos/quality-compare', { video_ids: videoIds }),
	analyze: (formData, signal) => api.post('/videos/analyze', formData, {
		headers: { 'Content-Type': 'multipart/form-data' },
		timeout: 300000, // 5 min — large file uploads can be slow
		signal,
	}),
	markWatched: (id, watched) => api.patch(`/videos/${id}/watched`, { watched }),
	setRating: (id, rating) => api.patch(`/videos/${id}/rating`, { rating }),
}

export const studiosAPI = {
	getAll: () => api.get('/studios'),
	getById: (id, includeGroups = false) => api.get(`/studios/${id}`, { params: { include_groups: includeGroups } }),
	create: (data) => api.post('/studios', data),
	update: (id, data) => api.put(`/studios/${id}`, data),
	delete: (id) => api.delete(`/studios/${id}`),
	resetMetadata: (id) => api.post(`/studios/${id}/reset-metadata`),
}

export const groupsAPI = {
	getAll: (studioId) => api.get('/groups', { params: studioId ? { studio_id: studioId } : {} }),
	getById: (id) => api.get(`/groups/${id}`),
	create: (data) => api.post('/groups', data),
	update: (id, data) => api.put(`/groups/${id}`, data),
	delete: (id) => api.delete(`/groups/${id}`),
	resetMetadata: (id) => api.post(`/groups/${id}/reset-metadata`),
}

export const tagsAPI = {
	getAll: () => api.get('/tags'),
	getById: (id) => api.get(`/tags/${id}`),
	create: (data) => api.post('/tags', data),
	update: (id, data) => api.put(`/tags/${id}`, data),
	delete: (id) => api.delete(`/tags/${id}`),
	merge: (data) => api.post('/tags/merge', data),
}

export const filesAPI = {
	scan: (data) => api.post('/files/scan', data),
	rename: (data) => api.post('/files/rename', data),
	move: (data) => api.post('/files/move', data),
	moveAcrossLibraries: (data) => api.post('/files/move-across-libraries', data),
	delete: (data) => api.delete('/files/delete', { data }),
}

export const videoListsAPI = {
	getAll: () => api.get('/lists'),
	getById: (id) => api.get(`/lists/${id}`),
	create: (data) => api.post('/lists', data),
	update: (id, data) => api.put(`/lists/${id}`, data),
	delete: (id) => api.delete(`/lists/${id}`),
	getItems: (id) => api.get(`/lists/${id}/items`),
	addVideos: (id, data) => api.post(`/lists/${id}/videos`, data),
	removeVideos: (id, data) => api.delete(`/lists/${id}/videos`, { data }),
	reorderItems: (id, itemIds) => api.post(`/lists/${id}/reorder`, { item_ids: itemIds }),
	getListsForVideo: (videoId, isScraped = false) => api.get('/lists/for-video', { params: { video_id: videoId, is_scraped: isScraped } }),
	exportList: (id) => api.get(`/lists/${id}/export`),
}

export const activityAPI = {
	getAll: (params) => api.get('/activity', { params }), // Get all activities with filters (status, task_type, limit)
	getRecent: (limit = 20) => api.get('/activity/recent', { params: { limit } }),
	getStatus: () => api.get('/activity/status'),
	getStats: () => api.get('/activity/stats'),
	getPaused: () => api.get('/activity/paused'), // Get all paused tasks
	getById: (id) => api.get(`/activity/${id}`),
	create: (data) => api.post('/activity', data),
	update: (id, data) => api.put(`/activity/${id}`, data),
	delete: (id) => api.delete(`/activity/${id}`),
	cleanOld: (days = 30) => api.post('/activity/clean', null, { params: { days } }),
	clearAll: () => api.post('/activity/clear-all'),
	pause: (id, checkpoint = {}) => api.post(`/activity/${id}/pause`, { checkpoint }), // Pause a running task
	resume: (id) => api.post(`/activity/${id}/resume`), // Resume a paused task
}

export const consoleLogAPI = {
	getAll: (params) => api.get('/console-logs', { params }), // Get all console logs with filters (source, level, search, page, limit)
	getStats: () => api.get('/console-logs/stats'), // Get console log statistics
	getById: (id) => api.get(`/console-logs/${id}`), // Get single console log
	create: (data) => api.post('/console-logs', data), // Create a new console log (source, level, message, details)
	delete: (id) => api.delete(`/console-logs/${id}`), // Delete a console log
	clearAll: () => api.post('/console-logs/clear'), // Clear all console logs
	cleanOld: (days = 30) => api.post('/console-logs/clean', null, { params: { days } }), // Clean old console logs
}

export const databaseAPI = {
	getStats: () => api.get('/database/stats'),
	optimize: () => api.post('/database/optimize'),
	backup: () => api.post('/database/backup'),
	listBackups: () => api.get('/database/backups'),
	restore: (backupPath) => api.post('/database/restore', { backup_path: backupPath }),
}

export const aiAPI = {
	// Performer linking
	linkPerformers: (data) => api.post('/ai/link-performers', data),
	applyLinks: (data) => api.post('/ai/apply-links', data),

	// Smart tagging
	suggestTags: (data) => api.post('/ai/suggest-tags', data),
	applyTagSuggestions: (data) => api.post('/ai/apply-tag-suggestions', data),

	// Scene detection
	detectScenes: (data) => api.post('/ai/detect-scenes', data),

	// Content classification
	classifyContent: (data) => api.post('/ai/classify-content', data),

	// Quality analysis
	analyzeQuality: (data) => api.post('/ai/analyze-quality', data),

	// Missing metadata detection
	detectMissingMetadata: (data) => api.post('/ai/detect-missing-metadata', data),

	// Duplicate detection
	detectDuplicates: (data) => api.post('/ai/detect-duplicates', data),

	// Auto-naming
	suggestNaming: (data) => api.post('/ai/suggest-naming', data),

	// Library analytics
	getLibraryAnalytics: () => api.get('/ai/library-analytics'),

	// Health check — auto-fix missing metadata, thumbnails, and unlinked performers
	runHealthCheck: () => api.post('/ai/health-check'),

	// Thumbnail quality
	analyzeThumbnailQuality: (data) => api.post('/ai/analyze-thumbnail-quality', data),

	// Chat (uses LLM with rule-based fallback)
	chat: (data) => api.post('/ai/chat', data),
	getChatHistory: (limit = 100) => api.get('/ai/chat/history', { params: { limit } }),
	clearChatHistory: () => api.delete('/ai/chat/history'),
	resetChat: () => api.post('/ai/chat/reset'),

	// Companion Supervisor
	trackActions: (actions) => api.post('/ai/companion/actions', { actions }),
	getCompanionEvaluations: (limit = 20) => api.get('/ai/companion/evaluations', { params: { limit } }),
	approveProposal: (id) => api.post(`/ai/companion/proposals/${id}/approve`),
	rejectProposal: (id) => api.post(`/ai/companion/proposals/${id}/reject`),

	// Roleplay watcher (AIInfluence NPC files)
	getRoleplayWatches: () => api.get('/ai/roleplay/watches'),
	addRoleplayWatch: (filePath) => api.post('/ai/roleplay/watches', { file_path: filePath }),
	removeRoleplayWatch: (id) => api.delete(`/ai/roleplay/watches/${id}`),
	getRoleplaySnapshots: () => api.get('/ai/roleplay/snapshots'),

	// Memory management
	saveMemory: (data) => api.post('/ai/memories', data),
	getMemories: (params) => api.get('/ai/memories', { params }),
	searchMemories: (query) => api.get('/ai/memories/search', { params: { query } }),
	deleteMemory: (id) => api.delete(`/ai/memories/${id}`),
	updateMemory: (id, data) => api.put(`/ai/memories/${id}`, data),

	// Companion status
	getStatus: () => api.get('/ai/status'),

	// Task Board
	getTasks: (params) => api.get('/ai/tasks', { params }),
	getTask: (id) => api.get(`/ai/tasks/${id}`),
	createTask: (data) => api.post('/ai/tasks', data),
	updateTask: (id, data) => api.put(`/ai/tasks/${id}`, data),
	deleteTask: (id) => api.delete(`/ai/tasks/${id}`),
	getTaskStats: () => api.get('/ai/tasks/stats'),

	// AI config (OpenRouter)
	getConfig: () => api.get('/ai/config'),
	updateConfig: (data) => api.put('/ai/config', data),
	testConnection: () => api.post('/ai/test-connection'),
}

// Dashboard API
export const dashboardAPI = {
	getStats: () => api.get('/dashboard/stats'),
	getFeed: (limit = 20, offset = 0, scraperType = '') => {
		const params = { limit, offset }
		if (scraperType) params.scraper_type = scraperType
		return api.get('/dashboard/feed', { params })
	},
	getScrapers: () => api.get('/dashboard/scrapers'),
	getRunningTasks: () => api.get('/dashboard/running-tasks'),
	executeQuickAction: (action, data) => api.post(`/dashboard/quick-action/${action}`, data),
	clearFeed: () => api.delete('/dashboard/feed'),
}

// Auto-Update API
export const autoUpdateAPI = {
	getConfigs: () => api.get('/auto-update/configs'),
	getConfig: (type) => api.get(`/auto-update/configs/${type}`),
	updateConfig: (type, data) => api.put(`/auto-update/configs/${type}`, data),
	toggle: (type, enabled) => api.post(`/auto-update/configs/${type}/toggle`, { enabled }),
	checkNow: (type) => api.post(`/auto-update/configs/${type}/check-now`),
	getStatus: () => api.get('/auto-update/status'),
}

// Video Links API (Edited↔Backup pairing)
export const videoLinksAPI = {
	suggest: (editedLibraryId, backupLibraryId) =>
		api.post('/video-links/suggest', { edited_library_id: editedLibraryId, backup_library_id: backupLibraryId }),
	getLinks: (editedLibraryId, status = '') =>
		api.get('/video-links', { params: { edited_library_id: editedLibraryId, status: status || undefined } }),
	updateStatus: (id, status) => api.patch(`/video-links/${id}`, { status }),
	bulkUpdateStatus: (ids, status) => api.post('/video-links/bulk-status', { ids, status }),
	getLinkForVideo: (videoId) => api.get(`/video-links/for-video/${videoId}`),
}

export const scraperMatchAPI = {
	findLibraryMatch: (scrapedVideoId) => api.get(`/scraper/videos/${scrapedVideoId}/library-match`),
}

// Scraper thread watch API
export const scraperThreadsAPI = {
	toggleWatch: (id) => api.post(`/scraper/threads/${id}/watch`),
	getWatched: () => api.get('/scraper/threads/watched'),
	getRecentPosts: (threadId, limit = 20) => api.get(`/scraper/threads/${threadId}/recent-posts`, { params: { limit } }),
	markPostAsRead: (postId) => api.post(`/scraper/posts/${postId}/mark-read`),
}

// Open a URL in Firefox via the backend
export const openInFirefox = (url) => api.post('/system/open-url', { url })

// Detect pagination pattern and site type from two page URLs
export const detectPagination = (page1Url, page2Url) =>
	api.post('/scraper/detect-pagination', { page1_url: page1Url, page2_url: page2Url })

// Test-scrape a listing page URL without saving (returns videos_found count)
export const testScrapePage = (url) =>
	api.get('/scraper/video/test', { params: { url } })

// Delete all scraped videos from a specific source hostname (e.g. "hqporner.com")
export const deleteVideosBySource = (source) =>
	api.delete(`/scraper/videos/source/${encodeURIComponent(source)}`)

// Bulk delete scraped videos by ID array
export const bulkDeleteVideos = (ids) =>
	api.post('/scraper/videos/bulk-delete', { ids })

// Sexuria Links API
export const sexuriaLinksAPI = {
	getStats: () => api.get('/scraper/sexuria/links/stats'),
	extractLinks: (videoId, url) => api.post(`/scraper/sexuria/items/${videoId}/extract-links`, { url }),
	getLinks: (videoId) => api.get(`/scraper/sexuria/items/${videoId}/links`),
	verifyItemLinks: (videoId) => api.post(`/scraper/sexuria/items/${videoId}/verify-links`),
	verifyAll: (limit = 100, cutoffDays = 7) => api.post('/scraper/sexuria/links/verify-all', null, { params: { limit, cutoff_days: cutoffDays } }),
}

// Simpcity Link Verification API
export const linkVerificationAPI = {
	getStats: () => api.get('/scraper/links/verification-stats'),
	getProviderHealth: () => api.get('/scraper/providers/health'),
	verifyThreadLinks: (threadId) => api.post(`/scraper/threads/${threadId}/verify-links`),
	getThreadLinkStats: (threadId) => api.get(`/scraper/threads/${threadId}/link-stats`),
	verifyAllLinks: (limit = 100, cutoffDays = 7) => api.post('/scraper/links/verify-all', null, { params: { limit, cutoff_days: cutoffDays } }),
}

// JDownloader 2 Integration API
export const jdownloaderAPI = {
	// Configuration
	getConfig: () => api.get('/jdownloader/config'),
	updateConfig: (config) => api.put('/jdownloader/config', config),
	testConnection: () => api.post('/jdownloader/test'),
	isAvailable: () => api.get('/jdownloader/available'),

	// Send links
	sendLinks: (links) => api.post('/jdownloader/send', { links }),
	sendLink: (url, packageName = '', videoId = null, autoStart = false) =>
		api.post('/jdownloader/send', {
			links: [{ url, packageName, videoId, autoStart }],
		}),
	sendThreadLinks: (threadId, options = {}) =>
		api.post(`/jdownloader/threads/${threadId}/send`, options),

	// History
	getHistory: (limit = 50, offset = 0) =>
		api.get('/jdownloader/history', { params: { limit, offset } }),
	clearHistory: () => api.delete('/jdownloader/history'),
	getPackageTimestamps: () => api.get('/jdownloader/package-timestamps'),

	// Download control
	getStatus: () => api.get('/jdownloader/status'),
	startDownloads: () => api.post('/jdownloader/start'),
	stopDownloads: () => api.post('/jdownloader/stop'),
	pauseDownloads: (pause) => api.post('/jdownloader/pause', { pause }),

	// System info & speed
	getSystemInfo: () => api.get('/jdownloader/system'),
	getSpeed: () => api.get('/jdownloader/speed'),
	setSpeedLimit: (bytesPerSecond) => api.post('/jdownloader/speed-limit', { bytesPerSecond }),

	// LinkGrabber
	getLinkGrabberLinks: () => api.get('/jdownloader/linkgrabber/links'),
	getLinkGrabberPackages: () => api.get('/jdownloader/linkgrabber/packages'),
	moveToDownloads: (linkIds = [], packageIds = []) =>
		api.post('/jdownloader/linkgrabber/move', { linkIds, packageIds }),
	removeLinkGrabberLinks: (linkIds = [], packageIds = []) =>
		api.post('/jdownloader/linkgrabber/remove', { linkIds, packageIds }),
	clearLinkGrabber: () => api.post('/jdownloader/linkgrabber/clear'),

	// Downloads
	getDownloadLinks: () => api.get('/jdownloader/downloads/links'),
	getDownloadPackages: () => api.get('/jdownloader/downloads/packages'),
	removeDownloadLinks: (linkIds = [], packageIds = []) =>
		api.post('/jdownloader/downloads/remove', { linkIds, packageIds }),
	forceDownload: (linkIds = [], packageIds = []) =>
		api.post('/jdownloader/downloads/force', { linkIds, packageIds }),
	cleanupFinished: () => api.post('/jdownloader/downloads/cleanup'),
	resetLinks: (linkIds = [], packageIds = []) =>
		api.post('/jdownloader/downloads/reset', { linkIds, packageIds }),

	// Common operations
	setLinkEnabled: (enabled, linkIds = [], packageIds = [], isDownload = false) =>
		api.post('/jdownloader/enable', { enabled, linkIds, packageIds, isDownload }),
	setLinkPriority: (priority, linkIds = [], packageIds = [], isDownload = false) =>
		api.post('/jdownloader/priority', { priority, linkIds, packageIds, isDownload }),
	renamePackage: (packageId, newName, isDownload = false) =>
		api.post('/jdownloader/rename', { packageId, newName, isDownload }),
}

// Scraper API — covers all endpoints used by ScraperPage.vue
export const scrapersAPI = {
	// ── General ──────────────────────────────────────────────────────────────
	getStats:           ()       => api.get('/scraper/stats'),
	getSession:         ()       => api.get('/scraper/session'),
	setSession:         (data)   => api.post('/scraper/session', data),
	openLoginHelper:    (site)   => api.post('/system/open-login-helper', { site }),
	openURL:            (url)    => api.post('/system/open-url', { url }),
	scrapeForumAll:     (data)   => api.post('/scraper/forum/scrape-all', data),
	autoLinkPerformers: ()       => api.post('/scraper/performers/auto-link'),
	checkLinkStatuses:  ()       => api.post('/scraper/links/check-status'),

	// ── Scraped videos (generic) ───────────────────────────────────────────
	getVideos:       (params)   => api.get('/scraper/videos', { params }),
	clearAllVideos:  ()         => api.delete('/scraper/videos'),
	clearSiteVideos: (site)     => api.delete(`/scraper/videos/site/${site}`),
	bulkDelete:      (ids)      => api.post('/scraper/videos/bulk-delete', { ids }),
	scrapeVideoUrl:   (data)     => api.post('/scraper/video', data),
	testScrapePage:   (url)      => api.get('/scraper/video/test', { params: { url } }),
	extractVideoUrl:       (url) => api.get('/scraper/extract-video-url', { params: { url } }),
	markVideoDownloaded:   (id)  => api.post(`/scraper/videos/${id}/mark-downloaded`),
	unmarkVideoDownloaded: (id)  => api.delete(`/scraper/videos/${id}/mark-downloaded`),

	// ── Threads ───────────────────────────────────────────────────────────
	getThreads:         (params) => api.get('/scraper/threads', { params }),
	startScrape:        (data)   => api.post('/scraper/threads/scrape', data),
	deleteThreads:      (data)   => api.delete('/scraper/threads', { data }),
	deleteAllThreads:   ()       => api.delete('/scraper/threads/all'),
	rescrapeThread:     (id)     => api.post(`/scraper/threads/${id}/rescrape`),
	watchThread:        (id)     => api.post(`/scraper/threads/${id}/watch`),
	verifyThreadLinks:  (id)     => api.post(`/scraper/threads/${id}/verify-links`),
	getThreadLinkStats: (id)     => api.get(`/scraper/threads/${id}/link-stats`),

	// ── Tabs & Jobs ───────────────────────────────────────────────────────
	getTabs:    ()       => api.get('/scraper/tabs'),
	deleteTab:  (id)     => api.delete(`/scraper/tabs/${id}`),
	startJob:   (data)   => api.post('/scraper/jobs/start', data),
	getJobs:    (tabId)  => api.get('/scraper/jobs', { params: { tabId } }),
	getJob:     (id)     => api.get(`/scraper/jobs/${id}`),
	cancelJob:  (id)     => api.post(`/scraper/jobs/${id}/cancel`),

	// ── PornHub ───────────────────────────────────────────────────────────
	pornhub: {
		getStats:    ()       => api.get('/scraper/pornhub/stats'),
		getKeywords: ()       => api.get('/scraper/pornhub/keywords'),
		getVideos:   (params) => api.get('/scraper/pornhub/videos', { params }),
		scrape:      (data)   => api.post('/scraper/pornhub/scrape', data),
		clearVideos: ()       => api.delete('/scraper/videos/site/pornhub'),
	},

	// ── LuxureTV ──────────────────────────────────────────────────────────
	luxuretv: {
		getStats:      ()       => api.get('/scraper/luxuretv/stats'),
		getCategories: ()       => api.get('/scraper/luxuretv/categories'),
		getVideos:     (params) => api.get('/scraper/luxuretv/videos', { params }),
		getPerformers: ()       => api.get('/scraper/luxuretv/performers'),
		getTags:       ()       => api.get('/scraper/luxuretv/tags'),
		scrape:        (data)   => api.post('/scraper/luxuretv/scrape', data),
		fetchEmbeds:   ()       => api.post('/scraper/luxuretv/fetch-embeds'),
		clearVideos:   ()       => api.delete('/scraper/videos/site/luxuretv'),
	},

	// ── Sexuria ───────────────────────────────────────────────────────────
	sexuria: {
		getStats:         ()         => api.get('/scraper/sexuria/stats'),
		getCategories:    ()         => api.get('/scraper/sexuria/categories'),
		getItems:         (params)   => api.get('/scraper/sexuria/items', { params }),
		getSearchOptions: ()         => api.get('/scraper/sexuria/search-options'),
		scrape:           (data)     => api.post('/scraper/sexuria/scrape', data),
		advancedSearch:   (data)     => api.post('/scraper/sexuria/advanced-search', data),
		getItemDetails:   (data)     => api.post('/scraper/sexuria/item-details', data),
		extractItemLinks: (id, url)  => api.post(`/scraper/sexuria/items/${id}/extract-links`, { url }),
		getItemLinks:     (id)       => api.get(`/scraper/sexuria/items/${id}/links`),
		verifyItemLinks:  (id)       => api.post(`/scraper/sexuria/items/${id}/verify-links`),
		clearItems:       ()         => api.delete('/scraper/videos/site/sexuria'),
	},

	// ── Rule34Video ───────────────────────────────────────────────────────
	rule34video: {
		getStats:            ()        => api.get('/scraper/rule34video/stats'),
		getVideos:           (params)  => api.get('/scraper/rule34video/videos', { params }),
		scrape:              (data)    => api.post('/scraper/rule34video/scrape', data),
		clearVideos:         ()        => api.delete('/scraper/videos/site/rule34video'),
		getFilterOptions:    ()        => api.get('/scraper/rule34video/filter-options'),
		getCategories:       ()        => api.get('/scraper/rule34video/categories'),
		getLocalCategories:  ()        => api.get('/scraper/rule34video/local-categories'),
		getModels:           ()        => api.get('/scraper/rule34video/models'),
		getVideoDetails:     (url)     => api.get('/scraper/rule34video/video-details', { params: { url } }),
		getDownloadURL:      (url)     => api.get('/scraper/rule34video/download-url', { params: { url } }),
	},

	// ── Motherless ────────────────────────────────────────────────────────
	motherless: {
		getStats:    ()       => api.get('/scraper/motherless/stats'),
		getVideos:   (params) => api.get('/scraper/motherless/videos', { params }),
		scrape:      (data)   => api.post('/scraper/motherless/scrape', data),
		clearVideos: ()       => api.delete('/scraper/videos/site/motherless'),
	},

	// ── LeakVids ──────────────────────────────────────────────────────────
	leakvids: {
		getStats:    ()       => api.get('/scraper/leakvids/stats'),
		getVideos:   (params) => api.get('/scraper/leakvids/videos', { params }),
		scrape:      (data)   => api.post('/scraper/leakvids/scrape', data),
		clearVideos: ()       => api.delete('/scraper/videos/site/leakvids'),
	},

	// ── HQPorner ──────────────────────────────────────────────────────────
	hqporner: {
		getStats:      ()       => api.get('/scraper/hqporner/stats'),
		getPerformers: ()       => api.get('/scraper/hqporner/performers'),
		getTags:       ()       => api.get('/scraper/hqporner/tags'),
		getVideos:     (params) => api.get('/scraper/hqporner/videos', { params }),
		scrape:        (data)   => api.post('/scraper/hqporner/scrape', data),
		clearVideos:   ()       => api.delete('/scraper/videos/site/hqporner'),
	},
}

// Asset URL helper
export const getAssetURL = (path) => {
	if (!path) return ''
	// Pass through external URLs as-is
	if (path.startsWith('http://') || path.startsWith('https://')) {
		return path
	}
	// Handle file:// URLs by stripping the protocol and converting to relative path
	if (path.startsWith('file://')) {
		// Remove file:/// prefix (keep the path after it)
		let localPath = path.replace(/^file:\/\/\/?/, '')
		// On Windows, file:///C:/... becomes C:/...
		// Convert to relative path if possible
		const relativePath = localPath.split('api\\assets\\')[1] || localPath.split('api/assets/')[1] || localPath
		return `${API_ORIGIN}/assets/${relativePath.replace(/\\/g, '/')}`
	}
	// Handle absolute paths on Windows
	if (path.includes(':\\')) {
		// Convert Windows path to web path
		const relativePath = path.split('api\\assets\\')[1] || path.split('api/assets/')[1]
		if (relativePath) {
			return `${API_ORIGIN}/assets/${relativePath.replace(/\\/g, '/')}`
		}
	}
	// Check if path already starts with /assets/ to avoid double prefix
	if (path.startsWith('/assets/')) {
		return `${API_ORIGIN}${path}`
	}
	return `${API_ORIGIN}/assets/${path}`
}

// Proxy a CDN image through the backend so the correct Referer header is sent
// (needed for hotlink-protected CDNs like HQPorner, Motherless, LeakVids, etc.)
export const getProxiedImageURL = (url) => {
	if (!url) return ''
	return `${API_ORIGIN}/api/v1/scraper/proxy-image?url=${encodeURIComponent(url)}`
}

export const siteHealthAPI = {
	check: (url) => api.get('/sites/status', { params: { url } }),
}

export const appConfigAPI = {
	get: () => api.get('/system/config'),
	update: (cfg) => api.put('/system/config', cfg),
}

export const systemAPI = {
	getHealth: () => api.get('/system/health'),
	checkUpdate: () => api.get('/system/update/check'),
	installUpdate: (downloadURL) => api.post('/system/update/install', { download_url: downloadURL }),
	getServices: () => api.get('/system/services'),
	controlService: (service, action) => api.post('/system/services/control', { service, action }),
	buildAndDeploy: () => api.post('/system/build-and-deploy'),
}

export default api
