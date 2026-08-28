package appstore

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------- 1Panel appstore data.yml 结构 ----------

type appDataYaml struct {
	Name        string   `yaml:"name"`
	Tags        []string `yaml:"tags"`
	Description string   `yaml:"description"`
	Additional  struct {
		Key         string   `yaml:"key"`
		Name        string   `yaml:"name"`
		Tags        []string `yaml:"tags"`
		ShortDescZh string   `yaml:"shortDescZh"`
		ShortDescEn string   `yaml:"shortDescEn"`
	} `yaml:"additionalProperties"`
}

type formFieldYaml struct {
	Default  string `yaml:"default"`
	EnvKey   string `yaml:"envKey"`
	LabelEn  string `yaml:"labelEn"`
	LabelZh  string `yaml:"labelZh"`
	Random   bool   `yaml:"random"`
	Required bool   `yaml:"required"`
	Rule     string `yaml:"rule"`
	Type     string `yaml:"type"`
	Label    struct {
		En string `yaml:"en"`
		Zh string `yaml:"zh"`
	} `yaml:"label"`
	Values []struct {
		Label string `yaml:"label"`
		Value string `yaml:"value"`
	} `yaml:"values"`
}

type versionDataYaml struct {
	Additional struct {
		FormFields []formFieldYaml `yaml:"formFields"`
	} `yaml:"additionalProperties"`
}

// ---------- 解析 ----------

// versionData 单个版本的数据
type versionData struct {
	Params []Param
	Tpl    string
	Dir    string // 版本目录(conf/scripts 等附加文件所在)
}

// LoadApps 扫描 appstore 数据目录,解析全部应用与分类。
// dir 为仓库根(含 apps/ 子目录)。
func LoadApps(dir string) ([]*App, []string, error) {
	appsDir := filepath.Join(dir, "apps")
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, nil, err
	}
	var apps []*App
	catSet := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		a, err := LoadApp(dir, e.Name())
		if err != nil || a == nil {
			continue
		}
		apps = append(apps, a)
		if a.Category != "" {
			catSet[a.Category] = true
		}
	}
	cats := make([]string, 0, len(catSet))
	for c := range catSet {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return apps, cats, nil
}

// LoadApp 解析单个应用(元数据 + 全部版本)
func LoadApp(dir, key string) (*App, error) {
	appDir := filepath.Join(dir, "apps", key)
	raw, err := os.ReadFile(filepath.Join(appDir, "data.yml"))
	if err != nil {
		return nil, err
	}
	var meta appDataYaml
	if err := yaml.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	key = strings.TrimSpace(meta.Additional.Key)
	if key == "" {
		key = filepath.Base(appDir)
	}
	name := strings.TrimSpace(meta.Additional.Name)
	if name == "" {
		name = meta.Name
	}
	desc := strings.TrimSpace(meta.Additional.ShortDescZh)
	if desc == "" {
		desc = meta.Additional.ShortDescEn
	}
	if desc == "" {
		desc = meta.Description
	}
	cat := ""
	if len(meta.Tags) > 0 {
		cat = meta.Tags[0]
	}
	if cat == "" && len(meta.Additional.Tags) > 0 {
		cat = meta.Additional.Tags[0]
	}

	// 版本目录(含 docker-compose.yml 的子目录)
	verEntries, err := os.ReadDir(appDir)
	if err != nil {
		return nil, err
	}
	var vers []string
	vd := map[string]*versionData{}
	for _, e := range verEntries {
		if !e.IsDir() {
			continue
		}
		v := e.Name()
		tpl, err := os.ReadFile(filepath.Join(appDir, v, "docker-compose.yml"))
		if err != nil {
			continue // 非版本目录(如 conf)
		}
		vers = append(vers, v)
		vd[v] = &versionData{
			Tpl:    string(tpl),
			Dir:    filepath.Join(appDir, v),
			Params: parseFormFields(filepath.Join(appDir, v, "data.yml")),
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(vers)))
	if len(vers) == 0 {
		return nil, os.ErrNotExist
	}

	// 最新版本参数 + 端口
	latest := vd[vers[0]]
	ports := []string{}
	for _, p := range latest.Params {
		if p.Rule == "paramPort" && p.Default != "" {
			ports = append(ports, p.Default)
		}
	}
	return &App{
		Key:         key,
		Name:        name,
		Icon:        "",
		Category:    cat,
		Description: desc,
		Ports:       ports,
		Versions:    vers,
		Params:      latest.Params,
		dir:         appDir,
		versions:    vd,
	}, nil
}

// parseFormFields 解析版本 data.yml 的 formFields → 前端参数
func parseFormFields(path string) []Param {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var vd versionDataYaml
	if err := yaml.Unmarshal(raw, &vd); err != nil {
		return nil
	}
	var out []Param
	for _, f := range vd.Additional.FormFields {
		key := strings.TrimSpace(f.EnvKey)
		if key == "" {
			continue
		}
		labelZh := strings.TrimSpace(f.LabelZh)
		if labelZh == "" {
			labelZh = f.Label.Zh
		}
		labelEn := strings.TrimSpace(f.LabelEn)
		if labelEn == "" {
			labelEn = f.Label.En
		}
		typ := f.Type
		// 1Panel 联动类型 → select
		if typ == "apps" || typ == "service" {
			typ = "select"
		}
		if typ != "text" && typ != "password" && typ != "number" && typ != "select" && typ != "checkbox" && typ != "textarea" {
			typ = "text"
		}
		p := Param{
			Key:      key,
			LabelZh:  labelZh,
			LabelEn:  labelEn,
			Type:     typ,
			Default:  f.Default,
			Required: f.Required,
			Random:   f.Random,
			Rule:     f.Rule,
		}
		for _, v := range f.Values {
			p.Options = append(p.Options, Option{Label: v.Label, Value: v.Value})
		}
		out = append(out, p)
	}
	return out
}
