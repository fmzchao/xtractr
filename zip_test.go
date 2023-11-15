package xtractr_test

import (
	"golift.io/xtractr"
	"io/ioutil"
	"os"
	"testing"
)

const (
	testzipFile     = "test_data/archive.rar"
	testZipDataSize = int64(20770)
)

func TestExtractZIP(t *testing.T) {
	// 创建临时目录来存放解压的文件
	tempDir, err := ioutil.TempDir("", "xtractr_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试用例
	tests := []struct {
		name      string
		filePath  string
		password  string
		wantErr   bool
		wantFiles int
	}{
		{
			name:      "No password",
			filePath:  "testdata/nopassword.zip",
			password:  "",
			wantErr:   false,
			wantFiles: 3, // 假设无密码ZIP包含3个文件
		},
		{
			name:      "With password",
			filePath:  "testdata/withpassword.zip",
			password:  "correctpassword",
			wantErr:   false,
			wantFiles: 2, // 假设有密码ZIP包含2个文件
		},
		{
			name:      "Wrong password",
			filePath:  "testdata/withpassword.zip",
			password:  "wrongpassword",
			wantErr:   true,
			wantFiles: 0,
		},
		{
			name:      "File does not exist",
			filePath:  "testdata/nonexistent.zip",
			password:  "",
			wantErr:   true,
			wantFiles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xFile := &xtractr.XFile{
				FilePath:  tt.filePath,
				OutputDir: tempDir,
				Password:  tt.password,
			}

			_, files, err := xtractr.ExtractZIP(xFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractZIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(files) != tt.wantFiles {
				t.Errorf("ExtractZIP() got %v files, want %v files", len(files), tt.wantFiles)
			}
		})
	}
}
