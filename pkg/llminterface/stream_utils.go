package llminterface

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	. "github.com/m4n5ter/another-me/pkg/option"
)

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
	var accumulatedParts []ContentPart
	var finalFinishReason Option[string]
	var lastError error

	for {
		select {
		case <-ctx.Done():
			// 上下文被取消
			return ChatOutputChunk{
				ContentParts: accumulatedParts,
				Error:        ctx.Err(),
				FinishReason: finalFinishReason, // 保留可能已收到的 FinishReason
			}, fmt.Errorf("context canceled: %w", ctx.Err())
		case chunk, ok := <-chunkChan:
			if !ok {
				// Channel 已关闭，聚合完成
				return ChatOutputChunk{
					ContentParts: accumulatedParts,
					Error:        lastError, // 如果在关闭前最后一个块有错误，也记录下来
					FinishReason: finalFinishReason,
				}, nil // Channel 正常关闭，即使最后一个块有错误，聚合本身是"完成"的
			}

			// 首先处理内容部分，即使块有错误，也要先将其内容添加到累积结果中
			if len(chunk.ContentParts) > 0 {
				accumulatedParts = append(accumulatedParts, chunk.ContentParts...)
			}

			if chunk.FinishReason.IsSome() {
				finalFinishReason = chunk.FinishReason
			}

			// 然后检查块内错误
			if chunk.Error != nil {
				// 如果块内有错误，记录下来。
				// 如果错误是 io.EOF，这通常表示流的正常结束，类似于 channel 关闭。
				// 其他错误则更严重。
				lastError = chunk.Error
				if !errors.Is(chunk.Error, io.EOF) {
					// 对于非 EOF 错误，我们立即停止并返回，但包含该 chunk 的内容
					return ChatOutputChunk{
						ContentParts: accumulatedParts, // 返回已累积的部分，包括当前 chunk 的内容
						Error:        chunk.Error,
						FinishReason: chunk.FinishReason.Or(finalFinishReason), // 使用当前 chunk 或之前记录的
					}, chunk.Error
				}
				// 如果是 io.EOF，则行为类似于 channel 关闭，我们将在下一次迭代或 select case 中处理。
				// 实际上，通常 io.EOF 错误后，channel 会很快关闭。
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
//
// 返回值:
//   - string: 连接所有文本部分的字符串。如果没有文本部分，返回空字符串。
//   - error: 如果在聚合过程中发生错误（例如上下文取消或 channel 中出现错误），则返回第一个遇到的错误。
//     如果聚合成功完成（没有错误或只有 io.EOF），则返回 nil。
func AggregateTextFromChunks(ctx context.Context, chunkChan <-chan ChatOutputChunk) (string, error) {
	var fullTextBuilder strings.Builder
	var firstError error
	var finalFinishReason Option[string]

	for {
		select {
		case <-ctx.Done():
			return fullTextBuilder.String(), ctx.Err()
		case chunk, ok := <-chunkChan:
			if !ok { // Channel closed
				return fullTextBuilder.String(), firstError
			}

			// 首先处理文本内容
			for _, part := range chunk.ContentParts {
				if part.Type == PartTypeText {
					fullTextBuilder.WriteString(part.Text)
				}
			}

			if chunk.FinishReason.IsSome() {
				finalFinishReason = chunk.FinishReason
			}

			// 然后处理错误
			if chunk.Error != nil {
				if firstError == nil && !errors.Is(chunk.Error, io.EOF) { // Store the first non-EOF error
					firstError = chunk.Error
				}
				if !errors.Is(chunk.Error, io.EOF) { // If a significant error occurs, stop processing
					return fullTextBuilder.String(), firstError
				}
				// If io.EOF, continue to drain any remaining text and then exit.
			}

			// If we got an EOF error, or a finish reason, and the channel is not yet closed,
			// we assume the stream is done.
			if errors.Is(chunk.Error, io.EOF) || finalFinishReason.IsSome() {
				// Drain any final messages if channel isn't closed yet by mistake.
				// This is a bit defensive.
				for c := range chunkChan {
					if c.Error != nil && firstError == nil && !errors.Is(c.Error, io.EOF) {
						firstError = c.Error
					}
					for _, p := range c.ContentParts {
						if p.Type == PartTypeText {
							fullTextBuilder.WriteString(p.Text)
						}
					}
				}
				return fullTextBuilder.String(), firstError
			}
		}
	}
}
