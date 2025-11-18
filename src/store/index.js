import { createStore } from 'vuex'
import { performersAPI, studiosAPI, tagsAPI, groupsAPI } from '@/services/api'

// Cache TTL (Time To Live) in milliseconds
const CACHE_TTL = {
	performers: 5 * 60 * 1000, // 5 minutes
	studios: 5 * 60 * 1000, // 5 minutes
	tags: 5 * 60 * 1000, // 5 minutes
	groups: 5 * 60 * 1000, // 5 minutes
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
			// Return cached data if valid and not forcing refresh
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
				const performers = response.data || []
				commit('SET_PERFORMERS', performers)
				return performers
			} catch (error) {
				commit('SET_PERFORMERS_LOADING', false)
				throw error
			}
		},

		async fetchStudios({ commit, state, getters }, forceRefresh = false) {
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
				const studios = response.data || []
				commit('SET_STUDIOS', studios)
				return studios
			} catch (error) {
				commit('SET_STUDIOS_LOADING', false)
				throw error
			}
		},

		async fetchTags({ commit, state, getters }, forceRefresh = false) {
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
				const response = await tagsAPI.getAll()
				const tags = response.data || []
				commit('SET_TAGS', tags)
				return tags
			} catch (error) {
				commit('SET_TAGS_LOADING', false)
				throw error
			}
		},

		async fetchGroups({ commit, state, getters }, { studioId = null, forceRefresh = false } = {}) {
			if (!forceRefresh && getters.isGroupsCacheValid && !studioId) {
				return state.groups.data
			}

			// Don't cache filtered requests
			if (studioId) {
				const response = await groupsAPI.getAll(studioId)
				return response.data || []
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
				const groups = response.data || []
				commit('SET_GROUPS', groups)
				return groups
			} catch (error) {
				commit('SET_GROUPS_LOADING', false)
				throw error
			}
		},

		invalidatePerformers({ commit }) {
			commit('INVALIDATE_PERFORMERS')
		},
		invalidateStudios({ commit }) {
			commit('INVALIDATE_STUDIOS')
		},
		invalidateTags({ commit }) {
			commit('INVALIDATE_TAGS')
		},
		invalidateGroups({ commit }) {
			commit('INVALIDATE_GROUPS')
		},
	},
})
