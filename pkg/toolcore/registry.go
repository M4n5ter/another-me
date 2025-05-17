package toolcore

import (
	"context"
	"fmt"
	"sync"
)

// Registry 结构体用于存储和管理一组已注册的 Tool。
// 它是线程安全的，允许多个 goroutine 并发地注册和检索工具。
type Registry struct {
	mu    sync.RWMutex    // mu 用于保护对内部 tools map 的并发访问。
	tools map[string]Tool // tools map 以工具的规范名称 (ToolSchema.Name) 为键，存储 Tool 实例。
}

// NewRegistry 创建并返回一个新的、空的 Registry 实例。
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register 方法用于向 Registry 中添加一个新的工具。
// 它会首先调用工具的 Schema() 方法获取其元数据，并使用元数据中的 Name 字段作为注册键。
// 如果工具的 Schema() 方法返回错误，或者 Schema 中的 Name 为空，则注册失败。
// 如果具有相同名称的工具已存在于注册表中，则注册失败并返回错误。
// ctx: 用于工具 Schema() 方法的上下文。
// tool: 需要注册的 Tool 实例。
func (r *Registry) Register(ctx context.Context, tool Tool) error {
	r.mu.Lock() // 获取写锁，因为要修改 tools map
	defer r.mu.Unlock()

	schema, err := tool.Schema(ctx)
	if err != nil {
		return fmt.Errorf("注册工具失败：无法获取工具的 schema: %w", err)
	}

	if schema.Name == "" {
		// 可以考虑定义一个更具体的错误类型，例如 ErrToolNameEmpty
		return fmt.Errorf("注册工具失败：工具 schema 中的 Name 字段不能为空")
	}

	if _, exists := r.tools[schema.Name]; exists {
		return fmt.Errorf("注册工具失败：名为 '%s' 的工具已存在", schema.Name)
	}

	r.tools[schema.Name] = tool
	return nil
}

// Get 方法根据工具的规范名称从 Registry 中检索一个已注册的 Tool 实例。
// name: 需要检索的工具的规范名称 (ToolSchema.Name)。
// 返回值: 如果找到工具，则返回对应的 Tool 实例和 true；否则返回 nil 和 false。
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock() // 获取读锁，因为只读取 tools map
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// GetAll 方法返回 Registry 中所有已注册工具的 Tool 实例列表。
func (r *Registry) GetAll() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListSchemas 方法返回 Registry 中所有已注册工具的 ToolSchema 元数据列表。
// 这对于向 LLM 或 UI 展示可用工具有用。
// ctx: 用于每个工具 Schema() 方法的上下文。
// 返回值: 一个包含所有 ToolSchema 的切片和可能的错误。
// 如果在获取任何一个工具的 schema 时发生错误，它会继续尝试获取其他工具的 schema，
// 并将所有错误信息汇总后返回（当前实现为返回遇到的第一个错误）。
func (r *Registry) ListSchemas(ctx context.Context) ([]ToolSchema, error) {
	r.mu.RLock() // 获取读锁，因为只读取 tools map，但会调用 tool.Schema()
	defer r.mu.RUnlock()

	schemas := make([]ToolSchema, 0, len(r.tools))
	var collectedErrors []error // 用于收集在获取 schema 过程中可能发生的多个错误

	for toolName, toolInstance := range r.tools {
		schema, err := toolInstance.Schema(ctx)
		if err != nil {
			// 即使某个工具的 Schema() 出错，也继续尝试获取其他工具的 Schema
			collectedErrors = append(collectedErrors, fmt.Errorf("获取工具 '%s' 的 schema 失败: %w", toolName, err))
			continue
		}
		schemas = append(schemas, schema)
	}

	if len(collectedErrors) > 0 {
		// 如果有多个错误，可以将它们组合成一个错误返回。
		// 为了简单起见，这里只返回第一个遇到的错误，并在消息中指明错误数量。
		// 更好的做法可能是返回一个包含所有错误的自定义错误类型或错误切片。
		return schemas, fmt.Errorf("列出工具 schemas 时遇到 %d 个错误；第一个错误: %w", len(collectedErrors), collectedErrors[0])
	}

	return schemas, nil
}
