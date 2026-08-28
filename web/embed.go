// Package web 承载前端构建产物(go:embed),供 cmd/docker-manager 的静态服务引用。
// 注意:go:embed 不允许跨目录(..)引用,故 embed 放在 web 包内;
// 构建前需先执行 `cd web && npm run build` 生成 dist。
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
