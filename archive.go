package logfile

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// ExtractTarGz extracts files from a tar archive.
func ExtractTarGz(gzipStream io.Reader) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		h, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(h.Name, os.ModePerm); err != nil {
				return errors.Wrapf(err, "ExtractTarGz: MkdirAll() failed")
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(h.Name), os.ModePerm); err != nil {
				return errors.Wrapf(err, "ExtractTarGz: MkdirAll() failed")
			}

			if err := createFile(h.Name, tarReader); err != nil {
				return err
			}
		default:
			return errors.Errorf("ExtractTarGz: unknown type: %b in %s", h.Typeflag, h.Name)
		}
	}
}

func createFile(fn string, r *tar.Reader) error {
	outFile, err := os.Create(fn)
	if err != nil {
		return errors.Wrapf(err, "ExtractTarGz: Create()")
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, r); err != nil {
		return errors.Wrapf(err, "ExtractTarGz: Copy()")
	}

	return nil
}

// CreateTarGz creates an archive file with archiveName like a/b/c.tar.gz.
func CreateTarGz(archiveName string, files []string) error {
	// Create output file
	out, err := os.Create(archiveName)
	if err != nil {
		return err
	}

	defer out.Close()

	// Create new Writers for gzip and tar
	// These writers are chained. Writing to the tar writer will
	// write to the gzip writer which in turn will write to
	// the "buf" writer
	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Iterate over files and add them to the tar archive
	for _, file := range files {
		if err := addToArchive(tw, file); err != nil {
			return err
		}
	}

	return nil
}

func addToArchive(tw *tar.Writer, filename string) error {
	// Open the f which will be written into the archive
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Get FileInfo about our f providing f size, mode, etc.
	info, err := f.Stat()
	if err != nil {
		return err
	}

	// Create a tar Header from the FileInfo data
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory structure would not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	header.Name = filename

	// Write f header to the tar archive
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	// Copy f content to tar archive
	if _, err := io.Copy(tw, f); err != nil {
		return err
	}

	return nil
}
