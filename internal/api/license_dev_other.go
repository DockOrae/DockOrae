//go:build !linux

package api

// dataDirDevID 非 Linux 平台不提供 (dev, inode) 特征
func dataDirDevID(dataDir string) (string, bool) {
	return "", false
}
