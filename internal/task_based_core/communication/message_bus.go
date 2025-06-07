package communication

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// MessageBus 消息总线
type MessageBus struct {
	mu     sync.RWMutex
	logger *slog.Logger

	// 订阅管理
	subscribers map[EventType][]EventHandler // 事件类型 -> 处理器列表
	eventChan   chan Event                   // 事件通道

	// 生命周期管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 配置
	bufferSize int // 事件缓冲区大小
	maxWorkers int // 最大处理协程数

	// 统计信息
	stats BusStats
}

// EventHandler 事件处理器
type EventHandler func(event Event)

// BusStats 消息总线统计信息
type BusStats struct {
	mu                sync.RWMutex
	TotalEvents       int64         `json:"total_events"`       // 总事件数
	ProcessedEvents   int64         `json:"processed_events"`   // 已处理事件数
	FailedEvents      int64         `json:"failed_events"`      // 失败事件数
	ActiveSubscribers int           `json:"active_subscribers"` // 活跃订阅者数
	AverageLatency    time.Duration `json:"average_latency"`    // 平均延迟
}

// NewMessageBus 创建新的消息总线
func NewMessageBus(bufferSize, maxWorkers int) *MessageBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &MessageBus{
		logger:      slog.Default().WithGroup("message_bus"),
		subscribers: make(map[EventType][]EventHandler),
		eventChan:   make(chan Event, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
		bufferSize:  bufferSize,
		maxWorkers:  maxWorkers,
	}

	// 启动事件处理协程
	bus.start()

	return bus
}

// Subscribe 订阅事件
func (bus *MessageBus) Subscribe(eventType EventType, handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.subscribers[eventType] == nil {
		bus.subscribers[eventType] = make([]EventHandler, 0)
	}

	bus.subscribers[eventType] = append(bus.subscribers[eventType], handler)

	// 更新统计信息
	bus.stats.mu.Lock()
	bus.stats.ActiveSubscribers++
	bus.stats.mu.Unlock()

	bus.logger.Info("订阅事件",
		"event_type", string(eventType),
		"total_subscribers", len(bus.subscribers[eventType]))
}

// Unsubscribe 取消订阅事件（移除所有该类型的处理器）
func (bus *MessageBus) Unsubscribe(eventType EventType) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if handlers, exists := bus.subscribers[eventType]; exists {
		// 更新统计信息
		bus.stats.mu.Lock()
		bus.stats.ActiveSubscribers -= len(handlers)
		bus.stats.mu.Unlock()

		delete(bus.subscribers, eventType)

		bus.logger.Info("取消订阅事件",
			"event_type", string(eventType),
			"removed_handlers", len(handlers))
	}
}

// Publish 发布事件
func (bus *MessageBus) Publish(event Event) error {
	select {
	case bus.eventChan <- event:
		// 更新统计信息
		bus.stats.mu.Lock()
		bus.stats.TotalEvents++
		bus.stats.mu.Unlock()

		bus.logger.Debug("事件发布",
			"event_id", event.EventID(),
			"event_type", string(event.EventType()),
			"source", event.Source())

		return nil
	case <-bus.ctx.Done():
		return context.Canceled
	default:
		// 通道已满，记录错误但不阻塞
		bus.logger.Warn("事件通道已满，丢弃事件",
			"event_id", event.EventID(),
			"event_type", string(event.EventType()))

		bus.stats.mu.Lock()
		bus.stats.FailedEvents++
		bus.stats.mu.Unlock()

		return ErrChannelFull
	}
}

// PublishSync 同步发布事件（阻塞直到处理完成）
func (bus *MessageBus) PublishSync(event Event) error {
	startTime := time.Now()

	select {
	case bus.eventChan <- event:
		// 等待事件处理完成（简化版本）
		time.Sleep(time.Millisecond) // 给处理器一点时间

		// 更新统计信息
		latency := time.Since(startTime)
		bus.stats.mu.Lock()
		bus.stats.TotalEvents++
		bus.stats.ProcessedEvents++

		// 计算平均延迟
		if bus.stats.AverageLatency == 0 {
			bus.stats.AverageLatency = latency
		} else {
			bus.stats.AverageLatency = (bus.stats.AverageLatency + latency) / 2
		}
		bus.stats.mu.Unlock()

		return nil
	case <-bus.ctx.Done():
		return context.Canceled
	}
}

// PublishWithPriority 带优先级发布事件
func (bus *MessageBus) PublishWithPriority(event Event, priority EventPriority) error {
	priorityEvent := &PriorityEvent{
		Event:    event,
		Priority: priority,
	}

	return bus.Publish(priorityEvent)
}

// GetStats 获取统计信息
func (bus *MessageBus) GetStats() BusStats {
	bus.stats.mu.RLock()
	defer bus.stats.mu.RUnlock()

	// 返回副本
	return BusStats{
		TotalEvents:       bus.stats.TotalEvents,
		ProcessedEvents:   bus.stats.ProcessedEvents,
		FailedEvents:      bus.stats.FailedEvents,
		ActiveSubscribers: bus.stats.ActiveSubscribers,
		AverageLatency:    bus.stats.AverageLatency,
	}
}

// Close 关闭消息总线
func (bus *MessageBus) Close() {
	bus.logger.Info("关闭消息总线")

	// 取消context
	bus.cancel()

	// 等待所有协程结束
	bus.wg.Wait()

	// 关闭事件通道
	close(bus.eventChan)

	bus.logger.Info("消息总线已关闭")
}

// IsRunning 检查消息总线是否正在运行
func (bus *MessageBus) IsRunning() bool {
	select {
	case <-bus.ctx.Done():
		return false
	default:
		return true
	}
}

// 内部方法

// start 启动事件处理协程
func (bus *MessageBus) start() {
	bus.logger.Info("启动消息总线",
		"buffer_size", bus.bufferSize,
		"max_workers", bus.maxWorkers)

	// 启动多个工作协程处理事件
	for i := 0; i < bus.maxWorkers; i++ {
		bus.wg.Add(1)
		go bus.eventWorker(i)
	}
}

// eventWorker 事件处理工作协程
func (bus *MessageBus) eventWorker(workerID int) {
	defer bus.wg.Done()

	bus.logger.Debug("事件处理协程启动", "worker_id", workerID)

	for {
		select {
		case event, ok := <-bus.eventChan:
			if !ok {
				bus.logger.Debug("事件通道已关闭", "worker_id", workerID)
				return
			}

			bus.processEvent(event, workerID)

		case <-bus.ctx.Done():
			bus.logger.Debug("事件处理协程退出", "worker_id", workerID)
			return
		}
	}
}

// processEvent 处理单个事件
func (bus *MessageBus) processEvent(event Event, workerID int) {
	startTime := time.Now()
	eventType := event.EventType()

	bus.logger.Debug("处理事件",
		"worker_id", workerID,
		"event_id", event.EventID(),
		"event_type", string(eventType))

	// 获取订阅者（读锁）
	bus.mu.RLock()
	handlers, exists := bus.subscribers[eventType]
	if !exists {
		bus.mu.RUnlock()
		bus.logger.Debug("没有订阅者", "event_type", string(eventType))
		return
	}

	// 复制处理器列表，避免长时间持有锁
	handlersCopy := make([]EventHandler, len(handlers))
	copy(handlersCopy, handlers)
	bus.mu.RUnlock()

	// 并发处理所有处理器
	var wg sync.WaitGroup
	for _, handler := range handlersCopy {
		wg.Add(1)
		go func(h EventHandler) {
			defer wg.Done()
			defer bus.recoverFromPanic(event.EventID())

			h(event)
		}(handler)
	}

	// 等待所有处理器完成
	wg.Wait()

	// 更新统计信息
	processingTime := time.Since(startTime)
	bus.stats.mu.Lock()
	bus.stats.ProcessedEvents++

	// 更新平均延迟
	if bus.stats.AverageLatency == 0 {
		bus.stats.AverageLatency = processingTime
	} else {
		bus.stats.AverageLatency = (bus.stats.AverageLatency + processingTime) / 2
	}
	bus.stats.mu.Unlock()

	bus.logger.Debug("事件处理完成",
		"worker_id", workerID,
		"event_id", event.EventID(),
		"processing_time", processingTime,
		"handlers_count", len(handlersCopy))
}

// recoverFromPanic 从panic中恢复
func (bus *MessageBus) recoverFromPanic(eventID string) {
	if r := recover(); r != nil {
		bus.logger.Error("事件处理器发生panic",
			"event_id", eventID,
			"error", r)

		// 更新失败统计
		bus.stats.mu.Lock()
		bus.stats.FailedEvents++
		bus.stats.mu.Unlock()
	}
}

// 错误定义
var (
	ErrChannelFull = fmt.Errorf("事件通道已满")
	ErrBusClosed   = fmt.Errorf("消息总线已关闭")
)

// MessageBusInterface 消息总线接口
type MessageBusInterface interface {
	Subscribe(eventType EventType, handler EventHandler)
	Unsubscribe(eventType EventType)
	Publish(event Event) error
	PublishSync(event Event) error
	PublishWithPriority(event Event, priority EventPriority) error
	GetStats() BusStats
	Close()
	IsRunning() bool
}

// 接口实现检查
var _ MessageBusInterface = (*MessageBus)(nil)
