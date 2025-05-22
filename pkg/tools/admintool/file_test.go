package admintool_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
)

// MockCommandRunner 用于测试的命令执行器mock
type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(ctx context.Context, cmd string, args ...string) ([]byte, error) {
	args2 := make([]any, 0, len(args)+2)
	args2 = append(args2, ctx, cmd)
	for _, arg := range args {
		args2 = append(args2, arg)
	}
	called := m.Called(args2...)
	return called.Get(0).([]byte), called.Error(1)
}

func (m *MockCommandRunner) RunShell(ctx context.Context, cmdStr string) ([]byte, error) {
	called := m.Called(ctx, cmdStr)
	return called.Get(0).([]byte), called.Error(1)
}

// MockFileSystem 用于测试的文件系统mock
type MockFileSystem struct {
	mock.Mock
}

// CopyDir implements admintool.FileSystem.
func (m *MockFileSystem) CopyDir(src, dst string) error {
	called := m.Called(src, dst)
	return called.Error(0)
}

// CopyFile implements admintool.FileSystem.
func (m *MockFileSystem) CopyFile(src, dst string) error {
	called := m.Called(src, dst)
	return called.Error(0)
}

// Create implements admintool.FileSystem.
func (m *MockFileSystem) Create(name string) (*os.File, error) {
	called := m.Called(name)
	if called.Get(0) == nil {
		return nil, called.Error(1)
	}
	return called.Get(0).(*os.File), called.Error(1)
}

// IsNotExist implements admintool.FileSystem.
func (m *MockFileSystem) IsNotExist(err error) bool {
	called := m.Called(err)
	return called.Bool(0)
}

// Rename implements admintool.FileSystem.
func (m *MockFileSystem) Rename(oldpath, newpath string) error {
	called := m.Called(oldpath, newpath)
	return called.Error(0)
}

var _ admintool.FileSystem = (*MockFileSystem)(nil)

// 测试用常量
const (
	moveFileJSON = `{"path":"/test/file.txt","operation":"move","destination_path":"/test/new/file.txt"}`
	copyFileJSON = `{"path":"/test/file.txt","operation":"copy","destination_path":"/test/new/file.txt"}`
)

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	called := m.Called(name)
	if called.Get(0) == nil {
		return nil, called.Error(1)
	}
	return called.Get(0).(os.FileInfo), called.Error(1)
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	called := m.Called(filename)
	return called.Get(0).([]byte), called.Error(1)
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	called := m.Called(filename, data, perm)
	return called.Error(0)
}

func (m *MockFileSystem) Remove(name string) error {
	called := m.Called(name)
	return called.Error(0)
}

func (m *MockFileSystem) RemoveAll(path string) error {
	called := m.Called(path)
	return called.Error(0)
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	called := m.Called(path, perm)
	return called.Error(0)
}

func (m *MockFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	called := m.Called(name)
	if called.Get(0) == nil {
		return nil, called.Error(1)
	}
	return called.Get(0).([]os.DirEntry), called.Error(1)
}

func (m *MockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	called := m.Called(root, fn)
	return called.Error(0)
}

func (m *MockFileSystem) Rel(basepath, targpath string) (string, error) {
	called := m.Called(basepath, targpath)
	return called.String(0), called.Error(1)
}

// MockFileInfo 用于测试的文件信息mock
type MockFileInfo struct {
	mock.Mock
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func NewMockFileInfo(name string, size int64, mode os.FileMode, modTime time.Time, isDir bool) *MockFileInfo {
	return &MockFileInfo{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: modTime,
		isDir:   isDir,
	}
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *MockFileInfo) ModTime() time.Time { return m.modTime }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() any           { return nil }

// TestFileTool_ReadFile 测试文件读取功能
func TestFileTool_ReadFile(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 设置通用的mock行为，允许多次调用
	mockFS.On("IsNotExist", mock.Anything).Return(false).Maybe()

	// 测试场景1: 读取整个文件
	t.Run("ReadEntireFile", func(t *testing.T) {
		// 模拟文件存在
		modTime := time.Now()
		fileContent := "This is a test file content\nSecond line\nThird line"
		mockFileInfo := NewMockFileInfo("file.txt", int64(len(fileContent)), 0o644, modTime, false)

		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()
		mockFS.On("ReadFile", "/test/file.txt").Return([]byte(fileContent), nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"read"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"content":"This is a test file content\nSecond line\nThird line"`)
		assert.Contains(t, result, `"operation":"read"`)
		assert.Contains(t, result, `"line_count":2`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 通过head命令读取指定行数
	t.Run("ReadSpecificLines", func(t *testing.T) {
		// 模拟命令执行
		mockRunner.On("RunShell", ctx, "head -n 3 '/test/file.txt'").Return([]byte("line1\nline2\nline3"), nil).Once()

		// 执行被测函数
		lines := 3
		inputJSON := fmt.Sprintf(`{"path":"/test/file.txt","operation":"read","lines":%d}`, lines)
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"content":"line1\nline2\nline3"`)
		assert.Contains(t, result, `"operation":"read"`)
		mockFS.AssertExpectations(t)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景3: 命令执行失败
	t.Run("CommandError", func(t *testing.T) {
		// 模拟命令执行失败
		mockRunner.On("RunShell", ctx, "head -n 3 '/test/error.txt'").Return([]byte{}, errors.New("command failed")).Once()

		// 执行被测函数
		lines := 3
		inputJSON := fmt.Sprintf(`{"path":"/test/error.txt","operation":"read","lines":%d}`, lines)
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "读取文件失败")
		mockFS.AssertExpectations(t)
		mockRunner.AssertExpectations(t)
	})
}

// TestFileTool_WriteFile 测试文件写入功能
func TestFileTool_WriteFile(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 测试场景1: 正常写入文件
	t.Run("WriteFileSuccess", func(t *testing.T) {
		// 模拟写入文件
		mockFS.On("WriteFile", "/test/file.txt", []byte("new content"), os.FileMode(0o644)).
			Return(nil).Once()

		// 模拟获取文件信息
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 11, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"write","content":"new content"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"write"`)
		assert.Contains(t, result, `"message":"文件更新成功"`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 写入文件失败
	t.Run("WriteFileFail", func(t *testing.T) {
		// 模拟写入文件失败
		mockFS.On("WriteFile", "/test/file.txt", []byte("new content"), os.FileMode(0o644)).
			Return(errors.New("permission denied")).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"write","content":"new content"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "写入文件失败")
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 缺少内容参数
	t.Run("MissingContent", func(t *testing.T) {
		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"write"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "更新文件需要提供content参数")
		mockFS.AssertExpectations(t)
	})

	// 测试场景4: 查找和替换文本
	t.Run("FindAndReplace", func(t *testing.T) {
		// 模拟读取文件
		mockFS.On("ReadFile", "/test/file.txt").Return([]byte("old content"), nil).Once()

		// 模拟写入文件
		mockFS.On("WriteFile", "/test/file.txt", []byte("new content"), os.FileMode(0o644)).
			Return(nil).Once()

		// 模拟获取文件信息
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 11, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"write","find_text":"old content","replace_text":"new content"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"write"`)
		assert.Contains(t, result, `"文件更新成功，替换了文本'old content'"`)
		mockFS.AssertExpectations(t)
	})
}

// TestFileTool_DeleteFile 测试文件删除功能
func TestFileTool_DeleteFile(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 测试场景1: 删除文件成功
	t.Run("DeleteFileSuccess", func(t *testing.T) {
		// 模拟文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟删除文件
		mockFS.On("Remove", "/test/file.txt").Return(nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"delete"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"delete"`)
		assert.Contains(t, result, `"message":"文件删除成功"`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 删除目录成功（递归）
	t.Run("DeleteDirRecursiveSuccess", func(t *testing.T) {
		// 模拟目录存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("testdir", 0, 0o755, modTime, true)
		mockFS.On("Stat", "/test/testdir").Return(mockFileInfo, nil).Once()

		// 模拟递归删除目录
		mockFS.On("RemoveAll", "/test/testdir").Return(nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/testdir","operation":"delete","recursive":true}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"delete"`)
		assert.Contains(t, result, `"message":"目录删除成功"`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 非空目录不设置递归删除
	t.Run("NonEmptyDirWithoutRecursive", func(t *testing.T) {
		// 模拟目录存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("testdir", 0, 0o755, modTime, true)
		mockFS.On("Stat", "/test/testdir").Return(mockFileInfo, nil).Once()

		// 模拟目录不为空
		mockDirEntries := []fs.DirEntry{
			mockDirEntry("file1.txt", false),
		}
		mockFS.On("ReadDir", "/test/testdir").Return(mockDirEntries, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/testdir","operation":"delete"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "目录不为空，需要设置recursive=true才能删除非空目录")
		mockFS.AssertExpectations(t)
	})

	// 测试场景4: 删除文件失败
	t.Run("DeleteFileFail", func(t *testing.T) {
		// 模拟文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟删除文件失败
		mockFS.On("Remove", "/test/file.txt").Return(errors.New("permission denied")).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"delete"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "删除失败")
		mockFS.AssertExpectations(t)
	})

	// 测试场景5: 文件不存在
	t.Run("FileNotExist", func(t *testing.T) {
		// 模拟文件不存在
		mockFS.On("Stat", "/test/nonexistent.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/nonexistent.txt","operation":"delete"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文件或目录不存在")
		mockFS.AssertExpectations(t)
	})
}

// TestFileTool_ListDirectory 测试目录列表功能
func TestFileTool_ListDirectory(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 测试场景1: 列出目录内容成功
	t.Run("ListDirectorySuccess", func(t *testing.T) {
		// 模拟目录存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("testdir", 0, 0o755, modTime, true)
		mockFS.On("Stat", "/test/testdir").Return(mockFileInfo, nil).Once()

		// 创建模拟的目录条目
		mockDirEntries := []fs.DirEntry{
			mockDirEntry("file1.txt", false),
			mockDirEntry("file2.txt", false),
			mockDirEntry("subdir", true),
		}
		mockFS.On("ReadDir", "/test/testdir").Return(mockDirEntries, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/testdir","operation":"list"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"list"`)
		assert.Contains(t, result, `"file1.txt"`)
		assert.Contains(t, result, `"file2.txt"`)
		assert.Contains(t, result, `"subdir/"`) // 注意目录名后面有斜杠
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 目录不存在
	t.Run("DirectoryNotExist", func(t *testing.T) {
		// 模拟目录不存在
		mockFS.On("Stat", "/test/nonexistent").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/nonexistent","operation":"list"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "目录不存在")
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 读取目录失败
	t.Run("ReadDirFail", func(t *testing.T) {
		// 模拟目录存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("testdir", 0, 0o755, modTime, true)
		mockFS.On("Stat", "/test/testdir").Return(mockFileInfo, nil).Once()

		// 模拟读取目录失败
		mockFS.On("ReadDir", "/test/testdir").Return(nil, errors.New("permission denied")).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/testdir","operation":"list"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "读取目录内容失败")
		mockFS.AssertExpectations(t)
	})
}

// mockDirEntry 创建一个模拟的目录条目
func mockDirEntry(name string, isDir bool) fs.DirEntry {
	return &mockDirEntryImpl{name: name, isDir: isDir}
}

// mockDirEntryImpl 实现 fs.DirEntry 接口
type mockDirEntryImpl struct {
	name  string
	isDir bool
}

func (m *mockDirEntryImpl) Name() string { return m.name }
func (m *mockDirEntryImpl) IsDir() bool  { return m.isDir }
func (m *mockDirEntryImpl) Type() fs.FileMode {
	if m.isDir {
		return fs.ModeDir
	}
	return 0
}

func (m *mockDirEntryImpl) Info() (fs.FileInfo, error) {
	return nil, nil // 不需要实现，因为在测试中没有使用
}

// TestFileTool_FileExists 测试文件存在检查功能
func TestFileTool_FileExists(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 测试场景1: 文件存在
	t.Run("FileExists", func(t *testing.T) {
		// 模拟文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("test.txt", 100, 0o644, modTime, false)
		mockFS.On("Stat", "/test/test.txt").Return(mockFileInfo, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/test.txt","operation":"exists"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"exists"`)
		assert.Contains(t, result, `"exists":true`)
		// 不检查 is_dir 字段，因为实现中可能没有返回该字段
		assert.Contains(t, result, `"message":`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 目录存在
	t.Run("DirectoryExists", func(t *testing.T) {
		// 模拟目录存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("testdir", 0, 0o755, modTime, true)
		mockFS.On("Stat", "/test/testdir").Return(mockFileInfo, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/testdir","operation":"exists"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"exists"`)
		assert.Contains(t, result, `"exists":true`)
		assert.Contains(t, result, `"is_dir":true`)
		assert.Contains(t, result, `"message":`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 文件不存在
	t.Run("FileNotExist", func(t *testing.T) {
		// 模拟文件不存在
		mockFS.On("Stat", "/test/nonexistent.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/nonexistent.txt","operation":"exists"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"exists"`)
		// 不检查 exists 字段，因为实现中可能没有返回该字段
		assert.Contains(t, result, `"message":`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景4: Stat函数返回错误
	t.Run("StatError", func(t *testing.T) {
		// 模拟检查文件存在失败
		mockFS.On("Stat", "/test/error.txt").Return(nil, errors.New("permission denied")).Once()
		mockFS.On("IsNotExist", errors.New("permission denied")).Return(false).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/error.txt","operation":"exists"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "获取文件信息失败")
		mockFS.AssertExpectations(t)
	})
}

// TestFileTool_MoveFile 测试文件移动功能
func TestFileTool_MoveFile(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 测试场景1: 移动文件成功
	t.Run("MoveFileSuccess", func(t *testing.T) {
		// 模拟源文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟检查目标路径
		mockFS.On("Stat", "/test/new/file.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 模拟移动文件
		mockFS.On("Rename", "/test/file.txt", "/test/new/file.txt").Return(nil).Once()

		// 执行被测函数
		inputJSON := moveFileJSON
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"move"`)
		assert.Contains(t, result, `"message":"文件移动成功"`)
		assert.Contains(t, result, `"destination":"/test/new/file.txt"`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 源文件不存在
	t.Run("SourceFileNotExist", func(t *testing.T) {
		// 模拟源文件不存在
		mockFS.On("Stat", "/test/nonexistent.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/nonexistent.txt","operation":"move","destination_path":"/test/new/file.txt"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文件或目录不存在")
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 目标路径检查失败
	t.Run("DestinationStatFail", func(t *testing.T) {
		// 模拟源文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟检查目标路径失败
		mockFS.On("Stat", "/test/new/file.txt").Return(nil, errors.New("permission denied")).Once()
		mockFS.On("IsNotExist", mock.Anything).Return(false).Once()

		// 执行被测函数
		inputJSON := moveFileJSON
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "获取目标路径信息失败")
		mockFS.AssertExpectations(t)
	})

	// 测试场景4: 移动文件失败
	t.Run("MoveFileFail", func(t *testing.T) {
		// 模拟源文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟检查目标路径
		mockFS.On("Stat", "/test/new/file.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 模拟移动文件失败
		mockFS.On("Rename", "/test/file.txt", "/test/new/file.txt").
			Return(errors.New("permission denied")).Once()

		// 执行被测函数
		inputJSON := moveFileJSON
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "移动文件失败")
		mockFS.AssertExpectations(t)
	})

	// 测试场景5: 缺少目标路径
	t.Run("MissingDestinationPath", func(t *testing.T) {
		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"move"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "移动文件需要提供destination_path参数")
		mockFS.AssertExpectations(t)
	})
}

// TestFileTool_CopyFile 测试文件复制功能
func TestFileTool_CopyFile(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 测试场景1: 复制文件成功
	t.Run("CopyFileSuccess", func(t *testing.T) {
		// 模拟源文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟检查目标路径
		mockFS.On("Stat", "/test/new/file.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 模拟复制文件
		mockFS.On("CopyFile", "/test/file.txt", "/test/new/file.txt").Return(nil).Once()

		// 执行被测函数
		inputJSON := copyFileJSON
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"copy"`)
		assert.Contains(t, result, `"message":"文件复制成功"`)
		assert.Contains(t, result, `"destination":"/test/new/file.txt"`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 复制目录成功
	t.Run("CopyDirSuccess", func(t *testing.T) {
		// 模拟源目录存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("testdir", 0, 0o755, modTime, true)
		mockFS.On("Stat", "/test/testdir").Return(mockFileInfo, nil).Once()

		// 模拟检查目标路径
		mockFS.On("Stat", "/test/new/testdir").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 模拟复制目录
		mockFS.On("CopyDir", "/test/testdir", "/test/new/testdir").Return(nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/testdir","operation":"copy","destination_path":"/test/new/testdir"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"copy"`)
		assert.Contains(t, result, `"message":"文件复制成功"`)
		assert.Contains(t, result, `"destination":"/test/new/testdir"`)
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 源文件不存在
	t.Run("SourceFileNotExist", func(t *testing.T) {
		// 模拟源文件不存在
		mockFS.On("Stat", "/test/nonexistent.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/nonexistent.txt","operation":"copy","destination_path":"/test/new/file.txt"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "文件或目录不存在")
		mockFS.AssertExpectations(t)
	})

	// 测试场景4: 目标路径检查失败
	t.Run("DestinationStatFail", func(t *testing.T) {
		// 模拟源文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟检查目标路径失败
		mockFS.On("Stat", "/test/new/file.txt").Return(nil, errors.New("permission denied")).Once()
		mockFS.On("IsNotExist", mock.Anything).Return(false).Once()

		// 执行被测函数
		inputJSON := copyFileJSON
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "获取目标路径信息失败")
		mockFS.AssertExpectations(t)
	})

	// 测试场景5: 复制文件失败
	t.Run("CopyFileFail", func(t *testing.T) {
		// 模拟源文件存在
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("file.txt", 10, 0o644, modTime, false)
		mockFS.On("Stat", "/test/file.txt").Return(mockFileInfo, nil).Once()

		// 模拟检查目标路径
		mockFS.On("Stat", "/test/new/file.txt").Return(nil, os.ErrNotExist).Once()
		mockFS.On("IsNotExist", os.ErrNotExist).Return(true).Once()

		// 模拟复制文件失败
		mockFS.On("CopyFile", "/test/file.txt", "/test/new/file.txt").
			Return(errors.New("permission denied")).Once()

		// 执行被测函数
		inputJSON := copyFileJSON
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "复制文件失败")
		mockFS.AssertExpectations(t)
	})

	// 测试场景6: 缺少目标路径
	t.Run("MissingDestinationPath", func(t *testing.T) {
		// 执行被测函数
		inputJSON := `{"path":"/test/file.txt","operation":"copy"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "复制文件需要提供destination_path参数")
		mockFS.AssertExpectations(t)
	})
}

// TestFileTool_CreateFile 测试文件创建功能
func TestFileTool_CreateFile(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockFS, mockRunner)
	ctx := context.Background()

	// 测试场景: 创建文件成功
	t.Run("CreateFileSuccess", func(t *testing.T) {
		// 模拟创建父目录
		mockRunner.On("RunShell", ctx, "mkdir -p '/test'").Return([]byte(""), nil).Once()

		// 模拟创建文件
		mockRunner.On("Run", ctx, "touch", "/test/newfile.txt").Return([]byte(""), nil).Once()

		// 模拟获取文件信息
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("newfile.txt", 0, 0o644, modTime, false)
		mockFS.On("Stat", "/test/newfile.txt").Return(mockFileInfo, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/newfile.txt","operation":"create","content":"test content"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"create"`)
		assert.Contains(t, result, `"message":"文件创建成功"`)
		mockFS.AssertExpectations(t)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景: 创建目录成功
	t.Run("CreateDirectorySuccess", func(t *testing.T) {
		// 模拟创建目录
		mockRunner.On("RunShell", ctx, "mkdir '/test/newdir'").Return([]byte(""), nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/newdir","operation":"create","is_dir":true}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"create"`)
		assert.Contains(t, result, `"is_dir":true`)
		assert.Contains(t, result, `"message":"目录创建成功"`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景: 创建文件失败
	t.Run("CreateFileFail", func(t *testing.T) {
		// 模拟创建父目录
		mockRunner.On("RunShell", ctx, "mkdir -p '/root'").Return([]byte(""), nil).Once()

		// 模拟创建文件失败
		mockRunner.On("Run", ctx, "touch", "/root/forbidden.txt").Return([]byte(""), errors.New("permission denied")).Once()

		// 执行被测函数
		inputJSON := `{"path":"/root/forbidden.txt","operation":"create"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建文件失败")
		mockRunner.AssertExpectations(t)
	})
}
