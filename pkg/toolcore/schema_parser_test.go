package toolcore

import (
	"sort"
	"testing"

	json "github.com/json-iterator/go"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"

	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
)

func TestConvertJSONSchemaToParams_Simple(t *testing.T) {
	rawSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":        map[string]any{"type": "string", "description": "User's name"},
			"age":         map[string]any{"type": "integer", "description": "User's age"},
			"is_verified": map[string]any{"type": "boolean", "description": "Verification status"},
		},
		"required": []any{"name"},
	}

	params := ConvertJSONSchemaToParams(rawSchema)
	require.Len(t, params, 3)

	// 排序 params 以确保稳定的断言顺序
	sort.Slice(params, func(i, j int) bool { return params[i].Name < params[j].Name })

	idx := sort.Search(len(params), func(i int) bool { return params[i].Name >= "age" })
	require.True(t, idx < len(params) && params[idx].Name == "age")
	assert.Equal(t, ParamTypeInteger, params[idx].Type)
	assert.False(t, params[idx].Required)

	idx = sort.Search(len(params), func(i int) bool { return params[i].Name >= "is_verified" })
	require.True(t, idx < len(params) && params[idx].Name == "is_verified")
	assert.Equal(t, ParamTypeBoolean, params[idx].Type)
	assert.False(t, params[idx].Required)

	idx = sort.Search(len(params), func(i int) bool { return params[i].Name >= "name" })
	require.True(t, idx < len(params) && params[idx].Name == "name")
	assert.Equal(t, ParamTypeString, params[idx].Type)
	assert.Equal(t, "User's name", params[idx].Description["en"])
	assert.True(t, params[idx].Required)
}

func TestConvertJSONSchemaToParams_WithEnum(t *testing.T) {
	rawSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{
				"type":        "string",
				"description": "User status",
				"enum":        []any{"active", "inactive", "pending"},
			},
		},
	}
	params := ConvertJSONSchemaToParams(rawSchema)
	require.Len(t, params, 1)
	statusParam := params[0]
	assert.Equal(t, "status", statusParam.Name)
	assert.True(t, statusParam.EnumValues.IsSome())
	assert.Equal(t, []any{"active", "inactive", "pending"}, statusParam.EnumValues.Unwrap())
}

func TestConvertJSONSchemaToParams_NestedObject(t *testing.T) {
	rawSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"user": map[string]any{
				"type":        "object",
				"description": "User details",
				"properties": map[string]any{
					"id":   map[string]any{"type": "string", "description": "User ID"},
					"role": map[string]any{"type": "string", "description": "User role"},
				},
				"required": []any{"id"},
			},
		},
		"required": []any{"user"},
	}

	params := ConvertJSONSchemaToParams(rawSchema)
	require.Len(t, params, 1)

	userParam := params[0]
	assert.Equal(t, "user", userParam.Name)
	assert.Equal(t, ParamTypeObject, userParam.Type)
	assert.True(t, userParam.Required)
	assert.True(t, userParam.Properties.IsSome())

	userProps := userParam.Properties.Unwrap()
	require.Len(t, userProps, 2)
	sort.Slice(userProps, func(i, j int) bool { return userProps[i].Name < userProps[j].Name })

	assert.Equal(t, "id", userProps[0].Name)
	assert.Equal(t, ParamTypeString, userProps[0].Type)
	assert.True(t, userProps[0].Required)

	assert.Equal(t, "role", userProps[1].Name)
	assert.Equal(t, ParamTypeString, userProps[1].Type)
	assert.False(t, userProps[1].Required)
}

func TestConvertJSONSchemaToParams_SimpleArray(t *testing.T) {
	rawSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tags": map[string]any{
				"type":        "array",
				"description": "List of tags",
				"items": map[string]any{
					"type":        "string",
					"description": "A single tag",
				},
			},
		},
	}
	params := ConvertJSONSchemaToParams(rawSchema)
	require.Len(t, params, 1)

	tagsParam := params[0]
	assert.Equal(t, "tags", tagsParam.Name)
	assert.Equal(t, ParamTypeArray, tagsParam.Type)
	assert.True(t, tagsParam.Items.IsSome())

	itemSchema := tagsParam.Items.Unwrap()
	assert.Equal(t, ParamTypeString, itemSchema.Type)
	assert.Equal(t, "A single tag", itemSchema.Description["en"])
}

func TestConvertJSONSchemaToParams_ArrayOfObjects(t *testing.T) {
	rawSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"users": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
						"id":   map[string]any{"type": "integer"},
					},
					"required": []any{"name"},
				},
			},
		},
	}
	params := ConvertJSONSchemaToParams(rawSchema)
	require.Len(t, params, 1)

	usersParam := params[0]
	assert.Equal(t, ParamTypeArray, usersParam.Type)
	assert.True(t, usersParam.Items.IsSome())

	itemObjSchema := usersParam.Items.Unwrap()
	assert.Equal(t, ParamTypeObject, itemObjSchema.Type)
	assert.True(t, itemObjSchema.Properties.IsSome())

	itemProps := itemObjSchema.Properties.Unwrap()
	require.Len(t, itemProps, 2)
	// 按名称排序 itemProps 以确保一致的断言顺序
	sort.Slice(itemProps, func(i, j int) bool {
		return itemProps[i].Name < itemProps[j].Name
	})

	// 排序后的预期顺序: id, name
	assert.Equal(t, "id", itemProps[0].Name)
	assert.Equal(t, ParamTypeInteger, itemProps[0].Type)
	assert.False(t, itemProps[0].Required)

	assert.Equal(t, "name", itemProps[1].Name)
	assert.Equal(t, ParamTypeString, itemProps[1].Type)
	assert.True(t, itemProps[1].Required, "Name within array item should be required")
}

func TestConvertJSONSchemaToParams_EmptyProperties(t *testing.T) {
	rawSchema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	params := ConvertJSONSchemaToParams(rawSchema)
	assert.Empty(t, params)

	rawSchemaNoPropsField := map[string]any{
		"type": "object",
	}
	paramsNoPropsField := ConvertJSONSchemaToParams(rawSchemaNoPropsField)
	assert.Empty(t, paramsNoPropsField)

	rawSchemaObjectTypeOnly := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"metadata": map[string]any{"type": "object", "description": "Any metadata"},
		},
	}
	paramsObjectOnly := ConvertJSONSchemaToParams(rawSchemaObjectTypeOnly)
	require.Len(t, paramsObjectOnly, 1)
	metaParam := paramsObjectOnly[0]
	assert.Equal(t, "metadata", metaParam.Name)
	assert.Equal(t, ParamTypeObject, metaParam.Type)
	assert.True(t, metaParam.Properties.IsSome(), "Properties should be Some for type object")
	assert.Empty(t, metaParam.Properties.Unwrap(), "Properties list should be empty for bare object type")
}

func TestConvertParamsToRawSchema_Simple(t *testing.T) {
	params := []ParameterDefinition{
		{Name: "name", Type: ParamTypeString, Description: map[string]string{"en": "User's name"}, Required: true},
		{Name: "age", Type: ParamTypeInteger, Description: map[string]string{"en": "User's age"}},
	}

	rawSchema := ConvertParamsToRawSchema(params, "TestSchema", "A test schema")

	assert.Equal(t, "object", rawSchema["type"])
	assert.Equal(t, "TestSchema", rawSchema["title"])
	assert.Equal(t, "A test schema", rawSchema["description"])

	properties, ok := rawSchema["properties"].(map[string]any)
	require.True(t, ok)
	require.Len(t, properties, 2)

	nameProp, ok := properties["name"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", nameProp["type"])
	assert.Equal(t, "User's name", nameProp["description"])

	required, ok := rawSchema["required"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"name"}, required)
}

func TestConvertParamsToRawSchema_NestedObject(t *testing.T) {
	params := []ParameterDefinition{
		{
			Name: "user", Type: ParamTypeObject, Required: true,
			Properties: Some([]ParameterDefinition{
				{Name: "id", Type: ParamTypeString, Required: true},
				{Name: "role", Type: ParamTypeString},
			}),
		},
	}
	rawSchema := ConvertParamsToRawSchema(params, "", "")
	properties, _ := rawSchema["properties"].(map[string]any)
	userProp, _ := properties["user"].(map[string]any)

	assert.Equal(t, "object", userProp["type"])
	userSubProps, _ := userProp["properties"].(map[string]any)
	require.Len(t, userSubProps, 2)
	idSubProp, _ := userSubProps["id"].(map[string]any)
	assert.Equal(t, "string", idSubProp["type"])

	userRequired, _ := userProp["required"].([]string)
	assert.Equal(t, []string{"id"}, userRequired)

	topRequired, _ := rawSchema["required"].([]string)
	assert.Equal(t, []string{"user"}, topRequired)
}

func TestConvertParamsToRawSchema_ArrayOfObjects(t *testing.T) {
	params := []ParameterDefinition{
		{
			Name: "users", Type: ParamTypeArray,
			Items: Some(ParameterDefinition{
				Type: ParamTypeObject,
				Properties: Some([]ParameterDefinition{
					{Name: "name", Type: ParamTypeString, Required: true},
					{Name: "id", Type: ParamTypeInteger},
				}),
			}),
		},
	}
	rawSchema := ConvertParamsToRawSchema(params, "", "")
	properties, _ := rawSchema["properties"].(map[string]any)
	usersProp, _ := properties["users"].(map[string]any)
	assert.Equal(t, "array", usersProp["type"])

	itemsSchema, ok := usersProp["items"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", itemsSchema["type"])

	itemSubProps, _ := itemsSchema["properties"].(map[string]any)
	require.Len(t, itemSubProps, 2)
	nameItemProp, _ := itemSubProps["name"].(map[string]any)
	assert.Equal(t, "string", nameItemProp["type"])

	itemRequired, _ := itemsSchema["required"].([]string)
	assert.Equal(t, []string{"name"}, itemRequired)
}

func TestConvertParamsToRawSchema_EmptyParams(t *testing.T) {
	params := []ParameterDefinition{}
	rawSchema := ConvertParamsToRawSchema(params, "Empty", "Empty schema")
	assert.Equal(t, "object", rawSchema["type"])
	assert.Equal(t, "Empty", rawSchema["title"])
	properties, ok := rawSchema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, properties, "Properties map should be empty for no params")
	_, requiredOk := rawSchema["required"]
	assert.False(t, requiredOk, "Required field should not exist for no params/no required params")
}

func TestConvertParamsToRawSchema_ObjectParamNoProperties(t *testing.T) {
	params := []ParameterDefinition{
		{
			Name: "metadata", Type: ParamTypeObject, Description: map[string]string{"en": "Any object"},
		},
	}
	rawSchema := ConvertParamsToRawSchema(params, "", "")
	properties, _ := rawSchema["properties"].(map[string]any)
	metadataProp, _ := properties["metadata"].(map[string]any)

	assert.Equal(t, "object", metadataProp["type"])
	metadataSubProps, ok := metadataProp["properties"].(map[string]any)
	require.True(t, ok, "properties field should exist for object type")
	assert.Empty(t, metadataSubProps, "properties of metadata object should be empty")
	_, requiredOk := metadataProp["required"]
	assert.False(t, requiredOk, "required should not be present if no sub-properties are required")

	paramsWithEmptyProps := []ParameterDefinition{
		{
			Name: "data", Type: ParamTypeObject,
			Properties: Some([]ParameterDefinition{}),
		},
	}
	rawSchemaEmpty := ConvertParamsToRawSchema(paramsWithEmptyProps, "", "")
	properties2, _ := rawSchemaEmpty["properties"].(map[string]any)
	dataProp, _ := properties2["data"].(map[string]any)
	assert.Equal(t, "object", dataProp["type"])
	dataSubProps, ok := dataProp["properties"].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, dataSubProps)
}

func TestConvertMCPInputSchemaToParams(t *testing.T) {
	mcpSchema := mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"query":       map[string]any{"type": "string", "description": "Search query"},
			"page_number": map[string]any{"type": "integer", "description": "Page number"},
			"filters": map[string]any{
				"type":        "object",
				"description": "Search filters",
				"properties": map[string]any{
					"type": map[string]any{"type": "string", "enum": []any{"image", "video"}},
					"size": map[string]any{"type": "integer"},
				},
				"required": []any{"type"},
			},
		},
		Required: []string{"query"},
	}

	params := ConvertMCPInputSchemaToParams(mcpSchema)
	require.Len(t, params, 3)

	// 按名称排序 params 以确保一致的断言顺序
	sort.Slice(params, func(i, j int) bool {
		return params[i].Name < params[j].Name
	})

	// 排序后的预期顺序: filters, page_number, query
	filtersParam := params[0]
	assert.Equal(t, "filters", filtersParam.Name)
	assert.Equal(t, ParamTypeObject, filtersParam.Type)
	assert.False(t, filtersParam.Required)
	assert.True(t, filtersParam.Properties.IsSome())

	filterProps := filtersParam.Properties.Unwrap()
	require.Len(t, filterProps, 2)
	// 按名称排序 filterProps 以确保一致的断言顺序
	sort.Slice(filterProps, func(i, j int) bool {
		return filterProps[i].Name < filterProps[j].Name
	})

	// 排序后的预期顺序: size, type
	assert.Equal(t, "size", filterProps[0].Name)
	assert.Equal(t, ParamTypeInteger, filterProps[0].Type)
	assert.False(t, filterProps[0].Required)

	assert.Equal(t, "type", filterProps[1].Name)
	assert.Equal(t, ParamTypeString, filterProps[1].Type)
	assert.True(t, filterProps[1].EnumValues.IsSome())
	assert.True(t, filterProps[1].Required)

	pageNumberParam := params[1]
	assert.Equal(t, "page_number", pageNumberParam.Name)
	assert.False(t, pageNumberParam.Required)
	assert.Equal(t, ParamTypeInteger, pageNumberParam.Type)

	queryParam := params[2]
	assert.Equal(t, "query", queryParam.Name)
	assert.True(t, queryParam.Required)
	assert.Equal(t, ParamTypeString, queryParam.Type)
}

func TestConvertMCPInputSchemaToParams_NilProperties(t *testing.T) {
	mcpSchema := mcp.ToolInputSchema{
		Type:       "object",
		Properties: nil,
		Required:   nil,
	}
	params := ConvertMCPInputSchemaToParams(mcpSchema)
	assert.Empty(t, params, "Should return empty params if MCP schema properties is nil")
}

func TestSchemaParamsRoundTrip(t *testing.T) {
	originalRawSchema := map[string]any{
		"type":        "object",
		"title":       "Complex User Schema",
		"description": "Schema for complex user object with nesting and arrays.",
		"properties": map[string]any{
			"user_id": map[string]any{"type": "string", "description": "Unique user identifier"},
			"profile": map[string]any{
				"type":        "object",
				"description": "User profile information",
				"properties": map[string]any{
					"full_name": map[string]any{"type": "string", "description": "User's full name"},
					"email":     map[string]any{"type": "string", "description": "User's email address"},
					"settings": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"notifications_enabled": map[string]any{"type": "boolean", "default": true},
						},
					},
				},
				"required": []string{"email", "full_name"}, // 使用 []string 以确保一致性，按字母顺序排序
			},
			"tags": map[string]any{
				"type":        "array",
				"description": "List of user tags",
				"items": map[string]any{
					"type":        "string",
					"description": "A single tag",
				},
			},
			"contacts": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"type":  map[string]any{"type": "string", "enum": []any{"phone", "email"}},
						"value": map[string]any{"type": "string"},
					},
					"required": []string{"type", "value"}, // 使用 []string 以确保一致性，按字母顺序排序
				},
			},
			"empty_object_prop": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		"required": []string{"profile", "user_id"}, // 使用 []string 以确保一致性，按字母顺序排序
	}

	params := ConvertJSONSchemaToParams(originalRawSchema)
	// 调试: 检查顶层 params 的 Required 标志
	for _, p := range params {
		t.Logf("Top-level param: %s, Required: %v", p.Name, p.Required)
		if p.Name == "profile" && p.Properties.IsSome() {
			for _, subP := range p.Properties.Unwrap() {
				t.Logf("  Profile sub-param: %s, Required: %v", subP.Name, subP.Required)
			}
		}
		if p.Name == "contacts" && p.Items.IsSome() {
			contactItem := p.Items.Unwrap()
			if contactItem.Type == ParamTypeObject && contactItem.Properties.IsSome() {
				for _, subP := range contactItem.Properties.Unwrap() {
					t.Logf("  Contacts.item sub-param: %s, Required: %v", subP.Name, subP.Required)
				}
			}
		}
	}

	reconstructedRawSchema := ConvertParamsToRawSchema(params,
		getStringFromMap(originalRawSchema, "title", ""),
		getStringFromMap(originalRawSchema, "description", ""),
	)

	assert.Equal(t, originalRawSchema["type"], reconstructedRawSchema["type"])
	assert.Equal(t, originalRawSchema["title"], reconstructedRawSchema["title"])
	assert.Equal(t, originalRawSchema["description"], reconstructedRawSchema["description"])

	// 比较顶层 required 字段 (顺序不敏感)
	originalRequired, _ := originalRawSchema["required"].([]string)
	reconstructedRequired, _ := reconstructedRawSchema["required"].([]string)
	assert.ElementsMatch(t, originalRequired, reconstructedRequired)

	originalProps := originalRawSchema["properties"].(map[string]any)
	reconstructedProps := reconstructedRawSchema["properties"].(map[string]any)
	require.Equal(t, len(originalProps), len(reconstructedProps), "Number of properties should match")

	// 比较 profile (嵌套对象)
	originalProfile := originalProps["profile"].(map[string]any)
	reconstructedProfile := reconstructedProps["profile"].(map[string]any)
	assert.Equal(t, originalProfile["type"], reconstructedProfile["type"])
	originalProfileRequired, _ := originalProfile["required"].([]string)
	reconstructedProfileRequired, _ := reconstructedProfile["required"].([]string)
	assert.ElementsMatch(t, originalProfileRequired, reconstructedProfileRequired)
	// 进一步检查 profile.properties 可以添加，例如使用 EqualValues 在子映射上
	assert.EqualValues(t, originalProfile["properties"], reconstructedProfile["properties"], "Profile properties should be equal")

	// 检查 contacts 数组项的 required 字段
	var originalContactsItemsRequired []string
	if originalContactsProp, ok := originalProps["contacts"].(map[string]any); ok {
		if originalContactsItemsMap, ok := originalContactsProp["items"].(map[string]any); ok {
			if req, found := originalContactsItemsMap["required"]; found {
				originalContactsItemsRequired, _ = req.([]string)
			}
		}
	}
	if originalContactsItemsRequired == nil {
		originalContactsItemsRequired = []string{}
	}

	var reconstructedContactsItemsRequired []string
	if reconstructedContactsProp, ok := reconstructedProps["contacts"].(map[string]any); ok {
		if reconstructedContactsItemsMap, ok := reconstructedContactsProp["items"].(map[string]any); ok {
			if req, found := reconstructedContactsItemsMap["required"]; found {
				reconstructedContactsItemsRequired, _ = req.([]string)
			}
		}
	}
	if reconstructedContactsItemsRequired == nil {
		reconstructedContactsItemsRequired = []string{}
	}
	assert.ElementsMatch(t, originalContactsItemsRequired, reconstructedContactsItemsRequired, "Contacts items required fields should match")

	// 将两者转换为 JSON 字符串以进行更可靠的比较，忽略 map 顺序问题
	originalJSON, err := json.MarshalIndent(originalRawSchema, "", "  ")
	require.NoError(t, err, "Failed to marshal originalRawSchema to JSON")

	reconstructedJSON, err := json.MarshalIndent(reconstructedRawSchema, "", "  ")
	require.NoError(t, err, "Failed to marshal reconstructedRawSchema to JSON")

	assert.JSONEq(t, string(originalJSON), string(reconstructedJSON), "Full schema round trip should be equivalent as JSON")
}

func TestConvertGenaiSchemaToParams(t *testing.T) {
	// 创建一个简单的 genai.Schema
	genaiSchema := &schema.Schema{
		Type:        genai.TypeObject,
		Title:       "Test Schema",
		Description: "A test schema for conversion",
		Properties: map[string]*schema.Schema{
			"name": {
				Type:        genai.TypeString,
				Description: "User name",
			},
			"age": {
				Type:        genai.TypeInteger,
				Description: "User age",
			},
			"is_active": {
				Type:        genai.TypeBoolean,
				Description: "Whether user is active",
			},
		},
		Required: []string{"name", "age"},
	}

	// 转换为 ParameterDefinition
	params := ConvertGenaiSchemaToParams(genaiSchema)

	// 验证结果
	assert.Len(t, params, 3)

	// 验证 age 参数 (按字母顺序：age 在第 0 位)
	ageParam := params[0]
	assert.Equal(t, "age", ageParam.Name)
	assert.Equal(t, ParamTypeInteger, ageParam.Type)
	assert.True(t, ageParam.Required)
	assert.Equal(t, "User age", ageParam.Description["en"])

	// 验证 is_active 参数 (按字母顺序：is_active 在第 1 位)
	isActiveParam := params[1]
	assert.Equal(t, "is_active", isActiveParam.Name)
	assert.Equal(t, ParamTypeBoolean, isActiveParam.Type)
	assert.False(t, isActiveParam.Required)
	assert.Equal(t, "Whether user is active", isActiveParam.Description["en"])

	// 验证 name 参数 (按字母顺序：name 在第 2 位)
	nameParam := params[2]
	assert.Equal(t, "name", nameParam.Name)
	assert.Equal(t, ParamTypeString, nameParam.Type)
	assert.True(t, nameParam.Required)
	assert.Equal(t, "User name", nameParam.Description["en"])
}

func TestConvertParamsToGenaiSchema(t *testing.T) {
	// 创建 ParameterDefinition 列表
	params := []ParameterDefinition{
		{
			Name: "name",
			Type: ParamTypeString,
			Description: map[string]string{
				"en": "User name",
				"zh": "用户名",
			},
			Required: true,
		},
		{
			Name: "age",
			Type: ParamTypeInteger,
			Description: map[string]string{
				"en": "User age",
				"zh": "用户年龄",
			},
			Required: true,
		},
		{
			Name: "tags",
			Type: ParamTypeArray,
			Description: map[string]string{
				"en": "User tags",
				"zh": "用户标签",
			},
			Required: false,
			Items: Some(ParameterDefinition{
				Name: "item",
				Type: ParamTypeString,
				Description: map[string]string{
					"en": "Tag item",
					"zh": "标签项",
				},
			}),
		},
	}

	// 转换为 genai.Schema
	genaiSchema := ConvertParamsToGenaiSchema(params, "Test Schema", "A test schema")

	// 验证结果
	assert.Equal(t, genai.TypeObject, genaiSchema.Type)
	assert.Equal(t, "Test Schema", genaiSchema.Title)
	assert.Equal(t, "A test schema", genaiSchema.Description)
	assert.Len(t, genaiSchema.Properties, 3)
	assert.ElementsMatch(t, []string{"name", "age"}, genaiSchema.Required)

	// 验证 name 属性
	nameSchema := genaiSchema.Properties["name"]
	assert.NotNil(t, nameSchema)
	assert.Equal(t, genai.TypeString, nameSchema.Type)
	assert.Equal(t, "User name", nameSchema.Description)

	// 验证 age 属性
	ageSchema := genaiSchema.Properties["age"]
	assert.NotNil(t, ageSchema)
	assert.Equal(t, genai.TypeInteger, ageSchema.Type)
	assert.Equal(t, "User age", ageSchema.Description)

	// 验证 tags 属性
	tagsSchema := genaiSchema.Properties["tags"]
	assert.NotNil(t, tagsSchema)
	assert.Equal(t, genai.TypeArray, tagsSchema.Type)
	assert.Equal(t, "User tags", tagsSchema.Description)
	assert.NotNil(t, tagsSchema.Items)
	assert.Equal(t, genai.TypeString, tagsSchema.Items.Type)
}

func TestConvertToolSchemaToGenaiSchema(t *testing.T) {
	// 创建 ToolSchema
	toolSchema := &ToolSchema{
		Name: "test_tool",
		LocalizedNames: map[string]string{
			"en": "Test Tool",
			"zh": "测试工具",
		},
		Descriptions: map[string]string{
			"en": "A test tool for conversion",
			"zh": "用于转换的测试工具",
		},
		InputParameters: []ParameterDefinition{
			{
				Name: "input",
				Type: ParamTypeString,
				Description: map[string]string{
					"en": "Input parameter",
					"zh": "输入参数",
				},
				Required: true,
			},
		},
	}

	// 转换为 genai.Schema
	genaiSchema := ConvertToolSchemaToGenaiSchema(toolSchema)

	// 验证结果
	assert.Equal(t, genai.TypeObject, genaiSchema.Type)
	assert.Equal(t, "Test Tool", genaiSchema.Title)
	assert.Equal(t, "A test tool for conversion", genaiSchema.Description)
	assert.Len(t, genaiSchema.Properties, 1)
	assert.Equal(t, []string{"input"}, genaiSchema.Required)

	// 验证 input 属性
	inputSchema := genaiSchema.Properties["input"]
	assert.NotNil(t, inputSchema)
	assert.Equal(t, genai.TypeString, inputSchema.Type)
	assert.Equal(t, "Input parameter", inputSchema.Description)
}

func TestConvertGenaiSchemaToToolSchema(t *testing.T) {
	// 创建 genai.Schema
	genaiSchema := &schema.Schema{
		Type:        genai.TypeObject,
		Title:       "Test Tool",
		Description: "A test tool for conversion",
		Properties: map[string]*schema.Schema{
			"input": {
				Type:        genai.TypeString,
				Description: "Input parameter",
			},
		},
		Required: []string{"input"},
	}

	localizedNames := map[string]string{
		"en": "Test Tool",
		"zh": "测试工具",
	}
	descriptions := map[string]string{
		"en": "A test tool for conversion",
		"zh": "用于转换的测试工具",
	}

	// 转换为 ToolSchema
	toolSchema := ConvertGenaiSchemaToToolSchema(genaiSchema, "test_tool", localizedNames, descriptions)

	// 验证结果
	assert.Equal(t, "test_tool", toolSchema.Name)
	assert.Equal(t, localizedNames, toolSchema.LocalizedNames)
	assert.Equal(t, descriptions, toolSchema.Descriptions)
	assert.Len(t, toolSchema.InputParameters, 1)

	// 验证 input 参数
	inputParam := toolSchema.InputParameters[0]
	assert.Equal(t, "input", inputParam.Name)
	assert.Equal(t, ParamTypeString, inputParam.Type)
	assert.True(t, inputParam.Required)
	assert.Equal(t, "Input parameter", inputParam.Description["en"])
}
