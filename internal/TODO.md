# Another Me - 智能体编排系统开发 TODO List

## 现有Agent用法参考 (Agent Usage Reference)

### 1. GUIAgent 用法
```go
// 位置: internal/gui_agent/gui_agent.go
// 用法模式:
guiAgent, err := guiagent.NewGUIAgent(ctx, chatAdapter)
result, err := guiAgent.Execute(ctx, instruction, screenshotURL)

// 返回结果: ExecutionResult 包含 ActionResult 和执行输出
type ExecutionResult struct {
    ActionResult    // 包含 Action, Thought, StartBox, EndBox 等
    ExecutionOutput string // 执行结果描述
}
```

### 2. ReActAgent 用法
```go
// 位置: pkg/reactagent/agent.go
// 用法模式:
reactAgent, err := reactagent.NewToolCallingAgentBuilder().
    WithLLMAdapter(chatAdapter).
    WithTaskEvaluator(chatAdapter).
    WithToolRegistry(registry).
    WithLogger(logger).
    WithMaxIterations(maxIter).
    WithSystemPrompt(systemPrompt).
    Build()

// 流式执行:
outputChan, err := reactAgent.Run(ctx, userInput, conversationID)
for chunk := range outputChan {
    // 处理不同类型的输出块: Text, Reasoning, ToolStart, ToolEnd, Error, Finish 等
}
```

### 3. Agent接口标准
从现有实现分析，Agent应该实现类似以下接口：
```go
type Agent interface {
    Execute(ctx context.Context, task Task, initialContext map[string]any) (ExecutionResult, error)
    // 或者对于流式Agent: Run(ctx context.Context, input string, conversationID string) (<-chan OutputChunk, error)
}
```

### 4. MindscapeClient用法
```go
// 位置: internal/mindscape/client.go
client := mindscape.NewClient(config)

// Memory操作:
client.Memory.StoreMemoryFragment(ctx, storeReq)
client.Memory.RecallMemory(ctx, recallReq)
client.Memory.GetUserProfile(ctx, userID)

// Sentinel操作:
client.Sentinel.CreateTask(ctx, createReq)
client.Sentinel.GetTask(ctx, taskID)
client.Sentinel.DeleteTask(ctx, taskID)
```

---

## 开发阶段划分

### 阶段 1: 核心接口与数据结构定义 ✅ (Phase 1: Core Interfaces & Data Structures)
- [x] 创建 `internal/core/` 目录结构
- [x] 定义核心接口 (`interfaces.go`)
  - [x] `Agent` 接口
  - [x] `MindscapeService` 接口  
  - [x] `DecisionMaker` 接口
  - [x] `AgentDispatcher` 接口
- [x] 定义核心数据结构
  - [x] `Task` 结构体
  - [x] `ExecutionResult` 结构体
  - [x] `MonitoringTask` 结构体
  - [x] `WakeupEvent` 结构体
  - [x] `MemoryItem` 结构体
  - [x] `TaskUpdate` 结构体
  - [x] `MonitorCondition` 结构体
- [x] 编写核心接口的单元测试

### 阶段 2: MindscapeConnector 实现 ✅ (Phase 2: MindscapeConnector Implementation)
- [x] 实现 `MindscapeConnector` (`internal/core/mindscape_connector.go`)
  - [x] 实现 `StoreMemory` 方法
  - [x] 实现 `RetrieveMemories` 方法
  - [x] 实现 `DelegateMonitoringTask` 方法
  - [x] 实现 `ClearOrUpdateMonitoringTasks` 方法
  - [x] 实现 `SetupWakeUpListener` 方法 (Webhook/MQ监听器)
- [x] 编写 `MindscapeConnector` 的单元测试
  - [x] Mock Mindscape Client 测试
  - [x] 记忆管理功能测试
  - [x] 监控任务管理测试
  - [x] 唤醒监听器测试
- [x] 实现 `WebhookWakeupListener` (`internal/core/wakeup_listener.go`)
- [x] 完整的API字段映射和类型转换
- [x] 本地队列持久化机制
- [x] 健康检查和错误恢复机制

### 阶段 3: Agent 适配器层 ✅ (Phase 3: Agent Adapter Layer)  
- [x] 创建现有Agent的适配器
  - [x] `GUIAgentAdapter` (`internal/core/agents/gui_adapter.go`)
    - [x] 适配现有GUIAgent的Execute方法到标准Agent接口
    - [x] 处理Task到instruction和screenshot的转换
  - [x] `ReActAgentAdapter` (`internal/core/agents/react_adapter.go`)
    - [x] 适配现有ReActAgent的Run方法到标准Agent接口
    - [x] 处理流式输出到标准ExecutionResult的转换
- [x] Agent适配器的单元测试

### 阶段 4: 决策引擎实现 ✅ (Phase 4: Decision Engine Implementation)
- [x] 实现 `DecisionMaker` (`internal/core/decision_maker.go`)
  - [x] 实现上下文信息整合逻辑
  - [x] 实现任务识别与优先级判断
  - [x] 实现Agent类型选择逻辑
  - [x] 实现监控条件定义逻辑
  - [x] 实现基于记忆和唤醒数据的决策逻辑
- [x] 决策引擎的单元测试
  - [x] 任务决策测试
  - [x] 监控模式切换测试  
  - [x] 记忆检索与利用测试

### 阶段 5: Agent调度器实现 ✅ (Phase 5: Agent Dispatcher Implementation)
- [x] 实现 `AgentDispatcher` (`internal/core/agent_dispatcher.go`)
  - [x] 支持多个同类型Agent实例注册
  - [x] Agent注册与管理（按Agent ID和Agent Name）
  - [x] 智能任务分发逻辑（负载均衡）
  - [x] Agent生命周期管理
  - [x] 执行结果收集与反馈
- [x] Agent调度器的单元测试
  - [x] 多Agent实例注册测试
  - [x] 负载均衡测试
  - [x] Agent注销与ID管理测试
  - [x] 专门化Agent选择测试

### 阶段 6: 主循环实现 ⏳ (Phase 6: Main Loop Implementation)
- [x] 实现 `MainLoop` (`internal/core/mainloop.go`)
  - [x] 系统初始化逻辑
  - [x] 主动运行循环
  - [x] 监控模式切换管理
  - [x] 唤醒事件处理
  - [x] 错误恢复机制
- [x] 主循环的单元测试
  - [x] 初始化测试
  - [x] 循环逻辑测试
  - [x] 错误恢复测试

### 阶段 7: 系统集成与测试 ⏳ (Phase 7: System Integration & Testing)
- [ ] 创建系统启动器 (`cmd/another-me/main.go`)
- [ ] 配置管理集成
- [ ] 日志系统集成
- [ ] 集成测试
  - [ ] 端到端工作流测试
  - [ ] Mindscape交互测试
  - [ ] Agent执行测试
  - [ ] 监控与唤醒测试
- [ ] 性能优化
  - [ ] 内存管理优化
  - [ ] 网络连接池优化
  - [ ] 错误重试机制优化

### 阶段 8: 高级功能实现 ⭐ (Phase 8: Advanced Features)
- [ ] 本地队列持久化 (Mindscape不可用时)
- [ ] 系统状态恢复机制
- [ ] 更智能的决策算法
- [ ] Agent动态加载机制
- [ ] 监控条件的优先级和冲突解决
- [ ] 性能监控与指标收集

---

## 当前进度 (Current Progress)

### ✅ 已完成 (Completed)
- [x] 需求分析和架构设计
- [x] 现有Agent用法调研
- [x] 阶段1: 核心接口与数据结构定义
- [x] 阶段2: MindscapeConnector 实现
- [x] 阶段3: Agent适配器层实现
- [x] 阶段4: 决策引擎实现
- [x] 阶段5: Agent调度器实现

### 🔄 进行中 (In Progress)  
- [ ] 阶段6: 主循环实现

### ⏳ 待开始 (Pending)
- [ ] 阶段7-8的所有任务

---

## 技术栈确认 (Technical Stack Confirmation)

### 已确认使用的库
- **测试框架**: `github.com/stretchr/testify/assert` + `github.com/stretchr/testify/mock`
- **JSON处理**: `github.com/json-iterator/go` (项目中已使用)
- **日志**: `log/slog` (Go标准库)
- **HTTP客户端**: 基于现有 `pkg/common/HTTPClient`
- **配置管理**: 基于现有 `pkg/config`
- **国际化**: 基于现有 `pkg/i18n`

### 项目结构约定
```
internal/core/
├── interfaces.go          # 核心接口定义
├── types.go              # 数据结构定义  
├── mainloop.go           # 主循环实现
├── decision_maker.go     # 决策引擎
├── agent_dispatcher.go   # Agent调度器
├── mindscape_connector.go # Mindscape连接器
├── agents/               # Agent适配器
│   ├── gui_adapter.go    # GUI Agent适配器
│   └── react_adapter.go  # ReAct Agent适配器
└── tests/               # 核心模块测试
    ├── *_test.go        # 单元测试
    └── integration_test.go # 集成测试
```

---

## 注意事项 (Important Notes)

1. **渐进式开发**: 每个阶段完成后进行测试，确保功能正确再进入下一阶段
2. **接口优先**: 所有模块都应该先定义接口，便于测试和后续扩展
3. **错误处理**: 每个组件都要有完善的错误处理和恢复机制
4. **日志记录**: 关键操作都要有详细的日志记录
5. **内存管理**: 特别注意长期运行的内存泄漏问题
6. **线程安全**: 考虑并发访问的线程安全问题

---

## 快速开始 (Quick Start)

开始开发时，按以下顺序执行：
1. ✅ 创建 `internal/core/` 目录结构
2. ✅ 定义核心接口和数据结构
3. ✅ 编写基础的单元测试框架
4. 🔄 逐步实现各个组件
5. 持续进行单元测试和集成测试
