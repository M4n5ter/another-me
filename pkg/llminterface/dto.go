package llminterface

import (
	. "github.com/m4n5ter/another-me/pkg/option"
)

// ChatInput 代表对 LLM 进行一次交互的完整输入。
type ChatInput struct {
	Messages       []InputMessage `json:"messages"` // 要发送给 LLM 的消息序列
	ConversationID string         `json:"-"`        // 会话ID，用于适配器内部追踪和管理会话状态 (例如关联到框架的会话记忆)，不直接发送给LLM。
	// 注意：模型名称 (Model), 最大令牌数 (MaxTokens), 温度 (Temperature) 等参数
	//      预计在具体的 ChatAdapter 实现中进行配置，或者通过其所适配的框架的原生机制进行设置，
	//      而不是在这个通用的 ChatInput 结构中定义。
}

// InputMessage 代表发送给 LLM 的单条消息结构，可以包含多个内容部分。
type InputMessage struct {
	Role       MessageRole    `json:"role"`                   // 消息的角色 (例如 "user", "assistant")
	Content    []ContentPart  `json:"content"`                // 消息的内容，可以是文本和图像的混合序列
	ToolCallID Option[string] `json:"tool_call_id,omitempty"` // 可选：当 Role 为 RoleToolResult 时，此字段标识此消息响应的是哪个工具调用。
	ToolName   Option[string] `json:"tool_name,omitempty"`    // 可选：当 Role 为 RoleToolResult 时，此字段指示执行的是哪个工具的名称。
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

// ContentPart 代表多模态消息中的一个独立部分，例如一段文本、一个图像或一个工具调用请求。
type ContentPart struct {
	Type           ContentPartType         `json:"type"`                       // 内容部分的类型 (例如 "text", "image_url", "tool_call_request")
	Text           string                  `json:"text,omitempty"`             // 当 Type 为 PartTypeText 时，此字段包含文本内容。
	ImageURL       Option[ImageURLContent] `json:"image_url,omitempty"`        // 当 Type 为 PartTypeImageURL 时，此字段包含图像 URL 的详细信息。
	ToolCallValues Option[ToolCallContent] `json:"tool_call_values,omitempty"` // 当 Type 为 PartTypeToolCallRequest 时，此字段包含工具调用的结构化信息。
}

// ContentPartType 定义了消息中单个内容部分的类型。
type ContentPartType string

// ContentPartType 的常量定义
const (
	PartTypeText            ContentPartType = "text"              // 文本类型的内容部分
	PartTypeImageURL        ContentPartType = "image_url"         // 图片 URL 类型的内容部分
	PartTypeToolCallRequest ContentPartType = "tool_call_request" // LLM发起的工具调用请求
)

// ImageURLContent 代表通过 URL 提供的图像输入。
// URL 可以是标准的 HTTP/HTTPS 链接，也可以是 Base64 编码的 Data URI。
type ImageURLContent struct {
	URL    string                 `json:"url"`              // 图像的 URL (例如 "http://..." 或 "data:image/jpeg;base64,...")
	Detail Option[ImageURLDetail] `json:"detail,omitempty"` // 可选：图像处理的细节级别
}

// ImageURLDetail 定义了图像处理的细节级别。
// 这通常对应于 LLM 提供商对图像输入的 "detail" 参数。
type ImageURLDetail string

// ImageURLDetail 的常量定义
const (
	ImageDetailLow  ImageURLDetail = "low"  // 低细节，处理速度快，但可能丢失信息
	ImageDetailHigh ImageURLDetail = "high" // 高细节，处理更精细，但可能更慢或消耗更多资源
	ImageDetailAuto ImageURLDetail = "auto" // 自动决定细节级别，通常是提供商的默认行为
)

// ToolCallContent 代表一个或多个工具调用请求。
type ToolCallContent struct {
	Calls []ToolCall `json:"calls"` // 工具调用列表
}

// ToolCall 代表一个对特定工具的调用请求。
type ToolCall struct {
	ID        string `json:"id"`        // 工具调用的唯一标识符，用于跟踪和匹配工具的调用和响应
	Name      string `json:"name"`      // 工具的名称，应与已注册的工具名称一致
	Arguments string `json:"arguments"` // 工具的参数，通常是一个JSON字符串
}

// ChatOutputChunk 代表 LLM 响应流中的一个数据块。
// 对于非流式调用，调用者需要聚合所有块以获得完整响应。
type ChatOutputChunk struct {
	// ContentParts 包含此数据块中的一个或多个内容部分 (例如文本、图像URL)。
	// 对于纯文本流，通常这里只有一个类型为 PartTypeText 的 ContentPart。
	// 对于多模态输出或需要发送结构化数据（如工具调用信息，如果以内容形式表示）时，
	// 这里可以包含多个或不同类型的 ContentPart。
	ContentParts []ContentPart  `json:"content_parts,omitempty"`
	Error        error          `json:"-"`                       // 在处理此数据块或终止流时发生的错误。如果非nil，则流在此处被视为失败。
	FinishReason Option[string] `json:"finish_reason,omitempty"` // 流结束的原因 (例如 "stop", "length")。通常在指示流终止的最后一个块中出现。
}

// LLMResponse 表示从LLM获取的完整响应
// 它代表了所有ChatOutputChunk合并后的最终结果
type LLMResponse struct {
	// Role 响应的角色，默认为assistant
	Role MessageRole `json:"role,omitempty"`

	// Content 包含合并后的完整内容，按照接收顺序组织
	Content []ContentPart `json:"content,omitempty"`

	// FullText 是所有文本内容的快捷访问方式
	// 它将所有PartTypeText类型的ContentPart合并成一个字符串
	FullText string `json:"full_text,omitempty"`

	// Error 如果在获取响应过程中发生错误，此字段将包含该错误
	Error error `json:"-"`

	// FinishReason 表示LLM生成停止的原因
	FinishReason Option[string] `json:"finish_reason,omitempty"`
}

// HasToolCalls 返回此响应是否包含工具调用请求
func (r *LLMResponse) HasToolCalls() bool {
	for _, part := range r.Content {
		if part.Type == PartTypeToolCallRequest && part.ToolCallValues.IsSome() {
			return true
		}
	}
	return false
}

// GetToolCalls 提取所有工具调用请求
// 如果响应中不包含工具调用，则返回空切片
func (r *LLMResponse) GetToolCalls() []ToolCall {
	var allCalls []ToolCall
	for _, part := range r.Content {
		if part.Type == PartTypeToolCallRequest && part.ToolCallValues.IsSome() {
			allCalls = append(allCalls, part.ToolCallValues.Unwrap().Calls...)
		}
	}
	return allCalls
}

// ToInputMessage 将LLMResponse转换为可直接添加到对话历史中的InputMessage
func (r *LLMResponse) ToInputMessage() InputMessage {
	role := r.Role
	if role == "" {
		role = RoleAssistant
	}

	return InputMessage{
		Role:    role,
		Content: r.Content,
	}
}

// ToUserMessage 创建一个包含该响应文本内容的用户消息
// 用于在需要用户视角时快速创建用户消息
func (r *LLMResponse) ToUserMessage() InputMessage {
	content := make([]ContentPart, 0, 1)
	if r.FullText != "" {
		content = append(content, ContentPart{
			Type: PartTypeText,
			Text: r.FullText,
		})
	} else {
		// 只复制文本部分
		for _, part := range r.Content {
			if part.Type == PartTypeText {
				content = append(content, part)
			}
		}
	}

	return InputMessage{
		Role:    RoleUser,
		Content: content,
	}
}
