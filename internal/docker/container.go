package docker

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func ListContainers(cli *client.Client, ctx context.Context, opts client.ContainerListOptions) ([]container.Summary, error) {
	res, err := cli.ContainerList(ctx, opts)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func InspectContainer(cli *client.Client, ctx context.Context, id string) (container.InspectResponse, error) {
	res, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return res.Container, nil
}

func CreateContainer(cli *client.Client, ctx context.Context, opts client.ContainerCreateOptions) (string, error) {
	res, err := cli.ContainerCreate(ctx, opts)
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

func StartContainer(cli *client.Client, ctx context.Context, id string) error {
	_, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
	return err
}

func StopContainer(cli *client.Client, ctx context.Context, id string, opts client.ContainerStopOptions) error {
	_, err := cli.ContainerStop(ctx, id, opts)
	return err
}

func RestartContainer(cli *client.Client, ctx context.Context, id string, opts client.ContainerRestartOptions) error {
	_, err := cli.ContainerRestart(ctx, id, opts)
	return err
}

func KillContainer(cli *client.Client, ctx context.Context, id string, signal string) error {
	_, err := cli.ContainerKill(ctx, id, client.ContainerKillOptions{Signal: signal})
	return err
}

func PauseContainer(cli *client.Client, ctx context.Context, id string) error {
	_, err := cli.ContainerPause(ctx, id, client.ContainerPauseOptions{})
	return err
}

func UnpauseContainer(cli *client.Client, ctx context.Context, id string) error {
	_, err := cli.ContainerUnpause(ctx, id, client.ContainerUnpauseOptions{})
	return err
}

func RenameContainer(cli *client.Client, ctx context.Context, id, newName string) error {
	_, err := cli.ContainerRename(ctx, id, client.ContainerRenameOptions{NewName: newName})
	return err
}

func RemoveContainer(cli *client.Client, ctx context.Context, id string, opts client.ContainerRemoveOptions) error {
	_, err := cli.ContainerRemove(ctx, id, opts)
	return err
}

func PruneContainers(cli *client.Client, ctx context.Context, opts client.ContainerPruneOptions) (container.PruneReport, error) {
	res, err := cli.ContainerPrune(ctx, opts)
	if err != nil {
		return container.PruneReport{}, err
	}
	return res.Report, nil
}

// WaitContainer 等待容器退出,返回退出码。
// 容器运行中退出 → (code, nil);等待过程出错/ctx 取消 → (0, err)。
func WaitContainer(cli *client.Client, ctx context.Context, id string) (int64, error) {
	res := cli.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})
	select {
	case r := <-res.Result:
		return r.StatusCode, nil
	case err := <-res.Error:
		return 0, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func ContainerStats(cli *client.Client, ctx context.Context, id string, opts client.ContainerStatsOptions) (*client.ContainerStatsResult, error) {
	res, err := cli.ContainerStats(ctx, id, opts)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
func ContainerLogs(cli *client.Client, ctx context.Context, id string, opts client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	return cli.ContainerLogs(ctx, id, opts)
}

func ExecCreate(cli *client.Client, ctx context.Context, id string, opts client.ExecCreateOptions) (string, error) {
	res, err := cli.ExecCreate(ctx, id, opts)
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

func ExecAttach(cli *client.Client, ctx context.Context, execID string, opts client.ExecAttachOptions) (*client.ExecAttachResult, error) {
	res, err := cli.ExecAttach(ctx, execID, opts)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func ExecResize(cli *client.Client, ctx context.Context, execID string, opts client.ExecResizeOptions) error {
	_, err := cli.ExecResize(ctx, execID, opts)
	return err
}
