import { createStore } from 'vuex'
import { performersAPI, studiosAPI, tagsAPI, groupsAPI } from '@/services/api'
import cacheService from '@/services/cacheService'

// Cache TTL (Time To Live) in milliseconds
const CACHE_TTL = {
	performers: 5 * 60 * 1000, // 5 minutes
	studios: 5 * 60 * 1000, // 5 minutes
	tags: 5 * 60 * 1000, // 5 minutes
	groups: 5 * 60 * 1000, // 5 minutes
}

// Cache keys for persistent storage
const CACHE_KEYS = {
	performers: '/api/v1/performers',
	studios: '/api/v1/studios',
	tags: '/api/v1/tags',
	groups: '/api/v1/groups',
}

export default createStore({
	state: {
		// Cached data
		performers: {
			data: [],
			timestamp: null,
			loading: false,
		},
		studios: {
			data: [],
			timestamp: null,
			loading: false,
		},
		tags: {
			data: [],
			timestamp: null,
			loading: false,
		},
		groups: {
			data: [],
			timestamp: null,
			loading: false,
		},
	},

	getters: {
		// Check if cache is valid
		isPerformersCacheValid: (state) => {
			if (!state.performers.timestamp) return false
			return Date.now() - state.performers.timestamp < CACHE_TTL.performers
		},
		isStudiosCacheValid: (state) => {
			if (!state.studios.timestamp) return false
			return Date.now() - state.studios.timestamp < CACHE_TTL.studios
		},
		isTagsCacheValid: (state) => {
			if (!state.tags.timestamp) return false
			return Date.now() - state.tags.timestamp < CACHE_TTL.tags
		},
		isGroupsCacheValid: (state) => {
			if (!state.groups.timestamp) return false
			return Date.now() - state.groups.timestamp < CACHE_TTL.groups
		},

		// Get cached data
		performers: (state) => state.performers.data,
		studios: (state) => state.studios.data,
		tags: (state) => state.tags.data,
		groups: (state) => state.groups.data,
	},

	mutations: {
		SET_PERFORMERS(state, performers) {
			state.performers.data = performers
			state.performers.timestamp = Date.now()
			state.performers.loading = false
		},
		SET_PERFORMERS_LOADING(state, loading) {
			state.performers.loading = loading
		},
		SET_STUDIOS(state, studios) {
			state.studios.data = studios
			state.studios.timestamp = Date.now()
			state.studios.loading = false
		},
		SET_STUDIOS_LOADING(state, loading) {
			state.studios.loading = loading
		},
		SET_TAGS(state, tags) {
			state.tags.data = tags
			state.tags.timestamp = Date.now()
			state.tags.loading = false
		},
		SET_TAGS_LOADING(state, loading) {
			state.tags.loading = loading
		},
		SET_GROUPS(state, groups) {
			state.groups.data = groups
			state.groups.timestamp = Date.now()
			state.groups.loading = false
		},
		SET_GROUPS_LOADING(state, loading) {
			state.groups.loading = loading
		},
		INVALIDATE_PERFORMERS(state) {
			state.performers.timestamp = null
		},
		INVALIDATE_STUDIOS(state) {
			state.studios.timestamp = null
		},
		INVALIDATE_TAGS(state) {
			state.tags.timestamp = null
		},
		INVALIDATE_GROUPS(state) {
			state.groups.timestamp = null
		},
	},

	actions: {
		async fetchPerformers({ commit, state, getters }, forceRefresh = false) {
			// Check persistent cache first (survives page reloads)
			if (!forceRefresh) {
				const cachedData = await cacheService.get(CACHE_KEYS.performers)
				if (cachedData) {
					commit('SET_PERFORMERS', cachedData)
					return cachedData
				}
			}

			// Return in-memory cached data if valid and not forcing refresh
			if (!forceRefresh && getters.isPerformersCacheValid) {
				return state.performers.data
			}

			// Prevent duplicate requests
			if (state.performers.loading) {
				return new Promise((resolve) => {
					const checkInterval = setInterval(() => {
						if (!state.performers.loading) {
							clearInterval(checkInterval)
							resolve(state.performers.data)
						}
					}, 100)
				})
			}

			commit('SET_PERFORMERS_LOADING', true)
			try {
				const response = await performersAPI.getAll()
				const performers = (response && response.data) || []
				commit('SET_PERFORMERS', performers)

				// Store in persistent cache
				await cacheService.set(CACHE_KEYS.performers, performers)

				return performers
			} catch (error) {
				commit('SET_PERFORMERS_LOADING', false)
				throw error
			}
		},

		async fetchStudios({ commit, state, getters }, forceRefresh = false) {
			// Check persistent cache first
			if (!forceRefresh) {
				const cachedData = await cacheService.get(CACHE_KEYS.studios)
				if (cachedData) {
					commit('SET_STUDIOS', cachedData)
					return cachedData
				}
			}

			if (!forceRefresh && getters.isStudiosCacheValid) {
				return state.studios.data
			}

			if (state.studios.loading) {
				return new Promise((resolve) => {
					const checkInterval = setInterval(() => {
						if (!state.studios.loading) {
							clearInterval(checkInterval)
							resolve(state.studios.data)
						}
					}, 100)
				})
			}

			commit('SET_STUDIOS_LOADING', true)
			try {
				const response = await studiosAPI.getAll()
				const studios = (response && response.data) || []
				commit('SET_STUDIOS', studios)

				// Store in persistent cache
				await cacheService.set(CACHE_KEYS.studios, studios)

				return studios
			} catch (error) {
				commit('SET_STUDIOS_LOADING', false)
				throw error
			}
		},

		async fetchTags({ commit, state, getters }, forceRefresh = false) {
			// Check persistent cache first
			if (!forceRefresh) {
				const cachedData = await cacheService.get(CACHE_KEYS.tags)
				if (cachedData) {
					commit('SET_TAGS', cachedData)
					return cachedData
				}
			}

			if (!forceRefresh && getters.isTagsCacheValid) {
				return state.tags.data
			}

			if (state.tags.loading) {
				return new Promise((resolve) => {
					const checkInterval = setInterval(() => {
						if (!state.tags.loading) {
							clearInterval(checkInterval)
							resolve(state.tags.data)
						}
					}, 100)
				})
			}

			commit('SET_TAGS_LOADING', true)
			try {
				const tags = await tagsAPI.getAll()
				// Tags API returns array directly (not wrapped in response.data)
				const tagsList = Array.isArray(tags) ? tags : []
				commit('SET_TAGS', tagsList)

				// Store in persistent cache
				await cacheService.set(CACHE_KEYS.tags, tagsList)

				return tagsList
			} catch (error) {
				commit('SET_TAGS_LOADING', false)
				throw error
			}
		},

		async fetchGroups({ commit, state, getters }, { studioId = null, forceRefresh = false } = {}) {
			// Check persistent cache first (only for non-filtered requests)
			if (!forceRefresh && !studioId) {
				const cachedData = await cacheService.get(CACHE_KEYS.groups)
				if (cachedData) {
					commit('SET_GROUPS', cachedData)
					return cachedData
				}
			}

			if (!forceRefresh && getters.isGroupsCacheValid && !studioId) {
				return state.groups.data
			}

			// Don't cache filtered requests
			if (studioId) {
				const response = await groupsAPI.getAll(studioId)
				return (response && response.data) || []
			}

			if (state.groups.loading) {
				return new Promise((resolve) => {
					const checkInterval = setInterval(() => {
						if (!state.groups.loading) {
							clearInterval(checkInterval)
							resolve(state.groups.data)
						}
					}, 100)
				})
			}

			commit('SET_GROUPS_LOADING', true)
			try {
				const response = await groupsAPI.getAll()
				const groups = (response && response.data) || []
				commit('SET_GROUPS', groups)

				// Store in persistent cache
				await cacheService.set(CACHE_KEYS.groups, groups)

				return groups
			} catch (error) {
				commit('SET_GROUPS_LOADING', false)
				throw error
			}
		},

		invalidatePerformers({ commit }) {
			commit('INVALIDATE_PERFORMERS')
			cacheService.invalidate(CACHE_KEYS.performers)
		},
		invalidateStudios({ commit }) {
			commit('INVALIDATE_STUDIOS')
			cacheService.invalidate(CACHE_KEYS.studios)
		},
		invalidateTags({ commit }) {
			commit('INVALIDATE_TAGS')
			cacheService.invalidate(CACHE_KEYS.tags)
		},
		invalidateGroups({ commit }) {
			commit('INVALIDATE_GROUPS')
			cacheService.invalidate(CACHE_KEYS.groups)
		},
	},
})
