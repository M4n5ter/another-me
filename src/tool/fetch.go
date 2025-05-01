package tool

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
)

// FetchTools 获取 fetch 工具。目前不可用！！！ mcp-server-fetch 给的 schema 中存在 exclusiveMinimum=0 这样的键值对，但是 openapi3.Schema{} 中的 exclusiveMinimum 类型是 bool
func FetchTools() ([]tool.BaseTool, error) {
	return StdIOMCP(context.Background(), "fetch", "0.0.0", "uvx", nil, "mcp-server-fetch@latest", "--ignore-robots-txt", `--user-agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0"`)
}
