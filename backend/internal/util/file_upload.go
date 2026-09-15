package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
}

// SaveUploadedFile 保存上传文件到 uploadDir/subdir，返回可访问的相对路径。
// subdir 用于隔离场景（cases 案件文件 / avatars 头像），案件文件不暴露公开静态路由。
func SaveUploadedFile(uploadDir, subdir string, maxMB int64, file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	if file.Size > maxMB*1024*1024 {
		return "", fmt.Errorf("file too large: %d bytes", file.Size)
	}
	dir := filepath.Join(uploadDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dst := filepath.Join(dir, name)
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("open upload file: %w", err)
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create destination file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", fmt.Errorf("write upload file: %w", err)
	}
	if subdir == "" {
		return "/uploads/" + name, nil
	}
	return "/uploads/" + subdir + "/" + name, nil
}

// ResolveUploadPath 把存储的文件 URL（/uploads/...）安全映射回磁盘路径，
// 拒绝目录穿越与绝对路径，保证解析结果始终落在 uploadDir 内。
func ResolveUploadPath(uploadDir, fileURL string) (string, error) {
	rel := strings.TrimPrefix(fileURL, "/uploads/")
	if rel == fileURL {
		return "", fmt.Errorf("invalid file url: %s", fileURL)
	}
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid file path: %s", fileURL)
	}
	full := filepath.Join(uploadDir, rel)
	absDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return "", fmt.Errorf("resolve upload dir: %w", err)
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("resolve file path: %w", err)
	}
	if absFull != absDir && !strings.HasPrefix(absFull, absDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("file path escapes upload dir: %s", fileURL)
	}
	return full, nil
}
