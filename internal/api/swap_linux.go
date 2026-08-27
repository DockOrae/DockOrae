//go:build linux

package api

import (
	"fmt"
	"os"
	"strings"
)

// readSwap 交换空间 (total, used) bytes,来源 /proc/meminfo
func readSwap() (total, used uint64, ok bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	var totalKb, freeKb uint64
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "SwapTotal:"):
			fmt.Sscanf(line, "SwapTotal: %d", &totalKb)
		case strings.HasPrefix(line, "SwapFree:"):
			fmt.Sscanf(line, "SwapFree: %d", &freeKb)
		}
	}
	if totalKb == 0 {
		// 未配置 swap:与 3x-ui 一致按 0 处理(不置 ok=false,让前端显示 0%)
		return 0, 0, true
	}
	return totalKb * 1024, (totalKb - freeKb) * 1024, true
}
