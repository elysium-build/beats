package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func (d *DockerClient) UploadFile(srcPath string, dstPath string, mode string) error {
	// client := d.dockerClient
	id, _, err := d.getContainerIDAndState()
	if err != nil {
		return err
	}
	destDir, _ := filepath.Split(dstPath)
	if _, err := d.Run("", "mkdir", "-p", destDir); err != nil {
		return err
	}
	sourceFile, err := os.Open(srcPath)
	if err != nil {
		panic(err)
	}

	defer sourceFile.Close()
	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
		CopyUIDGID:                false,
	}
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()
	return d.dockerClient.CopyToContainer(ctx, id, destDir, sourceFile, options)
}

func (d *DockerClient) getContainerIDAndState() (id string, state string, err error) {
	var containerDetail types.Container
	containerDetail, err = d.getContainerBriefDetails(true)
	if err != nil {
		return
	}
	id = containerDetail.ID
	state = containerDetail.State
	err = nil
	return
}

func (d *DockerClient) getContainerBriefDetails(all bool) (types.Container, error) {
	err := d.CheckClient()
	if err != nil {
		return types.Container{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	filter := filters.NewArgs()
	for _, value := range d.Filter {
		kv := strings.Split(value, ":")
		if len(kv) == 2 {
			filter.Add(kv[0], kv[1])
		}
	}
	d.actionMutex.RLock()
	containers, err := d.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: all, Filters: filter})
	d.actionMutex.RUnlock()
	if err != nil {
		return types.Container{}, err
	}
	if len(containers) == 0 {
		return types.Container{}, fmt.Errorf("Container %v doesnot exist", d.Filter)
	}
	if len(containers) > 1 {
		return types.Container{}, fmt.Errorf("Return more than 1 container with filter %v", d.Filter)
	}
	return containers[0], err
}

func (d *DockerClient) inspectContainer() (types.ContainerJSON, error) {
	err := d.CheckClient()
	if err != nil {
		return types.ContainerJSON{}, err
	}

	id, _, err := d.getContainerIDAndState()

	if err != nil {
		return types.ContainerJSON{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()
	d.actionMutex.RLock()
	containerJSON, err := d.dockerClient.ContainerInspect(ctx, id)
	d.actionMutex.RUnlock()
	return containerJSON, err
}
