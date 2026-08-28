package api

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func licenseGet(c *gin.Context, st *state.AppState) error {
	c.JSON(200, service.LicenseInfo(st))
	return nil
}

func licenseActivate(c *gin.Context, st *state.AppState) error {
	var req struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	info, err := service.LicenseDoActivate(st, req.Key)
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

// licenseActivateFile 上传许可文件激活
func licenseActivateFile(c *gin.Context, st *state.AppState) error {
	file, err := c.FormFile("file")
	if err != nil {
		return service.BadRequest("license.fileRequired")
	}
	f, err := file.Open()
	if err != nil {
		return service.BadRequest("license.fileRequired")
	}
	defer f.Close()
	buf := make([]byte, 64*1024)
	n, _ := f.Read(buf)
	key := strings.TrimSpace(string(buf[:n]))
	info, err := service.LicenseDoActivate(st, key)
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

func licenseDeactivate(c *gin.Context, st *state.AppState) error {
	if err := service.LicenseDeactivate(st); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func licenseDemoKey(c *gin.Context, st *state.AppState) error {
	c.JSON(200, gin.H{"key": service.DemoKey()})
	return nil
}

var _ = os.Getenv
