package agent

import "context"

type UserInteractor struct {
	/* config */
}

var _ Agent = (*UserInteractor)(nil)

func (w *UserInteractor) Run(ctx context.Context, input any) (any, error) {
	/* 占位符 */
	return "用户反馈", nil
}
