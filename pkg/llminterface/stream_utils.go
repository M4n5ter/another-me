package llminterface

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// ErrStreamCanceled 表示流处理被取消
var ErrStreamCanceled = errors.New("stream processing canceled")

// MergeChatOutputChunks 从提供的 channel 中聚合所有 ChatOutputChunk，
// 并返回一个包含所有累积内容部分的单个 ChatOutputChunk。
// 如果在流处理过程中遇到错误，或者上下文被取消，它会返回一个包含该错误的 chunk 和/或错误。
//
// 参数:
//
//	ctx: 用于控制取消操作的上下文。
//	chunkChan: 一个只读的 channel，从中接收 ChatOutputChunk 数据块。
//
// 返回值:
//   - ChatOutputChunk: 一个聚合了所有接收到的数据块内容的单个 ChatOutputChunk。
//     如果发生错误，此结构体的 Error 字段将被设置。
//     FinishReason 将是流中遇到的最后一个非空的 FinishReason。
//   - error: 如果在聚合过程中发生错误（例如上下文取消或 channel 中出现错误），则返回该错误。
//     如果聚合成功完成（即使 channel 中最后一个 chunk 带有错误），也可能返回 nil，
//     此时错误信息会封装在返回的 ChatOutputChunk 的 Error 字段中。
//     通常，当此 error 非 nil 时，表示聚合未正常完成。
func MergeChatOutputChunks(ctx context.Context, chunkChan <-chan ChatOutputChunk) (ChatOutputChunk, error) {
	// 预分配一个合理的初始容量，避免频繁扩容
	accumulatedParts := make([]ContentPart, 0, 16)
	var finalFinishReason Option[string]
	var lastError error
	var mutex sync.Mutex // 保护对共享状态的并发访问

	// 使用 select+for 模式以支持更及时的上下文取消检测
	for {
		select {
		case <-ctx.Done():
			// 上下文被取消
			return ChatOutputChunk{
				ContentParts: accumulatedParts,
				Error:        ctx.Err(),
				FinishReason: finalFinishReason,
			}, fmt.Errorf("%w: %w", ErrStreamCanceled, ctx.Err())

		case chunk, ok := <-chunkChan:
			if !ok {
				// Channel 已关闭，聚合完成
				return ChatOutputChunk{
					ContentParts: accumulatedParts,
					Error:        lastError,
					FinishReason: finalFinishReason,
				}, nil
			}

			// 使用互斥锁保护共享状态的修改
			mutex.Lock()

			// 首先处理内容部分
			if len(chunk.ContentParts) > 0 {
				accumulatedParts = append(accumulatedParts, chunk.ContentParts...)
			}

			// 更新完成原因（如果有）
			if chunk.FinishReason.IsSome() {
				finalFinishReason = chunk.FinishReason
			}

			// 处理错误
			if chunk.Error != nil {
				lastError = chunk.Error

				// 如果是严重错误（非EOF），立即返回但仍保留已处理的内容
				if !errors.Is(chunk.Error, io.EOF) {
					result := ChatOutputChunk{
						ContentParts: accumulatedParts,
						Error:        chunk.Error,
						FinishReason: chunk.FinishReason.Or(finalFinishReason),
					}
					mutex.Unlock()
					return result, chunk.Error
				}
				// EOF 错误视为流的自然结束，继续处理
			}
			mutex.Unlock()

			// 如果收到明确的完成信号，可以提前退出循环
			if chunk.FinishReason.IsSome() && (chunk.FinishReason.Unwrap() == "stop" ||
				chunk.FinishReason.Unwrap() == "length" || chunk.FinishReason.Unwrap() == "content_filter") {
				return ChatOutputChunk{
					ContentParts: accumulatedParts,
					Error:        lastError,
					FinishReason: finalFinishReason,
				}, nil
			}
		}
	}
}

// AggregateTextFromChunks 从提供的 channel 中聚合所有 ChatOutputChunk 的文本内容，
// 并返回一个包含所有文本部分连接的字符串。
// 这是处理 LLM 响应的常见用例。
// 如果遇到错误，返回到那一刻为止累积的文本和第一个遇到的错误。
//
// 参数:
//
//	ctx: 用于控制取消操作的上下文。
//	chunkChan: 一个只读的 channel，从中接收 ChatOutputChunk 数据块。
//	options...: 可选参数，如初始容量预估等。
//
// 返回值:
//   - string: 连接所有文本部分的字符串。如果没有文本部分，返回空字符串。
//   - error: 如果在聚合过程中发生错误（例如上下文取消或 channel 中出现错误），则返回第一个遇到的错误。
//     如果聚合成功完成（没有错误或只有 io.EOF），则返回 nil。
func AggregateTextFromChunks(ctx context.Context, chunkChan <-chan ChatOutputChunk, initialCapacity ...int) (string, error) {
	// 预估初始容量，提高内存使用效率
	capacity := 1024 // 默认初始容量
	if len(initialCapacity) > 0 && initialCapacity[0] > 0 {
		capacity = initialCapacity[0]
	}

	fullTextBuilder := strings.Builder{}
	fullTextBuilder.Grow(capacity) // 预分配内存

	var firstError error
	var mutex sync.Mutex // 保护对共享状态的并发访问

	buffer := make([]byte, 0, 256)

	// 添加文本到builder的辅助函数，处理缓冲区逻辑
	appendToBuilder := func(text string) {
		if len(buffer)+len(text) > cap(buffer) {
			// 缓冲区将溢出，先刷新
			fullTextBuilder.Write(buffer)
			buffer = buffer[:0] // 重置缓冲区
		}
		buffer = append(buffer, text...)
	}

	// 最终刷新缓冲区
	flushBuffer := func() {
		if len(buffer) > 0 {
			fullTextBuilder.Write(buffer)
		}
	}

	for {
		select {
		case <-ctx.Done():
			mutex.Lock()
			flushBuffer() // 确保刷新任何缓冲的内容
			result := fullTextBuilder.String()
			mutex.Unlock()
			return result, fmt.Errorf("%w: %w", ErrStreamCanceled, ctx.Err())

		case chunk, ok := <-chunkChan:
			if !ok { // Channel closed
				mutex.Lock()
				flushBuffer() // 确保刷新任何缓冲的内容
				result := fullTextBuilder.String()
				mutex.Unlock()
				return result, firstError
			}

			mutex.Lock()
			// 处理文本内容
			for _, part := range chunk.ContentParts {
				if part.Type == PartTypeText && part.Text != "" {
					appendToBuilder(part.Text)
				}
			}

			// 处理错误
			if chunk.Error != nil && firstError == nil && !errors.Is(chunk.Error, io.EOF) {
				firstError = chunk.Error
				// 对于严重错误，立即返回已收集的文本
				if !errors.Is(chunk.Error, io.EOF) {
					flushBuffer()
					result := fullTextBuilder.String()
					mutex.Unlock()
					return result, firstError
				}
			}

			// 处理流结束信号
			finishReason := chunk.FinishReason
			if finishReason.IsSome() || errors.Is(chunk.Error, io.EOF) {
				// 对于自然结束，尝试处理可能的剩余数据
				drainDone := false

				// 释放锁以避免死锁，因为我们要从通道读取
				mutex.Unlock()

				// 尝试排空通道，但设置超时以避免无限阻塞
				drainCtx, cancel := context.WithTimeout(context.Background(), defaultDrainTimeout)
				defer cancel()

				drainChan := make(chan struct{})
				go func() {
					defer close(drainChan)

					for {
						select {
						case <-drainCtx.Done():
							return
						case c, ok := <-chunkChan:
							if !ok {
								return
							}

							mutex.Lock()
							// 继续处理任何剩余文本
							for _, p := range c.ContentParts {
								if p.Type == PartTypeText && p.Text != "" {
									appendToBuilder(p.Text)
								}
							}

							// 记录第一个非EOF错误
							if c.Error != nil && firstError == nil && !errors.Is(c.Error, io.EOF) {
								firstError = c.Error
							}
							mutex.Unlock()
						}
					}
				}()

				// 等待排空完成或超时
				select {
				case <-drainChan:
					drainDone = true
				case <-drainCtx.Done():
					// 排空超时，这是正常的
				}

				mutex.Lock()
				flushBuffer()
				result := fullTextBuilder.String()
				mutex.Unlock()

				// 如果成功排空或者触发了超时，返回当前结果
				if drainDone || drainCtx.Err() != nil {
					return result, firstError
				}

				// 如果没有排空完成也没有超时，继续常规处理
				mutex.Lock()
			}
			mutex.Unlock()
		}
	}
}

// 当尝试排空通道时使用的默认超时时间
const defaultDrainTimeout = 100 * 1000000 // 100ms in nanoseconds
