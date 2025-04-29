package tool

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/m4n5ter/another-me/src/locale"
)

type (
	TaskEvaluatorTool struct {
		logger *slog.Logger
	}
	TaskEvaluatorToolArgs struct {
		IsComplete bool   `json:"is_complete"`
		Context    string `json:"context"`
	}
)

func NewTaskEvaluatorTool() *TaskEvaluatorTool {
	return &TaskEvaluatorTool{
		logger: slog.Default().WithGroup("task_evaluator"),
	}
}

var _ tool.InvokableTool = (*TaskEvaluatorTool)(nil)

func (t *TaskEvaluatorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "task_evaluator",
		Desc: locale.TaskEvaluatorDescription(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"is_complete": {
				Desc:     locale.TaskEvaluatorArgIsCompleteDescription(),
				Type:     schema.Boolean,
				Required: false,
			},
			"context": {
				Desc:     locale.TaskEvaluatorArgContextDescription(),
				Type:     schema.String,
				Required: false,
			},
		}),
	}, nil
}

func (t *TaskEvaluatorTool) InvokableRun(ctx context.Context, argumentsInJSON string, _opts ...tool.Option) (string, error) {
	var args TaskEvaluatorToolArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		t.logger.Error("failed to unmarshal arguments", "error", err)
		return "", err
	}

	if args.IsComplete {
		t.logger.Info("task is complete")
	} else {
		t.logger.Info("task is incomplete", "context", args.Context)
	}

	return argumentsInJSON, nil
}
