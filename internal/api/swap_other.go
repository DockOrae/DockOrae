//go:build !linux

package api

// readSwap Windows / 非 Linux 无 /proc/meminfo,返回空
func readSwap() (total, used uint64, ok bool) {
	return 0, 0, false
}
