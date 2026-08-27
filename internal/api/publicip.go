package api

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ---------------- 公网 IP(仿 3x-ui resolvePublicIPs:多服务列表 + 缓存) ----------------
// 3x-ui 在每次状态接口返回 publicIP(首次检测后进程内缓存);这里并入 /system/monitor
// 轮询,5 分钟缓存刷新,失败 30 秒后重试,完全容错。

var publicIPCtl = struct {
	sync.Mutex
	v4, v6 string
	at     time.Time
}{}

// ipv4Services / ipv6Services 公网 IP 查询服务(仿 3x-ui publicIPv4Services,依次尝试)
var ipv4Services = []string{
	"https://api4.ipify.org",
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
}
var ipv6Services = []string{
	"https://api6.ipify.org",
	"https://ipv6.icanhazip.com",
}

// publicIPs 返回 (ipv4, ipv6);空串 = 获取失败(前端显示 N/A)
func publicIPs() (string, string) {
	publicIPCtl.Lock()
	defer publicIPCtl.Unlock()

	if !publicIPCtl.at.IsZero() && time.Since(publicIPCtl.at) < 5*time.Minute {
		return publicIPCtl.v4, publicIPCtl.v6
	}
	client := &http.Client{Timeout: 3 * time.Second}
	v4, v6 := "", ""
	for _, svc := range ipv4Services {
		if ip := fetchPublicIP(client, svc); ip != "" {
			v4 = ip
			break
		}
	}
	for _, svc := range ipv6Services {
		if ip := fetchPublicIP(client, svc); ip != "" {
			v6 = ip
			break
		}
	}
	publicIPCtl.v4, publicIPCtl.v6 = v4, v6
	if v4 == "" && v6 == "" {
		// 全部失败:30 秒后重试
		publicIPCtl.at = time.Now().Add(-4*time.Minute + 30*time.Second)
	} else {
		publicIPCtl.at = time.Now()
	}
	return v4, v6
}

// systemPublicIP 独立接口(兼容旧调用)
func systemPublicIP(c *gin.Context, st *state.AppState) error {
	v4, v6 := publicIPs()
	c.JSON(200, gin.H{"ipv4": v4, "ipv6": v6})
	return nil
}

func fetchPublicIP(client *http.Client, url string) string {
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	// 与 3x-ui 一致:403/451 视为区域受限,不再尝试
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnavailableForLegalReasons {
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
