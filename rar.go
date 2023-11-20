package xtractr

/* How to extract a RAR file. */

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nwaples/rardecode"
)

func ExtractRAR(xFile *XFile) (int64, []string, []string, error) {
	if len(xFile.Passwords) == 0 && xFile.Password == "" {
		return extractRAR(xFile)
	}

	// Try all the passwords.
	passwords := xFile.Passwords

	if xFile.Password != "" { // If a single password is provided, try it first.
		passwords = append([]string{xFile.Password}, xFile.Passwords...)
	}

	for idx, password := range passwords {
		size, files, archives, err := extractRAR(&XFile{
			FilePath:  xFile.FilePath,
			OutputDir: xFile.OutputDir,
			FileMode:  xFile.FileMode,
			DirMode:   xFile.DirMode,
			Password:  password,
		})
		if err == nil {
			return size, files, archives, nil
		} else {
			fmt.Println(err)
		}

		// https://github.com/nwaples/rardecode/issues/28
		if strings.Contains(err.Error(), "incorrect password") {
			continue
		}

		return size, files, archives, fmt.Errorf("used password %d of %d: %w", idx+1, len(passwords), err)
	}

	// No password worked, try without a password.
	return extractRAR(&XFile{
		FilePath:  xFile.FilePath,
		OutputDir: xFile.OutputDir,
		FileMode:  xFile.FileMode,
		DirMode:   xFile.DirMode,
	})
}

// ExtractRAR extracts a rar file. to a destination. This wraps github.com/nwaples/rardecode.
func extractRAR(xFile *XFile) (int64, []string, []string, error) {
	rarReader, err := rardecode.OpenReader(xFile.FilePath, xFile.Password)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("rardecode.OpenReader: %w", err)
	}
	defer rarReader.Close()

	size, files, err := xFile.unrar(rarReader)
	if err != nil {
		lastFile := xFile.FilePath
		if volumes := rarReader.Volumes(); len(volumes) > 0 {
			lastFile = volumes[len(volumes)-1]
		}

		return size, files, rarReader.Volumes(), fmt.Errorf("%s: %w", lastFile, err)
	}

	return size, files, rarReader.Volumes(), nil
}

func (x *XFile) unrar(rarReader *rardecode.ReadCloser) (int64, []string, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Program recovered from error: %v\n", r)
		}
	}()

	files := []string{}
	size := int64(0)

	for {
		header, err := rarReader.Next()

		switch {
		case errors.Is(err, io.EOF):
			// 如果是正常的 EOF，结束循环
			return size, files, nil
		case err != nil:
			if errors.Is(err, io.ErrUnexpectedEOF) || strings.Contains(err.Error(), "copying io") || strings.Contains(err.Error(), "bad header crc") {
				// 如果是 unexpected EOF，忽略这个错误并继续
				return size, files, nil
			}
			// 其他错误，退出并返回错误
			return size, files, fmt.Errorf("rarReader.Next: %w", err)
		case header == nil:
			// 如果 header 为 nil，返回错误
			return size, files, fmt.Errorf("%w: %s", ErrInvalidHead, x.FilePath)
		}

		wfile := x.clean(header.Name)
		//nolint:gocritic // this 1-argument filepath.Join removes a ./ prefix should there be one.
		if !strings.HasPrefix(wfile, filepath.Join(x.OutputDir)) {
			// The file being written is trying to write outside of our base path. Malicious archive?
			return size, files, fmt.Errorf("%s: %w: %s != %s (from: %s)",
				x.FilePath, ErrInvalidPath, wfile, x.OutputDir, header.Name)
		}

		if header.IsDir {
			if err = os.MkdirAll(wfile, x.DirMode); err != nil {
				//return size, files, fmt.Errorf("os.MkdirAll: %w", err)
				log.Printf("Error creating directory: %v", err)
				continue
			}

			continue
		}

		if err = os.MkdirAll(filepath.Dir(wfile), x.DirMode); err != nil {
			log.Printf("Error creating directory: %v", err)
			continue
			//return size, files, fmt.Errorf("os.MkdirAll: %w", err)
		}

		fSize, err := writeFile(wfile, rarReader, x.FileMode, x.DirMode)
		if err != nil && (!strings.Contains(err.Error(), "unexpected EOF")) && (!strings.Contains(err.Error(), "copying io")) && (!strings.Contains(err.Error(), "bad header crc")) {
			return size, files, err
		}

		files = append(files, wfile)
		size += fSize
	}
}
