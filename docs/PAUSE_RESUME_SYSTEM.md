# Task Pause/Resume System

## Overview

This document describes the pause/resume functionality that allows long-running tasks (especially forum scraping) to be paused and resumed without losing progress. This is critical for operations that can take hours and may need to be interrupted for API/frontend restarts.

## Architecture

### Database Schema

**New Columns Added to `activities` and `activity_logs` tables:**
- `is_paused` (BOOLEAN): Whether the task is currently paused
- `paused_at` (DATETIME): When the task was paused
- `checkpoint` (TEXT/JSON): Serialized checkpoint data containing task state

### Models

**Activity & ActivityLog Models Enhanced:**
```go
type Activity struct {
    // ... existing fields ...
    IsPaused      bool                   `json:"is_paused"`
    PausedAt      *time.Time             `json:"paused_at,omitempty"`
    Checkpoint    string                 `json:"-"`  // DB storage
    CheckpointObj map[string]interface{} `json:"checkpoint,omitempty"`  // Runtime
}
```

**Methods:**
- `UnmarshalCheckpoint()` - Converts JSON checkpoint string to map
- `MarshalCheckpoint()` - Converts checkpoint map to JSON string

### Service Layer

**ActivityService New Methods:**

1. **PauseTask(id int64, checkpoint map[string]interface{}) error**
   - Pauses a running task
   - Saves checkpoint state
   - Broadcasts update to frontend via WebSocket
   - Only works on tasks with status='running'

2. **ResumeTask(id int64) (map[string]interface{}, error)**
   - Resumes a paused task
   - Returns checkpoint data so task can continue where it left off
   - Clears paused state
   - Broadcasts update to frontend

3. **SaveCheckpoint(id int64, checkpoint map[string]interface{}) error**
   - Saves checkpoint data without pausing
   - Useful for periodic progress saving during long operations
   - Allows recovery if process crashes

## Checkpoint Data Structure

The checkpoint is a flexible JSON object that can store any task-specific state. For forum scraping:

```json
{
  "current_thread_index": 45,
  "threads_completed": 44,
  "total_threads": 500,
  "current_thread_url": "https://...",
  "current_page": 3,
  "posts_scraped": 1234,
  "download_links_found": 89
}
```

## Usage Pattern

### 1. Task Implementation

Long-running tasks should:

```go
// Check if resuming from pause
checkpoint, err := activityService.ResumeTask(activityID)
startIndex := 0
if checkpoint != nil && checkpoint["current_thread_index"] != nil {
    startIndex = int(checkpoint["current_thread_index"].(float64))
}

// Main task loop
for i := startIndex; i < totalItems; i++ {
    // Check if task was paused
    activity, _ := activityService.GetByID(activityID)
    if activity.IsPaused {
        log.Printf("Task paused at item %d", i)
        return nil  // Exit gracefully
    }

    // Do work...

    // Save checkpoint periodically (every N items or time interval)
    if i % 10 == 0 {
        activityService.SaveCheckpoint(activityID, map[string]interface{}{
            "current_index": i,
            "items_completed": i,
        })
    }
}
```

### 2. API Endpoints (To Be Implemented)

```
POST /api/v1/activity/:id/pause
  - Pauses a running task
  - Body: { "checkpoint": { ... } }

POST /api/v1/activity/:id/resume
  - Resumes a paused task
  - Returns: checkpoint data

GET /api/v1/activity/:id
  - Includes is_paused, paused_at, checkpoint fields
```

### 3. Frontend Integration (To Be Implemented)

**ActivityTracker Component:**
- Show pause/resume button for running tasks
- Display "Paused" badge when task is paused
- Allow resuming with one click

**Implementation Notes:**
- Pause button should save current progress checkpoint before pausing
- Resume button should call resume API and restart the background task
- Handle cases where task completion happens while paused

## Benefits

1. **No Progress Loss**: Can pause hours-long scraping operations and resume exactly where left off
2. **Graceful Shutdown**: API can be restarted without losing scraping progress
3. **Resource Management**: Pause tasks during high-load periods
4. **User Control**: Users can manually pause/resume tasks as needed
5. **Crash Recovery**: Periodic checkpoint saving allows recovery from unexpected crashes

## Implementation Status

### ✅ Completed
- Database migrations for pause/resume columns
- Activity/ActivityLog model updates
- ActivityService pause/resume/checkpoint methods
- Checkpoint serialization/deserialization
- WebSocket broadcast integration

### 🔄 To Do
- API endpoints (POST /activity/:id/pause, POST /activity/:id/resume)
- Frontend pause/resume buttons in ActivityTracker
- Scraper checkpoint implementation
- Periodic checkpoint saving in long-running tasks
- Resume task auto-restart on app startup
- Testing with real forum scraping workload

## Technical Considerations

### Thread Safety
- Checkpoint saving should be thread-safe
- Use mutex if multiple goroutines access same task

### Checkpoint Size
- Keep checkpoint data minimal
- Only store essential state for resumption
- Avoid storing large data structures

### Resume Validation
- Verify checkpoint data integrity before resuming
- Handle cases where checkpoint data is corrupt or incomplete
- Provide fallback to restart from beginning if needed

### Auto-Resume on Startup
- Option to automatically resume paused tasks when API starts
- Query for paused tasks: `SELECT * FROM activity_logs WHERE is_paused = 1 AND status = 'running'`
- Present to user or auto-resume based on configuration

## Example: Forum Scraper Integration

```go
func (s *ScraperService) ScrapeForumAndSaveAll(forumURL string) error {
    activity, _ := s.activityService.StartTask("forum_scrape", ...)

    // Check for existing checkpoint
    existingActivity, _ := s.activityService.GetByID(activity.ID)
    var startIndex int
    if existingActivity.IsPaused {
        checkpoint, err := s.activityService.ResumeTask(activity.ID)
        if err == nil && checkpoint["thread_index"] != nil {
            startIndex = int(checkpoint["thread_index"].(float64))
        }
    }

    threads, _ := s.ScrapeForumCategory(forumURL)

    for i := startIndex; i < len(threads); i++ {
        // Check pause flag before each thread
        current, _ := s.activityService.GetByID(activity.ID)
        if current.IsPaused {
            return nil  // Graceful exit
        }

        // Scrape thread
        s.ScrapeThreadComplete(threads[i].URL)

        // Save checkpoint every 10 threads
        if i % 10 == 0 {
            s.activityService.SaveCheckpoint(activity.ID, map[string]interface{}{
                "thread_index": i,
                "threads_completed": i,
                "total_threads": len(threads),
            })
        }
    }

    s.activityService.CompleteTask(activity.ID, "Forum scrape completed")
    return nil
}
```

## Future Enhancements

1. **Auto-pause on errors**: Automatically pause instead of failing when encountering temporary errors
2. **Scheduled pause**: Pause tasks at specific times (e.g., pause scraping during business hours)
3. **Priority queueing**: Higher priority tasks can pause lower priority ones
4. **Checkpoint history**: Keep history of checkpoints for debugging
5. **Resume from specific checkpoint**: Allow choosing which checkpoint to resume from
