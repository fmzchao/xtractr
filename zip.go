package xtractr

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/yeka/zip"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

/* How to extract a ZIP file. */

// ExtractZIP extracts a zip file.. to a destination. Simple enough.
func ExtractZIP(xFile *XFile) (int64, []string, error) {
	zipReader, err := zip.OpenReader(xFile.FilePath)
	if err != nil {
		return 0, nil, fmt.Errorf("zip.OpenReader: %w", err)
	}
	defer zipReader.Close()

	files := []string{}
	size := int64(0)

	for _, zipFile := range zipReader.Reader.File {
		fSize, err := xFile.unzip(zipFile)
		if err != nil {
			return size, files, fmt.Errorf("%s: %w", xFile.FilePath, err)
		}

		files = append(files, filepath.Join(xFile.OutputDir, zipFile.Name)) //nolint: gosec
		size += fSize
	}

	return size, files, nil
}

func (x *XFile) unzip(zipFile *zip.File) (int64, error) { //nolint:dupl
	wfile := x.clean(zipFile.Name)
	if !strings.HasPrefix(wfile, x.OutputDir) {
		// The file being written is trying to write outside of our base path. Malicious archive?
		return 0, fmt.Errorf("%s: %w: %s (from: %s)", zipFile.FileInfo().Name(), ErrInvalidPath, wfile, zipFile.Name)
	}

	if strings.HasSuffix(wfile, "/") || zipFile.FileInfo().IsDir() {
		if err := os.MkdirAll(wfile, x.DirMode); err != nil {
			return 0, fmt.Errorf("making zipFile dir: %w", err)
		}

		return 0, nil
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
func ExtractZipWithPassword(xFile *XFile) (int64, []string, error) {
	// 打开ZIP文件
	zipReader, err := zip.OpenReader(xFile.FilePath)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to open zip file: %v", err)
	}
	defer zipReader.Close()
	//跳过zip为空的文件
	if len(zipReader.File) == 0 {
		return 0, nil, fmt.Errorf("zip file is empty")
	}
	files := []string{}
	size := int64(0)
	// 遍历ZIP文件中的每个文件/目录
	for _, f := range zipReader.File {
		if f.IsEncrypted() && xFile.Password == "" {
			return 0, nil, fmt.Errorf("zip file is encrypted, please set password")
		}
		// 为这个文件/目录设置密码
		if xFile.Password != "" && f.IsEncrypted() {
			f.SetPassword(xFile.Password)
		}
		//跳过隐藏文件和隐藏文件夹
		if strings.HasPrefix(f.Name, ".") || strings.Contains(f.Name, "__MACOSX") {
			continue
		}
		cleanedName := cleanFileName(f.Name)
		fpath := filepath.Join(xFile.OutputDir, cleanedName)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return 0, nil, fmt.Errorf("failed to create directory: %v", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			if strings.Contains(err.Error(), "illegal byte sequence") {
				// 记录错误，跳过当前文件
				log.Printf("Warning: Skipping file %s due to illegal byte sequence error: %v\n", f.Name, err)
				continue
			} else if strings.Contains(err.Error(), "is a directory") {
				// 记录错误，跳过当前文件
				log.Printf("Warning: Skipping file %s due to is a directory error: %v\n", f.Name, err)
				continue
			}
			return 0, nil, fmt.Errorf("failed to open file: %v", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return 0, nil, fmt.Errorf("failed to open file inside zip: %v", err)
		}

		s, err := io.Copy(outFile, rc)
		size += s
		files = append(files, filepath.Join(xFile.OutputDir, f.Name))
		outFile.Close()
		rc.Close()

		if err != nil {
			return 0, nil, fmt.Errorf("password error: %v", err)
		}
	}

	return size, files, nil
}

// cleanFileName 清理文件名中的非法字符，使用 Unicode 码点表示替换非法字符
func cleanFileName(name string) string {
	cleaned := ""
	for _, r := range name {
		if unicode.IsPrint(r) && r < 127 {
			cleaned += string(r)
		} else {
			cleaned += fmt.Sprintf("%X", r)
		}
	}
	if len(cleaned) > 255 {
		cleaned = CalculateMD5(cleaned)
	}
	return cleaned
}

// CalculateMD5 计算给定字符串的 MD5 哈希
func CalculateMD5(text string) string {
	hashes := md5.New()
	hashes.Write([]byte(text))
	md5Hash := hashes.Sum(nil)
	return hex.EncodeToString(md5Hash)
}
