package toolcore

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
)

// 示例：如何使用转换层在 genai.Schema 和 ParameterDefinition 之间进行转换

// ExampleGenaiToParams 演示如何将 genai.Schema 转换为 ParameterDefinition 列表
func ExampleGenaiToParams() {
	// 创建一个 genai.Schema 对象
	genaiSchema := &schema.Schema{
		Type:        genai.TypeObject,
		Title:       "User Information",
		Description: "A schema for user information",
		Properties: map[string]*schema.Schema{
			"name": {
				Type:        genai.TypeString,
				Description: "User's full name",
			},
			"age": {
				Type:        genai.TypeInteger,
				Description: "User's age in years",
			},
			"email": {
				Type:        genai.TypeString,
				Description: "User's email address",
			},
			"preferences": {
				Type:        genai.TypeObject,
				Description: "User preferences",
				Properties: map[string]*schema.Schema{
					"theme": {
						Type:        genai.TypeString,
						Description: "UI theme preference",
						Enum:        []string{"light", "dark", "auto"},
					},
					"notifications": {
						Type:        genai.TypeBoolean,
						Description: "Whether to receive notifications",
					},
				},
				Required: []string{"theme"},
			},
		},
		Required: []string{"name", "email"},
	}

	// 转换为 ParameterDefinition 列表
	params := ConvertGenaiSchemaToParams(genaiSchema)

	fmt.Printf("转换得到 %d 个参数:\n", len(params))
	for _, param := range params {
		fmt.Printf("- %s (%s): %s [必需: %t]\n",
			param.Name,
			param.Type,
			param.Description["en"],
			param.Required)

		// 打印嵌套属性（如果有）
		if param.Properties.IsSome() {
			nestedProps := param.Properties.Unwrap()
			for _, nested := range nestedProps {
				fmt.Printf("  - %s (%s): %s\n",
					nested.Name,
					nested.Type,
					nested.Description["en"])
			}
		}
	}
}

// ExampleParamsToGenai 演示如何将 ParameterDefinition 列表转换为 genai.Schema
func ExampleParamsToGenai() {
	// 创建 ParameterDefinition 列表
	params := []ParameterDefinition{
		{
			Name: "query",
			Type: ParamTypeString,
			Description: map[string]string{
				"en": "Search query string",
				"zh": "搜索查询字符串",
			},
			Required: true,
		},
		{
			Name: "limit",
			Type: ParamTypeInteger,
			Description: map[string]string{
				"en": "Maximum number of results",
				"zh": "最大结果数量",
			},
			Required:     false,
			DefaultValue: Some(any(10)),
		},
		{
			Name: "filters",
			Type: ParamTypeObject,
			Description: map[string]string{
				"en": "Search filters",
				"zh": "搜索过滤器",
			},
			Required: false,
			Properties: Some([]ParameterDefinition{
				{
					Name: "category",
					Type: ParamTypeString,
					Description: map[string]string{
						"en": "Filter by category",
						"zh": "按类别过滤",
					},
					Required:   false,
					EnumValues: Some([]any{"news", "blog", "documentation"}),
				},
				{
					Name: "date_range",
					Type: ParamTypeArray,
					Description: map[string]string{
						"en": "Date range filter [start, end]",
						"zh": "日期范围过滤器 [开始, 结束]",
					},
					Required: false,
					Items: Some(ParameterDefinition{
						Name: "item",
						Type: ParamTypeString,
						Description: map[string]string{
							"en": "Date in ISO format",
							"zh": "ISO 格式的日期",
						},
					}),
				},
			}),
		},
	}

	// 转换为 genai.Schema
	genaiSchema := ConvertParamsToGenaiSchema(params, "Search API", "Schema for search API parameters")

	fmt.Printf("生成的 genai.Schema:\n")
	fmt.Printf("Type: %s\n", genaiSchema.Type)
	fmt.Printf("Title: %s\n", genaiSchema.Title)
	fmt.Printf("Description: %s\n", genaiSchema.Description)
	fmt.Printf("Properties: %d\n", len(genaiSchema.Properties))
	fmt.Printf("Required: %v\n", genaiSchema.Required)

	// 打印属性详情
	for name, prop := range genaiSchema.Properties {
		fmt.Printf("- %s: %s (%s)\n", name, prop.Type, prop.Description)
		if len(prop.Enum) > 0 {
			fmt.Printf("  Enum: %v\n", prop.Enum)
		}
		if prop.Default != nil {
			fmt.Printf("  Default: %v\n", prop.Default)
		}
	}
}

// ExampleToolSchemaConversion 演示 ToolSchema 与 genai.Schema 之间的转换
func ExampleToolSchemaConversion() {
	// 创建一个 ToolSchema
	toolSchema := &ToolSchema{
		Name: "file_search",
		LocalizedNames: map[string]string{
			"en": "File Search",
			"zh": "文件搜索",
		},
		Descriptions: map[string]string{
			"en": "Search for files in the filesystem",
			"zh": "在文件系统中搜索文件",
		},
		InputParameters: []ParameterDefinition{
			{
				Name: "pattern",
				Type: ParamTypeString,
				Description: map[string]string{
					"en": "File name pattern (supports wildcards)",
					"zh": "文件名模式（支持通配符）",
				},
				Required: true,
			},
			{
				Name: "directory",
				Type: ParamTypeString,
				Description: map[string]string{
					"en": "Directory to search in",
					"zh": "要搜索的目录",
				},
				Required:     false,
				DefaultValue: Some(any(".")),
			},
			{
				Name: "recursive",
				Type: ParamTypeBoolean,
				Description: map[string]string{
					"en": "Whether to search recursively",
					"zh": "是否递归搜索",
				},
				Required:     false,
				DefaultValue: Some(any(false)),
			},
		},
		OutputParameters: []ParameterDefinition{
			{
				Name: "files",
				Type: ParamTypeArray,
				Description: map[string]string{
					"en": "List of found files",
					"zh": "找到的文件列表",
				},
				Items: Some(ParameterDefinition{
					Name: "file",
					Type: ParamTypeObject,
					Description: map[string]string{
						"en": "File information",
						"zh": "文件信息",
					},
					Properties: Some([]ParameterDefinition{
						{
							Name: "path",
							Type: ParamTypeString,
							Description: map[string]string{
								"en": "File path",
								"zh": "文件路径",
							},
						},
						{
							Name: "size",
							Type: ParamTypeInteger,
							Description: map[string]string{
								"en": "File size in bytes",
								"zh": "文件大小（字节）",
							},
						},
					}),
				}),
			},
		},
	}

	// 转换为 genai.Schema
	genaiSchema := ConvertToolSchemaToGenaiSchema(toolSchema)
	fmt.Printf("ToolSchema -> genai.Schema:\n")
	fmt.Printf("Title: %s\n", genaiSchema.Title)
	fmt.Printf("Description: %s\n", genaiSchema.Description)
	fmt.Printf("输入参数数量: %d\n", len(genaiSchema.Properties))

	// 反向转换
	localizedNames := map[string]string{
		"en": "File Search Tool",
		"zh": "文件搜索工具",
	}
	descriptions := map[string]string{
		"en": "Advanced file search functionality",
		"zh": "高级文件搜索功能",
	}

	reconstructedToolSchema := ConvertGenaiSchemaToToolSchema(
		genaiSchema,
		"advanced_file_search",
		localizedNames,
		descriptions,
	)

	fmt.Printf("\ngenai.Schema -> ToolSchema:\n")
	fmt.Printf("Name: %s\n", reconstructedToolSchema.Name)
	fmt.Printf("输入参数数量: %d\n", len(reconstructedToolSchema.InputParameters))
	fmt.Printf("输出参数数量: %d\n", len(reconstructedToolSchema.OutputParameters))
}

// ExampleIntegrationWithTool 演示如何在实际的工具实现中使用转换层
type ExampleTool struct {
	// 工具实现...
}

func (t *ExampleTool) Schema(ctx context.Context) (ToolSchema, error) {
	// 如果你已经有了 genai.Schema，可以轻松转换
	existingGenaiSchema := &schema.Schema{
		Type:        genai.TypeObject,
		Title:       "Example Tool",
		Description: "An example tool for demonstration",
		Properties: map[string]*schema.Schema{
			"input": {
				Type:        genai.TypeString,
				Description: "Input parameter",
			},
		},
		Required: []string{"input"},
	}

	// 转换为 ParameterDefinition
	inputParams := ConvertGenaiSchemaToParams(existingGenaiSchema)

	return ToolSchema{
		Name: "example_tool",
		LocalizedNames: map[string]string{
			"en": "Example Tool",
			"zh": "示例工具",
		},
		Descriptions: map[string]string{
			"en": "An example tool for demonstration",
			"zh": "用于演示的示例工具",
		},
		InputParameters: inputParams,
		OutputParameters: []ParameterDefinition{
			{
				Name: "result",
				Type: ParamTypeString,
				Description: map[string]string{
					"en": "Tool execution result",
					"zh": "工具执行结果",
				},
			},
		},
	}, nil
}

func (t *ExampleTool) Call(ctx context.Context, inputJSON string) (string, error) {
	// 工具执行逻辑...
	return `{"result": "success"}`, nil
}

// 编译时检查接口实现
var _ Tool = (*ExampleTool)(nil)
