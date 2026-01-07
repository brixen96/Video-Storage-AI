import axios from 'axios'

// Create axios instance with base configuration
const api = axios.create({
	baseURL: 'http://localhost:8080/api/v1', // Add /api/v1 prefix
	headers: {
		'Content-Type': 'application/json',
	},
	timeout: 600000, // Increase to 60 seconds
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
	fetchMetadata: (id, signal) => api.post(`/videos/${id}/fetch`, {}, { signal }),
	addTags: (id, tagIds, signal) => api.post(`/videos/${id}/tags`, { tag_ids: tagIds }, { signal }),
	removeTags: (id, tagIds, signal) => api.delete(`/videos/${id}/tags`, { data: { tag_ids: tagIds }, signal }),
	bulk: (operation, videoIds, data = {}, signal) => api.post('/videos/bulk', { operation, video_ids: videoIds, ...data }, { signal }),
	getThumbnail: (id) => `http://localhost:8080/api/v1/videos/${id}/thumbnail`,
	openInExplorer: (id, signal) => api.post(`/videos/${id}/open-in-explorer`, {}, { signal }),
	updateVideoMarksByPath: (filePath, marks, signal) => api.patch('/videos/marks-by-path', { file_path: filePath, ...marks }, { signal }),
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

	// Thumbnail quality
	analyzeThumbnailQuality: (data) => api.post('/ai/analyze-thumbnail-quality', data),

	// Chat (placeholder)
	chat: (data) => api.post('/ai/chat', data),

	// Memory management
	saveMemory: (data) => api.post('/ai/memories', data),
	getMemories: (params) => api.get('/ai/memories', { params }),
	searchMemories: (query) => api.get('/ai/memories/search', { params: { query } }),
	deleteMemory: (id) => api.delete(`/ai/memories/${id}`),
	updateMemory: (id, data) => api.put(`/ai/memories/${id}`, data),
}

// LM Studio AI Companion API
const lmStudioAPI = axios.create({
	baseURL: 'http://localhost:1234/v1',
	headers: {
		'Content-Type': 'application/json',
	},
	timeout: 1200000, // 2 minutes for tool calling iterations
})

export const aiCompanionAPI = {
	chat: async (messages, options = {}) => {
		try {
			const requestBody = {
				messages,
				temperature: options.temperature || 0.7,
				max_tokens: options.max_tokens || 2000,
				stream: false,
			}

			// Add tools if provided
			if (options.tools) {
				requestBody.tools = options.tools
			}

			// Add tool_choice if provided
			if (options.tool_choice) {
				requestBody.tool_choice = options.tool_choice
			}

			const response = await lmStudioAPI.post('/chat/completions', requestBody)
			return response.data
		} catch (error) {
			console.error('LM Studio API Error:', error)
			throw error
		}
	},
}

// Asset URL helper
export const getAssetURL = (path) => {
	if (!path) return ''
	// Handle absolute paths on Windows
	if (path.includes(':\\')) {
		// Convert Windows path to web path
		const relativePath = path.split('api\\assets\\')[1] || path.split('api/assets/')[1]
		if (relativePath) {
			return `http://localhost:8080/assets/${relativePath.replace(/\\/g, '/')}`
		}
	}
	// Check if path already starts with /assets/ to avoid double prefix
	if (path.startsWith('/assets/')) {
		return `http://localhost:8080${path}`
	}
	return `http://localhost:8080/assets/${path}`
}

export default api
