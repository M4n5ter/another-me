# 多智能体系统状态机

这是一个为多智能体任务执行系统设计的完整状态管理框架，提供了系统、任务和Worker的状态跟踪和管理功能。

## 🎯 核心特性

### 📊 三层状态管理

1. **系统状态 (SystemState)** - 跟踪整个系统的运行状态
2. **任务状态 (TaskState)** - 管理每个任务的生命周期
3. **Worker状态 (WorkerState)** - 监控Worker的工作状态

### 🔄 状态机设计

#### 系统状态流转
```
空闲 → 分析中 → 规划中 → 执行中 → 评估中 → 学习中 → 空闲
     ↓         ↓        ↓       ↓        ↓       ↓
   维护中    错误     错误     错误     错误     错误
     ↓         ↓        ↓       ↓        ↓       ↓
   关闭中    维护中   维护中   维护中   维护中   维护中
```

#### 任务状态流转
```
等待执行 → 分析任务 → 分解子任务 → 调度Worker → 运行中 → 已完成
    ↓         ↓         ↓          ↓        ↓      
已取消     已取消     已取消      已取消    已取消
    ↓         ↓         ↓          ↓        ↓
   None     执行失败   执行失败    执行失败  执行失败
              ↓         ↓          ↓        ↓
            重试中     重试中      重试中    暂停
              ↓         ↓          ↓        ↓
            运行中     运行中      运行中    运行中
```

#### Worker状态流转
```
空闲 → 运行中 → 忙碌 → 任务完成 → 空闲
  ↓      ↓      ↓       ↓         ↓
终止中  终止中  终止中  终止中    终止中
  ↓      ↓      ↓       ↓         ↓
已终止  已终止  已终止  已终止    已终止
```

### 🚀 主要功能

#### 状态管理
- ✅ 线程安全的状态读写
- ✅ 状态转换验证
- ✅ 完整的状态转换历史记录
- ✅ 实时状态查询和统计

#### 任务管理
- ✅ 任务创建、更新、删除
- ✅ 任务进度跟踪
- ✅ 父子任务关系管理
- ✅ 优先级支持（低、正常、高、紧急）

#### Worker管理
- ✅ Worker注册和注销
- ✅ 任务分配和调度
- ✅ 性能指标监控
- ✅ 工具权限管理

#### 监控和统计
- ✅ 系统运行时统计
- ✅ 任务和Worker数量统计
- ✅ 状态分布统计
- ✅ 历史状态转换查询

## 📖 使用示例

### 基本使用

```go
package main

import (
    "github.com/m4n5ter/another-me/internal/task_based_core/state"
)

func main() {
    // 创建状态管理器
    sm := state.NewStateManager()
    
    // 设置系统状态
    sm.SetSystemState(state.SystemStateAnalyzing, "开始分析用户请求")
    
    // 创建任务
    task := &state.TaskInfo{
        ID:       "task-001",
        Name:     "示例任务",
        State:    state.TaskStatePending,
        Priority: state.PriorityNormal,
        Metadata: make(map[string]interface{}),
    }
    sm.CreateTask(task)
    
    // 注册Worker
    worker := &state.WorkerInfo{
        ID:    "worker-001",
        Type:  "web_ui",
        State: state.WorkerStateIdle,
        Tools: []string{"click", "type", "scroll"},
    }
    sm.RegisterWorker(worker)
    
    // 分配任务
    sm.AssignTaskToWorker("worker-001", "task-001")
    sm.UpdateTaskState("task-001", state.TaskStateRunning, "开始执行")
    
    // 获取系统信息
    systemInfo := sm.GetSystemInfo()
    fmt.Printf("系统状态: %s, 运行时间: %v\n", 
        systemInfo.State.String(), systemInfo.Uptime)
}
```

### 上下文使用

```go
func useWithContext() {
    sm := state.NewStateManager()
    ctx := state.ContextWithStateManager(context.Background(), sm)
    
    // 在其他组件中使用
    handleRequest(ctx)
}

func handleRequest(ctx context.Context) {
    sm, ok := state.StateManagerFromContext(ctx)
    if !ok {
        log.Fatal("无法获取状态管理器")
    }
    
    // 使用状态管理器...
}
```

## 🏗️ 架构设计

### 核心组件

1. **StateManager** - 状态管理器核心
   - 线程安全的读写锁保护
   - 内存存储（可扩展为持久化）
   - 状态转换历史管理

2. **状态类型定义**
   - SystemState, TaskState, WorkerState 枚举
   - 优先级 Priority 枚举
   - 完整的字符串表示方法

3. **信息结构体**
   - TaskInfo - 任务完整信息
   - WorkerInfo - Worker完整信息  
   - SystemInfo - 系统统计信息
   - StateTransition - 状态转换记录

4. **验证函数**
   - CanTransition - 系统状态转换验证
   - CanTransitionTask - 任务状态转换验证
   - CanTransitionWorker - Worker状态转换验证

### 设计原则

- **线程安全**: 使用 RWMutex 保证并发安全
- **状态验证**: 所有状态转换都经过合法性验证
- **历史记录**: 完整记录所有状态变更，支持审计和调试
- **扩展性**: 支持自定义元数据和性能指标
- **可观测性**: 提供丰富的查询和统计接口

## 🧪 测试和验证

运行测试套件：
```bash
go test ./internal/task_based_core/state/... -v
```

运行演示程序：
```bash
go run examples/state_machine_demo/main.go
``` 