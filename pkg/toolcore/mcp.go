package toolcore

import (
	"context"
	"fmt"

	json "github.com/json-iterator/go"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// STDIOMCPTools 从STDIO MCP服务器获取所有工具
func STDIOMCPTools(command string, env []string, args ...string) ([]Tool, error) {
	c, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	if err := c.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	return getTools(c)
}

// SSEMCPTools 从SSE MCP服务器获取所有工具
func SSEMCPTools(sseURL string) ([]Tool, error) {
	c, err := client.NewSSEMCPClient(sseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	if err := c.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	return getTools(c)
}

// StreamableHTTPMCPTools 从Streamable HTTP MCP服务器获取所有工具
func StreamableHTTPMCPTools(url string) ([]Tool, error) {
	c, err := client.NewStreamableHttpClient(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	if err := c.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	return getTools(c)
}

func getTools(c client.MCPClient) ([]Tool, error) {
	// 初始化MCP客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "another-me",
		Version: "0.0.1",
	}
	_, err := c.Initialize(context.Background(), initRequest)
	if err != nil {
		errC := c.Close()
		err = fmt.Errorf("failed to initialize MCP client: %w", err)
		if errC != nil {
			err = fmt.Errorf("%w, failed to close MCP client: %w", err, errC)
		}
		return nil, err
	}

	// 获取工具列表
	toolsResult, err := c.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		errC := c.Close()
		err = fmt.Errorf("failed to list tools: %w", err)
		if errC != nil {
			err = fmt.Errorf("%w, failed to close MCP client: %w", err, errC)
		}
		return nil, err
	}

	if len(toolsResult.Tools) == 0 {
		errC := c.Close()
		err = fmt.Errorf("no tools available from MCP server")
		if errC != nil {
			err = fmt.Errorf("%w, failed to close MCP client: %w", err, errC)
		}
		return nil, err
	}

	tools := make([]Tool, 0, len(toolsResult.Tools))
	for _, mcpTool := range toolsResult.Tools {
		tools = append(tools, &mcpToolAdapter{
			client:  c,
			mcpTool: mcpTool,
		})
	}

	return tools, nil
}

// mcpToolAdapter 是一个适配器，将MCP工具转换为我们的Tool接口
type mcpToolAdapter struct {
	client  client.MCPClient
	mcpTool mcp.Tool
}

var _ Tool = (*mcpToolAdapter)(nil)

func (a *mcpToolAdapter) Schema(ctx context.Context) (ToolSchema, error) {
	schema := ToolSchema{
		Name: a.mcpTool.Name,
		Descriptions: map[string]string{
			"en": a.mcpTool.Description,
			"zh": a.mcpTool.Description,
		},
	}

	if len(a.mcpTool.RawInputSchema) > 0 {
		var rawSchema map[string]any
		if err := json.Unmarshal(a.mcpTool.RawInputSchema, &rawSchema); err != nil {
			return schema, fmt.Errorf("failed to unmarshal input schema: %w", err)
		}
		schema.InputParameters = ConvertJSONSchemaToParams(rawSchema)
	} else if a.mcpTool.InputSchema.Type != "" {
		schema.InputParameters = ConvertMCPInputSchemaToParams(a.mcpTool.InputSchema)
	}

	return schema, nil
}

func (a *mcpToolAdapter) Call(ctx context.Context, inputJSON string) (outputJSON string, err error) {
	args := make(map[string]any)
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("failed to unmarshal input: %w", err)
	}

	result, err := a.client.CallTool(ctx, mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: struct {
			Name      string    `json:"name"`
			Arguments any       `json:"arguments,omitempty"`
			Meta      *mcp.Meta `json:"_meta,omitempty"`
		}{
			Name:      a.mcpTool.Name,
			Arguments: args,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool: %w", err)
	}

	// 序列化结果
	resultJSON, err := json.MarshalToString(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("MCP tool execution failed: %s", resultJSON)
	}

	return resultJSON, nil
}
