package api

import (
	"bufio"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func imagesList(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.ImageList(c.Request.Context(), client.ImageListOptions{})
	if err != nil {
		return dockerError(err)
	}
	// 精简:列表只返回前端用到的字段(Id/RepoTags/Size/Created),全量含 Labels/RepoDigests 等大字段
	items := make([]imageListItem, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, imageListItem{ID: it.ID, RepoTags: it.RepoTags, Size: it.Size, Created: it.Created})
	}
	c.JSON(200, items)
	return nil
}

type imageListItem struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Size     int64    `json:"Size"`
	Created  int64    `json:"Created"`
}

func imagesInspect(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.ImageInspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res)
	return nil
}

type pullReq struct {
	FromImage string  `json:"from_image"`
	Tag       *string `json:"tag"`
}

// imagesPull 拉取镜像:响应为 application/x-ndjson 流(逐行进度 JSON)
func imagesPull(c *gin.Context, st *state.AppState) error {
	var req pullReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	if req.FromImage == "" {
		return BadRequest("image.nameEmpty")
	}
	ref := req.FromImage
	tag := "latest"
	if req.Tag != nil && *req.Tag != "" {
		tag = *req.Tag
	}
	// 镜像名已带 tag 时不再拼接(等价旧版 fromImage+tag 分离传参)
	lastSlash := strings.LastIndex(ref, "/")
	namePart := ref
	if lastSlash >= 0 {
		namePart = ref[lastSlash+1:]
	}
	if !strings.Contains(namePart, ":") {
		ref = ref + ":" + tag
	}

	pullRes, err := st.Docker.ImagePull(c.Request.Context(), ref, client.ImagePullOptions{})
	if err != nil {
		return dockerError(err)
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
	res, err := st.Docker.ImageRemove(c.Request.Context(), c.Param("id"), client.ImageRemoveOptions{
		Force:         parseBool(c.Query("force"), false),
		PruneChildren: false,
	})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Items)
	return nil
}

type tagReq struct {
	Repo string `json:"repo"`
	Tag  string `json:"tag"`
}

func imagesTag(c *gin.Context, st *state.AppState) error {
	var req tagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	_, err := st.Docker.ImageTag(c.Request.Context(), client.ImageTagOptions{
		Source: c.Param("id"),
		Target: req.Repo + ":" + req.Tag,
	})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func imagesPrune(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.ImagePrune(c.Request.Context(), client.ImagePruneOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Report)
	return nil
}
