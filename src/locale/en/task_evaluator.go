package en

const (
	TaskEvaluatorDescription = `
**Important**
If previous message is a 'task_evaluator' call, then you shouldn't call this tool.

**Task Evaluator**
Finalize or request refinement for the current task.

Call this when:
- All user requirements are fully satisfied (set is_complete=true)
- Avoids unnecessary iterations, redundancy, or waste.
- Additional input/clarification is needed (set is_complete=false with context)

When is_complete=true, the context is ignored, because the dialogue will terminate.
When is_complete=false, your context will be included in the system's next prompt, enabling iterative task refinement.
Provide clear, actionable contexts to guide the next steps, the context should be used to guide yourself to complete the task.
`

	TaskEvaluatorArgIsCompleteDescription = `Final completion state. Set to true ONLY when all requirements are fully satisfied.`

	TaskEvaluatorArgContextDescription = `
Context provides required guidance when task is incomplete:
- Clear description of missing elements
- Specific questions needing clarification
- Remaining steps to completion
Optional when task is complete. When 'is_complete' is false, the 'context' becomes the next prompt to you.
`
)
