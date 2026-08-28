// Package appstore 应用商店:应用清单定义与 compose 模板渲染(纯逻辑,无外部依赖)。
// 新增应用:在 apps 切片追加定义(compose 模板用 Go text/template,参数以 {{.key}} 引用)。
package appstore

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"text/template"
)

// Param 安装参数 schema
type Param struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // text | password | number | select
	Default  string   `json:"default"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

// App 应用定义
type App struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Ports       []string `json:"ports"`
	Params      []Param  `json:"params"`
	tpl         string   `json:"-"`
}

// Categories 应用分类
func Categories() []string {
	return []string{"database", "service", "tools"}
}

// All 全部应用
func All() []*App { return apps }

// Get 按 key 查找
func Get(key string) *App {
	for _, a := range apps {
		if a.Key == key {
			return a
		}
	}
	return nil
}

// Render 渲染 compose 模板:password 类型为空时自动生成随机 16 位
func Render(app *App, values map[string]string) (string, error) {
	data := map[string]string{"key": app.Key}
	for _, p := range app.Params {
		v := strings.TrimSpace(values[p.Key])
		if v == "" {
			v = p.Default
		}
		if p.Type == "password" && v == "" {
			v = randomHex(16)
		}
		data[p.Key] = v
	}
	t, err := template.New(app.Key).Parse(app.tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
