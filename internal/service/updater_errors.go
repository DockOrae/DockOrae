package service

import "errors"

// 更新错误分类:调用方/测试可用 errors.Is 判断错误类型,避免到处比较字符串。
// 用户可见的详细描述由 UpdateStatus.Error 承载,分类仅用于逻辑判断。
var (
	// ErrCheckFailed 更新检查失败(GitHub API 超时/限流/网络断开等)
	ErrCheckFailed = errors.New("update check failed")
	// ErrReleaseNotFound Release 不存在或 payload 无效
	ErrReleaseNotFound = errors.New("release not found")
	// ErrAssetNotFound 当前平台没有对应的发布资产
	ErrAssetNotFound = errors.New("asset not found")
	// ErrChecksumMismatch SHA256 校验不一致(下载损坏或包被篡改)
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrDownloadFailed 下载更新包失败
	ErrDownloadFailed = errors.New("download failed")
	// ErrInstallFailed 安装/替换失败
	ErrInstallFailed = errors.New("install failed")
	// ErrHealthCheckFailed 更新后健康检查失败
	ErrHealthCheckFailed = errors.New("health check failed")
	// ErrVersionMismatch 更新后实际版本与目标版本不一致
	ErrVersionMismatch = errors.New("version mismatch")
	// ErrDockerImageUnavailable Docker 镜像尚未发布到镜像仓库
	ErrDockerImageUnavailable = errors.New("docker image unavailable")
)
