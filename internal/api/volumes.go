package api

import (
	"github.com/gin-gonic/gin"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func volumesList(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.VolumeList(c.Request.Context(), client.VolumeListOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Items)
	return nil
}

func volumesInspect(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.VolumeInspect(c.Request.Context(), c.Param("name"), client.VolumeInspectOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.Data(200, "application/json", res.Raw)
	return nil
}

type createVolReq struct {
	Name   string  `json:"name"`
	Driver *string `json:"driver"`
}

func volumesCreate(c *gin.Context, st *state.AppState) error {
	var req createVolReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	if req.Name == "" {
		return BadRequest("volume.nameEmpty")
	}
	driver := "local"
	if req.Driver != nil && *req.Driver != "" {
		driver = *req.Driver
	}
	res, err := st.Docker.VolumeCreate(c.Request.Context(), client.VolumeCreateOptions{
		Name:   req.Name,
		Driver: driver,
	})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Volume)
	return nil
}

func volumesRemove(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.VolumeRemove(c.Request.Context(), c.Param("name"), client.VolumeRemoveOptions{
		Force: parseBool(c.Query("force"), false),
	})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func volumesPrune(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.VolumePrune(c.Request.Context(), client.VolumePruneOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Report)
	return nil
}
