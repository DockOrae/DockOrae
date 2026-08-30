package api

import (
	"bufio"
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/service"
)

func imagesList(c *gin.Context, d *Deps) error {
	items, err := d.Images.List(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func imagesInspect(c *gin.Context, d *Deps) error {
	insp, err := d.Images.Inspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	c.JSON(200, insp)
	return nil
}

// imagesPull 拉取镜像:响应为 application/x-ndjson 流(逐行进度 JSON)
func imagesPull(c *gin.Context, d *Deps) error {
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

	pullRes, err := d.Images.Pull(c.Request.Context(), ref)
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

func imagesRemove(c *gin.Context, d *Deps) error {
	if err := d.Images.Remove(c.Request.Context(), c.Param("id"), parseBool(c.Query("force"), false)); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func imagesPrune(c *gin.Context, d *Deps) error {
	report, err := d.Images.Prune(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}

func imagesTag(c *gin.Context, d *Deps) error {
	var req struct {
		Repo string `json:"repo"`
		Tag  string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.Images.Tag(c.Request.Context(), c.Param("id"), req.Repo, req.Tag); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}
