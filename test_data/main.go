package main

import (
	"fmt"
	"github.com/fmzchao/xtractr"
	"io"
	"os"
	"path/filepath"

	"github.com/yeka/zip"
)

func ExtractZipWithPassword(zipFilePath, outputDir, password string) error {
	// 打开ZIP文件
	zipReader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %v", err)
	}
	defer zipReader.Close()

	// 遍历ZIP文件中的每个文件/目录
	for _, f := range zipReader.File {
		// 为这个文件/目录设置密码
		f.SetPassword(password)

		fpath := filepath.Join(outputDir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to open file: %v", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file inside zip: %v", err)
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed to write file: %v", err)
		}
	}

	return nil
}
func main() {
	zipFile := "./withpassword.zip" // ZIP文件路径
	outputDir := "./temp/"          // 解压目标文件夹

	var zipxFile = &xtractr.XFile{
		FilePath:  zipFile,
		OutputDir: outputDir,
		Password:  "some_password",
	}
	num, fils, err := xtractr.ExtractZipWithPassword(zipxFile)
	if err != nil {
		fmt.Printf("Error extracting zip file: %s\n", err)
	} else {
		fmt.Println("Zip file extracted successfully.", num, fils)
	}
	/*	if err := ExtractZipWithPassword(zipFile, outputDir, password); err != nil {
			fmt.Printf("Error extracting zip file: %s\n", err)
		} else {
			fmt.Println("Zip file extracted successfully.")
		}*/

}
