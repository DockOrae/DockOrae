package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/DockOrae/DockOrae/internal/agent"
)

// ---- ContainerRecreate:正常流程(最小 ops fake) ----

// fakeRecreateOps 记录调用序列的 Agent 操作 fake
type fakeRecreateOps struct {
	seq          []string
	inspectRaw   json.RawMessage
	createID     string
	createReq    agent.ContainerCreateReq
	createErr    error
	stopErr      error
	removeErr    error
	renameErr    error
	startErr     error
	removeCallID []string
}

func (f *fakeRecreateOps) ContainerInspectRaw(_ context.Context, id string) (json.RawMessage, error) {
	f.seq = append(f.seq, "inspect:"+id)
	return f.inspectRaw, nil
}
func (f *fakeRecreateOps) ContainerCreate(_ context.Context, req agent.ContainerCreateReq) (string, error) {
	f.seq = append(f.seq, "create:"+req.Name)
	f.createReq = req
	return f.createID, f.createErr
}
func (f *fakeRecreateOps) ContainerStop(_ context.Context, id string, _ *int) error {
	f.seq = append(f.seq, "stop:"+id)
	return f.stopErr
}
func (f *fakeRecreateOps) ContainerRemove(_ context.Context, id string, _ bool, _ bool) error {
	f.seq = append(f.seq, "remove:"+id)
	f.removeCallID = append(f.removeCallID, id)
	return f.removeErr
}
func (f *fakeRecreateOps) ContainerRename(_ context.Context, id, newName string) error {
	f.seq = append(f.seq, "rename:"+id+">"+newName)
	return f.renameErr
}
func (f *fakeRecreateOps) ContainerStart(_ context.Context, id string) error {
	f.seq = append(f.seq, "start:"+id)
	return f.startErr
}

// inspectJSON 构造重建用 inspect 原始 JSON
func inspectJSON(name, image string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"Name": name,
		"Config": map[string]any{
			"Image":  image,
			"Cmd":    []string{"nginx", "-g", "daemon off;"},
			"Env":    []string{"A=1"},
			"Tty":    true,
			"Labels": map[string]string{"createdBy": "docker-manager"},
		},
		"HostConfig": map[string]any{
			"RestartPolicy": map[string]any{"Name": "always"},
			"Privileged":    true,
		},
		"NetworkSettings": map[string]any{
			"Networks": map[string]any{
				"bridge": map[string]any{"IPAMConfig": nil},
			},
		},
	})
	return raw
}

func TestContainerRecreateNormalFlow(t *testing.T) {
	fake := &fakeRecreateOps{
		inspectRaw: inspectJSON("/nginx", "nginx:latest"),
		createID:   "new-container-id",
	}
	svc := &ContainerService{ops: fake, license: func() bool { return true }}

	if err := svc.Recreate(context.Background(), "old-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 调用顺序:Inspect → Create(临时名)→ Stop(旧)→ Remove(旧)→ Rename(临时名→原名)→ Start
	seq := strings.Join(fake.seq, ",")
	if !strings.HasPrefix(seq, "inspect:old-id,create:nginx-recreate-") {
		t.Fatalf("bad sequence start: %s", seq)
	}
	for _, want := range []string{"stop:old-id", "remove:old-id", "rename:new-container-id>nginx", "start:new-container-id"} {
		if !strings.Contains(seq, want) {
			t.Errorf("missing %q in sequence: %s", want, seq)
		}
	}
	// 重建配置:HostConfig 原样透传 + 原网络接入
	if fake.createReq.Name == "" || !strings.HasPrefix(fake.createReq.Name, "nginx-recreate-") {
		t.Errorf("create should use temp name, got %q", fake.createReq.Name)
	}
	var hc map[string]any
	if err := json.Unmarshal(fake.createReq.HostConfig, &hc); err != nil {
		t.Fatalf("host_config invalid: %v", err)
	}
	if hc["Privileged"] != true {
		t.Errorf("privileged not preserved: %v", hc)
	}
	var nc struct {
		EndpointsConfig map[string]any `json:"EndpointsConfig"`
	}
	if err := json.Unmarshal(fake.createReq.NetworkingConfig, &nc); err != nil {
		t.Fatalf("networking_config invalid: %v", err)
	}
	if _, ok := nc.EndpointsConfig["bridge"]; !ok {
		t.Errorf("network endpoints not preserved: %v", nc)
	}
}

// TestContainerRecreateCreateFailureRollback 创建失败:不触碰旧容器
func TestContainerRecreateCreateFailureRollback(t *testing.T) {
	fake := &fakeRecreateOps{
		inspectRaw: inspectJSON("/nginx", "nginx:latest"),
		createErr:  jsonError("create failed"),
	}
	svc := &ContainerService{ops: fake, license: func() bool { return true }}
	if err := svc.Recreate(context.Background(), "old-id"); err == nil {
		t.Fatal("expected error")
	}
	seq := strings.Join(fake.seq, ",")
	if strings.Contains(seq, "remove:old-id") {
		t.Errorf("old container should not be touched on create failure: %s", seq)
	}
}

// TestContainerRecreateRemoveFailureRollback 删旧失败:回滚删除新容器
func TestContainerRecreateRemoveFailureRollback(t *testing.T) {
	fake := &fakeRecreateOps{
		inspectRaw: inspectJSON("/nginx", "nginx:latest"),
		createID:   "new-container-id",
		removeErr:  jsonError("remove failed"),
	}
	svc := &ContainerService{ops: fake, license: func() bool { return true }}
	if err := svc.Recreate(context.Background(), "old-id"); err == nil {
		t.Fatal("expected error")
	}
	// 新容器被回滚删除(remove 第二次调用针对 new-container-id)
	if len(fake.removeCallID) < 2 || fake.removeCallID[1] != "new-container-id" {
		t.Errorf("rollback should remove new container, calls: %v", fake.removeCallID)
	}
}

// TestContainerRecreateRenameFailureStartsTmp 改名失败:启动临时名容器保证可用
func TestContainerRecreateRenameFailureStartsTmp(t *testing.T) {
	fake := &fakeRecreateOps{
		inspectRaw: inspectJSON("/nginx", "nginx:latest"),
		createID:   "new-container-id",
		renameErr:  jsonError("name in use"),
	}
	svc := &ContainerService{ops: fake, license: func() bool { return true }}
	if err := svc.Recreate(context.Background(), "old-id"); err == nil {
		t.Fatal("expected error")
	}
	seq := strings.Join(fake.seq, ",")
	if !strings.Contains(seq, "start:new-container-id") {
		t.Errorf("tmp container should be started after rename failure: %s", seq)
	}
}

// TestContainerRecreateUnavailable ops 为空:Agent 不可用
func TestContainerRecreateUnavailable(t *testing.T) {
	svc := &ContainerService{ops: nil, license: func() bool { return true }}
	err := svc.Recreate(context.Background(), "old-id")
	ae, ok := err.(*ApiError)
	if !ok || ae.Status != 502 {
		t.Errorf("expected 502 agent unavailable, got %v", err)
	}
}

type jsonError string

func (e jsonError) Error() string { return string(e) }
