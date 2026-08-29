package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/moby/moby/client"

	"github.com/DockerManger/Docker_Manager_Go/internal/docker"
	"github.com/DockerManger/Docker_Manager_Go/internal/model"
	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// ComposeBin compose 可执行文件(COMPOSE_BIN 环境变量可覆盖,默认 docker-compose)
func ComposeBin() string {
	if b := os.Getenv("COMPOSE_BIN"); b != "" {
		return b
	}
	return "docker-compose"
}

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

// ComposeService 栈业务:依赖注入(docker client + 数据目录 + license 检查)
type ComposeService struct {
	docker     *client.Client
	composeDir string
	license    func() bool
}

// NewComposeService 生产构造:从 AppState 提取实际依赖
func NewComposeService(st *state.AppState) *ComposeService {
	return &ComposeService{
		docker:     st.Docker,
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

// Command 构造 compose 命令(未启动;流式执行由 api 层控制)
func (s *ComposeService) Command(project string, args ...string) *exec.Cmd {
	return exec.Command(ComposeBin(), append([]string{"-p", project, "-f", composeFile(s.composeDir, project)}, args...)...)
}

// Run 同步执行 compose 命令,返回 {ok, output}
func (s *ComposeService) Run(project string, args ...string) (map[string]any, error) {
	cmd := s.Command(project, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, NewApiError(502, msg)
	}
	output := strings.TrimSpace(string(out))
	return map[string]any{"ok": true, "output": output}, nil
}

// File 项目编排文件路径(供 api 层检查/读写)
func (s *ComposeService) File(project string) string {
	return composeFile(s.composeDir, project)
}

// Dir 项目目录
func (s *ComposeService) Dir(project string) string {
	return projectDir(s.composeDir, project)
}

// List 栈列表:按 compose label 统计容器,计算状态;外部 compose 不显示
func (s *ComposeService) List(ctx context.Context) ([]model.ComposeProject, error) {
	filters := make(client.Filters)
	filters.Add("label", "com.docker.compose.project")
	items, err := docker.ListContainers(s.docker, ctx, client.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
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
		if string(ctr.State) == "running" {
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
	filters := make(client.Filters)
	filters.Add("label", "com.docker.compose.project="+project)
	items, err := docker.ListContainers(s.docker, ctx, client.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return model.ComposeInspect{}, err
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
		Containers: model.ToContainerItems(items),
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
func (s *ComposeService) Remove(project string) error {
	_, _ = s.Run(project, "down")
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
