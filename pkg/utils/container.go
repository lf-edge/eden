package utils

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
)

// CreateDockerNetwork create network for docker`s containers
func CreateDockerNetwork(name string, enableIPv6 bool, ipv6Subnet string) error {
	log.Debugf("Try to create network %s", name)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("NewClientWithOpts: %w", err)
	}
	// check existing networks
	result, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("NetworkListOptions: %w", err)
	}
	for _, el := range result {
		if el.Name == name {
			var ipv6SubnetFound bool
			if enableIPv6 {
				for _, ipam := range el.IPAM.Config {
					if ipam.Subnet == ipv6Subnet {
						ipv6SubnetFound = true
					}
				}
			}
			obsoleteConfig := (el.EnableIPv6 != enableIPv6) || (enableIPv6 && !ipv6SubnetFound)
			if obsoleteConfig {
				if err := cli.NetworkRemove(ctx, el.ID); err != nil {
					return fmt.Errorf("failed to remove docker network %s "+
						"with obsolete IP settings: %w", name, err)
				}
			} else {
				return nil
			}
		}
	}
	if enableIPv6 {
		_, err = cli.NetworkCreate(ctx, name, types.NetworkCreate{
			EnableIPv6: true,
			IPAM: &network.IPAM{
				Driver: "default",
				Config: []network.IPAMConfig{
					{
						Subnet: ipv6Subnet,
					},
				},
			},
		})
	} else {
		_, err = cli.NetworkCreate(ctx, name, types.NetworkCreate{})
	}
	return err
}

func dockerVolumeName(containerName string) string {
	return fmt.Sprintf("%s_volume", containerName)
}

// RemoveGeneratedVolumeOfContainer remove volumes created by eden
func RemoveGeneratedVolumeOfContainer(containerName string) error {
	volumeName := dockerVolumeName(containerName)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("NewClientWithOpts: %w", err)
	}
	return cli.VolumeRemove(ctx, volumeName, true)
}

// CreateAndRunContainer run container with defined name from image with port and volume mapping and defined command
func CreateAndRunContainer(containerName string, imageName string, portMap map[string]string,
	volumeMap map[string]string, command []string, envs []string, enableIPv6 bool, ipv6Subnet string) error {
	log.Debugf("Try to start container from image %s with command %s", imageName, command)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("NewClientWithOpts: %w", err)
	}
	if err = PullImage(imageName); err != nil {
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
		if enableIPv6 {
			portBinding[port] = append(portBinding[port], nat.PortBinding{
				HostIP:   "::",
				HostPort: binding,
			})
		}
	}
	var mounts []mount.Mount
	userCurrent, err := user.Current()
	if err != nil {
		return err
	}
	user := fmt.Sprintf("%s:%s", userCurrent.Uid, userCurrent.Gid)
	for target, source := range volumeMap {
		if source != "" {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: source,
				Target: target,
			})
		} else {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeVolume,
				Source: dockerVolumeName(containerName),
				Target: target,
			})
			user = "" // non-root user required only for TypeBind for delete
		}
	}
	if err = CreateDockerNetwork(
		defaults.DefaultDockerNetworkName, enableIPv6, ipv6Subnet); err != nil {
		return fmt.Errorf("CreateDockerNetwork: %w", err)
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBinding,
		Mounts:       mounts,
		DNS:          []string{},
		DNSOptions:   []string{},
		DNSSearch:    []string{},
		NetworkMode:  container.NetworkMode(defaults.DefaultDockerNetworkName),
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Hostname:     containerName,
		Image:        imageName,
		Cmd:          command,
		ExposedPorts: portExposed,
		User:         user,
		Env:          envs,
	}, hostConfig, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("ContainerCreate: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("ContainerStart: %w", err)
	}

	log.Infof("started container: %s", resp.ID)
	return nil
}

// GetDockerNetworks returns gateways IPs of networks in docker
func GetDockerNetworks() ([]*net.IPNet, error) {
	var results []*net.IPNet
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("client.NewClientWithOpts: %w", err)
	}
	networkTypes := types.NetworkListOptions{}
	resp, err := cli.NetworkList(ctx, networkTypes)
	if err != nil {
		return nil, fmt.Errorf("GetNetworks: %w", err)
	}
	for _, el := range resp {
		for _, ipam := range el.IPAM.Config {
			_, ipnet, err := net.ParseCIDR(ipam.Subnet)
			if err == nil {
				results = append(results, ipnet)
			}
		}
	}
	return results, nil
}

// PullImage from docker
func PullImage(image string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("client.NewClientWithOpts: %w", err)
	}
	_, _, err = cli.ImageInspectWithRaw(ctx, image)
	if err == nil { // local image is ok
		return nil
	}
	resp, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("imagePull: %w", err)
	}
	if err = writeToLog(resp); err != nil {
		return fmt.Errorf("imagePull LOG: %w", err)
	}
	return nil
}

// HasImage see if the image is local
func HasImage(image string) (bool, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, fmt.Errorf("client.NewClientWithOpts: %w", err)
	}
	_, _, err = cli.ImageInspectWithRaw(ctx, image)
	if err == nil { // has the image
		return true, nil
	}
	// theoretically, this should distinguishe
	return false, nil
}

// CreateImage create new image from directory with tag
// If Dockerfile is inside the directory will use it
// otherwise will create image from scratch
func CreateImage(dir, tag, platform string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("client.NewClientWithOpts: %w", err)
	}
	dockerFile := filepath.Join(dir, "Dockerfile")
	if _, err := os.Stat(dockerFile); os.IsNotExist(err) {
		// write simple Dockerfile if not exists
		defer os.Remove(dockerFile)
		if err := os.WriteFile(dockerFile, []byte("FROM scratch\nCOPY . /\n"), 0777); err != nil {
			return err
		}
	}
	reader, err := archive.TarWithOptions(dir, &archive.TarOptions{})
	if err != nil {
		return err
	}

	imageBuildResponse, err := cli.ImageBuild(ctx, reader, types.ImageBuildOptions{
		Tags:     []string{tag},
		Platform: platform,
	})
	if err != nil {
		return err
	}
	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	return err
}

// TagImage set new tag to image
func TagImage(oldTag, newTag string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("client.NewClientWithOpts: %w", err)
	}
	if err := cli.ImageTag(ctx, oldTag, newTag); err != nil {
		return fmt.Errorf("unable to tag %s to %s", oldTag, newTag)
	}
	return nil
}

// PushImage from docker while optionally changing to a different remote registry
func PushImage(image, remote string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("client.NewClientWithOpts: %w", err)
	}
	remoteName := image
	if remote != "" {
		ref, err := name.ParseReference(image)
		if err != nil {
			return fmt.Errorf("error parsing name %s: %w", image, err)
		}
		remoteName = strings.Replace(ref.Name(), ref.Context().Registry.Name(), remote, 1)
	}
	if err := cli.ImageTag(ctx, image, remoteName); err != nil {
		return fmt.Errorf("unable to tag %s to %s", image, remoteName)
	}
	resp, err := cli.ImagePush(ctx, remoteName, types.ImagePushOptions{})
	if err != nil {
		return fmt.Errorf("imagePush: %w", err)
	}
	if err = writeToLog(resp); err != nil {
		return fmt.Errorf("imagePush LOG: %w", err)
	}
	return nil
}

// SaveImage get a reader to save an image
func SaveImage(image string) (io.ReadCloser, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("client.NewClientWithOpts: %w", err)
	}
	reader, err := cli.ImageSave(ctx, []string{image})
	if err != nil {
		return nil, err
	}
	return reader, err
}

// ExtractFromImage creates a container from an image, copies a file or directory from it, and then removes the container.
func ExtractFromImage(imageName, localPath, containerPath string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("client.NewClientWithOpts: %w", err)
	}

	// Create a temporary container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
	}, nil, nil, nil, "")
	if err != nil {
		return fmt.Errorf("error creating container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		if err := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			log.Errorf("ContainerRemove error: %s", err)
		}
	}()

	// Open a TAR-reader containing the copied file / directory from the container
	reader, _, err := cli.CopyFromContainer(ctx, containerID, containerPath)
	if err != nil {
		return fmt.Errorf("error copying from container: %w", err)
	}
	defer reader.Close()

	return ExtractFromTar(reader, localPath)
}

// SaveImageToTar creates tar from image
func SaveImageToTar(image, tarFile string) error {
	reader, err := SaveImage(image)
	if err != nil {
		return err
	}
	defer reader.Close()

	f, err := os.Create(tarFile)
	if err != nil {
		return fmt.Errorf("unable to create tar file %s: %w", tarFile, err)
	}
	defer f.Close()
	_, _ = io.Copy(f, reader)
	return nil
}

// StopContainer stop container and remove if remove is true
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
				timeout := 10
				//if err = cli.ContainerStop(ctx, cont.ID, &timeout); err != nil {
				if err = cli.ContainerStop(ctx, cont.ID, container.StopOptions{
					Timeout: &timeout,
				}); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return fmt.Errorf("container not found")
}

// StateContainer return state of container if found or "" state if not found
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
			return fmt.Sprintf("container with name %s is %s", strings.TrimLeft(cont.Names[0], "/"), cont.State), nil
		}
	}
	return "", nil
}

// StartContainer start container with containerName
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

// writeToLog from the build response to the log
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
		fmt.Println(string(n))
	}
	return nil
}

// RunDockerCommand is run wrapper for docker container
func RunDockerCommand(image string, command string, volumeMap map[string]string) (result string, err error) {
	log.Debugf("Try to call 'docker run %s %s' with volumes %s", image, command, volumeMap)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	if err := PullImage(image); err != nil {
		return "", err
	}
	mounts := make([]mount.Mount, 0, len(volumeMap)) // Preallocate the mounts slice with a capacity equal to the length of volumeMap
	for target, source := range volumeMap {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: source,
			Target: target,
		})
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image,
		Cmd:   strings.Fields(command),
		Tty:   true,
	}, &container.HostConfig{
		Mounts: mounts,
	},
		nil,
		nil,
		"")
	if err != nil {
		return "", err
	}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case <-statusCh:

	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", err
	}
	defer out.Close()
	b, err := io.ReadAll(out)

	if err := cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{RemoveVolumes: true}); err != nil {
		log.Errorf("ContainerRemove error: %s", err)
	}

	return string(b), err
}
