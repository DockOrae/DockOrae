//go:build windows

package service

import "os"

// dataDirDeviceID Windows 设备标识:环境变量 COMPUTERNAME(稳定,桌面/服务器场景不变)。
func dataDirDeviceID(dataDir string) (string, bool) {
	if name := os.Getenv("COMPUTERNAME"); name != "" {
		return name + "@docker-manager", true
	}
	return "", false
}
