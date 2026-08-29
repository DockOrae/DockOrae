package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
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

func licenseDemoKey(c *gin.Context, d *Deps) error {
	// SEC-001:demo key 仅限开发构建(版本未注入 → unknown,即本地 go run / 未打 tag 构建)。
	// 正式 release 由 CI 经 ldflags 注入版本号,此接口返回 403——
	// 杜绝"登录 → /license/demo → 永久 Pro key"的生产环境授权绕过。
	if service.DisplayVersion() == "unknown" {
		c.JSON(200, gin.H{"key": service.DemoKey()})
		return nil
	}
	return service.NewApiError(403, "license.demoUnavailable")
}
