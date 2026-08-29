package api

import (
	"github.com/gin-gonic/gin"

	"github.com/DockerManger/Docker_Manager_Go/internal/model"
	"github.com/DockerManger/Docker_Manager_Go/internal/service"
)

func networksList(c *gin.Context, d *Deps) error {
	items, err := d.Networks.List(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func networksInspect(c *gin.Context, d *Deps) error {
	raw, err := d.Networks.Raw(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	c.Data(200, "application/json", raw)
	return nil
}

func networksCreate(c *gin.Context, d *Deps) error {
	var req model.CreateNetworkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	id, err := d.Networks.Create(c.Request.Context(), req)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"id": id})
	return nil
}

func networksRemove(c *gin.Context, d *Deps) error {
	if err := d.Networks.Remove(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func networksPrune(c *gin.Context, d *Deps) error {
	report, err := d.Networks.Prune(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}
