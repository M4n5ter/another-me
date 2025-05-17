package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

// PredefinedChatResponse 定义了对 Chat 方法一次调用的预设响应。
type PredefinedChatResponse struct {
	ChunksToReturn []llminterface.ChatOutputChunk // 当次 Chat 调用成功时，要在 channel 上发送的数据块序列。
	ErrorToReturn  error                          // 当次 Chat 调用需要直接返回的初始错误。如果非 nil，则 ChunksToReturn 通常为空。
}

// MockChatAdapter 是 llminterface.ChatAdapter 的一个 mock 实现，主要用于测试。
// 它可以配置为返回预定义的响应，或者通过自定义函数来模拟 Chat 方法的行为。
type MockChatAdapter struct {
	mu sync.RWMutex // mu 用于保护对 mock 内部状态的并发访问，例如调用计数和预设响应队列。

	// --- GetFrameworkName 的配置 ---
	FrameworkNameResult string // GetFrameworkName 方法将返回此字符串。

	// --- Chat 方法的配置 ---
	// ChatFunc 是一个可选的自定义函数，用于完全控制 Chat 方法的行为。
	// 如果设置了此函数，则 Chat 方法会直接调用它。
	// 这对于复杂的测试场景非常有用。
	ChatFunc func(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error)

	// PredefinedChatResponses 存储一个预定义响应的队列。
	// 每次调用 Chat 方法时（如果 ChatFunc 未设置），会从队列头部取出一个响应来模拟行为。
	// 当队列为空时，后续的 Chat 调用可能会返回错误或 panic，具体取决于测试需求（当前实现为错误）。
	predefinedChatResponses []*PredefinedChatResponse
	chatCallCount           int // 记录 Chat 方法被调用的次数，用于按顺序提供预定义响应。

	// --- 调用记录 ---
	// RecordedChatInputs 记录了每次调用 Chat 方法时接收到的 llminterface.ChatInput。
	// 这对于断言 Chat 方法是否以预期的参数被调用非常有用。
	RecordedChatInputs []llminterface.ChatInput
}

// NewMockChatAdapter 创建并返回一个新的 MockChatAdapter 实例。
func NewMockChatAdapter() *MockChatAdapter {
	return &MockChatAdapter{
		RecordedChatInputs:      make([]llminterface.ChatInput, 0),
		predefinedChatResponses: make([]*PredefinedChatResponse, 0),
	}
}

// --- 接口实现: GetFrameworkName ---

// GetFrameworkName 返回预设的框架名称 (FrameworkNameResult)。
func (m *MockChatAdapter) GetFrameworkName() string {
	m.mu.RLock() // 使用读锁，因为 FrameworkNameResult 通常是只读的
	defer m.mu.RUnlock()
	return m.FrameworkNameResult
}

// --- 接口实现: Chat ---

// Chat 实现了 llminterface.ChatAdapter 接口的 Chat 方法。
// 它会记录调用参数，并根据预设的 ChatFunc 或 PredefinedChatResponses 返回响应。
func (m *MockChatAdapter) Chat(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
	m.mu.Lock() // 写锁保护 RecordedChatInputs 和 chatCallCount 的修改
	// 记录调用
	m.RecordedChatInputs = append(m.RecordedChatInputs, input)
	callNum := m.chatCallCount
	m.chatCallCount++
	m.mu.Unlock()

	// 如果 ChatFunc 已设置，则优先使用它
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, input)
	}

	// 否则，使用预定义的响应队列
	m.mu.RLock() // 读锁保护 predefinedChatResponses 的读取

	if callNum < len(m.predefinedChatResponses) {
		response := m.predefinedChatResponses[callNum]
		m.mu.RUnlock() // 尽快释放读锁

		if response.ErrorToReturn != nil {
			return nil, response.ErrorToReturn
		}

		// 创建并填充 channel
		outChan := make(chan llminterface.ChatOutputChunk, len(response.ChunksToReturn)) // 缓冲 channel
		go func() {
			defer close(outChan) // 确保 goroutine 结束时关闭 channel
			for _, chunk := range response.ChunksToReturn {
				// 检查上下文是否已取消，以允许提前终止流
				select {
				case <-ctx.Done():
					outChan <- llminterface.ChatOutputChunk{Error: ctx.Err()} // 发送上下文错误
					return                                                    // 终止 goroutine
				case outChan <- chunk: // 发送正常的 chunk
				}
			}
		}()
		return outChan, nil
	}
	m.mu.RUnlock() // 如果没有进入 if, 在这里释放读锁

	// 如果没有预定义的响应，并且 ChatFunc 也未设置，则返回错误
	// chatCallCount 在这里已经是递增后的值，所以用它来报告更准确
	return nil, fmt.Errorf("MockChatAdapter: Chat() called %d time(s), but no predefined response or ChatFunc was set for this call", m.GetChatCallCount()) // 使用 GetChatCallCount 以确保线程安全地读取
}

// --- Mock 辅助方法 ---

// SetFrameworkName 设置 GetFrameworkName 方法的返回值。
func (m *MockChatAdapter) SetFrameworkName(name string) {
	m.mu.Lock() // 写锁
	defer m.mu.Unlock()
	m.FrameworkNameResult = name
}

// AddPredefinedChatResponse 向预定义响应队列中添加一个响应。
// 每次调用 Chat 方法（且 ChatFunc 未设置时）会按顺序使用这些响应。
// chunks: 如果调用成功，将在返回的 channel 上发送的 ChatOutputChunk 序列。
// initialErr: 如果希望 Chat 方法直接返回错误，则设置此参数；否则设为 nil。
func (m *MockChatAdapter) AddPredefinedChatResponse(chunks []llminterface.ChatOutputChunk, initialErr error) {
	m.mu.Lock() // 写锁
	defer m.mu.Unlock()
	m.predefinedChatResponses = append(m.predefinedChatResponses, &PredefinedChatResponse{
		ChunksToReturn: chunks,
		ErrorToReturn:  initialErr,
	})
}

// ClearPredefinedChatResponses 清空所有预定义的聊天响应并重置调用计数。
func (m *MockChatAdapter) ClearPredefinedChatResponses() {
	m.mu.Lock() // 写锁
	defer m.mu.Unlock()
	m.predefinedChatResponses = make([]*PredefinedChatResponse, 0)
	m.chatCallCount = 0
}

// ClearRecordedChatInputs 清空所有已记录的 Chat 输入。
func (m *MockChatAdapter) ClearRecordedChatInputs() {
	m.mu.Lock() // 写锁
	defer m.mu.Unlock()
	m.RecordedChatInputs = make([]llminterface.ChatInput, 0)
}

// GetChatCallCount 返回 Chat 方法被调用的总次数。
func (m *MockChatAdapter) GetChatCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chatCallCount
}

// GetLastRecordedChatInput 返回最后一次调用 Chat 方法时传入的参数。
// 如果 Chat 从未被调用，则第二个返回值为 false。
func (m *MockChatAdapter) GetLastRecordedChatInput() (llminterface.ChatInput, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.RecordedChatInputs) == 0 {
		return llminterface.ChatInput{}, false
	}
	return m.RecordedChatInputs[len(m.RecordedChatInputs)-1], true
}

// GetRecordedChatInputAt 返回第 n 次（从0开始计数）调用 Chat 方法时传入的参数。
// 如果指定的调用次数无效（例如 Chat 调用次数不足），则第二个返回值为 false。
func (m *MockChatAdapter) GetRecordedChatInputAt(index int) (llminterface.ChatInput, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if index < 0 || index >= len(m.RecordedChatInputs) {
		return llminterface.ChatInput{}, false
	}
	return m.RecordedChatInputs[index], true
}
