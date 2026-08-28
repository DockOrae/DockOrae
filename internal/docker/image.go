package docker

import (
	"context"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

func ListImages(cli *client.Client, ctx context.Context, opts client.ImageListOptions) ([]image.Summary, error) {
	res, err := cli.ImageList(ctx, opts)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func InspectImage(cli *client.Client, ctx context.Context, id string) (image.InspectResponse, error) {
	res, err := cli.ImageInspect(ctx, id)
	if err != nil {
		return image.InspectResponse{}, err
	}
	return res.InspectResponse, nil
}

func PullImage(cli *client.Client, ctx context.Context, ref string, opts client.ImagePullOptions) (client.ImagePullResponse, error) {
	return cli.ImagePull(ctx, ref, opts)
}

func RemoveImage(cli *client.Client, ctx context.Context, id string, opts client.ImageRemoveOptions) error {
	_, err := cli.ImageRemove(ctx, id, opts)
	return err
}

func PruneImages(cli *client.Client, ctx context.Context, opts client.ImagePruneOptions) (image.PruneReport, error) {
	res, err := cli.ImagePrune(ctx, opts)
	if err != nil {
		return image.PruneReport{}, err
	}
	return res.Report, nil
}

func TagImage(cli *client.Client, ctx context.Context, source, target string) error {
	_, err := cli.ImageTag(ctx, client.ImageTagOptions{Source: source, Target: target})
	return err
}
