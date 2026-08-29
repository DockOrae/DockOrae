package api

import (
	"github.com/gin-gonic/gin"

	"github.com/DockerManger/Docker_Manager_Go/internal/service"
)

func systemHealth(c *gin.Context, d *Deps) error {
	c.JSON(200, gin.H{
		"ok":      true,
		"name":    "docker-manager-go",
		"version": service.DisplayVersion(),
	})
	return nil
}

// systemDefaultAccount 登录页提示"默认账号"的条件
func systemDefaultAccount(c *gin.Context, d *Deps) error {
	c.JSON(200, gin.H{"show": service.DefaultAccountShow(d.St)})
	return nil
}

func systemPublicConfig(c *gin.Context, d *Deps) error {
	c.JSON(200, service.PublicConfig(d.St))
	return nil
}

func systemLogin(c *gin.Context, d *Deps) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("login.errPwd")
	}
	// SEC-002:失败限速(IP+用户名,5 次失败锁 15 分钟)
	if err := loginThrottled(c.ClientIP(), req.Username); err != nil {
		return err
	}
	resp, err := service.Login(d.St, req.Username, req.Password, c.ClientIP())
	if err != nil {
		// 仅凭据错误(401)计入失败次数;服务异常不计数
		if ae, ok := err.(*service.ApiError); ok && ae.Status == 401 {
			loginGuardInst.fail(loginKey(c.ClientIP(), req.Username), loginMaxFails, loginLockTime)
		}
		return err
	}
	loginGuardInst.success(loginKey(c.ClientIP(), req.Username))
	c.JSON(200, resp)
	return nil
}

func systemLoginTotp(c *gin.Context, d *Deps) error {
	var req struct {
		Username string `json:"username"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("login.errTotpCode")
	}
	// SEC-002:TOTP 限速(3 次失败锁 5 分钟,防 6 位码穷举)
	if err := totpThrottled(c.ClientIP(), req.Username); err != nil {
		return err
	}
	resp, err := service.LoginTotp(d.St, req.Username, req.Code, c.ClientIP())
	if err != nil {
		if ae, ok := err.(*service.ApiError); ok && ae.Status == 401 {
			loginGuardInst.fail(loginKey(c.ClientIP(), req.Username), totpMaxFails, totpLockTime)
		}
		return err
	}
	loginGuardInst.success(loginKey(c.ClientIP(), req.Username))
	c.JSON(200, resp)
	return nil
}

func systemMe(c *gin.Context, d *Deps) error {
	resp, err := service.Me(d.St, c.GetString("username"))
	if err != nil {
		return err
	}
	c.JSON(200, resp)
	return nil
}

func systemUpdateProfile(c *gin.Context, d *Deps) error {
	var req struct {
		Nickname *string `json:"nickname"`
		Username *string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	resp, err := service.UpdateProfile(d.St, c.GetString("username"), req.Nickname, req.Username)
	if err != nil {
		return err
	}
	c.JSON(200, resp)
	return nil
}

func systemUploadAvatar(c *gin.Context, d *Deps) error {
	var req struct {
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("user.avatarInvalid")
	}
	fileName, err := service.UploadAvatar(d.St, c.GetString("username"), req.Data)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "avatar": fileName})
	return nil
}

func systemServeAvatar(c *gin.Context, d *Deps) error {
	data, err := service.AvatarData(d.St, c.Param("file"))
	if err != nil {
		return err
	}
	ctype := service.MimeByExt(c.Param("file"))
	c.Header("Content-Type", ctype)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(200, ctype, data)
	return nil
}

func wallpaperSave(c *gin.Context, d *Deps) error {
	var req struct {
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("wallpaper.invalid")
	}
	if err := service.WallpaperSave(d.St, req.Data); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func wallpaperGet(c *gin.Context, d *Deps) error {
	path, err := service.WallpaperPath(d.St)
	if err != nil {
		return err
	}
	c.File(path)
	return nil
}

func systemChangePassword(c *gin.Context, d *Deps) error {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.ChangePassword(d.St, c.GetString("username"), req.OldPassword, req.NewPassword); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemTotpSetup(c *gin.Context, d *Deps) error {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	secret, uri, err := service.TotpSetup(d.St, c.GetString("username"), req.Password)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"secret": secret, "uri": uri})
	return nil
}

func systemTotpEnable(c *gin.Context, d *Deps) error {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("totp.getKeyFirst")
	}
	if err := service.TotpEnable(d.St, c.GetString("username"), req.Code); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemTotpDisable(c *gin.Context, d *Deps) error {
	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.TotpDisable(d.St, c.GetString("username"), req.Password, req.Code); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemInfo(c *gin.Context, d *Deps) error {
	info, err := service.DockerInfo(d.St, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}
