# Another Me - 智能体编排架构设计

## 1. 概述 (Overview)

"Another Me" 旨在成为一个持续运行的智能体系统，通过不断学习用户行为和偏好，逐渐成为用户的数字助手或“另一个我”。本系统区别于传统的基于请求-响应模式的智能体，它将主动感知、决策，并在无明确任务时进入由 `Mindscape` 辅助的监控等待状态。

核心目标：
- **持续运行 (Continuous Operation):** 系统7x24小时不间断运行。
- **自主学习 (Autonomous Learning):** 通过 `Mindscape` 持久化和检索记忆，不断学习和适应用户。
- **主动感知与决策 (Proactive Sensing and Decision Making):** 根据当前上下文和记忆，自主决定下一步行动。
- **成本效益 (Cost-Effectiveness):** 在空闲时，通过 `Mindscape` 的监控能力降低不必要的Token消耗。

## 2. 核心组件 (Core Components)

系统核心编排逻辑将主要位于 `internal/core` 目录下（后续创建），主要包含以下组件：

### 2.1. `MainLoop` (主循环)
- **设想位置:** `internal/core/mainloop.go`
- **职责:**
    - 系统的心跳，负责驱动整个 "Another Me" 的持续运行。
    - 初始化系统，加载配置和与 `Mindscape` 建立连接。
    - 周期性地或在被 `Mindscape` 唤醒时，调用 `DecisionMaker` 进行决策。
    - 根据决策结果，通过 `AgentDispatcher` 分发任务给具体的执行单元 (`GUIAgent`, `ReActAgent`)。
    - 当 `DecisionMaker` 判断系统应进入监控状态时，协调 `MindscapeConnector` 设置监控任务，并使系统进入低功耗等待模式。

### 2.2. `DecisionMaker` (决策器)
- **设想位置:** `internal/core/decision_maker.go`
- **职责:**
    - "Another Me" 的大脑，负责制定行动计划。
    - 整合当前上下文信息：
        - `Mindscape` 唤醒时提供的数据。
        - 从 `Mindscape` 检索到的相关记忆。
        - 系统内部状态。
        - （可选）外部事件或传感器数据。
    - 判断当前是否有可执行的任务。
    - 如果有任务，确定任务目标、优先级，并选择合适的执行单元 (`GUIAgent` 或 `ReActAgent`)。
    - 如果无明确任务，定义需要 `Mindscape` 监控的条件和场景，以便在合适时机被唤醒。

### 2.3. `MindscapeConnector` (Mindscape 连接器)
- **设想位置:** `internal/core/mindscape_connector.go`
- **职责:**
    - 作为 "Another Me" 与 `Mindscape` 服务交互的统一接口。
    - 封装 `Mindscape` SDK (`internal/mindscape`) 的调用。
    - **记忆管理:**
        - `StoreMemory(memoryData)`: 将新的观察、学习成果、用户偏好等存入 `Mindscape`。
        - `RetrieveMemories(queryContext)`: 根据当前任务或上下文从 `Mindscape` 检索相关记忆。
    - **监控与唤醒:**
        - `DelegateMonitoringTask(taskDetails)`: 当系统空闲时，向 `Mindscape` 注册监控任务（例如：监控特定应用活动、用户输入模式等）。
        - `SetupWakeUpListener()`: 配置接收 `Mindscape` 唤醒信号的机制（例如，启动一个HTTP端点处理Webhook，或订阅MQ消息）。
        - 处理 `Mindscape` 发来的唤醒通知，并将相关数据传递给 `MainLoop`。

### 2.4. `AgentDispatcher` (智能体调度器)
- **设想位置:** `internal/core/agent_dispatcher.go`
- **职责:**
    - 根据 `DecisionMaker` 的指令，负责实例化和调用具体的执行单元。
    - 将任务目标、上下文信息和必要的记忆传递给选定的 `Agent`。
    - 管理 `Agent` 的执行生命周期（例如，超时控制、错误处理）。
    - 收集 `Agent` 的执行结果，并反馈给 `MainLoop` 或 `DecisionMaker` 进行后续处理（例如，更新记忆）。

### 2.5. 执行单元 (Execution Units)
- **`GUIAgent`:**
    - **位置:** `internal/gui_agent/gui_agent.go` (已存在)
    - **职责:** 负责执行与图形用户界面 (GUI) 相关的简单指令，如点击、输入、读取界面元素等。
- **`ReActAgent`:**
    - **位置:** `pkg/reactagent/agent.go` (已存在)
    - **职责:** 通用的基于 ReAct (Reasoning and Acting) 范式的智能体，能够执行更复杂的、需要思考和多步骤操作的任务。

### 2.6. 接口定义 (Interfaces)
- **设想位置:** `internal/core/interfaces.go`
- **职责:** 定义核心组件间的契约。
    - `Agent` 接口: `GUIAgent` 和 `ReActAgent` 都需要实现此接口，包含如 `Execute(taskContext)` 等方法。
    - `MindscapeService` 接口: `MindscapeConnector` 实现此接口，供其他组件调用。

## 3. 工作流程 (Workflow)

### 3.1. 启动与初始化 (Startup & Initialization)
1. `MainLoop` 启动。
2. 初始化日志、配置加载。
3. `MindscapeConnector` 初始化，与 `Mindscape` 服务建立连接，并设置唤醒监听器。
4. `MainLoop` 尝试从 `Mindscape` 加载初始记忆或状态。
5. 进入主运行循环。

### 3.2. 主动运行循环 (Active Operational Cycle)
1. `MainLoop` 调用 `DecisionMaker`。
2. `DecisionMaker` 分析当前上下文（包括从 `Mindscape` 获取的记忆、系统状态、上次任务结果等）。
3. **决策分支:**
    a. **有任务执行:**
        i. `DecisionMaker` 确定任务目标和适用 `Agent`。
        ii. `MainLoop` 通过 `AgentDispatcher` 将任务派发给选定的 `Agent` (`GUIAgent` 或 `ReActAgent`)。
        iii. `Agent` 执行任务，执行过程中可能与外部环境交互，或请求 `MindscapeConnector` 存取短期记忆。
        iv. `AgentDispatcher` 返回执行结果给 `MainLoop`。
        v. `MainLoop` (可能通过 `DecisionMaker`) 评估结果，决定是否需要将新的学习/观察存入 `Mindscape` (通过 `MindscapeConnector`)。
        vi. 返回步骤1，开始新的决策周期。
    b. **无任务执行 (进入监控模式):**
        i. `DecisionMaker` 判断当前无事可做，并定义需要 `Mindscape` 监控的条件（例如，用户打开某个特定应用，或一段时间无键盘鼠标活动后出现新的活动）。
        ii. `MainLoop` 指示 `MindscapeConnector` 将这些监控任务注册到 `Mindscape`。
        iii. `MainLoop` 使系统进入等待状态，等待 `Mindscape` 的唤醒信号。

### 3.3. 监控与唤醒 (Monitoring & Wake-up)
1. "Another Me" 处于等待状态。
2. `Mindscape` 根据设定的条件进行监控。
3. 当监控条件满足时，`Mindscape` 通过预设的机制 (Webhook/MQ) 发送唤醒信号，并附带观测到的数据。
4. `MindscapeConnector` 的唤醒监听器接收到信号和数据。
5. `MindscapeConnector` 通知 `MainLoop` 系统被唤醒，并传递相关数据。
6. `MainLoop` 恢复活动状态，将唤醒数据和从 `Mindscape` 检索到的相关记忆传递给 `DecisionMaker`。
7. 返回到 "主动运行循环" 的步骤1。

## 4. 数据流 (Data Flow)

- **记忆 (Memory):** `Agents` 产生观察 -> `MindscapeConnector` -> `Mindscape` (存储) -> `MindscapeConnector` -> `DecisionMaker` / `Agents` (检索使用)。
- **任务 (Tasks):** `DecisionMaker` 产生任务定义 -> `AgentDispatcher` -> `Agents` (执行)。
- **上下文 (Context):** `Mindscape` 唤醒数据, `Agent` 执行结果, 系统状态 -> `DecisionMaker`。
- **监控指令 (Monitoring Instructions):** `DecisionMaker` 定义监控条件 -> `MindscapeConnector` -> `Mindscape`。

## 5. 与 `mindscape` 的交互 (Interaction with `mindscape`)

- **记忆持久化:** 所有学习成果、用户画像、重要事件等都通过 `MindscapeConnector` 存入 `Mindscape`。
- **记忆检索:** 在决策和任务执行前，通过 `MindscapeConnector` 从 `Mindscape` 获取相关记忆，以提供上下文和指导。`Mindscape` 负责高级记忆管理（联想、图谱、向量检索）。
- **任务委托:** 当 "Another Me" 空闲时，将监控任务（如“当用户打开VSCode时通知我”）委托给 `Mindscape`。
- **唤醒机制:** `Mindscape` 通过 Webhook 或消息队列 (MQ) 唤醒 "Another Me"，并传递触发唤醒的事件数据。`MindscapeConnector` 负责监听和处理这些唤醒信号。

## 6. 错误处理与韧性 (Error Handling & Resilience)

- 各组件需实现健壮的错误处理机制。
- 与 `Mindscape` 的通信应包含重试和超时逻辑。
- `MainLoop` 需要能够从部分组件的临时故障中恢复，或记录严重错误并尝试重启关键流程。
- 持续运行要求对资源泄漏（内存、句柄等）高度关注。

## 7. 扩展性 (Extensibility)

- **新 Agent 类型:** 通过实现 `Agent` 接口，可以方便地集成新的执行单元。`AgentDispatcher` 需要更新以识别和调度新的 `Agent` 类型。
- **新决策逻辑:** `DecisionMaker` 的逻辑可以模块化，方便根据需求迭代和增强。
- **新监控能力:** 只要 `Mindscape` 支持，`MindscapeConnector` 可以扩展以支持新的监控任务类型。

## 8. 待讨论/未来考虑 (Open Questions / Future Considerations)

- `MainLoop` 的具体调度策略（固定周期轮询 vs. 事件驱动）。
- `DecisionMaker` 的复杂性：初期可以是基于规则的，未来可能演化为基于模型的。
- `Mindscape` 通信失败的降级策略。
- 安全性：与 `Mindscape` 的通信安全，本地敏感数据的处理。
- 详细的 API 接口设计 (`MindscapeConnector` 与 `Mindscape` 之间，以及内部组件之间)。
