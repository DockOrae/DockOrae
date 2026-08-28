// Package appstore 应用商店:应用清单定义与 compose 模板渲染(纯逻辑,无外部依赖)。
// 参数 schema 对齐 1Panel 应用商店:type/rule/random/多语言 label/select options。
// 新增应用:在 apps 切片追加定义(compose 模板用 Go text/template,参数以 {{.key}} 引用)。
package appstore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
)

// Option select 选项(1Panel values)
type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Param 安装参数 schema(对齐 1Panel formFields)
type Param struct {
	Key      string   `json:"key"`
	LabelZh  string   `json:"label_zh"`
	LabelEn  string   `json:"label_en"`
	Type     string   `json:"type"` // text | password | number | select | checkbox | textarea
	Default  string   `json:"default"`
	Required bool     `json:"required"`
	Random   bool     `json:"random"` // 安装时为空自动生成随机值(密码类)
	Rule     string   `json:"rule"`   // paramPort | paramCommon | paramRequired | ""
	Options  []Option `json:"options,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

// App 应用定义
type App struct {
	Key         string                  `json:"key"`
	Name        string                  `json:"name"`
	Icon        string                  `json:"icon"`
	Category    string                  `json:"category"`
	Description string                  `json:"description"`
	Ports       []string                `json:"ports"`
	Versions    []string                `json:"versions,omitempty"` // 可选版本列表(模板 image 用 {{.version}})
	Params      []Param                 `json:"params"`
	dir         string                  // 应用数据目录
	versions    map[string]*versionData // 各版本数据(参数 + compose 模板)
}

// ValidateParams 校验参数值:required/rule(paramPort/paramCommon)
func ValidateParams(params []Param, values map[string]string) error {
	for _, p := range params {
		v := strings.TrimSpace(values[p.Key])
		if v == "" && p.Default != "" {
			v = p.Default
		}
		if p.Required && v == "" && !p.Random {
			return &ParamError{Key: p.Key, Msg: "required"}
		}
		if v == "" {
			continue
		}
		switch p.Rule {
		case "paramPort":
			n, ok := parsePort(v)
			if !ok {
				return &ParamError{Key: p.Key, Msg: "port"}
			}
			_ = n
		case "paramCommon":
			if !isCommonName(v) {
				return &ParamError{Key: p.Key, Msg: "common"}
			}
		}
	}
	return nil
}

// ParamError 参数校验错误(key 供前端定位)
type ParamError struct {
	Key string
	Msg string
}

func (e *ParamError) Error() string { return e.Msg }

func parsePort(v string) (int, bool) {
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, n >= 1 && n <= 65535
}

func isCommonName(v string) bool {
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return v != ""
}

// GlobalParams 通用安装选项(所有应用表单都显示,1Panel 风格;渲染后统一注入 compose)
var GlobalParams = []Param{
	{Key: "container_name", LabelZh: "容器名称", LabelEn: "Container Name", Type: "text", Hint: "containerName"},
	{Key: "restart", LabelZh: "重启规则", LabelEn: "Restart Policy", Type: "select", Default: "always",
		Options: []Option{{Label: "always", Value: "always"}, {Label: "unless-stopped", Value: "unless-stopped"}, {Label: "on-failure", Value: "on-failure"}, {Label: "never", Value: "never"}}},
	{Key: "cpu_limit", LabelZh: "CPU 限制(核心数)", LabelEn: "CPU Limit (cores)", Type: "number", Default: "0", Hint: "cpuLimit"},
	{Key: "mem_limit", LabelZh: "内存限制(MB)", LabelEn: "Memory Limit (MB)", Type: "number", Default: "0", Hint: "memLimit"},
	{Key: "pull_first", LabelZh: "启动前拉取镜像", LabelEn: "Pull image before start", Type: "checkbox", Default: "true"},
}

// Render 渲染指定版本的 compose:${PANEL_XXX} 环境变量替换 + 面板内置变量 + 通用选项注入
// values:表单值(envKey → 值);default/random 自动补齐;未知变量置空
func Render(a *App, version string, values map[string]string) (string, error) {
	vd := a.versions[version]
	if vd == nil {
		return "", fmt.Errorf("version %s not found", version)
	}
	env := map[string]string{}
	for _, p := range vd.Params {
		v := strings.TrimSpace(values[p.Key])
		if v == "" && p.Default != "" {
			v = p.Default
		}
		if v == "" && p.Random {
			v = randomHex(16)
		}
		env[p.Key] = v
	}
	for k, v := range values {
		if k == "version" || k == "container_name" {
			continue
		}
		if _, ok := env[k]; !ok {
			env[k] = v
		}
	}
	// 面板内置注入(1Panel 同款)
	cn := strings.TrimSpace(values["container_name"])
	if cn == "" {
		cn = a.Key
	}
	env["CONTAINER_NAME"] = cn
	env["HOST_IP"] = hostIP()
	tz := strings.TrimSpace(values["time_zone"])
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	env["TIME_ZONE"] = tz
	env["TZ"] = tz

	rendered := os.Expand(vd.Tpl, func(k string) string { return env[k] })
	rendered = strings.ReplaceAll(rendered, "\r\n", "\n") // 仓库文件 CRLF → LF
	return postProcess(rendered, values), nil
}

// postProcess 注入通用选项:重启规则 / 容器名称 / CPU·内存限制(deploy 段)
func postProcess(yaml string, vals map[string]string) string {
	// 1. 重启规则(替换模板中的 restart: always)
	if r := vals["restart"]; r != "" {
		yaml = strings.ReplaceAll(yaml, "restart: always", "restart: "+r)
	}
	// 2. 容器名称(用户填写时替换模板中所有 container_name 值)
	if cn := vals["container_name"]; cn != "" {
		yaml = regexp.MustCompile(`(container_name:\s*)[^\s#]+`).ReplaceAllString(yaml, "container_name: "+cn)
	}
	// 3. CPU / 内存限制:在每个 service 定义头插入 deploy 段(值为 0 或空不注入)
	cpu, mem := vals["cpu_limit"], vals["mem_limit"]
	if (cpu == "" || cpu == "0") && (mem == "" || mem == "0") {
		return yaml
	}
	var limit strings.Builder
	if cpu != "" && cpu != "0" {
		limit.WriteString("          cpus: \"" + cpu + "\"\n")
	}
	if mem != "" && mem != "0" {
		limit.WriteString("          memory: " + mem + "M\n")
	}
	if limit.Len() == 0 {
		return yaml
	}
	deployBlock := "    deploy:\n      resources:\n        limits:\n" + limit.String()

	lines := strings.Split(yaml, "\n")
	var out strings.Builder
	inServices := false
	for _, line := range lines {
		out.WriteString(line)
		out.WriteString("\n")
		trimmed := strings.TrimSpace(line)
		if trimmed == "services:" {
			inServices = true
			continue
		}
		// 离开 services 段(顶层 volumes/networks 等声明,0 缩进)
		if inServices && !strings.HasPrefix(line, "  ") && strings.HasSuffix(line, ":") {
			switch strings.TrimSuffix(trimmed, ":") {
			case "volumes", "networks", "configs", "secrets":
				inServices = false
				continue
			}
		}
		// services 段内缩进 2 的 service 名行
		if inServices && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") &&
			strings.HasSuffix(line, ":") && !strings.HasPrefix(trimmed, "-") {
			out.WriteString(deployBlock)
		}
	}
	return strings.TrimRight(out.String(), "\n")
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// VersionData 指定版本的数据(参数 + compose 模板 + 附加文件目录)
func (a *App) VersionData(version string) *versionData {
	return a.versions[version]
}

// hostIP 宿主非回环 IPv4(供 compose 的 ${HOST_IP})
func hostIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() {
			if ip4 := ipn.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return ""
}
