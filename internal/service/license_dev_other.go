//go:build !linux && !windows

package service

import "os"

// dataDirDeviceID 其他平台设备标识:回退主机名。
func dataDirDeviceID(dataDir string) (string, bool) {
	if host, err := os.Hostname(); err == nil && host != "" {
		return host + "@docker-manager", true
	}
	return "", false
}
