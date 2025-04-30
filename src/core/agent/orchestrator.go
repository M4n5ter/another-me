package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/compose"
	"github.com/m4n5ter/another-me/src/core/trigger"
)

// OrchestratorInput 定义了编排器图的输入。
// 可能包括触发器类型、用户数据等。
type OrchestratorInput struct {
	TriggerType trigger.TriggerType
	Payload     any // 与触发器关联的可选负载
}

// OrchestratorState 表示由编排器管理的共享状态。
type OrchestratorState struct {
	// 示例状态字段 - 替换为实际所需的状态
	UserID         string
	Interests      []string
	PendingTasks   []any // 由信息收集器识别的任务
	NeedsUserInput bool
	LastRunOutput  any // 上次运行的 Worker Agent 的输出
	ShouldSleep    bool
}

// Orchestrator 协调不同的 Worker Agent。
type Orchestrator struct {
	logger *slog.Logger

	// Worker Agents
	infoGatherer   Agent // 信息收集器
	userInteractor Agent // 用户交互器
	learner        Agent // 学习器

	// Eino Graph
	compiledGraph compose.Runnable[OrchestratorInput, OrchestratorState]
}

// NewOrchestrator 创建一个新的编排器代理。
func NewOrchestrator( /* 传递必要的配置或依赖项 */ ) (*Orchestrator, error) {
	logger := slog.Default().WithGroup("orchestrator")

	// TODO: 将占位符替换为实际的代理初始化
	infoGatherer := &InfoGatherer{ /* config */ }
	userInteractor := &UserInteractor{ /* config */ }
	learner := &Learner{ /* config */ }

	o := &Orchestrator{
		logger:         logger,
		infoGatherer:   infoGatherer,
		userInteractor: userInteractor,
		learner:        learner,
	}

	// 在初始化期间构建和编译图
	err := o.buildAndCompileGraph(context.Background())
	if err != nil {
		return nil, fmt.Errorf("构建编排器图失败: %w", err)
	}

	logger.Info("编排器初始化成功。")
	return o, nil
}

// Run 使用特定的触发器输入启动主编排循环。
func (o *Orchestrator) Run(ctx context.Context, input OrchestratorInput) (*OrchestratorState, error) {
	o.logger.Info("编排器开始运行周期...", "trigger", input.TriggerType)

	if o.compiledGraph == nil {
		return nil, errors.New("编排器图未编译")
	}

	// 调用已编译的 Eino 图
	finalState, err := o.compiledGraph.Invoke(ctx, input)
	if err != nil {
		o.logger.Error("编排图调用失败", "error", err)
		// 根据错误，可能返回部分状态或 nil
		return nil, fmt.Errorf("图调用错误: %w", err)
	}

	// finalState 是图的 END 节点的输出
	o.logger.Info("编排器运行周期结束。", "shouldSleep", finalState.ShouldSleep)
	return &finalState, nil // 返回由图确定的最终状态
}

// buildAndCompileGraph 定义了用于编排的 Eino 图结构。
func (o *Orchestrator) buildAndCompileGraph(ctx context.Context) error {
	graphBuilder := compose.NewGraph[OrchestratorInput, OrchestratorState](compose.WithGenLocalState(func(ctx context.Context) *OrchestratorState {
		// TODO: 也许应该在这里从数据库加载状态

		return &OrchestratorState{}
	}))

	// --- 定义节点 ---
	// 根据输入加载初始状态的节点（例如，从负载中获取 UserID）
	err := graphBuilder.AddLambdaNode("load_state", compose.InvokableLambda(o.loadState))
	if err != nil {
		o.logger.Error("添加 load_state 节点失败", "error", err)
		return err
	}

	// 根据触发器和状态决定主要任务的节点
	err = graphBuilder.AddLambdaNode("dispatch_task", compose.InvokableLambda(o.dispatchTaskLogic), compose.WithStatePreHandler(func(ctx context.Context, input OrchestratorInput, state *OrchestratorState) (OrchestratorInput, error) {
		// TODO: 也许可以将输入更新到状态中，并且还可以将状态更新到数据库: input -> state -> db
		return input, nil
	}))
	if err != nil {
		o.logger.Error("添加 dispatch_task 节点失败", "error", err)
		return err
	}

	// 调用工作代理的节点
	err = graphBuilder.AddLambdaNode("call_info_gatherer", compose.InvokableLambda(o.callWorkerAgent(o.infoGatherer)))
	if err != nil {
		o.logger.Error("添加 call_info_gatherer 节点失败", "error", err)
		return err
	}

	err = graphBuilder.AddLambdaNode("call_user_interactor", compose.InvokableLambda(o.callWorkerAgent(o.userInteractor)))
	if err != nil {
		o.logger.Error("添加 call_user_interactor 节点失败", "error", err)
		return err
	}

	err = graphBuilder.AddLambdaNode("call_learner", compose.InvokableLambda(o.callWorkerAgent(o.learner)))
	if err != nil {
		o.logger.Error("添加 call_learner 节点失败", "error", err)
		return err
	}

	// 将 Worker Agent 的输出合并回主状态的节点
	err = graphBuilder.AddLambdaNode("merge_worker_output", compose.InvokableLambda(o.mergeWorkerOutput))
	if err != nil {
		o.logger.Error("添加 merge_worker_output 节点失败", "error", err)
		return err
	}

	// 决定编排器是否应休眠的节点
	err = graphBuilder.AddLambdaNode("decide_sleep", compose.InvokableLambda(o.decideSleep))
	if err != nil {
		o.logger.Error("添加 decide_sleep 节点失败", "error", err)
		return err
	}

	// 在结束前保存最终状态的节点（可选，可以是 decide_sleep 的一部分）
	err = graphBuilder.AddLambdaNode("save_state", compose.InvokableLambda(o.saveState))
	if err != nil {
		o.logger.Error("添加 save_state 节点失败", "error", err)
		return err
	}

	// --- 定义边和分支 ---
	err = graphBuilder.AddEdge(compose.START, "load_state")
	if err != nil {
		o.logger.Error("添加 START 到 load_state 边失败", "error", err)
		return err
	}

	err = graphBuilder.AddEdge("load_state", "dispatch_task")
	if err != nil {
		o.logger.Error("添加 load_state 到 dispatch_task 边失败", "error", err)
		return err
	}

	// 从 dispatch_task 分支到不同的 Worker Agent 或直接到休眠决策
	dispatchBranch := compose.NewGraphBranch[*OrchestratorState](
		// 条件函数：检查状态以决定下一个节点
		func(ctx context.Context, state *OrchestratorState) (string, error) {
			if state.NeedsUserInput {
				return "call_user_interactor", nil
			}
			if len(state.PendingTasks) > 0 {
				// 示例：如果存在任务但不需要用户输入，则优先进行信息收集
				// 或者可能需要交互来呈现任务？在此处细化逻辑。
				return "call_info_gatherer", nil // 或 "call_user_interactor"
			}
			// 添加学习器代理的条件，例如，定期运行或在交互后运行
			// if shouldRunLearner(state) { return "call_learner", nil }

			// 默认：如果没有其他事情可做，则决定是否需要休眠
			return "decide_sleep", nil
		},
		// 可能的目的地
		map[string]bool{
			"call_info_gatherer":   true,
			"call_user_interactor": true,
			"call_learner":         true,
			"decide_sleep":         true,
		},
	)
	err = graphBuilder.AddBranch("dispatch_task", dispatchBranch)
	if err != nil {
		o.logger.Error("添加 dispatch_task 分支失败", "error", err)
		return err
	}

	// 从工作节点到合并输出的边
	err = graphBuilder.AddEdge("call_info_gatherer", "merge_worker_output")
	if err != nil {
		o.logger.Error("添加 call_info_gatherer 到 merge_worker_output 边失败", "error", err)
		return err
	}

	// 合并后，循环回到分派任务或决定休眠
	// 选项 1：在工作代理运行后始终重新评估任务
	// _ = graphBuilder.AddEdge("merge_worker_output", "dispatch_task")
	// 选项 2：合并后直接进入休眠决策（更简单的第一步）
	err = graphBuilder.AddEdge("merge_worker_output", "decide_sleep")
	if err != nil {
		o.logger.Error("添加 merge_worker_output 到 decide_sleep 边失败", "error", err)
		return err
	}

	// 从休眠决策到保存状态，然后结束
	err = graphBuilder.AddEdge("decide_sleep", "save_state")
	if err != nil {
		o.logger.Error("添加 decide_sleep 到 save_state 边失败", "error", err)
		return err
	}

	err = graphBuilder.AddEdge("save_state", compose.END) // 使用 agent.END
	if err != nil {
		o.logger.Error("添加 save_state 到 END 边失败", "error", err)
		return err
	}

	// 编译图
	compiled, err := graphBuilder.Compile(ctx)
	if err != nil {
		o.logger.Error("编译编排图失败", "error", err)
		return err
	}
	o.compiledGraph = compiled // 存储编译后的图
	o.logger.Info("编排图构建并编译成功。")
	return nil
}

// --- LambdaNode 函数 ---

// loadState 加载运行的初始状态。
// 输入: OrchestratorInput; 输出: OrchestratorState
func (o *Orchestrator) loadState(ctx context.Context, input OrchestratorInput) (OrchestratorState, error) {
	o.logger.Info("加载状态...", "trigger", input.TriggerType)

	// TODO: 实现实际的状态加载逻辑（例如，根据负载中的 UserID 从数据库加载）
	state := OrchestratorState{
		UserID: "user123", // 示例
		// 加载其他字段...
	}
	return state, nil
}

// dispatchTaskLogic 根据当前状态决定下一个主要操作。
// 输入: OrchestratorState; 输出: OrchestratorState (传递给分支)
func (o *Orchestrator) dispatchTaskLogic(ctx context.Context, state OrchestratorState) (OrchestratorState, error) {
	o.logger.Info("分派任务...")
	// 这个节点主要将状态传递给后续的 Branch 节点。
	// 复杂的分派逻辑可以放在 Branch 条件函数本身中。
	// 如果需要在分支之前，我们可以在这里丰富状态。
	return state, nil
}

// callWorkerAgent 返回一个适用于 AddLambdaNode 的函数，该函数调用给定的工作代理。
// 输入: OrchestratorState; 输出: OrchestratorState (LastRunOutput 已更新)
func (o *Orchestrator) callWorkerAgent(worker Agent) func(context.Context, OrchestratorState) (OrchestratorState, error) {
	return func(ctx context.Context, state OrchestratorState) (OrchestratorState, error) {
		workerName := fmt.Sprintf("%T", worker) // 获取工作代理类型名称用于日志记录
		o.logger.Info(fmt.Sprintf("调用工作代理: %s", workerName))

		// TODO: 根据当前状态为特定工作代理准备适当的输入
		workerInput := state // 示例：传递整个状态，或派生特定输入

		output, err := worker.Run(ctx, workerInput)
		if err != nil {
			o.logger.Error(fmt.Sprintf("工作代理 %s 失败", workerName), "error", err)
			// 决定如何处理工作代理错误 - 用错误信息丰富状态？重试？使图失败？
			// 目前，只需将错误状态向上传递，合并节点需要处理它。
			state.LastRunOutput = map[string]any{"error": err.Error()} // 存储错误信息
			// 我们可能希望图终止或转到特定的错误处理节点
			// return state, err // 传播错误以停止图执行？
			return state, nil // 或者也许允许图继续进行合并/决定休眠？
		}

		o.logger.Info(fmt.Sprintf("工作代理 %s 完成", workerName))
		state.LastRunOutput = output // 存储成功的输出
		return state, nil
	}
}

// mergeWorkerOutput 将上次运行的工作代理的输出合并到状态中。
// 输入: OrchestratorState; 输出: OrchestratorState
func (o *Orchestrator) mergeWorkerOutput(ctx context.Context, state OrchestratorState) (OrchestratorState, error) {
	o.logger.Info("将工作代理输出合并到状态中...")
	// TODO: 实现基于 state.LastRunOutput 更新状态的逻辑
	// 这将很大程度上取决于每个工作代理返回的内容。
	// 示例：如果学习器更新了兴趣，则更新 state.Interests
	// 示例：如果信息收集器找到了任务，则更新 state.PendingTasks
	// 示例：如果交互器获得了反馈，则更新 state.NeedsUserInput 或触发学习器
	if state.LastRunOutput != nil {
		o.logger.Info("找到工作代理输出", "output", fmt.Sprintf("%+v", state.LastRunOutput)) // 记录输出详情
		// 在此处根据输出类型/内容添加合并逻辑
	}
	state.LastRunOutput = nil // 清除临时输出字段
	return state, nil
}

// decideSleep 根据当前状态确定编排器是否应休眠。
// 输入: OrchestratorState; 输出: OrchestratorState (ShouldSleep 已更新)
func (o *Orchestrator) decideSleep(ctx context.Context, state OrchestratorState) (OrchestratorState, error) {
	o.logger.Info("决定是否休眠...")
	// TODO: 实现决定休眠的逻辑。
	// 例如，如果没有待处理任务、不需要用户输入等，则休眠。
	shouldSleep := true // 目前默认为休眠
	if state.NeedsUserInput || len(state.PendingTasks) > 0 {
		shouldSleep = false
	}

	state.ShouldSleep = shouldSleep
	o.logger.Info("休眠决策", "shouldSleep", state.ShouldSleep)
	return state, nil
}

// saveState 在图结束前保存最终状态。
// 输入: OrchestratorState; 输出: OrchestratorState (图的最终输出)
func (o *Orchestrator) saveState(ctx context.Context, state OrchestratorState) (OrchestratorState, error) {
	o.logger.Info("保存最终状态...")
	// TODO: 实现实际的状态保存逻辑（例如，保存到数据库）
	// 保存 state.UserID, state.Interests, state.PendingTasks 等。

	// 这是最后一个节点，其输出成为图的输出。
	return state, nil
}
