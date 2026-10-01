# Thread Enhancement Feature

## Overview

AI-powered thread content enhancement that cleans up post content, extracts performer names, identifies content tags, and generates summaries.

## Architecture

### Backend Components

#### 1. ThreadEnhancer Service (`ai_companion_thread_enhancer.go`)
- **Core enhancement logic** for processing thread posts
- **Content cleaning** removes forum boilerplate, signatures, and clutter
- **Performer extraction** identifies names from post content using regex patterns
- **Tag extraction** identifies content categories (BBC, interracial, etc.)
- **Activity tracking** via ActivityService for progress updates

#### 2. AI Companion Tools (`ai_companion_tools_scraper.go`)
Three new tools registered for LLM use:

| Tool | Description |
|------|-------------|
| `enhance_thread` | Enhance a single thread with configurable options |
| `get_thread_enhancement_status` | Check status of enhancement jobs |
| `batch_enhance_threads` | Enhance multiple threads at once |

#### 3. API Endpoints (`scraper_handlers.go`)

```
POST /api/v1/scraper/threads/:id/enhance
GET  /api/v1/scraper/threads/:id/enhancement-status
```

### Frontend Components

#### Thread Page (`ScraperThreadPage.vue`)
- **AI Enhance Button** (purple gradient) in header actions
- **Enhancement Modal** with configurable options:
  - Clean Content
  - Extract Performers
  - Extract Tags
  - Generate Summary
  - **Validate Images** (NEW - uses Qwen3.8-omni-flash)
- **Progress polling** for real-time status updates

## Enhancement Options

```go
type EnhancementOptions struct {
    CleanContent      bool // Remove forum boilerplate
    ExtractPerformers bool // Extract performer names
    ExtractTags       bool // Extract content tags
    GenerateSummary   bool // Generate thread summary
    ValidateLinks     bool // Check link health (future)
    ValidateImages    bool // Validate images using AI vision (NEW)
}
```

## Content Cleaning Patterns

The enhancer removes common forum clutter:
- Signatures ("Sent from my iPhone")
- Tapatalk references
- Quote headers
- Reaction counts
- Navigation elements
- Excessive whitespace

## Performer Extraction

Uses regex patterns to find:
- "Starring: Name1, Name2"
- "Featuring: Name"
- "Models: Name"
- Names followed by "xxx", "adult", "model", "star"

Validates names:
- Must be 2+ words
- Max 50 characters
- Excludes common words (the, and, or, etc.)

## Tag Extraction

Identifies content categories:
- bbc, interracial, anal, blowjob, creampie
- threesome, group, teen, milf, amateur
- Professional studios (Brazzers, RealityKings, etc.)

## Image Validation (NEW)

Uses **Qwen3.8-omni-flash** for cost-effective image analysis:
- **Quality scoring**: 1-10 rating for each image
- **Content analysis**: Description and tags
- **Broken image detection**: Identifies invalid/unloadable images
- **NSFW detection**: Flags inappropriate content
- **Confidence scoring**: How certain the AI is about its analysis

### Cost Efficiency
- **Model**: Qwen3.8-omni-flash ($0.03/$0.13 per 1M tokens)
- **Batch processing**: 5 concurrent requests with rate limiting
- **Estimated cost**: ~$0.035 per 1000 images

## Usage

### Via AI Companion Chat
The AI can use these tools when users ask to:
- "Enhance this thread"
- "Clean up the posts"
- "Extract performers from thread 123"
- "Tag all posts in this thread"

### Via UI
1. Open a thread in the Scraper
2. Click the purple "AI Enhance" button (magic wand icon)
3. Select enhancement options
4. Click "Start Enhancement"
5. Monitor progress in the Activity panel

### Via API
```bash
# Enhance a thread with image validation
curl -X POST http://localhost:3000/api/v1/scraper/threads/123/enhance \
  -H "Content-Type: application/json" \
  -d '{
    "clean_content": true,
    "extract_performers": true,
    "extract_tags": true,
    "generate_summary": true,
    "validate_images": true
  }'

# Check status
curl http://localhost:3000/api/v1/scraper/threads/123/enhancement-status
```

## Future Enhancements

1. **AI Vision Analysis** - ✅ DONE: Uses Qwen3.8-omni-flash for cost-effective image analysis
2. **Link Validation** - Verify download links during enhancement
3. **Duplicate Detection** - Find similar threads
4. **Auto-Organization** - Move threads to correct categories
5. **Quality Scoring** - Rate thread content quality
6. **Smart Recommendations** - Suggest related threads

## Integration Points

- **Activity Service** - Progress tracking and logging
- **AI Companion** - LLM tool integration
- **WebSocket** - Real-time progress updates
- **Database** - Persistent metadata storage
- **Image Validator** - Qwen3.8-omni-flash integration for image analysis
