package reactagent

import (
	"context"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

// ReActAgent 代表一个 ReAct 智能体
type ReActAgent struct {
	ReAct
	Capabilities []string
}

// NewReActAgent 创建一个 ReAct 智能体
func NewReActAgent(react ReAct, capabilities []string) *ReActAgent {
	return &ReActAgent{
		ReAct:        react,
		Capabilities: capabilities,
	}
}

var _ ReAct = (*ReActAgent)(nil)

// AgentOutputChunk 代表 ReAct Agent 流式输出的一个数据块。
// 注意：工具调用的具体结果不会通过此通道流式传输，它们会被合并到内部消息历史中，
// LLM 对工具结果的后续思考（文本）才会通过此通道流式输出。
type AgentOutputChunk struct {
	Type AgentOutputChunkType `json:"type"` // 块类型

	// 当 Type 为 AgentChunkTypeText 时，此字段包含流式文本片段。
	// 对于其他类型，此字段通常为空。
	TextDelta string `json:"text_delta,omitempty"`

	// 当 Type 为 AgentChunkTypeToolStart 或 AgentChunkTypeToolEnd 时，包含工具调用ID。
	ToolCallID string `json:"tool_call_id,omitempty"`

	// 当 Type 为 AgentChunkTypeToolStart 或 AgentChunkTypeToolEnd 时，包含工具名称。
	ToolName string `json:"tool_name,omitempty"`

	// 当 Type 为 AgentChunkTypeToolStart 或 AgentChunkTypeToolEnd 时，包含工具调用参数。
	ToolArguments string `json:"tool_arguments,omitempty"`

	// 当 Type 为 AgentChunkTypeToolStart 或 AgentChunkTypeToolEnd 时，包含工具调用结果。
	ToolResult string `json:"tool_result,omitempty"`

	// 当 Type 为 AgentChunkTypeThought 时，此字段包含当前迭代LLM的完整思考文本。
	CurrentIterThoughtContent string `json:"thought_content,omitempty"`

	// 当 Type 为 AgentChunkTypeReasoning 时，此字段包含LLM的推理内容。
	ReasoningContent string `json:"reasoning_content,omitempty"`

	// 包含所有迭代过程中LLM的累积思考内容，只在最后一个chunk中设置。
	AccumulatedThoughts string `json:"accumulated_thoughts,omitempty"`

	// 完整的对话历史记录，包含所有消息，只在最后一个chunk中设置。
	// 可用于保存当前会话状态，以便未来继续。
	MessageHistory *llminterface.ChatInput `json:"message_history,omitempty"`

	// 当 Type 为 AgentChunkTypeError 时，此字段包含错误信息。
	// 对于 AgentChunkTypeMaxIter，也可能用此字段传递相关消息。
	Error string `json:"error,omitempty"`

	// 当 Type 为 AgentChunkTypeFinish 时，此字段可能包含最终的响应文本。
	FinalResponse string `json:"final_response,omitempty"`

	// 指示这是否是整个 ReAct 序列的最后一个块。
	// 对于 AgentChunkTypeFinish, AgentChunkTypeError, AgentChunkTypeMaxIter 应该为 true。
	IsLast bool `json:"is_last"`
}

// AgentOutputChunkType 定义了 Agent 输出块的类型。
type AgentOutputChunkType string

const (
	// AgentChunkTypeReasoning 推理内容
	AgentChunkTypeReasoning AgentOutputChunkType = "reasoning"
	// AgentChunkTypeText 纯文本输出
	AgentChunkTypeText AgentOutputChunkType = "text"
	// AgentChunkTypeToolStart 工具开始执行的信号
	AgentChunkTypeToolStart AgentOutputChunkType = "tool_start"
	// AgentChunkTypeToolEnd 工具结束执行的信号 (可选, 结果不从此输出)
	AgentChunkTypeToolEnd AgentOutputChunkType = "tool_end"
	// AgentChunkTypeError 错误输出
	AgentChunkTypeError AgentOutputChunkType = "error"
	// AgentChunkTypeThought LLM的完整思考步骤文本 (在决定工具调用前或最终回复前)
	AgentChunkTypeThought AgentOutputChunkType = "thought"
	// AgentChunkTypeFinish ReAct 序列正常结束
	AgentChunkTypeFinish AgentOutputChunkType = "finish"
	// AgentChunkTypeMaxIter 达到最大迭代次数
	AgentChunkTypeMaxIter AgentOutputChunkType = "max_iterations"
)

// Agent 接口定义了一个通用的智能体
type Agent interface {
	// Run 执行智能体的核心逻辑
	Run(ctx context.Context, userInput, conversationID string) (<-chan AgentOutputChunk, error)

	// ContinueFromCheckpoint 从某个检查点继续执行
	ContinueFromCheckpoint(ctx context.Context, userInput string, checkpoint *llminterface.ChatInput) (<-chan AgentOutputChunk, error)
}
