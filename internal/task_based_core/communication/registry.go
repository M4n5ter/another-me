package communication

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"sync"
	"time"
)

// ComponentRegistry 组件注册表
type ComponentRegistry struct {
	mu     sync.RWMutex
	logger *slog.Logger

	// 组件管理
	components map[string]*ComponentInfo // 组件ID -> 组件信息
	eventBus   *MessageBus               // 消息总线

	// 心跳管理
	heartbeats        map[string]time.Time // 组件ID -> 最后心跳时间
	heartbeatInterval time.Duration        // 心跳间隔
	heartbeatTimeout  time.Duration        // 心跳超时时间

	// 生命周期管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ComponentInfo 组件信息
type ComponentInfo struct {
	ID           string          `json:"id"`           // 组件ID
	Type         ComponentType   `json:"type"`         // 组件类型
	Name         string          `json:"name"`         // 组件名称
	Version      string          `json:"version"`      // 版本信息
	Status       ComponentStatus `json:"status"`       // 组件状态
	Capabilities []string        `json:"capabilities"` // 组件能力
	Config       map[string]any  `json:"config"`       // 组件配置
	Metadata     map[string]any  `json:"metadata"`     // 元数据

	// 时间信息
	RegisteredAt  time.Time `json:"registered_at"`  // 注册时间
	LastSeen      time.Time `json:"last_seen"`      // 最后见到时间
	LastHeartbeat time.Time `json:"last_heartbeat"` // 最后心跳时间

	// 通信信息
	EventHandler EventHandler `json:"-"` // 事件处理器（不序列化）
	MessageChan  chan Event   `json:"-"` // 消息通道（不序列化）
}

// ComponentStatus 组件状态
type ComponentStatus int

const (
	ComponentStatusRegistering  ComponentStatus = iota // 注册中
	ComponentStatusActive                              // 活跃
	ComponentStatusIdle                                // 空闲
	ComponentStatusBusy                                // 忙碌
	ComponentStatusError                               // 错误
	ComponentStatusOffline                             // 离线
	ComponentStatusUnregistered                        // 已注销
)

// String 返回组件状态字符串表示
func (s ComponentStatus) String() string {
	switch s {
	case ComponentStatusRegistering:
		return "注册中"
	case ComponentStatusActive:
		return "活跃"
	case ComponentStatusIdle:
		return "空闲"
	case ComponentStatusBusy:
		return "忙碌"
	case ComponentStatusError:
		return "错误"
	case ComponentStatusOffline:
		return "离线"
	case ComponentStatusUnregistered:
		return "已注销"
	default:
		return "未知状态"
	}
}

// NewComponentRegistry 创建新的组件注册表
func NewComponentRegistry(eventBus *MessageBus) *ComponentRegistry {
	ctx, cancel := context.WithCancel(context.Background())

	registry := &ComponentRegistry{
		logger:            slog.Default().WithGroup("component_registry"),
		components:        make(map[string]*ComponentInfo),
		eventBus:          eventBus,
		heartbeats:        make(map[string]time.Time),
		heartbeatInterval: 30 * time.Second, // 默认30秒心跳间隔
		heartbeatTimeout:  90 * time.Second, // 默认90秒心跳超时
		ctx:               ctx,
		cancel:            cancel,
	}

	// 订阅心跳事件
	registry.subscribeToEvents()

	// 启动心跳监控
	registry.startHeartbeatMonitor()

	return registry
}

// RegisterComponent 注册组件
func (r *ComponentRegistry) RegisterComponent(info *ComponentInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.components[info.ID]; exists {
		return fmt.Errorf("组件 %s 已存在", info.ID)
	}

	// 初始化组件信息
	now := time.Now()
	if info.RegisteredAt.IsZero() {
		info.RegisteredAt = now
	}
	info.LastSeen = now
	info.LastHeartbeat = now
	info.Status = ComponentStatusActive

	if info.Config == nil {
		info.Config = make(map[string]any)
	}
	if info.Metadata == nil {
		info.Metadata = make(map[string]any)
	}

	// 创建组件专用的消息通道
	info.MessageChan = make(chan Event, 100)

	r.components[info.ID] = info
	r.heartbeats[info.ID] = now

	// 发布组件注册事件
	if r.eventBus != nil {
		event := NewComponentEvent(EventTypeComponentRegistered, "registry", info.ID, info.Type)
		event.Capabilities = info.Capabilities
		event.Config = info.Config
		err := r.eventBus.Publish(event)
		if err != nil {
			r.logger.Error("组件注册事件发布失败", "component_id", info.ID, "error", err)
			return err
		}
	}

	r.logger.Info("组件注册成功",
		"component_id", info.ID,
		"type", string(info.Type),
		"name", info.Name,
		"capabilities", len(info.Capabilities))

	return nil
}

// UnregisterComponent 注销组件
func (r *ComponentRegistry) UnregisterComponent(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	component, exists := r.components[componentID]
	if !exists {
		return fmt.Errorf("组件 %s 不存在", componentID)
	}

	// 更新组件状态
	component.Status = ComponentStatusUnregistered

	// 关闭消息通道
	if component.MessageChan != nil {
		close(component.MessageChan)
	}

	// 清理相关数据
	delete(r.components, componentID)
	delete(r.heartbeats, componentID)

	// 发布组件注销事件
	if r.eventBus != nil {
		event := NewComponentEvent(EventTypeComponentUnregistered, "registry", componentID, component.Type)
		err := r.eventBus.Publish(event)
		if err != nil {
			r.logger.Error("组件注销事件发布失败", "component_id", componentID, "error", err)
			return err
		}
	}

	r.logger.Info("组件注销",
		"component_id", componentID,
		"type", string(component.Type))

	return nil
}

// GetComponent 获取组件信息
func (r *ComponentRegistry) GetComponent(componentID string) (*ComponentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	component, exists := r.components[componentID]
	if !exists {
		return nil, fmt.Errorf("组件 %s 不存在", componentID)
	}

	// 返回副本
	componentCopy := *component
	return &componentCopy, nil
}

// ListComponents 列出所有组件
func (r *ComponentRegistry) ListComponents() []*ComponentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	components := make([]*ComponentInfo, 0, len(r.components))
	for _, component := range r.components {
		componentCopy := *component
		components = append(components, &componentCopy)
	}

	return components
}

// ListComponentsByType 按类型列出组件
func (r *ComponentRegistry) ListComponentsByType(componentType ComponentType) []*ComponentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var components []*ComponentInfo
	for _, component := range r.components {
		if component.Type == componentType {
			componentCopy := *component
			components = append(components, &componentCopy)
		}
	}

	return components
}

// ListComponentsByStatus 按状态列出组件
func (r *ComponentRegistry) ListComponentsByStatus(status ComponentStatus) []*ComponentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var components []*ComponentInfo
	for _, component := range r.components {
		if component.Status == status {
			componentCopy := *component
			components = append(components, &componentCopy)
		}
	}

	return components
}

// UpdateComponentStatus 更新组件状态
func (r *ComponentRegistry) UpdateComponentStatus(componentID string, status ComponentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	component, exists := r.components[componentID]
	if !exists {
		return fmt.Errorf("组件 %s 不存在", componentID)
	}

	oldStatus := component.Status
	component.Status = status
	component.LastSeen = time.Now()

	r.logger.Info("组件状态更新",
		"component_id", componentID,
		"from", oldStatus.String(),
		"to", status.String())

	return nil
}

// UpdateComponentMetadata 更新组件元数据
func (r *ComponentRegistry) UpdateComponentMetadata(componentID string, metadata map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	component, exists := r.components[componentID]
	if !exists {
		return fmt.Errorf("组件 %s 不存在", componentID)
	}

	maps.Copy(component.Metadata, metadata)
	component.LastSeen = time.Now()

	return nil
}

// SendMessageToComponent 发送消息给特定组件
func (r *ComponentRegistry) SendMessageToComponent(componentID string, event Event) error {
	r.mu.RLock()
	component, exists := r.components[componentID]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("组件 %s 不存在", componentID)
	}

	select {
	case component.MessageChan <- event:
		r.logger.Debug("消息发送成功",
			"component_id", componentID,
			"event_type", string(event.EventType()))
		return nil
	default:
		r.logger.Warn("组件消息通道已满",
			"component_id", componentID,
			"event_type", string(event.EventType()))
		return fmt.Errorf("组件 %s 消息通道已满", componentID)
	}
}

// BroadcastToType 向特定类型的所有组件广播消息
func (r *ComponentRegistry) BroadcastToType(componentType ComponentType, event Event) {
	components := r.ListComponentsByType(componentType)

	for _, component := range components {
		if component.Status == ComponentStatusActive || component.Status == ComponentStatusIdle {
			err := r.SendMessageToComponent(component.ID, event)
			if err != nil {
				r.logger.Error("广播消息失败", "component_id", component.ID, "error", err)
			}
		}
	}

	r.logger.Debug("广播消息",
		"component_type", string(componentType),
		"event_type", string(event.EventType()),
		"target_count", len(components))
}

// GetComponentMessageChannel 获取组件的消息通道
func (r *ComponentRegistry) GetComponentMessageChannel(componentID string) (<-chan Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	component, exists := r.components[componentID]
	if !exists {
		return nil, fmt.Errorf("组件 %s 不存在", componentID)
	}

	return component.MessageChan, nil
}

// Heartbeat 处理组件心跳
func (r *ComponentRegistry) Heartbeat(componentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	component, exists := r.components[componentID]
	if !exists {
		return fmt.Errorf("组件 %s 不存在", componentID)
	}

	now := time.Now()
	component.LastHeartbeat = now
	component.LastSeen = now
	r.heartbeats[componentID] = now

	// 如果组件之前是离线状态，现在恢复在线
	if component.Status == ComponentStatusOffline {
		component.Status = ComponentStatusActive
		r.logger.Info("组件恢复在线", "component_id", componentID)
	}

	return nil
}

// GetRegistryStats 获取注册表统计信息
func (r *ComponentRegistry) GetRegistryStats() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]any)
	statusCounts := make(map[string]int)
	typeCounts := make(map[string]int)

	for _, component := range r.components {
		statusCounts[component.Status.String()]++
		typeCounts[string(component.Type)]++
	}

	stats["total_components"] = len(r.components)
	stats["status_distribution"] = statusCounts
	stats["type_distribution"] = typeCounts
	stats["heartbeat_interval_seconds"] = r.heartbeatInterval.Seconds()
	stats["heartbeat_timeout_seconds"] = r.heartbeatTimeout.Seconds()

	return stats
}

// Close 关闭组件注册表
func (r *ComponentRegistry) Close() {
	r.logger.Info("关闭组件注册表")

	// 取消context
	r.cancel()

	// 等待所有协程结束
	r.wg.Wait()

	// 关闭所有组件的消息通道
	r.mu.Lock()
	for _, component := range r.components {
		if component.MessageChan != nil {
			close(component.MessageChan)
		}
	}
	r.mu.Unlock()

	r.logger.Info("组件注册表已关闭")
}

// 内部方法

// subscribeToEvents 订阅事件
func (r *ComponentRegistry) subscribeToEvents() {
	if r.eventBus != nil {
		// 订阅心跳事件
		r.eventBus.Subscribe(EventTypeHeartbeat, func(event Event) {
			if componentEvent, ok := event.(*ComponentEvent); ok {
				err := r.Heartbeat(componentEvent.ComponentID)
				if err != nil {
					r.logger.Error("心跳处理失败", "component_id", componentEvent.ComponentID, "error", err)
				}
			}
		})
	}
}

// startHeartbeatMonitor 启动心跳监控
func (r *ComponentRegistry) startHeartbeatMonitor() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		ticker := time.NewTicker(r.heartbeatInterval)
		defer ticker.Stop()

		r.logger.Info("心跳监控启动",
			"interval", r.heartbeatInterval,
			"timeout", r.heartbeatTimeout)

		for {
			select {
			case <-ticker.C:
				r.checkHeartbeats()
			case <-r.ctx.Done():
				r.logger.Info("心跳监控退出")
				return
			}
		}
	}()
}

// checkHeartbeats 检查心跳超时
func (r *ComponentRegistry) checkHeartbeats() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	timeoutThreshold := now.Add(-r.heartbeatTimeout)

	for componentID, lastHeartbeat := range r.heartbeats {
		if lastHeartbeat.Before(timeoutThreshold) {
			component := r.components[componentID]
			if component != nil && component.Status != ComponentStatusOffline {
				component.Status = ComponentStatusOffline
				r.logger.Warn("组件心跳超时",
					"component_id", componentID,
					"last_heartbeat", lastHeartbeat,
					"timeout_threshold", timeoutThreshold)
			}
		}
	}
}

// ComponentRegistryInterface 组件注册表接口
type ComponentRegistryInterface interface {
	RegisterComponent(info *ComponentInfo) error
	UnregisterComponent(componentID string) error
	GetComponent(componentID string) (*ComponentInfo, error)
	ListComponents() []*ComponentInfo
	ListComponentsByType(componentType ComponentType) []*ComponentInfo
	ListComponentsByStatus(status ComponentStatus) []*ComponentInfo
	UpdateComponentStatus(componentID string, status ComponentStatus) error
	UpdateComponentMetadata(componentID string, metadata map[string]any) error
	SendMessageToComponent(componentID string, event Event) error
	BroadcastToType(componentType ComponentType, event Event)
	GetComponentMessageChannel(componentID string) (<-chan Event, error)
	Heartbeat(componentID string) error
	GetRegistryStats() map[string]any
	Close()
}

// 接口实现检查
var _ ComponentRegistryInterface = (*ComponentRegistry)(nil)
