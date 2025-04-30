package agent

import "context"

type InfoGatherer struct {
	/* config */
}

var _ Agent = (*InfoGatherer)(nil)

func (w *InfoGatherer) Run(ctx context.Context, input any) (any, error) {
	/* 占位符 */
	return "收集到的信息", nil
}
