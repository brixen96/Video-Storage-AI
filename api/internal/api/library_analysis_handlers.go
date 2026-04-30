package api

import (
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var libraryAnalysisSvc *services.LibraryAnalysisService

func ensureLibraryAnalysisSvc() *services.LibraryAnalysisService {
	if libraryAnalysisSvc == nil {
		libraryAnalysisSvc = services.NewLibraryAnalysisService()
	}
	return libraryAnalysisSvc
}

// analyzeLibraryStructure analyzes a library's folder structure and returns a plan.
// GET /api/v1/libraries/:id/analyze-structure
func analyzeLibraryStructure(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid library id", err)
		return
	}

	plan, err := ensureLibraryAnalysisSvc().AnalyzeLibraryStructure(id)
	if err != nil {
		respondInternalError(c, "failed to analyze library structure", err)
		return
	}

	respondSuccess(c, plan, "")
}

// applyLibraryStructurePlan applies a (possibly filtered) structure plan to the database.
// POST /api/v1/libraries/:id/apply-structure
func applyLibraryStructurePlan(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid library id", err)
		return
	}

	var req services.ApplyPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid request body", err)
		return
	}
	req.LibraryID = id

	result, err := ensureLibraryAnalysisSvc().ApplyPlan(&req)
	if err != nil {
		respondInternalError(c, "failed to apply structure plan", err)
		return
	}

	respondSuccess(c, result, "")
}
