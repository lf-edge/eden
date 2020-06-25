package utils

import (
	"archive/tar"
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/go-connections/nat"
	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

//CreateDockerNetwork create network for docker`s containers
func CreateDockerNetwork(name string) error {
	log.Debugf("Try to create network %s", name)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("NewClientWithOpts: %s", err)
	}
	//check existing networks
	result, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("NetworkListOptions: %s", err)
	}
	for _, el := range result {
		if el.Name == name {
			return nil
		}
	}
	_, err = cli.NetworkCreate(ctx, name, types.NetworkCreate{})
	return err
}

//CreateAndRunContainer run container with defined name from image with port and volume mapping and defined command
func CreateAndRunContainer(containerName string, imageName string, portMap map[string]string, volumeMap map[string]string, command []string) error {

	log.Debugf("Try to start container from image %s with command %s", imageName, command)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
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
	}
	var mounts []mount.Mount
	for target, source := range volumeMap {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: source,
			Target: target,
		})
	}
	if err = CreateDockerNetwork(defaults.DefaultDockerNetworkName); err != nil {
		return err
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBinding,
		Mounts:       mounts,
		DNS:          []string{},
		DNSOptions:   []string{},
		DNSSearch:    []string{},
		NetworkMode:  container.NetworkMode(defaults.DefaultDockerNetworkName),
	}
	userCurrent, err := user.Current()
	if err != nil {
		return err
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Hostname:     containerName,
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

//GetDockerNetworks returns gateways IPs of networks in docker
func GetDockerNetworks() ([]*net.IPNet, error) {
	var results []*net.IPNet
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("client.NewClientWithOpts: %s", err)
	}
	networkTypes := types.NetworkListOptions{}
	resp, err := cli.NetworkList(ctx, networkTypes)
	if err != nil {
		return nil, fmt.Errorf("GetNetworks: %s", err)
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

//PullImage from docker
func PullImage(image string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("client.NewClientWithOpts: %s", err)
	}
	_, _, err = cli.ImageInspectWithRaw(ctx, image)
	if err == nil { //local image is ok
		return nil
	}
	resp, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("imagePull: %s", err)
	}
	if err = writeToLog(resp); err != nil {
		return fmt.Errorf("imagePull LOG: %s", err)
	}
	return nil
}

//GenEVEImage from docker to outputFile only with defined configDir
func GenEVEImage(image, outputDir, command, format string, configDir string, size int) (fileName string, err error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}
	volumeMap := map[string]string{"/in": configDir, "/out": outputDir}
	dockerCommand := fmt.Sprintf("-f %s %s %d", format, command, size)
	if size == 0 {
		dockerCommand = fmt.Sprintf("-f %s %s", format, command)
	}
	u, err := RunDockerCommand(image, dockerCommand, volumeMap)
	if err != nil {
		log.Printf("error GenEVEImage: %v", err)
		return "", err
	}
	log.Debug(u)
	fileName = filepath.Join(outputDir, fmt.Sprintf("%s.raw", command))
	if format == "qcow2" {
		fileName = fileName + "." + format
	}
	return fileName, nil
}

//SaveImage from docker to outputDir only for path defaultEvePrefixInTar in docker rootfs
func SaveImage(image, outputDir, defaultEvePrefixInTar string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("client.NewClientWithOpts: %s", err)
	}
	reader, err := cli.ImageSave(ctx, []string{image})
	if err != nil {
		return err
	}
	defer reader.Close()
	if err = ExtractFilesFromDocker(reader, outputDir, defaultEvePrefixInTar); err != nil {
		return err
	}
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
			return fmt.Sprintf("container with name %s is %s", strings.TrimLeft(cont.Names[0], "/"), cont.State), nil
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

//writeToLog from the build response to the log
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
		log.Infoln(string(n))
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
	log.Infof("DockerImageRepack run: %s %s", commandPath, commandArgsString)
	return RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//ExtractFilesFromDocker extract all files from docker layer into directory
//if prefixDirectory is not empty, remove it from path
func ExtractFilesFromDocker(u io.ReadCloser, directory string, prefixDirectory string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("ExtractFilesFromDocker: MkdirAll() failed: %s", err.Error())
	}
	tarReader := tar.NewReader(u)
	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ExtractFilesFromDocker: Next() failed: %s", err.Error())
		}
		switch header.Typeflag {
		case tar.TypeReg:
			if strings.TrimSpace(filepath.Ext(header.Name)) == ".tar" {
				log.Infof("Extract layer %s", header.Name)
				if err = extractLayersFromDocker(tarReader, directory, prefixDirectory); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

//if prefixDirectory is empty, extract all
func extractLayersFromDocker(u io.Reader, directory string, prefixDirectory string) error {
	pathBuilder := func(oldPath string) string {
		return path.Join(directory, strings.TrimPrefix(oldPath, prefixDirectory))
	}
	tarReader := tar.NewReader(u)
	for true {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ExtractFilesFromDocker: Next() failed: %s", err.Error())
		}
		//extract only from directory of interest
		if prefixDirectory != "" && strings.TrimLeft(header.Name, prefixDirectory) == header.Name {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(pathBuilder(header.Name), 0755); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			if _, err := os.Lstat(pathBuilder(header.Name)); err == nil {
				err = os.Remove(pathBuilder(header.Name))
				if err != nil {
					return fmt.Errorf("ExtractFilesFromDocker: cannot remove old file: %s", err.Error())
				}
			}
			outFile, err := os.Create(pathBuilder(header.Name))
			if err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Create() failed: %s", err.Error())
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Copy() failed: %s", err.Error())
			}
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: outFile.Close() failed: %s", err.Error())
			}
		case tar.TypeSymlink:
			if _, err := os.Lstat(pathBuilder(header.Name)); err == nil {
				err = os.Remove(pathBuilder(header.Name))
				if err != nil {
					return fmt.Errorf("ExtractFilesFromDocker: cannot remove old symlink: %s", err.Error())
				}
			}
			if err := os.Symlink(pathBuilder(header.Linkname), pathBuilder(header.Name)); err != nil {
				return fmt.Errorf("ExtractFilesFromDocker: Symlink(%s, %s) failed: %s",
					pathBuilder(header.Name), pathBuilder(header.Linkname), err.Error())
			}
		default:
			return fmt.Errorf(
				"ExtractFilesFromDocker: uknown type: '%s' in %s",
				string([]byte{header.Typeflag}),
				header.Name)
		}
	}
	return nil
}

// getFileReaderInTar scan a tar stream to get a header and reader for the
// file data that matches the provided regexp
func getFileReaderInTar(f io.ReadSeeker, re *regexp.Regexp) (*tar.Header, io.Reader, error) {
	var (
		err error
		hdr *tar.Header
	)
	if _, err = f.Seek(0, 0); err != nil {
		return nil, nil, fmt.Errorf("unable to reset tar file reader %s: %v", re, err)
	}

	// get a new reader
	tr := tar.NewReader(f)

	// go through each file in the archive, looking for the file we want
	for {
		if hdr, err = tr.Next(); err != nil {
			if err == io.EOF {
				return nil, nil, fmt.Errorf("could not find file matching %s in tar stream: %v", re, err)
			}
			return nil, nil, fmt.Errorf("error reading information from tar stream: %v", err)
		}
		// if the name matches the format of the requested file, use it
		if re.MatchString(hdr.Name) {
			return hdr, tr, nil
		}
	}
}

// readFromTar given an io.ReadSeeker and a filename, get the contents of the
// file from the ReadSeeker
func readFromTar(f io.ReadSeeker, re *regexp.Regexp) ([]byte, error) {
	hdr, reader, err := getFileReaderInTar(f, re)
	if err != nil {
		return nil, err
	}
	// read the data
	b := make([]byte, hdr.Size)
	read, err := reader.Read(b)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading %s from tarfile: %v", re, err)
	}
	if read != len(b) {
		return nil, fmt.Errorf("file %s had mismatched size to tar header, expected %d, actual %d", re, len(b), read)
	}
	return b, nil
}

//ComputeShaOCITar compute the sha for an OCI tar file with the manifest inside
func ComputeShaOCITar(filename string) (string, error) {
	// extract the image manifest itself, and calculate its hash
	// then check the hash of each layer and the config against the information in the manifest
	// if all adds up, return the hash of the manifest, nil error

	var (
		f               *os.File
		err             error
		re              *regexp.Regexp
		manifestB, hash []byte
	)

	if re, err = regexp.Compile(`imagemanifest-([a-f0-9]+).json`); err != nil {
		return "", fmt.Errorf("unable to compile regexp to find imagemanifest: %v", err)
	}

	// open our tar file
	if f, err = os.Open(filename); err != nil {
		return "", err
	}
	defer f.Close()

	manifestB, err = readFromTar(f, re)
	if err != nil {
		return "", fmt.Errorf("error reading image manifest %s from tar %s: %v", re, filename, err)
	}
	hashArray := sha256.Sum256(manifestB)
	hash = hashArray[:]
	return fmt.Sprintf("%x", hash), nil
}

//RunDockerCommand is run wrapper for docker container
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
	var mounts []mount.Mount
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
		Mounts: mounts},
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

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", err
	}
	if b, err := ioutil.ReadAll(out); err == nil {
		return string(b), nil
	} else {
		return "", err
	}
}
