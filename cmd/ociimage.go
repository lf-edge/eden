package cmd

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/context"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"strings"
)

const (
	defaultFileToSave = "./test.tar"
	defaultImage      = "library/alpine"
	defaultRegistry   = "docker.io"
	defaultIsLocal    = false
)

var (
	fileToSave string
	image      string
	registry   string
	isLocal    bool
)

var ociImageCmd = &cobra.Command{
	Use:   "ociimage",
	Short: "do oci image manipulations",
	Long:  `Do oci image manipulations.`,
	Run: func(cmd *cobra.Command, args []string) {
		var imageManifest []byte
		var err error
		ref, err := name.ParseReference(image)
		if err != nil {
			log.Fatalf("parsing reference %q: %v", image, err)
		}
		var img v1.Image
		if !isLocal {
			desc, err := remote.Get(ref)
			if err != nil {
				log.Fatal(err)
			}
			img, err = desc.Image()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			ctx := context.Background()
			cli, err := client.NewClientWithOpts(client.FromEnv)
			if err != nil {
				log.Fatal(err)
			}
			cli.NegotiateAPIVersion(ctx)
			options := daemon.WithClient(cli)
			img, err = daemon.Image(ref, options)
			if err != nil {
				log.Fatal(err)
			}
		}
		imageManifest, err = img.RawManifest()
		if err != nil {
			log.Fatal(err)
		}
		err = tarball.WriteToFile(fileToSave, ref, img)
		if err != nil {
			log.Fatal(err)
		}

		// add the imageManifest to the tar file
		err = appendImageManifest(fileToSave, imageManifest)
		if err != nil {
			log.Fatalf("unable to append image manifest to tar at %s: %v", fileToSave, err)
		}
		err = appendImageRepositories(fileToSave, registry, image, imageManifest)
		if err != nil {
			log.Fatalf("unable to append image manifest to tar at %s: %v", fileToSave, err)
		}
	},
}

func ociImageInit() {
	ociImageCmd.Flags().StringVarP(&fileToSave, "output", "o", defaultFileToSave, "file to save")
	ociImageCmd.Flags().StringVarP(&image, "image", "i", defaultImage, "image to save")
	ociImageCmd.Flags().StringVarP(&registry, "registry", "r", defaultRegistry, "registry")
	ociImageCmd.Flags().BoolVarP(&isLocal, "local", "l", defaultIsLocal, "use local docker image")
}

// appendImageManifest add the given manifest to the given tar file. Opinionated
// about the name of the file to "imagemanifest-<hash>.json"
func appendImageManifest(tarFile string, manifest []byte) error {
	hash := sha256.Sum256(manifest)
	return appendToTarFile(tarFile, fmt.Sprintf("%s-%x.json", "imagemanifest", hash), manifest)
}

// appendToTarFile add the given bytes to the tar file with the given filename
func appendToTarFile(tarFile, filename string, content []byte) error {
	var (
		f   *os.File
		err error
	)
	// open the existing file
	if f, err = os.OpenFile(tarFile, os.O_RDWR, os.ModePerm); err != nil {
		return err
	}
	defer f.Close()
	// there always is padding at the end of a tar archive, so skip to the end
	// of the actual archive, so it will be read
	if _, err = f.Seek(-2<<9, io.SeekEnd); err != nil {
		return err
	}

	tw := tar.NewWriter(f)

	hdr := &tar.Header{
		Name: filename,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write %s tar header: %v", filename, err)
	}

	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("failed to write %s tar body: %v", filename, err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %v", err)
	}
	return nil
}
func appendImageRepositories(tarFile, registry, path string, imageManifest []byte) error {
	// get the top layer for the manifest bytes
	layerHash, err := DockerHashFromManifest(imageManifest)
	if err != nil {
		return fmt.Errorf("unable to get top layer hash: %v", err)
	}
	// need to take out the tag
	parts := strings.Split(path, ":")
	var tag, repo string
	switch len(parts) {
	case 0:
		return fmt.Errorf("malformed repository path %s", path)
	case 1:
		repo = parts[0]
		tag = "latest"
	case 2:
		repo = parts[0]
		tag = parts[1]
	default:
		return fmt.Errorf("malformed repository path has too many ':' %s", path)
	}
	fullRepo := fmt.Sprintf("%s/%s", registry, repo)
	// now build the tag we are after
	var data = map[string]map[string]string{}
	data[fullRepo] = map[string]string{}
	data[fullRepo][tag] = layerHash

	j, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("unable to convert repositories data to json: %v", err)
	}

	return appendToTarFile(tarFile, "repositories", j)
}

// LayersFromManifest get the descriptors for layers from a raw image manifest
func LayersFromManifest(imageManifest []byte) ([]v1.Descriptor, error) {
	manifest, err := v1.ParseManifest(bytes.NewReader(imageManifest))
	if err != nil {
		return nil, fmt.Errorf("unable to parse manifest: %v", err)
	}
	return manifest.Layers, nil
}

// DockerHashFromManifest get the sha256 hash as a string from a raw image
// manifest. The "docker hash" is what is used for the image, i.e. the topmost
// layer.
func DockerHashFromManifest(imageManifest []byte) (string, error) {
	layers, err := LayersFromManifest(imageManifest)
	if err != nil {
		return "", fmt.Errorf("unable to get layers: %v", err)
	}
	if len(layers) < 1 {
		return "", fmt.Errorf("no layers found")
	}
	return layers[len(layers)-1].Digest.Hex, nil
}
