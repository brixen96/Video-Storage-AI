# Development Guidelines - Video Storage AI

## Purpose
This document outlines critical patterns and practices to prevent bugs and ensure consistency across the codebase. It was created after encountering issues with the Zoo field implementation.

## Critical Checklist for Adding New Database Fields

When adding a new field to any database model, **ALL** of the following steps MUST be completed:

### 1. Database Schema
- [ ] Add field to migration/schema
- [ ] Verify field type is compatible with SQLite and Go

### 2. Go Model Definition
- [ ] Add field to struct in `internal/models/*.go`
- [ ] Add field to Create struct (if applicable)
- [ ] Add field to Update struct (if applicable)
- [ ] Add proper JSON tags

### 3. Service Layer - **EVERY** Query Method Must Include the Field

**Critical**: Missing a field in ANY query method will cause the field to return zero/false values!

For each model service file (e.g., `performer_service.go`), update **ALL** of these methods:

#### Read Operations
- [ ] `GetAll()` - SELECT query and Scan
- [ ] `GetByID()` - SELECT query and Scan
- [ ] `GetAllPaginated()` - SELECT query and Scan ⚠️ **Commonly forgotten!**
- [ ] `Search()` - SELECT query and Scan (if exists)
- [ ] `SearchPaginated()` - SELECT query and Scan (if exists)
- [ ] Any other custom query methods

#### Write Operations
- [ ] `Create()` - INSERT query parameters
- [ ] `Update()` - UPDATE query parameters
- [ ] Handle field in update logic (check if pointer is nil)

### 4. Type Handling for Boolean Fields

SQLite doesn't have a native boolean type. When working with boolean fields:

**Storage**: Always convert Go `bool` to `int` (0 or 1) before storing:
```go
var boolInt int
if value {
    boolInt = 1
} else {
    boolInt = 0
}
db.Exec(query, ..., boolInt, ...)
```

**Retrieval**: Always scan into `interface{}` and use type switch:
```go
var boolVal interface{}
err := rows.Scan(..., &boolVal, ...)

switch v := boolVal.(type) {
case int64:
    model.Field = v != 0
case bool:
    model.Field = v
default:
    model.Field = false  // Safe default
}
```

### 5. Related Files to Update

When adding a field that spans multiple entities:

#### Backend
- [ ] API handlers (`internal/api/*_handlers.go`)
- [ ] Any related service files
- [ ] API tests (if they exist)

#### Frontend
- [ ] API service file (`src/services/api.js`)
- [ ] All relevant Vue components
- [ ] Type definitions (if using TypeScript)

## Common Mistakes and How to Avoid Them

### Mistake 1: Missing Field in Paginated Query
**Problem**: `GetAllPaginated()` was missing the `zoo` field while `GetAll()` had it.
**Impact**: Frontend displayed wrong data because it uses paginated endpoint.
**Prevention**: Use the checklist above. Search codebase for ALL occurrences of SELECT queries.

### Mistake 2: Inconsistent Type Handling
**Problem**: Some methods scanned directly into `bool`, others into `int`, creating mixed database types.
**Impact**: Database had mixed int64 and bool values, causing unpredictable behavior.
**Prevention**: Always use the `interface{}` + type switch pattern for booleans.

### Mistake 3: Not Testing All Code Paths
**Problem**: Tested with `GetAll()` endpoint but frontend actually used `GetAllPaginated()`.
**Impact**: Wasted time debugging the wrong method.
**Prevention**:
- Check browser network tab to see which API endpoint is actually being called
- Test with the actual frontend, not just API directly

### Mistake 4: Assuming Code is Running
**Problem**: Added debug logging but it never appeared, suggesting wrong code path.
**Impact**: Spent time debugging why code wasn't working when it wasn't even running.
**Prevention**:
- Verify which endpoint the frontend calls
- Add logging to multiple places initially
- Check compiled binary timestamp to ensure rebuild succeeded

## Testing Checklist

When implementing a new field:

### Backend Tests
- [ ] Create operation stores value correctly
- [ ] Read operation retrieves value correctly
- [ ] Update operation modifies value correctly
- [ ] Paginated queries include the field
- [ ] Search queries include the field
- [ ] Field appears in all API responses

### Frontend Tests
- [ ] Field displays correctly on all relevant pages
- [ ] Field updates when modified
- [ ] Changes persist after page refresh
- [ ] Field appears in create/edit forms
- [ ] Field appears in list and detail views

## Quick Reference: Finding All Query Methods

To find all query methods that need updating when adding a field:

```bash
# Find all SELECT queries for a model
grep -n "SELECT.*FROM performers" api/internal/services/performer_service.go

# Find all Scan operations
grep -n "Scan(" api/internal/services/performer_service.go

# Find all db.Exec for INSERT/UPDATE
grep -n "db.Exec" api/internal/services/performer_service.go
```

## When in Doubt

1. **Check existing similar fields**: Look at how other fields like `scene_count` are handled
2. **Search the entire codebase**: Use grep/search to find ALL occurrences
3. **Test with frontend immediately**: Don't assume backend changes work until frontend confirms
4. **Add temporary logging**: When debugging, add obvious logging to verify code execution
5. **Check the database directly**: Use a DB viewer to verify actual stored values

## API Response Structure Guidelines

### Backend Response Format
**ALL** API responses MUST use the standardized response format via `models.SuccessResponse()` or `models.ErrorResponseMsg()`:

```go
// Success response structure
{
  "success": true,
  "message": "Operation completed successfully",
  "data": <actual data>,
  "timestamp": "2025-11-16T05:24:57Z"
}

// Error response structure
{
  "success": false,
  "error": "Error description",
  "timestamp": "2025-11-16T05:24:57Z"
}
```

### Frontend API Response Handling

**Critical Rule**: The Axios response interceptor (in `src/services/api.js`) automatically unwraps `response.data`, returning the backend's JSON response object.

**This means:**
- Backend returns: `{success: true, data: [...], message: "..."}`
- Axios interceptor unwraps to: `response.data` → `{success: true, data: [...], message: "..."}`
- Frontend receives: `{success: true, data: [...], message: "..."}`
- **To get actual data**: Access `response.data` (NOT just `response`)

### Common Response Handling Mistakes

#### Mistake 5: Incorrect Response Data Access
**Problem**: Accessing `response` instead of `response.data` after API call
**Example of Wrong Code**:
```javascript
const response = await performersAPI.getTags(performerId)
this.performerMasterTags = response  // WRONG - sets to full response object
```

**Example of Correct Code**:
```javascript
const response = await performersAPI.getTags(performerId)
// response = {success: true, data: [...], message: "..."}
this.performerMasterTags = response.data  // CORRECT - extracts array from data field
```

**Impact**: Component receives full response object instead of actual data, causing:
- Data not displaying in UI
- Type errors (expecting array, got object)
- Confusing debugging (data exists but isn't the right shape)

**Prevention**:
1. **ALWAYS** verify the response structure before writing frontend code
2. **ALWAYS** access `response.data` to get the actual payload
3. **ALWAYS** add console logging during development to verify data shape
4. Check browser DevTools Network tab to see actual API response
5. For paginated or wrapped responses, access nested data: `response.data.items`, `response.data.videos_updated`, etc.

### Response Structure Verification Checklist

When implementing a new API endpoint or fixing data access:

**Backend** (in handlers):
- [ ] Verify using `models.SuccessResponse(data, message)` for success
- [ ] Verify using `models.ErrorResponseMsg(message, details)` for errors
- [ ] Ensure `data` parameter contains the actual payload (array, object, etc.)
- [ ] Don't wrap data in extra objects unless necessary (e.g., return `tags` array directly, not `{tags: [...]})`)

**Frontend** (in components):
- [ ] Check API response in DevTools Network tab
- [ ] Verify accessing `response.data` to get payload
- [ ] Add console.log during development to verify data shape
- [ ] Test with actual API, not mock data
- [ ] Verify computed properties receive correct data type

### Quick Debug Steps for Response Issues

1. **Add logging immediately**:
```javascript
const response = await api.call()
console.log('Full response:', response)
console.log('Response data:', response.data)
console.log('Type check:', Array.isArray(response.data))
```

2. **Check Network tab**: See the actual JSON returned by backend

3. **Verify interceptor**: Check `src/services/api.js` response interceptor hasn't changed

4. **Test backend directly**: Use curl/Postman to see raw response

### Backend Response Best Practices

1. **Keep data unwrapped when possible**:
```go
// GOOD - Direct data in response
c.JSON(http.StatusOK, models.SuccessResponse(tags, "Tags retrieved"))
// Results in: {success: true, data: [...tags...]}

// AVOID - Extra wrapper
c.JSON(http.StatusOK, models.SuccessResponse(gin.H{"tags": tags}, "Tags retrieved"))
// Results in: {success: true, data: {tags: [...]}} - requires response.data.tags
```

2. **Be consistent with response format** across all endpoints

3. **Document nested structures** when necessary (e.g., pagination)

## Summary

The key lessons:

1. **When adding a database field, you must update EVERY query method that touches that table, not just the ones you think are being used.**

2. **When implementing API endpoints, ALWAYS verify the response structure on both backend and frontend before marking as complete.**

3. **The frontend MUST access `response.data` to get the actual payload, never just `response`.**

Missing even one query method or incorrectly accessing response data will cause subtle bugs that waste time and money to track down.

---

## Mistake 6: Adding Model Fields Without Implementing Filter Logic

**Date**: 2025-11-16

**What Happened**: Added `TagID` and `Zoo` fields to the `VideoSearchQuery` struct in `models/video.go`, but forgot to implement the actual filtering logic in `video_service.go`. The backend accepted these parameters but silently ignored them, making filters appear to not work.

**Root Cause**: Incomplete implementation - added the API contract (model fields) without implementing the business logic (SQL WHERE conditions).

**Correct Pattern**:

When adding new filter fields to a search query:

1. **Add the field to the model struct** (e.g., `models/video.go`)
2. **Implement the filter logic in the service** (e.g., `video_service.go`)
3. **Test the filter** to verify it works before moving to frontend

**Example of WRONG approach**:
```go
// models/video.go - Added fields
type VideoSearchQuery struct {
    TagID int64 `json:"tag_id" form:"tag_id"`
    Zoo   *bool `json:"zoo" form:"zoo"`
    // ...
}

// video_service.go - FORGOT TO IMPLEMENT LOGIC!
// Filter silently ignored, no errors thrown
```

**Example of CORRECT approach**:
```go
// models/video.go - Add fields
type VideoSearchQuery struct {
    TagID int64 `json:"tag_id" form:"tag_id"`
    Zoo   *bool `json:"zoo" form:"zoo"`
    // ...
}

// video_service.go - IMPLEMENT THE LOGIC
if query.TagID > 0 {
    joins = append(joins, "INNER JOIN video_tags vt2 ON v.id = vt2.video_id")
    conditions = append(conditions, "vt2.tag_id = ?")
    args = append(args, query.TagID)
}

if query.Zoo != nil {
    joins = append(joins, "INNER JOIN video_performers vp2 ON v.id = vp2.video_id")
    joins = append(joins, "INNER JOIN performers p ON vp2.performer_id = p.id")
    if *query.Zoo {
        conditions = append(conditions, "p.zoo = 1")
    } else {
        conditions = append(conditions, "p.zoo = 0")
    }
}
```

**Important Notes**:
- The `zoo` field exists on the **performers** table, not the videos table
- To filter videos by zoo content, join with `video_performers` and `performers` tables
- Zoo filtering checks if ANY performer in the video has `zoo = true`
- Always use unique join aliases (vt2, vp2, p) to avoid conflicts with existing joins

**Checklist for Adding Filters**:
- [ ] Add field to query model struct
- [ ] Implement SQL logic in service layer
- [ ] Handle JOINs if filtering by related tables
- [ ] Use correct SQL syntax for the data type (int, bool, string)
- [ ] Test the filter in the API before touching frontend
- [ ] Verify the filter in browser DevTools Network tab

---

## Mistake 7: CSS Class Name Collisions with Keep-Alive

**Date**: 2025-11-16

**What Happened**: When navigating from PerformersPage to VideosPage and back, the performer cards became extremely small and the card-info was missing. The page rendered fine on initial load, but broke on subsequent visits.

**Root Cause**: Multiple CSS class name collisions between `performers_page.css` and `videos_page.css`:
- `.performer-card` - Different sizing and aspect ratios
- `.performer-meta` - Different flex direction (row vs column)
- `.meta-item` - Different styling

When using Vue's `<keep-alive>` component, CSS from both pages remained active, causing the VideosPage styles to override the PerformersPage styles.

**Why It Happened**:
1. **`@import` in `<style scoped>` doesn't scope the imported CSS** - This is a Vue limitation
2. **Keep-Alive caches components** - Both component styles remain active in the DOM
3. **CSS specificity is the same** - Both use `.performer-card`, so load order determines which wins

**Correct Pattern**:

When creating page-specific CSS classes:

1. **Use unique class names per page/component** - Prefix with page name (e.g., `.video-performer-card` vs `.performer-card`)
2. **Never reuse generic class names across different pages**
3. **If using `<keep-alive>`, be extra careful about class name uniqueness**

**Example of WRONG approach**:
```css
/* performers_page.css */
.performer-card {
    grid-template-columns: repeat(auto-fill, minmax(650px, 1fr));
}

/* videos_page.css */
.performer-card {  /* COLLISION! */
    aspect-ratio: 16 / 9;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
}
```

**Example of CORRECT approach**:
```css
/* performers_page.css */
.performer-card {
    grid-template-columns: repeat(auto-fill, minmax(650px, 1fr));
}

/* videos_page.css */
.video-performer-card {  /* UNIQUE NAME */
    aspect-ratio: 16 / 9;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
}

.video-performer-meta {  /* UNIQUE NAME */
    flex-direction: column;
}

.video-meta-item {  /* UNIQUE NAME */
    font-size: 0.8rem;
}
```

**Complete Solution - All VideosPage Classes Prefixed with `vp-`**:

To completely eliminate CSS conflicts, ALL classes in `videos_page.css` were systematically renamed with a `vp-` prefix:

**Layout Classes**:
- `.top-bar` → `.vp-top-bar`
- `.page-content` → `.vp-page-content`
- `.filter-sidebar` → `.vp-filter-sidebar`
- `.filter-panel` → `.vp-filter-panel`
- `.filter-group` → `.vp-filter-group`
- `.main-content` → `.vp-main-content`
- `.video-grid` → `.vp-video-grid`
- `.video-list` → `.vp-video-list`

**Panel & Details Classes**:
- `.video-details-panel` → `.vp-video-details-panel`
- `.panel-header` → `.vp-panel-header`
- `.panel-body` → `.vp-panel-body`
- `.detail-section` → `.vp-detail-section`
- `.info-grid` → `.vp-info-grid`
- `.info-item` → `.vp-info-item`

**Performer Classes**:
- `.performers-grid` → `.vp-performers-grid`
- `.performers-list` → `.vp-performers-list`
- `.performer-card` → `.vp-performer-card`
- `.performer-item` → `.vp-performer-item`
- `.performer-preview-video` → `.vp-performer-preview-video`
- `.performer-image` → `.vp-performer-image`
- `.performer-placeholder` → `.vp-performer-placeholder`
- `.performer-info` → `.vp-performer-info`
- `.performer-meta` → `.vp-performer-meta`
- `.performer-stats` → `.vp-performer-stats`

**Meta & Content Classes**:
- `.meta-item` → `.vp-meta-item`
- `.stat-badge` → `.vp-stat-badge`
- `.action-buttons` → `.vp-action-buttons`
- `.studio-item` → `.vp-studio-item`
- `.tags-container` → `.vp-tags-container`
- `.tag-chip` → `.vp-tag-chip`

**Pagination & Menu Classes**:
- `.pagination-controls` → `.vp-pagination-controls`
- `.page-item` → `.vp-page-item`
- `.page-link` → `.vp-page-link`
- `.context-menu` → `.vp-context-menu`
- `.context-menu-item` → `.vp-context-menu-item`

**Total**: 40+ classes renamed with `vp-` prefix

**Important Notes**:
- `<style scoped>` only scopes CSS written directly in the component, NOT imported CSS files
- When using `<keep-alive>`, multiple components' CSS can be active simultaneously
- Always use unique, descriptive class names (prefix with page/component name)
- Test navigation between pages when using `<keep-alive>` to catch CSS conflicts

**Checklist for Page-Specific Styling**:
- [ ] Use unique class names prefixed with page/component name
- [ ] Avoid generic class names like `.card`, `.item`, `.container` in page-level CSS
- [ ] Test with `<keep-alive>` - navigate away and back to verify styles persist
- [ ] Consider using CSS modules or Vue's scoped styles for true isolation
- [ ] Search codebase for class name before creating new ones to avoid conflicts

---

## Critical UI/Styling Rules

### RULE: NEVER Use `text-muted` on Dark Backgrounds

**Rule**: **NEVER** use the Bootstrap class `text-muted` on dark backgrounds. **ALWAYS** use `text-light` instead.

**Why**: The `text-muted` class applies a dark gray color (`#6c757d`) which is nearly invisible on dark backgrounds. This creates poor contrast and makes text unreadable.

**Correct Pattern**:
```vue
<!-- WRONG -->
<p class="text-muted">This text is dark on dark background</p>

<!-- CORRECT -->
<p class="text-light">This text is light on dark background</p>
```

**When to Use Each**:
- **`text-muted`**: Only use on light backgrounds (white, light gray)
- **`text-light`**: Use on dark backgrounds (dark blue, black, dark gray)
- **`text-white`**: Use when you need pure white text on dark backgrounds

**Application**: This applies to ALL pages with dark backgrounds, including:
- TasksPage
- PerformersPage
- VideosPage
- Any other page using dark theme (`#0f0c29`, `#1a2942`, etc.)

**Checklist for Text Styling**:
- [ ] Check background color before choosing text class
- [ ] Use `text-light` or `text-white` on all dark backgrounds
- [ ] Never use `text-muted` unless background is light
- [ ] Test visibility by viewing the actual page, not just in code
