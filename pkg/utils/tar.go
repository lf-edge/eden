package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileToSave provides pattern to save or load files based on Location and Destination inside fs and tar
type FileToSave struct {
	Location    string
	Destination string
}

// MaxDecompressedContentSize is the maximum size of a file that can be written to disk after decompression.
// This is to prevent a DoS attack by unpacking a compressed file that is too big to be decompressed.
const MaxDecompressedContentSize = 1024 * 1024 * 1024 // 1 GB

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
		if errors.Is(err, io.EOF) {
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
			if err := os.Mkdir(newPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(newPath)
			if err != nil {
				return err
			}
			// Limit the size of the extracted file to prevent decompression bomb
			limitReader := io.LimitReader(tarReader, MaxDecompressedContentSize+1)
			bytesCopied, err := io.Copy(outFile, limitReader)
			if err != nil {
				return err
			}
			if bytesCopied > MaxDecompressedContentSize {
				return errors.New("maximum decompressed content size exceeded")
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
	return ExtractFromTar(r, destination)
}

// ExtractFromTar extracts files from a tar reader into the destination directory
func ExtractFromTar(u io.Reader, destination string) error {
	// path inside tar is relative
	pathBuilder := func(oldPath string) string {
		return path.Join(destination, oldPath)
	}
	tarReader := tar.NewReader(u)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("ExtractFromTar: Next() failed: %w", err)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(pathBuilder(header.Name), os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("ExtractFromTar: Mkdir() failed: %w", err)
			}
		case tar.TypeReg:
			if _, err := os.Lstat(pathBuilder(header.Name)); err == nil {
				err = os.Remove(pathBuilder(header.Name))
				if err != nil {
					return fmt.Errorf("ExtractFromTar: cannot remove old file: %w", err)
				}
			}
			outFile, err := os.OpenFile(pathBuilder(header.Name), os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("ExtractFromTar: OpenFile() failed: %w", err)
			}
			// Limit the size of the extracted file to prevent decompression bomb
			limitReader := io.LimitReader(tarReader, MaxDecompressedContentSize+1)
			bytesCopied, err := io.Copy(outFile, limitReader)
			if err != nil {
				return fmt.Errorf("ExtractFromTar: Copy() failed: %w", err)
			}
			if bytesCopied > MaxDecompressedContentSize {
				return fmt.Errorf("ExtractFromTar: Max decompressed content size reached")
			}
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("ExtractFromTar: outFile.Close() failed: %w", err)
			}
		case tar.TypeLink, tar.TypeSymlink:
			if _, err := os.Lstat(pathBuilder(header.Name)); err == nil {
				err = os.Remove(pathBuilder(header.Name))
				if err != nil {
					return fmt.Errorf("ExtractFromTar: cannot remove old symlink: %w", err)
				}
			}
			if err := os.Symlink(pathBuilder(header.Linkname), pathBuilder(header.Name)); err != nil {
				return fmt.Errorf("ExtractFromTar: Symlink(%s, %s) failed: %w",
					pathBuilder(header.Name), pathBuilder(header.Linkname), err)
			}
		default:
			return fmt.Errorf(
				"ExtractFromTar: unknown type: '%s' in %s",
				string([]byte{header.Typeflag}),
				header.Name)
		}
	}
	return nil
}
