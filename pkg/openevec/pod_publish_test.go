// Copyright (c) 2026 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package openevec_test

// Characterization tests for "eden pod publish". They stand up an in-process OCI
// registry and assert the bytes PodPublish puts on the wire, so that a change to
// how the publish path is wired up -- a different resolver, a different oras
// version -- has to reproduce the same artifact to pass.
//
// The published config embeds time.Now(), so the manifest digest is not stable
// across runs and is not asserted; every other field is. Layer digests are
// checked against the source files, which pins the payload exactly.
//
// The plain-HTTP setting the local publish depends on is not observable here:
// containerd's resolver enables plain HTTP for any loopback host regardless of
// that setting, so a loopback registry accepts the push either way. The option
// is covered in edge-containers by TestRegistryOptResolverOptions.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	ggcr "github.com/google/go-containerregistry/pkg/registry"
	"github.com/lf-edge/eden/pkg/openevec"
	edgeRegistry "github.com/lf-edge/edge-containers/pkg/registry"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// newLocalRegistry starts an in-process OCI registry and returns an OpenEVEC
// pointed at it along with its host:port.
func newLocalRegistry(t *testing.T) (*openevec.OpenEVEC, string) {
	t.Helper()
	srv := httptest.NewServer(ggcr.New(ggcr.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	hostPort := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("splitting %s: %v", hostPort, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parsing port %s: %v", portStr, err)
	}
	return openevec.CreateOpenEVEC(&openevec.EdenSetupArgs{Registry: openevec.RegistryConfig{IP: host, Port: port}}), hostPort
}

// writeArtifactFile creates a file of the given size whose contents depend only
// on the size, so its digest is reproducible.
func writeArtifactFile(t *testing.T, dir, name string, size int) string {
	t.Helper()
	p := filepath.Join(dir, name)
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i % 251)
	}
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatalf("writing %s: %v", p, err)
	}
	return p
}

func fileDigest(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func get(t *testing.T, url string) []byte {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("building request for %s: %v", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading %s: %v", url, err)
	}
	return b
}

// fetchManifest returns the raw manifest bytes and the parsed manifest for repo:tag.
func fetchManifest(t *testing.T, hostPort, repo, tag string) ([]byte, ocispec.Manifest) {
	t.Helper()
	raw := get(t, fmt.Sprintf("http://%s/v2/%s/manifests/%s", hostPort, repo, tag))
	var m ocispec.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshalling manifest: %v", err)
	}
	return raw, m
}

func fetchBlob(t *testing.T, hostPort, repo, digest string) []byte {
	t.Helper()
	return get(t, fmt.Sprintf("http://%s/v2/%s/blobs/%s", hostPort, repo, digest))
}

// wantLayer is the expected descriptor for one published layer.
type wantLayer struct {
	mediaType string
	role      string
	title     string
	path      string
}

func checkLayers(t *testing.T, got []ocispec.Descriptor, want []wantLayer) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d layers, want %d", len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		if g.MediaType != w.mediaType {
			t.Errorf("layer %d media type = %q, want %q", i, g.MediaType, w.mediaType)
		}
		if d := fileDigest(t, w.path); string(g.Digest) != d {
			t.Errorf("layer %d digest = %s, want %s", i, g.Digest, d)
		}
		fi, err := os.Stat(w.path)
		if err != nil {
			t.Fatalf("stat %s: %v", w.path, err)
		}
		if g.Size != fi.Size() {
			t.Errorf("layer %d size = %d, want %d", i, g.Size, fi.Size())
		}
		if got, want := g.Annotations[edgeRegistry.AnnotationRole], w.role; got != want {
			t.Errorf("layer %d role = %q, want %q", i, got, want)
		}
		if got, want := g.Annotations[edgeRegistry.AnnotationMediaType], w.mediaType; got != want {
			t.Errorf("layer %d mediaType annotation = %q, want %q", i, got, want)
		}
		if got, want := g.Annotations[ocispec.AnnotationTitle], w.title; got != want {
			t.Errorf("layer %d title = %q, want %q", i, got, want)
		}
	}
}

func TestPodPublishArtifacts(t *testing.T) {
	evec, hostPort := newLocalRegistry(t)
	dir := t.TempDir()
	kernel := writeArtifactFile(t, dir, "kernel", 1024)
	initrd := writeArtifactFile(t, dir, "initrd", 2048)
	root := writeArtifactFile(t, dir, "root.qcow2", 4096)

	if err := evec.PodPublish("app:1", kernel, initrd, root+":qcow2", "artifacts", "amd64", true, nil); err != nil {
		t.Fatalf("PodPublish: %v", err)
	}

	raw, m := fetchManifest(t, hostPort, "app", "1")
	if m.MediaType != ocispec.MediaTypeImageManifest {
		t.Errorf("manifest media type = %q, want %q", m.MediaType, ocispec.MediaTypeImageManifest)
	}
	// a registry rejects a manifest that does not declare schema version 2
	if !strings.Contains(string(raw), `"schemaVersion":2`) {
		t.Errorf("manifest does not declare schemaVersion 2:\n%s", raw)
	}
	if m.Config.MediaType != ocispec.MediaTypeImageConfig {
		t.Errorf("config media type = %q, want %q", m.Config.MediaType, ocispec.MediaTypeImageConfig)
	}
	if got := m.Config.Annotations[ocispec.AnnotationTitle]; got != "config.json" {
		t.Errorf("config title = %q, want %q", got, "config.json")
	}

	checkLayers(t, m.Layers, []wantLayer{
		{edgeRegistry.MimeTypeECIKernel, edgeRegistry.RoleKernel, "kernel", kernel},
		{edgeRegistry.MimeTypeECIInitrd, edgeRegistry.RoleInitrd, "initrd", initrd},
		{edgeRegistry.MimeTypeECIDiskQcow2, edgeRegistry.RoleRootDisk, "disk-root-root.qcow2", root},
	})

	var cfg ocispec.Image
	if err := json.Unmarshal(fetchBlob(t, hostPort, "app", string(m.Config.Digest)), &cfg); err != nil {
		t.Fatalf("unmarshalling config: %v", err)
	}
	if cfg.Author != edgeRegistry.DefaultAuthor {
		t.Errorf("config author = %q, want %q", cfg.Author, edgeRegistry.DefaultAuthor)
	}
	if cfg.Architecture != "amd64" {
		t.Errorf("config architecture = %q, want %q", cfg.Architecture, "amd64")
	}
	if cfg.OS != runtime.GOOS {
		t.Errorf("config OS = %q, want %q", cfg.OS, runtime.GOOS)
	}
	if cfg.RootFS.Type != "layers" {
		t.Errorf("config rootfs type = %q, want %q", cfg.RootFS.Type, "layers")
	}
	if len(cfg.RootFS.DiffIDs) != len(m.Layers) {
		t.Errorf("config has %d diffIDs, want %d", len(cfg.RootFS.DiffIDs), len(m.Layers))
	}
}

func TestPodPublishLegacy(t *testing.T) {
	evec, hostPort := newLocalRegistry(t)
	dir := t.TempDir()
	root := writeArtifactFile(t, dir, "root.raw", 4096)

	if err := evec.PodPublish("legacy:1", "", "", root+":raw", "legacy", "amd64", true, nil); err != nil {
		t.Fatalf("PodPublish: %v", err)
	}

	_, m := fetchManifest(t, hostPort, "legacy", "1")
	if m.Config.MediaType != ocispec.MediaTypeImageConfig {
		t.Errorf("config media type = %q, want %q", m.Config.MediaType, ocispec.MediaTypeImageConfig)
	}
	if len(m.Layers) != 1 {
		t.Fatalf("got %d layers, want 1", len(m.Layers))
	}
	// legacy wraps each file in a gzipped tar, so the layer no longer matches the
	// source file and carries the gzip media type.
	if m.Layers[0].MediaType != ocispec.MediaTypeImageLayerGzip {
		t.Errorf("layer media type = %q, want %q", m.Layers[0].MediaType, ocispec.MediaTypeImageLayerGzip)
	}
	if got := m.Layers[0].Annotations[edgeRegistry.AnnotationRole]; got != edgeRegistry.RoleRootDisk {
		t.Errorf("layer role = %q, want %q", got, edgeRegistry.RoleRootDisk)
	}
	if got := m.Layers[0].Annotations[edgeRegistry.AnnotationMediaType]; got != edgeRegistry.MimeTypeECIDiskRaw {
		t.Errorf("layer mediaType annotation = %q, want %q", got, edgeRegistry.MimeTypeECIDiskRaw)
	}
}

func TestPodPublishDiskTypes(t *testing.T) {
	for name, mime := range map[string]string{
		"raw":   edgeRegistry.MimeTypeECIDiskRaw,
		"vmdk":  edgeRegistry.MimeTypeECIDiskVmdk,
		"vhd":   edgeRegistry.MimeTypeECIDiskVhd,
		"iso":   edgeRegistry.MimeTypeECIDiskISO,
		"qcow":  edgeRegistry.MimeTypeECIDiskQcow,
		"qcow2": edgeRegistry.MimeTypeECIDiskQcow2,
		"ova":   edgeRegistry.MimeTypeECIDiskOva,
		"vhdx":  edgeRegistry.MimeTypeECIDiskVhdx,
	} {
		t.Run(name, func(t *testing.T) {
			evec, hostPort := newLocalRegistry(t)
			dir := t.TempDir()
			root := writeArtifactFile(t, dir, "root."+name, 512)

			if err := evec.PodPublish("disk:1", "", "", root+":"+name, "artifacts", "amd64", true, nil); err != nil {
				t.Fatalf("PodPublish: %v", err)
			}
			_, m := fetchManifest(t, hostPort, "disk", "1")
			checkLayers(t, m.Layers, []wantLayer{
				{mime, edgeRegistry.RoleRootDisk, "disk-root-root." + name, root},
			})
		})
	}
}

func TestPodPublishComponents(t *testing.T) {
	dir := t.TempDir()
	kernel := writeArtifactFile(t, dir, "kernel", 128)
	initrd := writeArtifactFile(t, dir, "initrd", 256)
	root := writeArtifactFile(t, dir, "root.raw", 512)
	extra := writeArtifactFile(t, dir, "extra.qcow2", 1024)

	tests := []struct {
		name           string
		kernel, initrd string
		root           string
		disks          []string
		want           []wantLayer
	}{
		{
			name:   "kernel only",
			kernel: kernel,
			want: []wantLayer{
				{edgeRegistry.MimeTypeECIKernel, edgeRegistry.RoleKernel, "kernel", kernel},
			},
		},
		{
			name: "root only",
			root: root + ":raw",
			want: []wantLayer{
				{edgeRegistry.MimeTypeECIDiskRaw, edgeRegistry.RoleRootDisk, "disk-root-root.raw", root},
			},
		},
		{
			name:   "kernel and initrd",
			kernel: kernel,
			initrd: initrd,
			want: []wantLayer{
				{edgeRegistry.MimeTypeECIKernel, edgeRegistry.RoleKernel, "kernel", kernel},
				{edgeRegistry.MimeTypeECIInitrd, edgeRegistry.RoleInitrd, "initrd", initrd},
			},
		},
		{
			name:  "root and additional disk",
			root:  root + ":raw",
			disks: []string{extra + ":qcow2"},
			want: []wantLayer{
				{edgeRegistry.MimeTypeECIDiskRaw, edgeRegistry.RoleRootDisk, "disk-root-root.raw", root},
				{edgeRegistry.MimeTypeECIDiskQcow2, edgeRegistry.RoleAdditionalDisk, "disk-0-extra.qcow2", extra},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evec, hostPort := newLocalRegistry(t)
			if err := evec.PodPublish("parts:1", tt.kernel, tt.initrd, tt.root, "artifacts", "amd64", true, tt.disks); err != nil {
				t.Fatalf("PodPublish: %v", err)
			}
			_, m := fetchManifest(t, hostPort, "parts", "1")
			checkLayers(t, m.Layers, tt.want)
		})
	}
}

func TestPodPublishRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	root := writeArtifactFile(t, dir, "root.raw", 128)

	tests := []struct {
		name   string
		root   string
		format string
		errStr string
	}{
		{"unknown format", root + ":raw", "tarball", "unknown format"},
		{"disk without type", root, "artifacts", "expected structure <path>:<type>"},
		{"unknown disk type", root + ":floppy", "artifacts", "unknown disk type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evec, _ := newLocalRegistry(t)
			err := evec.PodPublish("bad:1", "", "", tt.root, tt.format, "amd64", true, nil)
			if err == nil {
				t.Fatalf("expected an error containing %q", tt.errStr)
			}
			if !strings.Contains(err.Error(), tt.errStr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.errStr)
			}
		})
	}
}
