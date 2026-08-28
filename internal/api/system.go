package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func systemHealth(c *gin.Context, st *state.AppState) error {
	c.JSON(200, gin.H{
		"ok":      true,
		"name":    "docker-manager-go",
		"version": service.AppVersion,
	})
	return nil
}

// systemDefaultAccount 登录页提示"默认账号"的条件
func systemDefaultAccount(c *gin.Context, st *state.AppState) error {
	c.JSON(200, gin.H{"show": service.DefaultAccountShow(st)})
	return nil
}

func systemPublicConfig(c *gin.Context, st *state.AppState) error {
	c.JSON(200, service.PublicConfig(st))
	return nil
}

func systemLogin(c *gin.Context, st *state.AppState) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("login.errPwd")
	}
	resp, err := service.Login(st, req.Username, req.Password, c.ClientIP())
	if err != nil {
		return err
	}
	c.JSON(200, resp)
	return nil
}

func systemLoginTotp(c *gin.Context, st *state.AppState) error {
	var req struct {
		Username string `json:"username"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("login.errTotpCode")
	}
	resp, err := service.LoginTotp(st, req.Username, req.Code, c.ClientIP())
	if err != nil {
		return err
	}
	c.JSON(200, resp)
	return nil
}

func systemMe(c *gin.Context, st *state.AppState) error {
	resp, err := service.Me(st, c.GetString("username"))
	if err != nil {
		return err
	}
	c.JSON(200, resp)
	return nil
}

func systemUpdateProfile(c *gin.Context, st *state.AppState) error {
	var req struct {
		Nickname *string `json:"nickname"`
		Username *string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	resp, err := service.UpdateProfile(st, c.GetString("username"), req.Nickname, req.Username)
	if err != nil {
		return err
	}
	c.JSON(200, resp)
	return nil
}

func systemUploadAvatar(c *gin.Context, st *state.AppState) error {
	var req struct {
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("user.avatarInvalid")
	}
	fileName, err := service.UploadAvatar(st, c.GetString("username"), req.Data)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "avatar": fileName})
	return nil
}

func systemServeAvatar(c *gin.Context, st *state.AppState) error {
	data, err := service.AvatarData(st, c.Param("file"))
	if err != nil {
		return err
	}
	ctype := service.MimeByExt(c.Param("file"))
	c.Header("Content-Type", ctype)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(200, ctype, data)
	return nil
}

func wallpaperSave(c *gin.Context, st *state.AppState) error {
	var req struct {
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("wallpaper.invalid")
	}
	if err := service.WallpaperSave(st, req.Data); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func wallpaperGet(c *gin.Context, st *state.AppState) error {
	path, err := service.WallpaperPath(st)
	if err != nil {
		return err
	}
	c.File(path)
	return nil
}

func systemChangePassword(c *gin.Context, st *state.AppState) error {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.ChangePassword(st, c.GetString("username"), req.OldPassword, req.NewPassword); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemTotpSetup(c *gin.Context, st *state.AppState) error {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	secret, uri, err := service.TotpSetup(st, c.GetString("username"), req.Password)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"secret": secret, "uri": uri})
	return nil
}

func systemTotpEnable(c *gin.Context, st *state.AppState) error {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("totp.getKeyFirst")
	}
	if err := service.TotpEnable(st, c.GetString("username"), req.Code); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemTotpDisable(c *gin.Context, st *state.AppState) error {
	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.TotpDisable(st, c.GetString("username"), req.Password, req.Code); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemInfo(c *gin.Context, st *state.AppState) error {
	info, err := service.DockerInfo(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

var _ = http.StatusOK
