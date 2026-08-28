package logger

import (
	"io"
	"os"
	"sync"
	"time"
)
// Ring 内存环形日志缓冲(面板日志弹窗数据源)
type Ring struct {
	mu   sync.Mutex
	buf  []string
	next int
	size int
}

func NewRing(size int) *Ring {
	return &Ring{buf: make([]string, 0, size), size: size}
}

func (r *Ring) Write(p []byte) (int, error) {
	line := time.Now().Format("2006-01-02 15:04:05") + " " + string(p)
	r.mu.Lock()
	if len(r.buf) < r.size {
		r.buf = append(r.buf, line)
	} else {
		r.buf[r.next] = line
		r.next = (r.next + 1) % r.size
	}
	r.mu.Unlock()
	return len(p), nil
}

// Lines 返回最近的 n 行(时间序)
func (r *Ring) Lines(n int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || n > len(r.buf) {
		n = len(r.buf)
	}
	out := make([]string, 0, n)
	// 环形缓冲:从 next 位置读回
	start := len(r.buf) - n
	for i := start; i < len(r.buf); i++ {
		idx := (r.next + i) % r.size
		out = append(out, r.buf[idx])
	}
	return out
}

// LogRing 面板日志环形缓冲(日志弹窗数据源;供 cmd/main.go 与 service 引用)
var LogRing = NewRing(2000)

// MultiWriter 同时写 stdout 和环形缓冲
type MultiWriter struct {
	ring *Ring
}

func NewMultiWriter(ring *Ring) io.Writer {
	return &MultiWriter{ring: ring}
}

func (m *MultiWriter) Write(p []byte) (int, error) {
	_, _ = os.Stdout.Write(p)
	_, _ = m.ring.Write(p)
	return len(p), nil
}

