package admintool

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// FileSystem 定义了文件系统操作的接口，方便测试
type FileSystem interface {
	Stat(name string) (fs.FileInfo, error)
	Create(name string) (*os.File, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	ReadFile(name string) ([]byte, error)
	Remove(name string) error
	RemoveAll(path string) error
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(name string) ([]fs.DirEntry, error)
	IsNotExist(err error) bool
	Rename(oldpath, newpath string) error
	CopyFile(src, dst string) error
	CopyDir(src, dst string) error
}

// RealFileSystem 使用真实的 os 包实现 FileSystem 接口
type RealFileSystem struct{}

var _ FileSystem = (*RealFileSystem)(nil)

// Stat 获取文件信息
func (rfs *RealFileSystem) Stat(name string) (fs.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", name, err)
	}
	return info, nil
}

// Create 创建文件
func (rfs *RealFileSystem) Create(name string) (*os.File, error) {
	file, err := os.Create(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", name, err)
	}
	return file, nil
}

// WriteFile 写入文件
func (rfs *RealFileSystem) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return fmt.Errorf("failed to write %s: %w", name, os.WriteFile(name, data, perm))
}

// ReadFile 读取文件
func (rfs *RealFileSystem) ReadFile(name string) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", name, err)
	}
	return data, nil
}

// Remove 删除文件
func (rfs *RealFileSystem) Remove(name string) error {
	return fmt.Errorf("failed to remove %s: %w", name, os.Remove(name))
}

// RemoveAll 删除目录
func (rfs *RealFileSystem) RemoveAll(path string) error {
	return fmt.Errorf("failed to remove %s: %w", path, os.RemoveAll(path))
}

// MkdirAll 创建目录
func (rfs *RealFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return fmt.Errorf("failed to mkdir %s: %w", path, os.MkdirAll(path, perm))
}

// ReadDir 读取目录
func (rfs *RealFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir %s: %w", name, err)
	}
	return entries, nil
}

// IsNotExist 判断错误是否是文件不存在
func (rfs *RealFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// Rename 重命名文件或目录
func (rfs *RealFileSystem) Rename(oldpath, newpath string) error {
	return fmt.Errorf("failed to rename %s to %s: %w", oldpath, newpath, os.Rename(oldpath, newpath))
}

// CopyFile copies a single file from src to dst
func (rfs *RealFileSystem) CopyFile(src, dst string) error {
	return copyFileUtil(src, dst)
}

// CopyDir recursively copies a directory from src to dst
func (rfs *RealFileSystem) CopyDir(src, dst string) error {
	return copyDirUtil(src, dst)
}

// copyFileUtil is a helper function to copy a file.
func copyFileUtil(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", src, err)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", src, err)
	}
	defer func() {
		if err := source.Close(); err != nil {
			fmt.Printf("failed to close %s: %v\n", src, err)
		}
	}()

	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dst, err)
	}
	defer func() {
		if err := destination.Close(); err != nil {
			fmt.Printf("failed to close %s: %v\n", dst, err)
		}
	}()

	buf := make([]byte, 4096) // 4KB buffer
	for {
		n, errRead := source.Read(buf)
		if errRead != nil && errRead != io.EOF {
			return fmt.Errorf("failed to read %s: %w", src, errRead)
		}
		if n == 0 {
			break
		}

		if _, errWrite := destination.Write(buf[:n]); errWrite != nil {
			return fmt.Errorf("failed to write %s: %w", dst, errWrite)
		}
	}

	// 保留权限
	err = os.Chmod(dst, sourceFileStat.Mode())
	if err != nil {
		slog.Error("failed to chmod %s: %v", dst, err)
	}

	return nil
}

// copyDirUtil is a helper function to recursively copy a directory.
func copyDirUtil(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", src, err)
	}

	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to mkdir %s: %w", dst, err)
	}

	dirEntries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read dir %s: %w", src, err)
	}

	for _, entry := range dirEntries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDirUtil(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFileUtil(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
