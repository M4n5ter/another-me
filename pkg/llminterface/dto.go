package llminterface

import (
	. "github.com/m4n5ter/another-me/pkg/option"
)

// ContentPartType 定义了消息中单个内容部分的类型。
type ContentPartType string

// ContentPartType 的常量定义
const (
	PartTypeText     ContentPartType = "text"      // 文本类型的内容部分
	PartTypeImageURL ContentPartType = "image_url" // 图片 URL 类型的内容部分
)

// ImageURLDetail 定义了图像处理的细节级别。
// 这通常对应于 LLM 提供商对图像输入的 "detail" 参数。
type ImageURLDetail string

// ImageURLDetail 的常量定义
const (
	ImageDetailLow  ImageURLDetail = "low"  // 低细节，处理速度快，但可能丢失信息
	ImageDetailHigh ImageURLDetail = "high" // 高细节，处理更精细，但可能更慢或消耗更多资源
	ImageDetailAuto ImageURLDetail = "auto" // 自动决定细节级别，通常是提供商的默认行为
)

// ImageURLContent 代表通过 URL 提供的图像输入。
// URL 可以是标准的 HTTP/HTTPS 链接，也可以是 Base64 编码的 Data URI。
type ImageURLContent struct {
	URL    string                 `json:"url"`              // 图像的 URL (例如 "http://..." 或 "data:image/jpeg;base64,...")
	Detail Option[ImageURLDetail] `json:"detail,omitempty"` // 可选：图像处理的细节级别
}

// ContentPart 代表多模态消息中的一个独立部分，例如一段文本或一个图像。
type ContentPart struct {
	Type     ContentPartType         `json:"type"`                // 内容部分的类型 (例如 "text", "image_url")
	Text     string                  `json:"text,omitempty"`      // 当 Type 为 PartTypeText 时，此字段包含文本内容。
	ImageURL Option[ImageURLContent] `json:"image_url,omitempty"` // 当 Type 为 PartTypeImageURL 时，此字段包含图像 URL 的详细信息。
}

// MessageRole 定义了消息在对话历史中的角色。
type MessageRole string

// MessageRole 的常量定义
const (
	RoleUser       MessageRole = "user"        // 用户发出的消息
	RoleAssistant  MessageRole = "assistant"   // AI 助手或 LLM 发出的消息，用于构建对话历史
	RoleSystem     MessageRole = "system"      // 系统级指令，通常用于指导 LLM 的整体行为，具体支持情况取决于底层框架
	RoleToolResult MessageRole = "tool_result" // 工具调用结果消息，用于返回工具调用的结果
)

// InputMessage 代表发送给 LLM 的单条消息结构，可以包含多个内容部分。
type InputMessage struct {
	Role    MessageRole   `json:"role"`    // 消息的角色 (例如 "user", "assistant")
	Content []ContentPart `json:"content"` // 消息的内容，可以是文本和图像的混合序列
}

// ChatInput 代表对 LLM 进行一次交互的完整输入。
type ChatInput struct {
	Messages       []InputMessage `json:"messages"` // 要发送给 LLM 的消息序列
	ConversationID string         `json:"-"`        // 会话ID，用于适配器内部追踪和管理会话状态 (例如关联到框架的会话记忆)，不直接发送给LLM。
	// 注意：模型名称 (Model), 最大令牌数 (MaxTokens), 温度 (Temperature) 等参数
	//      预计在具体的 ChatAdapter 实现中进行配置，或者通过其所适配的框架的原生机制进行设置，
	//      而不是在这个通用的 ChatInput 结构中定义。
}

// ChatOutputChunk 代表 LLM 响应流中的一个数据块。
// 对于非流式调用，调用者需要聚合所有块的 TextDelta 来获得完整响应。
type ChatOutputChunk struct {
	TextDelta    string         `json:"text_delta"`              // 此数据块中增量生成的文本部分。
	Error        error          `json:"-"`                       // 在处理此数据块或终止流时发生的错误。如果非nil，则流在此处被视为失败。
	FinishReason Option[string] `json:"finish_reason,omitempty"` // 流结束的原因 (例如 "stop", "length")。通常在指示流终止的最后一个块中出现。
}
