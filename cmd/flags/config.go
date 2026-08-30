// Package flags 承载命令行 flag 解析结果(参考 OpenList cmd/flags 结构)。
// main 在启动时解析 flag 并写入这些变量,随后覆盖环境变量交给 config.Load。
package flags

var (
	// DataDir 数据目录(-data;默认取 $DATA_DIR 或 /data)
	DataDir string
	// Port 监听端口(-port;默认取 $PORT 或 8080)
	Port int
)
