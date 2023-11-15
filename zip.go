package xtractr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeka/zip"
)

// ExtractZIP extracts a zip file to a destination, supporting password-protected files if needed.
func ExtractZIP(xFile *XFile) (int64, []string, error) {
	// Open the zip file using yeka/zip
	zipReader, err := zip.OpenReader(xFile.FilePath)
	if err != nil {
		return 0, nil, fmt.Errorf("zip.OpenReader: %w", err)
	}
	defer zipReader.Close()

	files := []string{}
	size := int64(0)

	for _, zipFile := range zipReader.File {
		//过滤掉隐藏文件和目录,如: __MACOSX目录, .DS_Store
		if strings.Contains(zipFile.Name, "__MACOSX") || strings.HasPrefix(zipFile.Name, ".") {
			continue
		}
		fmt.Printf("Extracting %s\n", zipFile.Name)
		fSize, err := xFile.unzip(zipFile)
		if err != nil {
			return size, files, fmt.Errorf("%s: %w", xFile.FilePath, err)
		}

		files = append(files, filepath.Join(xFile.OutputDir, zipFile.Name))
		size += fSize
	}

	return size, files, nil
}

func (x *XFile) unzip(zipFile *zip.File) (int64, error) {
	wfile := x.clean(zipFile.Name)
	if !strings.HasPrefix(wfile, x.OutputDir) {
		return 0, fmt.Errorf("%s: %w: %s (from: %s)", zipFile.FileInfo().Name(), ErrInvalidPath, wfile, zipFile.Name)
	}

	if strings.HasSuffix(wfile, "/") || zipFile.FileInfo().IsDir() {
		if err := os.MkdirAll(wfile, x.DirMode); err != nil {
			return 0, fmt.Errorf("making zipFile dir: %w", err)
		}
		return 0, nil
	}
	fmt.Println("x.Password", x.Password)
	// Set the password for the file if needed
	if x.Password != "" && zipFile.IsEncrypted() {
		zipFile.SetPassword(x.Password)
	}
	zFile, err := zipFile.Open()
	if err != nil {
		return 0, fmt.Errorf("zipFile.Open: %w", err)
	}
	defer zFile.Close()

	s, err := writeFile(wfile, zFile, x.FileMode, x.DirMode)
	if err != nil {
		return s, fmt.Errorf("%s: %w: %s (from: %s)", zipFile.FileInfo().Name(), err, wfile, zipFile.Name)
	}

	return s, nil
}
