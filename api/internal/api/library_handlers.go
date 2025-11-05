package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var libraryService *services.LibraryService

// ensureLibraryService initializes the service if needed
func ensureLibraryService() *services.LibraryService {
	if libraryService == nil {
		libraryService = services.NewLibraryService()
	}
	return libraryService
}

// getLibraries retrieves all libraries
func getLibraries(c *gin.Context) {
	svc := ensureLibraryService()

	libraries, err := svc.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve libraries",
			err.Error(),
		))
		return
	}

	// Enrich with video counts
	for i := range libraries {
		count, _ := svc.GetVideoCount(libraries[i].ID)
		libraries[i].VideoCount = count
	}

	c.JSON(http.StatusOK, models.SuccessResponse(libraries, "Libraries retrieved successfully"))
}

// getLibrary retrieves a single library by ID
func getLibrary(c *gin.Context) {
	svc := ensureLibraryService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid library ID",
			err.Error(),
		))
		return
	}

	library, err := svc.GetWithStats(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Library not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(library, "Library retrieved successfully"))
}

// getPrimaryLibrary retrieves the primary library
func getPrimaryLibrary(c *gin.Context) {
	svc := ensureLibraryService()

	library, err := svc.GetPrimary()
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"No primary library found",
			err.Error(),
		))
		return
	}

	// Enrich with stats
	stats, err := svc.GetWithStats(library.ID)
	if err != nil {
		// If stats fail, return library without stats
		c.JSON(http.StatusOK, models.SuccessResponse(library, "Primary library retrieved successfully"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Primary library retrieved successfully"))
}

// createLibrary creates a new library
func createLibrary(c *gin.Context) {
	svc := ensureLibraryService()
	var create models.LibraryCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	library, err := svc.Create(&create)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create library",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(library, "Library created successfully"))
}

// updateLibrary updates an existing library
func updateLibrary(c *gin.Context) {
	svc := ensureLibraryService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid library ID",
			err.Error(),
		))
		return
	}

	var update models.LibraryUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	library, err := svc.Update(id, &update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update library",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(library, "Library updated successfully"))
}

// deleteLibrary deletes a library
func deleteLibrary(c *gin.Context) {
	svc := ensureLibraryService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid library ID",
			err.Error(),
		))
		return
	}

	err = svc.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete library",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Library deleted successfully"))
}
