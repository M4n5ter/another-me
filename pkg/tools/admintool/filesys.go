package admintool

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileSystem 定义文件系统操作接口
type FileSystem interface {
	// Stat 获取文件信息
	Stat(name string) (os.FileInfo, error)
	// ReadFile 读取文件内容
	ReadFile(filename string) ([]byte, error)
	// WriteFile 写入文件内容
	WriteFile(filename string, data []byte, perm os.FileMode) error
	// Remove 删除文件或空目录
	Remove(name string) error
	// RemoveAll 递归删除目录及内容
	RemoveAll(path string) error
	// MkdirAll 创建目录及所需的父目录
	MkdirAll(path string, perm os.FileMode) error
	// ReadDir 读取目录内容
	ReadDir(name string) ([]os.DirEntry, error)
	// WalkDir 递归遍历目录
	WalkDir(root string, fn fs.WalkDirFunc) error
	// Rel 获取相对路径
	Rel(basepath, targpath string) (string, error)
}

// RealFileSystem 真实文件系统操作实现
type RealFileSystem struct{}

// NewRealFileSystem 创建真实文件系统操作实例
func NewRealFileSystem() FileSystem {
	return &RealFileSystem{}
}

// Stat 调用os.Stat以获取文件信息
func (fs *RealFileSystem) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}
	return info, nil
}

// ReadFile 调用os.ReadFile读取整个文件内容
func (fs *RealFileSystem) ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	return data, nil
}

// WriteFile 调用os.WriteFile将数据写入文件
func (fs *RealFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.WriteFile(filename, data, perm); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

// Remove 调用os.Remove删除文件或空目录
func (fs *RealFileSystem) Remove(name string) error {
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

// RemoveAll 调用os.RemoveAll递归删除文件或目录
func (fs *RealFileSystem) RemoveAll(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("递归删除失败: %w", err)
	}
	return nil
}

// MkdirAll 调用os.MkdirAll递归创建目录
func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return nil
}

// ReadDir 调用os.ReadDir获取目录内容
func (fs *RealFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}
	return entries, nil
}

// WalkDir 调用filepath.WalkDir递归遍历目录
func (fs *RealFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	if err := filepath.WalkDir(root, fn); err != nil {
		return fmt.Errorf("遍历目录失败: %w", err)
	}
	return nil
}

// Rel 调用filepath.Rel计算相对路径
func (fs *RealFileSystem) Rel(basepath, targpath string) (string, error) {
	rel, err := filepath.Rel(basepath, targpath)
	if err != nil {
		return "", fmt.Errorf("计算相对路径失败: %w", err)
	}
	return rel, nil
}
