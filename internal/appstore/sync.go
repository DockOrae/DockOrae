package appstore

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AppStoreRepo 应用商店数据仓库(MinimaxFlora/docker-manager-apps,1Panel 同款结构)。
// 可用环境变量覆盖:DM_APPSTORE_REPO=user/repo
var AppStoreRepo = func() string {
	if r := os.Getenv("DM_APPSTORE_REPO"); r != "" {
		return r
	}
	return "MinimaxFlora/docker-manager-apps"
}()

// SyncURL 仓库 tarball 地址(可用 DM_APPSTORE_URL 覆盖,如内网镜像)
func SyncURL() string {
	if u := os.Getenv("DM_APPSTORE_URL"); u != "" {
		return u
	}
	return fmt.Sprintf("https://codeload.github.com/%s/tar.gz/refs/heads/master", AppStoreRepo)
}

// Sync 下载仓库 tarball 并解压到 dir(原子替换)。dir 为仓库根(内含 apps/ 子目录)。
func Sync(dir string) error {
	tmp, err := os.CreateTemp("", "appstore-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(SyncURL())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s %d", SyncURL(), resp.StatusCode)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	tmp.Close()

	stage := dir + ".tmp"
	os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return err
	}
	if err := extractTarGz(tmp.Name(), stage); err != nil {
		os.RemoveAll(stage)
		return err
	}
	if _, err := os.Stat(filepath.Join(stage, "apps")); err != nil {
		os.RemoveAll(stage)
		return fmt.Errorf("bad appstore archive: apps/ missing")
	}

	backup := dir + ".old"
	os.RemoveAll(backup)
	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, backup); err != nil {
			os.RemoveAll(stage)
			return err
		}
	}
	if err := os.Rename(stage, dir); err != nil {
		_ = os.Rename(backup, dir)
		return err
	}
	os.RemoveAll(backup)
	return nil
}

// extractTarGz 解压 tar.gz,去除仓库根目录段(docker-manager-apps-master/)
func extractTarGz(tarGz, dest string) error {
	f, err := os.Open(tarGz)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		parts := strings.SplitN(hdr.Name, "/", 2)
		if len(parts) < 2 {
			continue // 仓库根目录
		}
		name := parts[1]
		if name == "" || strings.Contains(name, "..") {
			continue // 防路径穿越
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			// 符号链接等:跳过
		}
	}
	return nil
}
