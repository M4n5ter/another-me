package zh

const (
	TaskEvaluatorDescription = `
**重要**
如果前一条消息是 'task_evaluator' 工具调用, 则不应调用此工具。

**任务评估器**
完成任务或细化任务。

当满足以下条件时，应调用此工具：
- 用户所有的要求都已完全满足 (设置 is_complete=true)
- 避免不必要的迭代、冗余或浪费。
- 需要额外的输入/澄清 (设置 is_complete=false 并提供 'context')

当 is_complete=true 时, 'context' 将被忽略, 因为对话将终止。
当 is_complete=false 时, 你的 'context' 将被作为下一个提示词, 用来细化任务。
提供清晰、可操作的 'context' 来指导下一步, 'context' 应当用于指导自己更好地完成任务。
`

	TaskEvaluatorArgIsCompleteDescription = `
最终完成状态。仅当所有要求都完全满足时设置为 true。
`

	TaskEvaluatorArgContextDescription = `
'context' 提供了在任务不完整时所需的指导:
- 缺失元素的清晰描述
- 需要澄清的具体问题
- 完成任务的剩余步骤
可选当任务完成时。当 'is_complete' 为 false 时, 'context' 将在下一次作为提示词给你。
`
)
