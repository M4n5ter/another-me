# Another Me - 智能体编排架构设计

## 1. 概述 (Overview)

"Another Me" 旨在成为一个持续运行的智能体系统，通过不断学习用户行为和偏好，逐渐成为用户的数字助手或"另一个我"。本系统区别于传统的基于请求-响应模式的智能体，它将主动感知、决策，并在无明确任务时，依赖 `Mindscape` 进行智能监控和在适当时机被唤醒。

核心目标：
- **持续运行 (Continuous Operation):** 系统7x24小时不间断运行，具备自主性。
- **自主学习与适应 (Autonomous Learning and Adaptation):** 通过与 `Mindscape` 交互，持久化观察、经验和用户偏好，不断学习并调整自身行为。
- **主动感知与决策 (Proactive Sensing and Decision Making):** 基于当前上下文、从 `Mindscape` 检索的记忆以及 `Mindscape` 的唤醒信号，自主决定下一步行动。
- **成本效益 (Cost-Effectiveness):** 在系统判断无明确任务执行时，将监控责任委托给 `Mindscape`，系统进入低功耗等待状态，以显著降低不必要的Token消耗和计算资源占用。`Mindscape` 在满足预设条件时唤醒 "Another Me"。

## 2. 核心组件 (Core Components)

系统核心编排逻辑将主要位于 `internal/core` 目录下（后续创建），主要包含以下组件：

### 2.1. `MainLoop` (主循环)
- **设想位置:** `internal/core/mainloop.go`
- **职责:**
    - 系统的心跳，驱动 "Another Me" 的持续运行和状态管理。
    - 初始化系统，加载配置，并确保与 `MindscapeConnector` 建立稳定连接。
    - 周期性地（或在被 `Mindscape` 唤醒时）激活 `DecisionMaker` 进行决策。
    - 根据 `DecisionMaker` 的决策结果，通过 `AgentDispatcher` 分发任务给具体的执行单元 (`GUIAgent`, `ReActAgent`)。
    - 当 `DecisionMaker` 判断系统应进入监控等待状态时，协调 `MindscapeConnector` 向 `Mindscape` 注册或更新监控任务，并使系统进入低功耗等待模式，直到接收到 `Mindscape` 的唤醒信号。
    - 处理来自 `MindscapeConnector` 的唤醒事件，并将唤醒数据传递给 `DecisionMaker`。

### 2.2. `DecisionMaker` (决策器)
- **设想位置:** `internal/core/decision_maker.go`
- **职责:**
    - "Another Me" 的大脑，负责制定行动计划或决定进入监控状态。
    - 整合当前上下文信息进行决策：
        - `MainLoop` 传递的 `Mindscape` 唤醒数据（如果适用）。
        - 通过 `MindscapeConnector` 从 `Mindscape` 检索到的相关记忆（例如，用户习惯、历史任务模式、当前目标等）。
        - 系统内部状态和最近的活动历史。
        - （可选）外部事件或传感器数据接口。
    - 判断当前是否有可执行的任务，或者是否需要基于唤醒数据启动新任务。
    - 如果有任务：确定任务目标、优先级，选择合适的执行单元 (`GUIAgent` 或 `ReActAgent`)，并构建任务执行所需的初始上下文。
    - 如果无明确任务或当前任务完成：定义需要 `Mindscape` 监控的条件和场景（例如，监控特定应用启动、用户长时间无交互后的首次交互、特定关键词出现等），以便在合适时机被唤醒。这些监控条件应具体且可操作。
    - 将决策结果（执行任务、更新监控、或继续等待）返回给 `MainLoop`。

### 2.3. `MindscapeConnector` (Mindscape 连接器)
- **设想位置:** `internal/core/mindscape_connector.go`
- **职责:**
    - 作为 "Another Me" 与 `Mindscape` 服务交互的唯一、统一接口层。
    - 封装 `Mindscape` SDK (`internal/mindscape`) 的所有调用细节，对其他内部组件屏蔽 `Mindscape` 的具体实现。
    - **记忆管理:**
        - `StoreMemory(ctx context.Context, memoryData any)`: 将新的观察、学习成果、用户偏好、任务执行摘要等结构化或非结构化数据安全地存入 `Mindscape`。
        - `RetrieveMemories(ctx context.Context, queryContext any) ([]MemoryItem, error)`: 根据 `DecisionMaker` 或 `Agent` 提供的查询上下文（例如，当前任务描述、用户意图关键词、时间范围等）从 `Mindscape` 检索相关记忆。`Mindscape` 负责实现高级检索逻辑（如联想、图谱、向量相似度）。
    - **监控任务委派:**
        - `DelegateMonitoringTask(ctx context.Context, taskDetails MonitoringTask)`: 接收 `DecisionMaker` 定义的监控任务，将其转换为 `Mindscape` 可理解的格式，并注册到 `Mindscape` 服务。这包括监控条件、触发频率、回调配置等。
        - `ClearOrUpdateMonitoringTasks(ctx context.Context, taskUpdate TaskUpdate)`: 根据需要清除或更新已在 `Mindscape` 注册的监控任务。
    - **唤醒机制接口:**
        - `SetupWakeUpListener(handler func(wakeupData WakeupEvent) error)`: 初始化并管理用于接收 `Mindscape` 唤醒信号的机制（例如，启动一个轻量级HTTP服务器监听Webhook回调，或订阅指定的消息队列主题）。当收到唤醒信号时，调用注册的 `handler`（通常由 `MainLoop` 提供）。
        - 对唤醒信号进行初步验证和解析，确保数据完整性，但不负责具体业务逻辑处理。

### 2.4. `AgentDispatcher` (智能体调度器)
- **设想位置:** `internal/core/agent_dispatcher.go`
- **职责:**
    - 根据 `DecisionMaker` 的指令，负责实例化、配置和调用具体的执行单元 (`Agent`)。
    - 将任务目标、执行所需的上下文信息（可能包含从 `Mindscape` 检索的记忆片段）传递给选定的 `Agent`。
    - 管理 `Agent` 的执行生命周期，包括启动、监控执行状态、处理超时和基本错误。
    - 收集 `Agent` 的执行结果（成功、失败、产出数据等），并将其反馈给 `MainLoop`，以便 `DecisionMaker` 进行后续评估和可能的记忆存储。

### 2.5. 执行单元 (Execution Units)
- **`GUIAgent`:**
    - **位置:** `internal/gui_agent/gui_agent.go` (已存在)
    - **职责:** 负责执行与图形用户界面 (GUI) 相关的原子化或简单指令，如点击特定元素、在输入框输入文本、读取屏幕上的特定信息等。其操作通常是确定性的。
- **`ReActAgent`:**
    - **位置:** `pkg/reactagent/agent.go` (已存在)
    - **职责:** 通用的基于 ReAct (Reasoning and Acting) 范式的智能体，能够执行更复杂的、需要多轮思考、工具调用和决策的任务。它本身可能包含一个小的 ReAct 循环。

### 2.6. 接口定义 (Interfaces)
- **设想位置:** `internal/core/interfaces.go`
- **职责:** 定义核心组件间的契约，促进模块化和可测试性。
    - `Agent` 接口: `GUIAgent` 和 `ReActAgent` 都需要实现此接口。
      ```go
      type Agent interface {
          Execute(ctx context.Context, taskDescription Task, initialContext map[string]any) (ExecutionResult, error)
          // 可能还有 Name() string, Type() AgentType 等方法
      }
      ```
    - `MindscapeService` 接口: `MindscapeConnector` 实现此接口，供其他组件依赖注入和调用。
      ```go
      type MindscapeService interface {
          StoreMemory(ctx context.Context, memoryData any) error
          RetrieveMemories(ctx context.Context, queryContext any) ([]MemoryItem, error)
          DelegateMonitoringTask(ctx context.Context, taskDetails MonitoringTask) (string, error) // 返回监控任务ID
          ClearOrUpdateMonitoringTasks(ctx context.Context, taskUpdate TaskUpdate) error
          SetupWakeUpListener(handler func(wakeupData WakeupEvent) error) error
      }
      ```
    - 其他如 `Task`, `ExecutionResult`, `MonitoringTask`, `WakeupEvent`, `MemoryItem` 等数据结构也在此定义。

### 2.7. 关键数据结构释义 (Key Data Structures Explained)
- **`Task`**: 描述一个需要 `Agent` 执行的具体任务。
  ```go
  type Task struct {
      ID          string // 任务唯一标识
      Type        string // 任务类型 (例如 "gui_click", "react_plan_and_execute")
      Description string // 任务的自然语言描述
      AgentType   AgentType // 指定执行此任务的Agent类型
      Parameters  map[string]any // 任务执行所需的具体参数 (例如，点击坐标，ReAct Agent的目标)
      Priority    int    // 任务优先级
      // ... 其他元数据
  }
  ```
- **`ExecutionResult`**: `Agent` 执行任务后的结果。
  ```go
  type ExecutionResult struct {
      TaskID      string // 对应的任务ID
      Status      string // 执行状态 ("success", "failure", "in_progress")
      Output      any    // Agent执行的主要产出物 (例如，GUI Agent的截图路径，ReAct Agent的最终答案)
      Observations []string // Agent在执行过程中的重要观察或中间步骤的文本描述
      Error       string // 如果执行失败，记录错误信息
      // ... 其他性能指标或元数据
  }
  ```
- **`MonitoringTask`**: 定义一个需要委托给 `Mindscape` 的监控任务。这是 "Another Me" 内部的数据结构，`MindscapeConnector` 会将其转换为 `Mindscape` API 所需的格式。
  ```go
  type MonitoringTask struct {
      ID                  Option[string]   // 监控任务的唯一ID (由Mindscape生成并返回，对应Mindscape的UUID)
      Description         string           // 监控任务的自然语言描述 (Another Me 内部使用)
      MindscapeTaskType   string           // 期望在Mindscape中使用的任务类型 (例如 "generic_condition_monitor")
                                        // MindscapeConnector 将基于此类型和以下Conditions/TargetData构造发送给Mindscape的Parameters
      Conditions          []MonitorCondition // 触发唤醒的一组条件
      TargetData          []string         // 满足条件时，Mindscape需要采集并返回的数据点 (用于填充WakeupEvent.ObservedData)
      NotificationMethods []string         // 通知方式，如 ["webhook", "mq"]
      WebhookURL          Option[string]   // 如果 NotificationMethods 含 "webhook", 此为回调URL
      MQTopic             Option[string]   // 如果 NotificationMethods 含 "mq", 此为消息队列主题
      MaxRetries          Option[int]      // 通知传递的最大重试次数 (可选, Mindscape有默认值)
      IsEnabled           bool             // Another Me 内部标记，是否希望此任务在Mindscape中实际处于活动状态
      // ... 其他配置，如频率限制、持续时间等
  }
  type MonitorCondition struct {
      Type     string // e.g., "application_start", "text_on_screen", "user_idle_then_active"
      Property string // e.g., "application_name", "text_pattern", "idle_duration_seconds"
      Operator string // e.g., "equals", "contains", "greater_than"
      Value    any    // The value to compare against
  }
  ```
- **`TaskUpdate`**: 用于更新或清除 `Mindscape` 中的监控任务。
  ```go
  type TaskUpdate struct {
      TasksToUpdate []MonitoringTask // 需要更新的监控任务详情
      TaskIDsToDelete []string     // 需要删除的监控任务ID
  }
  ```
- **`WakeupEvent`**: `Mindscape` 唤醒 "Another Me" 时传递的数据。
  ```go
  type WakeupEvent struct {
      MonitoringTaskID string         // 触发唤醒的监控任务ID
      TriggerTime      time.Time      // 唤醒条件满足的时间
      ObservedData     map[string]any // Mindscape观测到的数据 (根据MonitoringTask.TargetData定义)
      Reason           string         // 简述唤醒原因
      // ... 其他元数据
  }
  ```
- **`MemoryItem`**: 表示存储在 `Mindscape` 中的一条记忆。
  ```go
  type MemoryItem struct {
      ID          string         // 记忆唯一标识
      Timestamp   time.Time      // 记忆产生的时间
      Type        string         // 记忆类型 (例如 "observation", "user_preference", "task_summary", "error_log")
      Content     any            // 记忆的具体内容 (可以是文本、结构化数据、指向其他资源的指针等)
      Keywords    []string       // 用于检索的关键词
      Importance  float64        // 记忆的重要性评分 (0.0 - 1.0)
      RelatedIDs  []string       // 与此记忆相关的其他记忆ID (用于构建记忆图谱)
      // ... 其他元数据，如来源、置信度等
  }
  ```

## 3. 工作流程 (Workflow)

### 3.1. 启动与初始化 (Startup & Initialization)
1.  `MainLoop` 启动。
2.  加载配置、初始化日志系统。
3.  `MindscapeConnector` 初始化，与 `Mindscape` 服务建立连接。
4.  `MindscapeConnector` 调用 `SetupWakeUpListener`，传入一个由 `MainLoop` 实现的回调函数，用于处理接收到的唤醒信号。
5.  `MainLoop` 可选地尝试从 `Mindscape` 加载初始状态或核心记忆。
6.  进入主运行循环。

### 3.2. 主动运行循环 (Active Operational Cycle)
1.  `MainLoop` 激活 `DecisionMaker`。
2.  `DecisionMaker` 整合当前所有可用信息：
    *   如果是被 `Mindscape` 唤醒，则包含唤醒数据。
    *   调用 `MindscapeConnector.RetrieveMemories()` 获取与当前情境相关的记忆。
    *   分析系统内部状态。
3.  **决策分支:**
    a.  **有任务执行 / 基于唤醒数据需执行新任务:**
        i.  `DecisionMaker` 确定任务目标、所需 `Agent` 类型及任务上下文。
        ii. `MainLoop` 通过 `AgentDispatcher` 实例化并派发任务给选定的 `Agent` (`GUIAgent` 或 `ReActAgent`)，传递必要的上下文。
        iii. `Agent` 执行任务。在执行过程中，`Agent` 可能会通过 `MindscapeConnector` 存储临时的观察或检索更细粒度的记忆。
        iv. `AgentDispatcher` 将 `Agent` 的执行结果（成功/失败、输出、观察）返回给 `MainLoop`。
        v.  `MainLoop` 将结果传递给 `DecisionMaker` 进行评估。`DecisionMaker` 判断是否有新的学习成果或重要观察需要持久化。
        vi. 如果需要，`MainLoop` (或 `DecisionMaker` 指示) 调用 `MindscapeConnector.StoreMemory()` 将新信息存入 `Mindscape`。
        vii.返回步骤1，开始新的决策周期。
    b.  **无任务执行 / 当前任务完成，进入监控模式:**
        i.  `DecisionMaker` 判断当前无明确主动任务，并定义需要 `Mindscape` 监控的具体条件（例如，用户打开了特定应用如 "VSCode"，或屏幕上出现文本 "项目已完成"，或特定时间段内无交互后首次出现键盘/鼠标活动）。
        ii. `MainLoop` 指示 `MindscapeConnector.DelegateMonitoringTask()` 将这些监控任务注册到 `Mindscape`。
        iii. `MainLoop` 使系统进入低功耗等待状态，主要依赖 `MindscapeConnector` 的唤醒监听器。

### 3.3. 监控与唤醒 (Monitoring & Wake-up)
1.  "Another Me" 系统处于低功耗等待状态，CPU 和网络活动降至最低。
2.  `Mindscape` 服务根据 `MindscapeConnector` 注册的条件独立进行外部环境或用户行为的监控。
3.  当任一监控条件满足时，`Mindscape` 通过预设的机制 (如 Webhook 或 MQ 消息) 发送唤醒信号，信号中包含触发唤醒的事件类型和观测到的相关数据。
4.  `MindscapeConnector` 的唤醒监听器（例如，HTTP端点或MQ消费者）接收到该信号和数据。
5.  监听器调用 `MainLoop` 注册的回调函数，将解析后的 `WakeupEvent` 数据传递给 `MainLoop`。
6.  `MainLoop` 被唤醒，恢复活动状态。它将唤醒数据作为重要上下文输入，传递给 `DecisionMaker`。
7.  系统返回到 "主动运行循环" 的步骤 2.a，`DecisionMaker` 将基于唤醒数据和新检索的记忆来决定下一步行动。

## 4. 数据流 (Data Flow)

- **记忆 (Memory):**
    - **产生:** `Agents` 执行任务产生观察/学习 -> `AgentDispatcher` -> `MainLoop` -> `DecisionMaker` (评估) -> `MindscapeConnector.StoreMemory()` -> `Mindscape` (持久化存储)。
    - **使用:** `DecisionMaker` (决策前) / `Agents` (执行中，按需) -> `MindscapeConnector.RetrieveMemories()` -> 从 `Mindscape` 获取。
- **任务 (Tasks):**
    - `DecisionMaker` (根据上下文和记忆) 产生任务定义 -> `MainLoop` -> `AgentDispatcher` -> `Agents` (执行)。
- **上下文 (Context for Decision Making):**
    - `Mindscape` (通过唤醒信号) -> `MindscapeConnector` -> `MainLoop` -> `DecisionMaker`.
    - `Agent` 执行结果 -> `AgentDispatcher` -> `MainLoop` -> `DecisionMaker`.
    - 系统内部状态 -> `DecisionMaker`.
- **监控指令 (Monitoring Instructions):**
    - `DecisionMaker` (判断无任务时) 定义监控条件 -> `MainLoop` -> `MindscapeConnector.DelegateMonitoringTask()` -> `Mindscape` (执行监控)。
- **唤醒数据 (Wake-up Data):**
    - `Mindscape` (监控条件满足) -> 唤醒机制 (Webhook/MQ) -> `MindscapeConnector` 的监听器 -> `MainLoop` -> `DecisionMaker`。

## 5. 与 `mindscape` 的交互 (Interaction with `mindscape`)

- **记忆的持久化与检索:** "Another Me" 依赖 `Mindscape` 作为其长期记忆库。所有重要的学习成果、用户画像、任务历史、关键事件等都通过 `MindscapeConnector` 存入 `Mindscape`。在决策和任务执行前，也通过 `MindscapeConnector` 从 `Mindscape` 获取相关的历史记忆、模式和知识，以提供上下文和指导。`Mindscape` 自身负责高级记忆管理功能，如联想推理、知识图谱构建、向量化检索和记忆衰减等。
- **智能监控任务委托:** 当 "Another Me" 的 `DecisionMaker` 判断系统可以进入空闲或等待状态时，它会定义一系列具体的监控条件，并通过 `MindscapeConnector` 将这些监控任务委托给 `Mindscape`。例如："当比特币价格达到120000美元时通知我"，或者"Elon Musk 发推文时通知我"。
- **唤醒机制与数据传递:** `Mindscape` 在其监控的条件满足后，会通过预先配置好的通道（如Webhook URL或MQ主题）向 "Another Me" 的 `MindscapeConnector` 发送唤醒信号。这个信号必须包含触发唤醒的事件详情和相关的上下文数据（例如，哪个应用被打开、屏幕上出现的文本片段、用户输入的初步内容等）。`MindscapeConnector` 负责监听这些信号，验证其来源，解析数据，并通过回调机制激活 `MainLoop`，将数据传递给 `DecisionMaker`。

## 6. 错误处理与韧性 (Error Handling & Resilience)

- 各核心组件需实现健壮的错误处理，避免单点故障导致整个系统崩溃。
- `MindscapeConnector` 与 `Mindscape` 的所有网络通信应包含超时控制、重试逻辑（针对可恢复错误）和熔断机制。
- `MainLoop` 需要能够从其子组件（如 `DecisionMaker`, `AgentDispatcher`）的临时故障中优雅恢复，或在关键组件失败时记录详细错误并尝试安全重启或进入受限模式。
- 鉴于系统的持续运行特性，必须高度关注资源管理，防止内存泄漏、句柄泄漏等问题。
- `Mindscape` 不可用时的降级策略：`DecisionMaker` 应能感知到 `Mindscape` 的不可用状态，并采取备用逻辑（例如，基于短期记忆运行，或进入更长时间的本地化等待，定期尝试重连）。**此外，对于计划发送给 `Mindscape` 的数据（如新的记忆条目、监控任务更新），应实现一个本地持久化队列。当 `MindscapeConnector` 检测到 `Mindscape` 服务恢复时，会自动处理队列中的数据，确保信息最终一致性。**

## 7. 扩展性 (Extensibility)

- **新 Agent 类型:** 通过实现统一的 `Agent` 接口，可以方便地集成新的执行单元（例如，专门用于特定应用操作的 `Agent`）。`AgentDispatcher` 需要能够动态注册和调度新的 `Agent` 类型。
- **决策逻辑的演进:** `DecisionMaker` 的内部逻辑可以设计为可插拔的策略模式。初期可能基于规则和启发式方法，未来可以逐步引入更复杂的基于机器学习模型的决策引擎。
- **监控能力的增强:** 只要 `Mindscape` 服务支持新的监控维度和条件类型，`MindscapeConnector` 就可以相应扩展，使得 "Another Me" 能够利用这些新的监控能力。
- **与外部服务的集成:** 可以通过定义新的 `Agent` 或在 `DecisionMaker` 中增加逻辑来与更多的外部服务和API交互。

## 8. 待讨论/未来考虑 (Open Questions / Future Considerations)

- `MainLoop` 的具体调度策略：是完全依赖 `Mindscape` 唤醒，还是结合一个保底的低频自主检查周期？
- `DecisionMaker` 的复杂性与演化路径：如何从简单的规则驱动平滑过渡到更智能的决策模型？
- `Mindscape` 通信的加密与安全认证机制。
- 本地缓存策略：对于频繁访问的记忆或 `Mindscape` 短暂不可用时的关键数据，是否需要在 "Another Me" 侧实现一定程度的本地缓存？
- 用户隐私与数据安全：如何确保在学习用户行为和存储记忆时的隐私保护。
- `Mindscape` 监控任务的优先级和冲突解决：如果定义了多个可能冲突或重叠的监控任务，`Mindscape` 如何处理？"Another Me" 如何管理这些任务？
- "Another Me" 自身状态的持久化与恢复：除了 `Mindscape` 中的记忆，系统自身运行的一些核心状态（如当前目标、进行中的长任务上下文）是否也需要持久化，以便在重启后恢复？
