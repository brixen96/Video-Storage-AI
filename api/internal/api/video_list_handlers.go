package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

// Video list service singleton
var videoListService *services.VideoListService

// ensureVideoListService returns the video list service singleton
func ensureVideoListService() *services.VideoListService {
	if videoListService == nil {
		videoListService = services.NewVideoListService(database.GetDB())
	}
	return videoListService
}

// getAllListsHandler returns all video lists
func getAllListsHandler(c *gin.Context) {
	svc := ensureVideoListService()

	lists, err := svc.GetAllLists()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get lists",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(lists, "Lists retrieved successfully"))
}

// getListByIDHandler returns a single list by ID
func getListByIDHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	svc := ensureVideoListService()

	list, err := svc.GetListByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"List not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(list, "List retrieved successfully"))
}

// createListHandler creates a new video list
func createListHandler(c *gin.Context) {
	var create models.VideoListCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	svc := ensureVideoListService()

	list, err := svc.CreateList(&create)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create list",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(list, "List created successfully"))
}

// updateListHandler updates an existing video list
func updateListHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	var update models.VideoListUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	svc := ensureVideoListService()

	list, err := svc.UpdateList(id, &update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update list",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(list, "List updated successfully"))
}

// deleteListHandler deletes a video list
func deleteListHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	svc := ensureVideoListService()

	err = svc.DeleteList(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete list",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "List deleted successfully"))
}

// getListItemsHandler returns all items in a list
func getListItemsHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	svc := ensureVideoListService()

	// Get list details
	list, err := svc.GetListByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"List not found",
			err.Error(),
		))
		return
	}

	// Get items
	items, err := svc.GetListItems(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get items",
			err.Error(),
		))
		return
	}

	// Get stats
	stats, err := svc.GetListStats(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"list":  list,
		"items": items,
		"stats": stats,
	}, "List items retrieved successfully"))
}

// addToListHandler adds videos to a list
func addToListHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	var request models.AddToListRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if len(request.VideoIDs) == 0 && len(request.ScrapedVideoIDs) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			"At least one video ID or scraped video ID is required",
		))
		return
	}

	svc := ensureVideoListService()

	err = svc.AddVideosToList(id, &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to add videos",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Videos added to list successfully"))
}

// removeFromListHandler removes videos from a list
func removeFromListHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	var request models.RemoveFromListRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if len(request.VideoIDs) == 0 && len(request.ScrapedVideoIDs) == 0 && len(request.ItemIDs) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			"At least one video ID, scraped video ID, or item ID is required",
		))
		return
	}

	svc := ensureVideoListService()

	err = svc.RemoveVideosFromList(id, &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to remove videos",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Videos removed from list successfully"))
}

// getListStatsHandler returns statistics about a list
func getListStatsHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	svc := ensureVideoListService()

	stats, err := svc.GetListStats(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Stats retrieved successfully"))
}

// getListsForVideoHandler returns all lists containing a video
func getListsForVideoHandler(c *gin.Context) {
	videoID, err := strconv.ParseInt(c.Query("video_id"), 10, 64)
	if err != nil && c.Query("video_id") != "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid video ID",
			"Video ID must be a number",
		))
		return
	}

	scrapedID, err := strconv.ParseInt(c.Query("scraped_video_id"), 10, 64)
	if err != nil && c.Query("scraped_video_id") != "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid scraped video ID",
			"Scraped video ID must be a number",
		))
		return
	}

	if videoID == 0 && scrapedID == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			"Either video_id or scraped_video_id is required",
		))
		return
	}

	svc := ensureVideoListService()

	var lists []models.VideoList
	if scrapedID > 0 {
		lists, err = svc.GetListsForVideo(scrapedID, true)
	} else {
		lists, err = svc.GetListsForVideo(videoID, false)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get lists",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(lists, "Lists retrieved successfully"))
}

// reorderListItemsHandler reorders items within a list
func reorderListItemsHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			"List ID must be a number",
		))
		return
	}

	var request struct {
		ItemIDs []int64 `json:"item_ids"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	svc := ensureVideoListService()

	err = svc.ReorderListItems(id, request.ItemIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to reorder items",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Items reordered successfully"))
}

// updateItemNotesHandler updates the notes for a list item
func updateItemNotesHandler(c *gin.Context) {
	listID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid list ID",
			"List ID must be a number",
		))
		return
	}

	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid item ID",
			"Item ID must be a number",
		))
		return
	}

	var request struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	svc := ensureVideoListService()

	err = svc.UpdateItemNotes(listID, itemID, request.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update notes",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Notes updated successfully"))
}
