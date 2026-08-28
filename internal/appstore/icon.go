package appstore

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AppIconRepo 应用图标仓库(与 AppStoreRepo 同一仓库,apps/<key>/logo.png)。
// 可用环境变量 DM_APPSTORE_REPO=user/repo 覆盖。
var AppIconRepo = AppStoreRepo

var iconCache = struct {
	sync.Mutex
	m map[string][]byte
}{m: map[string][]byte{}}

// FetchIcon 获取应用图标:本地同步目录 → 内存缓存 → 磁盘缓存 → 远程(jsDelivr → GitHub raw)。
// appstoreDir 为仓库根(内含 apps/),可为空跳过本地。
func FetchIcon(key string, appstoreDir, cacheDir string) ([]byte, error) {
	if !ValidKey(key) {
		return nil, fmt.Errorf("invalid app key")
	}
	// 本地同步目录优先(仓库已含 logo.png)
	if appstoreDir != "" {
		if b, err := os.ReadFile(filepath.Join(appstoreDir, "apps", key, "logo.png")); err == nil && len(b) > 0 {
			cachePut(key, b)
			return b, nil
		}
	}
	iconCache.Lock()
	if b, ok := iconCache.m[key]; ok {
		iconCache.Unlock()
		return b, nil
	}
	iconCache.Unlock()

	if cacheDir != "" {
		disk := filepath.Join(cacheDir, "app-icons", key+".png")
		if b, err := os.ReadFile(disk); err == nil && len(b) > 0 {
			cachePut(key, b)
			return b, nil
		}
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var lastErr error
	for _, u := range iconURLs(key) {
		resp, err := client.Get(u)
		if err != nil {
			lastErr = err
			continue
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			if err != nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("status %d", resp.StatusCode)
			}
			continue
		}
		if cacheDir != "" {
			disk := filepath.Join(cacheDir, "app-icons", key+".png")
			_ = os.MkdirAll(filepath.Dir(disk), 0o755)
			_ = os.WriteFile(disk, b, 0o644)
		}
		cachePut(key, b)
		return b, nil
	}
	return nil, lastErr
}

func iconURLs(key string) []string {
	p := fmt.Sprintf("apps/%s/logo.png", key)
	return []string{
		fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s@master/%s", AppIconRepo, p),
		fmt.Sprintf("https://raw.githubusercontent.com/%s/master/%s", AppIconRepo, p),
	}
}

func cachePut(key string, b []byte) {
	iconCache.Lock()
	iconCache.m[key] = b
	iconCache.Unlock()
}
