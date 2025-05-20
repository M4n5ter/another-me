package common

// ForceWriteToChannel 向通道中写入数据，如果通道满了，则丢弃通道开头的数据并写入新数据
func ForceWriteToChannel[T any](ch chan T, data T) {
	select {
	case ch <- data:
	default:
		select {
		case <-ch: // 从通道读取并丢弃一个元素
			// 然后尝试写入新数据
			select {
			case ch <- data:
			default:
				// 极少数情况：可能在我们丢弃后又有其他goroutine写入
				// 递归调用确保写入成功
				ForceWriteToChannel(ch, data)
			}
		default:
		}
	}
}
