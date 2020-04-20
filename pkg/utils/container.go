package utils

import (
	"bufio"
	"fmt"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

//CreateAndRunContainer run container with defined name from image with port and volume mapping and defined command
func CreateAndRunContainer(containerName string, imageName string, portMap map[string]string, volumeMap map[string]string, command []string) error {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	if _, err = cli.ImagePull(ctx, imageName, types.ImagePullOptions{}); err != nil {
		return err
	}
	portBinding := nat.PortMap{}
	portExposed := nat.PortSet{}
	for intport, binding := range portMap {
		port, err := nat.NewPort("tcp", intport)
		if err != nil {
			return err
		}
		portExposed[port] = struct{}{}
		portBinding[port] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: binding,
			},
		}
	}
	var mounts []mount.Mount
	for target, source := range volumeMap {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: source,
			Target: target,
		})
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBinding,
		Mounts:       mounts,
	}
	userCurrent, err := user.Current()
	if err != nil {
		return err
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        imageName,
		Cmd:          command,
		ExposedPorts: portExposed,
		User:         fmt.Sprintf("%s:%s", userCurrent.Uid, userCurrent.Gid),
	}, hostConfig, nil, containerName)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	log.Infof("started container: %s", resp.ID)
	return nil
}

//StopContainer stop container and remove if remove is true
func StopContainer(containerName string, remove bool) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}
	for _, cont := range containers {
		isFound := false
		for _, name := range cont.Names {
			if strings.Contains(name, containerName) {
				isFound = true
				break
			}
		}
		if isFound {
			if cont.State != "running" {
				if remove {
					if err = cli.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{}); err != nil {
						return err
					}
				}
				return nil
			}
			if remove {
				if err = cli.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
					return err
				}
			} else {
				timeout := time.Duration(10) * time.Second
				if err = cli.ContainerStop(ctx, cont.ID, &timeout); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return fmt.Errorf("container not found")
}

//StateContainer return state of container if found or "" state if not found
func StateContainer(containerName string) (state string, err error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return "", err
	}
	for _, cont := range containers {
		isFound := false
		for _, name := range cont.Names {
			if strings.Contains(name, containerName) {
				isFound = true
				break
			}
		}
		if isFound {
			return cont.State, nil
		}
	}
	return "", nil
}

//StartContainer start container with containerName
func StartContainer(containerName string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}
	for _, cont := range containers {
		isFound := false
		for _, name := range cont.Names {
			if strings.Contains(name, containerName) {
				isFound = true
				break
			}
		}
		if isFound {
			if err = cli.ContainerStart(ctx, cont.ID, types.ContainerStartOptions{}); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

//BuildContainer build container with tagName using dockerFile
func BuildContainer(dockerFile, tagName string) error {
	ctx := context.Background()
	dockerFileTarReader, err := archive.TarWithOptions(filepath.Dir(dockerFile), &archive.TarOptions{
		ExcludePatterns: nil,
		ChownOpts:       &idtools.Identity{UID: 0, GID: 0},
	})
	if err != nil {
		return err
	}
	buildArgs := make(map[string]*string)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	resp, err := cli.ImageBuild(
		ctx,
		dockerFileTarReader,
		types.ImageBuildOptions{
			Dockerfile: filepath.Base(dockerFile),
			Tags:       []string{tagName},
			NoCache:    false,
			Remove:     true,
			BuildArgs:  buildArgs,
		})
	if err != nil {
		return err
	}
	return writeToLog(resp.Body)
}

//writes from the build response to the log
func writeToLog(reader io.ReadCloser) error {
	defer reader.Close()
	rd := bufio.NewReader(reader)
	for {
		n, _, err := rd.ReadLine()
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		log.Debugln(string(n))
	}
	return nil
}

//DockerImageRepack export image to tar and repack it
func DockerImageRepack(commandPath string, distImage string, imageTag string) (err error) {
	distImageDir := filepath.Dir(distImage)
	if _, err := os.Stat(distImageDir); os.IsNotExist(err) {
		if err = os.MkdirAll(distImageDir, 0755); err != nil {
			return err
		}
	}
	commandArgsString := fmt.Sprintf("ociimage -i %s -o %s -l",
		imageTag, distImage)
	log.Debugf("DockerImageRepack run: %s %s", commandPath, commandArgsString)
	_, _, err = RunCommandAndWait(commandPath, strings.Fields(commandArgsString)...)
	return
}
