//go:build linux

package service

import "golang.org/x/sys/unix"

// ReadDisk 根文件系统磁盘占用 (total, used) bytes
// 容器内 overlay 的 statfs 反映宿主机根盘(等价旧版 libc::statvfs)
func ReadDisk() (total, used uint64, ok bool) {
	var st unix.Statfs_t
	if err := unix.Statfs("/", &st); err != nil {
		return 0, 0, false
	}
	total = st.Blocks * uint64(st.Bsize)
	avail := st.Bavail * uint64(st.Bsize)
	used = total - avail
	return total, used, true
}
