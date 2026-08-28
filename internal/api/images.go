package api

import (
	"bufio"
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func imagesList(c *gin.Context, st *state.AppState) error {
	items, err := service.ImagesList(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func imagesInspect(c *gin.Context, st *state.AppState) error {
	insp, err := service.ImageInspect(st, c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	c.JSON(200, insp)
	return nil
}

// imagesPull 拉取镜像:响应为 application/x-ndjson 流(逐行进度 JSON)
func imagesPull(c *gin.Context, st *state.AppState) error {
	var req model.PullImageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if req.FromImage == "" {
		return service.BadRequest("image.nameEmpty")
	}
	tag := "latest"
	if req.Tag != nil && *req.Tag != "" {
		tag = *req.Tag
	}
	ref := service.ImagePullRef(req.FromImage, tag)

	pullRes, err := service.ImagePull(st, c.Request.Context(), ref)
	if err != nil {
		return err
	}
	defer pullRes.Close()

	c.Header("Content-Type", "application/x-ndjson")
	c.Status(200)
	flusher, _ := c.Writer.(interface{ Flush() })
	w := bufio.NewWriter(c.Writer)
	for msg, err := range pullRes.JSONMessages(c.Request.Context()) {
		if err != nil {
			line, _ := json.Marshal(gin.H{"error": err.Error()})
			_, _ = w.Write(append(line, '\n'))
			_ = w.Flush()
			if flusher != nil {
				flusher.Flush()
			}
			break
		}
		line, _ := json.Marshal(msg)
		_, _ = w.Write(append(line, '\n'))
		_ = w.Flush()
		if flusher != nil {
			flusher.Flush()
		}
	}
	return nil
}

func imagesRemove(c *gin.Context, st *state.AppState) error {
	if err := service.ImageRemove(st, c.Request.Context(), c.Param("id"), parseBool(c.Query("force"), false)); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func imagesPrune(c *gin.Context, st *state.AppState) error {
	report, err := service.ImagesPrune(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}

func imagesTag(c *gin.Context, st *state.AppState) error {
	var req struct {
		Repo string `json:"repo"`
		Tag  string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.ImageTag(st, c.Request.Context(), c.Param("id"), req.Repo, req.Tag); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}
