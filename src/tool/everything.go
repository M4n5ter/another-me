package tool

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
)

func Everything() ([]tool.BaseTool, error) {
	return StdIOMCP(context.Background(), "everything", "0.0.0", "npx", nil, "-y", "@modelcontextprotocol/server-everything")
}
