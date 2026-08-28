//go:build !linux

package service

// ReadDisk Windows / 非 Linux 无 statvfs,返回空
func ReadDisk() (total, used uint64, ok bool) {
	return 0, 0, false
}
