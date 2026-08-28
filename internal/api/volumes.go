package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
)

func volumesList(c *gin.Context, d *Deps) error {
	items, err := d.Volumes.List(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func volumesInspect(c *gin.Context, d *Deps) error {
	raw, err := d.Volumes.Raw(c.Request.Context(), c.Param("name"))
	if err != nil {
		return err
	}
	c.Data(200, "application/json", raw)
	return nil
}

func volumesCreate(c *gin.Context, d *Deps) error {
	var req model.CreateVolumeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	vol, err := d.Volumes.Create(c.Request.Context(), req)
	if err != nil {
		return err
	}
	c.JSON(200, vol)
	return nil
}

func volumesRemove(c *gin.Context, d *Deps) error {
	if err := d.Volumes.Remove(c.Request.Context(), c.Param("name"), parseBool(c.Query("force"), false)); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func volumesPrune(c *gin.Context, d *Deps) error {
	report, err := d.Volumes.Prune(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}
