package service

import (
	"context"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/DockerManger/Docker_Manager_Go/internal/appstore"
)

// ---- ContainerRecreate:正常流程(最小 ops fake) ----

// fakeContainerOps 记录调用序列的容器操作 fake
type fakeContainerOps struct {
	seq          []string
	inspectResp  container.InspectResponse
	createID     string
	createOpts   client.ContainerCreateOptions
	createErr    error
	stopErr      error
	removeErr    error
	renameErr    error
	startErr     error
	removeCallID []string
}

func (f *fakeContainerOps) Inspect(_ context.Context, id string) (container.InspectResponse, error) {
	f.seq = append(f.seq, "inspect:"+id)
	return f.inspectResp, nil
}
func (f *fakeContainerOps) Create(_ context.Context, opts client.ContainerCreateOptions) (string, error) {
	f.seq = append(f.seq, "create:"+opts.Name)
	f.createOpts = opts
	return f.createID, f.createErr
}
func (f *fakeContainerOps) Stop(_ context.Context, id string, _ client.ContainerStopOptions) error {
	f.seq = append(f.seq, "stop:"+id)
	return f.stopErr
}
func (f *fakeContainerOps) Remove(_ context.Context, id string, _ client.ContainerRemoveOptions) error {
	f.seq = append(f.seq, "remove:"+id)
	f.removeCallID = append(f.removeCallID, id)
	return f.removeErr
}
func (f *fakeContainerOps) Rename(_ context.Context, id, newName string) error {
	f.seq = append(f.seq, "rename:"+id+">"+newName)
	return f.renameErr
}
func (f *fakeContainerOps) Start(_ context.Context, id string) error {
	f.seq = append(f.seq, "start:"+id)
	return f.startErr
}

func TestContainerRecreateNormalFlow(t *testing.T) {
	insp := container.InspectResponse{
		Name: "/nginx",
		Config: &container.Config{
			Image: "nginx:latest",
		},
		HostConfig:      &container.HostConfig{},
		NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{}},
	}
	fake := &fakeContainerOps{
		inspectResp: insp,
		createID:    "new-container-id",
	}
	svc := &ContainerService{ops: fake, license: func() bool { return true }}

	if err := svc.Recreate(context.Background(), "old-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 调用顺序:Inspect → Create(临时名)→ Stop(旧)→ Remove(旧)→ Rename(临时名→原名)→ Start
	want := []string{
		"inspect:old-id",
		"create:nginx-recreate-", // 临时名(随机后缀,前缀匹配)
		"stop:old-id",
		"remove:old-id",
		"rename:new-container-id>nginx",
		"start:new-container-id",
	}
	if len(fake.seq) != len(want) {
		t.Fatalf("call sequence = %v, want %d calls", fake.seq, len(want))
	}
	for i, w := range want {
		if !strings.HasPrefix(fake.seq[i], w) {
			t.Errorf("call[%d] = %q, want prefix %q", i, fake.seq[i], w)
		}
	}
	// 创建用临时名而非原名(旧容器占用原名,同名必然 409)
	if fake.createOpts.Name == "nginx" {
		t.Errorf("create must use temp name, got %q", fake.createOpts.Name)
	}
	// 删除的是旧容器(newID 不应被删)
	if len(fake.removeCallID) != 1 || fake.removeCallID[0] != "old-id" {
		t.Errorf("remove calls = %v, want only old-id", fake.removeCallID)
	}
}

// TestContainerRecreateRemoveFailedRollback 删旧失败 → 回滚删除临时容器,旧容器不受影响
func TestContainerRecreateRemoveFailedRollback(t *testing.T) {
	fake := &fakeContainerOps{
		inspectResp: container.InspectResponse{
			Name:            "/nginx",
			Config:          &container.Config{Image: "nginx:latest"},
			HostConfig:      &container.HostConfig{},
			NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{}},
		},
		createID:  "new-id",
		removeErr: errFakeRemove,
	}
	svc := &ContainerService{ops: fake, license: func() bool { return true }}
	if err := svc.Recreate(context.Background(), "old-id"); err == nil {
		t.Fatal("expected error when remove fails")
	}
	// 回滚:临时容器被删除(remove:new-id)
	found := false
	for _, id := range fake.removeCallID {
		if id == "new-id" {
			found = true
		}
	}
	if !found {
		t.Errorf("rollback should remove temp container, remove calls = %v", fake.removeCallID)
	}
}

// TestContainerRecreateCreateFailed 创建失败 → 旧容器完好(无后续操作)
func TestContainerRecreateCreateFailed(t *testing.T) {
	fake := &fakeContainerOps{
		inspectResp: container.InspectResponse{
			Name:            "/nginx",
			Config:          &container.Config{Image: "nginx:latest"},
			HostConfig:      &container.HostConfig{},
			NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{}},
		},
		createErr: errFakeCreate,
	}
	svc := &ContainerService{ops: fake, license: func() bool { return true }}
	if err := svc.Recreate(context.Background(), "old-id"); err == nil {
		t.Fatal("expected error when create fails")
	}
	if len(fake.seq) != 2 { // inspect + create
		t.Errorf("after create failure only inspect+create expected, got %v", fake.seq)
	}
}

var errFakeRemove = &ApiError{Status: 500, Message: "fake remove err"}
var errFakeCreate = &ApiError{Status: 500, Message: "fake create err"}

// ---- 纯函数:ValidKey / ValidateProject / ImagePullRef ----

func TestAppKeyValidation(t *testing.T) {
	valid := []string{"mysql", "nginx-proxy-manager", "portainer-ce", "2fauth", "a.b_c-1"}
	for _, k := range valid {
		if !appstore.ValidKey(k) {
			t.Errorf("ValidKey(%q) = false, want true", k)
		}
	}
	invalid := []string{"", "../etc", "a/b", "..", ".", "a b", "../x"}
	for _, k := range invalid {
		if appstore.ValidKey(k) {
			t.Errorf("ValidKey(%q) = true, want false", k)
		}
	}
}

func TestValidateProject(t *testing.T) {
	valid := []string{"gitea", "my-stack_1"}
	for _, p := range valid {
		if _, err := ValidateProject(p); err != nil {
			t.Errorf("ValidateProject(%q) error: %v", p, err)
		}
	}
	invalid := []string{"", "../x", "a/b", "a b", strings.Repeat("x", 65)}
	for _, p := range invalid {
		if _, err := ValidateProject(p); err == nil {
			t.Errorf("ValidateProject(%q) should fail", p)
		}
	}
}

func TestImagePullRef(t *testing.T) {
	cases := []struct{ img, tag, want string }{
		{"nginx", "latest", "nginx:latest"},
		{"nginx:1.25", "latest", "nginx:1.25"}, // 已带 tag:不拼接
		{"registry.local/x/y", "1.0", "registry.local/x/y:1.0"},
		{"registry.local/x/y:2.0", "1.0", "registry.local/x/y:2.0"},
	}
	for _, c := range cases {
		if got := ImagePullRef(c.img, c.tag); got != c.want {
			t.Errorf("ImagePullRef(%q, %q) = %q, want %q", c.img, c.tag, got, c.want)
		}
	}
}
