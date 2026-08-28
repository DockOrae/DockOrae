package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func volumesList(c *gin.Context, st *state.AppState) error {
	items, err := service.VolumesList(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

// volumesDrivers 可用卷驱动列表(local + 已启用插件)
func volumesDrivers(c *gin.Context, st *state.AppState) error {
	drivers, err := service.VolumeDrivers(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"drivers": drivers})
	return nil
}

func volumesInspect(c *gin.Context, st *state.AppState) error {
	raw, err := service.VolumeRaw(st, c.Request.Context(), c.Param("name"))
	if err != nil {
		return err
	}
	c.Data(200, "application/json", raw)
	return nil
}

func volumesCreate(c *gin.Context, st *state.AppState) error {
	var req model.CreateVolumeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	vol, err := service.VolumeCreate(st, c.Request.Context(), req)
	if err != nil {
		return err
	}
	c.JSON(200, vol)
	return nil
}

func volumesRemove(c *gin.Context, st *state.AppState) error {
	if err := service.VolumeRemove(st, c.Request.Context(), c.Param("name"), parseBool(c.Query("force"), false)); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func volumesPrune(c *gin.Context, st *state.AppState) error {
	report, err := service.VolumesPrune(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}
