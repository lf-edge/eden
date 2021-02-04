package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileToSave provides pattern to save or load files based on Location and Destination inside fs and tar
type FileToSave struct {
	Location    string
	Destination string
}

// CreateTarGz generates tar.gz file in dstFile by putting files and directories described in paths
func CreateTarGz(dstFile string, paths []FileToSave) error {
	tarFile, err1 := os.Create(dstFile)
	if err1 != nil {
		return err1
	}
	defer tarFile.Close()
	gz := gzip.NewWriter(tarFile)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for _, path := range paths {
		walker := func(f string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}
			relFilePath := f
			if filepath.IsAbs(path.Location) {
				relFilePath, err = filepath.Rel(path.Location, f)
				if err != nil {
					return err
				}
			}
			hdr.Name = filepath.Join(path.Destination, relFilePath)
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if fi.Mode().IsDir() {
				return nil
			}
			srcFile, err := os.Open(f)
			if err != nil {
				return err
			}
			defer srcFile.Close()
			_, err = io.Copy(tw, srcFile)
			if err != nil {
				return err
			}
			return nil
		}
		if err := filepath.Walk(path.Location, walker); err != nil {
			fmt.Printf("failed to add %s to tar: %s\n", path.Location, err)
		}
	}
	return nil
}

func resolvePath(curPath string, paths []FileToSave) (string, error) {
	if paths == nil {
		return curPath, nil
	}
	for _, el := range paths {
		if strings.HasPrefix(curPath, el.Location) {
			return strings.Replace(curPath, el.Location, el.Destination, 1), nil
		}
	}
	return "", os.ErrNotExist
}

// UnpackTarGz observes tar.gz file in srcFile and extracts files and directories described in paths
func UnpackTarGz(srcFile string, paths []FileToSave) error {
	f, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer f.Close()
	gzf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzf)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		newPath, err := resolvePath(header.Name, paths)
		if err != nil {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if info, err := os.Stat(newPath); !os.IsNotExist(err) {
				if info.IsDir() {
					continue
				}
			}
			if err := os.Mkdir(newPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(newPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			outFile.Close()
		default:
			return err
		}

	}
	return nil
}

// Untar extracts files from srcFile tar into destination
func Untar(srcFile string, destination string) error {
	r, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer r.Close()
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}
		target := filepath.Join(destination, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}
}
