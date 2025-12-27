// AI Companion Tools - Functions the AI can call to interact with the database

import { databaseAPI, librariesAPI, performersAPI, videosAPI, tagsAPI, aiAPI } from './api'

/**
 * Tool definitions for LM Studio function calling
 * These are the capabilities the AI has to interact with the database
 */
export const toolDefinitions = [
	{
		type: 'function',
		function: {
			name: 'get_library_stats',
			description: 'Get overall statistics about the video library including total videos, performers, tags, studios, and database size.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'get_all_libraries',
			description: 'Get a list of all video libraries with their names, paths, and video counts.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'search_performers',
			description: 'Search for performers by name or get all performers. Returns performer details including name, birth date, video count, and metadata.',
			parameters: {
				type: 'object',
				properties: {
					query: {
						type: 'string',
						description: 'Search query to filter performers by name. Leave empty to get all performers.',
					},
					limit: {
						type: 'number',
						description: 'Maximum number of results to return (default: 50)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'get_performer_by_name',
			description: 'Get detailed information about a specific performer by exact name match.',
			parameters: {
				type: 'object',
				properties: {
					name: {
						type: 'string',
						description: 'The exact name of the performer',
					},
				},
				required: ['name'],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'search_videos',
			description: 'Search for videos by title, performer, tag, or other criteria. Returns video details including title, performers, tags, duration, file path, and library.',
			parameters: {
				type: 'object',
				properties: {
					query: {
						type: 'string',
						description: 'Search query to filter videos',
					},
					library_id: {
						type: 'number',
						description: 'Filter by specific library ID',
					},
					performer_id: {
						type: 'number',
						description: 'Filter by specific performer ID',
					},
					tag_id: {
						type: 'number',
						description: 'Filter by specific tag ID',
					},
					limit: {
						type: 'number',
						description: 'Maximum number of results to return (default: 20)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'get_all_tags',
			description: 'Get all tags with their names and video counts. Useful for understanding what tags exist in the library.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'analyze_library_issues',
			description: 'Analyze the library for potential issues like missing metadata, duplicate videos, low quality thumbnails, or videos without performers/tags.',
			parameters: {
				type: 'object',
				properties: {
					check_type: {
						type: 'string',
						enum: ['missing_metadata', 'duplicates', 'no_performers', 'no_tags', 'all'],
						description: 'Type of issue to check for. Use "all" to check everything.',
					},
				},
				required: ['check_type'],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'get_oldest_performers',
			description: 'Get the oldest performers based on birth date. Returns performers sorted by age (oldest first).',
			parameters: {
				type: 'object',
				properties: {
					limit: {
						type: 'number',
						description: 'Number of oldest performers to return (default: 1)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'get_youngest_performers',
			description: 'Get the youngest performers based on birth date. Returns performers sorted by age (youngest first).',
			parameters: {
				type: 'object',
				properties: {
					limit: {
						type: 'number',
						description: 'Number of youngest performers to return (default: 1)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'get_top_performers',
			description: 'Get performers with the most videos. Returns performers sorted by video count.',
			parameters: {
				type: 'object',
				properties: {
					limit: {
						type: 'number',
						description: 'Number of top performers to return (default: 10)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'save_memory',
			description: 'Store important information in long-term memory. Use this to remember user preferences, important facts about the library, or insights you discover. This memory persists forever and can be recalled later.',
			parameters: {
				type: 'object',
				properties: {
					key: {
						type: 'string',
						description: 'A unique identifier for this memory (e.g., "user_favorite_performer", "library_scan_schedule")',
					},
					value: {
						type: 'string',
						description: 'The information to remember',
					},
					category: {
						type: 'string',
						enum: ['preference', 'fact', 'insight', 'task', 'note'],
						description: 'Type of memory being stored',
					},
					importance: {
						type: 'number',
						description: 'Importance level (1-10, where 10 is critical)',
					},
				},
				required: ['key', 'value', 'category'],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'recall_memories',
			description: 'Search and retrieve stored memories. Use this to recall past conversations, user preferences, or facts you previously learned.',
			parameters: {
				type: 'object',
				properties: {
					query: {
						type: 'string',
						description: 'Search query to find relevant memories',
					},
					category: {
						type: 'string',
						enum: ['preference', 'fact', 'insight', 'task', 'note', 'all'],
						description: 'Filter by memory category (default: all)',
					},
					limit: {
						type: 'number',
						description: 'Maximum number of memories to return (default: 10)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'update_memory',
			description: 'Update an existing memory with new information.',
			parameters: {
				type: 'object',
				properties: {
					key: {
						type: 'string',
						description: 'The key of the memory to update',
					},
					value: {
						type: 'string',
						description: 'The new value',
					},
				},
				required: ['key', 'value'],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'auto_link_performers',
			description: 'Automatically detect and link performers to videos based on filename analysis. Finds performer matches in video filenames and suggests links.',
			parameters: {
				type: 'object',
				properties: {
					auto_apply: {
						type: 'boolean',
						description: 'Automatically apply 100% confidence matches (default: true)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'suggest_smart_tags',
			description: 'AI-powered tag suggestions based on video content analysis. Analyzes video titles and metadata to suggest relevant tags.',
			parameters: {
				type: 'object',
				properties: {
					auto_apply: {
						type: 'boolean',
						description: 'Automatically apply high confidence tags (85%+) (default: true)',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'detect_scenes',
			description: 'Analyze videos to detect scene changes and chapters. Useful for breaking down long videos into segments.',
			parameters: {
				type: 'object',
				properties: {
					video_id: {
						type: 'number',
						description: 'Optional specific video ID to analyze. If not provided, analyzes all videos.',
					},
				},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'classify_content',
			description: 'Classify video content into categories based on analysis of metadata, tags, and performers.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'analyze_video_quality',
			description: 'Analyze video quality including resolution, bitrate, and encoding quality. Identifies low-quality videos that need attention.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'detect_duplicates',
			description: 'Find duplicate or similar videos in the library based on file size, duration, and content analysis.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'suggest_auto_naming',
			description: 'Generate intelligent file naming suggestions based on video metadata, performers, studio, and content.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
	{
		type: 'function',
		function: {
			name: 'analyze_thumbnail_quality',
			description: 'Analyze the quality of video thumbnails and identify videos with poor or missing thumbnails.',
			parameters: {
				type: 'object',
				properties: {},
				required: [],
			},
		},
	},
]

/**
 * Tool execution functions
 * These actually perform the operations when the AI calls a tool
 */
export const executeTool = async (toolName, args) => {
	console.log(`Executing tool: ${toolName}`, args)

	switch (toolName) {
		case 'get_library_stats': {
			const response = await databaseAPI.getStats()
			const stats = response.data || response
			return {
				total_videos: stats.video_count || 0,
				total_performers: stats.performer_count || 0,
				total_tags: stats.tag_count || 0,
				total_studios: stats.studio_count || 0,
				total_groups: stats.group_count || 0,
				database_size_mb: stats.size ? (stats.size / (1024 * 1024)).toFixed(2) : 0,
			}
		}

		case 'get_all_libraries': {
			const response = await librariesAPI.getAll()
			const libraries = Array.isArray(response) ? response : response.data || []
			return libraries.map((lib) => ({
				id: lib.id,
				name: lib.name,
				path: lib.path,
				video_count: lib.video_count || 0,
				is_primary: lib.primary || false,
			}))
		}

		case 'search_performers': {
			const params = {
				per_page: args.limit || 50,
			}
			if (args.query) {
				params.search = args.query
			}
			const response = await performersAPI.getAll(params)
			const performers = Array.isArray(response) ? response : response.data || []
			return performers.map((p) => ({
				id: p.id,
				name: p.name,
				birth_date: p.metadata?.birthdate || p.metadata?.date_of_birth || null,
				video_count: p.video_count || 0,
				metadata: p.metadata,
			}))
		}

		case 'get_performer_by_name': {
			const response = await performersAPI.getAll({ per_page: 500 })
			const performers = Array.isArray(response) ? response : response.data || []
			const performer = performers.find((p) => p.name.toLowerCase() === args.name.toLowerCase())
			if (!performer) {
				return { error: 'Performer not found' }
			}
			return {
				id: performer.id,
				name: performer.name,
				birth_date: performer.metadata?.birthdate || performer.metadata?.date_of_birth || null,
				video_count: performer.video_count || 0,
				metadata: performer.metadata,
			}
		}

		case 'search_videos': {
			const params = {
				per_page: args.limit || 20,
			}
			if (args.query) params.search = args.query
			if (args.library_id) params.library_id = args.library_id
			if (args.performer_id) params.performer_id = args.performer_id
			if (args.tag_id) params.tag_id = args.tag_id

			const response = await videosAPI.search(params)
			const videos = Array.isArray(response) ? response : response.data || []
			return videos.map((v) => ({
				id: v.id,
				title: v.title,
				file_path: v.file_path,
				duration: v.duration,
				library_id: v.library_id,
				performers: v.performers || [],
				tags: v.tags || [],
			}))
		}

		case 'get_all_tags': {
			const response = await tagsAPI.getAll({ per_page: 500 })
			const tags = Array.isArray(response) ? response : response.data || []
			return tags.map((t) => ({
				id: t.id,
				name: t.name,
				video_count: t.video_count || 0,
			}))
		}

		case 'analyze_library_issues': {
			const checkType = args.check_type || 'all'
			const issues = []

			// Get all videos to analyze
			const videosResponse = await videosAPI.getAll({ per_page: 500 })
			const videos = Array.isArray(videosResponse) ? videosResponse : videosResponse.data || []

			if (checkType === 'missing_metadata' || checkType === 'all') {
				const missingMetadata = videos.filter((v) => !v.title || !v.duration)
				if (missingMetadata.length > 0) {
					issues.push({
						type: 'missing_metadata',
						count: missingMetadata.length,
						description: `${missingMetadata.length} videos are missing title or duration information`,
					})
				}
			}

			if (checkType === 'no_performers' || checkType === 'all') {
				const noPerformers = videos.filter((v) => !v.performers || v.performers.length === 0)
				if (noPerformers.length > 0) {
					issues.push({
						type: 'no_performers',
						count: noPerformers.length,
						description: `${noPerformers.length} videos have no performers linked`,
					})
				}
			}

			if (checkType === 'no_tags' || checkType === 'all') {
				const noTags = videos.filter((v) => !v.tags || v.tags.length === 0)
				if (noTags.length > 0) {
					issues.push({
						type: 'no_tags',
						count: noTags.length,
						description: `${noTags.length} videos have no tags`,
					})
				}
			}

			return {
				total_videos_analyzed: videos.length,
				issues_found: issues.length,
				issues: issues,
			}
		}

		case 'get_oldest_performers': {
			const response = await performersAPI.getAll({ per_page: 500 })
			const performers = Array.isArray(response) ? response : response.data || []

			const withBirthdate = performers
				.filter((p) => p.metadata?.birthdate || p.metadata?.date_of_birth)
				.map((p) => {
					const birthdate = p.metadata?.birthdate || p.metadata?.date_of_birth
					return {
						id: p.id,
						name: p.name,
						birth_date: birthdate,
						age: new Date().getFullYear() - new Date(birthdate).getFullYear(),
						video_count: p.video_count || 0,
						parsed_date: new Date(birthdate),
					}
				})
				.filter((p) => !isNaN(p.parsed_date.getTime()))
				.sort((a, b) => a.parsed_date - b.parsed_date)

			return withBirthdate.slice(0, args.limit || 1)
		}

		case 'get_youngest_performers': {
			const response = await performersAPI.getAll({ per_page: 500 })
			const performers = Array.isArray(response) ? response : response.data || []

			const withBirthdate = performers
				.filter((p) => p.metadata?.birthdate || p.metadata?.date_of_birth)
				.map((p) => {
					const birthdate = p.metadata?.birthdate || p.metadata?.date_of_birth
					return {
						id: p.id,
						name: p.name,
						birth_date: birthdate,
						age: new Date().getFullYear() - new Date(birthdate).getFullYear(),
						video_count: p.video_count || 0,
						parsed_date: new Date(birthdate),
					}
				})
				.filter((p) => !isNaN(p.parsed_date.getTime()))
				.sort((a, b) => b.parsed_date - a.parsed_date)

			return withBirthdate.slice(0, args.limit || 1)
		}

		case 'get_top_performers': {
			const response = await performersAPI.getAll({ per_page: 500 })
			const performers = Array.isArray(response) ? response : response.data || []

			return performers
				.map((p) => ({
					id: p.id,
					name: p.name,
					video_count: p.video_count || 0,
					birth_date: p.metadata?.birthdate || p.metadata?.date_of_birth || null,
				}))
				.sort((a, b) => b.video_count - a.video_count)
				.slice(0, args.limit || 10)
		}

		case 'save_memory': {
			// Save to backend database
			const memoryData = {
				key: args.key,
				value: args.value,
				category: args.category,
				importance: args.importance || 5,
				created_at: new Date().toISOString(),
			}

			try {
				await aiAPI.saveMemory(memoryData)
				return {
					success: true,
					message: `Memory saved: ${args.key}`,
					data: memoryData,
				}
			} catch (error) {
				// Fallback to localStorage if backend fails
				const memories = JSON.parse(localStorage.getItem('ai_memories') || '[]')
				memories.push(memoryData)
				localStorage.setItem('ai_memories', JSON.stringify(memories))
				return {
					success: true,
					message: `Memory saved locally: ${args.key}`,
					data: memoryData,
				}
			}
		}

		case 'recall_memories': {
			try {
				// Try backend first
				const params = {
					category: args.category !== 'all' ? args.category : undefined,
					limit: args.limit || 10,
				}

				let response
				if (args.query) {
					response = await aiAPI.searchMemories(args.query)
				} else {
					response = await aiAPI.getMemories(params)
				}

				const memories = Array.isArray(response) ? response : response.data || []
				return memories
			} catch (error) {
				// Fallback to localStorage
				const memories = JSON.parse(localStorage.getItem('ai_memories') || '[]')

				let filtered = memories
				if (args.category && args.category !== 'all') {
					filtered = memories.filter((m) => m.category === args.category)
				}
				if (args.query) {
					const query = args.query.toLowerCase()
					filtered = filtered.filter(
						(m) => m.key.toLowerCase().includes(query) || m.value.toLowerCase().includes(query)
					)
				}

				return filtered.slice(0, args.limit || 10)
			}
		}

		case 'update_memory': {
			try {
				// Try backend first
				const memories = await aiAPI.getMemories({ key: args.key })
				const memoryList = Array.isArray(memories) ? memories : memories.data || []

				if (memoryList.length > 0) {
					const memory = memoryList[0]
					await aiAPI.updateMemory(memory.id, { value: args.value })
					return {
						success: true,
						message: `Memory updated: ${args.key}`,
					}
				} else {
					return { error: 'Memory not found' }
				}
			} catch (error) {
				// Fallback to localStorage
				const memories = JSON.parse(localStorage.getItem('ai_memories') || '[]')
				const index = memories.findIndex((m) => m.key === args.key)

				if (index >= 0) {
					memories[index].value = args.value
					memories[index].updated_at = new Date().toISOString()
					localStorage.setItem('ai_memories', JSON.stringify(memories))
					return {
						success: true,
						message: `Memory updated locally: ${args.key}`,
					}
				} else {
					return { error: 'Memory not found' }
				}
			}
		}

		case 'auto_link_performers': {
			const data = {
				auto_apply: args.auto_apply !== undefined ? args.auto_apply : true,
			}
			const response = await aiAPI.linkPerformers(data)
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_analyzed: result.videos_analyzed || 0,
				matches_found: result.matches_found || 0,
				auto_applied: result.auto_applied || 0,
				suggestions: result.suggestions || [],
				message: `Auto-link complete: ${result.matches_found} matches found, ${result.auto_applied || 0} automatically applied`,
			}
		}

		case 'suggest_smart_tags': {
			const data = {
				auto_apply: args.auto_apply !== undefined ? args.auto_apply : true,
			}
			const response = await aiAPI.suggestTags(data)
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_analyzed: result.videos_analyzed || 0,
				tags_suggested: result.tags_suggested || 0,
				suggestions: result.suggestions || [],
				message: `Smart tagging complete: ${result.tags_suggested} tag suggestions generated`,
			}
		}

		case 'detect_scenes': {
			const data = args.video_id ? { video_id: args.video_id } : {}
			const response = await aiAPI.detectScenes(data)
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_analyzed: result.videos_analyzed || 0,
				scenes_detected: result.scenes_detected || 0,
				results: result.results || [],
				message: `Scene detection complete: ${result.scenes_detected} scenes found`,
			}
		}

		case 'classify_content': {
			const response = await aiAPI.classifyContent({})
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_classified: result.videos_classified || 0,
				categories: result.categories || [],
				message: `Content classification complete: ${result.videos_classified} videos classified`,
			}
		}

		case 'analyze_video_quality': {
			const response = await aiAPI.analyzeQuality({})
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_analyzed: result.videos_analyzed || 0,
				low_quality_count: result.low_quality_count || 0,
				results: result.results || [],
				message: `Quality analysis complete: ${result.low_quality_count} low-quality videos found`,
			}
		}

		case 'detect_duplicates': {
			const response = await aiAPI.detectDuplicates({})
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_analyzed: result.videos_analyzed || 0,
				duplicate_groups: result.duplicate_groups || 0,
				results: result.results || [],
				message: `Duplicate detection complete: ${result.duplicate_groups} duplicate groups found`,
			}
		}

		case 'suggest_auto_naming': {
			const response = await aiAPI.suggestNaming({})
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_analyzed: result.videos_analyzed || 0,
				suggestions: result.suggestions || [],
				message: `Auto-naming complete: ${result.suggestions?.length || 0} naming suggestions generated`,
			}
		}

		case 'analyze_thumbnail_quality': {
			const response = await aiAPI.analyzeThumbnailQuality({})
			const result = Array.isArray(response) ? response : response.data || response
			return {
				success: true,
				videos_analyzed: result.videos_analyzed || 0,
				poor_quality_count: result.poor_quality_count || 0,
				results: result.results || [],
				message: `Thumbnail analysis complete: ${result.poor_quality_count} videos with poor thumbnails`,
			}
		}

		default:
			throw new Error(`Unknown tool: ${toolName}`)
	}
}
