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
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockRunner, mockFS)
	ctx := context.Background()

	// 测试场景1: 通过head命令读取指定行数
	t.Run("ReadSpecificLines", func(t *testing.T) {
		// 设置mock预期
		mockRunner.On("RunShell", ctx, "head -n 5 '/test/file.txt'").
			Return([]byte("line1\nline2\nline3\nline4\nline5"), nil).Once()

		// 执行被测函数
		lines := 5
		inputJSON := fmt.Sprintf(`{"path":"/test/file.txt","operation":"read","lines":%d}`, lines)
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"content":"line1\nline2\nline3\nline4\nline5"`)
		assert.Contains(t, result, `"operation":"read"`)
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 命令执行失败
	t.Run("CommandError", func(t *testing.T) {
		// 设置mock预期
		mockRunner.On("RunShell", ctx, "head -n 3 '/nonexistent/file.txt'").
			Return([]byte{}, errors.New("command failed")).Once()

		// 执行被测函数
		lines := 3
		inputJSON := fmt.Sprintf(`{"path":"/nonexistent/file.txt","operation":"read","lines":%d}`, lines)
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "读取文件失败")
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 读取整个文件
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
}

// TestFileTool_CreateFile 测试文件创建功能
func TestFileTool_CreateFile(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewFileToolWithDeps(i18n.GlobalManager, mockRunner, mockFS)
	ctx := context.Background()

	// 测试场景: 创建文件
	t.Run("CreateFile", func(t *testing.T) {
		// 添加创建父目录的mock
		mockRunner.On("RunShell", ctx, "mkdir -p '/test'").
			Return([]byte(""), nil).Once()

		// 创建文件前先检查文件是否存在，这里模拟文件不存在
		mockRunner.On("Run", ctx, "touch", "/test/newfile.txt").
			Return([]byte(""), nil).Once()

		// 模拟获取文件信息
		modTime := time.Now()
		mockFileInfo := NewMockFileInfo("newfile.txt", 12, 0o644, modTime, false)
		mockFS.On("Stat", "/test/newfile.txt").Return(mockFileInfo, nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/newfile.txt","operation":"create","content":"test content"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"create"`)
		assert.Contains(t, result, `"message":"文件创建成功"`)
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})

	// 测试场景: 创建目录
	t.Run("CreateDirectory", func(t *testing.T) {
		// 模拟RunShell调用
		mockRunner.On("RunShell", ctx, "mkdir '/test/newdir'").
			Return([]byte(""), nil).Once()

		// 执行被测函数
		inputJSON := `{"path":"/test/newdir","operation":"create","is_dir":true}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"create"`)
		assert.Contains(t, result, `"message":"目录创建成功"`)
		assert.Contains(t, result, `"is_dir":true`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景: 创建文件失败
	t.Run("CreateFileFail", func(t *testing.T) {
		// 添加创建父目录的mock
		mockRunner.On("RunShell", ctx, "mkdir -p '/root'").
			Return([]byte(""), nil).Once()

		// 模拟创建文件失败
		mockRunner.On("Run", ctx, "touch", "/root/forbidden.txt").
			Return([]byte("permission denied"), errors.New("permission denied")).Once()

		// 执行被测函数
		inputJSON := `{"path":"/root/forbidden.txt","operation":"create"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建文件失败")
		mockRunner.AssertExpectations(t)
	})
}
