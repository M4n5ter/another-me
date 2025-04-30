package agent

import "context"

type Learner struct {
	/* config */
}

var _ Agent = (*Learner)(nil)

func (w *Learner) Run(ctx context.Context, input any) (any, error) {
	/* 占位符 */
	return "学习到的用户画像", nil
}
