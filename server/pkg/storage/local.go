package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage 文件存储接口
type Storage interface {
	Save(tenantID uint64, date, filename string, reader io.Reader) (string, error)
}

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	basePath string
}

// NewLocalStorage 创建本地存储实例
func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{basePath: basePath}
}

// Save 保存文件到 ./uploads/{tenantID}/{date}/{filename}，返回相对路径
func (s *LocalStorage) Save(tenantID uint64, date, filename string, reader io.Reader) (string, error) {
	// 构建目录路径
	dir := filepath.Join(s.basePath, fmt.Sprintf("%d", tenantID), date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建完整文件路径
	fullPath := filepath.Join(dir, filename)
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 返回相对路径（用于访问）
	relativePath := filepath.Join("uploads", fmt.Sprintf("%d", tenantID), date, filename)
	return relativePath, nil
}

// Delete 删除文件
func (s *LocalStorage) Delete(relativePath string) error {
	// 从相对路径推导完整路径
	fullPath := filepath.Join(filepath.Dir(s.basePath), relativePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}
