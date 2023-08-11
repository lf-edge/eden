package utils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	v1tarball "github.com/google/go-containerregistry/pkg/v1/tarball"
	"oras.land/oras-go/pkg/auth"
	"oras.land/oras-go/pkg/auth/docker"
)

// LoadRegistry push image into registry
func LoadRegistry(image, remote string) (string, error) {
	localImage, err := HasImage(image)
	if err != nil {
		return "", fmt.Errorf("error checking for local image %s: %v", image, err)
	}
	var (
		hash string
		img  v1.Image
	)
	ref, err := name.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("invalid image name %s: %v", image, err)
	}
	// we need a Tag, not just any Reference, so check if we had a valid one
	tag, err := name.NewTag(image)
	if err != nil {
		// it was a hash reference, so we need to get rid of the hash and try again
		in := strings.LastIndex(image, "@sha256")
		if in <= 0 {
			return "", fmt.Errorf("image was not a tag or hash reference, could not process: %s", image)
		}
		tag, err = name.NewTag(image[:in])
		if err != nil {
			return "", fmt.Errorf("could not process image %s: %v", image, err)
		}
	}
	destImage := fmt.Sprintf("%s/%s:%s", remote, tag.Context().RepositoryStr(), tag.Identifier())
	if localImage {
		// get the image, upload
		// we cannot simply use PushImage, because it requires setting TLS certs and security
		// at the *engine* level.
		// Also, we cannot use daemon.Image due to some version mismatches
		// Also `docker save` is not great, since it never stores the originals, but it is the best we have.
		tmpFileName := strings.ReplaceAll(image, ":", "_")
		tmpFileName = strings.ReplaceAll(tmpFileName, "/", "_")
		dir, err := os.MkdirTemp("", "edenSave")
		if err != nil {
			return "", fmt.Errorf("unable to create temporary dir: %v", err)
		}
		defer os.RemoveAll(dir)
		tmpFilePath := path.Join(dir, tmpFileName)
		if err := SaveImageToTar(image, tmpFilePath); err != nil {
			return "", fmt.Errorf("unable to save image file %s: %v", ref, err)
		}
		img, err = v1tarball.ImageFromPath(tmpFilePath, &tag)
		if err != nil {
			return "", fmt.Errorf("unable to get a v1.Image from the tarfile %s: %v", tmpFilePath, err)
		}
		// get each layer, and convert it to a proper layer to write
		if err := crane.Push(img, destImage); err != nil {
			return "", fmt.Errorf("error pushing to %s: %v", remote, err)
		}
	} else {
		if err := crane.Copy(image, destImage); err != nil {
			return "", fmt.Errorf("unable to copy from %s to %s: %v", image, destImage, err)
		}
	}
	img, err = crane.Pull(destImage)
	if err != nil {
		return "", fmt.Errorf("error pulling manifest for %s: %v", destImage, err)
	}
	manifest, err := img.RawManifest()
	if err != nil {
		return "", fmt.Errorf("could not get raw manifest")
	}
	hash = fmt.Sprintf("sha256:%x", sha256.Sum256(manifest))
	return hash, nil
}

// RegistryHTTP for http access to local registry
type RegistryHTTP struct {
	remotes.Resolver
	ctx context.Context
}

// NewRegistryHTTP creates new RegistryHTTP with plainHTTP resolver
func NewRegistryHTTP(ctx context.Context) (context.Context, *RegistryHTTP, error) {
	cli, err := docker.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get authenticating client to registry: %v", err)
	}
	resolver, err := cli.ResolverWithOpts(auth.WithResolverPlainHTTP())
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get resolver for registry: %v", err)
	}
	return ctx, &RegistryHTTP{Resolver: resolver, ctx: ctx}, nil
}

// Finalize wrapper
func (r *RegistryHTTP) Finalize(_ context.Context) error {
	return nil
}

// Context wrapper
func (r *RegistryHTTP) Context() context.Context {
	return r.ctx
}
