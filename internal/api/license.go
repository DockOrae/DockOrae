package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/internal/service"
)

func licenseGet(c *gin.Context, d *Deps) error {
	c.JSON(200, service.LicenseInfo(d.St))
	return nil
}

// licenseVerifyNow POST /license/verify 手动触发一次在线验证(吊销即时触达)。
func licenseVerifyNow(c *gin.Context, d *Deps) error {
	c.JSON(200, service.VerifyNow(d.St))
	return nil
}

func licenseActivate(c *gin.Context, d *Deps) error {
	var req struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	info, err := service.LicenseDoActivate(d.St, req.Key)
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

// licenseActivateFile 上传许可文件激活
func licenseActivateFile(c *gin.Context, d *Deps) error {
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
	info, err := service.LicenseDoActivate(d.St, key)
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

func licenseDeactivate(c *gin.Context, d *Deps) error {
	if err := service.LicenseDeactivate(d.St); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}
