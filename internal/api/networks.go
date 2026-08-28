package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func networksList(c *gin.Context, st *state.AppState) error {
	items, err := service.NetworksList(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func networksInspect(c *gin.Context, st *state.AppState) error {
	raw, err := service.NetworkRaw(st, c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	c.Data(200, "application/json", raw)
	return nil
}

func networksCreate(c *gin.Context, st *state.AppState) error {
	var req model.CreateNetworkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	id, err := service.NetworkCreate(st, c.Request.Context(), req)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"id": id})
	return nil
}

func networksRemove(c *gin.Context, st *state.AppState) error {
	if err := service.NetworkRemove(st, c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func networksPrune(c *gin.Context, st *state.AppState) error {
	report, err := service.NetworksPrune(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}
