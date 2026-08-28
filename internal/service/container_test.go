package service

import (
	"context"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
)

// portOf 解析端口字符串为 network.Port(测试辅助)
func portOf(t *testing.T, s string) network.Port {
	t.Helper()
	p, err := network.ParsePort(s)
	if err != nil {
		t.Fatalf("parse port %s: %v", s, err)
	}
	return p
}

// ---- buildContainerConfig:参数转换(纯函数) ----

func TestBuildContainerConfigPorts(t *testing.T) {
	ip := "192.168.1.5"
	req := model.CreateContainerReq{
		Image: "nginx",
		Ports: []model.PortMap{
			{Container: "80", Host: 8080},                   // TCP 默认值
			{Container: "443/tcp", Host: 8443, HostIP: &ip}, // 显式协议 + HostIP
			{Container: "invalid-port", Host: 1},            // 无效端口:跳过
			{Container: "53/udp", Host: 5353},               // UDP
			{Container: "8080/udp", Host: 0},                // 主机端口 0(随机)
		},
	}
	cfg, hc, err := buildContainerConfig(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TCP 默认值:80 → 80/tcp
	if _, ok := cfg.ExposedPorts[portOf(t, "80/tcp")]; !ok {
		t.Errorf("missing exposed 80/tcp, got %v", cfg.ExposedPorts)
	}
	if len(hc.PortBindings[portOf(t, "80/tcp")]) != 1 || hc.PortBindings[portOf(t, "80/tcp")][0].HostPort != "8080" {
		t.Errorf("bad binding for 80/tcp: %v", hc.PortBindings[portOf(t, "80/tcp")])
	}
	if hc.PortBindings[portOf(t, "80/tcp")][0].HostIP.String() != "0.0.0.0" {
		t.Errorf("default HostIP should be 0.0.0.0, got %s", hc.PortBindings[portOf(t, "80/tcp")][0].HostIP)
	}

	// 显式协议 + HostIP
	if _, ok := cfg.ExposedPorts[portOf(t, "443/tcp")]; !ok {
		t.Errorf("missing exposed 443/tcp")
	}
	if hc.PortBindings[portOf(t, "443/tcp")][0].HostIP.String() != ip {
		t.Errorf("HostIP not applied: %s", hc.PortBindings[portOf(t, "443/tcp")][0].HostIP)
	}

	// 无效端口跳过:仅 4 个有效端口
	if len(cfg.ExposedPorts) != 4 {
		t.Errorf("invalid port should be skipped, exposed = %v", cfg.ExposedPorts)
	}

	// UDP
	if _, ok := cfg.ExposedPorts[portOf(t, "53/udp")]; !ok {
		t.Errorf("missing exposed 53/udp")
	}
	if hc.PortBindings[portOf(t, "53/udp")][0].HostPort != "5353" {
		t.Errorf("bad binding for 53/udp: %v", hc.PortBindings[portOf(t, "53/udp")])
	}

	// 主机端口 0:HostPort 应为 "0"
	if hc.PortBindings[portOf(t, "8080/udp")][0].HostPort != "0" {
		t.Errorf("host port 0 should map to \"0\", got %q", hc.PortBindings[portOf(t, "8080/udp")][0].HostPort)
	}
}

func TestBuildContainerConfigVolumes(t *testing.T) {
	host := "/data"
	vol := "myvolume"
	ro := "ro"
	req := model.CreateContainerReq{
		Image: "nginx",
		Volumes: []model.VolumeMap{
			{Host: &host, Container: "/var/www", Mode: &ro}, // bind + ro
			{Volume: &vol, Container: "/data"},              // named volume + 默认 rw
		},
	}
	_, hc, err := buildContainerConfig(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expect := []string{"/data:/var/www:ro", "myvolume:/data:rw"}
	for _, e := range expect {
		found := false
		for _, b := range hc.Binds {
			if b == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing bind %q, got %v", e, hc.Binds)
		}
	}
}

func TestBuildContainerConfigVolumeSrcMissing(t *testing.T) {
	req := model.CreateContainerReq{
		Image: "nginx",
		Volumes: []model.VolumeMap{
			{Container: "/data"}, // 无 Host 也无 Volume
		},
	}
	_, _, err := buildContainerConfig(req)
	ae, ok := err.(*ApiError)
	if !ok || ae.Status != 400 || ae.Message != "container.mountSrcMissing" {
		t.Errorf("expected 400 mountSrcMissing, got %v", err)
	}
}

func TestBuildContainerConfigRestartPolicy(t *testing.T) {
	cases := []struct {
		policy string
		want   container.RestartPolicyMode
	}{
		{"always", container.RestartPolicyAlways},
		{"unless-stopped", container.RestartPolicyUnlessStopped},
		{"on-failure", container.RestartPolicyOnFailure},
		{"no", container.RestartPolicyDisabled},
	}
	for _, c := range cases {
		p := c.policy
		req := model.CreateContainerReq{Image: "nginx", RestartPolicy: &p}
		_, hc, err := buildContainerConfig(req)
		if err != nil {
			t.Fatalf("policy %s: %v", c.policy, err)
		}
		if hc.RestartPolicy.Name != c.want {
			t.Errorf("policy %s: got %s, want %s", c.policy, hc.RestartPolicy.Name, c.want)
		}
	}
	// nil 策略 → disabled
	req := model.CreateContainerReq{Image: "nginx"}
	_, hc, _ := buildContainerConfig(req)
	if hc.RestartPolicy.Name != container.RestartPolicyDisabled {
		t.Errorf("nil policy: got %s, want disabled", hc.RestartPolicy.Name)
	}
}

func TestBuildContainerConfigFlags(t *testing.T) {
	netName := "mynet"
	req := model.CreateContainerReq{
		Image:      "nginx",
		Network:    &netName,
		Tty:        Ptr(true),
		Privileged: Ptr(true),
		Cmd:        []string{"-g", "daemon off;"},
		Env:        []string{"A=1"},
	}
	cfg, hc, err := buildContainerConfig(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Tty || !cfg.AttachStdin || !cfg.OpenStdin {
		t.Errorf("tty should set Tty/AttachStdin/OpenStdin")
	}
	if !hc.Privileged {
		t.Errorf("privileged not applied")
	}
	if string(hc.NetworkMode) != netName {
		t.Errorf("network mode: got %s, want %s", hc.NetworkMode, netName)
	}
	if len(cfg.Cmd) != 2 || cfg.Cmd[0] != "-g" {
		t.Errorf("cmd not passed: %v", cfg.Cmd)
	}
	if len(cfg.Env) != 1 || cfg.Env[0] != "A=1" {
		t.Errorf("env not passed: %v", cfg.Env)
	}
	if cfg.Labels["createdBy"] != "docker-manager" {
		t.Errorf("missing createdBy label: %v", cfg.Labels)
	}
	if cfg.Image != "nginx" {
		t.Errorf("image not passed: %s", cfg.Image)
	}
}

// ---- ContainerService.Create:业务规则(license / image 校验,无需真实 docker) ----

// newTestContainerService 构造可测实例:license 可注入,docker 为 nil(校验路径不会触达)
func newTestContainerService(license bool) *ContainerService {
	return &ContainerService{
		docker:  nil,
		license: func() bool { return license },
	}
}

func TestContainerCreateLicenseBlocked(t *testing.T) {
	svc := newTestContainerService(false)
	_, err := svc.Create(context.Background(), model.CreateContainerReq{Image: "nginx"})
	ae, ok := err.(*ApiError)
	if !ok || ae.Status != 403 || ae.Message != "license.required" {
		t.Errorf("expected 403 license.required, got %v", err)
	}
}

func TestContainerCreateImageEmpty(t *testing.T) {
	svc := newTestContainerService(true)
	_, err := svc.Create(context.Background(), model.CreateContainerReq{})
	ae, ok := err.(*ApiError)
	if !ok || ae.Status != 400 || ae.Message != "container.imageEmpty" {
		t.Errorf("expected 400 container.imageEmpty, got %v", err)
	}
}

// TestContainerCreateVolumeErrorPropagated 卷参数错误在调用 docker 前返回
func TestContainerCreateVolumeErrorPropagated(t *testing.T) {
	svc := newTestContainerService(true)
	_, err := svc.Create(context.Background(), model.CreateContainerReq{
		Image:   "nginx",
		Volumes: []model.VolumeMap{{Container: "/data"}},
	})
	ae, ok := err.(*ApiError)
	if !ok || ae.Status != 400 || !strings.Contains(ae.Message, "mountSrcMissing") {
		t.Errorf("expected 400 mountSrcMissing, got %v", err)
	}
}
