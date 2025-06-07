package llminterface

import (
	"context"

	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// ChatAdapter 是与 LLM (通过特定框架适配)进行交互的统一接口。
// 它旨在提供一个简洁的抽象层，允许应用代码以一致的方式发送请求并接收响应，
// 而不论底层使用的是哪个 Go LLM 框架 (例如 langchaingo, eino 等)。
type ChatAdapter interface {
	// Chat 方法用于向 LLM 发起一次对话交互。
	//
	// 重要：此方法总是返回一个只读的 channel (<-chan ChatOutputChunk)。
	// 这是为了统一流式和非流式的使用场景。
	//
	//   - 对于流式使用：调用者可以持续从 channel 中读取 ChatOutputChunk 数据块，
	//     直到 channel 关闭，从而实时处理 LLM 的增量响应。
	//
	//   - 对于非流式使用（即获取完整响应）：调用者需要聚合所有从 channel 中接收到的
	//     ChatOutputChunk 的 TextDelta 字段，直到 channel 关闭，从而构建出完整的响应文本。
	//     在聚合过程中，应检查每个 chunk 的 Error 字段。
	//
	// 输入 (input ChatInput): 包含了要发送给 LLM 的消息序列以及可选的会话ID。
	//      ChatInput.ConversationID 可被适配器实现用于在底层框架中维持会话上下文。
	//
	// 返回值:
	//   - (<-chan ChatOutputChunk): 一个用于接收 LLM 响应数据块的 channel。
	//                           适配器实现负责在交互完成或发生不可恢复错误时关闭此 channel。
	//   - error: 如果在尝试发起聊天请求的初始阶段就发生错误（例如配置错误、无法连接服务等），
	//            则直接返回一个非 nil 的 error。若成功发起请求（即使后续流中可能出错），则返回 nil。
	//
	// 错误处理:
	//   - 初始错误: 通过 Chat 方法的第二个返回值直接返回。
	//   - 流处理过程中的错误: 通过 ChatOutputChunk.Error 字段在 channel 中传递。
	//                     通常，当 chunk.Error 非 nil 时，表示流中出现问题，channel 随后会被关闭。
	Chat(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error)

	// ProduceJSON 方法用于生成 JSON 格式的响应。
	// 输入 ChatInput 包含要发送给 LLM 的消息序列以及可选的会话ID。
	// JSONSchema 是可选的，如果提供，则表示需要返回的 JSON 格式，不提供的话，则需要在提示词中明确返回的 JSON 格式。
	//
	// 返回值是一个 JSON 字符串，如果发生错误，则返回一个非 nil 的 error。
	ProduceJSON(ctx context.Context, input ChatInput, JSONSchema Option[schema.Schema]) (string, error)

	// RegisterTools 方法用于向适配器注册一组工具。
	// 这允许适配器在底层框架中使用工具。
	RegisterTools(ctx context.Context, registry *toolcore.Registry) error

	// GetFrameworkName 返回此适配器实例所适配的底层框架的名称。
	// 例如 "langchaingo", "eino", "custom-ollama-api" 等。
	// 这有助于调试和了解当前使用的是哪个具体的 LLM 交互实现。
	GetFrameworkName() string
}
