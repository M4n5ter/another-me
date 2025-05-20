package toolcore

import (
	"context"
	"fmt"
	"slices"

	json "github.com/json-iterator/go"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	. "github.com/m4n5ter/another-me/pkg/option"
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

		schema.InputParameters = convertRawSchemaToParams(rawSchema)
	} else if a.mcpTool.InputSchema.Type != "" {
		schema.InputParameters = convertInputSchemaToParams(a.mcpTool.InputSchema)
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
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	if result.IsError {
		return "", fmt.Errorf("MCP tool execution failed: %s", string(resultBytes))
	}

	return string(resultBytes), nil
}

// convertInputSchemaToParams 将MCP的ToolInputSchema转换为 []ParameterDefinition
func convertInputSchemaToParams(schema mcp.ToolInputSchema) []ParameterDefinition {
	params := make([]ParameterDefinition, 0, len(schema.Properties))

	for name, propRaw := range schema.Properties {
		prop, ok := propRaw.(map[string]any)
		if !ok {
			continue
		}

		param := ParameterDefinition{
			Name: name,
			Description: map[string]string{
				"en": getStringProp(prop, "description", ""),
				"zh": getStringProp(prop, "description", ""),
			},
			Required: isRequired(name, schema.Required),
		}

		// 设置参数类型
		setParamType(&param, prop)

		// 处理枚举值
		if enumValues, ok := prop["enum"]; ok {
			if enumArr, ok := enumValues.([]any); ok && len(enumArr) > 0 {
				param.EnumValues = Some(enumArr)
			}
		}

		params = append(params, param)
	}

	return params
}

// convertRawSchemaToParams 将原始JSON Schema转换为参数定义
func convertRawSchemaToParams(rawSchema map[string]any) []ParameterDefinition {
	params := []ParameterDefinition{}

	// 获取属性对象
	properties, ok := rawSchema["properties"].(map[string]any)
	if !ok {
		return params
	}

	// 获取必需属性列表
	var required []string
	if reqArray, ok := rawSchema["required"].([]any); ok {
		for _, v := range reqArray {
			if str, ok := v.(string); ok {
				required = append(required, str)
			}
		}
	}

	// 解析每个属性
	for name, propRaw := range properties {
		prop, ok := propRaw.(map[string]any)
		if !ok {
			continue
		}

		// 创建基本参数定义
		param := ParameterDefinition{
			Name: name,
			Description: map[string]string{
				"en": getStringProp(prop, "description", ""),
			},
			Required: isRequired(name, required),
		}

		// 设置参数类型
		setParamType(&param, prop)

		// 处理枚举值
		if enumValues, ok := prop["enum"]; ok {
			if enumArr, ok := enumValues.([]any); ok && len(enumArr) > 0 {
				param.EnumValues = Some(enumArr)
			}
		}

		params = append(params, param)
	}

	return params
}

func setParamType(param *ParameterDefinition, prop map[string]any) {
	typeStr := getStringProp(prop, "type", "")
	switch typeStr {
	case "string":
		param.Type = ParamTypeString
	case "number":
		param.Type = ParamTypeNumber
	case "integer":
		param.Type = ParamTypeInteger
	case "boolean":
		param.Type = ParamTypeBoolean
	case "object":
		param.Type = ParamTypeObject
	case "array":
		param.Type = ParamTypeArray
		// 处理数组项类型
		if items, ok := prop["items"].(map[string]any); ok {
			itemType := getStringProp(items, "type", "")
			itemParam := ParameterDefinition{
				Type: paramTypeFromString(itemType),
			}
			// 设置项描述
			if desc, ok := items["description"].(string); ok {
				itemParam.Description = map[string]string{"en": desc}
			}
			param.Items = Some(itemParam)
		}
	default:
		param.Type = ParamTypeAny
	}
}

// paramTypeFromString 将字符串类型转换为ParamType
func paramTypeFromString(typeStr string) ParameterType {
	switch typeStr {
	case "string":
		return ParamTypeString
	case "number":
		return ParamTypeNumber
	case "integer":
		return ParamTypeInteger
	case "boolean":
		return ParamTypeBoolean
	case "object":
		return ParamTypeObject
	case "array":
		return ParamTypeArray
	default:
		return ParamTypeAny
	}
}

// getStringProp 获取属性中的字符串值
func getStringProp(prop map[string]any, key, defaultValue string) string {
	if val, ok := prop[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// isRequired 检查参数是否是必需的
func isRequired(name string, required []string) bool {
	return slices.Contains(required, name)
}
