 ```mermaid
 graph TB
    subgraph "多智能体系统架构"
        subgraph "Core Layer - 核心层"
            SM[StateManager<br/>状态管理器]
            MB[MessageBus<br/>消息总线]
            CR[ComponentRegistry<br/>组件注册表]
            TD[TaskDAG<br/>任务依赖图]
        end
        
        subgraph "Integration Layer - 集成层"
            SC[StateController<br/>状态控制器]
            EM[EventMapper<br/>事件映射器]
            SB[StateBridge<br/>状态桥接器]
        end
        
        subgraph "Agent Layer - 智能体层"
            OR[Orchestrator<br/>主脑编排器]
            W1[Worker-1<br/>GUI工作器]
            W2[Worker-2<br/>数据工作器]
            W3[Worker-N<br/>其他工作器]
        end
        
        subgraph "Event Flow - 事件流"
            E1[StateChanged]
            E2[TaskAssigned]
            E3[WorkerStatus]
            E4[SystemEvent]
        end
    end
    
    %% 状态管理连接
    SC --> SM
    SC --> MB
    
    %% 事件映射连接
    EM --> MB
    EM --> SM
    
    %% 状态桥接连接
    SB --> SM
    SB --> CR
    SB --> TD
    
    %% 组件通信连接
    OR --> MB
    W1 --> MB
    W2 --> MB
    W3 --> MB
    
    %% 注册连接
    OR --> CR
    W1 --> CR
    W2 --> CR
    W3 --> CR
    
    %% 事件流
    MB --> E1
    MB --> E2
    MB --> E3
    MB --> E4
    
    %% 反馈连接
    E1 --> SC
    E2 --> SC
    E3 --> SC
    E4 --> SC
 ```