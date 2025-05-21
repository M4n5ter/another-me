package admintool_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
)

// MockDirEntry 用于测试的目录项mock
type MockDirEntry struct {
	mock.Mock
	name  string
	isDir bool
}

func NewMockDirEntry(name string, isDir bool) *MockDirEntry {
	return &MockDirEntry{
		name:  name,
		isDir: isDir,
	}
}

func (m *MockDirEntry) Name() string { return m.name }
func (m *MockDirEntry) IsDir() bool  { return m.isDir }
func (m *MockDirEntry) Type() os.FileMode {
	if m.isDir {
		return os.ModeDir
	}
	return 0
}

func (m *MockDirEntry) Info() (os.FileInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(os.FileInfo), args.Error(1)
}

// TestArchiveTool_Compress 测试压缩功能
func TestArchiveTool_Compress(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewArchiveToolWithDeps(i18n.GlobalManager, mockRunner, mockFS)
	ctx := context.Background()

	// 测试场景1: 压缩单个文件
	t.Run("CompressSingleFile", func(t *testing.T) {
		// 模拟文件存在
		fileInfo := NewMockFileInfo("file.txt", 1024, 0o644, time.Now(), false)
		mockFS.On("Stat", "/test/file.txt").Return(fileInfo, nil).Once()

		// 模拟执行压缩命令 - 注意：为单个文件时，会根据文件名自动生成targetPath
		// 注意verboseFlag为空时会被removeEmpty函数移除
		mockRunner.On("Run", ctx, "zip", "file.txt.zip", "/test/file.txt").
			Return([]byte("adding: file.txt (stored 0%)"), nil).Once()

		// 执行被测函数
		inputJSON := `{"operation":"compress","source_path":"/test/file.txt","format":"zip"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"compress"`)
		assert.Contains(t, result, `"format":"zip"`)
		assert.Contains(t, result, `"message":"成功创建压缩文件:`)
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 压缩时源文件不存在
	t.Run("CompressSourceNotExist", func(t *testing.T) {
		// 模拟文件不存在
		mockFS.On("Stat", "/test/nonexist.txt").Return(nil, os.ErrNotExist).Once()

		// 执行被测函数
		inputJSON := `{"operation":"compress","source_path":"/test/nonexist.txt","format":"zip"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "源路径不存在")
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 压缩多个文件
	t.Run("CompressMultipleFiles", func(t *testing.T) {
		// 模拟多个文件存在
		fileInfo1 := NewMockFileInfo("file1.txt", 1024, 0o644, time.Now(), false)
		fileInfo2 := NewMockFileInfo("file2.txt", 2048, 0o644, time.Now(), false)
		mockFS.On("Stat", "/test/file1.txt").Return(fileInfo1, nil).Once()
		mockFS.On("Stat", "/test/file2.txt").Return(fileInfo2, nil).Once()

		// 模拟执行压缩命令 - 多个文件时，使用指定的目标路径
		mockRunner.On("Run", ctx, "zip", "archive.zip", "/test/file1.txt", "/test/file2.txt").
			Return([]byte("adding: file1.txt (stored 0%)\nadding: file2.txt (stored 0%)"), nil).Once()

		// 执行被测函数
		inputJSON := `{"operation":"compress","source_paths":["/test/file1.txt","/test/file2.txt"],"target_path":"archive.zip","format":"zip"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"compress"`)
		assert.Contains(t, result, `"format":"zip"`)
		assert.Contains(t, result, `"target_path":"archive.zip"`)
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})

	// 测试场景4: 使用tar.gz格式压缩
	t.Run("CompressWithTarGz", func(t *testing.T) {
		// 模拟文件存在
		fileInfo := NewMockFileInfo("folder", 0, 0o755, time.Now(), true)
		mockFS.On("Stat", "/test/folder").Return(fileInfo, nil).Once()

		// 模拟执行压缩命令 - 注意tar命令的参数格式与zip不同
		mockRunner.On("Run", ctx, "tar", "czf", "folder.tar.gz", "/test/folder").
			Return([]byte("folder/"), nil).Once()

		// 执行被测函数
		inputJSON := `{"operation":"compress","source_path":"/test/folder","format":"targz"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"compress"`)
		assert.Contains(t, result, `"format":"targz"`)
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})
}

// TestArchiveTool_Extract 测试解压功能
func TestArchiveTool_Extract(t *testing.T) {
	// 准备mock和测试环境
	mockRunner := new(MockCommandRunner)
	mockFS := new(MockFileSystem)
	tool := admintool.NewArchiveToolWithDeps(i18n.GlobalManager, mockRunner, mockFS)
	ctx := context.Background()

	// 测试场景1: 解压zip文件
	t.Run("ExtractZipFile", func(t *testing.T) {
		// 模拟源文件存在
		fileInfo := NewMockFileInfo("archive.zip", 2048, 0o644, time.Now(), false)
		mockFS.On("Stat", "/test/archive.zip").Return(fileInfo, nil).Once()

		// 模拟创建目标目录
		mockFS.On("MkdirAll", "/test/archive", os.FileMode(0o755)).Return(nil).Once()

		// 模拟解压命令
		mockRunner.On("Run", ctx, "unzip", "/test/archive.zip", "-d", "/test/archive").
			Return([]byte("inflating: file1.txt\ninflating: file2.txt"), nil).Once()

		// 执行被测函数
		inputJSON := `{"operation":"extract","source_path":"/test/archive.zip","target_path":"/test/archive"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"extract"`)
		assert.Contains(t, result, `"format":"zip"`)
		assert.Contains(t, result, `"message":"成功解压文件到: /test/archive"`)
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})

	// 测试场景2: 解压时源文件不存在
	t.Run("ExtractSourceNotExist", func(t *testing.T) {
		// 模拟文件不存在
		mockFS.On("Stat", "/test/nonexist.zip").Return(nil, os.ErrNotExist).Once()

		// 执行被测函数
		inputJSON := `{"operation":"extract","source_path":"/test/nonexist.zip"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "源文件不存在")
		mockFS.AssertExpectations(t)
	})

	// 测试场景3: 解压tar.gz文件
	t.Run("ExtractTarGzFile", func(t *testing.T) {
		// 模拟源文件存在
		fileInfo := NewMockFileInfo("archive.tar.gz", 3072, 0o644, time.Now(), false)
		mockFS.On("Stat", "/test/archive.tar.gz").Return(fileInfo, nil).Once()

		// 模拟创建目标目录
		mockFS.On("MkdirAll", "/test/archive", os.FileMode(0o755)).Return(nil).Once()

		// 模拟解压命令
		mockRunner.On("Run", ctx, "tar", "xzf", "/test/archive.tar.gz", "-C", "/test/archive").
			Return([]byte("extracting: ./file1.txt\nextracting: ./file2.txt"), nil).Once()

		// 执行被测函数
		inputJSON := `{"operation":"extract","source_path":"/test/archive.tar.gz","target_path":"/test/archive"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"operation":"extract"`)
		assert.Contains(t, result, `"format":"targz"`)
		assert.Contains(t, result, `"message":"成功解压文件到: /test/archive"`)
		mockRunner.AssertExpectations(t)
		mockFS.AssertExpectations(t)
	})

	// 测试场景4: 解压目标目录创建失败
	t.Run("ExtractTargetDirCreateFail", func(t *testing.T) {
		// 模拟源文件存在
		fileInfo := NewMockFileInfo("archive.zip", 2048, 0o644, time.Now(), false)
		mockFS.On("Stat", "/test/archive.zip").Return(fileInfo, nil).Once()

		// 模拟创建目标目录失败
		mockFS.On("MkdirAll", "/root/protected", os.FileMode(0o755)).Return(os.ErrPermission).Once()

		// 执行被测函数
		inputJSON := `{"operation":"extract","source_path":"/test/archive.zip","target_path":"/root/protected"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建目标目录失败")
		mockFS.AssertExpectations(t)
	})
}

// 创建一个Schema测试，确保工具正确实现了接口
func TestArchiveTool_Schema(t *testing.T) {
	// 创建工具实例
	tool := admintool.NewArchiveTool(i18n.GlobalManager)
	ctx := context.Background()

	// 获取模式
	schema, err := tool.Schema(ctx)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, "archive", schema.Name)
	assert.NotEmpty(t, schema.Descriptions)
	assert.NotEmpty(t, schema.InputParameters)
	assert.NotEmpty(t, schema.OutputParameters)

	// 验证必要的参数存在
	foundOperation := false
	foundSourcePath := false

	for _, param := range schema.InputParameters {
		if param.Name == "operation" {
			foundOperation = true
		}
		if param.Name == "source_path" {
			foundSourcePath = true
		}
	}

	assert.True(t, foundOperation, "Schema should include 'operation' parameter")
	assert.True(t, foundSourcePath, "Schema should include 'source_path' parameter")
}
