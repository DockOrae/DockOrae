package docker

import (
	"context"

	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

func DockerInfo(cli *client.Client, ctx context.Context) (system.Info, error) {
	res, err := cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		return system.Info{}, err
	}
	return res.Info, nil
}
