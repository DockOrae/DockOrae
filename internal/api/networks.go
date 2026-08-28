package api

import (
	"net/netip"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func networksList(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.NetworkList(c.Request.Context(), client.NetworkListOptions{})
	if err != nil {
		return dockerError(err)
	}
	// 精简:仅返回列表展示字段;新版 moby /networks 不返回容器端点信息(与旧行为一致)
	items := make([]networkListItem, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, networkListItem{
			ID: it.ID, Name: it.Name, Driver: it.Driver, Scope: it.Scope, IPAM: it.IPAM,
		})
	}
	c.JSON(200, items)
	return nil
}

type networkListItem struct {
	ID     string       `json:"Id"`
	Name   string       `json:"Name"`
	Driver string       `json:"Driver"`
	Scope  string       `json:"Scope"`
	IPAM   network.IPAM `json:"IPAM"`
}

func networksInspect(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.NetworkInspect(c.Request.Context(), c.Param("id"), client.NetworkInspectOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.Data(200, "application/json", res.Raw)
	return nil
}

type createNetReq struct {
	Name     string  `json:"name"`
	Driver   *string `json:"driver"`
	Subnet   *string `json:"subnet"`
	Gateway  *string `json:"gateway"`
	Internal *bool   `json:"internal"`
}

func networksCreate(c *gin.Context, st *state.AppState) error {
	var req createNetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	if req.Name == "" {
		return BadRequest("network.nameEmpty")
	}
	driver := "bridge"
	if req.Driver != nil && *req.Driver != "" {
		driver = *req.Driver
	}
	opts := client.NetworkCreateOptions{
		Driver:    driver,
		Internal:  req.Internal != nil && *req.Internal,
		EnableIPv6: ptr(false),
	}
	if req.Subnet != nil || req.Gateway != nil {
		ipamCfg := network.IPAMConfig{}
		if req.Subnet != nil {
			if p, err := netip.ParsePrefix(*req.Subnet); err == nil {
				ipamCfg.Subnet = p
			}
		}
		if req.Gateway != nil {
			if a, err := netip.ParseAddr(*req.Gateway); err == nil {
				ipamCfg.Gateway = a
			}
		}
		opts.IPAM = &network.IPAM{Driver: "default", Config: []network.IPAMConfig{ipamCfg}}
	}
	res, err := st.Docker.NetworkCreate(c.Request.Context(), req.Name, opts)
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"id": res.ID, "warning": res.Warning})
	return nil
}

func ptr[T any](v T) *T { return &v }

func networksRemove(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.NetworkRemove(c.Request.Context(), c.Param("id"), client.NetworkRemoveOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func networksPrune(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.NetworkPrune(c.Request.Context(), client.NetworkPruneOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Report)
	return nil
}
