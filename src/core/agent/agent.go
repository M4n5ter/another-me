package agent

import "context"

// Agent 定义了所有 Worker Agent 的接口。
type Agent interface {
	// Run 执行代理的逻辑。
	// 输入和输出类型可能因代理而异，
	// 使用 'any' 提供灵活性，但推荐使用具体类型或结构体，
	// 在可能的情况下，为具体的实现提供具体类型。
	Run(ctx context.Context, input any) (any, error)
}
