//go:build !linux

package api

// readDisk Windows / 非 Linux 无 statvfs,返回空
func readDisk() (total, used uint64, ok bool) {
	return 0, 0, false
}
