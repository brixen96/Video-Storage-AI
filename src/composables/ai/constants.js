/**
 * Static configuration for AI Companion — prompts, commands, labels, rules.
 * Import from composables/ai/constants.js
 */

// Human-readable labels for Vue Router view names
export const PAGE_LABELS = {
	Home: 'Dashboard', Videos: 'Videos Library', VideoPlayer: 'Video Player',
	ScrapedVideoPlayer: 'Video Player', Performers: 'Performers', PerformerDetails: 'Performer details',
	Libraries: 'Libraries', Browser: 'File Browser', Studios: 'Studios', Tags: 'Tags',
	Activity: 'Activity Feed', Settings: 'Settings', Tasks: 'Tasks', EditList: 'Edit List',
	Scraper: 'Scraper', ScraperThread: 'Scraper Thread', SystemHealth: 'System Health',
	Backups: 'Backups', JDownloader: 'JDownloader', AI: 'AI Config', Analyzer: 'Video Analyzer',
}

// Context-aware suggested prompts — keyed by Vue Router view name
export const PAGE_SUGGESTIONS = {
	Home: [
		{ text: "Give me today's briefing", icon: ['fas', 'sun'] },
		{ text: "What's happening right now?", icon: ['fas', 'bolt'] },
		{ text: 'Library overview', icon: ['fas', 'chart-bar'] },
		{ text: 'Check for errors', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Show library insights', icon: ['fas', 'lightbulb'] },
		{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
	],
	Videos: [
		{ text: 'Find duplicate videos', icon: ['fas', 'copy'] },
		{ text: 'Auto-tag my untagged videos', icon: ['fas', 'tags'] },
		{ text: 'Show my top rated videos', icon: ['fas', 'star'] },
		{ text: 'What videos are missing metadata?', icon: ['fas', 'exclamation-circle'] },
		{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
		{ text: 'Show my favorite videos', icon: ['fas', 'heart'] },
	],
	Browser: [
		{ text: 'Help me organize my downloads', icon: ['fas', 'folder-open'] },
		{ text: "What's in my Downloads library?", icon: ['fas', 'download'] },
		{ text: 'Find unmapped video files', icon: ['fas', 'search-location'] },
		{ text: 'Scan for orphaned files', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Move these files to Backup', icon: ['fas', 'hdd'] },
		{ text: 'Which files should I keep?', icon: ['fas', 'question-circle'] },
	],
	Performers: [
		{ text: 'Who are my top performers?', icon: ['fas', 'users'] },
		{ text: 'Tell me about my top performer', icon: ['fas', 'user-circle'] },
		{ text: 'Find performers without photos', icon: ['fas', 'camera'] },
		{ text: 'Show performer statistics', icon: ['fas', 'chart-bar'] },
		{ text: 'Merge duplicate performers', icon: ['fas', 'user-check'] },
	],
	PerformerDetails: [
		{ text: 'Summarise this performer', icon: ['fas', 'user-circle'] },
		{ text: 'Find all videos with this performer', icon: ['fas', 'film'] },
		{ text: 'Suggest tags for this performer', icon: ['fas', 'tags'] },
		{ text: 'Rate all unwatched videos for this performer', icon: ['fas', 'star'] },
	],
	Libraries: [
		{ text: 'Check library health', icon: ['fas', 'heartbeat'] },
		{ text: 'Find missing files in my libraries', icon: ['fas', 'exclamation-circle'] },
		{ text: 'How much storage am I using?', icon: ['fas', 'hdd'] },
		{ text: 'Scan my libraries for new videos', icon: ['fas', 'sync-alt'] },
	],
	Studios: [
		{ text: 'What are my top studios?', icon: ['fas', 'building'] },
		{ text: 'Show studio statistics', icon: ['fas', 'chart-bar'] },
		{ text: 'Find videos without a studio', icon: ['fas', 'exclamation-circle'] },
		{ text: 'Merge duplicate studios', icon: ['fas', 'clone'] },
	],
	Tags: [
		{ text: 'What are my most used tags?', icon: ['fas', 'tags'] },
		{ text: 'Find videos with no tags', icon: ['fas', 'tag'] },
		{ text: 'Auto-tag my untagged videos', icon: ['fas', 'magic'] },
		{ text: 'Suggest tag categories for my library', icon: ['fas', 'sitemap'] },
	],
	Activity: [
		{ text: "What's happened recently?", icon: ['fas', 'history'] },
		{ text: 'Show scraper activity', icon: ['fas', 'spider'] },
		{ text: 'Any errors in the last 24 hours?', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Summarise this week', icon: ['fas', 'chart-bar'] },
	],
	Tasks: [
		{ text: "What tasks are running right now?", icon: ['fas', 'circle-notch'] },
		{ text: 'Show completed tasks', icon: ['fas', 'check-circle'] },
		{ text: 'Are there any failed tasks?', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Cancel all running tasks', icon: ['fas', 'stop'] },
	],
	Scraper: [
		{ text: "What's scraping right now?", icon: ['fas', 'spider'] },
		{ text: 'Scraper summary', icon: ['fas', 'chart-bar'] },
		{ text: 'Check for scraper errors', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Show recent scraper activity', icon: ['fas', 'history'] },
	],
	ScraperThread: [
		{ text: 'Summarise this scraper', icon: ['fas', 'spider'] },
		{ text: 'Show errors for this scraper', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'What did this scraper find recently?', icon: ['fas', 'history'] },
	],
	JDownloader: [
		{ text: 'JDownloader status', icon: ['fas', 'download'] },
		{ text: "What's downloading right now?", icon: ['fas', 'circle-notch'] },
		{ text: 'Show download queue', icon: ['fas', 'list'] },
		{ text: 'Any failed downloads?', icon: ['fas', 'exclamation-triangle'] },
	],
	Analyzer: [
		{ text: 'Which file is best for Topaz upscaling?', icon: ['fas', 'star'] },
		{ text: 'Explain the sharpness scores', icon: ['fas', 'search-plus'] },
		{ text: 'Find my highest quality videos', icon: ['fas', 'trophy'] },
		{ text: 'What does perceptual efficiency mean?', icon: ['fas', 'question-circle'] },
	],
	AI: [
		{ text: "Give me today's briefing", icon: ['fas', 'sun'] },
		{ text: 'Show library insights', icon: ['fas', 'lightbulb'] },
		{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
		{ text: 'Find duplicate videos', icon: ['fas', 'copy'] },
	],
	EditList: [
		{ text: 'Help me build an edit list', icon: ['fas', 'film'] },
		{ text: 'Find videos for a compilation', icon: ['fas', 'list-check'] },
		{ text: 'Suggest scenes by performer', icon: ['fas', 'user'] },
	],
	SystemHealth: [
		{ text: 'How is my system doing?', icon: ['fas', 'heartbeat'] },
		{ text: 'Check for errors in the last scan', icon: ['fas', 'exclamation-triangle'] },
		{ text: 'Show database statistics', icon: ['fas', 'database'] },
	],
}

// Fallback prompts when no page-specific set exists
export const GENERIC_PROMPTS = [
	{ text: "What's happening right now?", icon: ['fas', 'bolt'] },
	{ text: 'Library overview', icon: ['fas', 'chart-bar'] },
	{ text: 'Check for errors', icon: ['fas', 'exclamation-triangle'] },
	{ text: 'Show library insights', icon: ['fas', 'lightbulb'] },
	{ text: 'What should I watch next?', icon: ['fas', 'play-circle'] },
	{ text: "Give me today's briefing", icon: ['fas', 'sun'] },
]

// /command palette definitions
export const SLASH_COMMANDS = [
	{ cmd: 'briefing', icon: ['fas', 'sun'], label: "Today's Briefing", fill: "Give me today's briefing" },
	{ cmd: 'status', icon: ['fas', 'bolt'], label: 'Current Status', fill: "What's happening right now?" },
	{ cmd: 'health', icon: ['fas', 'heartbeat'], label: 'Library Health', fill: "How's my library doing?" },
	{ cmd: 'search', icon: ['fas', 'search'], label: 'Search Videos', fill: 'Search for videos about ', param: true },
	{ cmd: 'tag', icon: ['fas', 'tags'], label: 'Auto-tag Untagged', fill: 'Auto-tag my untagged videos' },
	{ cmd: 'duplicates', icon: ['fas', 'copy'], label: 'Find Duplicates', fill: 'Find duplicate videos in my library' },
	{ cmd: 'performers', icon: ['fas', 'users'], label: 'Top Performers', fill: 'What are my top performers?' },
	{ cmd: 'watch', icon: ['fas', 'play-circle'], label: 'Watch Next', fill: 'What should I watch next?' },
	{ cmd: 'favorites', icon: ['fas', 'heart'], label: 'My Favorites', fill: 'Show my favorite videos' },
	{ cmd: 'errors', icon: ['fas', 'exclamation-circle'], label: 'Check Errors', fill: 'Check for errors' },
	{ cmd: 'orphaned', icon: ['fas', 'exclamation-triangle'], label: 'Orphaned Files', fill: 'Scan for orphaned files in my library' },
	{ cmd: 'stats', icon: ['fas', 'chart-bar'], label: 'Library Stats', fill: 'Library overview' },
	{ cmd: 'tasks', icon: ['fas', 'tasks'], label: 'Running Tasks', fill: 'What tasks are currently running?' },
	{ cmd: 'jdownloader', icon: ['fas', 'download'], label: 'JDownloader Status', fill: 'JDownloader status' },
	{ cmd: 'scrapers', icon: ['fas', 'spider'], label: 'Scraper Summary', fill: 'Scraper summary' },
]

// Heuristic follow-up action chips derived from response content
export const FOLLOW_UP_RULES = [
	{ pattern: /untagged|no tags|missing tags/i, label: 'Auto-tag now', message: 'Auto-tag my untagged videos', icon: ['fas', 'tags'] },
	{ pattern: /duplicate/i, label: 'Find duplicates', message: 'Find duplicate videos in my library', icon: ['fas', 'copy'] },
	{ pattern: /orphaned/i, label: 'Scan orphaned files', message: 'Scan for orphaned files in my library', icon: ['fas', 'exclamation-triangle'] },
	{ pattern: /thumbnail.{0,40}(missing|broken|fail|unavailable)|missing.{0,20}thumbnail/i, label: 'Fix thumbnails', message: 'Run a health check to fix thumbnails', icon: ['fas', 'image'] },
	{ pattern: /error.{0,40}(scraper|download|jdownloader)|(scraper|download|jdownloader).{0,40}error/i, label: 'Check errors', message: 'Check for errors', icon: ['fas', 'exclamation-circle'] },
	{ pattern: /watch next|recommend.{0,20}watch|what.{0,20}watch/i, label: 'Suggest something to watch', message: 'What should I watch next?', icon: ['fas', 'play-circle'] },
	{ pattern: /daily briefing|today.{0,10}summary/i, label: "Today's briefing", message: "Give me today's briefing", icon: ['fas', 'sun'] },
	{ pattern: /top.{0,10}performer|best performer/i, label: 'Top performers', message: 'What are my top performers?', icon: ['fas', 'users'] },
	{ pattern: /top.{0,10}rated|highest rated|best rated/i, label: 'Top rated videos', message: 'Show my top rated videos', icon: ['fas', 'star'] },
]

// Friendly labels for tool names shown during streaming
export const TOOL_LABELS = {
	search_videos: 'Searching videos',
	get_video_details: 'Fetching video details',
	update_video_metadata: 'Updating video metadata',
	get_favorites: 'Loading favorites',
	get_top_rated: 'Loading top-rated videos',
	set_video_rating: 'Setting rating',
	toggle_video_favorite: 'Toggling favorite',
	suggest_videos_to_watch: 'Finding watch suggestions',
	get_library_videos: 'Loading library videos',
	find_duplicate_videos: 'Scanning for duplicates',
	find_orphaned_videos: 'Scanning for orphaned files',
	delete_video_records: 'Deleting records',
	find_untagged_videos: 'Finding untagged videos',
	bulk_apply_tags: 'Bulk applying tags',
	suggest_tags_for_video: 'Suggesting tags',
	apply_tags_to_video: 'Applying tags',
	remove_tags_from_video: 'Removing tags',
	get_all_tags: 'Loading tags',
	search_performers: 'Searching performers',
	get_top_performers: 'Loading top performers',
	get_performer_stats: 'Loading performer stats',
	link_performer_to_video: 'Linking performer',
	unlink_performer_from_video: 'Unlinking performer',
	get_all_libraries: 'Loading libraries',
	scan_library_structure: 'Scanning library structure',
	analyze_library_structure: 'Analyzing library structure',
	suggest_video_placement: 'Finding best placement',
	move_video_to_library: 'Moving video',
	list_folder_contents: 'Listing folder contents',
	create_folder: 'Creating folder',
	rename_folder: 'Renaming folder',
	delete_folder: 'Deleting folder',
	rename_video_file: 'Renaming file',
	delete_video_file: 'Deleting file',
	get_scraper_status: 'Checking scrapers',
	get_jdownloader_status: 'Checking JDownloader',
	search_scraped_content: 'Searching scraped content',
	suggest_scraper_config: 'Analyzing scraper config',
	save_memory: 'Saving memory',
	recall_memories: 'Recalling memories',
	get_video_link_stats: 'Loading link stats',
	get_unlinked_videos: 'Finding unlinked videos',
	auto_link_by_path: 'Auto-linking by path',
	get_library_stats: 'Loading library stats',
	get_daily_briefing: 'Preparing briefing',
	get_library_health: 'Checking library health',
	get_running_tasks: 'Checking running tasks',
	get_recent_activity: 'Loading recent activity',
	get_recent_errors: 'Checking errors',
}

// Welcome message shown on first open
export const WELCOME_MESSAGE = `# 👋 Hello! I'm your AI Companion

I'm the **intelligent core** of Video Storage AI with advanced capabilities:

## 🎯 What I Can Do:
- 📊 **Analyze** your video library health and statistics
- 🔍 **Find** duplicates, performers, and specific content
- ❤️ **List** your favorites and top-rated videos
- 📈 **Predict** library growth and provide insights
- ⭐ **Evaluate** performer quality and metadata completeness
- 🚨 **Monitor** for errors and issues
- 💡 **Suggest** optimizations and improvements

## 🧠 Smart Features:
- **Markdown Support** - I can format responses beautifully
- **Code Highlighting** - Share code snippets with syntax colors
- **Memory System** - I remember our conversations
- **Real-time Data** - All answers are from your actual database

Try one of the suggested prompts above, or ask me anything about your library!`
