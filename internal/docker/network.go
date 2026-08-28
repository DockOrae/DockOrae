package docker

import (
	"context"
	"encoding/json"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

func ListNetworks(cli *client.Client, ctx context.Context, opts client.NetworkListOptions) ([]network.Summary, error) {
	res, err := cli.NetworkList(ctx, opts)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func InspectNetwork(cli *client.Client, ctx context.Context, id string) (network.Inspect, json.RawMessage, error) {
	res, err := cli.NetworkInspect(ctx, id, client.NetworkInspectOptions{})
	if err != nil {
		return network.Inspect{}, nil, err
	}
	return res.Network, res.Raw, nil
}

func CreateNetwork(cli *client.Client, ctx context.Context, name string, opts client.NetworkCreateOptions) (string, error) {
	res, err := cli.NetworkCreate(ctx, name, opts)
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

func RemoveNetwork(cli *client.Client, ctx context.Context, id string, opts client.NetworkRemoveOptions) error {
	_, err := cli.NetworkRemove(ctx, id, opts)
	return err
}

func PruneNetworks(cli *client.Client, ctx context.Context, opts client.NetworkPruneOptions) (network.PruneReport, error) {
	res, err := cli.NetworkPrune(ctx, opts)
	if err != nil {
		return network.PruneReport{}, err
	}
	return res.Report, nil
}
