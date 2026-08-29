//go:build linux

package service

import (
	"fmt"
	"os"
	"syscall"
)

// dataDirDeviceID Linux 设备标识:数据目录的 (dev, inode) 特征。
//
// 挂载卷(如 /data)的 (dev, inode) 在宿主机上稳定 —— 容器重建/升级后不变,
// 这是在线设备绑定的前提(否则每次容器重建 device_id 变化,授权全部失效)。
func dataDirDeviceID(dataDir string) (string, bool) {
	st, err := os.Stat(dataDir)
	if err != nil {
		return "", false
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d-%d@docker-manager", sys.Dev, sys.Ino), true
	}
	return "", false
}
