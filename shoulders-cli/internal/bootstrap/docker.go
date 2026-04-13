package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// dockerClient returns a new Docker API client configured from the
// environment (DOCKER_HOST, DOCKER_API_VERSION, etc.).
func dockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

// containerExists checks whether a Docker container with the given name exists.
func containerExists(ctx context.Context, containerName string) (bool, error) {
	cli, err := dockerClient()
	if err != nil {
		return false, fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close() //nolint:errcheck // best-effort cleanup

	_, err = cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("inspect container %q: %w", containerName, err)
	}
	return true, nil
}

// removeContainer force-removes a Docker container by name, ignoring
// not-found errors.
func removeContainer(ctx context.Context, name string) error {
	cli, err := dockerClient()
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close() //nolint:errcheck // best-effort cleanup

	err = cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("remove container %q: %w", name, err)
	}
	return nil
}

// removeVolume force-removes a Docker volume by name, ignoring not-found
// errors.
func removeVolume(ctx context.Context, name string) error {
	cli, err := dockerClient()
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close() //nolint:errcheck // best-effort cleanup

	err = cli.VolumeRemove(ctx, name, true)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("remove volume %q: %w", name, err)
	}
	return nil
}

// listContainerNames returns names of all containers whose name starts with
// the given prefix (including stopped containers).
func listContainerNames(ctx context.Context, namePrefix string) ([]string, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close() //nolint:errcheck // best-effort cleanup

	f := filters.NewArgs(filters.Arg("name", "^"+namePrefix))
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var names []string
	for _, c := range containers {
		for _, n := range c.Names {
			// Docker prefixes names with "/".
			names = append(names, strings.TrimPrefix(n, "/"))
		}
	}
	return names, nil
}

// stopContainer stops a running Docker container by name, ignoring
// not-found errors.
func stopContainer(ctx context.Context, name string) error {
	cli, err := dockerClient()
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close() //nolint:errcheck // best-effort cleanup

	err = cli.ContainerStop(ctx, name, container.StopOptions{})
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("stop container %q: %w", name, err)
	}
	return nil
}

// startContainer starts a stopped Docker container by name.
func startContainer(ctx context.Context, name string) error {
	cli, err := dockerClient()
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close() //nolint:errcheck // best-effort cleanup

	err = cli.ContainerStart(ctx, name, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("start container %q: %w", name, err)
	}
	return nil
}
