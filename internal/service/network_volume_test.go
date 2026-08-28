package service

import (
	"context"
	"testing"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
)

func TestNetworkCreateNameEmpty(t *testing.T) {
	svc := &NetworkService{docker: nil}
	_, err := svc.Create(context.Background(), model.CreateNetworkReq{})
	ae, ok := err.(*ApiError)
	if !ok || ae.Status != 400 || ae.Message != "network.nameEmpty" {
		t.Errorf("expected 400 network.nameEmpty, got %v", err)
	}
}

func TestVolumeCreateNameEmpty(t *testing.T) {
	svc := &VolumeService{docker: nil}
	_, err := svc.Create(context.Background(), model.CreateVolumeReq{})
	ae, ok := err.(*ApiError)
	if !ok || ae.Status != 400 || ae.Message != "volume.nameEmpty" {
		t.Errorf("expected 400 volume.nameEmpty, got %v", err)
	}
}
