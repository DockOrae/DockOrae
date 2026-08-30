// Package public 承载前端构建产物(go:embed),供 cmd/docker-manager 的静态服务引用。
// 前端源码已独立到 DockOrae/DockOrae-Frontend 仓库,构建时下载其
// rolling release 的 dist 资产解压到本目录(见 Makefile web 目标 / Dockerfile Stage 1)。
package public

import "embed"

//go:embed all:dist
var Dist embed.FS
