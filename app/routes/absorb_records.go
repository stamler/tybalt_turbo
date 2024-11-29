package routes

import (
	"fmt"
	"net/http"

	"tybalt/utilities"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// AbsorbRequest defines the structure for the absorb request body
type AbsorbRequest struct {
	IdsToAbsorb []string `json:"ids_to_absorb"`
}

func CreateAbsorbRecordsHandler(app core.App, collectionName string) echo.HandlerFunc {
	// This handler absorbs multiple records into one target record for a given collection.
	// It performs the following actions:
	// 1. Validates the request body contains a list of IDs to absorb
	// 2. Calls the AbsorbRecords function to perform the absorption
	return func(c echo.Context) error {
		targetId := c.PathParam("id")
		var request AbsorbRequest
		if err := c.Bind(&request); err != nil {
			return apis.NewBadRequestError("Invalid request body", nil)
		}

		if len(request.IdsToAbsorb) == 0 {
			return apis.NewBadRequestError("No IDs provided to absorb", nil)
		}

		// Check if trying to absorb a record into itself
		for _, id := range request.IdsToAbsorb {
			if id == targetId {
				return apis.NewBadRequestError("Cannot absorb a record into itself", nil)
			}
		}

		// Check if user has the absorb claim
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		hasAbsorbClaim, err := utilities.HasClaim(app.Dao(), authRecord.Id, "absorb")
		if err != nil {
			return apis.NewBadRequestError("Failed to check user claims", err)
		}
		if !hasAbsorbClaim {
			return apis.NewForbiddenError("User does not have permission to absorb records", nil)
		}

		// Check if the collection is supported
		_, err = GetConfigsAndTable(collectionName)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to absorb records", err)
		}

		// First verify that the target record exists
		_, err = app.Dao().FindRecordById(collectionName, targetId)
		if err != nil {
			return apis.NewNotFoundError("Failed to find target record", err)
		}

		// Then verify that all records to absorb exist
		for _, id := range request.IdsToAbsorb {
			_, err := app.Dao().FindRecordById(collectionName, id)
			if err != nil {
				return apis.NewNotFoundError("Failed to find record to absorb", err)
			}
		}

		err = AbsorbRecords(app, collectionName, targetId, request.IdsToAbsorb)
		if err != nil {
			// Customize the error response as needed
			return apis.NewApiError(http.StatusInternalServerError, "Failed to absorb records", err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Successfully absorbed %d records into %s", len(request.IdsToAbsorb), targetId),
		})
	}
}
