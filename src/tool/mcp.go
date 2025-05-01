package tool

import (
	"context"
	"log/slog"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var mcpLogger = slog.Default().WithGroup("mcp")

func StdIOMCP(ctx context.Context, name, version string, command string, env []string, args ...string) ([]tool.BaseTool, error) {
	cli, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		mcpLogger.Error("创建 STDIO MCP 客户端失败", "error", err, "command", command, "env", env, "args", args)
		return nil, err
	}

	return getMCP(ctx, cli, name, version)
}

func SSEMCP(ctx context.Context, url, name, version string) ([]tool.BaseTool, error) {
	cli, err := client.NewSSEMCPClient(url)
	if err != nil {
		mcpLogger.Error("创建 SSE MCP 客户端失败", "error", err, "url", url)
		return nil, err
	}

	return getMCP(ctx, cli, name, version)
}

func StreamableHttpMCP(ctx context.Context, url, name, version string) ([]tool.BaseTool, error) {
	cli, err := client.NewStreamableHttpClient(url)
	if err != nil {
		mcpLogger.Error("创建 Streamable HTTP MCP 客户端失败", "error", err, "url", url)
		return nil, err
	}

	return getMCP(ctx, cli, name, version)
}

func getMCP(ctx context.Context, cli *client.Client, name, version string) ([]tool.BaseTool, error) {
	logger := mcpLogger.With("name", name, "version", version)

	err := cli.Start(ctx)
	if err != nil {
		logger.Error("启动 MCP 客户端失败", "error", err)
		return nil, err
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    name,
		Version: version,
	}

	response, err := cli.Initialize(ctx, initRequest)
	if err != nil {
		logger.Error("初始化 MCP 客户端失败", "error", err)
		return nil, err
	}

	logger.Info("初始化 MCP 客户端成功", "response", response)

	tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
	if err != nil {
		logger.Error("获取 MCP 工具失败", "error", err)
		return nil, err
	}

	return tools, nil
}
