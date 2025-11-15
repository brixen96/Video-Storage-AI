import axios from 'axios'

// Create axios instance with base configuration
const api = axios.create({
	baseURL: 'http://localhost:8080/api/v1', // Add /api/v1 prefix
	headers: {
		'Content-Type': 'application/json',
	},
	timeout: 600000, // Increase to 60 seconds
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
	getAll: () => api.get('/libraries'),
	getPrimary: () => api.get('/libraries/primary'),
	getById: (id) => api.get(`/libraries/${id}`),
	create: (data) => api.post('/libraries', data),
	update: (id, data) => api.put(`/libraries/${id}`, data),
	delete: (id) => api.delete(`/libraries/${id}`),
	browse: (id, path = '', metadata = false) => api.get(`/libraries/${id}/browse`, { params: { path, metadata: metadata ? 'true' : 'false' } }),
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
}

export const videosAPI = {
	getAll: (params) => api.get('/videos', { params }),
	getById: (id) => api.get(`/videos/${id}`),
	search: (params) => api.get('/videos/search', { params }),
	create: (data) => api.post('/videos', data),
	update: (id, data) => api.put(`/videos/${id}`, data),
	delete: (id) => api.delete(`/videos/${id}`),
	scan: (libraryId) => api.post('/videos/scan', { library_id: libraryId }),
	fetchMetadata: (id) => api.post(`/videos/${id}/fetch`),
	addTags: (id, tagIds) => api.post(`/videos/${id}/tags`, { tag_ids: tagIds }),
	removeTags: (id, tagIds) => api.delete(`/videos/${id}/tags`, { data: { tag_ids: tagIds } }),
	bulk: (operation, videoIds, data = {}) => api.post('/videos/bulk', { operation, video_ids: videoIds, ...data }),
	getThumbnail: (id) => `http://localhost:8080/api/v1/videos/${id}/thumbnail`,
	openInExplorer: (id) => api.post(`/videos/${id}/open-in-explorer`),
	updateVideoMarksByPath: (filePath, marks) =>
		api.patch('/videos/marks-by-path', { file_path: filePath, ...marks }),
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
	delete: (data) => api.delete('/files/delete', { data }),
}

export const activityAPI = {
	getAll: (params) => api.get('/activity', { params }), // Get all activities with filters (status, task_type, limit)
	getRecent: (limit = 20) => api.get('/activity/recent', { params: { limit } }),
	getStatus: () => api.get('/activity/status'),
	getStats: () => api.get('/activity/stats'),
	getById: (id) => api.get(`/activity/${id}`),
	create: (data) => api.post('/activity', data),
	update: (id, data) => api.put(`/activity/${id}`, data),
	delete: (id) => api.delete(`/activity/${id}`),
	cleanOld: (days = 30) => api.post('/activity/clean', null, { params: { days } }),
}

export const aiAPI = {
	chat: (data) => api.post('/ai/chat', data),
	suggestTags: (data) => api.post('/ai/suggest-tags', data),
	suggestNaming: (data) => api.post('/ai/suggest-naming', data),
	analyzeLibrary: (data) => api.post('/ai/analyze-library', data),
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
	return `http://localhost:8080/assets/${path}`
}

export default api
