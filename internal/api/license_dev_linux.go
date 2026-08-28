//go:build linux

package api

import (
	"fmt"
	"os"
	"syscall"
)

// dataDirDevID 数据目录的 (dev, inode) 特征 —— 挂载卷在宿主机上稳定,
// Docker 容器重建后不变(与随机的容器 hostname 不同)
func dataDirDevID(dataDir string) (string, bool) {
	st, err := os.Stat(dataDir)
	if err != nil {
		return "", false
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d-%d", sys.Dev, sys.Ino), true
	}
	return "", false
}
