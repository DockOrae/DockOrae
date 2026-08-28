//go:build !linux

package service

// ReadSwap Windows / 非 Linux 无 /proc/meminfo,返回空
func ReadSwap() (total, used uint64, ok bool) {
	return 0, 0, false
}
