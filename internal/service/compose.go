package service

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/state"
)

func SortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// ValidateProject 项目名校验(字母数字 _ - ,≤64)
func ValidateProject(p string) (string, error) {
	if p == "" || len(p) > 64 {
		return "", BadRequest("compose.nameInvalid")
	}
	for _, r := range p {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", BadRequest("compose.nameInvalid")
	}
	return p, nil
}

// ComposeService 栈业务(§11:YAML/项目管理在面板,执行在 Agent)。
type ComposeService struct {
	agent      *agent.Client
	composeDir string
	license    func() bool
}

// NewComposeService 生产构造:从 AppState 提取实际依赖
func NewComposeService(st *state.AppState) *ComposeService {
	return &ComposeService{
		agent:      st.Agent,
		composeDir: st.ComposeDir,
		license:    func() bool { return LicenseFeatureActive(st, "compose") },
	}
}

// projectDir 项目目录(composeDir + project)
func projectDir(composeDir, project string) string {
	return filepath.Join(composeDir, project)
}

// composeFile 项目编排文件路径
func composeFile(composeDir, project string) string {
	return filepath.Join(projectDir(composeDir, project), "docker-compose.yml")
}

// File 项目编排文件路径(供 api 层检查/读写)
func (s *ComposeService) File(project string) string {
	return composeFile(s.composeDir, project)
}

// Dir 项目目录
func (s *ComposeService) Dir(project string) string {
	return projectDir(s.composeDir, project)
}

// List 栈列表:按 compose label 统计容器,计算状态;外部 compose 不显示。
// 容器数据来自 Agent,面板数据目录校验在本地。
func (s *ComposeService) List(ctx context.Context) ([]model.ComposeProject, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	items, err := s.agent.ContainerList(ctx, true)
	if err != nil {
		return nil, err
	}
	projects := map[string][2]int{} // name -> [total, running]
	for _, ctr := range items {
		name := ctr.Labels["com.docker.compose.project"]
		if name == "" {
			continue
		}
		entry := projects[name]
		entry[0]++
		if ctr.State == "running" {
			entry[1]++
		}
		projects[name] = entry
	}
	names := make([]string, 0, len(projects))
	for name := range projects {
		names = append(names, name)
	}
	SortStrings(names)

	out := make([]model.ComposeProject, 0, len(names))
	for _, name := range names {
		_, hasFile := os.Stat(composeFile(s.composeDir, name))
		if hasFile != nil {
			continue // 外部 compose(宿主直接部署):面板不管理,不显示
		}
		total, running := projects[name][0], projects[name][1]
		status := "stopped"
		if running == total {
			status = "running"
		} else if running > 0 {
			status = "partial"
		}
		out = append(out, model.ComposeProject{
			Project:  name,
			Services: total,
			Running:  running,
			Status:   status,
			Managed:  true,
		})
	}
	return out, nil
}

// Inspect 栈详情:容器列表(精简)+ 编排文件内容
func (s *ComposeService) Inspect(ctx context.Context, project string) (model.ComposeInspect, error) {
	if s.agent == nil {
		return model.ComposeInspect{}, agentUnavailable()
	}
	items, err := s.agent.ContainerList(ctx, true)
	if err != nil {
		return model.ComposeInspect{}, err
	}
	var matched []agent.ContainerSummary
	for _, ctr := range items {
		if ctr.Labels["com.docker.compose.project"] == project {
			matched = append(matched, ctr)
		}
	}
	var yaml *string
	managed := false
	if raw, err := os.ReadFile(composeFile(s.composeDir, project)); err == nil {
		s := string(raw)
		yaml = &s
		managed = true
	}
	return model.ComposeInspect{
		Project:    project,
		Managed:    managed,
		Containers: model.ToContainerItems(matched),
		Yaml:       yaml,
	}, nil
}

// SaveYaml 校验许可证并写入编排文件(composeUp/composeUpdate 共用)
func (s *ComposeService) SaveYaml(project, yaml string) error {
	if !s.license() {
		return NewApiError(403, "license.required")
	}
	dir := projectDir(s.composeDir, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(composeFile(s.composeDir, project), []byte(yaml), 0o644)
}

// Remove 停栈并删除编排目录
func (s *ComposeService) Remove(ctx context.Context, project string) error {
	_, _ = s.Run(ctx, project, "down")
	return os.RemoveAll(projectDir(s.composeDir, project))
}

// Adopt 接管外部创建的栈:把 yaml 保存到面板数据目录,使其变为面板管理。
// 仅保存文件不部署(用户后续可编辑/up);不做许可证限制(接管≠创建)。
func (s *ComposeService) Adopt(project, yaml string) error {
	if strings.TrimSpace(yaml) == "" {
		return BadRequest("compose.yamlEmpty")
	}
	dir := projectDir(s.composeDir, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(composeFile(s.composeDir, project), []byte(yaml), 0o644)
}

// ---------- Agent 执行 ----------

// collectFiles 收集项目目录附加文件(相对路径 → base64),docker-compose.yml 除外
func (s *ComposeService) collectFiles(project string) map[string]string {
	files := map[string]string{}
	root := projectDir(s.composeDir, project)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "docker-compose.yml" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files[filepath.ToSlash(rel)] = base64.StdEncoding.EncodeToString(raw)
		return nil
	})
	return files
}

// Run 同步执行 compose 动作(start/stop/restart/down/build):面板保存 YAML → Agent 执行
func (s *ComposeService) Run(ctx context.Context, project, action string, args ...string) (map[string]any, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	yamlRaw, err := os.ReadFile(s.File(project))
	if err != nil {
		return nil, NewApiError(404, "compose.notManaged")
	}
	all := append([]string{action}, args...)
	res, err := s.agent.ComposeRun(ctx, project, string(yamlRaw), s.collectFiles(project), all...)
	if err != nil {
		return nil, agentToApiError(err)
	}
	return map[string]any{"ok": res.OK, "output": res.Output}, nil
}

// UpStream 流式 compose up:面板保存 YAML → Agent 执行,返回 NDJSON 流
func (s *ComposeService) UpStream(ctx context.Context, project, yaml, args string) (*agent.StreamBody, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	if err := s.SaveYaml(project, yaml); err != nil {
		return nil, err
	}
	return s.agent.ComposeUpStream(ctx, agent.ComposeStreamReq{
		Project: project,
		Yaml:    yaml,
		Files:   s.collectFiles(project),
	}, args)
}

// PullStream 流式 compose pull(应用商店安装预拉取)
func (s *ComposeService) PullStream(ctx context.Context, project string) (*agent.StreamBody, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	yamlRaw, err := os.ReadFile(s.File(project))
	if err != nil {
		return nil, NewApiError(404, "compose.notManaged")
	}
	return s.agent.ComposePullStream(ctx, agent.ComposeStreamReq{
		Project: project,
		Yaml:    string(yamlRaw),
		Files:   s.collectFiles(project),
	})
}

// agentToApiError Agent 错误 → 面板 ApiError(透传状态码与消息)
func agentToApiError(err error) error {
	var ae *agent.AgentError
	if errors.As(err, &ae) {
		return NewApiError(ae.Status, ae.Message)
	}
	return err
}
