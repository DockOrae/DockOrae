package docker

import (
	"context"
	"encoding/json"

	"github.com/moby/moby/api/types/plugin"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

func ListVolumes(cli *client.Client, ctx context.Context, opts client.VolumeListOptions) ([]volume.Volume, error) {
	res, err := cli.VolumeList(ctx, opts)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func InspectVolume(cli *client.Client, ctx context.Context, name string) (volume.Volume, json.RawMessage, error) {
	res, err := cli.VolumeInspect(ctx, name, client.VolumeInspectOptions{})
	if err != nil {
		return volume.Volume{}, nil, err
	}
	return res.Volume, res.Raw, nil
}

func CreateVolume(cli *client.Client, ctx context.Context, opts client.VolumeCreateOptions) (volume.Volume, error) {
	res, err := cli.VolumeCreate(ctx, opts)
	if err != nil {
		return volume.Volume{}, err
	}
	return res.Volume, nil
}

func RemoveVolume(cli *client.Client, ctx context.Context, name string, force bool) error {
	_, err := cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: force})
	return err
}

func PruneVolumes(cli *client.Client, ctx context.Context, opts client.VolumePruneOptions) (volume.PruneReport, error) {
	res, err := cli.VolumePrune(ctx, opts)
	if err != nil {
		return volume.PruneReport{}, err
	}
	return res.Report, nil
}

func ListPlugins(cli *client.Client, ctx context.Context) ([]plugin.Plugin, error) {
	res, err := cli.PluginList(ctx, client.PluginListOptions{})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}
